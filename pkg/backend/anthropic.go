package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config ProviderConfig) Provider {
	return &anthropicProvider{config: config}
}

// anthropicProvider implements Provider interface for Anthropic
type anthropicProvider struct {
	config ProviderConfig
}

func (p *anthropicProvider) Name() string {
	return "anthropic"
}

// handleAnthropicError handles Anthropic-specific API error responses
func (p *anthropicProvider) handleAnthropicError(statusCode int, body []byte) error {
	var errorResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("Anthropic API error %d: unable to parse error response: %s", statusCode, string(body))
	}
	return fmt.Errorf("Anthropic API error %d: %s (type: %s)",
		statusCode,
		errorResp.Error.Message,
		errorResp.Error.Type)
}

func (p *anthropicProvider) CreateCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Set up the model from config if not provided in request
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" {
		return nil, fmt.Errorf("model must be specified")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Convert messages to Anthropic format
	// Anthropic requires separating system messages from conversation messages
	var systemMessage string
	var conversationMessages []Message

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			if systemMessage != "" {
				systemMessage += "\n\n" + msg.Content
			} else {
				systemMessage = msg.Content
			}
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	// Create the Anthropic request
	anthropicReq := map[string]interface{}{
		"model":    model,
		"messages": conversationMessages,
	}

	if systemMessage != "" {
		anthropicReq["system"] = systemMessage
	}

	if req.MaxTokens != nil {
		anthropicReq["max_tokens"] = *req.MaxTokens
	} else {
		// Anthropic requires max_tokens to be specified
		anthropicReq["max_tokens"] = 4096
	}

	if req.Temperature != nil {
		anthropicReq["temperature"] = *req.Temperature
	}

	// Marshal the request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set up URL
	url := p.config.URL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Anthropic-specific headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Create HTTP client with timeout
	timeout := time.Duration(p.config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	// Make the request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleAnthropicError(resp.StatusCode, body)
	}

	// Parse Anthropic response format
	var anthropicResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to standard format
	var content string
	if len(anthropicResp.Content) > 0 && anthropicResp.Content[0].Type == "text" {
		content = anthropicResp.Content[0].Text
	}

	response := &ChatCompletionResponse{
		ID:    anthropicResp.ID,
		Model: anthropicResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    RoleAssistant,
					Content: content,
				},
				FinishReason: anthropicResp.StopReason,
			},
		},
		Usage: &Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	return response, nil
}

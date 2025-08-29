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

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config ProviderConfig) Provider {
	return &openAIProvider{config: config}
}

// openAIProvider implements Provider interface for OpenAI
type openAIProvider struct {
	config ProviderConfig
}

func (p *openAIProvider) Name() string {
	return "openai"
}

// handleOpenAIError handles OpenAI-specific API error responses
func (p *openAIProvider) handleOpenAIError(statusCode int, body []byte) error {
	var apiErr APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("OpenAI API error %d: unable to parse error response: %s", statusCode, string(body))
	}
	return fmt.Errorf("OpenAI API error %d: %s (type: %s, code: %s)",
		statusCode,
		apiErr.Error.Message,
		apiErr.Error.Type,
		apiErr.Error.Code)
}

func (p *openAIProvider) CreateCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Set up the model from config if not provided in request
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" {
		return nil, fmt.Errorf("model must be specified")
	}

	// Create the OpenAI request (simplified structure)
	openAIReq := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
	}

	if req.MaxTokens != nil {
		openAIReq["max_tokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		openAIReq["temperature"] = *req.Temperature
	}

	// Marshal the request
	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Set up URL
	url := p.config.URL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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
		return nil, p.handleOpenAIError(resp.StatusCode, body)
	}

	// Parse successful response
	var openAIResp ChatCompletionResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &openAIResp, nil
}

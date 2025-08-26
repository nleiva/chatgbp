package llm

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nleiva/chatgbt/backend"
)

// Client provides LLM interactions with proper context support
type Client struct {
	config     backend.LLMConfig
	httpClient *http.Client
}

// NewClient creates a new LLM client with configurable timeout
func NewClient(config backend.LLMConfig, timeout time.Duration) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewClientWithDefaults creates a new LLM client with default timeout
func NewClientWithDefaults(config backend.LLMConfig) *Client {
	return NewClient(config, 30*time.Second)
}

// Chat sends a chat request and returns the response
func (c *Client) Chat(ctx context.Context, messages []backend.Message) (string, error) {
	response, _, err := c.ChatWithUsage(ctx, messages)
	return response, err
}

// ChatWithUsage sends a chat request and returns the response with usage information
func (c *Client) ChatWithUsage(ctx context.Context, messages []backend.Message) (string, *backend.Usage, error) {
	if err := c.validateConfig(); err != nil {
		return "", nil, err
	}

	requestBody := backend.ChatRequest{
		Model:    c.config.Model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr backend.APIErrorResponse
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return "", nil, fmt.Errorf("error %d: unable to parse error response: %s", resp.StatusCode, string(body))
		}
		return "", nil, fmt.Errorf("error %d: %s (type: %s, code: %s)",
			resp.StatusCode,
			apiErr.Error.Message,
			apiErr.Error.Type,
			apiErr.Error.Code)
	}

	var chatResponse backend.ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(chatResponse.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices returned in response")
	}

	return chatResponse.Choices[0].Message.Content, chatResponse.Usage, nil
}

// validateConfig validates the LLM configuration
func (c *Client) validateConfig() error {
	if c.config.APIKey == "" {
		return fmt.Errorf("missing API key")
	}
	if c.config.URL == "" {
		return fmt.Errorf("missing API URL")
	}
	if c.config.Model == "" {
		return fmt.Errorf("missing model name")
	}
	return nil
}

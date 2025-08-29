package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/nleiva/chatgbt/pkg/backend"
)

// Client provides LLM interactions with proper context support using the new provider system
type Client struct {
	provider backend.Provider
}

// NewClient creates a new LLM client with configurable timeout
func NewClient(config backend.LLMConfig, timeout time.Duration) (*Client, error) {
	// Convert LLMConfig to ProviderConfig
	providerConfig := backend.ProviderConfig{
		Name:    config.Provider,
		APIKey:  config.APIKey,
		URL:     config.URL,
		Model:   config.Model,
		Timeout: int(timeout.Seconds()),
	}

	// Create the provider
	provider, err := backend.CreateProvider(providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Client{
		provider: provider,
	}, nil
}

// CreateCompletion creates a chat completion using the configured provider
func (c *Client) CreateCompletion(ctx context.Context, req *backend.ChatCompletionRequest) (*backend.ChatCompletionResponse, error) {
	return c.provider.CreateCompletion(ctx, req)
}

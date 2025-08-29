package backend

import (
	"context"
	"fmt"
)

// Provider interface defines the contract for LLM providers
type Provider interface {
	// CreateCompletion creates a new chat completion
	CreateCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
	// Name returns the provider name
	Name() string
}

// ProviderName represents the different LLM provider names
type ProviderName string

const (
	ProviderNameOpenAI    ProviderName = "openai"
	ProviderNameAnthropic ProviderName = "anthropic"
	ProviderNameBedrock   ProviderName = "bedrock"
)

// ProviderConfig holds configuration for provider selection and initialization
type ProviderConfig struct {
	Name    ProviderName `json:"name"`    // Provider name (openai, anthropic, bedrock)
	APIKey  string       `json:"api_key"` // API key for authentication
	URL     string       `json:"url"`     // API endpoint URL
	Model   string       `json:"model"`   // Model identifier
	Timeout int          `json:"timeout"` // Request timeout in seconds
}

// LLMConfig holds configuration for LLM API interactions (legacy compatibility)
type LLMConfig struct {
	APIKey    string       `json:"api_key"`    // API key for authentication
	URL       string       `json:"url"`        // API endpoint URL
	Model     string       `json:"model"`      // Model identifier
	Provider  ProviderName `json:"provider"`   // Provider name (openai, anthropic, bedrock)
	ShowUsage bool         `json:"show_usage"` // Whether to return token usage information in responses
}

// Role represents the different message roles in a conversation
type Role string

const (
	RoleSystem    Role = "system"    // System messages help set the behavior of the assistant
	RoleUser      Role = "user"      // User messages are requests or comments from the end-user
	RoleAssistant Role = "assistant" // Assistant messages are responses from the AI assistant
)

// Message represents a single message in the conversation
type Message struct {
	Role    Role   `json:"role"`    // The role of the message author
	Content string `json:"content"` // The contents of the message
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string    `json:"model"`                 // ID of the model to use
	Messages    []Message `json:"messages"`              // A list of messages comprising the conversation
	MaxTokens   *int      `json:"max_tokens,omitempty"`  // The maximum number of tokens that can be generated
	Temperature *float64  `json:"temperature,omitempty"` // Sampling temperature between 0 and 2
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`      // Unique identifier for the chat completion
	Model   string   `json:"model"`   // Model used for the chat completion
	Choices []Choice `json:"choices"` // List of completion choices
	Usage   *Usage   `json:"usage"`   // Usage statistics for the completion request
}

// Choice represents a single completion choice
type Choice struct {
	Index        int     `json:"index"`         // Index of the choice in the list
	Message      Message `json:"message"`       // The generated message
	FinishReason string  `json:"finish_reason"` // Reason the completion finished
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // Number of tokens in the prompt
	CompletionTokens int `json:"completion_tokens"` // Number of tokens in the generated completion
	TotalTokens      int `json:"total_tokens"`      // Total number of tokens used
}

// APIErrorResponse represents an error response from the API
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError represents detailed error information from API
type APIError struct {
	Message string `json:"message"` // Human-readable error message
	Type    string `json:"type"`    // Error type
	Code    string `json:"code"`    // Error code
}

// CreateProvider creates a new provider instance based on the configuration
func CreateProvider(config ProviderConfig) (Provider, error) {
	switch config.Name {
	case ProviderNameOpenAI:
		return NewOpenAIProvider(config), nil
	case ProviderNameAnthropic:
		return NewAnthropicProvider(config), nil
	case ProviderNameBedrock:
		// TODO: Implement Bedrock provider
		return nil, fmt.Errorf("bedrock provider not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Name)
	}
}

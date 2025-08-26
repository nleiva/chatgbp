package backend

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Note: Removed global httpClient variable to avoid global mutable state.
// Each client should have its own HTTP client instance for safety and configurability.

// Role represents the different message roles in a conversation
// As defined in the OpenAI Chat Completions API
type Role string

const (
	RoleSystem    Role = "system"    // System messages help set the behavior of the assistant
	RoleUser      Role = "user"      // User messages are requests or comments from the end-user
	RoleAssistant Role = "assistant" // Assistant messages are responses from the AI assistant
	RoleTool      Role = "tool"      // Tool messages contain results from tool/function calls
)

// Common OpenAI model identifiers
const (
	ModelGPT4oMini  = "gpt-4o-mini"   // Fast and affordable model
	ModelGPT4o      = "gpt-4o"        // High-intelligence flagship model
	ModelGPT4Turbo  = "gpt-4-turbo"   // Previous generation high-intelligence model
	ModelGPT35Turbo = "gpt-3.5-turbo" // Fast, inexpensive model for simple tasks
)

// API endpoint constants
const (
	DefaultChatCompletionsURL = "https://api.openai.com/v1/chat/completions"
)

// Finish reason constants
const (
	FinishReasonStop          = "stop"           // Model hit a natural stop point or provided stop sequence
	FinishReasonLength        = "length"         // Maximum number of tokens specified in the request was reached
	FinishReasonContentFilter = "content_filter" // Content was omitted due to a flag from content filters
	FinishReasonToolCalls     = "tool_calls"     // Model called a tool
)

// LLMConfig holds configuration for OpenAI API interactions
// This struct contains all necessary parameters to make requests to the OpenAI Chat Completions API
type LLMConfig struct {
	APIKey    string `json:"api_key"`    // OpenAI API key for authentication
	URL       string `json:"url"`        // API endpoint URL (default: https://api.openai.com/v1/chat/completions)
	Model     string `json:"model"`      // Model identifier (e.g., "gpt-4o-mini", "gpt-4", "gpt-3.5-turbo")
	ShowUsage bool   `json:"show_usage"` // Whether to return token usage information in responses
}

// validateConfig validates the LLM configuration
func validateConfig(cfg LLMConfig) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("missing API key")
	}
	if cfg.URL == "" {
		return fmt.Errorf("missing API URL")
	}
	if cfg.Model == "" {
		return fmt.Errorf("missing model name")
	}
	return nil
}

// makeRequest performs the HTTP request and returns the parsed response
func makeRequest(cfg LLMConfig, messages []Message) (*ChatResponse, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	requestBody := ChatRequest{
		Model:    cfg.Model,
		Messages: messages,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", cfg.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// Create HTTP client with proper timeout for this request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr APIErrorResponse
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return nil, fmt.Errorf("error %d: unable to parse error response: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("error %d: %s (type: %s, code: %s)",
			resp.StatusCode,
			apiErr.Error.Message,
			apiErr.Error.Type,
			apiErr.Error.Code)
	}

	var chatResponse ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	if len(chatResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned in response")
	}

	return &chatResponse, nil
}

// ChatWithLLMWithUsage returns both the reply and token usage (if available)
func ChatWithLLMWithUsage(cfg LLMConfig, messages []Message) (string, *Usage, error) {
	resp, err := makeRequest(cfg, messages)
	if err != nil {
		return "", nil, err
	}
	return resp.Choices[0].Message.Content, resp.Usage, nil
}

// Message represents a single message in the conversation
// As defined in OpenAI Chat Completions API: https://platform.openai.com/docs/api-reference/chat/create
type Message struct {
	Role       Role   `json:"role"`                   // The role of the message author (system, user, assistant, tool)
	Content    string `json:"content"`                // The contents of the message
	Name       string `json:"name,omitempty"`         // An optional name for the participant (useful for multi-user conversations)
	ToolCallID string `json:"tool_call_id,omitempty"` // Tool call that this message is responding to (for tool role only)
}

// ChatRequest represents the request payload for OpenAI Chat Completions API
// Documentation: https://platform.openai.com/docs/api-reference/chat/create
type ChatRequest struct {
	Model            string             `json:"model"`                       // ID of the model to use
	Messages         []Message          `json:"messages"`                    // A list of messages comprising the conversation so far
	MaxTokens        *int               `json:"max_tokens,omitempty"`        // The maximum number of tokens that can be generated
	Temperature      *float64           `json:"temperature,omitempty"`       // Sampling temperature between 0 and 2
	TopP             *float64           `json:"top_p,omitempty"`             // Nucleus sampling parameter
	N                *int               `json:"n,omitempty"`                 // Number of chat completion choices to generate
	Stream           *bool              `json:"stream,omitempty"`            // Whether to stream partial message deltas
	Stop             []string           `json:"stop,omitempty"`              // Up to 4 sequences where the API will stop generating tokens
	PresencePenalty  *float64           `json:"presence_penalty,omitempty"`  // Penalty for new tokens based on their existing frequency
	FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"` // Penalty for new tokens based on their frequency in the text
	LogitBias        map[string]float64 `json:"logit_bias,omitempty"`        // Modify likelihood of specified tokens appearing
	User             string             `json:"user,omitempty"`              // Unique identifier representing your end-user
	Seed             *int               `json:"seed,omitempty"`              // System fingerprint for reproducible outputs
	Tools            []Tool             `json:"tools,omitempty"`             // List of tools the model may call
	ToolChoice       interface{}        `json:"tool_choice,omitempty"`       // Controls which (if any) tool is called
	ResponseFormat   *ResponseFormat    `json:"response_format,omitempty"`   // Format that the model must output
}

// ChatResponse represents the response from OpenAI Chat Completions API
// Documentation: https://platform.openai.com/docs/api-reference/chat/object
type ChatResponse struct {
	ID                string   `json:"id"`                 // Unique identifier for the chat completion
	Object            string   `json:"object"`             // Object type, always "chat.completion"
	Created           int64    `json:"created"`            // Unix timestamp of when the completion was created
	Model             string   `json:"model"`              // Model used for the chat completion
	SystemFingerprint string   `json:"system_fingerprint"` // Fingerprint of the system configuration
	Choices           []Choice `json:"choices"`            // List of completion choices
	Usage             *Usage   `json:"usage,omitempty"`    // Usage statistics for the completion request
}

// Usage represents token usage information for a completion request
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`                       // Number of tokens in the prompt
	CompletionTokens        int                      `json:"completion_tokens"`                   // Number of tokens in the generated completion
	TotalTokens             int                      `json:"total_tokens"`                        // Total number of tokens used
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // Breakdown of completion tokens
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`     // Breakdown of prompt tokens
}

// CompletionTokensDetails provides a breakdown of completion tokens
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // Tokens generated for reasoning
}

// PromptTokensDetails provides a breakdown of prompt tokens
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"` // Number of cached tokens in the prompt
}

// Choice represents a single completion choice
type Choice struct {
	Index        int       `json:"index"`              // Index of the choice in the list
	Message      Message   `json:"message"`            // The generated message
	Logprobs     *Logprobs `json:"logprobs,omitempty"` // Log probability information for tokens
	FinishReason string    `json:"finish_reason"`      // Reason the completion finished (stop, length, content_filter, tool_calls)
}

// Logprobs represents log probability information for the choice's tokens
type Logprobs struct {
	Content []TokenLogprob `json:"content,omitempty"` // Log probability information for the choice's tokens
}

// TokenLogprob represents log probability information for a single token
type TokenLogprob struct {
	Token       string       `json:"token"`           // The token
	Logprob     float64      `json:"logprob"`         // Log probability of the token
	Bytes       []int        `json:"bytes,omitempty"` // Byte representation of the token
	TopLogprobs []TopLogprob `json:"top_logprobs"`    // List of most likely tokens and their log probabilities
}

// TopLogprob represents a top log probability alternative
type TopLogprob struct {
	Token   string  `json:"token"`           // The token
	Logprob float64 `json:"logprob"`         // Log probability of the token
	Bytes   []int   `json:"bytes,omitempty"` // Byte representation of the token
}

// Tool represents a tool that can be called by the model
type Tool struct {
	Type     string       `json:"type"`     // Type of tool (currently only "function" is supported)
	Function ToolFunction `json:"function"` // Function definition
}

// ToolFunction represents a function that can be called
type ToolFunction struct {
	Name        string      `json:"name"`                  // Name of the function
	Description string      `json:"description,omitempty"` // Description of the function
	Parameters  interface{} `json:"parameters,omitempty"`  // Parameters the function accepts (JSON Schema object)
}

// ResponseFormat specifies the format that the model must output
type ResponseFormat struct {
	Type string `json:"type"` // Must be "text" or "json_object"
}

// APIErrorResponse represents an error response from the OpenAI API
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError represents detailed error information from OpenAI API
type APIError struct {
	Message string `json:"message"`         // Human-readable error message
	Type    string `json:"type"`            // Error type (e.g., "invalid_request_error")
	Param   string `json:"param,omitempty"` // The parameter that caused the error
	Code    string `json:"code,omitempty"`  // Error code (e.g., "invalid_api_key")
}

// ChatWithLLM is a convenience function that returns only the response content
// without usage information, for simpler use cases
func ChatWithLLM(cfg LLMConfig, messages []Message) (string, error) {
	resp, err := makeRequest(cfg, messages)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// NewChatRequest creates a new ChatRequest with sensible defaults
func NewChatRequest(model string, messages []Message) *ChatRequest {
	return &ChatRequest{
		Model:    model,
		Messages: messages,
	}
}

// WithTemperature sets the temperature parameter for more creative or deterministic responses
func (r *ChatRequest) WithTemperature(temp float64) *ChatRequest {
	r.Temperature = &temp
	return r
}

// WithMaxTokens sets the maximum number of tokens to generate
func (r *ChatRequest) WithMaxTokens(maxTokens int) *ChatRequest {
	r.MaxTokens = &maxTokens
	return r
}

// WithUser sets a unique identifier for the end-user (useful for abuse monitoring)
func (r *ChatRequest) WithUser(user string) *ChatRequest {
	r.User = user
	return r
}

// NewMessage creates a new Message with the specified role and content
func NewMessage(role Role, content string) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

// NewSystemMessage creates a system message (used to set assistant behavior)
func NewSystemMessage(content string) Message {
	return NewMessage(RoleSystem, content)
}

// NewUserMessage creates a user message (requests or comments from end-user)
func NewUserMessage(content string) Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates an assistant message (AI responses)
func NewAssistantMessage(content string) Message {
	return NewMessage(RoleAssistant, content)
}

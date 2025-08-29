package app

import (
	"context"
	"strings"
	"time"

	"github.com/nleiva/chatgbt/pkg/backend"
	"github.com/nleiva/chatgbt/pkg/llm"
)

// ChatSession represents a conversation session with shared logic for CLI and Web modes
type ChatSession struct {
	ID               string
	Messages         []backend.Message
	SystemPrompt     string
	ConversationType string

	// Dependencies
	LLMClient      LLMClient
	Logger         Logger
	ContextManager *backend.ContextManager
}

// SessionConfig holds configuration for creating a new session
type SessionConfig struct {
	ID               string
	ConversationType string
	SystemPrompt     string
	LLMConfig        backend.LLMConfig
	BudgetConfig     backend.TokenBudgetConfig
	MaxTokens        int
	KeepRecent       int
	SummaryEnabled   bool
}

// NewChatSession creates a new chat session with all dependencies initialized
func NewChatSession(config SessionConfig) (*ChatSession, error) {
	// Initialize LLM client
	llmClient, err := llm.NewClient(config.LLMConfig, 30*time.Second)
	if err != nil {
		return nil, err
	}

	// Initialize metrics logger
	logger, err := NewMetricsLogger(config.ID, config.ConversationType, config.BudgetConfig)
	if err != nil {
		return nil, err
	}

	// Initialize context manager
	contextManager := backend.NewContextManager(config.MaxTokens, config.KeepRecent, config.SummaryEnabled)

	// Initialize messages with system prompt
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant."
	}

	session := &ChatSession{
		ID:               config.ID,
		Messages:         []backend.Message{{Role: backend.RoleSystem, Content: systemPrompt}},
		SystemPrompt:     systemPrompt,
		ConversationType: config.ConversationType,
		LLMClient:        llmClient,
		Logger:           logger,
		ContextManager:   contextManager,
	}

	return session, nil
}

// ProcessUserMessage handles a user message and returns the assistant's response
func (s *ChatSession) ProcessUserMessage(userMessage string) (*ChatResponse, error) {
	// Auto-prune context if needed
	if s.ContextManager.ShouldPrune(s.Messages) {
		s.AutoPrune()
	}

	// Check message bounds to prevent memory issues
	if len(s.Messages) > 1000 { // Hard limit to prevent unbounded growth
		s.AutoPrune()
	}

	// Add user message
	s.Messages = append(s.Messages, backend.Message{
		Role:    backend.RoleUser,
		Content: userMessage,
	})

	// Classify prompt type
	promptType := ClassifyPrompt(userMessage)

	// Get LLM response with timing and timeout
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create completion request
	req := &backend.ChatCompletionRequest{
		Messages: s.Messages,
	}

	resp, err := s.LLMClient.CreateCompletion(ctx, req)
	responseTime := time.Since(startTime)

	var reply string
	var usage *backend.Usage
	if err == nil && len(resp.Choices) > 0 {
		reply = resp.Choices[0].Message.Content
		usage = resp.Usage
	}

	// Log the interaction
	s.Logger.LogInteraction(backend.InteractionLog{
		Usage:        usage,
		ResponseTime: responseTime,
		Success:      err == nil,
		ErrorType:    getErrorType(err),
		PromptType:   promptType,
	})

	if err != nil {
		// Remove failed user message
		s.removeLastUserMessage()
		return nil, err
	}

	// Add assistant response
	s.Messages = append(s.Messages, backend.Message{
		Role:    backend.RoleAssistant,
		Content: reply,
	})

	// Prepare budget warnings
	budgetStatus := s.Logger.GetBudgetStatus()
	var warnings []string
	if len(budgetStatus.Warnings) > 0 {
		warnings = budgetStatus.Warnings
	}

	return &ChatResponse{
		Content:      reply,
		Usage:        usage,
		ResponseTime: responseTime,
		Warnings:     warnings,
		PromptType:   promptType,
	}, nil
}

// Reset resets the conversation with a new system prompt
func (s *ChatSession) Reset(systemPrompt string) {
	if systemPrompt == "" {
		systemPrompt = s.SystemPrompt
	}
	s.SystemPrompt = systemPrompt
	s.Messages = []backend.Message{{Role: backend.RoleSystem, Content: systemPrompt}}
}

// UpdateSystemPrompt updates the system prompt and resets the conversation
func (s *ChatSession) UpdateSystemPrompt(newPrompt string) {
	s.SystemPrompt = newPrompt
	s.Reset(newPrompt)
}

// AutoPrune performs automatic context pruning
func (s *ChatSession) AutoPrune() bool {
	originalTokens := s.ContextManager.EstimateTokens(s.Messages)

	newMessages, pruned := s.ContextManager.PruneContext(s.Messages, originalTokens)
	if pruned {
		s.Messages = newMessages
		return true
	}
	return false
}

// GetContextStats returns current context statistics
func (s *ChatSession) GetContextStats() backend.ContextStats {
	return s.ContextManager.GetContextStats(s.Messages)
}

// GetSessionSummary returns session metrics summary
func (s *ChatSession) GetSessionSummary() backend.SessionSummary {
	return s.Logger.GetSessionSummary()
}

// GetBudgetStatus returns current budget status
func (s *ChatSession) GetBudgetStatus() backend.BudgetStatus {
	return s.Logger.GetBudgetStatus()
}

// GetPromptTypeBreakdown returns a breakdown of prompt types used in this session
func (s *ChatSession) GetPromptTypeBreakdown() map[string]int {
	return s.Logger.GetPromptTypeBreakdown()
}

// Close properly closes the session
func (s *ChatSession) Close() error {
	return s.Logger.Close()
}

// removeLastUserMessage removes the last user message (used on errors)
func (s *ChatSession) removeLastUserMessage() {
	if len(s.Messages) > 0 && s.Messages[len(s.Messages)-1].Role == backend.RoleUser {
		s.Messages = s.Messages[:len(s.Messages)-1]
	}
}

// ChatResponse represents the response from processing a user message
type ChatResponse struct {
	Content      string
	Usage        *backend.Usage
	ResponseTime time.Duration
	Warnings     []string
	PromptType   string
}

// getErrorType converts an error to a classification string
func getErrorType(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "api"):
		return "api_error"
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "timeout"):
		return "network_error"
	case strings.Contains(errStr, "auth") || strings.Contains(errStr, "unauthorized"):
		return "auth_error"
	case strings.Contains(errStr, "quota") || strings.Contains(errStr, "limit"):
		return "quota_error"
	default:
		return "unknown_error"
	}
}

package app

import (
	"fmt"
	"time"

	"github.com/nleiva/chatgbt/backend"
	"github.com/nleiva/chatgbt/llm"
)

// NewLLMClient creates a new LLM client with default timeout
func NewLLMClient(config backend.LLMConfig) LLMClient {
	return llm.NewClientWithDefaults(config)
}

// MetricsLoggerAdapter adapts backend.MetricsLogger to implement the app.Logger interface
type MetricsLoggerAdapter struct {
	*backend.MetricsLogger
}

// NewMetricsLogger creates a new metrics logger that implements the app.Logger interface
func NewMetricsLogger(sessionID, conversationType string, budgetCfg backend.TokenBudgetConfig) (Logger, error) {
	metricsLogger, err := backend.NewMetricsLogger(sessionID, conversationType, budgetCfg)
	if err != nil {
		return nil, err
	}

	return &MetricsLoggerAdapter{
		MetricsLogger: metricsLogger,
	}, nil
}

// GetBudgetStatus implements the Logger interface by calling CheckBudgetStatus
func (m *MetricsLoggerAdapter) GetBudgetStatus() backend.BudgetStatus {
	return m.CheckBudgetStatus()
}

// GetPromptTypeBreakdown implements the Logger interface
func (m *MetricsLoggerAdapter) GetPromptTypeBreakdown() map[string]int {
	return m.MetricsLogger.GetPromptTypeBreakdown()
}

// SessionManagerConfig holds configuration for creating session managers
type SessionManagerConfig struct {
	LLMConfig      backend.LLMConfig         // LLM client configuration
	BudgetConfig   backend.TokenBudgetConfig // Financial tracking & usage warnings
	MaxTokens      int                       // Auto-prune conversation context at this limit
	KeepRecent     int                       // Number of recent exchanges to preserve when pruning
	SummaryEnabled bool                      // Enable context summaries for pruned content
}

// DefaultSessionManagerConfig returns sensible defaults for session management
func DefaultSessionManagerConfig(llmConfig backend.LLMConfig, budgetConfig backend.TokenBudgetConfig) SessionManagerConfig {
	return SessionManagerConfig{
		LLMConfig:      llmConfig,
		BudgetConfig:   budgetConfig,
		MaxTokens:      6000,
		KeepRecent:     3,
		SummaryEnabled: true,
	}
}

// NewChatSessionWithDefaults creates a new chat session with default configuration
func NewChatSessionWithDefaults(id, conversationType, systemPrompt string, llmConfig backend.LLMConfig, budgetConfig backend.TokenBudgetConfig) (*ChatSession, error) {
	config := SessionConfig{
		ID:               id,
		ConversationType: conversationType,
		SystemPrompt:     systemPrompt,
		LLMConfig:        llmConfig,
		BudgetConfig:     budgetConfig,
		MaxTokens:        6000,
		KeepRecent:       3,
		SummaryEnabled:   true,
	}

	return NewChatSession(config)
}

// GenerateSessionID creates a unique session ID based on the mode and current time
func GenerateSessionID(mode string) string {
	return fmt.Sprintf("%s_%d", mode, time.Now().Unix())
}

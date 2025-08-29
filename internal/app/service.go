package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/nleiva/chatgbt/pkg/backend"
)

// LLMClient defines the interface for Large Language Model interactions.
// It provides both simple chat and chat-with-usage methods for different use cases.
type LLMClient interface {
	CreateCompletion(ctx context.Context, req *backend.ChatCompletionRequest) (*backend.ChatCompletionResponse, error)
}

// InteractionLogger handles logging of individual interactions
type InteractionLogger interface {
	LogInteraction(log backend.InteractionLog)
}

// SessionReporter provides session metrics and analysis
type SessionReporter interface {
	GetSessionSummary() backend.SessionSummary
	GetBudgetStatus() backend.BudgetStatus
	GetPromptTypeBreakdown() map[string]int
}

// Closer handles resource cleanup
type Closer interface {
	Close() error
}

// Logger combines all logging interfaces for backward compatibility
// Consider using the individual interfaces in new code
type Logger interface {
	InteractionLogger
	SessionReporter
	Closer
}

// DirectQueryService handles single-query interactions for quick responses.
// It coordinates between the LLM client, logger, and output writer to process
// user queries and display results with optional usage statistics.
type DirectQueryService struct {
	client LLMClient
	logger Logger
	writer io.Writer
}

// NewDirectQueryService creates a new direct query service with the specified dependencies.
func NewDirectQueryService(client LLMClient, logger Logger, writer io.Writer) *DirectQueryService {
	return &DirectQueryService{
		client: client,
		logger: logger,
		writer: writer,
	}
}

// Execute performs a direct query and returns the result
func (s *DirectQueryService) Execute(ctx context.Context, query string, showUsage bool) error {
	messages := []backend.Message{
		{Role: backend.RoleUser, Content: query},
	}

	// Add timeout if none exists
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	start := time.Now()
	// Create completion request
	req := &backend.ChatCompletionRequest{
		Messages: messages,
	}

	resp, err := s.client.CreateCompletion(ctx, req)
	responseTime := time.Since(start)

	var response string
	var usage *backend.Usage
	if err == nil && len(resp.Choices) > 0 {
		response = resp.Choices[0].Message.Content
		usage = resp.Usage
	}

	if err != nil {
		s.logger.LogInteraction(backend.InteractionLog{
			Usage:        nil,
			ResponseTime: responseTime,
			Success:      false,
			ErrorType:    err.Error(),
			PromptType:   "user_query",
		})
		return err
	}

	s.logger.LogInteraction(backend.InteractionLog{
		Usage:        usage,
		ResponseTime: responseTime,
		Success:      true,
		ErrorType:    "",
		PromptType:   "user_query",
	})

	// Print the response
	if _, writeErr := s.writer.Write([]byte(response + "\n")); writeErr != nil {
		return writeErr
	}

	// Print usage stats if enabled
	if showUsage && usage != nil {
		summary := s.logger.GetSessionSummary()
		if _, writeErr := io.WriteString(s.writer,
			fmt.Sprintf("Tokens: %d | Cost: $%.4f | Time: %.1fs\n",
				usage.TotalTokens, summary.EstimatedCost, responseTime.Seconds())); writeErr != nil {
			return writeErr
		}
	}

	return nil
}

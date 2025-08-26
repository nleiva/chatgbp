package backend

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SessionMetrics tracks metrics for a single conversation session
type SessionMetrics struct {
	SessionID        string              `json:"session_id"`
	StartTime        time.Time           `json:"start_time"`
	EndTime          *time.Time          `json:"end_time,omitempty"`
	TotalRequests    int                 `json:"total_requests"`
	SuccessfulReqs   int                 `json:"successful_requests"`
	FailedReqs       int                 `json:"failed_requests"`
	TotalTokens      int                 `json:"total_tokens"`
	PromptTokens     int                 `json:"prompt_tokens"`
	CompletionTokens int                 `json:"completion_tokens"`
	Interactions     []InteractionMetric `json:"interactions"`
	ConversationType string              `json:"conversation_type"` // "quick", "debug", "creative", etc.
	EstimatedCost    float64             `json:"estimated_cost"`
}

// InteractionMetric tracks a single request/response cycle
type InteractionMetric struct {
	Timestamp      time.Time `json:"timestamp"`
	RequestTokens  int       `json:"request_tokens"`
	ResponseTokens int       `json:"response_tokens"`
	TotalTokens    int       `json:"total_tokens"`
	ResponseTime   int64     `json:"response_time_ms"`
	Success        bool      `json:"success"`
	ErrorType      string    `json:"error_type,omitempty"`
	PromptType     string    `json:"prompt_type"` // "system", "user", "code_help", etc.
}

// MetricsLogger handles session logging and token budget tracking
type MetricsLogger struct {
	session   *SessionMetrics
	logFile   *os.File
	budgetCfg TokenBudgetConfig
}

// TokenBudgetConfig defines token usage limits and warnings
type TokenBudgetConfig struct {
	DailyLimit     int     `json:"daily_limit"`     // Max tokens per day
	SessionLimit   int     `json:"session_limit"`   // Max tokens per session
	WarnThreshold  float64 `json:"warn_threshold"`  // Warn at % of limit (0.8 = 80%)
	PruneThreshold int     `json:"prune_threshold"` // Prune context when session exceeds this
	CostPerToken   float64 `json:"cost_per_token"`  // Estimated cost per token
}

// NewMetricsLogger creates a new metrics logger with session tracking
func NewMetricsLogger(sessionID string, conversationType string, budgetCfg TokenBudgetConfig) (*MetricsLogger, error) {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Open log file for this session
	logFileName := filepath.Join(logsDir, fmt.Sprintf("session_%s_%s.jsonl",
		time.Now().Format("2006-01-02"), sessionID))

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	session := &SessionMetrics{
		SessionID:        sessionID,
		StartTime:        time.Now(),
		ConversationType: conversationType,
		Interactions:     make([]InteractionMetric, 0),
	}

	return &MetricsLogger{
		session:   session,
		logFile:   logFile,
		budgetCfg: budgetCfg,
	}, nil
}

// InteractionLog represents the details of a single interaction for logging
type InteractionLog struct {
	Usage        *Usage        `json:"usage,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Success      bool          `json:"success"`
	ErrorType    string        `json:"error_type,omitempty"`
	PromptType   string        `json:"prompt_type"`
}

// LogInteraction records a single API interaction using a structured log
func (ml *MetricsLogger) LogInteraction(log InteractionLog) {
	interaction := InteractionMetric{
		Timestamp:    time.Now(),
		ResponseTime: log.ResponseTime.Milliseconds(),
		Success:      log.Success,
		ErrorType:    log.ErrorType,
		PromptType:   log.PromptType,
	}

	if log.Usage != nil {
		interaction.RequestTokens = log.Usage.PromptTokens
		interaction.ResponseTokens = log.Usage.CompletionTokens
		interaction.TotalTokens = log.Usage.TotalTokens

		// Update session totals
		ml.session.TotalTokens += log.Usage.TotalTokens
		ml.session.PromptTokens += log.Usage.PromptTokens
		ml.session.CompletionTokens += log.Usage.CompletionTokens
		ml.session.EstimatedCost += float64(log.Usage.TotalTokens) * ml.budgetCfg.CostPerToken
	}

	ml.session.TotalRequests++
	if log.Success {
		ml.session.SuccessfulReqs++
	} else {
		ml.session.FailedReqs++
	}

	ml.session.Interactions = append(ml.session.Interactions, interaction)

	// Write to log file
	if logLine, err := json.Marshal(interaction); err == nil {
		ml.logFile.WriteString(string(logLine) + "\n")
		ml.logFile.Sync()
	}
}

// LogInteractionLegacy provides backward compatibility for the old method signature
// Deprecated: Use LogInteraction with InteractionLog struct instead
func (ml *MetricsLogger) LogInteractionLegacy(usage *Usage, responseTime time.Duration, success bool, errorType string, promptType string) {
	ml.LogInteraction(InteractionLog{
		Usage:        usage,
		ResponseTime: responseTime,
		Success:      success,
		ErrorType:    errorType,
		PromptType:   promptType,
	})
}

// CheckBudgetStatus returns warnings and recommendations based on current usage
func (ml *MetricsLogger) CheckBudgetStatus() BudgetStatus {
	status := BudgetStatus{
		SessionTokens: ml.session.TotalTokens,
		SessionCost:   ml.session.EstimatedCost,
		SessionLimit:  ml.budgetCfg.SessionLimit,
		DailyLimit:    ml.budgetCfg.DailyLimit,
		ShouldPrune:   ml.session.TotalTokens > ml.budgetCfg.PruneThreshold,
	}

	// Check session budget
	if ml.budgetCfg.SessionLimit > 0 {
		sessionUsage := float64(ml.session.TotalTokens) / float64(ml.budgetCfg.SessionLimit)
		if sessionUsage > ml.budgetCfg.WarnThreshold {
			status.Warnings = append(status.Warnings,
				fmt.Sprintf("Session token usage at %.1f%% of limit (%d/%d tokens)",
					sessionUsage*100, ml.session.TotalTokens, ml.budgetCfg.SessionLimit))
		}
		if sessionUsage > 1.0 {
			status.OverBudget = true
		}
	}

	// Add daily usage check here (would need to read previous sessions)
	// For now, just check if we're getting expensive
	if ml.session.EstimatedCost > 1.0 {
		status.Warnings = append(status.Warnings,
			fmt.Sprintf("Session cost: $%.3f", ml.session.EstimatedCost))
	}

	return status
}

// GetSessionSummary returns a summary of the current session
func (ml *MetricsLogger) GetSessionSummary() SessionSummary {
	duration := time.Since(ml.session.StartTime)

	avgResponseTime := int64(0)
	if len(ml.session.Interactions) > 0 {
		var totalTime int64
		for _, interaction := range ml.session.Interactions {
			totalTime += interaction.ResponseTime
		}
		avgResponseTime = totalTime / int64(len(ml.session.Interactions))
	}

	return SessionSummary{
		Duration:         duration,
		TotalRequests:    ml.session.TotalRequests,
		SuccessRate:      float64(ml.session.SuccessfulReqs) / float64(ml.session.TotalRequests),
		TotalTokens:      ml.session.TotalTokens,
		EstimatedCost:    ml.session.EstimatedCost,
		AvgResponseTime:  avgResponseTime,
		ConversationType: ml.session.ConversationType,
	}
}

// GetPromptTypeBreakdown returns a breakdown of prompt types used in this session
func (ml *MetricsLogger) GetPromptTypeBreakdown() map[string]int {
	breakdown := make(map[string]int)

	for _, interaction := range ml.session.Interactions {
		if interaction.PromptType != "" {
			breakdown[interaction.PromptType]++
		}
	}

	return breakdown
}

// Close finalizes the session and closes log files
func (ml *MetricsLogger) Close() error {
	now := time.Now()
	ml.session.EndTime = &now

	// Write final session summary
	if sessionData, err := json.Marshal(ml.session); err == nil {
		ml.logFile.WriteString("SESSION_SUMMARY: " + string(sessionData) + "\n")
	}

	return ml.logFile.Close()
}

// BudgetStatus represents current budget status and warnings
type BudgetStatus struct {
	SessionTokens int
	SessionCost   float64
	SessionLimit  int
	DailyLimit    int
	Warnings      []string
	OverBudget    bool
	ShouldPrune   bool
}

// SessionSummary provides a summary of session metrics
type SessionSummary struct {
	Duration         time.Duration
	TotalRequests    int
	SuccessRate      float64
	TotalTokens      int
	EstimatedCost    float64
	AvgResponseTime  int64
	ConversationType string
}

// DefaultBudgetConfig returns sensible defaults for token budgeting
func DefaultBudgetConfig() TokenBudgetConfig {
	return TokenBudgetConfig{
		DailyLimit:     50000,    // 50k tokens per day
		SessionLimit:   10000,    // 10k tokens per session
		WarnThreshold:  0.8,      // Warn at 80% usage
		PruneThreshold: 8000,     // Prune context at 8k tokens
		CostPerToken:   0.000002, // Approximate GPT-3.5-turbo cost
	}
}

// LogBasicInfo logs non-sensitive information for debugging
func LogBasicInfo(message string, data interface{}) {
	logData := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   message,
		"data":      data,
	}

	if jsonData, err := json.Marshal(logData); err == nil {
		log.Printf("METRICS: %s", string(jsonData))
	}
}

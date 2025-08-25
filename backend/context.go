package backend

import (
	"fmt"
	"strings"
)

// ContextManager handles conversation pruning and summarization
type ContextManager struct {
	maxTokens      int
	keepRecent     int // Number of recent exchanges to always keep
	summaryEnabled bool
}

// NewContextManager creates a new context manager
func NewContextManager(maxTokens, keepRecent int, summaryEnabled bool) *ContextManager {
	return &ContextManager{
		maxTokens:      maxTokens,
		keepRecent:     keepRecent,
		summaryEnabled: summaryEnabled,
	}
}

// PruneContext reduces message array size when approaching token limits
func (cm *ContextManager) PruneContext(messages []Message, currentTokens int) ([]Message, bool) {
	if currentTokens <= cm.maxTokens {
		return messages, false
	}

	// Always keep system message if present
	systemMessage := Message{}
	startIdx := 0
	if len(messages) > 0 && messages[0].Role == RoleSystem {
		systemMessage = messages[0]
		startIdx = 1
	}

	// Calculate how many recent exchanges to keep (user + assistant pairs)
	// Keep at least the last few exchanges for context continuity
	userMessages := make([]Message, 0)
	for i := startIdx; i < len(messages); i++ {
		userMessages = append(userMessages, messages[i])
	}

	// If we have fewer messages than keepRecent*2, just return original
	if len(userMessages) <= cm.keepRecent*2 {
		return messages, false
	}

	// Strategy 1: Keep system + recent exchanges
	recentStart := len(userMessages) - (cm.keepRecent * 2)
	if recentStart < 0 {
		recentStart = 0
	}

	prunedMessages := make([]Message, 0)

	// Add system message back if it existed
	if systemMessage.Content != "" {
		prunedMessages = append(prunedMessages, systemMessage)
	}

	// Add summary of pruned content if enabled
	if cm.summaryEnabled && recentStart > 0 {
		summaryContent := cm.createSummary(userMessages[:recentStart])
		if summaryContent != "" {
			prunedMessages = append(prunedMessages, Message{
				Role:    RoleSystem,
				Content: fmt.Sprintf("Previous conversation summary: %s", summaryContent),
			})
		}
	}

	// Add recent messages
	prunedMessages = append(prunedMessages, userMessages[recentStart:]...)

	return prunedMessages, true
}

// createSummary creates a simple summary of pruned messages
func (cm *ContextManager) createSummary(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}

	var topics []string
	var userQuestions int
	var assistantResponses int

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			userQuestions++
			// Extract key topics (simple keyword detection)
			content := strings.ToLower(msg.Content)
			if strings.Contains(content, "code") || strings.Contains(content, "debug") || strings.Contains(content, "error") {
				if !contains(topics, "code/debugging") {
					topics = append(topics, "code/debugging")
				}
			}
			if strings.Contains(content, "explain") || strings.Contains(content, "how") || strings.Contains(content, "what") {
				if !contains(topics, "explanations") {
					topics = append(topics, "explanations")
				}
			}
			if strings.Contains(content, "write") || strings.Contains(content, "create") || strings.Contains(content, "generate") {
				if !contains(topics, "content creation") {
					topics = append(topics, "content creation")
				}
			}
		case RoleAssistant:
			assistantResponses++
		}
	}

	summary := fmt.Sprintf("Discussed %s across %d exchanges",
		strings.Join(topics, ", "), userQuestions)

	if len(topics) == 0 {
		summary = fmt.Sprintf("General conversation with %d exchanges", userQuestions)
	}

	return summary
}

// EstimateTokens provides a rough estimate of token count for a message array
// This is a simplified approximation - real tokenization would be more accurate
func (cm *ContextManager) EstimateTokens(messages []Message) int {
	totalChars := 0
	for _, msg := range messages {
		// Count characters in role and content, plus some overhead for JSON structure
		totalChars += len(msg.Role) + len(msg.Content) + 20 // 20 chars overhead per message
	}

	// Rough approximation: 1 token ≈ 4 characters for English text
	// This is conservative - actual tokenization varies
	return totalChars / 4
}

// ShouldPrune checks if context pruning is recommended
func (cm *ContextManager) ShouldPrune(messages []Message) bool {
	estimatedTokens := cm.EstimateTokens(messages)
	return estimatedTokens > cm.maxTokens
}

// GetContextStats returns statistics about the current context
func (cm *ContextManager) GetContextStats(messages []Message) ContextStats {
	estimatedTokens := cm.EstimateTokens(messages)

	userMsgs := 0
	assistantMsgs := 0
	systemMsgs := 0

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			userMsgs++
		case RoleAssistant:
			assistantMsgs++
		case RoleSystem:
			systemMsgs++
		}
	}

	return ContextStats{
		TotalMessages:     len(messages),
		UserMessages:      userMsgs,
		AssistantMessages: assistantMsgs,
		SystemMessages:    systemMsgs,
		EstimatedTokens:   estimatedTokens,
		TokenLimit:        cm.maxTokens,
		UtilizationPct:    float64(estimatedTokens) / float64(cm.maxTokens) * 100,
		ShouldPrune:       estimatedTokens > cm.maxTokens,
	}
}

// ContextStats provides information about current context usage
type ContextStats struct {
	TotalMessages     int
	UserMessages      int
	AssistantMessages int
	SystemMessages    int
	EstimatedTokens   int
	TokenLimit        int
	UtilizationPct    float64
	ShouldPrune       bool
}

// helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

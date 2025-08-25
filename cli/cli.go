package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nleiva/chatgbt/backend"
)

const (
	cmdExit   = "exit"
	cmdReset  = "/reset"
	cmdSystem = "/system"
	cmdBudget = "/budget"
	cmdStats  = "/stats"
	cmdPrune  = "/prune"
)

// Enhanced CLI session with metrics and budget tracking
type CLISession struct {
	cfg            backend.LLMConfig
	metrics        *backend.MetricsLogger
	contextManager *backend.ContextManager
	reader         *bufio.Reader
	systemPrompt   string
	messages       []backend.Message
	sessionID      string
}

// printMOTD displays the ChatGBT ASCII art banner
func printMOTD() {
	fmt.Print(`
 ██████╗██╗  ██╗ █████╗ ████████╗ ██████╗ ██████╗ ████████╗
██╔════╝██║  ██║██╔══██╗╚══██╔══╝██╔════╝ ██╔══██╗╚══██╔══╝
██║     ███████║███████║   ██║   ██║  ███╗██████╔╝   ██║   
██║     ██╔══██║██╔══██║   ██║   ██║   ██║██╔══██╗   ██║   
╚██████╗██║  ██║██║  ██║   ██║   ╚██████╔╝██████╔╝   ██║   
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═════╝    ╚═╝   
                                                            
        🤖 Language Model Assistant for Educational Purposes
        ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`)
}

// resetMessages resets the conversation with the current system prompt
func (s *CLISession) resetMessages() {
	s.messages = []backend.Message{{Role: backend.RoleSystem, Content: s.systemPrompt}}
}

// readMultilineInput reads user input until an empty line is entered
func (s *CLISession) readMultilineInput() (string, error) {
	fmt.Println("You (end with empty line):")
	var userLines []string
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			switch err.Error() {
			case "EOF":
				if len(userLines) == 0 {
					return "", err
				}
				// If we have some input, treat it as end of input
				return strings.TrimSpace(strings.Join(userLines, "\n")), nil
			default:
				return "", err
			}
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		userLines = append(userLines, line)
	}
	return strings.TrimSpace(strings.Join(userLines, "\n")), nil
}

// handleSystemPromptUpdate handles the /system command
func (s *CLISession) handleSystemPromptUpdate() error {
	fmt.Print("Enter new system prompt: ")
	newPrompt, err := s.reader.ReadString('\n')
	if err != nil {
		return err
	}
	s.systemPrompt = strings.TrimSpace(newPrompt)
	s.resetMessages()
	fmt.Println("System prompt updated.")
	return nil
}

// showBudgetStatus displays current budget and usage information
func (s *CLISession) showBudgetStatus() {
	status := s.metrics.CheckBudgetStatus()

	fmt.Println("\nBudget Status:")
	fmt.Printf("   Session Tokens: %d", status.SessionTokens)
	if status.SessionLimit > 0 {
		fmt.Printf(" / %d (%.1f%%)", status.SessionLimit,
			float64(status.SessionTokens)/float64(status.SessionLimit)*100)
	}
	fmt.Printf("\n   Estimated Cost: $%.4f\n", status.SessionCost)

	if len(status.Warnings) > 0 {
		fmt.Println("   Warnings:")
		for _, warning := range status.Warnings {
			fmt.Printf("     - %s\n", warning)
		}
	}

	if status.ShouldPrune {
		fmt.Println("   Recommendation: Consider pruning context (/prune)")
	}

	fmt.Println()
}

// showContextStats displays current context statistics
func (s *CLISession) showContextStats() {
	stats := s.contextManager.GetContextStats(s.messages)

	fmt.Println("\nContext Statistics:")
	fmt.Printf("   Messages: %d total (%d user, %d assistant, %d system)\n",
		stats.TotalMessages, stats.UserMessages, stats.AssistantMessages, stats.SystemMessages)
	fmt.Printf("   Estimated Tokens: %d / %d (%.1f%%)\n",
		stats.EstimatedTokens, stats.TokenLimit, stats.UtilizationPct)

	if stats.ShouldPrune {
		fmt.Println("   Warning: Context approaching token limit")
	}

	// Show session summary
	summary := s.metrics.GetSessionSummary()
	fmt.Printf("   Session Duration: %v\n", summary.Duration.Round(time.Second))
	fmt.Printf("   Requests: %d (%.1f%% success rate)\n",
		summary.TotalRequests, summary.SuccessRate*100)
	if summary.AvgResponseTime > 0 {
		fmt.Printf("   Avg Response Time: %dms\n", summary.AvgResponseTime)
	}
	fmt.Println()
}

// pruneContext manually triggers context pruning
func (s *CLISession) pruneContext() {
	originalCount := len(s.messages)
	originalTokens := s.contextManager.EstimateTokens(s.messages)

	newMessages, pruned := s.contextManager.PruneContext(s.messages, originalTokens)

	if pruned {
		s.messages = newMessages
		newTokens := s.contextManager.EstimateTokens(s.messages)
		fmt.Printf("Context pruned: %d -> %d messages, ~%d -> ~%d tokens\n",
			originalCount, len(s.messages), originalTokens, newTokens)
	} else {
		fmt.Println("No pruning needed - context within limits")
	}
}

// handleUserInput processes a user message and gets model response
func (s *CLISession) handleUserInput(userInput string) error {
	// Check if we should prune before adding new input
	if s.contextManager.ShouldPrune(s.messages) {
		fmt.Println("Auto-pruning context due to token limit...")
		s.pruneContext()
	}

	s.messages = append(s.messages, backend.Message{Role: backend.RoleUser, Content: userInput})

	// Determine prompt type for metrics using switch statement
	promptType := "general"
	lowerInput := strings.ToLower(userInput)
	switch {
	case strings.Contains(lowerInput, "code") || strings.Contains(lowerInput, "debug"):
		promptType = "code_help"
	case strings.Contains(lowerInput, "explain") || strings.Contains(lowerInput, "how"):
		promptType = "explanation"
	case strings.Contains(lowerInput, "write") || strings.Contains(lowerInput, "create"):
		promptType = "creative"
	}

	startTime := time.Now()
	var (
		reply string
		usage *backend.Usage
		err   error
	)

	switch s.cfg.ShowUsage {
	case true:
		reply, usage, err = backend.ChatWithLLMWithUsage(s.cfg, s.messages)
	default:
		reply, err = backend.ChatWithLLM(s.cfg, s.messages)
	}

	responseTime := time.Since(startTime)

	// Log the interaction
	errorType := ""
	if err != nil {
		errorType = "api_error"
	}
	s.metrics.LogInteraction(usage, responseTime, err == nil, errorType, promptType)

	if err != nil {
		fmt.Println("Error:", err)
		// Remove the failed user message
		if len(s.messages) > 0 && s.messages[len(s.messages)-1].Role == backend.RoleUser {
			s.messages = s.messages[:len(s.messages)-1]
		}
		return err
	}

	fmt.Println("\nLLM:\n" + reply + "\n")

	// Show token usage if enabled
	if usage != nil {
		fmt.Printf("[Tokens: prompt=%d, completion=%d, total=%d | Response: %dms]\n",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, responseTime.Milliseconds())

		// Check budget status after each response
		status := s.metrics.CheckBudgetStatus()
		if len(status.Warnings) > 0 {
			fmt.Printf("Budget: %s\n", status.Warnings[0])
		}
	}

	s.messages = append(s.messages, backend.Message{Role: backend.RoleAssistant, Content: reply})
	return nil
}

// Run starts the enhanced CLI mode with metrics and budget tracking
func Run(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) error {
	printMOTD()
	fmt.Println("Welcome to the interactive LLM chat!")
	fmt.Println("Commands: 'exit', '/reset', '/system', '/budget', '/stats', '/prune'")
	fmt.Println()

	// Initialize session
	sessionID := fmt.Sprintf("cli_%d", time.Now().Unix())

	metrics, err := backend.NewMetricsLogger(sessionID, "cli_session", budgetCfg)
	if err != nil {
		fmt.Printf("Warning: Could not initialize metrics logging: %v\n", err)
	}
	defer func() {
		if metrics != nil {
			summary := metrics.GetSessionSummary()
			fmt.Printf("\nSession Summary: %d requests, %.1f%% success, $%.4f cost, %v duration\n",
				summary.TotalRequests, summary.SuccessRate*100, summary.EstimatedCost, summary.Duration.Round(time.Second))
			metrics.Close()
		}
	}()

	contextManager := backend.NewContextManager(6000, 3, true) // 6k tokens, keep 3 recent exchanges, enable summaries

	session := &CLISession{
		cfg:            cfg,
		metrics:        metrics,
		contextManager: contextManager,
		reader:         bufio.NewReader(os.Stdin),
		systemPrompt:   "You are a helpful assistant.",
		sessionID:      sessionID,
	}
	session.resetMessages()

	for {
		userInput, inputErr := session.readMultilineInput()
		if inputErr != nil {
			switch inputErr.Error() {
			case "EOF":
				fmt.Println("\nThanks for using chatGBT! Goodbye!")
				return nil
			default:
				fmt.Println("Error reading input:", inputErr)
				continue
			}
		}

		switch userInput {
		case cmdExit:
			fmt.Println("\nThanks for using chatGBT! Goodbye!")
			return nil
		case cmdReset:
			fmt.Println("Conversation reset.")
			session.resetMessages()
			continue
		case cmdSystem:
			if err := session.handleSystemPromptUpdate(); err != nil {
				fmt.Println("Error reading system prompt:", err)
			}
			continue
		case cmdBudget:
			session.showBudgetStatus()
			continue
		case cmdStats:
			session.showContextStats()
			continue
		case cmdPrune:
			session.pruneContext()
			continue
		case "":
			continue
		}

		if err := session.handleUserInput(userInput); err != nil {
			// Error already handled in handleUserInput
			continue
		}
	}
}

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nleiva/chatgbt/app"
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

// CLIHandler handles the CLI-specific UI interactions and session management
type CLIHandler struct {
	session *app.ChatSession
	reader  *bufio.Reader
}

// NewCLIHandler creates a new CLI handler with the configured session
func NewCLIHandler(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) (*CLIHandler, error) {
	sessionID := app.GenerateSessionID("cli")
	session, err := app.NewChatSessionWithDefaults(
		sessionID,
		"cli_session",
		"You are a helpful assistant.",
		cfg,
		budgetCfg,
	)
	if err != nil {
		return nil, err
	}

	return &CLIHandler{
		session: session,
		reader:  bufio.NewReader(os.Stdin),
	}, nil
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

// readMultilineInput reads user input until an empty line is entered
func (h *CLIHandler) readMultilineInput() (string, error) {
	fmt.Println("You (end with empty line):")
	var userLines []string
	for {
		line, err := h.reader.ReadString('\n')
		if err != nil {
			switch err.Error() {
			case "EOF":
				if len(userLines) == 0 {
					return "", err
				}
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
func (h *CLIHandler) handleSystemPromptUpdate() error {
	fmt.Print("Enter new system prompt: ")
	newPrompt, err := h.reader.ReadString('\n')
	if err != nil {
		return err
	}
	h.session.UpdateSystemPrompt(strings.TrimSpace(newPrompt))
	fmt.Println("System prompt updated.")
	return nil
}

// showBudgetStatus displays current budget and usage information
func (h *CLIHandler) showBudgetStatus() {
	status := h.session.GetBudgetStatus()

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
func (h *CLIHandler) showContextStats() {
	stats := h.session.GetContextStats()

	fmt.Println("\nContext Statistics:")
	fmt.Printf("   Messages: %d total (%d user, %d assistant, %d system)\n",
		stats.TotalMessages, stats.UserMessages, stats.AssistantMessages, stats.SystemMessages)
	fmt.Printf("   Estimated Tokens: %d / %d (%.1f%%)\n",
		stats.EstimatedTokens, stats.TokenLimit, stats.UtilizationPct)

	if stats.ShouldPrune {
		fmt.Println("   Warning: Context approaching token limit")
	}

	// Show session summary
	summary := h.session.GetSessionSummary()
	fmt.Printf("   Session Duration: %v\n", summary.Duration.Round(time.Second))
	fmt.Printf("   Requests: %d (%.1f%% success rate)\n",
		summary.TotalRequests, summary.SuccessRate*100)
	if summary.AvgResponseTime > 0 {
		fmt.Printf("   Avg Response Time: %dms\n", summary.AvgResponseTime)
	}

	// Show prompt type breakdown
	promptBreakdown := h.session.GetPromptTypeBreakdown()
	if len(promptBreakdown) > 0 {
		fmt.Println("\nPrompt Type Breakdown:")
		for promptType, count := range promptBreakdown {
			percentage := float64(count) / float64(summary.TotalRequests) * 100
			fmt.Printf("   %s: %d (%.1f%%)\n", promptType, count, percentage)
		}
	}

	fmt.Println()
}

// pruneContext manually triggers context pruning
func (h *CLIHandler) pruneContext() {
	beforeStats := h.session.GetContextStats()
	pruned := h.session.AutoPrune()

	if pruned {
		afterStats := h.session.GetContextStats()
		fmt.Printf("Context pruned: %d -> %d messages, ~%d -> ~%d tokens\n",
			beforeStats.TotalMessages, afterStats.TotalMessages,
			beforeStats.EstimatedTokens, afterStats.EstimatedTokens)
	} else {
		fmt.Println("No pruning needed - context within limits")
	}
}

// handleUserInput processes a user message and gets model response
func (h *CLIHandler) handleUserInput(userInput string) error {
	response, err := h.session.ProcessUserMessage(userInput)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Println("\nLLM:\n" + response.Content + "\n")

	// Show token usage if available
	if response.Usage != nil {
		fmt.Printf("[Tokens: prompt=%d, completion=%d, total=%d | Response: %dms]\n",
			response.Usage.PromptTokens, response.Usage.CompletionTokens,
			response.Usage.TotalTokens, response.ResponseTime.Milliseconds())

		// Show budget warnings if any
		if len(response.Warnings) > 0 {
			fmt.Printf("Budget: %s\n", response.Warnings[0])
		}
	}

	return nil
}

// Run starts the enhanced CLI mode with the new architecture
func (h *CLIHandler) Run() error {
	printMOTD()
	fmt.Println("Welcome to the interactive LLM chat!")
	fmt.Println("Commands: 'exit', '/reset', '/system', '/budget', '/stats', '/prune'")
	fmt.Println()

	for {
		userInput, inputErr := h.readMultilineInput()
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
			h.session.Reset("")
		case cmdSystem:
			if err := h.handleSystemPromptUpdate(); err != nil {
				fmt.Println("Error reading system prompt:", err)
			}
		case cmdBudget:
			h.showBudgetStatus()
		case cmdStats:
			h.showContextStats()
		case cmdPrune:
			h.pruneContext()
		case "":
			// Empty input, continue to next iteration
		default:
			// Handle user input for chat
			if err := h.handleUserInput(userInput); err != nil {
				// Error already handled in handleUserInput
			}
		}
	}
}

// Close properly closes the CLI handler and session
func (h *CLIHandler) Close() error {
	if h.session != nil {
		summary := h.session.GetSessionSummary()
		fmt.Printf("\nSession Summary: %d requests, %.1f%% success, $%.4f cost, %v duration\n",
			summary.TotalRequests, summary.SuccessRate*100, summary.EstimatedCost, summary.Duration.Round(time.Second))
		return h.session.Close()
	}
	return nil
}

// Run is the main entry point for CLI mode
func Run(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) error {
	handler, err := NewCLIHandler(cfg, budgetCfg)
	if err != nil {
		return fmt.Errorf("failed to create CLI handler: %w", err)
	}

	defer func() {
		if closeErr := handler.Close(); closeErr != nil {
			fmt.Printf("Warning: Error closing CLI handler: %v\n", closeErr)
		}
	}()

	return handler.Run()
}

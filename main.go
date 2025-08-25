package main

import (
	"cmp"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nleiva/chatgbt/backend"
	"github.com/nleiva/chatgbt/cli"
	"github.com/nleiva/chatgbt/web"
)

const (
	// Default configuration
	defaultModel = "gpt-3.5-turbo"
	defaultURL   = "https://api.openai.com/v1/chat/completions"
	defaultPort  = 3000
)

// showUsage displays the usage information
func showUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <mode> [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nModes:\n")
	fmt.Fprintf(os.Stderr, "  cli           Start in CLI mode (interactive terminal)\n")
	fmt.Fprintf(os.Stderr, "  web           Start in web mode (HTTP server)\n")
	fmt.Fprintf(os.Stderr, "  \"<query>\"     Quick query mode (non-interactive)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
	fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY   Required: Your OpenAI API key\n")
	fmt.Fprintf(os.Stderr, "  MODEL           Optional: Model to use (default: gpt-3.5-turbo)\n")
	fmt.Fprintf(os.Stderr, "  PORT            Optional: Web server port number (default: 3000)\n")
	fmt.Fprintf(os.Stderr, "  TOKEN_BUDGET    Optional: Session token budget (default: 10000)\n")
	fmt.Fprintf(os.Stderr, "  COST_BUDGET     Optional: Session cost budget in USD (default: $0.02)\n")
}

// createBudgetConfig reads budget configuration from environment variables
func createBudgetConfig(w io.Writer) backend.TokenBudgetConfig {
	cfg := backend.DefaultBudgetConfig()

	// Read TOKEN_BUDGET environment variable
	switch tokenBudgetStr := os.Getenv("TOKEN_BUDGET"); tokenBudgetStr {
	case "":
		// No TOKEN_BUDGET set, use default
	default:
		tokenBudget, err := strconv.Atoi(tokenBudgetStr)
		switch err {
		case nil:
			cfg.SessionLimit = tokenBudget
		default:
			fmt.Fprintf(w, "Warning: Invalid TOKEN_BUDGET value '%s', using default %d\n",
				tokenBudgetStr, cfg.SessionLimit)
		}
	}

	// Read COST_BUDGET environment variable
	switch costBudgetStr := os.Getenv("COST_BUDGET"); costBudgetStr {
	case "":
		// No COST_BUDGET set, use default
	default:
		costBudget, err := strconv.ParseFloat(costBudgetStr, 64)
		switch err {
		case nil:
			// Calculate session limit based on cost budget and cost per token
			cfg.SessionLimit = int(costBudget / cfg.CostPerToken)
		default:
			fmt.Fprintf(w, "Warning: Invalid COST_BUDGET value '%s', using default\n", costBudgetStr)
		}
	}

	return cfg
}

// parsePort reads and validates the PORT environment variable
func parsePort(w io.Writer) int {
	switch portStr := os.Getenv("PORT"); portStr {
	case "":
		return defaultPort
	default:
		port, err := strconv.Atoi(portStr)
		switch err {
		case nil:
			return cmp.Or(port, defaultPort) // Ensure non-zero port
		default:
			fmt.Fprintf(w, "Warning: Invalid PORT value '%s', using default %d\n", portStr, defaultPort)
			return defaultPort
		}
	}
}

func createConfig() (backend.LLMConfig, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return backend.LLMConfig{}, fmt.Errorf("missing API key. Please set the OPENAI_API_KEY environment variable")
	}

	// Get model from environment or use default
	model := os.Getenv("MODEL")
	if model == "" {
		model = defaultModel
	}

	return backend.LLMConfig{
		APIKey:    apiKey,
		URL:       defaultURL,
		Model:     model,
		ShowUsage: true, // Set to false to use simple reply only
	}, nil
}

// runDirectQuery handles single-query mode for quick interactions
func runDirectQuery(w io.Writer, cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig, query string) error {
	// Create a simple conversation with just the user query
	messages := []backend.Message{
		{Role: backend.RoleUser, Content: query},
	}

	// Create metrics logger for tracking usage
	metricsLogger, err := backend.NewMetricsLogger("direct_query", "quick", budgetCfg)
	if err != nil {
		return fmt.Errorf("failed to create metrics logger: %w", err)
	}
	defer metricsLogger.Close()

	// Make the API call with timing
	start := time.Now()
	response, usage, err := backend.ChatWithLLMWithUsage(cfg, messages)
	responseTime := time.Since(start)

	if err != nil {
		metricsLogger.LogInteraction(nil, responseTime, false, err.Error(), "user_query")
		return fmt.Errorf("API call failed: %w", err)
	}

	// Log the successful interaction
	metricsLogger.LogInteraction(usage, responseTime, true, "", "user_query")

	// Print the response
	fmt.Fprintf(w, "%s\n", response)

	// Print usage stats if enabled
	if cfg.ShowUsage {
		summary := metricsLogger.GetSessionSummary()
		fmt.Fprintf(w, "Tokens: %d | Cost: $%.4f | Time: %.1fs\n",
			usage.TotalTokens, summary.EstimatedCost, responseTime.Seconds())
	}

	return nil
}

func run(w io.Writer, args []string) error {
	if len(args) < 2 {
		showUsage()
		return fmt.Errorf("mode argument required")
	}

	mode := args[1]

	cfg, err := createConfig()
	if err != nil {
		return err
	}

	budgetCfg := createBudgetConfig(w)

	switch mode {
	case "cli":
		// Start CLI mode
		return cli.Run(cfg, budgetCfg)
	case "web":
		// Configure web server
		port := parsePort(w)
		address := net.JoinHostPort("", strconv.Itoa(port))
		server := web.NewServer(cfg, budgetCfg)

		return server.Run(address)
	default:
		// Check if it's a direct query (starts and ends with quotes or just treat as query)
		query := mode
		if len(args) > 2 {
			// Join all remaining args as the query
			query = strings.Join(args[1:], " ")
		}

		// Handle direct query mode
		return runDirectQuery(w, cfg, budgetCfg, query)
	}
}

func main() {
	if err := run(os.Stderr, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting chatGBT: %s\n", err)
		os.Exit(1)
	}
}

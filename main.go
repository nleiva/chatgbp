package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/nleiva/chatgbt/app"
	"github.com/nleiva/chatgbt/cli"
	"github.com/nleiva/chatgbt/config"
	"github.com/nleiva/chatgbt/web"
)

// printUsage displays the usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <mode> [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nModes:\n")
	fmt.Fprintf(os.Stderr, "  cli           Start in CLI mode (interactive terminal)\n")
	fmt.Fprintf(os.Stderr, "  web           Start in web mode (HTTP server)\n")
	fmt.Fprintf(os.Stderr, "  \"<query>\"     Quick query mode (non-interactive)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
	fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY   Required: Your OpenAI API key\n")
	fmt.Fprintf(os.Stderr, "  MODEL           Optional: Model to use (default: %s)\n", config.DefaultModel)
	fmt.Fprintf(os.Stderr, "  PORT            Optional: Web server port number (default: %d)\n", config.DefaultPort)
	fmt.Fprintf(os.Stderr, "  TOKEN_BUDGET    Optional: Session token budget (default: 10000)\n")
	fmt.Fprintf(os.Stderr, "  COST_BUDGET     Optional: Session cost budget in USD (default: $0.02)\n")
}

// runDirectQuery handles single-query mode for quick interactions
func runDirectQuery(query string, cfg *config.Config) error {
	// Create LLM client
	client := app.NewLLMClient(cfg.LLM)

	// Create metrics logger
	logger, err := app.NewMetricsLogger("direct_query", "quick", cfg.Budget)
	if err != nil {
		return fmt.Errorf("failed to create metrics logger: %w", err)
	}
	defer logger.Close()

	// Create and execute the service
	service := app.NewDirectQueryService(client, logger, os.Stdout)
	ctx := context.Background()

	return service.Execute(ctx, query, cfg.LLM.ShowUsage)
}

func run(args []string) error {
	if len(args) < 2 {
		printUsage()
		return fmt.Errorf("mode argument required")
	}

	mode := args[1]

	// Load configuration from environment
	cfg, err := config.LoadFromEnv(os.Stderr)
	if err != nil {
		return err
	}

	switch mode {
	case "cli":
		// Start CLI mode
		return cli.Run(cfg.LLM, cfg.Budget)
	case "web":
		// Configure web server
		address := net.JoinHostPort("", strconv.Itoa(cfg.Port))
		server := web.NewServer(cfg.LLM, cfg.Budget)
		return server.Run(address)
	default:
		// Handle direct query mode
		query := mode
		if len(args) > 2 {
			// Join all remaining args as the query
			query = strings.Join(args[1:], " ")
		}
		return runDirectQuery(query, cfg)
	}
}

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting chatGBT: %s\n", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/nleiva/chatgbt/backend"
	"github.com/nleiva/chatgbt/cli"
	"github.com/nleiva/chatgbt/config"
	"github.com/nleiva/chatgbt/web"
)

// Mode represents a runnable application mode
type Mode interface {
	Run(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) error
}

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

func run(args []string) error {
	if len(args) < 2 {
		printUsage()
		return fmt.Errorf("mode argument required")
	}

	modeArg := args[1]

	// Load configuration from environment
	cfg, err := config.LoadFromEnv(os.Stderr)
	if err != nil {
		return err
	}

	var mode Mode

	switch modeArg {
	case "cli":
		mode = cli.NewCLIRunner()
	case "web":
		address := net.JoinHostPort("", strconv.Itoa(cfg.Port))
		mode = web.NewWebRunner(address)
	default:
		// Handle direct query mode
		query := modeArg
		if len(args) > 2 {
			// Join all remaining args as the query
			query = strings.Join(args[1:], " ")
		}
		mode = cli.NewDirectQueryRunner(query, cfg.LLM.ShowUsage)
	}

	return mode.Run(cfg.LLM, cfg.Budget)
}

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting chatGBT: %s\n", err)
		os.Exit(1)
	}
}

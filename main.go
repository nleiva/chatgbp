package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/nleiva/chatgbt/cli"
	"github.com/nleiva/chatgbt/config"
	"github.com/nleiva/chatgbt/web"
)

// Mode represents a runnable application mode
type Mode interface {
	Run() error
}

// CLI wraps the CLI functionality
type CLI struct {
	cfg *config.Config
}

func NewCLI(cfg *config.Config) *CLI {
	return &CLI{cfg: cfg}
}

func (c *CLI) Run() error {
	return cli.Run(c.cfg.LLM, c.cfg.Budget)
}

// Web wraps the web server functionality
type Web struct {
	cfg *config.Config
}

func NewWeb(cfg *config.Config) *Web {
	return &Web{cfg: cfg}
}

func (w *Web) Run() error {
	address := net.JoinHostPort("", strconv.Itoa(w.cfg.Port))
	server := web.NewServer(w.cfg.LLM, w.cfg.Budget)
	return server.Run(address)
}

// DirectQuery wraps the direct query functionality
type DirectQuery struct {
	query string
	cfg   *config.Config
}

func NewDirectQuery(query string, cfg *config.Config) *DirectQuery {
	return &DirectQuery{query: query, cfg: cfg}
}

func (d *DirectQuery) Run() error {
	return cli.RunDirect(d.query, d.cfg.LLM, d.cfg.Budget, d.cfg.LLM.ShowUsage)
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
		mode = NewCLI(cfg)
	case "web":
		mode = NewWeb(cfg)
	default:
		// Handle direct query mode
		query := modeArg
		if len(args) > 2 {
			// Join all remaining args as the query
			query = strings.Join(args[1:], " ")
		}
		mode = NewDirectQuery(query, cfg)
	}

	return mode.Run()
}

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting chatGBT: %s\n", err)
		os.Exit(1)
	}
}

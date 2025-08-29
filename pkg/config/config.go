package config

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/nleiva/chatgbt/pkg/backend"
)

const (
	// Default configuration values
	DefaultModel    = "gpt-3.5-turbo"
	DefaultURL      = "https://api.openai.com/v1/chat/completions"
	DefaultPort     = 3000
	DefaultProvider = "openai"
)

// Config holds all application configuration for the chatgbt application.
// It combines LLM configuration, budget settings, and server configuration
// into a single struct for easy management.
type Config struct {
	LLM    backend.LLMConfig         // LLM client configuration
	Budget backend.TokenBudgetConfig // Token usage and cost limits
	Port   int                       // HTTP server port for web mode
}

// Validate checks the configuration for correctness
func (c *Config) Validate() error {
	if c.LLM.APIKey == "" {
		return fmt.Errorf("API_KEY is required")
	}
	if c.LLM.URL == "" {
		return fmt.Errorf("API URL cannot be empty")
	}
	if c.LLM.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1-65535, got %d", c.Port)
	}
	if c.Budget.SessionLimit <= 0 {
		return fmt.Errorf("session limit must be positive, got %d", c.Budget.SessionLimit)
	}
	return nil
}

// LoadFromEnv loads configuration from environment variables and returns
// a fully configured Config struct. It writes warnings to w for any
// invalid environment variable values encountered.
func LoadFromEnv(w io.Writer) (*Config, error) {
	llmCfg, err := loadLLMConfig()
	if err != nil {
		return nil, err
	}

	budgetCfg := loadBudgetConfig(w)
	port := loadPort(w)

	config := &Config{
		LLM:    llmCfg,
		Budget: budgetCfg,
		Port:   port,
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadLLMConfig creates LLM configuration from environment variables
func loadLLMConfig() (backend.LLMConfig, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return backend.LLMConfig{}, fmt.Errorf("missing API key: please set the API_KEY environment variable")
	}

	model := os.Getenv("MODEL")
	if model == "" {
		model = DefaultModel
	}

	// For now, we're defaulting to OpenAI, but this can be extended later
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = DefaultProvider
	}

	return backend.LLMConfig{
		APIKey:    apiKey,
		URL:       DefaultURL,
		Model:     model,
		Provider:  backend.ProviderName(provider),
		ShowUsage: true,
	}, nil
}

// loadBudgetConfig reads budget configuration from environment variables
func loadBudgetConfig(w io.Writer) backend.TokenBudgetConfig {
	cfg := backend.DefaultBudgetConfig()

	if err := loadTokenBudget(&cfg, w); err != nil {
		fmt.Fprintf(w, "Warning: %v\n", err)
	}

	if err := loadCostBudget(&cfg, w); err != nil {
		fmt.Fprintf(w, "Warning: %v\n", err)
	}

	return cfg
}

// loadTokenBudget reads and validates TOKEN_BUDGET environment variable
func loadTokenBudget(cfg *backend.TokenBudgetConfig, w io.Writer) error {
	tokenBudgetStr := os.Getenv("TOKEN_BUDGET")
	if tokenBudgetStr == "" {
		return nil // Use default
	}

	tokenBudget, err := strconv.Atoi(tokenBudgetStr)
	if err != nil {
		return fmt.Errorf("invalid TOKEN_BUDGET value '%s': %w, using default %d",
			tokenBudgetStr, err, cfg.SessionLimit)
	}

	if tokenBudget <= 0 {
		return fmt.Errorf("TOKEN_BUDGET must be positive, got %d, using default %d",
			tokenBudget, cfg.SessionLimit)
	}

	cfg.SessionLimit = tokenBudget
	return nil
}

// loadCostBudget reads and validates COST_BUDGET environment variable
func loadCostBudget(cfg *backend.TokenBudgetConfig, w io.Writer) error {
	costBudgetStr := os.Getenv("COST_BUDGET")
	if costBudgetStr == "" {
		return nil // Use default
	}

	costBudget, err := strconv.ParseFloat(costBudgetStr, 64)
	if err != nil {
		return fmt.Errorf("invalid COST_BUDGET value '%s': %w", costBudgetStr, err)
	}

	if costBudget <= 0 {
		return fmt.Errorf("COST_BUDGET must be positive, got %.4f", costBudget)
	}

	// Calculate session limit based on cost budget and cost per token
	cfg.SessionLimit = int(costBudget / cfg.CostPerToken)
	return nil
}

// loadPort reads and validates the PORT environment variable
func loadPort(w io.Writer) int {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		return DefaultPort
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Fprintf(w, "Warning: Invalid PORT value '%s': %v, using default %d\n",
			portStr, err, DefaultPort)
		return DefaultPort
	}

	if port <= 0 || port > 65535 {
		fmt.Fprintf(w, "Warning: PORT must be between 1-65535, got %d, using default %d\n",
			port, DefaultPort)
		return DefaultPort
	}

	return port
}

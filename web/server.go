package web

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/nleiva/chatgbt/backend"
	"github.com/nleiva/chatgbt/web/templates"
)

const (
	defaultSystemPrompt = "You are a helpful assistant."
	defaultPort         = ":3000"
	htmlContentType     = "text/html; charset=utf-8"
)

type Server struct {
	app            *fiber.App
	cfg            backend.LLMConfig
	budgetCfg      backend.TokenBudgetConfig
	messages       []backend.Message
	metrics        *backend.MetricsLogger
	contextManager *backend.ContextManager
	sessionID      string
}

type ChatSession struct {
	Messages []backend.Message `json:"messages"`
}

// NewServer creates a new web server instance with budget tracking
func NewServer(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())

	// Initialize session
	sessionID := fmt.Sprintf("web_%d", time.Now().Unix())

	// Initialize metrics logger
	metrics, err := backend.NewMetricsLogger(sessionID, "web_session", budgetCfg)
	if err != nil {
		log.Printf("Warning: Could not initialize metrics logging: %v", err)
	}

	// Initialize context manager
	contextManager := backend.NewContextManager(6000, 3, true) // 6k tokens, keep 3 recent exchanges, enable summaries

	server := &Server{
		app:            app,
		cfg:            cfg,
		budgetCfg:      budgetCfg,
		messages:       []backend.Message{{Role: backend.RoleSystem, Content: defaultSystemPrompt}},
		metrics:        metrics,
		contextManager: contextManager,
		sessionID:      sessionID,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Serve static files
	s.app.Static("/static", "./web/static")

	// Favicon handler
	s.app.Get("/favicon.ico", s.handleFavicon)

	// Main chat page
	s.app.Get("/", s.handleHome)

	// API endpoints
	s.app.Post("/chat", s.handleChat)
	s.app.Post("/reset", s.handleReset)
	s.app.Post("/system", s.handleSystemPrompt)
	s.app.Get("/status", s.handleStatus)
}

func (s *Server) handleHome(c *fiber.Ctx) error {
	c.Set("Content-Type", htmlContentType)
	return s.renderComponent(c, templates.ChatPage())
}

// renderComponent is a helper to render templ components
func (s *Server) renderComponent(c *fiber.Ctx, component templ.Component) error {
	return component.Render(c.Context(), c.Response().BodyWriter())
}

// resetToSystemPrompt resets conversation with given system prompt
func (s *Server) resetToSystemPrompt(prompt string) {
	s.messages = []backend.Message{{Role: backend.RoleSystem, Content: prompt}}
}

// removeLastUserMessage removes the last user message from conversation history
func (s *Server) removeLastUserMessage() {
	if len(s.messages) > 0 && s.messages[len(s.messages)-1].Role == backend.RoleUser {
		s.messages = s.messages[:len(s.messages)-1]
	}
}

func (s *Server) handleFavicon(c *fiber.Ctx) error {
	// Return a simple 204 No Content for favicon requests
	// This prevents the 404 error without needing an actual favicon file
	return c.SendStatus(204)
}

func (s *Server) handleChat(c *fiber.Ctx) error {
	userMessage := c.FormValue("message")
	if userMessage == "" {
		return c.Status(400).SendString("Message is required")
	}

	// Check if we should prune before adding new input
	if s.contextManager != nil && s.contextManager.ShouldPrune(s.messages) {
		log.Println("Auto-pruning context due to token limit...")
		newMessages, pruned := s.contextManager.PruneContext(s.messages, s.contextManager.EstimateTokens(s.messages))
		if pruned {
			s.messages = newMessages
		}
	}

	// Add user message to conversation
	s.messages = append(s.messages, backend.Message{
		Role:    backend.RoleUser,
		Content: userMessage,
	})

	// Determine prompt type for metrics
	promptType := "general"
	lowerInput := strings.ToLower(userMessage)
	switch {
	case strings.Contains(lowerInput, "code") || strings.Contains(lowerInput, "debug"):
		promptType = "code_help"
	case strings.Contains(lowerInput, "explain") || strings.Contains(lowerInput, "how"):
		promptType = "explanation"
	case strings.Contains(lowerInput, "write") || strings.Contains(lowerInput, "create"):
		promptType = "creative"
	}

	// Get AI response with timing
	startTime := time.Now()
	var reply string
	var usage *backend.Usage
	var err error

	switch s.cfg.ShowUsage {
	case true:
		reply, usage, err = backend.ChatWithLLMWithUsage(s.cfg, s.messages)
	default:
		reply, err = backend.ChatWithLLM(s.cfg, s.messages)
	}

	responseTime := time.Since(startTime)

	// Log the interaction
	if s.metrics != nil {
		errorType := ""
		if err != nil {
			errorType = "api_error"
		}
		s.metrics.LogInteraction(usage, responseTime, err == nil, errorType, promptType)
	}

	if err != nil {
		// Remove user message from history on error
		s.removeLastUserMessage()

		// Show error message
		return s.renderComponent(c, templates.MessageComponent(string(backend.RoleAssistant), "Error: "+err.Error()))
	}

	// Add assistant message to conversation
	s.messages = append(s.messages, backend.Message{
		Role:    backend.RoleAssistant,
		Content: reply,
	})

	// Prepare warning message if any
	var warningMsg string
	if s.metrics != nil {
		status := s.metrics.CheckBudgetStatus()
		if len(status.Warnings) > 0 {
			warningMsg = fmt.Sprintf("⚠️ %s", status.Warnings[0])
		}
	}

	// Render everything as a single response
	return s.renderComponent(c, templates.ChatResponseComponent(
		userMessage,
		reply,
		usage,
		responseTime.Milliseconds(),
		warningMsg,
	))
}

func (s *Server) handleReset(c *fiber.Ctx) error {
	s.resetToSystemPrompt(defaultSystemPrompt)

	// Return the welcome screen HTML
	return c.SendString(`<div class="welcome-screen">
		<h2>How can I help you today?</h2>
		<p>I'm ChatGBT, your AI assistant. Ask me anything, and I'll do my best to help you with information, analysis, creative tasks, and more.</p>
	</div>`)
}

func (s *Server) handleSystemPrompt(c *fiber.Ctx) error {
	newPrompt := c.FormValue("prompt")
	if newPrompt == "" {
		return c.Status(400).SendString("System prompt is required")
	}

	s.resetToSystemPrompt(newPrompt)

	return c.SendString(`<div class="message system">
		<div class="message-role">system</div>
		<div class="message-content">System prompt updated.</div>
	</div>`)
}

// handleStatus returns budget and session status as JSON
func (s *Server) handleStatus(c *fiber.Ctx) error {
	if s.metrics == nil {
		return c.JSON(fiber.Map{
			"error": "Metrics not available",
		})
	}

	status := s.metrics.CheckBudgetStatus()
	summary := s.metrics.GetSessionSummary()

	var contextStats map[string]interface{}
	if s.contextManager != nil {
		stats := s.contextManager.GetContextStats(s.messages)
		contextStats = map[string]interface{}{
			"total_messages":     stats.TotalMessages,
			"user_messages":      stats.UserMessages,
			"assistant_messages": stats.AssistantMessages,
			"system_messages":    stats.SystemMessages,
			"estimated_tokens":   stats.EstimatedTokens,
			"token_limit":        stats.TokenLimit,
			"utilization_pct":    stats.UtilizationPct,
			"should_prune":       stats.ShouldPrune,
		}
	}

	return c.JSON(fiber.Map{
		"budget": fiber.Map{
			"session_tokens": status.SessionTokens,
			"session_limit":  status.SessionLimit,
			"session_cost":   status.SessionCost,
			"warnings":       status.Warnings,
			"should_prune":   status.ShouldPrune,
		},
		"session": fiber.Map{
			"total_requests":    summary.TotalRequests,
			"success_rate":      summary.SuccessRate,
			"estimated_cost":    summary.EstimatedCost,
			"duration_seconds":  summary.Duration.Seconds(),
			"avg_response_time": summary.AvgResponseTime,
		},
		"context": contextStats,
	})
}

// Run starts the web server with graceful shutdown
func (s *Server) Run(port string) error {
	if port == "" {
		port = defaultPort
	}

	// Setup graceful shutdown
	if s.metrics != nil {
		defer func() {
			summary := s.metrics.GetSessionSummary()
			log.Printf("Session Summary: %d requests, %.1f%% success, $%.4f cost, %v duration",
				summary.TotalRequests, summary.SuccessRate*100, summary.EstimatedCost, summary.Duration)
			s.metrics.Close()
		}()
	}

	log.Printf("Starting web server on http://localhost%s", port)
	log.Printf("Budget: %d tokens, cost limit: $%.4f", s.budgetCfg.SessionLimit,
		float64(s.budgetCfg.SessionLimit)*s.budgetCfg.CostPerToken)
	return s.app.Listen(port)
}

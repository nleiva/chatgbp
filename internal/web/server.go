package web

import (
	"fmt"
	"log"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/nleiva/chatgbt/internal/app"
	"github.com/nleiva/chatgbt/internal/web/templates"
	"github.com/nleiva/chatgbt/pkg/backend"
)

const (
	defaultSystemPrompt = "You are a helpful assistant."
	defaultAddress      = ":3000"
	htmlContentType     = "text/html; charset=utf-8"
	sessionCookieName   = "chatgbt_session_id"
	sessionMaxAge       = 24 * time.Hour
)

// Server represents the web server with session management
type Server struct {
	app            *fiber.App
	sessionManager app.SessionManager
}

// WebRunner handles web server mode with consistent signature
type WebRunner struct {
	address string
}

// NewWebRunner creates a new web runner for the specified address
func NewWebRunner(address string) *WebRunner {
	return &WebRunner{address: address}
}

// Run starts the web server with the provided configuration
func (w *WebRunner) Run(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) error {
	server := NewServer(cfg, budgetCfg)
	return server.Run(w.address)
}

// NewServer creates a new web server instance with session management
func NewServer(cfg backend.LLMConfig, budgetCfg backend.TokenBudgetConfig) *Server {
	fiberApp := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})

	// Middleware
	fiberApp.Use(logger.New())
	fiberApp.Use(recover.New())

	// Initialize session manager
	sessionManager := app.NewInMemorySessionManager(cfg, budgetCfg, sessionMaxAge)

	server := &Server{
		app:            fiberApp,
		sessionManager: sessionManager,
	}

	server.setupRoutes()

	// Start cleanup routine for expired sessions
	go server.startSessionCleanup()

	return server
}

// startSessionCleanup runs a background cleanup routine for expired sessions
func (s *Server) startSessionCleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleaned := s.sessionManager.CleanupExpiredSessions()
		if cleaned > 0 {
			log.Printf("Cleaned up %d expired sessions", cleaned)
		}
	}
}

// getOrCreateSession gets an existing session or creates a new one for the user
func (s *Server) getOrCreateSession(c *fiber.Ctx) (*app.ChatSession, error) {
	sessionID := c.Cookies(sessionCookieName)

	if sessionID != "" {
		if session, err := s.sessionManager.GetSession(sessionID); err == nil {
			return session, nil
		}
	}

	// Create new session
	session, err := s.sessionManager.CreateSession("web_user")
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		MaxAge:   int(sessionMaxAge.Seconds()),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return session, nil
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

func (s *Server) handleFavicon(c *fiber.Ctx) error {
	// Return a simple 204 No Content for favicon requests
	return c.SendStatus(204)
}

func (s *Server) handleChat(c *fiber.Ctx) error {
	session, err := s.getOrCreateSession(c)
	if err != nil {
		return c.Status(500).SendString("Failed to get session: " + err.Error())
	}

	userMessage := c.FormValue("message")
	if userMessage == "" {
		return c.Status(400).SendString("Message is required")
	}

	// Process the user message using the session
	response, err := session.ProcessUserMessage(userMessage)
	if err != nil {
		// Show error message
		return s.renderComponent(c, templates.MessageComponent(string(backend.RoleAssistant), "Error: "+err.Error()))
	}

	// Prepare warning message if any
	var warningMsg string
	if len(response.Warnings) > 0 {
		warningMsg = fmt.Sprintf("⚠️ %s", response.Warnings[0])
	}

	// Render everything as a single response
	return s.renderComponent(c, templates.ChatResponseComponent(
		userMessage,
		response.Content,
		response.Usage,
		response.ResponseTime.Milliseconds(),
		warningMsg,
	))
}

func (s *Server) handleReset(c *fiber.Ctx) error {
	session, err := s.getOrCreateSession(c)
	if err != nil {
		return c.Status(500).SendString("Failed to get session: " + err.Error())
	}

	session.Reset(defaultSystemPrompt)

	// Return the welcome screen HTML
	return c.SendString(`<div class="welcome-screen">
		<h2>How can I help you today?</h2>
		<p>I'm ChatGBT, your AI assistant. Ask me anything, and I'll do my best to help you with information, analysis, creative tasks, and more.</p>
	</div>`)
}

func (s *Server) handleSystemPrompt(c *fiber.Ctx) error {
	session, err := s.getOrCreateSession(c)
	if err != nil {
		return c.Status(500).SendString("Failed to get session: " + err.Error())
	}

	newPrompt := c.FormValue("prompt")
	if newPrompt == "" {
		return c.Status(400).SendString("System prompt is required")
	}

	session.UpdateSystemPrompt(newPrompt)

	return c.SendString(`<div class="message system">
		<div class="message-role">system</div>
		<div class="message-content">System prompt updated.</div>
	</div>`)
}

// handleStatus returns budget and session status as JSON
func (s *Server) handleStatus(c *fiber.Ctx) error {
	session, err := s.getOrCreateSession(c)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get session: " + err.Error(),
		})
	}

	budgetStatus := session.GetBudgetStatus()
	sessionSummary := session.GetSessionSummary()
	contextStats := session.GetContextStats()

	return c.JSON(fiber.Map{
		"budget": fiber.Map{
			"session_tokens": budgetStatus.SessionTokens,
			"session_limit":  budgetStatus.SessionLimit,
			"session_cost":   budgetStatus.SessionCost,
			"warnings":       budgetStatus.Warnings,
			"should_prune":   budgetStatus.ShouldPrune,
		},
		"session": fiber.Map{
			"total_requests":    sessionSummary.TotalRequests,
			"success_rate":      sessionSummary.SuccessRate,
			"estimated_cost":    sessionSummary.EstimatedCost,
			"duration_seconds":  sessionSummary.Duration.Seconds(),
			"avg_response_time": sessionSummary.AvgResponseTime,
		},
		"context": fiber.Map{
			"total_messages":     contextStats.TotalMessages,
			"user_messages":      contextStats.UserMessages,
			"assistant_messages": contextStats.AssistantMessages,
			"system_messages":    contextStats.SystemMessages,
			"estimated_tokens":   contextStats.EstimatedTokens,
			"token_limit":        contextStats.TokenLimit,
			"utilization_pct":    contextStats.UtilizationPct,
			"should_prune":       contextStats.ShouldPrune,
		},
	})
}

// Run starts the web server with graceful shutdown
func (s *Server) Run(address string) error {
	if address == "" {
		address = defaultAddress
	}

	log.Printf("Starting web server on http://localhost%s", address)
	log.Printf("Session management: enabled with %v max age", sessionMaxAge)

	return s.app.Listen(address)
}

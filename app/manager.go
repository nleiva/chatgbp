package app

import (
	"sync"
	"time"

	"github.com/nleiva/chatgbt/backend"
)

// SessionManager handles creation and lifecycle of chat sessions
type SessionManager interface {
	CreateSession(userID string) (*ChatSession, error)
	GetSession(sessionID string) (*ChatSession, error)
	CloseSession(sessionID string) error
	CleanupExpiredSessions() int
}

// InMemorySessionManager implements SessionManager with in-memory storage
type InMemorySessionManager struct {
	sessions     map[string]*ChatSession
	sessionAge   map[string]time.Time
	mutex        sync.RWMutex
	llmConfig    backend.LLMConfig
	budgetConfig backend.TokenBudgetConfig
	maxAge       time.Duration
}

// NewInMemorySessionManager creates a new session manager
func NewInMemorySessionManager(llmConfig backend.LLMConfig, budgetConfig backend.TokenBudgetConfig, maxAge time.Duration) *InMemorySessionManager {
	return &InMemorySessionManager{
		sessions:     make(map[string]*ChatSession),
		sessionAge:   make(map[string]time.Time),
		llmConfig:    llmConfig,
		budgetConfig: budgetConfig,
		maxAge:       maxAge,
	}
}

// CreateSession creates a new chat session for a user
func (sm *InMemorySessionManager) CreateSession(userID string) (*ChatSession, error) {
	sessionID := GenerateSessionID(userID)

	config := SessionConfig{
		ID:               sessionID,
		ConversationType: "web",
		SystemPrompt:     "You are ChatGBT, a helpful AI assistant.",
		LLMConfig:        sm.llmConfig,
		BudgetConfig:     sm.budgetConfig,
		MaxTokens:        8000,
		KeepRecent:       10,
		SummaryEnabled:   true,
	}

	session, err := NewChatSession(config)
	if err != nil {
		return nil, NewSessionError("failed to create session", sessionID, err)
	}

	sm.mutex.Lock()
	sm.sessions[sessionID] = session
	sm.sessionAge[sessionID] = time.Now()
	sm.mutex.Unlock()

	return session, nil
}

// GetSession retrieves an existing session
func (sm *InMemorySessionManager) GetSession(sessionID string) (*ChatSession, error) {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return nil, NewSessionError("session not found", sessionID, nil)
	}

	// Update last access time
	sm.mutex.Lock()
	sm.sessionAge[sessionID] = time.Now()
	sm.mutex.Unlock()

	return session, nil
}

// CloseSession closes and removes a session
func (sm *InMemorySessionManager) CloseSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return NewSessionError("session not found", sessionID, nil)
	}

	if err := session.Close(); err != nil {
		return NewSessionError("failed to close session", sessionID, err)
	}

	delete(sm.sessions, sessionID)
	delete(sm.sessionAge, sessionID)
	return nil
}

// CleanupExpiredSessions removes sessions older than maxAge
func (sm *InMemorySessionManager) CleanupExpiredSessions() int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	var expired []string

	for sessionID, lastAccess := range sm.sessionAge {
		if now.Sub(lastAccess) > sm.maxAge {
			expired = append(expired, sessionID)
		}
	}

	for _, sessionID := range expired {
		if session, exists := sm.sessions[sessionID]; exists {
			session.Close() // Best effort cleanup
		}
		delete(sm.sessions, sessionID)
		delete(sm.sessionAge, sessionID)
	}

	return len(expired)
}

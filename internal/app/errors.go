package app

import "fmt"

// SessionError represents session-related errors
type SessionError struct {
	SessionID string
	Operation string
	Cause     error
}

func (e *SessionError) Error() string {
	return fmt.Sprintf("session error [%s] during %s: %v", e.SessionID, e.Operation, e.Cause)
}

func (e *SessionError) Unwrap() error {
	return e.Cause
}

// NewSessionError creates a new session error
func NewSessionError(sessionID, operation string, cause error) *SessionError {
	return &SessionError{
		SessionID: sessionID,
		Operation: operation,
		Cause:     cause,
	}
}

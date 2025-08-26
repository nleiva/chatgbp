package app

import "fmt"

// APIError represents an error from the LLM API
type APIError struct {
	Code    int
	Message string
	Type    string
	Cause   error
}

func (e *APIError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.Code, e.Type, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

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

// ValidationError represents configuration or input validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s (value: %v): %s", e.Field, e.Value, e.Message)
}

// NewAPIError creates a new API error
func NewAPIError(code int, message, errorType string, cause error) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Type:    errorType,
		Cause:   cause,
	}
}

// NewSessionError creates a new session error
func NewSessionError(sessionID, operation string, cause error) *SessionError {
	return &SessionError{
		SessionID: sessionID,
		Operation: operation,
		Cause:     cause,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

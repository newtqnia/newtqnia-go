package newtqnia

import (
	"fmt"
	"time"
)

// APIError is returned for a non-successful HTTP response.
type APIError struct {
	StatusCode int
	RequestID  string
	Code       string
	Message    string
	RetryAfter time.Duration
	Body       string
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("newtqnia: %s (status %d, request %s)", e.Message, e.StatusCode, e.RequestID)
	}
	return fmt.Sprintf("newtqnia: %s (status %d)", e.Message, e.StatusCode)
}

// ValidationError reports invalid client configuration or request parameters.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return "newtqnia: " + e.Message }

// AuthenticationError reports an HTTP 401 response.
type AuthenticationError struct{ *APIError }

func (e *AuthenticationError) Unwrap() error { return e.APIError }

// AuthorizationError reports an HTTP 403 response.
type AuthorizationError struct{ *APIError }

func (e *AuthorizationError) Unwrap() error { return e.APIError }

// NotFoundError reports an HTTP 404 response.
type NotFoundError struct{ *APIError }

func (e *NotFoundError) Unwrap() error { return e.APIError }

// ConflictError reports an HTTP 409 response.
type ConflictError struct{ *APIError }

func (e *ConflictError) Unwrap() error { return e.APIError }

// RateLimitError reports an HTTP 429 response.
type RateLimitError struct{ *APIError }

func (e *RateLimitError) Unwrap() error { return e.APIError }

// ServerError reports an HTTP 5xx response.
type ServerError struct{ *APIError }

func (e *ServerError) Unwrap() error { return e.APIError }

// NetworkError reports a transport failure after retries are exhausted.
type NetworkError struct{ Err error }

func (e *NetworkError) Error() string { return "newtqnia: unable to reach the API: " + e.Err.Error() }
func (e *NetworkError) Unwrap() error { return e.Err }

// TimeoutError reports expiry of the client request timeout.
type TimeoutError struct {
	Duration time.Duration
	Err      error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("newtqnia: request timed out after %s", e.Duration)
}
func (e *TimeoutError) Unwrap() error { return e.Err }

// Package model defines response structures for the Hokku webhook service.
// Following SOLID SRP principle: handles only response data representation.
package model

import (
	"net/http"
	"time"
)

// APIResponse represents the standard API response structure
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// WebhookResponse represents the response after successful webhook processing
type WebhookResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Uptime    string            `json:"uptime"`
	Version   string            `json:"version"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Response builders following builder pattern for consistency

// NewSuccessResponse creates a successful API response
func NewSuccessResponse(message string, data interface{}) *APIResponse {
	return &APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(message, errorDetail string) *APIResponse {
	return &APIResponse{
		Success:   false,
		Message:   message,
		Error:     errorDetail,
		Timestamp: time.Now().UTC(),
	}
}

// NewWebhookResponse creates a webhook processing response
func NewWebhookResponse(id, filename, path string, size int64) *WebhookResponse {
	return &WebhookResponse{
		ID:       id,
		Filename: filename,
		Path:     path,
		Size:     size,
	}
}

// NewHealthResponse creates a health check response
func NewHealthResponse(status string, checks map[string]string, uptime, version string) *HealthResponse {
	return &HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
		Uptime:    uptime,
		Version:   version,
	}
}

// HTTPStatusCode returns appropriate HTTP status code based on response
func (r *APIResponse) HTTPStatusCode() int {
	if r.Success {
		return http.StatusOK
	}

	// Determine status code based on error content
	switch {
	case contains(r.Error, "unauthorized"):
		return http.StatusUnauthorized
	case contains(r.Error, "validation"), contains(r.Error, "invalid"):
		return http.StatusBadRequest
	case contains(r.Error, "not found"):
		return http.StatusNotFound
	case contains(r.Error, "insufficient space"), contains(r.Error, "disk"):
		return http.StatusInsufficientStorage
	default:
		return http.StatusInternalServerError
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr))))
}

// containsMiddle checks if substr exists in the middle of s
func containsMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr)-1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

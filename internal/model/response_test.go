package model

import (
	"net/http"
	"testing"
)

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	response := NewSuccessResponse("test message", data)

	if !response.Success {
		t.Errorf("expected Success to be true")
	}

	if response.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", response.Message)
	}

	// Check data content (can't directly compare maps)
	if dataMap, ok := response.Data.(map[string]string); !ok {
		t.Errorf("expected data to be map[string]string")
	} else if dataMap["key"] != "value" {
		t.Errorf("expected data key 'value', got '%s'", dataMap["key"])
	}

	if response.Error != "" {
		t.Errorf("expected empty error field, got '%s'", response.Error)
	}

	if response.Timestamp.IsZero() {
		t.Errorf("expected timestamp to be set")
	}
}

func TestNewErrorResponse(t *testing.T) {
	response := NewErrorResponse("test error", "detailed error")

	if response.Success {
		t.Errorf("expected Success to be false")
	}

	if response.Message != "test error" {
		t.Errorf("expected message 'test error', got '%s'", response.Message)
	}

	if response.Error != "detailed error" {
		t.Errorf("expected error 'detailed error', got '%s'", response.Error)
	}

	if response.Data != nil {
		t.Errorf("expected nil data, got %v", response.Data)
	}

	if response.Timestamp.IsZero() {
		t.Errorf("expected timestamp to be set")
	}
}

func TestNewWebhookResponse(t *testing.T) {
	response := NewWebhookResponse("test-id", "test.json", "/path/to/test.json", 1024)

	if response.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", response.ID)
	}

	if response.Filename != "test.json" {
		t.Errorf("expected filename 'test.json', got '%s'", response.Filename)
	}

	if response.Path != "/path/to/test.json" {
		t.Errorf("expected path '/path/to/test.json', got '%s'", response.Path)
	}

	if response.Size != 1024 {
		t.Errorf("expected size 1024, got %d", response.Size)
	}
}

func TestNewHealthResponse(t *testing.T) {
	checks := map[string]string{"disk": "ok", "memory": "ok"}
	response := NewHealthResponse("healthy", checks, "1h30m", "1.0.0")

	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response.Status)
	}

	if response.Uptime != "1h30m" {
		t.Errorf("expected uptime '1h30m', got '%s'", response.Uptime)
	}

	if response.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", response.Version)
	}

	if len(response.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(response.Checks))
	}

	if response.Timestamp.IsZero() {
		t.Errorf("expected timestamp to be set")
	}
}

func TestAPIResponse_HTTPStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		response *APIResponse
		expected int
	}{
		{
			name:     "success response",
			response: &APIResponse{Success: true},
			expected: http.StatusOK,
		},
		{
			name:     "unauthorized error",
			response: &APIResponse{Success: false, Error: "unauthorized access"},
			expected: http.StatusUnauthorized,
		},
		{
			name:     "validation error",
			response: &APIResponse{Success: false, Error: "validation failed"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "invalid payload error",
			response: &APIResponse{Success: false, Error: "invalid request format"},
			expected: http.StatusBadRequest,
		},
		{
			name:     "not found error",
			response: &APIResponse{Success: false, Error: "resource not found"},
			expected: http.StatusNotFound,
		},
		{
			name:     "disk space error",
			response: &APIResponse{Success: false, Error: "insufficient space available"},
			expected: http.StatusInsufficientStorage,
		},
		{
			name:     "disk operation error",
			response: &APIResponse{Success: false, Error: "disk write failed"},
			expected: http.StatusInsufficientStorage,
		},
		{
			name:     "generic error",
			response: &APIResponse{Success: false, Error: "something went wrong"},
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.HTTPStatusCode()

			if result != tt.expected {
				t.Errorf("expected status code %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "unauthorized", "unauthorized", true},
		{"prefix match", "unauthorized access", "unauthorized", true},
		{"suffix match", "access unauthorized", "unauthorized", true},
		{"middle match", "some unauthorized action", "unauthorized", true},
		{"no match", "authorized access", "unauthorized", false},
		{"empty string", "", "unauthorized", false},
		{"empty substring", "unauthorized", "", true},
		{"case sensitive", "Unauthorized", "unauthorized", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)

			if result != tt.expected {
				t.Errorf("expected %t, got %t for contains('%s', '%s')", tt.expected, result, tt.s, tt.substr)
			}
		})
	}
}

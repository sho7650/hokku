package errors

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrInvalidPayload", ErrInvalidPayload, "invalid payload"},
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized"},
		{"ErrInsufficientSpace", ErrInsufficientSpace, "insufficient disk space"},
		{"ErrFileExists", ErrFileExists, "file already exists"},
		{"ErrInvalidPath", ErrInvalidPath, "invalid file path"},
		{"ErrValidationFailed", ErrValidationFailed, "validation failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestWrapValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		err      error
		expected string
		wantNil  bool
	}{
		{
			name:     "wrap validation error",
			field:    "username",
			err:      ErrInvalidPayload,
			expected: "validation failed for field username: invalid payload",
		},
		{
			name:    "nil error returns nil",
			field:   "username",
			err:     nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapValidationError(tt.field, tt.err)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if result.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Error())
			}

			// Test that the original error is preserved
			if !errors.Is(result, tt.err) {
				t.Errorf("wrapped error should contain original error")
			}
		})
	}
}

func TestWrapFileError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		path      string
		err       error
		expected  string
		wantNil   bool
	}{
		{
			name:      "wrap file error",
			operation: "write",
			path:      "/tmp/test.json",
			err:       ErrInsufficientSpace,
			expected:  "file write failed for /tmp/test.json: insufficient disk space",
		},
		{
			name:      "nil error returns nil",
			operation: "write",
			path:      "/tmp/test.json",
			err:       nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapFileError(tt.operation, tt.path, tt.err)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if result.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Error())
			}

			// Test that the original error is preserved
			if !errors.Is(result, tt.err) {
				t.Errorf("wrapped error should contain original error")
			}
		})
	}
}

func TestWrapConfigError(t *testing.T) {
	origErr := errors.New("config file not found")
	result := WrapConfigError("storage_path", origErr)

	expected := "configuration error for storage_path: config file not found"
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}

	// Test error unwrapping
	if !errors.Is(result, origErr) {
		t.Errorf("wrapped error should contain original error")
	}
}

func TestWrapDiskError(t *testing.T) {
	origErr := errors.New("permission denied")
	result := WrapDiskError("check_space", origErr)

	expected := "disk operation check_space failed: permission denied"
	if result.Error() != expected {
		t.Errorf("expected %q, got %q", expected, result.Error())
	}

	// Test error unwrapping
	if !errors.Is(result, origErr) {
		t.Errorf("wrapped error should contain original error")
	}
}

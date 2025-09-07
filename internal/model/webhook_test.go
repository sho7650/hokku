package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWebhookPayload_GenerateID(t *testing.T) {
	tests := []struct {
		name     string
		payload  *WebhookPayload
		wantUUID bool
	}{
		{
			name:     "generates ID when empty",
			payload:  &WebhookPayload{},
			wantUUID: true,
		},
		{
			name:     "preserves existing ID",
			payload:  &WebhookPayload{ID: "existing-id"},
			wantUUID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.payload.ID
			tt.payload.GenerateID()

			if tt.wantUUID {
				// Should generate a valid UUID
				if tt.payload.ID == "" {
					t.Errorf("expected ID to be generated, got empty string")
				}

				// Validate UUID format
				if _, err := uuid.Parse(tt.payload.ID); err != nil {
					t.Errorf("expected valid UUID, got %s: %v", tt.payload.ID, err)
				}
			} else {
				// Should preserve existing ID
				if tt.payload.ID != originalID {
					t.Errorf("expected ID %s to be preserved, got %s", originalID, tt.payload.ID)
				}
			}
		})
	}
}

func TestWebhookPayload_SetTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		payload *WebhookPayload
		wantSet bool
	}{
		{
			name:    "sets timestamp when zero",
			payload: &WebhookPayload{},
			wantSet: true,
		},
		{
			name:    "preserves existing timestamp",
			payload: &WebhookPayload{Timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			wantSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalTimestamp := tt.payload.Timestamp
			before := time.Now().UTC()

			tt.payload.SetTimestamp()

			after := time.Now().UTC()

			if tt.wantSet {
				// Should set timestamp within reasonable time window
				if tt.payload.Timestamp.IsZero() {
					t.Errorf("expected timestamp to be set")
				}

				if tt.payload.Timestamp.Before(before) || tt.payload.Timestamp.After(after) {
					t.Errorf("timestamp %v should be between %v and %v", tt.payload.Timestamp, before, after)
				}
			} else {
				// Should preserve existing timestamp
				if !tt.payload.Timestamp.Equal(originalTimestamp) {
					t.Errorf("expected timestamp %v to be preserved, got %v", originalTimestamp, tt.payload.Timestamp)
				}
			}
		})
	}
}

func TestWebhookPayload_String(t *testing.T) {
	payload := &WebhookPayload{
		ID:          "test-id",
		Title:       "Test Title",
		Description: "Test Description",
		Source:      "test-source",
		Type:        "test-type",
		Timestamp:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"key1":   "value1",
			"key2":   "value2",
			"secret": "should-not-be-logged",
		},
	}

	result := payload.String()

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("String() should return valid JSON: %v", err)
	}

	// Should contain non-sensitive fields
	requiredFields := []string{"id", "title", "description", "source", "type", "timestamp", "data_keys"}
	for _, field := range requiredFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("expected field %s in JSON output", field)
		}
	}

	// Should contain data keys but not values
	if dataKeys, ok := parsed["data_keys"].([]interface{}); ok {
		if len(dataKeys) != 3 {
			t.Errorf("expected 3 data keys, got %d", len(dataKeys))
		}
	} else {
		t.Errorf("expected data_keys to be array")
	}

	// Should NOT contain sensitive data values
	if strings.Contains(result, "should-not-be-logged") {
		t.Errorf("String() should not include sensitive data values")
	}
}

func TestWebhookPayload_GetFileName(t *testing.T) {
	tests := []struct {
		name     string
		payload  *WebhookPayload
		expected string
	}{
		{
			name: "with timestamp and ID",
			payload: &WebhookPayload{
				ID:        "test-uuid",
				Title:     "Test Title",
				Timestamp: time.Date(2023, 1, 1, 12, 30, 45, 0, time.UTC),
			},
			expected: "2023-01-01_12-30-45_test-uuid_Test_Title.json",
		},
		{
			name: "with unsafe characters in title",
			payload: &WebhookPayload{
				ID:        "test-uuid",
				Title:     "Test/Title:With*Unsafe?Chars",
				Timestamp: time.Date(2023, 1, 1, 12, 30, 45, 0, time.UTC),
			},
			expected: "2023-01-01_12-30-45_test-uuid_Test_Title_With_Unsafe_Chars.json",
		},
		{
			name: "with long title (should be truncated)",
			payload: &WebhookPayload{
				ID:        "test-uuid",
				Title:     "This is a very long title that should be truncated to a reasonable length",
				Timestamp: time.Date(2023, 1, 1, 12, 30, 45, 0, time.UTC),
			},
			expected: "2023-01-01_12-30-45_test-uuid_This_is_a_very_long_title_that_s.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payload.GetFileName()

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}

			// Should always end with .json
			if !strings.HasSuffix(result, ".json") {
				t.Errorf("filename should end with .json")
			}

			// Should not contain unsafe characters
			unsafeChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
			for _, char := range unsafeChars {
				if strings.Contains(result, char) {
					t.Errorf("filename should not contain unsafe character: %s", char)
				}
			}
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "safe string unchanged",
			input:    "SafeTitle",
			expected: "SafeTitle",
		},
		{
			name:     "unsafe characters replaced",
			input:    "Unsafe/Title:With*Special?Chars",
			expected: "Unsafe_Title_With_Special_Chars",
		},
		{
			name:     "spaces replaced",
			input:    "Title With Spaces",
			expected: "Title_With_Spaces",
		},
		{
			name:     "long string truncated",
			input:    "This is a very long string that should be truncated",
			expected: "This_is_a_very_long_string_that_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForFilename(tt.input)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}

			// Should never exceed 32 characters
			if len(result) > 32 {
				t.Errorf("result length %d should not exceed 32 characters", len(result))
			}
		})
	}
}

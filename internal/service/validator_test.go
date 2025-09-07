package service

import (
	"hokku/internal/config"
	"hokku/internal/model"
	"strings"
	"testing"
	"time"
)

// Test implementation of PayloadValidator following TDD principles
func TestPayloadValidator(t *testing.T) {
	cfg := &config.Config{
		MaxTitleLength:    64,
		MaxDescLength:     512,
		MaxDataSize:       5 * 1024 * 1024, // 5MB
		AllowedExtensions: []string{"json", "txt", "log"},
	}

	validator := NewPayloadValidator(cfg)

	t.Run("Validate successful", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title:       "Valid Test Webhook",
			Description: "Valid description",
			Data:        map[string]interface{}{"key": "value", "number": 42},
		}
		payload.GenerateID()
		payload.SetTimestamp()

		err := validator.Validate(payload)
		if err != nil {
			t.Errorf("Validate() error = %v, expected nil", err)
		}
	})

	t.Run("Validate nil payload", func(t *testing.T) {
		err := validator.Validate(nil)
		if err == nil {
			t.Error("Validate() should return error for nil payload")
		}
	})

	t.Run("ValidateStructure successful", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title: "Valid Title",
			Data:  map[string]interface{}{"required": "data"},
		}

		err := validator.ValidateStructure(payload)
		if err != nil {
			t.Errorf("ValidateStructure() error = %v, expected nil", err)
		}
	})

	t.Run("ValidateStructure missing title", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Data: map[string]interface{}{"some": "data"},
		}

		err := validator.ValidateStructure(payload)
		if err == nil {
			t.Error("ValidateStructure() should return error for missing title")
		}
		if !strings.Contains(err.Error(), "title") {
			t.Errorf("Error should mention title, got: %v", err)
		}
	})

	t.Run("ValidateStructure missing data", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title: "Valid Title",
		}

		err := validator.ValidateStructure(payload)
		if err == nil {
			t.Error("ValidateStructure() should return error for missing data")
		}
		if !strings.Contains(err.Error(), "data") {
			t.Errorf("Error should mention data, got: %v", err)
		}
	})

	t.Run("ValidateStructure empty data", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title: "Valid Title",
			Data:  map[string]interface{}{},
		}

		err := validator.ValidateStructure(payload)
		if err == nil {
			t.Error("ValidateStructure() should return error for empty data")
		}
	})

	t.Run("ValidateContent successful", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title:       "Valid Title",
			Description: "Valid description with reasonable length",
			Data:        map[string]interface{}{"key": "value"},
		}

		err := validator.ValidateContent(payload)
		if err != nil {
			t.Errorf("ValidateContent() error = %v, expected nil", err)
		}
	})
}

func TestPayloadValidatorLengthValidation(t *testing.T) {
	cfg := &config.Config{
		MaxTitleLength:    10,  // Short for testing
		MaxDescLength:     20,  // Short for testing
		MaxDataSize:       100, // Small for testing
		AllowedExtensions: []string{"json"},
	}

	validator := NewPayloadValidator(cfg)

	tests := []struct {
		name    string
		payload *model.WebhookPayload
		wantErr bool
		errMsg  string
	}{
		{
			name: "title too long",
			payload: &model.WebhookPayload{
				Title: strings.Repeat("a", 15), // Longer than limit
				Data:  map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "title",
		},
		{
			name: "description too long",
			payload: &model.WebhookPayload{
				Title:       "Valid",
				Description: strings.Repeat("b", 25), // Longer than limit
				Data:        map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "description",
		},
		{
			name: "data too large",
			payload: &model.WebhookPayload{
				Title: "Valid",
				Data:  map[string]interface{}{"large": strings.Repeat("x", 200)}, // Larger than limit when serialized
			},
			wantErr: true,
			errMsg:  "data size",
		},
		{
			name: "all valid lengths",
			payload: &model.WebhookPayload{
				Title:       "Short",                              // Within limit
				Description: "Short desc",                         // Within limit
				Data:        map[string]interface{}{"key": "val"}, // Small data
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateContent(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestPayloadValidatorSecurityValidation(t *testing.T) {
	cfg := &config.Config{
		MaxTitleLength:    100,
		MaxDescLength:     500,
		MaxDataSize:       1024 * 1024,
		AllowedExtensions: []string{"json", "txt"},
	}

	validator := NewPayloadValidator(cfg)

	securityTests := []struct {
		name    string
		payload *model.WebhookPayload
		wantErr bool
		errMsg  string
	}{
		{
			name: "null byte in title",
			payload: &model.WebhookPayload{
				Title: "malicious\x00title",
				Data:  map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "unsafe characters",
		},
		{
			name: "null byte in description",
			payload: &model.WebhookPayload{
				Title:       "Valid Title",
				Description: "malicious\x00description",
				Data:        map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "unsafe characters",
		},
		{
			name: "control characters in title",
			payload: &model.WebhookPayload{
				Title: "title\x01\x02\x03",
				Data:  map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "unsafe characters",
		},
		{
			name: "invalid UTF-8 in title",
			payload: &model.WebhookPayload{
				Title: "title\xff\xfe",
				Data:  map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "invalid UTF-8",
		},
		{
			name: "valid unicode characters",
			payload: &model.WebhookPayload{
				Title:       "Valid 中文 Ñoël",
				Description: "Unicode test ñoël 中文字符",
				Data:        map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "extremely nested data structure",
			payload: &model.WebhookPayload{
				Title: "Nested Test",
				Data: map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": map[string]interface{}{
							"level3": map[string]interface{}{
								"level4": map[string]interface{}{
									"level5": map[string]interface{}{
										"level6": map[string]interface{}{
											"deep": "value",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "nesting depth",
		},
		{
			name: "very long string in data",
			payload: &model.WebhookPayload{
				Title: "String Test",
				Data: map[string]interface{}{
					"long_string": strings.Repeat("x", 15000), // Longer than MaxStringLength
				},
			},
			wantErr: true,
			errMsg:  "string length",
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateContent(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestPayloadValidatorBusinessRules(t *testing.T) {
	cfg := &config.Config{
		MaxTitleLength:    64,
		MaxDescLength:     512,
		MaxDataSize:       1024 * 1024,
		AllowedExtensions: []string{"json", "txt"},
	}

	validator := NewPayloadValidator(cfg)

	businessTests := []struct {
		name    string
		payload *model.WebhookPayload
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid source and type",
			payload: &model.WebhookPayload{
				Title:  "Test",
				Source: "api.example.com",
				Type:   "user.created",
				Data:   map[string]interface{}{"user_id": 123},
			},
			wantErr: false,
		},
		{
			name: "source too long",
			payload: &model.WebhookPayload{
				Title:  "Test",
				Source: strings.Repeat("x", 200), // Longer than 128 char limit
				Data:   map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "source",
		},
		{
			name: "type too long",
			payload: &model.WebhookPayload{
				Title: "Test",
				Type:  strings.Repeat("x", 50), // Longer than 32 char limit
				Data:  map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "type",
		},
		{
			name: "reserved field names in data",
			payload: &model.WebhookPayload{
				Title: "Test",
				Data: map[string]interface{}{
					"id":        "should_not_be_set", // Reserved field
					"timestamp": "2023-01-01",        // Reserved field
					"normal":    "allowed",
				},
			},
			wantErr: true,
			errMsg:  "reserved field",
		},
		{
			name: "data with allowed field names",
			payload: &model.WebhookPayload{
				Title: "Test",
				Data: map[string]interface{}{
					"user_id":    123,
					"action":     "create",
					"metadata":   map[string]interface{}{"version": 1},
					"custom_key": "custom_value",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range businessTests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
				t.Errorf("Error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestPayloadValidatorEdgeCases(t *testing.T) {
	cfg := &config.Config{
		MaxTitleLength:    64,
		MaxDescLength:     512,
		MaxDataSize:       1024 * 1024,
		AllowedExtensions: []string{"json"},
	}

	validator := NewPayloadValidator(cfg)

	t.Run("payload with all fields populated", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title:       "Complete Test",
			Description: "Complete description",
			Source:      "test.api",
			Type:        "test.event",
			Data:        map[string]interface{}{"complete": true},
			ID:          "test-id",
			Timestamp:   time.Now(),
		}

		err := validator.Validate(payload)
		if err != nil {
			t.Errorf("Validate() should accept complete payload, error = %v", err)
		}
	})

	t.Run("payload with minimal required fields", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title: "Minimal",
			Data:  map[string]interface{}{"minimal": true},
		}

		err := validator.Validate(payload)
		if err != nil {
			t.Errorf("Validate() should accept minimal payload, error = %v", err)
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title:       "", // Empty title should fail
			Description: "", // Empty description is OK
			Source:      "", // Empty source is OK
			Type:        "", // Empty type is OK
			Data:        map[string]interface{}{"key": "value"},
		}

		err := validator.Validate(payload)
		if err == nil {
			t.Error("Validate() should reject empty title")
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkPayloadValidatorValidate(b *testing.B) {
	cfg := &config.Config{
		MaxTitleLength:    64,
		MaxDescLength:     512,
		MaxDataSize:       1024 * 1024,
		AllowedExtensions: []string{"json", "txt"},
	}

	validator := NewPayloadValidator(cfg)

	payload := &model.WebhookPayload{
		Title:       "Benchmark Test Webhook",
		Description: "Performance test payload with reasonable content",
		Source:      "benchmark.test",
		Type:        "benchmark.event",
		Data: map[string]interface{}{
			"user_id": 123,
			"action":  "test",
			"metadata": map[string]interface{}{
				"version":   1,
				"timestamp": "2023-01-01T00:00:00Z",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate(payload)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkPayloadValidatorValidateStructure(b *testing.B) {
	cfg := &config.Config{
		MaxTitleLength: 64,
		MaxDescLength:  512,
		MaxDataSize:    1024 * 1024,
	}

	validator := NewPayloadValidator(cfg)

	payload := &model.WebhookPayload{
		Title: "Structure Test",
		Data:  map[string]interface{}{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateStructure(payload)
		if err != nil {
			b.Fatalf("Structure validation failed: %v", err)
		}
	}
}

func BenchmarkPayloadValidatorValidateContent(b *testing.B) {
	cfg := &config.Config{
		MaxTitleLength:    64,
		MaxDescLength:     512,
		MaxDataSize:       1024 * 1024,
		AllowedExtensions: []string{"json"},
	}

	validator := NewPayloadValidator(cfg)

	payload := &model.WebhookPayload{
		Title:       "Content Test",
		Description: "Content validation benchmark",
		Data: map[string]interface{}{
			"field1": "value1",
			"field2": 42,
			"field3": []interface{}{"a", "b", "c"}, // Use []interface{} instead of []string
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateContent(payload)
		if err != nil {
			b.Fatalf("Content validation failed: %v", err)
		}
	}
}

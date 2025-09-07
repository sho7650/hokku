package service

import (
	"encoding/json"
	"hokku/internal/config"
	"hokku/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServiceIntegration demonstrates the services working together
// in a realistic webhook processing scenario.
func TestServiceIntegration(t *testing.T) {
	// Setup temporary directory for testing
	tempDir := t.TempDir()

	// Create configuration
	cfg := &config.Config{
		StoragePath:       tempDir,
		MaxFileSize:       1024 * 1024, // 1MB
		Port:              8080,
		Environment:       "development",
		MaxTitleLength:    64,
		MaxDescLength:     512,
		MaxDataSize:       5 * 1024 * 1024, // 5MB
		AllowedExtensions: []string{"json", "txt", "log"},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Create services
	validator := NewPayloadValidator(cfg)
	fileStore := NewFileStore(cfg)

	t.Run("complete webhook processing flow", func(t *testing.T) {
		// Create a realistic webhook payload
		payload := &model.WebhookPayload{
			Title:       "User Registration Event",
			Description: "New user registered in the system",
			Source:      "auth.api.example.com",
			Type:        "user.registration.completed",
			Data: map[string]interface{}{
				"user_id":  12345,
				"email":    "user@example.com",
				"username": "newuser123",
				"plan":     "premium",
				"metadata": map[string]interface{}{
					"signup_method": "oauth_google",
					"referrer":      "https://example.com/signup",
					"user_agent":    "Mozilla/5.0 (compatible)",
				},
				"preferences": map[string]interface{}{
					"notifications": true,
					"newsletter":    false,
					"theme":         "dark",
				},
			},
		}

		// Generate ID and timestamp
		payload.GenerateID()
		payload.SetTimestamp()

		// Step 1: Validate the payload
		if err := validator.Validate(payload); err != nil {
			t.Fatalf("Payload validation failed: %v", err)
		}

		// Step 2: Check disk space before writing
		availableSpace, err := fileStore.CheckDiskSpace()
		if err != nil {
			t.Fatalf("Disk space check failed: %v", err)
		}

		if availableSpace <= 0 {
			t.Fatal("Insufficient disk space")
		}

		// Step 3: Store the payload
		filePath, err := fileStore.Write(payload)
		if err != nil {
			t.Fatalf("File storage failed: %v", err)
		}

		// Step 4: Verify the file was created correctly
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("File was not created: %s", filePath)
		}

		// Step 5: Verify file contents
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read stored file: %v", err)
		}

		var storedPayload model.WebhookPayload
		if err := json.Unmarshal(fileData, &storedPayload); err != nil {
			t.Fatalf("Failed to unmarshal stored payload: %v", err)
		}

		// Verify data integrity
		if storedPayload.ID != payload.ID {
			t.Errorf("ID mismatch: got %s, want %s", storedPayload.ID, payload.ID)
		}

		if storedPayload.Title != payload.Title {
			t.Errorf("Title mismatch: got %s, want %s", storedPayload.Title, payload.Title)
		}

		if storedPayload.Source != payload.Source {
			t.Errorf("Source mismatch: got %s, want %s", storedPayload.Source, payload.Source)
		}

		if storedPayload.Type != payload.Type {
			t.Errorf("Type mismatch: got %s, want %s", storedPayload.Type, payload.Type)
		}

		// Verify data field integrity
		if storedData, ok := storedPayload.Data["user_id"]; !ok {
			t.Error("user_id missing from stored data")
		} else if userID, ok := storedData.(float64); !ok || int(userID) != 12345 {
			t.Errorf("user_id mismatch: got %v, want 12345", storedData)
		}

		// Verify nested data
		if metadata, ok := storedPayload.Data["metadata"]; !ok {
			t.Error("metadata missing from stored data")
		} else if metaMap, ok := metadata.(map[string]interface{}); !ok {
			t.Error("metadata is not a map")
		} else if signupMethod, ok := metaMap["signup_method"]; !ok {
			t.Error("signup_method missing from metadata")
		} else if signupMethod != "oauth_google" {
			t.Errorf("signup_method mismatch: got %v, want oauth_google", signupMethod)
		}

		// Verify file path security
		if !filepath.IsAbs(filePath) {
			t.Error("File path should be absolute")
		}

		if !filepath.HasPrefix(filePath, tempDir) {
			t.Errorf("File path %s should be within temp directory %s", filePath, tempDir)
		}

		// Verify filename is safe
		filename := filepath.Base(filePath)
		if filename == "" || filename == "." || filename == ".." {
			t.Errorf("Invalid filename: %s", filename)
		}
	})

	t.Run("integration with malicious payload", func(t *testing.T) {
		// Create a payload with potential security issues
		maliciousPayload := &model.WebhookPayload{
			Title:       "../../../etc/passwd", // Path traversal (null byte removed for this test)
			Description: "Attempt to access system files",
			Source:      "malicious.com",  // Simplified for validation
			Type:        "malicious.type", // Simplified for validation
			Data: map[string]interface{}{
				"id":               "should_be_rejected", // Reserved field
				"malicious_script": "<script>alert('xss')</script>",
				"file_path":        "../../../etc/passwd",
			},
		}

		maliciousPayload.GenerateID()
		maliciousPayload.SetTimestamp()

		// Validation should catch the reserved field
		err := validator.Validate(maliciousPayload)
		if err == nil {
			t.Error("Validation should have failed for malicious payload with reserved field")
		}

		// Remove the reserved field to test file storage security
		delete(maliciousPayload.Data, "id")

		// Validation should pass now (path traversal in title will be handled by file storage)
		if err := validator.Validate(maliciousPayload); err != nil {
			t.Fatalf("Validation failed after removing reserved field: %v", err)
		}

		// File storage should handle the malicious title safely
		filePath, err := fileStore.Write(maliciousPayload)
		if err != nil {
			t.Fatalf("File storage should handle malicious payload safely: %v", err)
		}

		// Verify the file is stored safely
		if !filepath.HasPrefix(filePath, tempDir) {
			t.Errorf("Malicious payload file path should still be safe: %s", filePath)
		}

		// Verify filename was sanitized
		filename := filepath.Base(filePath)
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
			t.Errorf("Filename should be sanitized: %s", filename)
		}

		// Test a payload with null bytes in title (should be rejected by validation)
		nullBytePayload := &model.WebhookPayload{
			Title: "malicious\x00title",
			Data:  map[string]interface{}{"test": true},
		}
		nullBytePayload.GenerateID()
		nullBytePayload.SetTimestamp()

		// This should fail validation due to null bytes
		err = validator.Validate(nullBytePayload)
		if err == nil {
			t.Error("Validation should reject payloads with null bytes in title")
		}
	})

	t.Run("integration with edge case payloads", func(t *testing.T) {
		edgeCases := []struct {
			name    string
			payload *model.WebhookPayload
		}{
			{
				name: "minimal payload",
				payload: &model.WebhookPayload{
					Title: "Min",
					Data:  map[string]interface{}{"key": "value"},
				},
			},
			{
				name: "unicode payload",
				payload: &model.WebhookPayload{
					Title:       "测试 Тест テスト 🚀",
					Description: "Unicode support test with émojis 🎉",
					Data:        map[string]interface{}{"message": "Hello 世界 мир 世界"},
				},
			},
			{
				name: "maximum length payload",
				payload: &model.WebhookPayload{
					Title:       "Maximum length title that is exactly 64 characters long test",
					Description: "This description is designed to test the maximum allowed length for descriptions in the webhook payload validation system and should be accepted.",
					Data: map[string]interface{}{
						"large_data": map[string]interface{}{
							"level1": map[string]interface{}{
								"level2": map[string]interface{}{
									"level3": "deep nesting test",
								},
							},
						},
					},
				},
			},
		}

		for _, tc := range edgeCases {
			t.Run(tc.name, func(t *testing.T) {
				tc.payload.GenerateID()
				tc.payload.SetTimestamp()

				// Validate
				if err := validator.Validate(tc.payload); err != nil {
					t.Fatalf("Validation failed for %s: %v", tc.name, err)
				}

				// Store
				filePath, err := fileStore.Write(tc.payload)
				if err != nil {
					t.Fatalf("Storage failed for %s: %v", tc.name, err)
				}

				// Verify
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Fatalf("File not created for %s: %s", tc.name, filePath)
				}
			})
		}
	})
}

// TestServiceErrorHandling tests error propagation between services
func TestServiceErrorHandling(t *testing.T) {
	// Create invalid configuration
	cfg := &config.Config{
		StoragePath:    "", // Invalid: empty path
		MaxFileSize:    -1, // Invalid: negative size
		Port:           0,  // Invalid: zero port
		MaxTitleLength: 0,  // Invalid: zero length
		MaxDescLength:  -1, // Invalid: negative length
		MaxDataSize:    0,  // Invalid: zero size
	}

	// Configuration validation should fail
	if err := cfg.Validate(); err == nil {
		t.Error("Configuration validation should fail for invalid config")
	}

	// Create valid configuration for service testing
	tempDir := t.TempDir()
	validCfg := &config.Config{
		StoragePath:       tempDir,
		MaxFileSize:       100, // Very small for testing file storage
		Port:              8080,
		MaxTitleLength:    10,  // Very small for testing
		MaxDescLength:     20,  // Very small for testing
		MaxDataSize:       100, // Very small for testing
		AllowedExtensions: []string{"json"},
		Environment:       "development",
	}

	validator := NewPayloadValidator(validCfg)
	fileStore := NewFileStore(validCfg)

	// Test validation errors propagate correctly
	invalidPayload := &model.WebhookPayload{
		Title: "This title is way too long for the configured limit",
		Data:  map[string]interface{}{"key": "value"},
	}

	err := validator.Validate(invalidPayload)
	if err == nil {
		t.Error("Validation should fail for oversized title")
	}

	// Test file storage errors
	validPayload := &model.WebhookPayload{
		Title: "Valid",
		Data:  map[string]interface{}{"large": "This data is definitely too large for the configured limit of 100 bytes and should cause a failure when serialized to JSON format as it will exceed the size"},
	}
	validPayload.GenerateID()
	validPayload.SetTimestamp()

	_, err = fileStore.Write(validPayload)
	if err == nil {
		t.Error("File storage should fail for oversized payload")
	}
}

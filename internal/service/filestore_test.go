package service

import (
	"encoding/json"
	"hokku/internal/config"
	"hokku/internal/model"
	"hokku/pkg/security"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test implementation of FileStore following TDD principles
func TestFileStore(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	cfg := &config.Config{
		StoragePath: tempDir,
		MaxFileSize: 1024 * 1024, // 1MB for testing
	}

	fs := NewFileStore(cfg)

	t.Run("Write successful", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title:       "Test Webhook",
			Description: "Test description",
			Data:        map[string]interface{}{"key": "value"},
		}
		payload.GenerateID()
		payload.SetTimestamp()

		filePath, err := fs.Write(payload)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created at path: %s", filePath)
		}

		// Verify file contents
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		var writtenPayload model.WebhookPayload
		if err := json.Unmarshal(data, &writtenPayload); err != nil {
			t.Fatalf("Failed to unmarshal written file: %v", err)
		}

		if writtenPayload.Title != payload.Title {
			t.Errorf("Title mismatch: got %s, want %s", writtenPayload.Title, payload.Title)
		}

		// Verify path is within storage directory
		if !strings.HasPrefix(filePath, tempDir) {
			t.Errorf("File path %s is not within storage directory %s", filePath, tempDir)
		}
	})

	t.Run("Write with path traversal attack", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title:       "../../../etc/passwd",
			Description: "Malicious payload",
			Data:        map[string]interface{}{"attack": true},
		}
		payload.GenerateID()
		payload.SetTimestamp()

		// Path traversal in title should be sanitized and file created safely
		filePath, err := fs.Write(payload)
		if err != nil {
			t.Fatalf("Write() should sanitize path traversal attempts, error = %v", err)
		}

		// Verify the file is within the safe directory
		if !strings.HasPrefix(filePath, tempDir) {
			t.Errorf("File not created in safe directory: %s", filePath)
		}

		// Verify the filename was sanitized (no path separators)
		filename := filepath.Base(filePath)
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
			t.Errorf("Filename not properly sanitized: %s", filename)
		}

		// Verify file was actually created
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", filePath)
		}
	})

	t.Run("Write nil payload", func(t *testing.T) {
		_, err := fs.Write(nil)
		if err == nil {
			t.Error("Write() should return error for nil payload")
		}
	})

	t.Run("Write payload without ID", func(t *testing.T) {
		payload := &model.WebhookPayload{
			Title: "Test",
			Data:  map[string]interface{}{"key": "value"},
		}
		// Don't generate ID - should be handled by FileStore

		filePath, err := fs.Write(payload)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Verify ID was generated
		if payload.ID == "" {
			t.Error("FileStore should generate ID if missing")
		}

		// Verify file was created
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created at path: %s", filePath)
		}
	})

	t.Run("Write creates parent directories", func(t *testing.T) {
		// Test with nested directory structure
		nestedDir := filepath.Join(tempDir, "nested", "deep", "structure")
		cfg := &config.Config{
			StoragePath: nestedDir,
			MaxFileSize: 1024 * 1024,
		}

		fs := NewFileStore(cfg)

		payload := &model.WebhookPayload{
			Title: "Test Nested",
			Data:  map[string]interface{}{"nested": true},
		}
		payload.GenerateID()
		payload.SetTimestamp()

		filePath, err := fs.Write(payload)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		// Verify nested directories were created
		if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
			t.Error("Parent directories were not created")
		}

		// Verify file was created
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File was not created at path: %s", filePath)
		}
	})
}

func TestFileStoreCheckDiskSpace(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		StoragePath: tempDir,
		MaxFileSize: 1024 * 1024,
	}

	fs := NewFileStore(cfg)

	t.Run("CheckDiskSpace successful", func(t *testing.T) {
		available, err := fs.CheckDiskSpace()
		if err != nil {
			t.Fatalf("CheckDiskSpace() error = %v", err)
		}

		if available <= 0 {
			t.Errorf("CheckDiskSpace() returned non-positive value: %d", available)
		}
	})

	t.Run("CheckDiskSpace with non-existent directory", func(t *testing.T) {
		cfg := &config.Config{
			StoragePath: "/non/existent/path",
			MaxFileSize: 1024 * 1024,
		}

		fs := NewFileStore(cfg)

		_, err := fs.CheckDiskSpace()
		if err == nil {
			t.Error("CheckDiskSpace() should return error for non-existent directory")
		}
	})
}

func TestFileStoreSecurityScenarios(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		StoragePath: tempDir,
		MaxFileSize: 1024 * 1024,
	}

	fs := NewFileStore(cfg)

	securityTests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "null byte in title",
			title:   "file\x00.exe",
			wantErr: false, // Should be sanitized, not error
		},
		{
			name:    "path traversal in title",
			title:   "../../../etc/passwd",
			wantErr: false, // Should be sanitized, not error
		},
		{
			name:    "Windows reserved name",
			title:   "CON",
			wantErr: false, // Should be sanitized, not error
		},
		{
			name:    "very long title",
			title:   strings.Repeat("a", 1000),
			wantErr: false, // Should be truncated, not error
		},
		{
			name:    "title with unsafe characters",
			title:   "file<>:\"|?*.txt",
			wantErr: false, // Should be sanitized, not error
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &model.WebhookPayload{
				Title: tt.title,
				Data:  map[string]interface{}{"test": true},
			}
			payload.GenerateID()
			payload.SetTimestamp()

			filePath, err := fs.Write(payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file was created safely within storage directory
				if !strings.HasPrefix(filePath, tempDir) {
					t.Errorf("Unsafe file path: %s not within %s", filePath, tempDir)
				}

				// Verify file actually exists
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("File was not created: %s", filePath)
				}

				// Verify filename is safe
				filename := filepath.Base(filePath)
				if err := security.IsSecurePath(filename, ""); err != nil {
					t.Errorf("Filename is not secure: %s, error: %v", filename, err)
				}
			}
		})
	}
}

func TestFileStoreErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Write to read-only directory", func(t *testing.T) {
		// Create read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		err = os.Chmod(readOnlyDir, 0444) // Read-only
		if err != nil {
			t.Fatalf("Failed to make directory read-only: %v", err)
		}

		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

		cfg := &config.Config{
			StoragePath: readOnlyDir,
			MaxFileSize: 1024 * 1024,
		}

		fs := NewFileStore(cfg)

		payload := &model.WebhookPayload{
			Title: "Test",
			Data:  map[string]interface{}{"test": true},
		}
		payload.GenerateID()
		payload.SetTimestamp()

		_, err = fs.Write(payload)
		if err == nil {
			t.Error("Write() should fail for read-only directory")
		}
	})

	t.Run("Insufficient disk space simulation", func(t *testing.T) {
		// This test simulates insufficient disk space by trying to write
		// a file larger than the configured max file size
		cfg := &config.Config{
			StoragePath: tempDir,
			MaxFileSize: 100, // Very small limit
		}

		fs := NewFileStore(cfg)

		// Create large payload that exceeds limit
		largeData := make(map[string]interface{})
		largeData["large_field"] = strings.Repeat("x", 200) // Larger than limit

		payload := &model.WebhookPayload{
			Title: "Large Payload",
			Data:  largeData,
		}
		payload.GenerateID()
		payload.SetTimestamp()

		_, err := fs.Write(payload)
		if err == nil {
			t.Error("Write() should fail for payload exceeding size limit")
		}
	})
}

func TestFileStoreConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		StoragePath: tempDir,
		MaxFileSize: 1024 * 1024,
	}

	fs := NewFileStore(cfg)

	// Test concurrent writes to ensure thread safety
	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errs := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			payload := &model.WebhookPayload{
				Title: "Concurrent Test " + string(rune(id)),
				Data:  map[string]interface{}{"goroutine_id": id},
			}
			payload.GenerateID()
			payload.SetTimestamp()

			filePath, err := fs.Write(payload)
			if err != nil {
				errs <- err
				return
			}
			results <- filePath
		}(i)
	}

	// Collect results
	var filePaths []string
	for i := 0; i < numGoroutines; i++ {
		select {
		case filePath := <-results:
			filePaths = append(filePaths, filePath)
		case err := <-errs:
			t.Fatalf("Concurrent write failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent writes")
		}
	}

	// Verify all files were created and are unique
	if len(filePaths) != numGoroutines {
		t.Errorf("Expected %d files, got %d", numGoroutines, len(filePaths))
	}

	pathSet := make(map[string]bool)
	for _, path := range filePaths {
		if pathSet[path] {
			t.Errorf("Duplicate file path generated: %s", path)
		}
		pathSet[path] = true

		// Verify file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File was not created: %s", path)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkFileStoreWrite(b *testing.B) {
	tempDir := b.TempDir()
	cfg := &config.Config{
		StoragePath: tempDir,
		MaxFileSize: 10 * 1024 * 1024,
	}

	fs := NewFileStore(cfg)

	payload := &model.WebhookPayload{
		Title:       "Benchmark Test",
		Description: "Performance test payload",
		Data:        map[string]interface{}{"benchmark": true, "iteration": 0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Update payload to avoid duplicate detection
		payload.Data["iteration"] = i
		payload.GenerateID()
		payload.SetTimestamp()

		_, err := fs.Write(payload)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

func BenchmarkFileStoreCheckDiskSpace(b *testing.B) {
	tempDir := b.TempDir()
	cfg := &config.Config{
		StoragePath: tempDir,
		MaxFileSize: 1024 * 1024,
	}

	fs := NewFileStore(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fs.CheckDiskSpace()
		if err != nil {
			b.Fatalf("CheckDiskSpace failed: %v", err)
		}
	}
}

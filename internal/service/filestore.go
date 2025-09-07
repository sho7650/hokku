package service

import (
	"encoding/json"
	"fmt"
	"hokku/internal/config"
	"hokku/internal/model"
	"hokku/pkg/errors"
	"hokku/pkg/security"
	"os"
	"path/filepath"
	"syscall"
)

// FileStoreImpl implements the FileStore interface for file persistence operations.
// Following SOLID SRP: handles only file storage concerns.
type FileStoreImpl struct {
	config *config.Config
}

// NewFileStore creates a new FileStore implementation with the provided configuration.
func NewFileStore(cfg *config.Config) FileStore {
	return &FileStoreImpl{
		config: cfg,
	}
}

// Write persists a webhook payload to storage and returns the file path.
// It ensures:
// - Proper payload initialization (ID, timestamp)
// - Secure filename generation and path validation
// - Directory creation with proper permissions
// - Atomic file writing with size validation
// - Error context wrapping for debugging
func (fs *FileStoreImpl) Write(payload *model.WebhookPayload) (string, error) {
	if payload == nil {
		return "", errors.WrapValidationError("payload", fmt.Errorf("payload cannot be nil"))
	}

	// Initialize payload metadata if missing
	if payload.ID == "" {
		payload.GenerateID()
	}
	if payload.Timestamp.IsZero() {
		payload.SetTimestamp()
	}

	// Generate secure filename from payload
	baseFilename, err := fs.generateSecureFilename(payload)
	if err != nil {
		return "", errors.WrapFileError("filename generation", "", err)
	}

	// Construct full file path within storage directory
	fullPath := filepath.Join(fs.config.GetStoragePath(), baseFilename)

	// Additional security validation
	if err := security.IsSecurePath(fullPath, fs.config.GetStoragePath()); err != nil {
		return "", errors.WrapFileError("path validation", fullPath, err)
	}

	// Ensure parent directory exists with proper permissions
	parentDir := filepath.Dir(fullPath)
	if err := fs.ensureDirectoryExists(parentDir); err != nil {
		return "", errors.WrapFileError("directory creation", parentDir, err)
	}

	// Marshal payload to JSON
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", errors.WrapFileError("json marshaling", fullPath, err)
	}

	// Validate payload size against configured limit
	if int64(len(data)) > fs.config.GetMaxFileSize() {
		return "", errors.WrapFileError("size validation", fullPath,
			fmt.Errorf("payload size %d exceeds limit %d", len(data), fs.config.GetMaxFileSize()))
	}

	// Write file atomically using a temporary file and rename
	if err := fs.writeFileAtomically(fullPath, data); err != nil {
		return "", errors.WrapFileError("atomic write", fullPath, err)
	}

	return fullPath, nil
}

// CheckDiskSpace validates sufficient space is available for storage.
// Returns available bytes or error if insufficient space or filesystem issues.
func (fs *FileStoreImpl) CheckDiskSpace() (int64, error) {
	storagePath := fs.config.GetStoragePath()

	// Ensure storage directory exists
	if err := fs.ensureDirectoryExists(storagePath); err != nil {
		return 0, errors.WrapDiskError("directory access", err)
	}

	// Get filesystem statistics
	var stat syscall.Statfs_t
	err := syscall.Statfs(storagePath, &stat)
	if err != nil {
		return 0, errors.WrapDiskError("filesystem stat", err)
	}

	// Calculate available space in bytes
	availableBytes := int64(stat.Bavail) * int64(stat.Bsize)

	// Check if we have sufficient space (at least 2x max file size as buffer)
	minimumRequired := fs.config.GetMaxFileSize() * 2
	if availableBytes < minimumRequired {
		return availableBytes, errors.WrapDiskError("insufficient space",
			fmt.Errorf("available %d bytes < required %d bytes", availableBytes, minimumRequired))
	}

	return availableBytes, nil
}

// generateSecureFilename creates a secure filename from the webhook payload.
// It first sanitizes the title, then creates a secure filename to prevent path traversal.
func (fs *FileStoreImpl) generateSecureFilename(payload *model.WebhookPayload) (string, error) {
	// Sanitize the title first to prevent path traversal in the generated filename
	sanitizedTitle, err := security.SanitizeFilename(payload.Title)
	if err != nil {
		// Use fallback title if sanitization fails
		sanitizedTitle = "webhook"
	}

	// Use timestamp and ID with sanitized title
	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = payload.Timestamp
	}

	filename := fmt.Sprintf("%s_%s_%s.json",
		timestamp.Format("2006-01-02_15-04-05"),
		payload.ID,
		sanitizedTitle,
	)

	// Final security check on the generated filename
	finalFilename, err := security.SanitizeFilename(filename)
	if err != nil {
		// Ultimate fallback to secure generated filename
		fallbackName, genErr := security.GenerateSecureFilename("webhook", "json")
		if genErr != nil {
			return "", fmt.Errorf("filename generation failed: %w", genErr)
		}
		return fallbackName, nil
	}

	return finalFilename, nil
}

// ensureDirectoryExists creates the directory and any necessary parent directories
// with secure permissions (0755 for directories).
func (fs *FileStoreImpl) ensureDirectoryExists(dirPath string) error {
	// Validate the directory path for security
	if err := security.ValidatePath(dirPath, ""); err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	// Check if directory already exists
	if info, err := os.Stat(dirPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", dirPath)
		}
		return nil // Directory exists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	// Create directory with secure permissions (0755)
	// MkdirAll creates parent directories as needed
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// writeFileAtomically writes data to a file atomically using temp file + rename.
// This ensures that the file is either completely written or not at all,
// preventing partial writes in case of interruption.
func (fs *FileStoreImpl) writeFileAtomically(filePath string, data []byte) error {
	// Create temporary file in the same directory for atomic rename
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp-webhook-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tempPath := tempFile.Name()

	// Cleanup function to remove temp file on failure
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // Best effort cleanup
	}()

	// Write data to temporary file
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close temporary file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Set secure file permissions (0644 - readable by owner/group, no execute)
	if err := os.Chmod(tempPath, 0644); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Atomically rename temporary file to final path
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// Package service defines the core business interfaces for the Hokku webhook service.
// Following SOLID principles: Interface Segregation (ISP) and Dependency Inversion (DIP).
package service

import "hokku/internal/model"

// FileStore defines the interface for file storage operations.
// Following SRP: handles only file persistence concerns.
type FileStore interface {
	// Write persists a webhook payload to storage and returns the file path.
	// Returns the full file path on success, error on failure.
	Write(payload *model.WebhookPayload) (string, error)

	// CheckDiskSpace validates sufficient space is available for storage.
	// Returns available bytes or error if insufficient space.
	CheckDiskSpace() (int64, error)
}

// PayloadValidator defines the interface for webhook payload validation.
// Following SRP: handles only validation concerns.
type PayloadValidator interface {
	// Validate performs comprehensive validation of webhook payload.
	// Returns nil on success, validation error on failure.
	Validate(payload *model.WebhookPayload) error

	// ValidateStructure validates the payload structure and required fields.
	// Returns nil on success, structural validation error on failure.
	ValidateStructure(payload *model.WebhookPayload) error

	// ValidateContent performs content-level validation (size, format, etc.).
	// Returns nil on success, content validation error on failure.
	ValidateContent(payload *model.WebhookPayload) error
}

// HealthChecker defines the interface for system health monitoring.
// Following SRP: handles only health check concerns.
type HealthChecker interface {
	// Check performs comprehensive system health check.
	// Returns health status map with component statuses.
	Check() map[string]interface{}

	// CheckDisk validates disk space and filesystem health.
	// Returns disk status information.
	CheckDisk() map[string]interface{}

	// CheckMemory validates memory usage and availability.
	// Returns memory status information.
	CheckMemory() map[string]interface{}
}

// ConfigProvider defines the interface for configuration management.
// Following SRP: handles only configuration concerns.
type ConfigProvider interface {
	// GetStoragePath returns the configured storage directory path.
	GetStoragePath() string

	// GetMaxFileSize returns the maximum allowed file size in bytes.
	GetMaxFileSize() int64

	// GetPort returns the configured server port.
	GetPort() int

	// GetAuthToken returns the configured authentication token.
	GetAuthToken() string

	// IsProduction returns true if running in production mode.
	IsProduction() bool
}

// Package config provides configuration management for the Hokku webhook service.
// Following SOLID SRP principle: handles only configuration concerns.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds the application configuration following the ConfigProvider interface.
type Config struct {
	StoragePath string
	MaxFileSize int64
	Port        int
	AuthToken   string
	Environment string

	// Validation rules
	AllowedExtensions []string
	MaxTitleLength    int
	MaxDescLength     int
	MaxDataSize       int64
}

// Default configuration values
const (
	DefaultStoragePath    = "./storage"
	DefaultMaxFileSize    = 10 * 1024 * 1024 // 10MB
	DefaultPort           = 8080
	DefaultEnvironment    = "development"
	DefaultMaxTitleLength = 64
	DefaultMaxDescLength  = 512
	DefaultMaxDataSize    = 5 * 1024 * 1024 // 5MB for data field
)

// DefaultAllowedExtensions defines the safe file extensions allowed
var DefaultAllowedExtensions = []string{
	"json", "txt", "log", "csv", "xml", "yaml", "yml",
}

// New creates a new configuration instance with default values and environment overrides.
func New() *Config {
	cfg := &Config{
		StoragePath:       getEnvString("HOKKU_STORAGE_PATH", DefaultStoragePath),
		MaxFileSize:       getEnvInt64("HOKKU_MAX_FILE_SIZE", DefaultMaxFileSize),
		Port:              getEnvInt("HOKKU_PORT", DefaultPort),
		AuthToken:         getEnvString("HOKKU_AUTH_TOKEN", ""),
		Environment:       getEnvString("HOKKU_ENV", DefaultEnvironment),
		AllowedExtensions: getEnvStringSlice("HOKKU_ALLOWED_EXTENSIONS", DefaultAllowedExtensions),
		MaxTitleLength:    getEnvInt("HOKKU_MAX_TITLE_LENGTH", DefaultMaxTitleLength),
		MaxDescLength:     getEnvInt("HOKKU_MAX_DESC_LENGTH", DefaultMaxDescLength),
		MaxDataSize:       getEnvInt64("HOKKU_MAX_DATA_SIZE", DefaultMaxDataSize),
	}

	return cfg
}

// GetStoragePath returns the configured storage directory path.
func (c *Config) GetStoragePath() string {
	return c.StoragePath
}

// GetMaxFileSize returns the maximum allowed file size in bytes.
func (c *Config) GetMaxFileSize() int64 {
	return c.MaxFileSize
}

// GetPort returns the configured server port.
func (c *Config) GetPort() int {
	return c.Port
}

// GetAuthToken returns the configured authentication token.
func (c *Config) GetAuthToken() string {
	return c.AuthToken
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// GetAllowedExtensions returns the list of allowed file extensions.
func (c *Config) GetAllowedExtensions() []string {
	return c.AllowedExtensions
}

// GetMaxTitleLength returns the maximum allowed title length.
func (c *Config) GetMaxTitleLength() int {
	return c.MaxTitleLength
}

// GetMaxDescLength returns the maximum allowed description length.
func (c *Config) GetMaxDescLength() int {
	return c.MaxDescLength
}

// GetMaxDataSize returns the maximum allowed data field size.
func (c *Config) GetMaxDataSize() int64 {
	return c.MaxDataSize
}

// Validate performs configuration validation to ensure all required values are present
// and within acceptable ranges.
func (c *Config) Validate() error {
	if c.StoragePath == "" {
		return fmt.Errorf("storage path cannot be empty")
	}

	// Ensure storage path is absolute for security
	if !filepath.IsAbs(c.StoragePath) {
		abs, err := filepath.Abs(c.StoragePath)
		if err != nil {
			return fmt.Errorf("failed to resolve storage path: %w", err)
		}
		c.StoragePath = abs
	}

	if c.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive, got %d", c.MaxFileSize)
	}

	if c.MaxFileSize > 100*1024*1024 { // 100MB safety limit
		return fmt.Errorf("max file size too large: %d bytes (max 100MB)", c.MaxFileSize)
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port number: %d (must be 1-65535)", c.Port)
	}

	if c.IsProduction() && c.AuthToken == "" {
		return fmt.Errorf("auth token is required in production environment")
	}

	if c.MaxTitleLength <= 0 || c.MaxTitleLength > 1024 {
		return fmt.Errorf("invalid max title length: %d (must be 1-1024)", c.MaxTitleLength)
	}

	if c.MaxDescLength < 0 || c.MaxDescLength > 4096 {
		return fmt.Errorf("invalid max description length: %d (must be 0-4096)", c.MaxDescLength)
	}

	if c.MaxDataSize <= 0 {
		return fmt.Errorf("max data size must be positive, got %d", c.MaxDataSize)
	}

	// Validate allowed extensions
	for _, ext := range c.AllowedExtensions {
		if ext == "" {
			return fmt.Errorf("empty extension in allowed extensions list")
		}
		if strings.Contains(ext, ".") {
			return fmt.Errorf("extension should not contain dot: %s", ext)
		}
	}

	return nil
}

// Environment helper functions

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// Package errors provides centralized error definitions and utilities
// following Google Go style guidelines with sentinel errors.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure cases
// Following Google Go style: use errors.New for simple sentinel errors
var (
	// ErrInvalidPayload indicates the webhook payload is malformed or invalid
	ErrInvalidPayload = errors.New("invalid payload")

	// ErrUnauthorized indicates authentication failure
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInsufficientSpace indicates not enough disk space for operation
	ErrInsufficientSpace = errors.New("insufficient disk space")

	// ErrFileExists indicates file already exists when uniqueness is required
	ErrFileExists = errors.New("file already exists")

	// ErrInvalidPath indicates the file path is invalid or unsafe
	ErrInvalidPath = errors.New("invalid file path")

	// ErrValidationFailed indicates payload validation failed
	ErrValidationFailed = errors.New("validation failed")
)

// Error wrapping helpers following Google Go style
// Pattern: descriptive message with %w at the end

// WrapValidationError wraps validation errors with field context
func WrapValidationError(field string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("validation failed for field %s: %w", field, err)
}

// WrapFileError wraps file operation errors with context
func WrapFileError(operation, path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("file %s failed for %s: %w", operation, path, err)
}

// WrapConfigError wraps configuration-related errors
func WrapConfigError(key string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("configuration error for %s: %w", key, err)
}

// WrapDiskError wraps disk space or filesystem errors
func WrapDiskError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("disk operation %s failed: %w", operation, err)
}

// Package security provides security-focused utilities for the Hokku webhook service.
// Following Google Go security practices and defensive programming principles.
package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// ErrPathTraversal indicates a path traversal attempt was detected
	ErrPathTraversal = errors.New("path traversal attempt detected")
	// ErrInvalidFilename indicates the filename contains invalid characters or format
	ErrInvalidFilename = errors.New("invalid filename")
	// ErrUnsafeCharacters indicates unsafe characters were detected in the input
	ErrUnsafeCharacters = errors.New("unsafe characters detected")
	// ErrFilenameTooLong indicates the filename exceeds maximum length limits
	ErrFilenameTooLong = errors.New("filename too long")
	// ErrEmptyFilename indicates an empty filename was provided
	ErrEmptyFilename = errors.New("empty filename")
)

// Path security constants following OWASP guidelines
const (
	MaxFilenameLength = 255  // Standard filesystem limit
	MaxPathLength     = 4096 // Standard path limit
	MinRandomBytes    = 8    // Minimum bytes for secure random generation
)

// Unsafe characters that should never appear in filenames (including spaces)
var unsafeChars = regexp.MustCompile(`[<>:"/\\|?* \x00-\x1f]`)

// Reserved Windows filenames (for cross-platform compatibility)
var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true,
	"COM9": true, "LPT1": true, "LPT2": true, "LPT3": true,
	"LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true,
	"LPT8": true, "LPT9": true,
}

// ValidatePath validates that a file path is safe and doesn't contain path traversal attempts.
// It checks for:
// - Path traversal patterns (../, ..\)
// - Absolute paths when relative expected
// - Null bytes and other dangerous characters
// - Excessively long paths
func ValidatePath(path string, baseDir string) error {
	if path == "" {
		return ErrEmptyFilename
	}

	// Check for null bytes (security vulnerability)
	if strings.Contains(path, "\x00") {
		return ErrUnsafeCharacters
	}

	// Check path length
	if len(path) > MaxPathLength {
		return fmt.Errorf("path too long (%d > %d): %w", len(path), MaxPathLength, ErrInvalidFilename)
	}

	// Clean the path and check for traversal attempts
	cleaned := filepath.Clean(path)

	// Detect path traversal attempts
	if strings.Contains(cleaned, "..") {
		return ErrPathTraversal
	}

	// If baseDir is provided, ensure path stays within it
	if baseDir != "" {
		absPath := filepath.Join(baseDir, cleaned)
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve base directory: %w", err)
		}

		absFile, err := filepath.Abs(absPath)
		if err != nil {
			return fmt.Errorf("failed to resolve file path: %w", err)
		}

		// Ensure the resolved path is within the base directory
		relPath, err := filepath.Rel(absBase, absFile)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return ErrPathTraversal
		}
	}

	return nil
}

// SanitizeFilename creates a safe filename from user input.
// It:
// - Removes or replaces unsafe characters
// - Handles reserved names
// - Ensures proper length limits
// - Maintains readability while ensuring security
func SanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", ErrEmptyFilename
	}

	// Check for valid UTF-8
	if !utf8.ValidString(filename) {
		return "", ErrUnsafeCharacters
	}

	// Remove leading/trailing whitespace
	cleaned := strings.TrimSpace(filename)
	if cleaned == "" {
		return "", ErrEmptyFilename
	}

	// Replace unsafe characters with underscore
	sanitized := unsafeChars.ReplaceAllString(cleaned, "_")

	// Handle multiple consecutive underscores
	sanitized = regexp.MustCompile(`_+`).ReplaceAllString(sanitized, "_")

	// Remove leading/trailing underscores and dots (security best practice)
	sanitized = strings.Trim(sanitized, "_.")

	if sanitized == "" {
		return "", ErrInvalidFilename
	}

	// Check for Windows reserved names (cross-platform safety)
	name := strings.ToUpper(strings.SplitN(sanitized, ".", 2)[0])
	if reservedNames[name] {
		sanitized = "_" + sanitized
	}

	// Enforce length limits
	if len(sanitized) > MaxFilenameLength {
		// Preserve extension if present
		ext := filepath.Ext(sanitized)
		if len(ext) > 0 && len(ext) < 10 { // Reasonable extension length
			nameOnly := sanitized[:len(sanitized)-len(ext)]
			maxNameLength := MaxFilenameLength - len(ext)
			if maxNameLength > 0 {
				sanitized = nameOnly[:maxNameLength] + ext
			} else {
				sanitized = sanitized[:MaxFilenameLength]
			}
		} else {
			sanitized = sanitized[:MaxFilenameLength]
		}
	}

	return sanitized, nil
}

// GenerateSecureFilename creates a cryptographically secure filename with optional prefix.
// Format: prefix_timestamp_randomhex.extension
func GenerateSecureFilename(prefix, extension string) (string, error) {
	if prefix != "" {
		var err error
		prefix, err = SanitizeFilename(prefix)
		if err != nil {
			return "", fmt.Errorf("invalid prefix: %w", err)
		}
	}

	// Generate cryptographically secure random bytes
	randomBytes := make([]byte, MinRandomBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}

	randomHex := hex.EncodeToString(randomBytes)

	// Build filename
	var parts []string
	if prefix != "" {
		parts = append(parts, prefix)
	}
	parts = append(parts, randomHex)

	filename := strings.Join(parts, "_")

	if extension != "" {
		// Sanitize extension
		cleanExt := strings.TrimPrefix(extension, ".")
		cleanExt, err := SanitizeFilename(cleanExt)
		if err != nil {
			return "", fmt.Errorf("invalid extension: %w", err)
		}
		filename += "." + cleanExt
	}

	return filename, nil
}

// IsSecurePath performs comprehensive security validation on a file path.
// It combines path validation and filename validation for complete security.
func IsSecurePath(fullPath, baseDir string) error {
	// Validate the path structure
	if err := ValidatePath(fullPath, baseDir); err != nil {
		return err
	}

	// Validate the filename component
	filename := filepath.Base(fullPath)
	_, err := SanitizeFilename(filename)
	if err != nil {
		return fmt.Errorf("invalid filename in path: %w", err)
	}

	return nil
}

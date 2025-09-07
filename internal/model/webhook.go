// Package model defines the core data structures for the Hokku webhook service.
// Following SOLID SRP principle: this package handles only data representation.
package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WebhookPayload represents the incoming webhook data structure.
// Validation tags follow go-playground/validator conventions.
type WebhookPayload struct {
	// Core required fields
	Title       string                 `json:"title" validate:"required,max=64"`
	Description string                 `json:"description,omitempty" validate:"max=512"`
	Data        map[string]interface{} `json:"data" validate:"required"`

	// Metadata fields (auto-populated)
	ID        string    `json:"id,omitempty"`        // UUID generated server-side
	Timestamp time.Time `json:"timestamp,omitempty"` // Server timestamp

	// Optional webhook source information
	Source string `json:"source,omitempty" validate:"omitempty,max=128"`
	Type   string `json:"type,omitempty" validate:"omitempty,max=32"`
}

// GenerateID generates a new UUID for the payload if not already set
func (p *WebhookPayload) GenerateID() {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
}

// SetTimestamp sets the current timestamp if not already set
func (p *WebhookPayload) SetTimestamp() {
	if p.Timestamp.IsZero() {
		p.Timestamp = time.Now().UTC()
	}
}

// String returns a JSON representation of the payload for logging
func (p *WebhookPayload) String() string {
	// Create a copy without sensitive data for logging
	logPayload := struct {
		ID          string    `json:"id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Source      string    `json:"source,omitempty"`
		Type        string    `json:"type,omitempty"`
		Timestamp   time.Time `json:"timestamp"`
		DataKeys    []string  `json:"data_keys"` // Only include keys, not values
	}{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Source:      p.Source,
		Type:        p.Type,
		Timestamp:   p.Timestamp,
	}

	// Extract data keys for logging without exposing values
	for key := range p.Data {
		logPayload.DataKeys = append(logPayload.DataKeys, key)
	}

	b, _ := json.Marshal(logPayload)
	return string(b)
}

// GetFileName generates a filename for storing this payload
// Format: YYYY-MM-DD_HH-MM-SS_UUID_title
func (p *WebhookPayload) GetFileName() string {
	timestamp := p.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	// Sanitize title for filename (basic sanitization)
	safeTitle := sanitizeForFilename(p.Title)

	return fmt.Sprintf("%s_%s_%s.json",
		timestamp.Format("2006-01-02_15-04-05"),
		p.ID,
		safeTitle,
	)
}

// sanitizeForFilename removes unsafe characters from title for filename use
func sanitizeForFilename(title string) string {
	// Simple sanitization - replace unsafe chars with underscore
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := title
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length to reasonable filename length
	if len(result) > 32 {
		result = result[:32]
	}

	return result
}

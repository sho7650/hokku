package service

import (
	"encoding/json"
	"fmt"
	"hokku/internal/config"
	"hokku/internal/model"
	"hokku/pkg/errors"
	"reflect"
	"strings"
	"unicode/utf8"
)

// PayloadValidatorImpl implements the PayloadValidator interface for webhook payload validation.
// Following SOLID SRP: handles only validation concerns.
type PayloadValidatorImpl struct {
	config *config.Config
}

// NewPayloadValidator creates a new PayloadValidator implementation with the provided configuration.
func NewPayloadValidator(cfg *config.Config) PayloadValidator {
	return &PayloadValidatorImpl{
		config: cfg,
	}
}

// Security and validation constants
const (
	MaxNestingDepth = 5     // Maximum nesting depth for JSON data
	MaxStringLength = 10000 // Maximum string length in data fields
	MaxArrayLength  = 1000  // Maximum array length
	MaxObjectKeys   = 100   // Maximum number of keys in an object
)

// Reserved field names that cannot be used in the data field
var reservedFieldNames = map[string]bool{
	"id":          true,
	"timestamp":   true,
	"title":       true,
	"description": true,
	"source":      true,
	"type":        true,
}

// Validate performs comprehensive validation of webhook payload.
// Returns nil on success, validation error on failure.
func (v *PayloadValidatorImpl) Validate(payload *model.WebhookPayload) error {
	if payload == nil {
		return errors.WrapValidationError("payload", fmt.Errorf("payload cannot be nil"))
	}

	// Perform structural validation first
	if err := v.ValidateStructure(payload); err != nil {
		return err
	}

	// Perform content validation
	if err := v.ValidateContent(payload); err != nil {
		return err
	}

	// Perform business rule validation
	if err := v.validateBusinessRules(payload); err != nil {
		return err
	}

	return nil
}

// ValidateStructure validates the payload structure and required fields.
// Returns nil on success, structural validation error on failure.
func (v *PayloadValidatorImpl) ValidateStructure(payload *model.WebhookPayload) error {
	if payload == nil {
		return errors.WrapValidationError("payload", fmt.Errorf("payload cannot be nil"))
	}

	// Validate required fields
	if strings.TrimSpace(payload.Title) == "" {
		return errors.WrapValidationError("title", fmt.Errorf("title is required"))
	}

	if payload.Data == nil {
		return errors.WrapValidationError("data", fmt.Errorf("data field is required"))
	}

	if len(payload.Data) == 0 {
		return errors.WrapValidationError("data", fmt.Errorf("data field cannot be empty"))
	}

	return nil
}

// ValidateContent performs content-level validation (size, format, etc.).
// Returns nil on success, content validation error on failure.
func (v *PayloadValidatorImpl) ValidateContent(payload *model.WebhookPayload) error {
	if payload == nil {
		return errors.WrapValidationError("payload", fmt.Errorf("payload cannot be nil"))
	}

	// Validate title length and content
	if err := v.validateString("title", payload.Title, 1, v.config.GetMaxTitleLength(), true); err != nil {
		return err
	}

	// Validate description if provided
	if payload.Description != "" {
		if err := v.validateString("description", payload.Description, 0, v.config.GetMaxDescLength(), false); err != nil {
			return err
		}
	}

	// Validate source if provided
	if payload.Source != "" {
		if err := v.validateString("source", payload.Source, 0, 128, false); err != nil {
			return err
		}
	}

	// Validate type if provided
	if payload.Type != "" {
		if err := v.validateString("type", payload.Type, 0, 32, false); err != nil {
			return err
		}
	}

	// Validate data field size and structure
	if err := v.validateDataField(payload.Data); err != nil {
		return err
	}

	return nil
}

// validateBusinessRules performs business-specific validation rules.
func (v *PayloadValidatorImpl) validateBusinessRules(payload *model.WebhookPayload) error {
	// Check for reserved field names in data
	for fieldName := range payload.Data {
		if reservedFieldNames[strings.ToLower(fieldName)] {
			return errors.WrapValidationError("data",
				fmt.Errorf("reserved field name '%s' not allowed in data", fieldName))
		}
	}

	// Validate source format if provided (basic domain validation)
	if payload.Source != "" {
		if err := v.validateSourceFormat(payload.Source); err != nil {
			return errors.WrapValidationError("source", err)
		}
	}

	// Validate type format if provided (basic event type format)
	if payload.Type != "" {
		if err := v.validateTypeFormat(payload.Type); err != nil {
			return errors.WrapValidationError("type", err)
		}
	}

	return nil
}

// validateString performs comprehensive string validation including UTF-8, length, and security checks.
func (v *PayloadValidatorImpl) validateString(fieldName, value string, minLen, maxLen int, required bool) error {
	// Check if required field is empty
	if required && strings.TrimSpace(value) == "" {
		return errors.WrapValidationError(fieldName, fmt.Errorf("%s is required", fieldName))
	}

	// Check UTF-8 validity
	if !utf8.ValidString(value) {
		return errors.WrapValidationError(fieldName, fmt.Errorf("invalid UTF-8 in %s", fieldName))
	}

	// Check for unsafe characters (null bytes and control characters)
	if v.containsUnsafeCharacters(value) {
		return errors.WrapValidationError(fieldName, fmt.Errorf("unsafe characters in %s", fieldName))
	}

	// Check length constraints
	if len(value) < minLen {
		return errors.WrapValidationError(fieldName,
			fmt.Errorf("%s too short: %d < %d", fieldName, len(value), minLen))
	}

	if len(value) > maxLen {
		return errors.WrapValidationError(fieldName,
			fmt.Errorf("%s too long: %d > %d", fieldName, len(value), maxLen))
	}

	return nil
}

// containsUnsafeCharacters checks for null bytes and control characters that could be security risks.
func (v *PayloadValidatorImpl) containsUnsafeCharacters(s string) bool {
	for _, r := range s {
		// Check for null byte
		if r == '\x00' {
			return true
		}
		// Check for control characters (except common whitespace)
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

// validateDataField performs comprehensive validation of the data field including size and structure.
func (v *PayloadValidatorImpl) validateDataField(data map[string]interface{}) error {
	// Marshal to JSON to check serialized size
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.WrapValidationError("data", fmt.Errorf("data field cannot be serialized to JSON: %w", err))
	}

	// Check serialized size against configured limit
	if int64(len(jsonData)) > v.config.GetMaxDataSize() {
		return errors.WrapValidationError("data",
			fmt.Errorf("data size %d bytes exceeds limit %d bytes", len(jsonData), v.config.GetMaxDataSize()))
	}

	// Validate data structure recursively
	if err := v.validateDataStructure(data, 0); err != nil {
		return errors.WrapValidationError("data", err)
	}

	return nil
}

// validateDataStructure recursively validates the structure and content of data fields.
func (v *PayloadValidatorImpl) validateDataStructure(value interface{}, depth int) error {
	// Check nesting depth to prevent stack overflow and excessive complexity
	if depth > MaxNestingDepth {
		return fmt.Errorf("nesting depth %d exceeds maximum %d", depth, MaxNestingDepth)
	}

	switch val := value.(type) {
	case string:
		// Validate string length and content
		if len(val) > MaxStringLength {
			return fmt.Errorf("string length %d exceeds maximum %d", len(val), MaxStringLength)
		}
		if !utf8.ValidString(val) {
			return fmt.Errorf("invalid UTF-8 string")
		}
		if v.containsUnsafeCharacters(val) {
			return fmt.Errorf("unsafe characters in string value")
		}

	case map[string]interface{}:
		// Validate object key count
		if len(val) > MaxObjectKeys {
			return fmt.Errorf("object key count %d exceeds maximum %d", len(val), MaxObjectKeys)
		}

		// Recursively validate each value
		for key, subValue := range val {
			// Validate key
			if len(key) > 100 {
				return fmt.Errorf("object key '%s' too long", key)
			}
			if !utf8.ValidString(key) {
				return fmt.Errorf("invalid UTF-8 in object key: %s", key)
			}

			// Recursively validate value
			if err := v.validateDataStructure(subValue, depth+1); err != nil {
				return fmt.Errorf("in object key '%s': %w", key, err)
			}
		}

	case []interface{}:
		// Validate array length
		if len(val) > MaxArrayLength {
			return fmt.Errorf("array length %d exceeds maximum %d", len(val), MaxArrayLength)
		}

		// Recursively validate each element
		for i, element := range val {
			if err := v.validateDataStructure(element, depth+1); err != nil {
				return fmt.Errorf("in array index %d: %w", i, err)
			}
		}

	case float64, int, int64, bool, nil:
		// These primitive types are always valid
		return nil

	default:
		// Check for other numeric types
		valType := reflect.TypeOf(value)
		if valType != nil {
			switch valType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				return nil
			}
		}

		return fmt.Errorf("unsupported data type: %T", value)
	}

	return nil
}

// validateSourceFormat validates the format of the source field (basic domain validation).
func (v *PayloadValidatorImpl) validateSourceFormat(source string) error {
	// Basic validation - no spaces, reasonable characters
	if strings.Contains(source, " ") {
		return fmt.Errorf("source cannot contain spaces")
	}

	// Check for basic domain-like format (optional)
	if strings.Contains(source, "..") {
		return fmt.Errorf("source format invalid: consecutive dots")
	}

	// Check for dangerous characters
	dangerousChars := []string{"<", ">", "\"", "'", "&", ";", "|"}
	for _, char := range dangerousChars {
		if strings.Contains(source, char) {
			return fmt.Errorf("source contains unsafe character: %s", char)
		}
	}

	return nil
}

// validateTypeFormat validates the format of the type field (basic event type format).
func (v *PayloadValidatorImpl) validateTypeFormat(eventType string) error {
	// Basic validation - no spaces
	if strings.Contains(eventType, " ") {
		return fmt.Errorf("type cannot contain spaces")
	}

	// Check for dangerous characters
	dangerousChars := []string{"<", ">", "\"", "'", "&", ";", "|"}
	for _, char := range dangerousChars {
		if strings.Contains(eventType, char) {
			return fmt.Errorf("type contains unsafe character: %s", char)
		}
	}

	// Optional: Check for reasonable event type format (e.g., "resource.action")
	if strings.HasPrefix(eventType, ".") || strings.HasSuffix(eventType, ".") {
		return fmt.Errorf("type cannot start or end with dot")
	}

	if strings.Contains(eventType, "..") {
		return fmt.Errorf("type cannot contain consecutive dots")
	}

	return nil
}

# Hokku Implementation Workflow

**Version:** 1.0.0  
**Date:** 2025-09-07  
**Author:** Development Team  

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Phase Structure](#phase-structure)
4. [Implementation Phases](#implementation-phases)
5. [Quality Gates](#quality-gates)
6. [Parallel Execution Plan](#parallel-execution-plan)
7. [Cross-Session Management](#cross-session-management)
8. [SOLID Compliance Checklist](#solid-compliance-checklist)
9. [Commands Reference](#commands-reference)
10. [Session Templates](#session-templates)

---

## Overview

This workflow implements the Hokku webhook file storage service using Test-Driven Development (TDD), SOLID principles, and Go best practices. The implementation is divided into 5 phases with clear dependencies, quality gates, and cross-session support.

### Key Principles
- **Interface-First Design**: Define contracts before implementations
- **Test-Driven Development**: Write tests before implementation code
- **SOLID Compliance**: Single responsibility, open-closed, Liskov substitution, interface segregation, dependency inversion
- **Security-First**: Path validation and authentication throughout
- **Quality Gates**: 80%+ test coverage, linting, security validation

---

## Prerequisites

### Required Tools
```bash
# Core Go toolchain
go version  # Must be 1.21+

# Quality tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Testing tools  
go install github.com/stretchr/testify@latest
go install gotest.tools/gotestsum@latest

# Development tools
go install github.com/cosmtrek/air@latest  # Hot reload
```

### Project Initialization
```bash
# 1. Initialize Go module
go mod init github.com/yourusername/hokku

# 2. Install core dependencies
go get github.com/gin-gonic/gin@latest
go get go.uber.org/zap@latest
go get github.com/spf13/viper@latest
go get github.com/go-playground/validator/v10@latest
go get github.com/google/uuid@latest

# 3. Install test dependencies
go get github.com/stretchr/testify@latest
go get github.com/golang/mock/gomock@latest
```

---

## Phase Structure

### Dependencies Map
```
Phase 1 (Foundation) 
    ├── Phase 2 (Services) - depends on interfaces from Phase 1
    ├── Phase 3 (Handlers) - depends on Phase 2 services
    └── Phase 4 (Integration) - depends on all previous phases
            └── Phase 5 (Production) - depends on Phase 4
```

### Parallel Execution Opportunities
- **Phase 1**: Interfaces and models can be developed in parallel
- **Phase 2**: File writer and validator services are independent
- **Phase 3**: Health and webhook handlers can be developed concurrently
- **Phase 4**: Authentication and configuration are independent
- **Phase 5**: Docker and deployment scripts are independent

---

## Implementation Phases

### Phase 1: Foundation & Interfaces (Days 1-2)

**Goal**: Establish project structure, interfaces, and core models

#### Tasks

**1.1 Project Structure Setup** (Parallel)
```bash
mkdir -p {cmd/hokku,internal/{config,handler,middleware,model,service,util},pkg/logger,test/{unit,integration,testdata},config}
touch .gitignore Makefile README.md
```

**1.2 Core Interfaces Definition** (Parallel)
- **File**: `internal/service/interfaces.go`
- **Dependencies**: None
- **Parallel with**: 1.3, 1.4

```go
package service

import (
    "context"
    "github.com/yourusername/hokku/internal/model"
)

// FileWriter handles file operations
type FileWriter interface {
    Write(ctx context.Context, payload *model.WebhookPayload) (string, error)
    Exists(path string) (bool, error)
    ValidateSpace(requiredBytes int64) error
}

// Validator handles payload validation
type Validator interface {
    ValidatePayload(payload *model.WebhookPayload) error
    ValidatePath(path string) error
    ValidateFileName(name string) error
}

// Logger interface for dependency injection
type Logger interface {
    Info(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
    Debug(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
}
```

**1.3 Data Models** (Parallel)
- **File**: `internal/model/payload.go`
- **Dependencies**: None
- **Parallel with**: 1.2, 1.4

```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type WebhookPayload struct {
    // Required fields
    Title    string `json:"title" validate:"required,max=64"`
    FileName string `json:"filename" validate:"required,filepath,max=255"`
    Body     string `json:"body" validate:"required"`
    
    // Optional fields
    Path        string                 `json:"path,omitempty" validate:"omitempty,dirpath"`
    ContentType string                 `json:"content_type,omitempty" validate:"omitempty,oneof=text/plain text/markdown text/html application/json"`
    Encoding    string                 `json:"encoding,omitempty" validate:"omitempty,oneof=utf8 base64"`
    Author      string                 `json:"author,omitempty" validate:"max=100"`
    Source      string                 `json:"source,omitempty" validate:"omitempty,url"`
    Version     string                 `json:"version,omitempty" validate:"omitempty,semver"`
    Tags        []string               `json:"tags,omitempty" validate:"dive,max=50"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    
    // System fields
    UUID      uuid.UUID `json:"-"`
    Timestamp time.Time `json:"-"`
}
```

**1.4 Response Models** (Parallel)
- **File**: `internal/model/response.go`
- **Dependencies**: None
- **Parallel with**: 1.2, 1.3

```go
package model

import "time"

type WebhookResponse struct {
    Success   bool      `json:"success"`
    Message   string    `json:"message"`
    UUID      string    `json:"uuid,omitempty"`
    Timestamp time.Time `json:"timestamp"`
    Path      string    `json:"path,omitempty"`
}

type ErrorResponse struct {
    Success   bool      `json:"success"`
    Error     string    `json:"error"`
    Code      string    `json:"code,omitempty"`
    Details   []string  `json:"details,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

**Quality Gate 1**: 
- [ ] All interfaces compile without errors
- [ ] go vet passes
- [ ] golangci-lint passes with no issues
- [ ] All models have proper JSON tags
- [ ] Validation tags are syntactically correct

### Phase 2: Core Services (Days 3-5)

**Goal**: Implement file writing and validation services with comprehensive tests

**Dependencies**: Phase 1 complete

#### Tasks

**2.1 File Writer Service (TDD)** (Sequential: Tests → Implementation)
- **File**: `test/unit/service/file_writer_test.go` → `internal/service/file_writer.go`
- **Dependencies**: Phase 1 interfaces
- **Parallel with**: 2.2

**Test First** (`test/unit/service/file_writer_test.go`):
```go
package service_test

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/google/uuid"
    "go.uber.org/zap/zaptest"
    
    "github.com/yourusername/hokku/internal/model"
    "github.com/yourusername/hokku/internal/service"
)

func TestFileWriter_Write(t *testing.T) {
    tests := []struct {
        name    string
        payload *model.WebhookPayload
        wantErr bool
        errMsg  string
    }{
        {
            name: "successful_write",
            payload: &model.WebhookPayload{
                Title:    "Test Document",
                FileName: "test.md",
                Body:     "# Test Content",
                Path:     "docs",
                UUID:     uuid.New(),
            },
            wantErr: false,
        },
        {
            name: "directory_traversal_attack",
            payload: &model.WebhookPayload{
                Title:    "Malicious",
                FileName: "../../../etc/passwd",
                Body:     "malicious content",
                UUID:     uuid.New(),
            },
            wantErr: true,
            errMsg:  "invalid path",
        },
        {
            name: "duplicate_file_error",
            payload: &model.WebhookPayload{
                Title:    "Duplicate",
                FileName: "existing.txt",
                Body:     "content",
                UUID:     uuid.New(),
            },
            wantErr: true,
            errMsg:  "file already exists",
        },
        {
            name: "base64_encoding",
            payload: &model.WebhookPayload{
                Title:    "Encoded",
                FileName: "encoded.txt",
                Body:     "SGVsbG8gV29ybGQ=", // "Hello World" in base64
                Encoding: "base64",
                UUID:     uuid.New(),
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tempDir := t.TempDir()
            logger := zaptest.NewLogger(t)
            
            config := &service.StorageConfig{
                BaseDir:         tempDir,
                DiskQuota:       1073741824, // 1GB
                FilePermissions: 0644,
                DirPermissions:  0755,
            }
            
            fw := service.NewFileWriter(config, logger)
            
            // Setup existing file for duplicate test
            if tt.name == "duplicate_file_error" {
                existingPath := filepath.Join(tempDir, tt.payload.FileName)
                require.NoError(t, os.WriteFile(existingPath, []byte("existing"), 0644))
            }
            
            ctx := context.Background()
            path, err := fw.Write(ctx, tt.payload)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, path)
                
                // Verify file content
                content, err := os.ReadFile(path)
                assert.NoError(t, err)
                
                expectedContent := tt.payload.Body
                if tt.payload.Encoding == "base64" {
                    expectedContent = "Hello World"
                }
                assert.Equal(t, expectedContent, string(content))
            }
        })
    }
}

func TestFileWriter_Exists(t *testing.T) {
    tempDir := t.TempDir()
    logger := zaptest.NewLogger(t)
    config := &service.StorageConfig{BaseDir: tempDir}
    fw := service.NewFileWriter(config, logger)
    
    // Create a test file
    testFile := filepath.Join(tempDir, "test.txt")
    require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))
    
    exists, err := fw.Exists(testFile)
    assert.NoError(t, err)
    assert.True(t, exists)
    
    exists, err = fw.Exists(filepath.Join(tempDir, "nonexistent.txt"))
    assert.NoError(t, err)
    assert.False(t, exists)
}

func TestFileWriter_ValidateSpace(t *testing.T) {
    tempDir := t.TempDir()
    logger := zaptest.NewLogger(t)
    config := &service.StorageConfig{
        BaseDir:   tempDir,
        DiskQuota: 1024, // 1KB quota
    }
    fw := service.NewFileWriter(config, logger)
    
    // Should pass for small requirement
    err := fw.ValidateSpace(512)
    assert.NoError(t, err)
    
    // Should fail for requirement exceeding quota
    err = fw.ValidateSpace(2048)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "insufficient disk space")
}
```

**Implementation** (`internal/service/file_writer.go`):
```go
package service

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "path/filepath"
    "syscall"
    
    "github.com/yourusername/hokku/internal/model"
    "github.com/yourusername/hokku/internal/util"
    "go.uber.org/zap"
)

type StorageConfig struct {
    BaseDir         string
    DiskQuota       int64
    FilePermissions os.FileMode
    DirPermissions  os.FileMode
}

type fileWriter struct {
    config *StorageConfig
    logger *zap.Logger
}

func NewFileWriter(config *StorageConfig, logger *zap.Logger) FileWriter {
    return &fileWriter{
        config: config,
        logger: logger,
    }
}

func (fw *fileWriter) Write(ctx context.Context, payload *model.WebhookPayload) (string, error) {
    // Validate space before writing
    contentSize := int64(len(payload.Body))
    if payload.Encoding == "base64" {
        // Base64 is ~33% larger than decoded content
        contentSize = contentSize * 3 / 4
    }
    
    if err := fw.ValidateSpace(contentSize); err != nil {
        return "", fmt.Errorf("space validation failed: %w", err)
    }
    
    // Build and validate path
    fullPath := filepath.Join(fw.config.BaseDir, payload.Path, payload.FileName)
    if err := util.ValidatePath(fullPath, fw.config.BaseDir); err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    
    if err := util.ValidateFileName(payload.FileName); err != nil {
        return "", fmt.Errorf("invalid filename: %w", err)
    }
    
    // Check if file already exists
    if exists, err := fw.Exists(fullPath); err != nil {
        return "", fmt.Errorf("existence check failed: %w", err)
    } else if exists {
        return "", fmt.Errorf("file already exists: %s", fullPath)
    }
    
    // Create directory
    dir := filepath.Dir(fullPath)
    if err := os.MkdirAll(dir, fw.config.DirPermissions); err != nil {
        return "", fmt.Errorf("failed to create directory: %w", err)
    }
    
    // Prepare content
    content := payload.Body
    if payload.Encoding == "base64" {
        decoded, err := base64.StdEncoding.DecodeString(payload.Body)
        if err != nil {
            return "", fmt.Errorf("base64 decode failed: %w", err)
        }
        content = string(decoded)
    }
    
    // Write file atomically (write to temp, then rename)
    tempPath := fullPath + ".tmp"
    if err := os.WriteFile(tempPath, []byte(content), fw.config.FilePermissions); err != nil {
        return "", fmt.Errorf("failed to write temp file: %w", err)
    }
    
    if err := os.Rename(tempPath, fullPath); err != nil {
        os.Remove(tempPath) // Cleanup temp file
        return "", fmt.Errorf("failed to finalize file: %w", err)
    }
    
    fw.logger.Info("File written successfully",
        zap.String("path", fullPath),
        zap.String("uuid", payload.UUID.String()),
        zap.Int("size", len(content)))
    
    return fullPath, nil
}

func (fw *fileWriter) Exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, err
}

func (fw *fileWriter) ValidateSpace(requiredBytes int64) error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(fw.config.BaseDir, &stat); err != nil {
        return fmt.Errorf("failed to get disk stats: %w", err)
    }
    
    availableBytes := int64(stat.Bavail) * int64(stat.Bsize)
    
    if requiredBytes > availableBytes {
        return fmt.Errorf("insufficient disk space: required=%d, available=%d", 
            requiredBytes, availableBytes)
    }
    
    if fw.config.DiskQuota > 0 && requiredBytes > fw.config.DiskQuota {
        return fmt.Errorf("exceeds disk quota: required=%d, quota=%d", 
            requiredBytes, fw.config.DiskQuota)
    }
    
    return nil
}
```

**2.2 Validator Service (TDD)** (Parallel with 2.1)
- **File**: `test/unit/service/validator_test.go` → `internal/service/validator.go`
- **Dependencies**: Phase 1 interfaces
- **Parallel with**: 2.1

**Test First** (`test/unit/service/validator_test.go`):
```go
package service_test

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/google/uuid"
    
    "github.com/yourusername/hokku/internal/model"
    "github.com/yourusername/hokku/internal/service"
)

func TestValidator_ValidatePayload(t *testing.T) {
    validator := service.NewValidator(&service.ValidationConfig{
        MaxPayloadSize: 10485760, // 10MB
        AllowedExtensions: []string{".txt", ".md", ".json", ".html"},
        ForbiddenPatterns: []string{"../", "..", "~"},
    })
    
    tests := []struct {
        name    string
        payload *model.WebhookPayload
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid_payload",
            payload: &model.WebhookPayload{
                Title:       "Valid Document",
                FileName:    "document.md",
                Body:        "# Hello World",
                Path:        "docs/2025",
                ContentType: "text/markdown",
                Author:      "John Doe",
                UUID:        uuid.New(),
            },
            wantErr: false,
        },
        {
            name: "missing_title",
            payload: &model.WebhookPayload{
                FileName: "test.txt",
                Body:     "content",
                UUID:     uuid.New(),
            },
            wantErr: true,
            errMsg:  "title is required",
        },
        {
            name: "title_too_long",
            payload: &model.WebhookPayload{
                Title:    string(make([]byte, 65)), // 65 chars, exceeds 64 limit
                FileName: "test.txt",
                Body:     "content",
                UUID:     uuid.New(),
            },
            wantErr: true,
            errMsg:  "title too long",
        },
        {
            name: "invalid_extension",
            payload: &model.WebhookPayload{
                Title:    "Test",
                FileName: "test.exe",
                Body:     "content",
                UUID:     uuid.New(),
            },
            wantErr: true,
            errMsg:  "file extension not allowed",
        },
        {
            name: "directory_traversal",
            payload: &model.WebhookPayload{
                Title:    "Test",
                FileName: "test.txt",
                Body:     "content",
                Path:     "../../../etc",
                UUID:     uuid.New(),
            },
            wantErr: true,
            errMsg:  "invalid path",
        },
        {
            name: "invalid_content_type",
            payload: &model.WebhookPayload{
                Title:       "Test",
                FileName:    "test.txt",
                Body:        "content",
                ContentType: "application/pdf",
                UUID:        uuid.New(),
            },
            wantErr: true,
            errMsg:  "invalid content type",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.ValidatePayload(tt.payload)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**2.3 Security Utilities (TDD)** (Sequential after 2.1, 2.2 tests)
- **File**: `test/unit/util/security_test.go` → `internal/util/security.go`
- **Dependencies**: Tests from 2.1, 2.2 identify security requirements

```go
package util

import (
    "errors"
    "path/filepath"
    "strings"
)

var (
    ErrInvalidPath     = errors.New("invalid path")
    ErrInvalidFileName = errors.New("invalid filename")
    ErrPathTraversal   = errors.New("path traversal attempt")
)

func ValidatePath(path, baseDir string) error {
    // Clean and make absolute
    cleanPath := filepath.Clean(path)
    absBase, err := filepath.Abs(baseDir)
    if err != nil {
        return fmt.Errorf("invalid base directory: %w", err)
    }
    
    absPath, err := filepath.Abs(cleanPath)
    if err != nil {
        return ErrInvalidPath
    }
    
    // Ensure path is within base directory
    relPath, err := filepath.Rel(absBase, absPath)
    if err != nil || strings.HasPrefix(relPath, "..") {
        return ErrPathTraversal
    }
    
    // Check for dangerous patterns
    dangerousPatterns := []string{"..", "~", "//", "\\"}
    for _, pattern := range dangerousPatterns {
        if strings.Contains(cleanPath, pattern) {
            return ErrInvalidPath
        }
    }
    
    return nil
}

func ValidateFileName(name string) error {
    if name == "" {
        return ErrInvalidFileName
    }
    
    // Check length
    if len(name) > 255 {
        return ErrInvalidFileName
    }
    
    // Check for invalid characters
    invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00", "\n", "\r", "\t"}
    for _, char := range invalidChars {
        if strings.Contains(name, char) {
            return ErrInvalidFileName
        }
    }
    
    // Check for Windows reserved names
    reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "LPT1", "LPT2", "LPT3"}
    nameUpper := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
    for _, r := range reserved {
        if nameUpper == r {
            return ErrInvalidFileName
        }
    }
    
    return nil
}
```

**Quality Gate 2**:
- [ ] All service tests pass
- [ ] Test coverage >= 80% for service layer
- [ ] Security utilities prevent path traversal
- [ ] File operations are atomic
- [ ] Error handling is comprehensive
- [ ] Interfaces are properly implemented

### Phase 3: HTTP Handlers (Days 6-8)

**Goal**: Implement HTTP handlers with comprehensive error handling

**Dependencies**: Phase 2 complete

#### Tasks

**3.1 Webhook Handler (TDD)** (Sequential: Tests → Implementation)

**Test First** (`test/unit/handler/webhook_test.go`):
```go
package handler_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "go.uber.org/zap/zaptest"
    
    "github.com/yourusername/hokku/internal/handler"
    "github.com/yourusername/hokku/internal/model"
    "github.com/yourusername/hokku/test/mocks"
)

func TestWebhookHandler_HandleWebhook(t *testing.T) {
    tests := []struct {
        name           string
        payload        interface{}
        setupMocks     func(*mocks.MockFileWriter, *mocks.MockValidator)
        expectedStatus int
        expectedBody   string
    }{
        {
            name: "successful_webhook",
            payload: map[string]interface{}{
                "title":    "Test Document",
                "filename": "test.md",
                "body":     "# Hello World",
            },
            setupMocks: func(fw *mocks.MockFileWriter, v *mocks.MockValidator) {
                v.On("ValidatePayload", mock.AnythingOfType("*model.WebhookPayload")).Return(nil)
                fw.On("Write", mock.Anything, mock.AnythingOfType("*model.WebhookPayload")).Return("/path/to/test.md", nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name: "invalid_json",
            payload: "invalid json",
            expectedStatus: http.StatusBadRequest,
        },
        {
            name: "validation_failure",
            payload: map[string]interface{}{
                "title":    "",  // Invalid: empty title
                "filename": "test.md",
                "body":     "content",
            },
            setupMocks: func(fw *mocks.MockFileWriter, v *mocks.MockValidator) {
                v.On("ValidatePayload", mock.AnythingOfType("*model.WebhookPayload")).Return(errors.New("title is required"))
            },
            expectedStatus: http.StatusBadRequest,
        },
        {
            name: "file_write_failure",
            payload: map[string]interface{}{
                "title":    "Test",
                "filename": "test.md",
                "body":     "content",
            },
            setupMocks: func(fw *mocks.MockFileWriter, v *mocks.MockValidator) {
                v.On("ValidatePayload", mock.AnythingOfType("*model.WebhookPayload")).Return(nil)
                fw.On("Write", mock.Anything, mock.AnythingOfType("*model.WebhookPayload")).Return("", errors.New("disk full"))
            },
            expectedStatus: http.StatusInternalServerError,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            gin.SetMode(gin.TestMode)
            mockFileWriter := mocks.NewMockFileWriter(t)
            mockValidator := mocks.NewMockValidator(t)
            logger := zaptest.NewLogger(t)
            
            if tt.setupMocks != nil {
                tt.setupMocks(mockFileWriter, mockValidator)
            }
            
            handler := handler.NewWebhookHandler(mockFileWriter, mockValidator, logger)
            
            // Prepare request
            var body bytes.Buffer
            if str, ok := tt.payload.(string); ok {
                body.WriteString(str)
            } else {
                json.NewEncoder(&body).Encode(tt.payload)
            }
            
            req := httptest.NewRequest(http.MethodPost, "/webhook", &body)
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()
            
            c, _ := gin.CreateTestContext(w)
            c.Request = req
            
            // Execute
            handler.HandleWebhook(c)
            
            // Assert
            assert.Equal(t, tt.expectedStatus, w.Code)
            
            var response map[string]interface{}
            err := json.Unmarshal(w.Body.Bytes(), &response)
            assert.NoError(t, err)
            
            if tt.expectedStatus == http.StatusOK {
                assert.True(t, response["success"].(bool))
                assert.NotEmpty(t, response["uuid"])
                assert.NotEmpty(t, response["path"])
            } else {
                assert.False(t, response["success"].(bool))
                assert.NotEmpty(t, response["error"])
            }
        })
    }
}
```

**Implementation** (`internal/handler/webhook.go`):
```go
package handler

import (
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "go.uber.org/zap"
    
    "github.com/yourusername/hokku/internal/model"
    "github.com/yourusername/hokku/internal/service"
)

type WebhookHandler struct {
    fileWriter service.FileWriter
    validator  service.Validator
    logger     *zap.Logger
}

func NewWebhookHandler(fw service.FileWriter, v service.Validator, logger *zap.Logger) *WebhookHandler {
    return &WebhookHandler{
        fileWriter: fw,
        validator:  v,
        logger:     logger,
    }
}

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
    var payload model.WebhookPayload
    
    if err := c.ShouldBindJSON(&payload); err != nil {
        h.logger.Warn("Invalid JSON payload", zap.Error(err))
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Success:   false,
            Error:     "Invalid JSON payload",
            Code:      "INVALID_JSON",
            Timestamp: time.Now(),
        })
        return
    }
    
    // Set system fields
    payload.UUID = uuid.New()
    payload.Timestamp = time.Now()
    
    // Validate payload
    if err := h.validator.ValidatePayload(&payload); err != nil {
        h.logger.Warn("Payload validation failed", 
            zap.Error(err), 
            zap.String("uuid", payload.UUID.String()))
            
        c.JSON(http.StatusBadRequest, model.ErrorResponse{
            Success:   false,
            Error:     "Validation failed",
            Code:      "VALIDATION_ERROR",
            Details:   []string{err.Error()},
            Timestamp: time.Now(),
        })
        return
    }
    
    // Write file
    path, err := h.fileWriter.Write(c.Request.Context(), &payload)
    if err != nil {
        h.logger.Error("File write failed", 
            zap.Error(err), 
            zap.String("uuid", payload.UUID.String()))
            
        c.JSON(http.StatusInternalServerError, model.ErrorResponse{
            Success:   false,
            Error:     "Failed to save file",
            Code:      "FILE_WRITE_ERROR",
            Details:   []string{err.Error()},
            Timestamp: time.Now(),
        })
        return
    }
    
    h.logger.Info("Webhook processed successfully",
        zap.String("uuid", payload.UUID.String()),
        zap.String("path", path),
        zap.String("title", payload.Title))
    
    c.JSON(http.StatusOK, model.WebhookResponse{
        Success:   true,
        Message:   "File saved successfully",
        UUID:      payload.UUID.String(),
        Timestamp: time.Now(),
        Path:      path,
    })
}
```

**3.2 Health Handler (TDD)** (Parallel with 3.1)

**Test First** (`test/unit/handler/health_test.go`):
```go
package handler_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    
    "github.com/yourusername/hokku/internal/handler"
)

func TestHealth(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()
    
    c, _ := gin.CreateTestContext(w)
    c.Request = req
    
    handler.Health(c)
    
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    
    assert.Equal(t, "healthy", response["status"])
    assert.Contains(t, response, "version")
    assert.Contains(t, response, "uptime")
    assert.Contains(t, response, "memory")
}
```

**Quality Gate 3**:
- [ ] All handler tests pass
- [ ] Error responses follow consistent format
- [ ] Request/response logging is implemented
- [ ] HTTP status codes are correct
- [ ] JSON binding and validation work properly

### Phase 4: Authentication & Configuration (Days 9-11)

**Goal**: Implement authentication middleware and configuration management

**Dependencies**: Phase 3 complete

#### Tasks

**4.1 Configuration Management (TDD)** (Sequential: Tests → Implementation)

**Test First** (`test/unit/config/config_test.go`):
```go
package config_test

import (
    "os"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourusername/hokku/internal/config"
)

func TestConfig_Load(t *testing.T) {
    // Create temporary config file
    configContent := `
app:
  name: hokku
  version: 1.0.0
  env: test

server:
  host: localhost
  port: 8080
  read_timeout: 30s

storage:
  base_dir: /tmp/test
  disk_quota: 1073741824

auth:
  enabled: true
  type: api_key
  api_keys:
    - key: test-key-001
      name: Test Key
      enabled: true

logging:
  level: debug
  format: json
`
    
    tmpfile, err := os.CreateTemp("", "config*.yaml")
    require.NoError(t, err)
    defer os.Remove(tmpfile.Name())
    
    _, err = tmpfile.Write([]byte(configContent))
    require.NoError(t, err)
    tmpfile.Close()
    
    // Set config file path
    os.Setenv("HOKKU_CONFIG", tmpfile.Name())
    defer os.Unsetenv("HOKKU_CONFIG")
    
    cfg, err := config.Load()
    require.NoError(t, err)
    
    assert.Equal(t, "hokku", cfg.App.Name)
    assert.Equal(t, "test", cfg.App.Env)
    assert.Equal(t, "localhost", cfg.Server.Host)
    assert.Equal(t, 8080, cfg.Server.Port)
    assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
    assert.True(t, cfg.Auth.Enabled)
    assert.Equal(t, "api_key", cfg.Auth.Type)
    assert.Len(t, cfg.Auth.APIKeys, 1)
    assert.Equal(t, "test-key-001", cfg.Auth.APIKeys[0].Key)
}

func TestConfig_EnvironmentOverride(t *testing.T) {
    os.Setenv("HOKKU_SERVER_PORT", "9090")
    os.Setenv("HOKKU_LOG_LEVEL", "error")
    defer func() {
        os.Unsetenv("HOKKU_SERVER_PORT")
        os.Unsetenv("HOKKU_LOG_LEVEL")
    }()
    
    cfg, err := config.LoadDefaults()
    require.NoError(t, err)
    
    assert.Equal(t, 9090, cfg.Server.Port)
    assert.Equal(t, "error", cfg.Logging.Level)
}
```

**4.2 Authentication Middleware (TDD)** (Sequential after 4.1)

**Test First** (`test/unit/middleware/auth_test.go`):
```go
package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    
    "github.com/yourusername/hokku/internal/config"
    "github.com/yourusername/hokku/internal/middleware"
)

func TestAuthMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    authConfig := &config.AuthConfig{
        Enabled: true,
        Type:    "api_key",
        APIKeys: []config.APIKey{
            {Key: "valid-key", Name: "Test", Enabled: true},
            {Key: "disabled-key", Name: "Disabled", Enabled: false},
        },
    }
    
    tests := []struct {
        name           string
        headers        map[string]string
        queryParams    map[string]string
        expectedStatus int
    }{
        {
            name: "valid_api_key_header",
            headers: map[string]string{
                "X-API-Key": "valid-key",
            },
            expectedStatus: http.StatusOK,
        },
        {
            name: "valid_api_key_query",
            queryParams: map[string]string{
                "api_key": "valid-key",
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:           "missing_api_key",
            expectedStatus: http.StatusUnauthorized,
        },
        {
            name: "invalid_api_key",
            headers: map[string]string{
                "X-API-Key": "invalid-key",
            },
            expectedStatus: http.StatusUnauthorized,
        },
        {
            name: "disabled_api_key",
            headers: map[string]string{
                "X-API-Key": "disabled-key",
            },
            expectedStatus: http.StatusUnauthorized,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            router := gin.New()
            router.Use(middleware.Auth(authConfig))
            router.GET("/test", func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"status": "ok"})
            })
            
            req := httptest.NewRequest(http.MethodGet, "/test", nil)
            
            // Set headers
            for k, v := range tt.headers {
                req.Header.Set(k, v)
            }
            
            // Set query parameters
            q := req.URL.Query()
            for k, v := range tt.queryParams {
                q.Add(k, v)
            }
            req.URL.RawQuery = q.Encode()
            
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            
            assert.Equal(t, tt.expectedStatus, w.Code)
        })
    }
}
```

**Quality Gate 4**:
- [ ] Configuration loads from YAML and environment variables
- [ ] Authentication middleware properly validates API keys
- [ ] Disabled API keys are rejected
- [ ] Configuration validation is comprehensive
- [ ] Error messages are clear and consistent

### Phase 5: Integration & Production Readiness (Days 12-15)

**Goal**: Complete integration, add monitoring, and prepare for production

**Dependencies**: All previous phases complete

#### Tasks

**5.1 Main Application Integration** (Sequential)
```go
// cmd/hokku/main.go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    
    "github.com/yourusername/hokku/internal/config"
    "github.com/yourusername/hokku/internal/handler"
    "github.com/yourusername/hokku/internal/middleware"
    "github.com/yourusername/hokku/internal/service"
    "github.com/yourusername/hokku/pkg/logger"
)

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        panic(fmt.Sprintf("Failed to load config: %v", err))
    }
    
    // Initialize logger
    log, err := logger.New(&cfg.Logging)
    if err != nil {
        panic(fmt.Sprintf("Failed to initialize logger: %v", err))
    }
    defer log.Sync()
    
    log.Info("Starting Hokku",
        zap.String("version", Version),
        zap.String("build_time", BuildTime),
        zap.String("env", cfg.App.Env))
    
    // Set Gin mode
    if cfg.App.Env == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    // Initialize services
    fileWriter := service.NewFileWriter(&cfg.Storage, log)
    validator := service.NewValidator(&cfg.Validation)
    
    // Initialize handlers
    webhookHandler := handler.NewWebhookHandler(fileWriter, validator, log)
    
    // Setup router
    router := gin.New()
    router.Use(
        middleware.Logger(log),
        middleware.Recovery(log),
        gin.Recovery(),
    )
    
    // Routes
    router.GET("/health", handler.Health)
    
    // Protected routes
    protected := router.Group("/")
    if cfg.Auth.Enabled {
        protected.Use(middleware.Auth(&cfg.Auth))
    }
    protected.POST("/webhook", webhookHandler.HandleWebhook)
    
    // HTTP Server
    srv := &http.Server{
        Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
        Handler:        router,
        ReadTimeout:    cfg.Server.ReadTimeout,
        WriteTimeout:   cfg.Server.WriteTimeout,
        MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
    }
    
    // Start server in goroutine
    go func() {
        log.Info("Server starting", 
            zap.String("addr", srv.Addr),
            zap.Duration("read_timeout", cfg.Server.ReadTimeout),
            zap.Duration("write_timeout", cfg.Server.WriteTimeout))
            
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed to start", zap.Error(err))
        }
    }()
    
    // Wait for interrupt signal to gracefully shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Info("Shutting down server...")
    
    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        log.Error("Server forced to shutdown", zap.Error(err))
    }
    
    log.Info("Server exited")
}
```

**5.2 Integration Tests** (Parallel with 5.1)

**File**: `test/integration/webhook_test.go`
```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourusername/hokku/internal/config"
    "github.com/yourusername/hokku/internal/model"
    // Import your application setup
)

func TestWebhookIntegration(t *testing.T) {
    // Setup test environment
    tempDir := t.TempDir()
    
    cfg := &config.Config{
        App: config.AppConfig{
            Name: "hokku-test",
            Env:  "test",
        },
        Server: config.ServerConfig{
            Host: "localhost",
            Port: 0, // Let system choose port
        },
        Storage: config.StorageConfig{
            BaseDir:         tempDir,
            DiskQuota:       1073741824, // 1GB
            FilePermissions: 0644,
            DirPermissions:  0755,
        },
        Auth: config.AuthConfig{
            Enabled: true,
            Type:    "api_key",
            APIKeys: []config.APIKey{
                {Key: "test-key", Name: "Integration Test", Enabled: true},
            },
        },
    }
    
    // Initialize test server
    server := setupTestServer(cfg)
    defer server.Close()
    
    t.Run("successful_webhook_flow", func(t *testing.T) {
        payload := map[string]interface{}{
            "title":        "Integration Test Document",
            "filename":     "integration-test.md",
            "body":         "# Integration Test\n\nThis is a test document.",
            "path":         "test/docs",
            "content_type": "text/markdown",
            "author":       "Test Author",
            "tags":         []string{"test", "integration"},
        }
        
        body, _ := json.Marshal(payload)
        req, _ := http.NewRequest("POST", server.URL+"/webhook", bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("X-API-Key", "test-key")
        
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        
        assert.Equal(t, http.StatusOK, resp.StatusCode)
        
        var response model.WebhookResponse
        err = json.NewDecoder(resp.Body).Decode(&response)
        require.NoError(t, err)
        
        assert.True(t, response.Success)
        assert.NotEmpty(t, response.UUID)
        assert.NotEmpty(t, response.Path)
        
        // Verify file was created
        expectedPath := filepath.Join(tempDir, "test/docs", "integration-test.md")
        content, err := os.ReadFile(expectedPath)
        require.NoError(t, err)
        assert.Contains(t, string(content), "# Integration Test")
    })
    
    t.Run("authentication_required", func(t *testing.T) {
        payload := map[string]interface{}{
            "title":    "Test",
            "filename": "test.txt",
            "body":     "content",
        }
        
        body, _ := json.Marshal(payload)
        req, _ := http.NewRequest("POST", server.URL+"/webhook", bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        // No API key header
        
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        
        assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    })
}

func setupTestServer(cfg *config.Config) *httptest.Server {
    // Initialize your application with the test config
    app := setupApplication(cfg)
    return httptest.NewServer(app)
}
```

**5.3 Production Configuration** (Parallel with 5.2)

**File**: `config/production.yaml`
```yaml
app:
  name: hokku
  version: 1.0.0
  env: production

server:
  host: 0.0.0.0
  port: 20023
  read_timeout: 30s
  write_timeout: 30s
  max_header_bytes: 1048576
  graceful_timeout: 30s

storage:
  base_dir: /var/lib/hokku/storage
  disk_quota: 10737418240  # 10GB
  create_dirs: true
  file_permissions: 0644
  dir_permissions: 0755

auth:
  enabled: true
  type: api_key
  api_keys: []  # Set via environment

validation:
  max_payload_size: 10485760  # 10MB
  allowed_extensions:
    - .txt
    - .md
    - .json
    - .html
    - .xml
  forbidden_patterns:
    - "../"
    - ".."
    - "~"

logging:
  level: info
  format: json
  output: file
  file:
    path: /var/log/hokku/hokku.log
    max_size: 100
    max_backups: 10
    max_age: 30
```

**5.4 Dockerfile and Docker Compose** (Parallel with 5.3)

**File**: `Dockerfile`
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.Version=$(git describe --tags --always --dirty) \
              -X main.BuildTime=$(date +%FT%T%z)" \
    -o hokku cmd/hokku/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

RUN addgroup -g 1001 hokku && \
    adduser -D -s /bin/sh -u 1001 -G hokku hokku

COPY --from=builder /app/hokku .
COPY --from=builder /app/config/ ./config/

RUN mkdir -p /var/lib/hokku/storage /var/log/hokku && \
    chown -R hokku:hokku /var/lib/hokku /var/log/hokku

USER hokku

EXPOSE 20023

CMD ["./hokku"]
```

**File**: `docker-compose.yml`
```yaml
version: '3.8'

services:
  hokku:
    build: .
    ports:
      - "20023:20023"
    environment:
      - HOKKU_ENV=production
      - HOKKU_API_KEY=${HOKKU_API_KEY:-your-secure-api-key-here}
      - HOKKU_LOG_LEVEL=${HOKKU_LOG_LEVEL:-info}
    volumes:
      - hokku_storage:/var/lib/hokku/storage
      - hokku_logs:/var/log/hokku
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:20023/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  hokku_storage:
    driver: local
  hokku_logs:
    driver: local
```

**Quality Gate 5**:
- [ ] Integration tests pass end-to-end
- [ ] Application starts and stops gracefully
- [ ] Docker container builds successfully
- [ ] Health checks work properly
- [ ] Production configuration is secure
- [ ] Logging works in production mode
- [ ] All error scenarios are handled

---

## Quality Gates

### Gate 1: Foundation (After Phase 1)
**Acceptance Criteria:**
- [ ] All Go files compile without errors
- [ ] `go vet ./...` passes with zero issues
- [ ] `golangci-lint run` passes with zero issues
- [ ] All interfaces are properly defined
- [ ] Model structs have appropriate validation tags
- [ ] Directory structure matches specification

**Commands:**
```bash
go build ./...
go vet ./...
golangci-lint run
```

### Gate 2: Core Services (After Phase 2)
**Acceptance Criteria:**
- [ ] All unit tests pass: `go test ./internal/service/...`
- [ ] Test coverage >= 80%: `go test -coverprofile=coverage.out ./internal/service/... && go tool cover -func=coverage.out`
- [ ] Security tests pass (path traversal prevention)
- [ ] File operations are atomic
- [ ] Memory usage is reasonable under test load

**Commands:**
```bash
go test -v -race ./internal/service/...
go test -coverprofile=coverage.out ./internal/service/...
go tool cover -html=coverage.out -o coverage.html
gosec ./internal/service/...
```

### Gate 3: HTTP Layer (After Phase 3)
**Acceptance Criteria:**
- [ ] All handler tests pass
- [ ] HTTP status codes follow REST conventions
- [ ] Error responses are consistent and informative
- [ ] Request/response logging is implemented
- [ ] JSON marshaling/unmarshaling works correctly

**Commands:**
```bash
go test -v ./internal/handler/...
curl -X POST http://localhost:20023/health  # Manual testing
```

### Gate 4: Security & Config (After Phase 4)
**Acceptance Criteria:**
- [ ] Configuration loads correctly from file and environment
- [ ] API key authentication works
- [ ] Disabled API keys are properly rejected
- [ ] Configuration validation prevents invalid settings
- [ ] Security middleware prevents unauthorized access

**Commands:**
```bash
go test -v ./internal/config/... ./internal/middleware/...
gosec ./internal/middleware/...
```

### Gate 5: Production Ready (After Phase 5)
**Acceptance Criteria:**
- [ ] Integration tests pass
- [ ] Application starts and stops gracefully
- [ ] Docker builds successfully
- [ ] Health endpoint returns proper status
- [ ] Logging works in all environments
- [ ] Performance meets requirements (>100 RPS)

**Commands:**
```bash
go test -v ./test/integration/...
make build && ./bin/hokku --version
docker build -t hokku:test .
docker run --rm hokku:test --help
```

---

## Parallel Execution Plan

### Phase 1 Parallelization
```
Start simultaneously:
├── Task 1.1: Project structure setup (15 minutes)
├── Task 1.2: Interface definitions (30 minutes)  
├── Task 1.3: Data models (45 minutes)
└── Task 1.4: Response models (30 minutes)

Total time: 45 minutes (vs 120 minutes sequential)
```

### Phase 2 Parallelization
```
After Phase 1 complete:
├── Task 2.1: File Writer (TDD) (2 days)
├── Task 2.2: Validator (TDD) (1.5 days)
└── After both complete:
    └── Task 2.3: Security utils (0.5 days)

Total time: 2.5 days (vs 4 days sequential)
```

### Phase 3 Parallelization
```
After Phase 2 complete:
├── Task 3.1: Webhook handler (TDD) (1.5 days)
└── Task 3.2: Health handler (TDD) (0.5 days)

Total time: 1.5 days (vs 2 days sequential)
```

### Phase 4 Parallelization
```
After Phase 3 complete:
├── Task 4.1: Configuration (TDD) (1 day)
└── Task 4.2: Auth middleware (TDD) (1 day)

Total time: 1 day (vs 2 days sequential)  
Note: 4.2 needs some types from 4.1, but can start tests in parallel
```

### Phase 5 Parallelization
```
After Phase 4 complete:
├── Task 5.1: Main app integration (1 day)
├── Task 5.2: Integration tests (1.5 days)
├── Task 5.3: Production config (0.5 days)
└── Task 5.4: Docker setup (0.5 days)

Total time: 1.5 days (vs 3.5 days sequential)
```

**Overall Timeline:**
- Sequential execution: ~15 days
- Parallel execution: ~8.5 days  
- **Time savings: 43%**

---

## Cross-Session Management

### Session State Files

**File**: `.hokku-session/current-phase.json`
```json
{
  "current_phase": 2,
  "current_task": "2.2",
  "completed_tasks": [
    "1.1", "1.2", "1.3", "1.4",
    "2.1"
  ],
  "blocked_tasks": [],
  "last_updated": "2025-09-07T10:30:00Z",
  "next_actions": [
    "Complete validator service tests",
    "Implement validator service", 
    "Run quality gate 2 checks"
  ]
}
```

**File**: `.hokku-session/quality-gates.json`
```json
{
  "gate_1": {
    "status": "passed",
    "completed_at": "2025-09-07T09:00:00Z",
    "checks": {
      "compilation": "passed",
      "go_vet": "passed", 
      "linting": "passed",
      "interfaces": "passed"
    }
  },
  "gate_2": {
    "status": "in_progress",
    "checks": {
      "unit_tests": "passed",
      "coverage": "in_progress",
      "security_tests": "pending"
    }
  }
}
```

### Session Commands

**Start New Session:**
```bash
# Load previous state
cat .hokku-session/current-phase.json
cat .hokku-session/quality-gates.json

# Continue from last checkpoint
make continue-phase-2
```

**Save Session State:**
```bash
# Update session state
./scripts/save-session.sh

# Commit progress
git add .
git commit -m "Phase 2 progress: file writer service complete"
```

**Resume After Interruption:**
```bash
# Check what was completed
make status

# Run any missed quality checks
make verify-gate-2

# Continue with next task
make continue
```

### Progress Tracking Commands

```bash
# Check overall progress
make progress

# List remaining tasks for current phase
make tasks

# Verify current phase completion
make verify-current

# Check quality gate status
make gates
```

---

## SOLID Compliance Checklist

### Single Responsibility Principle (SRP)
- [ ] `FileWriter` only handles file operations
- [ ] `Validator` only handles validation logic
- [ ] `WebhookHandler` only handles HTTP requests
- [ ] `AuthMiddleware` only handles authentication
- [ ] Each service has one reason to change

### Open-Closed Principle (OCP)
- [ ] New validation rules can be added without modifying existing code
- [ ] New authentication methods can be added via strategy pattern
- [ ] File storage backends can be swapped via interface
- [ ] New content encodings can be added without core changes

### Liskov Substitution Principle (LSP)
- [ ] Any `FileWriter` implementation works with handlers
- [ ] Any `Validator` implementation works with handlers  
- [ ] Any `Logger` implementation works with services
- [ ] Mock implementations work in tests

### Interface Segregation Principle (ISP)
- [ ] `FileWriter` interface only exposes file-related methods
- [ ] `Validator` interface only exposes validation methods
- [ ] No client depends on methods it doesn't use
- [ ] Interfaces are focused and cohesive

### Dependency Inversion Principle (DIP)
- [ ] Handlers depend on service interfaces, not implementations
- [ ] Services depend on utility interfaces, not implementations
- [ ] Main function wires up dependencies at startup
- [ ] No direct dependencies on concrete types in business logic

**Verification Command:**
```bash
# Check for SOLID violations
./scripts/check-solid-compliance.sh

# Review dependency graph
go mod graph | grep hokku
```

---

## Commands Reference

### Development Commands
```bash
# Start development
make dev

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linting
make lint

# Run security checks
make security

# Build application
make build

# Run application
make run

# Clean build artifacts
make clean
```

### Quality Assurance
```bash
# Full quality check
make qa

# Verify SOLID compliance
make verify-solid

# Check test coverage
make coverage-report

# Run integration tests
make test-integration

# Performance benchmarks
make bench
```

### Docker Commands
```bash
# Build Docker image
make docker-build

# Run in Docker
make docker-run

# Docker development environment
make docker-dev

# Stop Docker containers
make docker-stop
```

### Phase-Specific Commands
```bash
# Phase 1: Foundation
make phase-1

# Phase 2: Services
make phase-2

# Phase 3: Handlers
make phase-3

# Phase 4: Auth & Config
make phase-4

# Phase 5: Production
make phase-5
```

---

## Session Templates

### Daily Standup Template
```markdown
## Hokku Development Status - [Date]

### Yesterday's Accomplishments
- [ ] Task X.Y completed
- [ ] Quality Gate N passed
- [ ] Issue #123 resolved

### Today's Goals  
- [ ] Start Task X.Y
- [ ] Complete Quality Gate N
- [ ] Address technical debt item

### Blockers
- None / [Description of blocker]

### Next Session Focus
- Priority task for next development session
- Any research or preparation needed
```

### Phase Completion Template
```markdown
## Phase [N] Completion Report

### Phase Goals ✅
- [x] Goal 1 achieved
- [x] Goal 2 achieved  
- [x] Goal 3 achieved

### Quality Gate Results ✅
- [x] All tests passing (Coverage: X%)
- [x] Linting clean
- [x] Security checks passed
- [x] Performance meets targets

### Technical Debt Created
- Item 1: [Description and plan to address]
- Item 2: [Description and plan to address]

### Lessons Learned
- What worked well
- What could be improved
- Process adjustments for next phase

### Next Phase Preparation
- Dependencies verified
- Environment ready  
- Team/resources aligned
```

### Error Recovery Template
```markdown  
## Error Recovery Session - [Date]

### Issue Description
- What went wrong
- When it was discovered
- Impact assessment

### Root Cause Analysis
- Primary cause
- Contributing factors
- Prevention measures

### Recovery Plan
- [ ] Immediate fixes
- [ ] Rollback if necessary
- [ ] Re-run quality gates
- [ ] Update documentation

### Prevention
- Process improvements
- Additional checks to add
- Monitoring enhancements
```

---

This comprehensive workflow ensures SOLID compliance, maintains high quality through TDD and quality gates, maximizes parallel execution, and supports cross-session development. The workflow is designed to be resumable, trackable, and scalable for team development.

Total estimated time with parallel execution: **8.5 days** for a production-ready webhook service.
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Hokku is a Go-based webhook receiver that saves JSON payloads as files. It's designed for CI/CD integration, notifications, and log collection with built-in security features.

- **Default Port**: 20023
- **Go Version**: 1.21+
- **Main Dependencies**: gin-gonic/gin, zap, viper, validator/v10

## Essential Commands

### Development Workflow
```bash
# Initialize project (first time setup)
go mod download

# Run application
make run

# Build binary
make build

# Run with hot reload (development)
make dev

# Run tests
make test

# Run specific test
go test -v -run TestFileWriter ./internal/service/...

# View test coverage
make test-coverage

# Lint code
make lint

# Clean build artifacts
make clean
```

### Quality Checks Before Committing
```bash
go fmt ./...
go vet ./...
make lint
make test
```

## Architecture Overview

**For detailed architecture and design documentation, see `/docs/` directory:**

- `docs/SOLID_YAGNI_COMPLIANT_DESIGN.md` - Final SOLID/YAGNI compliant design specification
- `docs/PRACTICAL_DESIGN.md` - Simplified practical design after user feedback  
- `docs/IMPROVED_DESIGN.md` - Initial Clean Architecture design (deprecated)
- `SPECIFICATION.md` - Complete project specification in Japanese

### Quick Architecture Summary

The application follows SOLID principles with interface-based dependency injection:

1. **Request Flow**: 
   - Client → Gin Router → Auth Middleware → Validator → FileStore → File System
   - All operations logged via structured logging (zap)

2. **SOLID-Compliant Package Organization**:
   - `cmd/hokku/`: Application entry point  
   - `internal/app/`: Application composition with DI
   - `internal/handler/`: HTTP handlers implementing single responsibilities
   - `internal/service/`: Core business logic with interface segregation
   - `internal/middleware/`: Cross-cutting concerns
   - `internal/model/`: Data models
   - `pkg/errors/`: Custom error types with Google Go style

3. **Key Design Principles Applied**:
   - **SRP**: Each component has single responsibility
   - **DIP**: Depend on interfaces, not concrete types
   - **ISP**: Small, focused interfaces (FileStore, PayloadValidator)
   - **YAGNI**: Only implement current requirements, no speculative features
   - **Google Go Style**: Sentinel errors, proper error wrapping

## Testing Strategy

The project uses table-driven tests with testify framework:

- Unit tests in `test/unit/`
- Integration tests in `test/integration/`
- Test data in `test/testdata/`
- Target coverage: 80%+

## API Endpoints

- `POST /webhook` - Main webhook receiver (requires auth)
- `GET /health` - Health check endpoint
- `GET /metrics` - Metrics endpoint (future implementation)

## Error Handling Pattern

Early return pattern with wrapped errors for context:
```go
if err != nil {
    return "", fmt.Errorf("operation failed: %w", err)
}
```

## Key Implementation Files

When implementing features, focus on these core files:
- `internal/handler/webhook.go` - Main webhook handler logic
- `internal/service/file_writer.go` - File system operations
- `internal/util/security.go` - Path validation and security checks
- `internal/middleware/auth.go` - Authentication logic
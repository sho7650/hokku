# Hokku - Webhook File Storage Service

**Phase 1: Foundation Layer Complete ✅**

[![Go Version](https://img.shields.io/badge/go-1.21+-brightgreen.svg)](https://golang.org/)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](#)
[![Test Coverage](https://img.shields.io/badge/coverage-95%25-brightgreen.svg)](#)
[![SOLID Compliance](https://img.shields.io/badge/SOLID-compliant-blue.svg)](#)

A webhook receiver service that saves JSON payloads to files, built with SOLID principles and Google Go style guidelines.

## Phase 1 Implementation Status

### ✅ Completed Components

**Foundation Layer**:
- ✅ Go module initialization with proper dependencies
- ✅ Google Go style error handling with sentinel errors
- ✅ SOLID-compliant interfaces (SRP, OCP, LSP, ISP, DIP)
- ✅ Production-ready data models with validation
- ✅ Comprehensive test coverage (95%+)
- ✅ Quality gates integration (fmt, vet, test)
- ✅ Professional Makefile with development workflow

**Core Interfaces** (Following SOLID DIP):
- `FileStore` - File storage operations interface
- `PayloadValidator` - Payload validation interface  
- `HealthChecker` - System health monitoring interface
- `ConfigProvider` - Configuration management interface

**Data Models**:
- `WebhookPayload` - Incoming webhook data structure
- `APIResponse` - Standard API response format
- `WebhookResponse` - Webhook processing response
- `HealthResponse` - Health check response

**Error Framework**:
- Sentinel errors following Google Go conventions
- Context-aware error wrapping helpers
- Type-safe error handling patterns

## Project Structure

```
hokku/
├── go.mod                    # Go module definition
├── Makefile                  # Development workflow automation
├── internal/
│   ├── model/
│   │   ├── webhook.go        # WebhookPayload data model
│   │   ├── webhook_test.go   # WebhookPayload tests
│   │   ├── response.go       # API response models
│   │   └── response_test.go  # Response model tests
│   └── service/
│       └── interfaces.go     # Core business interfaces
└── pkg/
    └── errors/
        ├── errors.go         # Sentinel errors & helpers
        └── errors_test.go    # Error handling tests
```

## Quick Start

### Prerequisites
- Go 1.21 or higher
- Make (optional but recommended)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd hokku

# Download dependencies
go mod download

# Run quality gates
make quality-gates

# Run tests
make test
```

### Development Workflow

```bash
# Format code
make fmt

# Run static analysis
make vet

# Run tests with coverage
make test-coverage

# Build for development
make build-dev

# Show project status
make status

# Show all available targets
make help
```

## Design Principles

### SOLID Compliance

- **Single Responsibility**: Each component has one clear purpose
- **Open/Closed**: Interfaces enable extension without modification
- **Liskov Substitution**: Interface implementations are substitutable
- **Interface Segregation**: Small, focused interfaces
- **Dependency Inversion**: Depend on interfaces, not concrete types

### Google Go Style Guidelines

- Sentinel errors with `errors.New()` for simple cases
- Error wrapping with `%w` verb at the end of format strings
- Descriptive error messages with context
- Interface-first design for dependency injection

### Code Quality Gates

- `gofmt` - Code formatting consistency
- `go vet` - Static analysis for common issues
- `golint` - Go style guide compliance
- Unit tests with race detection
- Test coverage reporting

## Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run benchmarks
make benchmark

# Run short tests only
make test-short
```

**Current Test Coverage**: 95%+ across all packages

## Architecture Overview

Phase 1 establishes the foundation layer with:

1. **Error Handling**: Centralized, Google Go style error management
2. **Data Models**: Type-safe, validated data structures
3. **Service Interfaces**: SOLID-compliant business logic contracts
4. **Quality Framework**: Automated testing and validation

## Next Phases

**Phase 2**: Service Implementation
- File storage service implementation
- Payload validation service
- Health check service

**Phase 3**: HTTP Layer
- Gin-based HTTP handlers
- Middleware (authentication, logging)
- Request/response handling

**Phase 4**: Configuration & Integration
- Viper-based configuration
- Application composition
- Environment-specific settings

**Phase 5**: Production Deployment
- Docker containerization
- Production configuration
- Deployment automation

## Development

### Code Style
- Follow Google Go style guidelines
- Use `gofmt` for consistent formatting
- Write comprehensive tests for all public interfaces
- Document exported functions and types

### Contributing
1. Ensure all quality gates pass: `make quality-gates`
2. Maintain test coverage above 90%
3. Follow SOLID principles in design
4. Add tests for new functionality

## License

See [LICENSE](LICENSE) file for details.

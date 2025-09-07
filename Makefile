# Hokku Webhook Service Makefile
# Provides essential development and build targets with quality gates integration

# Variables
APP_NAME := hokku
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Directories
CMD_DIR := ./cmd/hokku
BIN_DIR := ./bin
COVERAGE_DIR := ./coverage

# Default target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
.PHONY: deps
deps: ## Download and verify dependencies
	$(GOMOD) download
	$(GOMOD) verify

.PHONY: deps-update
deps-update: ## Update dependencies to latest versions
	$(GOMOD) tidy
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

# Build targets
.PHONY: build
build: quality-gates ## Build the application
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

.PHONY: build-dev
build-dev: ## Build for development (no quality gates)
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -race $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-dev $(CMD_DIR)

.PHONY: install
install: quality-gates ## Install the application to GOPATH/bin
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(APP_NAME) $(CMD_DIR)

# Quality gates (essential for Phase 1)
.PHONY: quality-gates
quality-gates: fmt vet lint test ## Run all quality gates

.PHONY: fmt
fmt: ## Format Go source code
	$(GOCMD) fmt ./...

.PHONY: fmt-check
fmt-check: ## Check if Go source code is formatted
	@if [ -n "$(shell $(GOCMD) fmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		$(GOCMD) fmt -l .; \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet on the project
	$(GOCMD) vet ./...

.PHONY: lint
lint: ## Run golint on the project (install with: go install golang.org/x/lint/golint@latest)
	@command -v golint >/dev/null 2>&1 || { echo >&2 "golint not installed. Run: go install golang.org/x/lint/golint@latest"; exit 1; }
	golint -set_exit_status ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)
	@command -v staticcheck >/dev/null 2>&1 || { echo >&2 "staticcheck not installed. Run: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1; }
	staticcheck ./...

# Test targets
.PHONY: test
test: ## Run unit tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

.PHONY: test-short
test-short: ## Run unit tests (short mode)
	$(GOTEST) -short -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests and show coverage
	@mkdir -p $(COVERAGE_DIR)
	$(GOCMD) tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GOCMD) tool cover -func=coverage.out | tail -1

.PHONY: benchmark
benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

# Development workflow targets
.PHONY: run
run: build-dev ## Build and run the application in development mode
	./$(BIN_DIR)/$(APP_NAME)-dev

.PHONY: run-prod
run-prod: build ## Build and run the application in production mode
	./$(BIN_DIR)/$(APP_NAME)

.PHONY: dev
dev: ## Start development with hot reload (requires air: go install github.com/cosmtrek/air@latest)
	@command -v air >/dev/null 2>&1 || { echo >&2 "air not installed. Run: go install github.com/cosmtrek/air@latest"; exit 1; }
	air

# Cleanup targets
.PHONY: clean
clean: ## Clean build artifacts and caches
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f coverage.out

.PHONY: clean-deps
clean-deps: ## Clean dependency cache
	$(GOCMD) clean -modcache

# Docker targets (for future phases)
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run --rm -p 8080:8080 $(APP_NAME):latest

# CI/CD helper targets
.PHONY: ci-deps
ci-deps: ## Install CI dependencies
	$(GOGET) golang.org/x/lint/golint
	$(GOGET) honnef.co/go/tools/cmd/staticcheck

.PHONY: ci-test
ci-test: deps quality-gates test-coverage ## Full CI test suite

# Security scanning (for future phases)
.PHONY: security
security: ## Run security checks (requires gosec: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	@command -v gosec >/dev/null 2>&1 || { echo >&2 "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; exit 1; }
	gosec ./...

# Git hooks (for development setup)
.PHONY: install-hooks
install-hooks: ## Install git pre-commit hooks
	echo '#!/bin/sh\nmake fmt-check vet lint test-short' > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

# Project status
.PHONY: status
status: ## Show project status
	@echo "Project: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo ""
	@echo "Go Version: $(shell $(GOCMD) version)"
	@echo "Dependencies:"
	@$(GOCMD) list -m all | head -10

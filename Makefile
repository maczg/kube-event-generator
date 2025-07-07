# Kubernetes Event Generator Makefile

# Variables
BINARY_NAME := keg
BINARY_PATH := bin/$(BINARY_NAME)
MAIN_PATH := cmd/keg/main.go
MODULE := github.com/maczg/kube-event-generator

# Go variables
GO := go
GOFLAGS := -v
LDFLAGS := -s -w
TEST_FLAGS := -race -cover
INTEGRATION_TEST_FLAGS := -tags=integration

# Docker variables
DOCKER_IMAGE := kube-event-generator
DOCKER_TAG := latest

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: all build clean test test-unit test-integration test-coverage lint fmt vet deps docker-build docker-push help

# Default target
all: clean deps lint test build

# Build the binary
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)Build complete: $(BINARY_PATH)$(NC)"

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf bin/
	@rm -rf results/
	@rm -f coverage.out coverage.html
	@$(GO) clean -testcache
	@echo "$(GREEN)Clean complete$(NC)"

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	$(GO) test $(TEST_FLAGS) ./pkg/...

# Run integration tests
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	$(GO) test $(TEST_FLAGS) $(INTEGRATION_TEST_FLAGS) ./test/integration/...

# Generate test coverage report
test-coverage:
	@echo "$(GREEN)Generating coverage report...$(NC)"
	@$(GO) test -coverprofile=coverage.out ./pkg/...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GO) fmt ./...

# Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GO) vet ./...

# Download dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy

# Verify dependencies
verify:
	@echo "$(GREEN)Verifying dependencies...$(NC)"
	$(GO) mod verify

# Install the binary
install: build
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)Installed to $(GOPATH)/bin/$(BINARY_NAME)$(NC)"

# Uninstall the binary
uninstall:
	@echo "$(YELLOW)Uninstalling $(BINARY_NAME)...$(NC)"
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)Uninstalled$(NC)"

# Run the application
run: build
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BINARY_PATH)

# Docker build
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Docker push
docker-push: docker-build
	@echo "$(GREEN)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Start local environment
local-env:
	@echo "$(GREEN)Starting local environment...$(NC)"
	docker-compose -f docker/docker-compose.yaml up -d
	@echo "$(GREEN)Local environment started$(NC)"

# Stop local environment
local-env-stop:
	@echo "$(YELLOW)Stopping local environment...$(NC)"
	docker-compose -f docker/docker-compose.yaml down
	@echo "$(GREEN)Local environment stopped$(NC)"

# Generate mocks
generate-mocks:
	@echo "$(GREEN)Generating mocks...$(NC)"
	@if command -v mockgen >/dev/null 2>&1; then \
		go generate ./...; \
	else \
		echo "$(YELLOW)mockgen not installed. Install with: go install github.com/golang/mock/mockgen@latest$(NC)"; \
	fi

# Run static analysis
static-analysis: vet lint
	@echo "$(GREEN)Running static analysis...$(NC)"
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "$(YELLOW)staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest$(NC)"; \
	fi

# Check for security vulnerabilities
security-check:
	@echo "$(GREEN)Checking for security vulnerabilities...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./...; \
	else \
		echo "$(YELLOW)gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)"; \
	fi

# Run all checks
check: fmt vet lint test security-check

# Create a release
release: check
	@echo "$(GREEN)Creating release...$(NC)"
	@read -p "Enter version (e.g., v1.0.0): " VERSION; \
	git tag -a $$VERSION -m "Release $$VERSION"; \
	git push origin $$VERSION; \
	echo "$(GREEN)Release $$VERSION created$(NC)"

# Show help
help:
	@echo "$(GREEN)Kubernetes Event Generator - Available targets:$(NC)"
	@echo "  $(YELLOW)all$(NC)              - Clean, download deps, lint, test, and build"
	@echo "  $(YELLOW)build$(NC)            - Build the binary"
	@echo "  $(YELLOW)clean$(NC)            - Clean build artifacts"
	@echo "  $(YELLOW)test$(NC)             - Run all tests"
	@echo "  $(YELLOW)test-unit$(NC)        - Run unit tests"
	@echo "  $(YELLOW)test-integration$(NC) - Run integration tests"
	@echo "  $(YELLOW)test-coverage$(NC)    - Generate test coverage report"
	@echo "  $(YELLOW)lint$(NC)             - Run linter"
	@echo "  $(YELLOW)fmt$(NC)              - Format code"
	@echo "  $(YELLOW)vet$(NC)              - Run go vet"
	@echo "  $(YELLOW)deps$(NC)             - Download dependencies"
	@echo "  $(YELLOW)verify$(NC)           - Verify dependencies"
	@echo "  $(YELLOW)install$(NC)          - Install the binary"
	@echo "  $(YELLOW)uninstall$(NC)        - Uninstall the binary"
	@echo "  $(YELLOW)run$(NC)              - Run the application"
	@echo "  $(YELLOW)docker-build$(NC)     - Build Docker image"
	@echo "  $(YELLOW)docker-push$(NC)      - Push Docker image"
	@echo "  $(YELLOW)local-env$(NC)        - Start local environment"
	@echo "  $(YELLOW)local-env-stop$(NC)   - Stop local environment"
	@echo "  $(YELLOW)generate-mocks$(NC)   - Generate mocks"
	@echo "  $(YELLOW)static-analysis$(NC)  - Run static analysis"
	@echo "  $(YELLOW)security-check$(NC)   - Check for security vulnerabilities"
	@echo "  $(YELLOW)check$(NC)            - Run all checks"
	@echo "  $(YELLOW)release$(NC)          - Create a release"
	@echo "  $(YELLOW)help$(NC)             - Show this help message"

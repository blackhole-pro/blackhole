# Blackhole Makefile

# Variables
BINARY_NAME=blackhole
BINARY_DIR=bin
BUILD_DIR=build
GO=go
GOFLAGS=-v

# Service directories
SERVICES=identity storage ledger social analytics telemetry indexer wallet

# Build the main binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/blackhole

# Build all services
.PHONY: build-services
build-services: $(SERVICES)

# Build individual services
.PHONY: $(SERVICES)
$(SERVICES):
	@echo "Building $@ service..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$@ ./internal/services/$@

# Run tests (excluding integration tests)
.PHONY: test
test:
	@echo "Running tests (excluding integration)..."
	$(GO) test -v -short ./...

# Run only integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	$(GO) test -v ./test/integration/...

# Run all tests including integration
.PHONY: test-all
test-all:
	@echo "Running all tests..."
	$(GO) test -v ./...

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GO) test -race -v -short ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out -short ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Lint the code
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR) $(BUILD_DIR) coverage.out coverage.html

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# Generate protobuf files
.PHONY: proto
proto:
	@echo "Generating protobuf files..."
	@find . -name "*.proto" -exec protoc --go_out=. --go-grpc_out=. {} \;

# Development mode with hot reload
.PHONY: dev
dev:
	@echo "Starting development mode..."
	$(HOME)/go/bin/air

# Cross-compilation
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/blackhole
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/blackhole
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/blackhole
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/blackhole

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t blackhole:latest .

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the main binary"
	@echo "  build-services - Build all service binaries"
	@echo "  test           - Run tests (excluding integration)"
	@echo "  test-integration - Run only integration tests"
	@echo "  test-all       - Run all tests including integration"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  update-deps    - Update dependencies"
	@echo "  proto          - Generate protobuf files"
	@echo "  dev            - Start development mode with hot reload"
	@echo "  build-all      - Cross-compile for all platforms"
	@echo "  docker-build   - Build Docker image"
	@echo "  help           - Show this help message"

# Default target
.DEFAULT_GOAL := build
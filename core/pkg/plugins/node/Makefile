# Node Plugin Makefile
PLUGIN_NAME := node-plugin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Platforms to build for
PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64

# Go build flags
LDFLAGS := -ldflags "\
	-X main.Version=$(VERSION) \
	-X main.BuildDate=$(BUILD_DATE) \
	-X main.GitCommit=$(GIT_COMMIT)"

# Directories
BIN_DIR := bin
DIST_DIR := dist
PROTO_DIR := proto

.PHONY: all
all: proto build

.PHONY: proto
proto:
	@echo "==> Generating protobuf code..."
	@protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/v1/*.proto

.PHONY: build
build: $(PLATFORMS)

.PHONY: $(PLATFORMS)
$(PLATFORMS): proto
	@echo "==> Building $(PLUGIN_NAME) for $@..."
	@mkdir -p $(BIN_DIR)/$@
	@GOOS=$(word 1,$(subst -, ,$@)) GOARCH=$(word 2,$(subst -, ,$@)) \
		CGO_ENABLED=0 go build $(LDFLAGS) \
		-o $(BIN_DIR)/$@/$(PLUGIN_NAME) .

.PHONY: build-local
build-local: proto
	@echo "==> Building $(PLUGIN_NAME) for local platform..."
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(PLUGIN_NAME) .

.PHONY: test
test:
	@echo "==> Running tests..."
	@go test -v -race -cover ./...

.PHONY: lint
lint:
	@echo "==> Running linter..."
	@golangci-lint run ./...

.PHONY: clean
clean:
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(DIST_DIR)

.PHONY: package
package: build docs
	@echo "==> Creating plugin package..."
	@mkdir -p $(DIST_DIR)
	
	# Create temporary packaging directory
	@mkdir -p $(DIST_DIR)/tmp
	
	# Copy required files
	@cp plugin.yaml $(DIST_DIR)/tmp/
	@cp -r $(BIN_DIR) $(DIST_DIR)/tmp/
	@cp -r $(PROTO_DIR) $(DIST_DIR)/tmp/
	@cp -r docs $(DIST_DIR)/tmp/
	@cp LICENSE $(DIST_DIR)/tmp/ 2>/dev/null || echo "No LICENSE file"
	@cp CHANGELOG.md $(DIST_DIR)/tmp/ 2>/dev/null || echo "No CHANGELOG.md"
	
	# Copy examples if they exist
	@if [ -d examples ]; then cp -r examples $(DIST_DIR)/tmp/; fi
	
	# Create the plugin package
	@cd $(DIST_DIR)/tmp && \
		tar -czf ../$(PLUGIN_NAME)-$(VERSION).plugin * && \
		cd ../.. && \
		rm -rf $(DIST_DIR)/tmp
	
	@echo "==> Package created: $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin"
	@echo "==> Package size: $$(du -h $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin | cut -f1)"

.PHONY: docs
docs:
	@echo "==> Ensuring documentation exists..."
	@mkdir -p docs
	@if [ ! -f docs/README.md ]; then \
		echo "# Node Plugin\n\nP2P networking and node management plugin for Blackhole." > docs/README.md; \
	fi
	@if [ ! -f docs/API.md ]; then \
		echo "# Node Plugin API\n\nSee proto/v1/node.proto for the complete API definition." > docs/API.md; \
	fi

.PHONY: install-local
install-local: package
	@echo "==> Installing plugin locally..."
	@mkdir -p ~/.blackhole/plugins/node/$(VERSION)
	@tar -xzf $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin \
		-C ~/.blackhole/plugins/node/$(VERSION)
	@ln -sfn $(VERSION) ~/.blackhole/plugins/node/current
	@echo "==> Plugin installed to ~/.blackhole/plugins/node/$(VERSION)"

.PHONY: verify
verify: package
	@echo "==> Verifying plugin package..."
	@mkdir -p $(DIST_DIR)/verify
	@tar -tzf $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin > $(DIST_DIR)/verify/contents.txt
	@echo "==> Package contents:"
	@cat $(DIST_DIR)/verify/contents.txt
	@rm -rf $(DIST_DIR)/verify
	
	# Verify required files
	@tar -tzf $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin | grep -q "^plugin.yaml$$" || \
		(echo "ERROR: plugin.yaml not found in package" && exit 1)
	@tar -tzf $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin | grep -q "^bin/" || \
		(echo "ERROR: bin/ directory not found in package" && exit 1)
	@tar -tzf $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin | grep -q "^proto/" || \
		(echo "ERROR: proto/ directory not found in package" && exit 1)
	@echo "==> Package verification passed ✓"

.PHONY: run
run: build-local
	@echo "==> Running $(PLUGIN_NAME) locally..."
	@$(BIN_DIR)/$(PLUGIN_NAME)

.PHONY: dev
dev: proto
	@echo "==> Running $(PLUGIN_NAME) in development mode..."
	@go run $(LDFLAGS) .

.PHONY: deps
deps:
	@echo "==> Installing dependencies..."
	@go mod download
	@go mod tidy

.PHONY: update-deps
update-deps:
	@echo "==> Updating dependencies..."
	@go get -u ./...
	@go mod tidy

.PHONY: release
release: clean test lint package verify
	@echo "==> Release build complete!"
	@echo "==> Plugin package: $(DIST_DIR)/$(PLUGIN_NAME)-$(VERSION).plugin"

# Platform-specific development builds
.PHONY: darwin
darwin: darwin-amd64 darwin-arm64

.PHONY: linux  
linux: linux-amd64 linux-arm64

# Help target
.PHONY: help
help:
	@echo "Node Plugin Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          Build everything (proto + all platforms)"
	@echo "  proto        Generate protobuf code"
	@echo "  build        Build for all platforms"
	@echo "  build-local  Build for current platform only"
	@echo "  test         Run tests"
	@echo "  lint         Run linter"
	@echo "  clean        Clean build artifacts"
	@echo "  package      Create plugin package"
	@echo "  verify       Verify plugin package"
	@echo "  install-local Install plugin locally"
	@echo "  run          Build and run locally"
	@echo "  dev          Run in development mode"
	@echo "  release      Full release build (clean, test, lint, package, verify)"
	@echo ""
	@echo "Platform targets:"
	@echo "  darwin-amd64 Build for macOS Intel"
	@echo "  darwin-arm64 Build for macOS Apple Silicon"
	@echo "  linux-amd64  Build for Linux x64"
	@echo "  linux-arm64  Build for Linux ARM64"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION      Plugin version (current: $(VERSION))"
	@echo "  GIT_COMMIT   Git commit hash (current: $(GIT_COMMIT))"
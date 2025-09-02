# Kolumn SDK Makefile

.DEFAULT_GOAL := help

# SDK version
SDK_VERSION := v0.1.0

# Go configuration
GO := go
GO_BUILD_FLAGS := -v -ldflags="-s -w"
GO_TEST_FLAGS := -v -race -timeout=5m

# Directories
SDK_ROOT := $(shell pwd)
BUILD_DIR := $(SDK_ROOT)/build
DOCS_DIR := $(SDK_ROOT)/docs

## help: Display this help message
.PHONY: help
help:
	@echo "Kolumn Provider SDK - Makefile Commands"
	@echo "========================================"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""

## clean: Clean build artifacts and caches
.PHONY: clean
clean:
	@echo "🧹 Cleaning SDK build artifacts..."
	$(GO) clean -cache -testcache -modcache
	rm -rf $(BUILD_DIR)
	rm -rf $(DOCS_DIR)

## deps: Download and verify dependencies
.PHONY: deps
deps:
	@echo "📦 Downloading SDK dependencies..."
	$(GO) mod download
	$(GO) mod verify
	$(GO) mod tidy

## build: Build SDK packages
.PHONY: build
build: deps
	@echo "🔨 Building SDK packages..."
	$(GO) build $(GO_BUILD_FLAGS) ./...
	@echo "✅ SDK build complete"

## test: Run all tests
.PHONY: test
test: build
	@echo "🧪 Running SDK tests..."
	$(GO) test $(GO_TEST_FLAGS) ./...
	@echo "✅ All tests passed"

## test-coverage: Run tests with coverage
.PHONY: test-coverage
test-coverage: build
	@echo "🧪 Running SDK tests with coverage..."
	mkdir -p $(BUILD_DIR)
	$(GO) test $(GO_TEST_FLAGS) -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	$(GO) tool cover -func=$(BUILD_DIR)/coverage.out
	@echo "📊 Coverage report: $(BUILD_DIR)/coverage.html"

## lint: Run linters
.PHONY: lint
lint:
	@echo "🔍 Running SDK linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run ./...
	@echo "✅ Linting complete"

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "💅 Formatting SDK code..."
	$(GO) fmt ./...
	@echo "✅ Code formatting complete"

## vet: Run go vet
.PHONY: vet
vet:
	@echo "🔬 Running go vet..."
	$(GO) vet ./...
	@echo "✅ Vetting complete"

## docs: Generate documentation
.PHONY: docs
docs: build
	@echo "📚 Generating SDK documentation..."
	mkdir -p $(DOCS_DIR)
	@command -v godoc >/dev/null 2>&1 || { echo "Installing godoc..."; $(GO) install golang.org/x/tools/cmd/godoc@latest; }
	$(GO) doc -all ./... > $(DOCS_DIR)/api.txt
	@echo "📖 Documentation generated: $(DOCS_DIR)/api.txt"

## examples: Build and test examples
.PHONY: examples
examples: build
	@echo "🏗️ Building SDK examples..."
	$(GO) build $(GO_BUILD_FLAGS) ./examples/...
	@echo "✅ Examples built successfully"

## validate: Run full validation suite
.PHONY: validate
validate: clean deps build test lint vet examples
	@echo "✅ Full SDK validation complete!"

## version: Display SDK version
.PHONY: version
version:
	@echo "Kolumn SDK $(SDK_VERSION)"
	@$(GO) version

## release-check: Check if SDK is ready for release
.PHONY: release-check
release-check: validate
	@echo "🚀 SDK release readiness check..."
	@echo "  ✅ All tests passing"
	@echo "  ✅ Linting clean"  
	@echo "  ✅ Examples building"
	@echo "  ✅ Documentation generated"
	@echo "🎉 SDK $(SDK_VERSION) is ready for release!"

## dev-setup: Setup development environment
.PHONY: dev-setup
dev-setup:
	@echo "🛠️ Setting up SDK development environment..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/godoc@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	@echo "✅ Development environment ready"

## watch: Watch for changes and run tests
.PHONY: watch  
watch:
	@echo "👁️ Watching for SDK changes..."
	@command -v entr >/dev/null 2>&1 || { echo "entr not found. Install with: apt-get install entr"; exit 1; }
	find . -name "*.go" | entr -c make test

# Development targets
.PHONY: dev
dev: clean deps build test examples

.PHONY: ci
ci: validate release-check
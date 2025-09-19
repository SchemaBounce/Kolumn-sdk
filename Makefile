# Makefile for Kolumn SDK Development
# Provides common development tasks and quality checks

# Variables
SHELL := /bin/bash
GO_VERSION := 1.21
PROJECT_NAME := kolumn-sdk
COVERAGE_THRESHOLD := 80

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

# Phony targets
.PHONY: help setup install-tools clean build test test-verbose test-coverage lint fmt check pre-commit validate-examples check-docs security deps-update deps-verify verify-all ci

## Help
help: ## Show this help message
	@echo -e "$(GREEN)Kolumn SDK Development Commands$(NC)"
	@echo -e "$(BLUE)===============================$(NC)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "$(YELLOW)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo
	@echo -e "$(BLUE)Common workflows:$(NC)"
	@echo -e "  $(YELLOW)make setup$(NC)     - Set up development environment"
	@echo -e "  $(YELLOW)make verify-all$(NC) - Run all quality checks"
	@echo -e "  $(YELLOW)make ci$(NC)        - Run CI/CD pipeline locally"

## Setup and Installation
setup: install-tools deps-verify ## Set up development environment
	@echo -e "$(GREEN)Setting up development environment...$(NC)"
	@echo -e "$(BLUE)Installing pre-commit hooks...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		pre-commit install --hook-type commit-msg; \
		echo -e "$(GREEN)✅ Pre-commit hooks installed$(NC)"; \
	else \
		echo -e "$(YELLOW)⚠️  pre-commit not found. Install with: pip install pre-commit$(NC)"; \
	fi
	@echo -e "$(GREEN)✅ Development environment ready!$(NC)"

install-tools: ## Install required development tools
	@echo -e "$(GREEN)Installing development tools...$(NC)"
	@echo -e "$(BLUE)Installing Go tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/securecodewarrior/github-action-add-sarif@latest || echo "gosec install skipped"
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/go-critic/go-critic/cmd/gocritic@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo -e "$(GREEN)✅ Go tools installed$(NC)"
	@echo -e "$(YELLOW)Note: Also install pre-commit: pip install pre-commit$(NC)"

## Building
build: ## Build all packages
	@echo -e "$(GREEN)Building all packages...$(NC)"
	@go build ./...
	@echo -e "$(GREEN)✅ Build completed successfully$(NC)"

clean: ## Clean build artifacts and cache
	@echo -e "$(GREEN)Cleaning build artifacts...$(NC)"
	@go clean -cache -testcache -modcache
	@rm -rf dist/ coverage.* *.out *.prof
	@find . -name "*.exe" -type f -delete 2>/dev/null || true
	@echo -e "$(GREEN)✅ Clean completed$(NC)"

## Testing
test: ## Run tests
	@echo -e "$(GREEN)Running tests...$(NC)"
	@go test -race ./...
	@echo -e "$(GREEN)✅ Tests passed$(NC)"

test-verbose: ## Run tests with verbose output
	@echo -e "$(GREEN)Running tests with verbose output...$(NC)"
	@go test -race -v ./...

test-coverage: ## Run tests with coverage analysis
	@echo -e "$(GREEN)Running tests with coverage analysis...$(NC)"
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo -e "$(BLUE)Coverage report generated: coverage.html$(NC)"
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo -e "$(BLUE)Total coverage: $${COVERAGE}%$(NC)"; \
	if (( $$(echo "$${COVERAGE} < $(COVERAGE_THRESHOLD)" | bc -l) )); then \
		echo -e "$(RED)❌ Coverage ($${COVERAGE}%) is below threshold ($(COVERAGE_THRESHOLD)%)$(NC)"; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ Coverage meets threshold$(NC)"; \
	fi

## Code Quality
lint: ## Run linting
	@echo -e "$(GREEN)Running linter...$(NC)"
	@golangci-lint run ./...
	@echo -e "$(GREEN)✅ Linting passed$(NC)"

fmt: ## Format code
	@echo -e "$(GREEN)Formatting code...$(NC)"
	@gofmt -s -w .
	@gofumpt -w .
	@goimports -w .
	@echo -e "$(GREEN)✅ Code formatted$(NC)"

check: lint test ## Run basic checks (lint + test)

## Pre-commit and Validation
pre-commit: ## Run pre-commit hooks manually
	@echo -e "$(GREEN)Running pre-commit hooks...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo -e "$(RED)❌ pre-commit not installed. Run: pip install pre-commit$(NC)"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ Pre-commit hooks passed$(NC)"

validate-examples: ## Validate example code
	@echo -e "$(GREEN)Validating example code...$(NC)"
	@./scripts/validate-examples.sh
	@echo -e "$(GREEN)✅ Examples validated$(NC)"

check-docs: ## Check documentation coverage
	@echo -e "$(GREEN)Checking documentation coverage...$(NC)"
	@./scripts/check-go-docs.sh
	@echo -e "$(GREEN)✅ Documentation checked$(NC)"

check-interfaces: ## Check Provider interface compliance
	@echo -e "$(GREEN)Checking interface compliance...$(NC)"
	@./scripts/check-interface-compliance.sh
	@echo -e "$(GREEN)✅ Interface compliance checked$(NC)"

check-naming: ## Check binary naming conventions
	@echo -e "$(GREEN)Checking binary naming conventions...$(NC)"
	@./scripts/check-binary-naming.sh
	@echo -e "$(GREEN)✅ Binary naming checked$(NC)"

check-schemas: ## Validate provider schemas
	@echo -e "$(GREEN)Validating provider schemas...$(NC)"
	@./scripts/validate-schemas.sh
	@echo -e "$(GREEN)✅ Schemas validated$(NC)"

check-banned-terms: ## Check for banned terms (terraform references)
	@echo -e "$(GREEN)Checking for banned terms...$(NC)"
	@./scripts/check-banned-terms.sh
	@echo -e "$(GREEN)✅ No banned terms found$(NC)"

## Security
security: ## Run security checks
	@echo -e "$(GREEN)Running security checks...$(NC)"
	@echo -e "$(BLUE)Running gosec...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./...; \
		echo -e "$(GREEN)✅ gosec security scan passed$(NC)"; \
	else \
		echo -e "$(YELLOW)⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/github-action-add-sarif@latest$(NC)"; \
	fi
	@echo -e "$(BLUE)Running secret detection...$(NC)"
	@if command -v detect-secrets >/dev/null 2>&1; then \
		detect-secrets scan --baseline .secrets.baseline; \
		echo -e "$(GREEN)✅ Secret detection passed$(NC)"; \
	else \
		echo -e "$(YELLOW)⚠️  detect-secrets not installed. Install with: pip install detect-secrets$(NC)"; \
	fi

## Dependencies
deps-update: ## Update dependencies
	@echo -e "$(GREEN)Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo -e "$(GREEN)✅ Dependencies updated$(NC)"

deps-verify: ## Verify dependencies
	@echo -e "$(GREEN)Verifying dependencies...$(NC)"
	@go mod verify
	@go mod tidy
	@echo -e "$(GREEN)✅ Dependencies verified$(NC)"

deps-graph: ## Show dependency graph
	@echo -e "$(GREEN)Generating dependency graph...$(NC)"
	@go mod graph

## Comprehensive Validation
verify-all: fmt check-banned-terms check-docs check-interfaces check-naming check-schemas validate-examples lint test security ## Run all quality checks
	@echo -e "$(GREEN)🎉 All quality checks passed!$(NC)"
	@echo -e "$(BLUE)✅ Code formatting$(NC)"
	@echo -e "$(BLUE)✅ Banned terms check$(NC)"
	@echo -e "$(BLUE)✅ Documentation coverage$(NC)"
	@echo -e "$(BLUE)✅ Interface compliance$(NC)"
	@echo -e "$(BLUE)✅ Binary naming$(NC)"
	@echo -e "$(BLUE)✅ Schema validation$(NC)"
	@echo -e "$(BLUE)✅ Example validation$(NC)"
	@echo -e "$(BLUE)✅ Linting$(NC)"
	@echo -e "$(BLUE)✅ Tests$(NC)"
	@echo -e "$(BLUE)✅ Security checks$(NC)"

## CI/CD Pipeline
ci: deps-verify verify-all test-coverage ## Run full CI/CD pipeline locally
	@echo -e "$(GREEN)🚀 CI/CD pipeline completed successfully!$(NC)"
	@echo -e "$(BLUE)This simulates the full CI/CD pipeline that would run on pull requests.$(NC)"

## Release Preparation
release-check: verify-all ## Check if code is ready for release
	@echo -e "$(GREEN)Checking release readiness...$(NC)"
	@echo -e "$(BLUE)Checking git status...$(NC)"
	@if [[ -n $$(git status --porcelain) ]]; then \
		echo -e "$(RED)❌ Working directory is not clean$(NC)"; \
		git status --short; \
		exit 1; \
	else \
		echo -e "$(GREEN)✅ Working directory is clean$(NC)"; \
	fi
	@echo -e "$(BLUE)Checking for CHANGELOG updates...$(NC)"
	@if [[ -f CHANGELOG.md ]]; then \
		echo -e "$(GREEN)✅ CHANGELOG.md found$(NC)"; \
	else \
		echo -e "$(YELLOW)⚠️  No CHANGELOG.md found$(NC)"; \
	fi
	@echo -e "$(GREEN)✅ Code is ready for release!$(NC)"

## Development Helpers
watch-test: ## Watch for changes and run tests
	@echo -e "$(GREEN)Watching for changes and running tests...$(NC)"
	@echo -e "$(YELLOW)Note: This requires 'entr'. Install with your package manager.$(NC)"
	@find . -name "*.go" | entr -c make test

dev-loop: ## Development loop with formatting and testing
	@echo -e "$(GREEN)Starting development loop...$(NC)"
	@while true; do \
		make fmt && make test; \
		echo -e "$(BLUE)Waiting for changes... (Ctrl+C to stop)$(NC)"; \
		find . -name "*.go" | entr -d -c echo "Files changed"; \
	done

## Information
version: ## Show Go version and tool versions
	@echo -e "$(GREEN)Environment Information$(NC)"
	@echo -e "$(BLUE)======================$(NC)"
	@echo -e "$(YELLOW)Go version:$(NC)"
	@go version
	@echo -e "$(YELLOW)Module information:$(NC)"
	@go list -m
	@echo -e "$(YELLOW)Installed tools:$(NC)"
	@echo -n "  golangci-lint: "; golangci-lint version 2>/dev/null || echo "not installed"
	@echo -n "  gofumpt: "; gofumpt -version 2>/dev/null || echo "not installed"
	@echo -n "  pre-commit: "; pre-commit --version 2>/dev/null || echo "not installed"
	@echo -n "  detect-secrets: "; detect-secrets --version 2>/dev/null || echo "not installed"

status: ## Show project status
	@echo -e "$(GREEN)Project Status$(NC)"
	@echo -e "$(BLUE)==============$(NC)"
	@echo -e "$(YELLOW)Git status:$(NC)"
	@git status --short || echo "Not a git repository"
	@echo -e "$(YELLOW)Go modules:$(NC)"
	@go list -m all | wc -l | xargs echo "  Dependencies:"
	@echo -e "$(YELLOW)Test files:$(NC)"
	@find . -name "*_test.go" | wc -l | xargs echo "  Test files:"
	@echo -e "$(YELLOW)Example files:$(NC)"
	@find examples -name "*.go" 2>/dev/null | wc -l | xargs echo "  Example files:" || echo "  Example files: 0"
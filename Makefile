# Project metadata
BINARY_NAME    := openmetric-proxycheck-exporter
VERSION        := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME     := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go settings
GO             := go
GOOS           ?= $(shell $(GO) env GOOS)
GOARCH         ?= $(shell $(GO) env GOARCH)

# Build settings
BUILD_DIR      := .build
LDFLAGS        := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Tool settings
GOLANGCI_LINT  := golangci-lint

# Output binary for current platform
BINARY         := $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)

# Default target
.DEFAULT_GOAL  := help

# ==============================================================================
# Build targets
# ==============================================================================

.PHONY: build
build: ## Build binary for current platform
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY) .
	@echo "Built: $(BINARY)"

.PHONY: build-linux-amd64
build-linux-amd64: ## Build for linux/amd64
	GOOS=linux GOARCH=amd64 $(MAKE) build

.PHONY: build-linux-arm64
build-linux-arm64: ## Build for linux/arm64
	GOOS=linux GOARCH=arm64 $(MAKE) build

.PHONY: build-darwin-amd64
build-darwin-amd64: ## Build for darwin/amd64
	GOOS=darwin GOARCH=amd64 $(MAKE) build

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build for darwin/arm64
	GOOS=darwin GOARCH=arm64 $(MAKE) build

.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 ## Build for all platforms

# ==============================================================================
# Test targets
# ==============================================================================

.PHONY: test
test: ## Run unit tests
	$(GO) test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	$(GO) test -race -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	@mkdir -p $(BUILD_DIR)
	$(GO) test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# ==============================================================================
# Code quality targets
# ==============================================================================

.PHONY: fmt
fmt: ## Format code with gofmt
	$(GO) fmt ./...

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: check
check: fmt lint test ## Run fmt, lint and test

# ==============================================================================
# Utility targets
# ==============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)

.PHONY: install
install: ## Install binary to GOPATH/bin
	$(GO) install $(LDFLAGS) .

.PHONY: run
run: build ## Run the binary locally
	./$(BINARY) -config config.yaml

.PHONY: help
help: ## Show this help message
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

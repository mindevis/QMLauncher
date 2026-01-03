# QMLauncher Makefile - CLI Version Only
.PHONY: help build clean lint fmt test deps linux macos windows release

# Build configuration
APP_NAME := QMLauncher
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build

# Detect current platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
	CURRENT_PLATFORM := linux
	CURRENT_ARCH := $(shell echo $(UNAME_M) | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
	APP_SUFFIX :=
endif
ifeq ($(UNAME_S),Darwin)
	CURRENT_PLATFORM := darwin
	CURRENT_ARCH := $(shell echo $(UNAME_M) | sed 's/x86_64/amd64/' | sed 's/arm64/arm64/')
	APP_SUFFIX :=
endif
ifeq ($(OS),Windows_NT)
	CURRENT_PLATFORM := windows
	CURRENT_ARCH := amd64
	APP_SUFFIX := .exe
endif

# Default target
help: ## Show this help message
	@echo "QMLauncher CLI - Command-line Minecraft launcher"
	@echo ""
	@echo "Build targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^build.*:.*?## / {printf "    %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Development and maintenance:"
	@awk 'BEGIN {FS = ":.*?## "} /^lint.*:.*?## / || /^fmt.*:.*?## / || /^vet.*:.*?## / || /^test.*:.*?## / || /^check.*:.*?## / || /^clean.*:.*?## / {printf "    %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build CLI application for current platform
	@echo "Building $(APP_NAME) CLI $(VERSION) for $(CURRENT_PLATFORM)/$(CURRENT_ARCH)..."
	@mkdir -p $(BUILD_DIR)
	go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX) .
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX)"

linux: ## Build for Linux (amd64)
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-linux-amd64 .
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-linux-amd64"

macos: ## Build for macOS (amd64)
	@echo "Building for macOS amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64 .
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64"

macos-arm64: ## Build for macOS (arm64)
	@echo "Building for macOS arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-arm64 .
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-darwin-arm64"

windows: ## Build for Windows (amd64)
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe .
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe"

release: linux macos macos-arm64 windows ## Build for all platforms

# Development and maintenance
lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

check: lint vet test ## Run all checks

deps: ## Download dependencies
	go mod download
	go mod tidy

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean
# QMLauncher Makefile
.PHONY: help build dev clean lint fmt test deps frontend-lint frontend-fmt linux macos windows release linux macos windows release

# Build configuration
APP_NAME := QMLauncher
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build
PROJECT_ROOT := $(shell pwd)

# Detect current platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
	CURRENT_PLATFORM := linux
	CURRENT_ARCH := $(UNAME_M)
	APP_SUFFIX :=
endif
ifeq ($(UNAME_S),Darwin)
	CURRENT_PLATFORM := darwin
	CURRENT_ARCH := $(UNAME_M)
	APP_SUFFIX :=
endif
ifeq ($(OS),Windows_NT)
	CURRENT_PLATFORM := windows
	CURRENT_ARCH := amd64
	APP_SUFFIX := .exe
endif

# Default target
help: ## Show this help message
	@echo "QMLauncher - Desktop application built with Wails, Go and Vue.js"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the application for production
	wails build

build-dev: ## Build the application in development mode
	wails build -dev

build-debug: ## Build the application with debug information
	wails build -debug

build-clean: ## Clean build artifacts and rebuild
	rm -rf $(BUILD_DIR)/$(APP_NAME)*
	wails build

# Development targets
dev: ## Run the application in development mode
	wails dev

dev-clean: ## Clean and run development mode
	rm -rf frontend/node_modules/.vite
	wails dev

# Go targets
lint: ## Run golangci-lint on Go code
	golangci-lint run

fmt: ## Format Go code
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

mod-tidy: ## Clean up go.mod and go.sum
	go mod tidy

test: ## Run Go tests
	go test ./...

# Frontend targets
frontend-install: ## Install frontend dependencies
	cd frontend && npm install

frontend-dev: ## Run frontend development server only
	cd frontend && npm run dev

frontend-build: ## Build frontend for production
	cd frontend && npm run build

frontend-lint: ## Run ESLint on frontend code
	cd frontend && npm run lint

frontend-fmt: ## Format frontend code
	cd frontend && npm run format

# Combined targets
deps: mod-tidy frontend-install ## Install all dependencies (Go and Node.js)

check: fmt lint test frontend-lint ## Run all checks (format, lint, test)

clean: ## Clean all build artifacts and caches
	rm -rf $(BUILD_DIR)/$(APP_NAME)*
	rm -rf frontend/node_modules/.vite
	rm -rf frontend/dist
	go clean

# Install development tools
install-tools: ## Install development tools (golangci-lint, goimports)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Docker targets (optional)
docker-build: ## Build application in Docker
	docker build -t qmlauncher .

docker-run: ## Run application in Docker
	docker run --rm -it qmlauncher

# Cross-platform build targets
define build_target
	@echo "Building $(APP_NAME) v$(VERSION) for $(1) $(2)..."
	mkdir -p $(BUILD_DIR)
	@if [ "$(1)" = "darwin" ] && [ "$(UNAME_S)" != "Darwin" ]; then \
		echo "⚠️  Cross-compilation to macOS is not supported on $(UNAME_S)"; \
		echo "   Please build on macOS or use CI/CD with macOS runners"; \
		exit 1; \
	fi
	wails build -platform $(1)/$(2)
	@if [ -f build/bin/$(APP_NAME)$(if $(filter windows,$(1)),.exe,) ]; then \
		mv build/bin/$(APP_NAME)$(if $(filter windows,$(1)),.exe,) $(BUILD_DIR)/$(APP_NAME)-$(1)-$(2)$(if $(filter windows,$(1)),.exe,); \
		echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-$(1)-$(2)$(if $(filter windows,$(1)),.exe,)"; \
	else \
		echo "❌ Build failed or file not found"; \
		exit 1; \
	fi
endef

linux: ## Build for Linux (current architecture)
	$(call build_target,linux,$(CURRENT_ARCH))

linux-amd64: ## Build for Linux AMD64
	$(call build_target,linux,amd64)

linux-arm64: ## Build for Linux ARM64
	$(call build_target,linux,arm64)

macos: ## Build for macOS (current architecture)
	$(call build_target,darwin,$(CURRENT_ARCH))

macos-amd64: ## Build for macOS AMD64 (Intel)
	$(call build_target,darwin,amd64)

macos-arm64: ## Build for macOS ARM64 (Apple Silicon)
	$(call build_target,darwin,arm64)

windows: ## Build for Windows (current architecture)
	$(call build_target,windows,$(CURRENT_ARCH))

windows-amd64: ## Build for Windows AMD64
	$(call build_target,windows,amd64)

windows-arm64: ## Build for Windows ARM64
	$(call build_target,windows,arm64)

# Release targets (legacy - use platform-specific commands above)
release-linux: linux-amd64 ## Build for Linux (legacy)
release-windows: windows-amd64 ## Build for Windows (legacy)
release-darwin: macos-amd64 ## Build for macOS (legacy)

release: linux-amd64 windows-amd64 ## Build for all major platforms (AMD64)
	@echo ""
	@echo "🎉 Release builds completed!"
	@echo "Built applications are in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/ | grep $(APP_NAME)

release-all: linux-amd64 linux-arm64 windows-amd64 windows-arm64 ## Build for all platforms and architectures (except macOS cross-compilation)
	@echo ""
	@echo "🎉 All platform builds completed!"
	@echo "Built applications are in $(BUILD_DIR)/"
	@echo "Note: macOS builds require building on macOS or using CI/CD with macOS runners"
	@ls -la $(BUILD_DIR)/ | grep $(APP_NAME)

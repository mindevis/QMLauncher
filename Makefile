# QMLauncher Makefile - CLI Version Only
.PHONY: help build clean lint fmt test deps linux macos windows release

# Build configuration
APP_NAME := QMLauncher
VERSION := 1.0.0
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

# Icon management
check-icon: ## Check if icon files exist and are valid
	@echo "Checking icon files..."
	@cd assets && ./check-icon.sh

convert-icons: ## Convert PNG icon to other formats (ICO, ICNS)
	@echo "Converting icons..."
	@cd assets && ./convert-icons.sh

prepare-icons: ## Prepare icons for all platforms (convert if needed)
	@echo "Preparing icons for all platforms..."
	@if [ -f "assets/icon.png" ]; then \
		echo "PNG icon found, converting to other formats..."; \
		$(MAKE) convert-icons; \
	else \
		echo "No PNG icon found in assets/ - icons will not be used"; \
	fi

# Default target
help: ## Show this help message
	@echo "QMLauncher CLI - Command-line Minecraft launcher"
	@echo ""
	@echo "Build targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^build.*:.*?## / {printf "    %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Icon management:"
	@awk 'BEGIN {FS = ":.*?## "} /^check-icon.*:.*?## / || /^convert-icons.*:.*?## / || /^prepare-icons.*:.*?## / {printf "    %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Platform builds:"
	@awk 'BEGIN {FS = ":.*?## "} /^linux.*:.*?## / || /^macos.*:.*?## / || /^windows.*:.*?## / {printf "    %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Development and maintenance:"
	@awk 'BEGIN {FS = ":.*?## "} /^check.*:.*?## / || /^lint.*:.*?## / || /^fmt.*:.*?## / || /^vet.*:.*?## / || /^test.*:.*?## / || /^install-hooks.*:.*?## / || /^deps.*:.*?## / || /^clean.*:.*?## / {printf "    %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

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
	@if [ -f "assets/icon.png" ]; then \
		echo "Copying icon.png for Linux..."; \
		cp assets/icon.png $(BUILD_DIR)/; \
	fi

macos: prepare-icons ## Build for macOS (amd64) with icon if available
	@echo "Building for macOS amd64..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f "assets/icon.icns" ]; then \
		echo "Building with ICNS icon..."; \
		GOOS=darwin GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64 .; \
	else \
		echo "Building without icon (ICNS not found)..."; \
		GOOS=darwin GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64 .; \
	fi
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64"

macos-arm64: prepare-icons ## Build for macOS (arm64) with icon if available
	@echo "Building for macOS arm64..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f "assets/icon.icns" ]; then \
		echo "Building with ICNS icon..."; \
		GOOS=darwin GOARCH=arm64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-arm64 .; \
	else \
		echo "Building without icon (ICNS not found)..."; \
		GOOS=darwin GOARCH=arm64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-arm64 .; \
	fi
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-darwin-arm64"

windows: prepare-icons ## Build for Windows (amd64) with icon if available
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f "assets/icon.ico" ]; then \
		echo "Building with ICO icon..."; \
		GOOS=windows GOARCH=amd64 go build -tags cli -ldflags "-H windowsgui -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe .; \
	else \
		echo "Building without icon (ICO not found)..."; \
		GOOS=windows GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe .; \
	fi
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe"

# Build targets with icons
build-with-icon: prepare-icons ## Build CLI application for current platform with icon
	@echo "Building $(APP_NAME) CLI $(VERSION) for $(CURRENT_PLATFORM)/$(CURRENT_ARCH) with icon..."
	@mkdir -p $(BUILD_DIR)
	@if [ "$(CURRENT_PLATFORM)" = "windows" ]; then \
		if [ -f "assets/icon.ico" ]; then \
			echo "Using ICO icon for Windows..."; \
			go build -tags cli -ldflags "-H windowsgui -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX) .; \
		else \
			echo "ICO icon not found for Windows, building without icon..."; \
			go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX) .; \
		fi \
	elif [ "$(CURRENT_PLATFORM)" = "darwin" ]; then \
		if [ -f "assets/icon.icns" ]; then \
			echo "Using ICNS icon for macOS..."; \
			go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX) .; \
		else \
			echo "ICNS icon not found for macOS, building without icon..."; \
			go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX) .; \
		fi \
	else \
		echo "Building for $(CURRENT_PLATFORM) (icons not embedded)..."; \
		go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX) .; \
	fi
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-$(CURRENT_PLATFORM)-$(CURRENT_ARCH)$(APP_SUFFIX)"

windows-with-icon: prepare-icons ## Build for Windows (amd64) with icon
	@echo "Building for Windows amd64 with icon..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f "assets/icon.ico" ]; then \
		echo "Using ICO icon..."; \
		GOOS=windows GOARCH=amd64 go build -tags cli -ldflags "-H windowsgui -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe .; \
	else \
		echo "ICO icon not found, building without icon..."; \
		GOOS=windows GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe .; \
	fi
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-windows-amd64.exe"

macos-with-icon: prepare-icons ## Build for macOS (amd64) with icon
	@echo "Building for macOS amd64 with icon..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f "assets/icon.icns" ]; then \
		echo "Using ICNS icon..."; \
		GOOS=darwin GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64 .; \
	else \
		echo "ICNS icon not found, building without icon..."; \
		GOOS=darwin GOARCH=amd64 go build -tags cli -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64 .; \
	fi
	@echo "✓ Built: $(BUILD_DIR)/$(APP_NAME)-cli-darwin-amd64"

release: prepare-icons linux macos macos-arm64 windows ## Build for all platforms with icons

# Development and maintenance
lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

check: fmt vet test ## Run basic checks (formatting, vet, tests)
	@if [ "$$SKIP_LINT" != "1" ]; then \
		echo "🔍 Running lint checks..."; \
		$(MAKE) lint || { echo "⚠️  Lint issues found. Run 'SKIP_LINT=1 make check' to skip."; exit 1; }; \
	fi

deps: ## Download dependencies
	go mod download
	go mod tidy

install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	mkdir -p .git/hooks
	cp hooks/pre-push .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	@echo "✅ Git hooks installed"

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean
.PHONY: all build clean test install lint fmt vet run help completion install-completion-bash install-completion-zsh install-completion-fish install-completion-powershell

# Variables
BINARY_NAME=adoctl
CMD_DIR=cmd/adoctl
BUILD_DIR=dist
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Linker flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Go build flags
GOFLAGS=-v

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

## build-release: Build release binaries for multiple platforms
build-release: clean
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "Release binaries built in $(BUILD_DIR)/"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@go clean
	@echo "Cleaned"

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

## test-cover: Run tests with coverage report
test-cover: test
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## install: Install binary to GOPATH/bin
install:
	@echo "Installing to GOPATH/bin..."
	go install $(LDFLAGS) ./$(CMD_DIR)
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin" && exit 1)
	golangci-lint run ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## run: Build and run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run in development mode
dev:
	go run $(LDFLAGS) ./$(CMD_DIR) $(ARGS)

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

## completion: Generate shell completions
completion: build
	@echo "Generating shell completions..."
	@mkdir -p completions
	@$(BUILD_DIR)/$(BINARY_NAME) completion bash > completions/$(BINARY_NAME).bash
	@$(BUILD_DIR)/$(BINARY_NAME) completion zsh > completions/$(BINARY_NAME).zsh
	@$(BUILD_DIR)/$(BINARY_NAME) completion fish > completions/$(BINARY_NAME).fish
	@$(BUILD_DIR)/$(BINARY_NAME) completion powershell > completions/$(BINARY_NAME).ps1
	@echo "Completions generated in completions/"

## install-completion-bash: Install bash completion
install-completion-bash:
	@echo "Installing bash completion..."
	@mkdir -p ~/.local/share/bash-completion/completions
	@$(BUILD_DIR)/$(BINARY_NAME) completion bash > ~/.local/share/bash-completion/completions/$(BINARY_NAME)
	@echo "Add to ~/.bashrc: source ~/.local/share/bash-completion/completions/$(BINARY_NAME)"
	@echo "Or run: source ~/.local/share/bash-completion/completions/$(BINARY_NAME)"

## install-completion-zsh: Install zsh completion
install-completion-zsh:
	@echo "Installing zsh completion..."
	@mkdir -p ~/.zsh/completions
	@$(BUILD_DIR)/$(BINARY_NAME) completion zsh > ~/.zsh/completions/_$(BINARY_NAME)
	@if ! grep -q "~/.zsh/completions" ~/.zshrc 2>/dev/null; then \
		echo "" >> ~/.zshrc; \
		echo "# Add adoctl completions" >> ~/.zshrc; \
		echo "fpath=(~/.zsh/completions \$$${fpath})" >> ~/.zshrc; \
		echo "Added completion setup to ~/.zshrc"; \
		echo ""; \
		echo "IMPORTANT: If you already have 'compinit' in your .zshrc, just reload with: exec zsh"; \
		echo "If you don't see 'compinit' in .zshrc, add it after the fpath line."; \
	else \
		echo "Completion path already configured in ~/.zshrc"; \
	fi
	@echo "Reload your shell: exec zsh"

## install-completion-fish: Install fish completion
install-completion-fish:
	@echo "Installing fish completion..."
	@mkdir -p ~/.config/fish/completions
	@$(BUILD_DIR)/$(BINARY_NAME) completion fish > ~/.config/fish/completions/$(BINARY_NAME).fish
	@echo "Fish completion installed. Reload your fish shell or run: source ~/.config/fish/completions/$(BINARY_NAME).fish"

## install-completion-powershell: Install PowerShell completion
install-completion-powershell:
	@echo "Installing PowerShell completion..."
	@$(if command -v pwsh > /dev/null; then \
		$(BUILD_DIR)/$(BINARY_NAME) completion powershell >> $(pwsh -NoProfile -Command '$PROFILE'); \
		echo "PowerShell completion installed. Reload your PowerShell."; \
	else \
		echo "PowerShell (pwsh) not found."; \
	fi)

## help: Show this help message
help:
	@echo "$(BINARY_NAME) - Build and Development Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help


.PHONY: build test lint clean tidy install help all

# Binary name
BINARY_NAME=ralph

# Build output directory
BUILD_DIR=bin

# Version information (can be overridden)
VERSION?=0.1.0
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION?=$(shell go version | awk '{print $$3}')

# Linker flags for version information
LDFLAGS=-ldflags "-X github.com/JanDeDobbeleer/copilot-ralph/pkg/version.Version=$(VERSION) \
                  -X github.com/JanDeDobbeleer/copilot-ralph/pkg/version.Commit=$(COMMIT) \
                  -X github.com/JanDeDobbeleer/copilot-ralph/pkg/version.BuildDate=$(BUILD_DATE) \
                  -X github.com/JanDeDobbeleer/copilot-ralph/pkg/version.GoVersion=$(GO_VERSION)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the ralph binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ralph
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

install: ## Install ralph to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/ralph
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -w -s .
	@echo "Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy
	go mod verify
	@echo "Dependencies tidied"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean
	@echo "Clean complete"

all: tidy fmt vet lint test build ## Run all checks and build

# Development helpers
dev-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development dependencies installed"

run: build ## Build and run ralph
	@$(BUILD_DIR)/$(BINARY_NAME)

run-init: build ## Build and run ralph init
	@$(BUILD_DIR)/$(BINARY_NAME) init

run-version: build ## Build and run ralph version
	@$(BUILD_DIR)/$(BINARY_NAME) version

# Docker targets (for future use)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run: docker-build ## Build and run Docker container
	docker run --rm -it $(BINARY_NAME):$(VERSION)

# Release targets
release-snapshot: ## Create a snapshot release with goreleaser
	@echo "Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not installed. Install with:"; \
		echo "  go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# Quick commands
.PHONY: b t l c
b: build   ## Alias for build
t: test    ## Alias for test
l: lint    ## Alias for lint
c: clean   ## Alias for clean

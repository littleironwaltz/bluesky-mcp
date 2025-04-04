.PHONY: build build-cli test lint clean run run-cli

# Binary names
BINARY_NAME=bluesky-mcp
CLI_BINARY_NAME=bluesky-mcp-cli
# Build directory
BUILD_DIR=bin

# Get the current directory
CURRENT_DIR=$(shell pwd)

# Default build target
build:
	@echo "Building ${BINARY_NAME} server..."
	@go build -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/bluesky-mcp

# Build CLI tool
build-cli:
	@echo "Building ${CLI_BINARY_NAME} CLI..."
	@go build -o ${BUILD_DIR}/${CLI_BINARY_NAME} ./cmd/cli

# Build both server and CLI
build-all: build build-cli

# Run the application
run: build
	@echo "Running ${BINARY_NAME} server..."
	@./${BUILD_DIR}/${BINARY_NAME}

# Run the CLI tool
run-cli: build-cli
	@echo "Running ${CLI_BINARY_NAME} CLI..."
	@./${BUILD_DIR}/${CLI_BINARY_NAME}

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

# Run linter
lint:
	@echo "Running linter..."
	@go install golang.org/x/lint/golint@latest
	@golint ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ${BUILD_DIR}

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/bluesky-mcp
	@GOOS=darwin GOARCH=amd64 go build -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ./cmd/bluesky-mcp
	@GOOS=windows GOARCH=amd64 go build -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/bluesky-mcp

# Default target
all: deps lint test build-all
.PHONY: build test lint clean run

# Binary name
BINARY_NAME=bluesky-mcp
# Build directory
BUILD_DIR=bin

# Get the current directory
CURRENT_DIR=$(shell pwd)

# Default build target
build:
	@echo "Building ${BINARY_NAME}..."
	@go build -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/bluesky-mcp

# Run the application
run: build
	@echo "Running ${BINARY_NAME}..."
	@./${BUILD_DIR}/${BINARY_NAME}

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
all: deps lint test build
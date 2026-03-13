.PHONY: all build format docs clean test help

# Default target - run format, docs, and build
all: format docs build

# Default target
help:
	@echo "Available commands:"
	@echo "  make build    - Build the service binary"
	@echo "  make format   - Format Go code"
	@echo "  make docs     - Regenerate Swagger documentation"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Clean build artifacts"

# Build the service
build:
	@echo "Building service..."
	@go build -o bin/service ./cmd/service

# Format Go code
format:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w . || true

# Regenerate Swagger documentation
docs:
	@echo "Regenerating Swagger docs..."
	@swag init -g cmd/service/main.go --parseDependency --parseInternal
	@echo "Swagger docs regenerated successfully"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf docs/
	@echo "Clean complete"

.PHONY: help build test clean migrate-up migrate-down migrate-reset dev-setup build-all build-windows build-linux build-darwin deploy

# Default target
help:
	@echo "Available targets:"
	@echo "  build       - Build the application for current platform"
	@echo "  build-all   - Build for all platforms (Linux, Windows, macOS)"
	@echo "  build-windows - Build for Windows"
	@echo "  build-linux - Build for Linux"
	@echo "  build-darwin - Build for macOS"
	@echo "  test        - Run all tests"
	@echo "  test-unit   - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  clean       - Clean build artifacts and databases"
	@echo "  migrate-up  - Run database migrations"
	@echo "  migrate-down - Rollback database migrations"
	@echo "  migrate-reset - Reset database and re-run migrations"
	@echo "  dev-setup   - Set up development environment"
	@echo "  deploy      - Deploy to development environment"
	@echo "  lint        - Run linter"
	@echo "  deps        - Install dependencies"

# Build the application
build:
	go build -o bin/bank-api cmd/server/main.go

# Build for all platforms
build-all:
	chmod +x scripts/build.sh
	./scripts/build.sh

# Build for Windows
build-windows:
	go build -ldflags "-X main.Version=dev -X main.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o build/bank-api-windows-amd64.exe cmd/server/main.go

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=dev -X main.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o build/bank-api-linux-amd64 cmd/server/main.go

# Build for macOS
build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=dev -X main.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o build/bank-api-darwin-amd64 cmd/server/main.go

# Run all tests
test:
	go test -v ./...

# Run unit tests only
test-unit:
	go test -v ./internal/handlers/tests/unit/...

# Run integration tests only
test-integration:
	go test -v ./internal/handlers/tests/integration/...

# Clean build artifacts and databases
clean:
	rm -rf bin/
	rm -f bank.db
	rm -f internal/db/bank.db
	rm -f *.db

# Run database migrations
migrate-up:
	go run cmd/migrate/main.go up

# Rollback database migrations
migrate-down:
	go run cmd/migrate/main.go down

# Reset database and re-run migrations
migrate-reset:
	go run cmd/migrate/main.go reset

# Set up development environment
dev-setup:
	@echo "Setting up development environment..."
	@mkdir -p bin
	@mkdir -p internal/db
	@mkdir -p build
	go mod tidy
	go run cmd/migrate/main.go up
	@echo "Development environment ready!"

# Deploy to development environment
deploy:
	@echo "Deploying to development environment..."
	@mkdir -p build
	go build -o build/bank-api cmd/server/main.go
	@echo "Starting development server..."
	./build/bank-api

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
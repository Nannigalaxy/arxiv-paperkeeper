.PHONY: build run fetch test clean docker-build docker-run compose-restart

# Build the application
build:
	@echo "Building ArXiv Paperkeeper..."
	@go build -o bin/arxiv-paperkeeper ./cmd/server

# Run the server
run:
	@echo "Starting server..."
	@go run ./cmd/server server

# Fetch papers manually
fetch:
	@echo "Fetching papers from arXiv..."
	@go run ./cmd/server fetch

# Run database migrations
migrate:
	@echo "Running database migrations..."
	@go run ./cmd/server migrate

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f data/arxiv.db

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t arxiv-paperkeeper:latest .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -d \
		-p 8080:8080 \
		-v $(PWD)/data:/root/data \
		--name arxiv-paperkeeper \
		arxiv-paperkeeper:latest

# Stop Docker container
docker-stop:
	@docker stop arxiv-paperkeeper
	@docker rm arxiv-paperkeeper

# Docker Compose - Start
compose-up:
	@echo "Starting with Docker Compose..."
	@docker compose up -d

# Docker Compose - Stop
compose-down:
	@echo "Stopping Docker Compose..."
	@docker compose down

# Docker Compose - Logs
compose-logs:
	@docker compose logs -f

# Docker Compose - Restart (rebuild and up)
compose-restart:
	@echo "Restarting Docker Compose with rebuild..."
	@docker compose down
	@docker compose up -d --build

# Development mode with auto-reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	@golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the server"
	@echo "  fetch        - Manually fetch papers from arXiv"
	@echo "  migrate      - Run database migrations"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-stop  - Stop Docker container"
	@echo "  compose-up   - Start with Docker Compose"
	@echo "  compose-down - Stop Docker Compose"
	@echo "  compose-restart - Rebuild and restart Docker Compose"
	@echo "  compose-logs - View Docker Compose logs"
	@echo "  dev          - Run in development mode with auto-reload"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  help         - Show this help message"

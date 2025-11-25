.PHONY: help migrate-up migrate-down migrate-status migrate-create test build run

# Default target
help:
	@echo "BrandishBot_Go - Makefile Commands"
	@echo ""
	@echo "Migration Commands:"
	@echo "  make migrate-up           - Run all pending migrations"
	@echo "  make migrate-down         - Rollback the last migration"
	@echo "  make migrate-status       - Show migration status"
	@echo "  make migrate-create NAME= - Create a new migration file"
	@echo ""
	@echo "Development Commands:"
	@echo "  make test                 - Run all tests with coverage"
	@echo "  make build                - Build all binaries"
	@echo "  make run                  - Run the application"
	@echo "  make swagger              - Generate Swagger docs"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-up            - Start services with Docker Compose"
	@echo "  make docker-down          - Stop services"

# Database connection string from environment
DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Migration commands
migrate-up:
	@echo "Running migrations..."
	@goose -dir migrations postgres "$(DB_URL)" up

migrate-down:
	@echo "Rolling back migration..."
	@goose -dir migrations postgres "$(DB_URL)" down

migrate-status:
	@echo "Migration status:"
	@goose -dir migrations postgres "$(DB_URL)" status

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	@goose -dir migrations create $(NAME) sql

# Development commands
test:
	@echo "Running tests..."
	@go test ./... -cover -race

test-coverage:
	@echo "Generating coverage report..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

build:
	@echo "Building binaries..."
	@go build -o bin/brandishbot cmd/app/main.go
	@go build -o bin/setup cmd/setup/main.go
	@echo "Build complete: bin/"

run:
	@echo "Starting BrandishBot..."
	@go run cmd/app/main.go

swagger:
	@echo "Generating Swagger documentation..."
	@$$HOME/go/bin/swag init -g cmd/app/main.go --output ./docs/swagger
	@echo "Swagger docs updated: docs/swagger/"

# Docker commands
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

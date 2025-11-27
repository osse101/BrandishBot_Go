.PHONY: help migrate-up migrate-down migrate-status migrate-create test build run

# Tool paths
GOOSE := $(shell command -v goose 2> /dev/null || echo $(HOME)/go/bin/goose)
SWAG := $(shell command -v swag 2> /dev/null || echo $(HOME)/go/bin/swag)
LINT := $(shell command -v golangci-lint 2> /dev/null || echo $(HOME)/go/bin/golangci-lint)

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
	@echo "  make test-coverage        - Generate  HTML coverage report"
	@echo "  make test-coverage-check  - Verify 80%+ coverage threshold"
	@echo "  make lint                 - Run code linters"
	@echo "  make lint-fix             - Run linters with auto-fix"
	@echo "  make build                - Build all binaries"
	@echo "  make run                  - Run the application"
	@echo "  make swagger              - Generate Swagger docs"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-up            - Start services with Docker Compose"
	@echo "  make docker-down          - Stop services"
	@echo ""
	@echo "Test Database Commands:"
	@echo "  make test-integration     - Run integration tests (uses testcontainers)"
	@echo "  make test-staging         - Run staging integration tests"
	@echo "  make db-test-up           - Start test database on port 5433"
	@echo "  make db-test-down         - Stop test database"
	@echo "  make migrate-up-test      - Run migrations on test database"
	@echo "  make db-seed-test         - Load test seed data"
	@echo "  make db-export            - Export production DB to backup.sql"
	@echo "  make db-import            - Import backup.sql to test DB"
	@echo "  make db-clean-test        - Clean test database"

# Database connection string from environment
DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Migration commands
migrate-up:
	@echo "Running migrations..."
	@$(GOOSE) -dir migrations postgres "$(DB_URL)" up

migrate-down:
	@echo "Rolling back migration..."
	@$(GOOSE) -dir migrations postgres "$(DB_URL)" down

migrate-status:
	@echo "Migration status:"
	@$(GOOSE) -dir migrations postgres "$(DB_URL)" status

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	@$(GOOSE) -dir migrations create $(NAME) sql

# Development commands
test:
	@echo "Running tests..."
	@go test ./... -cover -race

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p logs
	@go test -coverprofile=logs/coverage.out -covermode=atomic ./...
	@go tool cover -html=logs/coverage.out -o logs/coverage.html
	@COVERAGE=$$(go tool cover -func=logs/coverage.out | grep total | awk '{print $$3}'); \
	echo "Coverage report generated: logs/coverage.html"; \
	echo "Total Coverage: $$COVERAGE"

test-coverage-check:
	@echo "Checking coverage threshold (80%)..."
	@mkdir -p logs
	@go test -coverprofile=logs/coverage.out -covermode=atomic ./... >/dev/null 2>&1
	@COVERAGE=$$(go tool cover -func=logs/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=80; \
	if [ $$(echo "$$COVERAGE < $$THRESHOLD" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below $$THRESHOLD% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets $$THRESHOLD% threshold"; \
	fi

lint:
	@echo "Running linters..."
	@$(LINT) run ./...

lint-fix:
	@echo "Running linters with auto-fix..."
	@$(LINT) run --fix ./...

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

# Test database commands
test-integration:
	@echo "Running integration tests..."
	@go test ./internal/database/postgres -v -timeout=30s

test-staging:
	@echo "Running staging integration tests..."
	@echo "Target: $${STAGING_URL:-http://localhost:8080}"
	@go test -tags=staging -v ./tests/staging

db-test-up:
	@echo "Starting test database..."
	@docker-compose -f docker-compose.test.yml up -d
	@sleep 2
	@echo "Test database ready on port 5433"

db-test-down:
	@echo "Stopping test database..."
	@docker-compose -f docker-compose.test.yml down

migrate-up-test:
	@echo "Running migrations on test database..."
	@goose -dir migrations postgres "postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable" up

migrate-down-test:
	@echo "Rolling back test database migration..."
	@goose -dir migrations postgres "postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable" down

migrate-status-test:
	@echo "Test database migration status:"
	@goose -dir migrations postgres "postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable" status

db-seed-test:
	@echo "Seeding test database..."
	@docker exec -i brandishbot_test_db psql -U testuser -d testdb < scripts/setup_test_user.sql
	@docker exec -i brandishbot_test_db psql -U testuser -d testdb < scripts/seed_test_recipe.sql
	@echo "Test database seeded successfully"

db-export:
	@echo "Exporting production database..."
	@docker exec brandishbot_go-db-1 pg_dump -U $(DB_USER) -d $(DB_NAME) > backup.sql
	@echo "Database exported to backup.sql"

db-import:
	@echo "Importing data into test database..."
	@docker exec -i brandishbot_test_db psql -U testuser -d testdb < backup.sql
	@echo "Data imported successfully"

db-clean-test:
	@echo "Cleaning test database..."
	@docker exec brandishbot_test_db psql -U testuser -d testdb -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "Test database cleaned"
	@echo "Run 'make migrate-up-test' to reinitialize schema"


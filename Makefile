.PHONY: help migrate-up migrate-down migrate-status migrate-create test build run clean docker-build docker-up docker-down deploy-staging deploy-production rollback-staging rollback-production health-check-staging health-check-prod

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
	@echo "  make build                - Build all binaries to bin/"
	@echo "  make clean                - Remove build artifacts (bin/)"
	@echo "  make run                  - Run the application from bin/app"
	@echo "  make swagger              - Generate Swagger docs"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-up            - Start services with Docker Compose"
	@echo "  make docker-down          - Stop services"
	@echo "  make docker-build         - Rebuild Docker images (no cache, slower but clean)"
	@echo "  make docker-build-fast    - Build Docker images (with cache, faster for dev)"
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
	@echo ""
	@echo "Deployment Commands:"
	@echo "  make deploy-staging       - Deploy to staging environment"
	@echo "  make deploy-production    - Deploy to production environment (requires confirmation)"
	@echo "  make rollback-staging     - Rollback staging to previous version"
	@echo "  make rollback-production  - Rollback production to previous version"
	@echo "  make health-check-staging - Check staging environment health"
	@echo "  make health-check-prod    - Check production environment health"

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

# Build targets
build:
	@echo "Building all binaries to bin/..."
	@mkdir -p bin
	@go build -o bin/app ./cmd/app
	@go build -o bin/discord_bot ./cmd/discord
	@echo "✓ Built: bin/app"
	@echo "✓ Built: bin/discord_bot"

# Discord bot - Run locally
.PHONY: discord-run
discord-run:
	@echo "Starting Discord bot..."
	@./bin/discord_bot

# Discord bot - View logs (Docker)
.PHONY: discord-logs
discord-logs:
	@docker-compose logs -f discord

# Discord - Build Docker image
.PHONY: docker-discord-build
docker-discord-build:
	@echo "Building Discord bot Docker image..."
	@docker build -f Dockerfile.discord -t brandishbot-discord:dev .
	@echo "✓ Built: brandishbot-discord:dev"

# Discord - Start Discord service only
.PHONY: docker-discord-up
docker-discord-up:
	@echo "Starting Discord bot service..."
	@docker-compose up -d discord
	@echo "✓ Discord bot started"

# Discord - Restart Discord service
.PHONY: docker-discord-restart
docker-discord-restart:
	@echo "Restarting Discord bot..."
	@docker-compose restart discord
	@echo "✓ Discord bot restarted"

# Development shortcuts
run:
	@echo "Starting BrandishBot from bin/app..."
	@./bin/app

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✓ Removed bin/ directory"

swagger:
	@echo "Generating Swagger documentation..."
	@$$HOME/go/bin/swag init -g cmd/app/main.go --output ./docs/swagger
	@echo "Swagger docs updated: docs/swagger/"

# Docker commands
docker-up:
	@echo "Starting Docker services..."
	@docker compose up -d

docker-down:
	@echo "Stopping Docker services..."
	@docker compose down

docker-build:
	@echo "Rebuilding Docker images (no cache)..."
	@docker compose build --no-cache
	@echo "Docker images rebuilt successfully"

docker-build-fast:
	@echo "Building Docker images (with cache, faster)..."
	@DOCKER_BUILDKIT=1 docker compose build
	@echo "Docker images built successfully"

docker-logs:
	@docker compose logs -f

push-staging:
	@echo "Pushing staging image..."
	@./scripts/push_image.sh staging $$(git describe --tags --always --dirty)

push-production:
	@echo "Pushing production image..."
	@./scripts/push_image.sh production $$(git describe --tags --always --dirty)

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
	@docker compose -f docker compose.test.yml up -d
	@sleep 2
	@echo "Test database ready on port 5433"

db-test-down:
	@echo "Stopping test database..."
	@docker compose -f docker compose.test.yml down

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

# Deployment commands
deploy-staging:
	@echo "Deploying to staging..."
	@./scripts/deploy.sh staging $$(git describe --tags --always)

deploy-production:
	@echo "Deploying to production..."
	@./scripts/deploy.sh production $$(git describe --tags --always)

rollback-staging:
	@echo "Rolling back staging..."
	@./scripts/rollback.sh staging

rollback-production:
	@echo "Rolling back production..."
	@./scripts/rollback.sh production

health-check-staging:
	@./scripts/health-check.sh staging

health-check-prod:
	@./scripts/health-check.sh production


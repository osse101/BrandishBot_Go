.PHONY: help migrate-up migrate-down migrate-status migrate-create test build run clean docker-build docker-up docker-down deploy-staging deploy-production rollback-staging rollback-production health-check-staging health-check-prod install-hooks reset-staging seed-staging validate-staging admin-install admin-dev admin-build admin-clean

# Tool paths
GOOSE   := go run github.com/pressly/goose/v3/cmd/goose
SWAG    := go run github.com/swaggo/swag/cmd/swag
LINT    := go run github.com/golangci/golangci-lint/cmd/golangci-lint
MOCKERY := go run github.com/vektra/mockery/v2
SQLC    := go run github.com/sqlc-dev/sqlc/cmd/sqlc

# Load environment variables from .env if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

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
	@echo "  make generate             - Generate sqlc code"
	@echo "  make install-hooks        - Install git hooks (pre-commit formatting)"
	@echo "  make setup                - Setup development environment (deps, docker, db, migrations)"
	@echo ""
	@echo "Benchmark Commands:"
	@echo "  make bench                - Run all benchmarks"
	@echo "  make bench-hot            - Run hot path benchmarks only"
	@echo "  make bench-save           - Run benchmarks and save timestamped results"
	@echo "  make bench-baseline       - Set current results as baseline"
	@echo "  make bench-compare        - Compare current benchmarks to baseline"
	@echo "  make bench-profile        - Profile hot paths (CPU + memory)"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-up            - Start services with Docker Compose"
	@echo "  make docker-down          - Stop services"
	@echo "  make docker-build         - Rebuild Docker images (no cache, slower but clean)"
	@echo "  make docker-build-fast    - Build Docker images (with cache, faster for dev)"
	@echo "  make docker-version       - Show version of local Docker image"
	@echo ""
	@echo "Monitoring Commands:"
	@echo "  make monitoring-up        - Start Prometheus + Grafana"
	@echo "  make monitoring-down      - Stop monitoring stack"
	@echo "  make monitoring-restart   - Restart monitoring services"
	@echo "  make monitoring-logs      - View monitoring logs"
	@echo "  make monitoring-status    - Check monitoring health"
	@echo "  make prometheus-reload    - Hot reload Prometheus config"
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
	@echo "  make reset-staging        - Full staging reset (down + volume rm + up)"
	@echo "  make seed-staging         - Seed staging with test data"
	@echo "  make validate-staging     - Run validation tests against staging"
	@echo ""
	@echo "Audit & Security:"
	@echo "  make test-migrations      - Test migration up/down/idempotency"
	@echo "  make test-security        - Run security integration tests"
	@echo "  make check-deps           - Check for required dependencies"
	@echo "  make check-db             - Ensure Docker database is running"

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
	@mkdir -p logs
	@go test ./... -coverprofile=logs/coverage.out -covermode=atomic -race

unit:
	@echo "Running unit tests (fast)..."
	@go test -short ./...

watch:
	@echo "Watching for changes to run unit tests..."
	@if command -v entr > /dev/null; then \
		find . -name "*.go" | entr -c $(MAKE) unit; \
	else \
		echo "Error: 'entr' is not installed. Please install it to use this feature."; \
		exit 1; \
	fi

test-coverage:
	@go run ./cmd/devtool check-coverage --html logs/coverage.out 0

test-coverage-check:
	@go run ./cmd/devtool check-coverage logs/coverage.out 80

lint:
	@echo "Running linters..."
	@$(LINT) run ./...

lint-fix:
	@echo "Running linters with auto-fix..."
	@$(LINT) run --fix ./...

install-hooks:
	@echo "Installing git hooks..."
	@echo "#!/bin/sh" > .git/hooks/pre-commit
	@echo "go run ./cmd/devtool pre-commit" >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✓ Git hooks installed"

# Benchmark commands
.PHONY: bench bench-hot bench-save bench-baseline bench-compare bench-profile

bench:
	@go run ./cmd/devtool bench run

bench-hot:
	@go run ./cmd/devtool bench hot

bench-save:
	@go run ./cmd/devtool bench save

bench-baseline:
	@go run ./cmd/devtool bench baseline

bench-compare:
	@go run ./cmd/devtool bench compare

bench-profile:
	@go run ./cmd/devtool bench profile

# Build targets
build:
	@go run ./cmd/devtool build

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
setup:
	@echo "🚀 Starting environment setup..."
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
	fi
	@$(MAKE) check-deps
	@$(MAKE) docker-up
	@$(MAKE) check-db
	@$(MAKE) migrate-up
	@$(MAKE) generate
	@echo "✅ Setup complete!"

run:
	@echo "Starting BrandishBot from bin/app..."
	@./bin/app

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✓ Removed bin/ directory"

swagger:
	@echo "Generating Swagger documentation..."
	@$(SWAG) init -g cmd/app/main.go --output ./docs/swagger
	@echo "Swagger docs updated: docs/swagger/"

generate:
	@echo "Generating Swagger documentation..."
	@$(MAKE) swagger
	@echo "Generating sqlc code..."
	@$(SQLC) generate
	@echo "✓ sqlc code generated"
	@echo "Generating progression keys from config..."
	@go run ./cmd/gen-progression-keys -config configs/progression_tree.json -output internal/progression/keys.go
	@echo "✓ progression keys generated"
	@echo "Generating mocks..."
	@$(MOCKERY)
	@echo "✓ mocks generated"
	@go mod tidy

# Docker commands
docker-up:
	@echo "Starting Docker services..."
	@docker compose up -d

docker-down:
	@echo "Stopping Docker services..."
	@docker compose down

docker-build:
	@echo "Rebuilding Docker images (no cache)..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	BUILD_TIME=$$(date -u '+%Y-%m-%d_%H:%M'); \
	GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	echo "Building with VERSION=$$VERSION BUILD_TIME=$$BUILD_TIME GIT_COMMIT=$$GIT_COMMIT"; \
	VERSION=$$VERSION BUILD_TIME=$$BUILD_TIME GIT_COMMIT=$$GIT_COMMIT docker compose build --no-cache
	@echo "Docker images rebuilt successfully"

docker-build-fast:
	@echo "Building Docker images (with cache, faster)..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	BUILD_TIME=$$(date -u '+%Y-%m-%d_%H:%M'); \
	GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	VERSION=$$VERSION BUILD_TIME=$$BUILD_TIME GIT_COMMIT=$$GIT_COMMIT DOCKER_BUILDKIT=1 docker compose build
	@echo "Docker images built successfully"

docker-version:
	@echo "Local Docker image version info:"
	@echo "================================"
	@docker inspect brandishbot:dev --format='Version:    {{index .Config.Labels "org.opencontainers.image.version"}}' 2>/dev/null || echo "Image not found. Run 'make docker-build' first."
	@docker inspect brandishbot:dev --format='Git Commit: {{index .Config.Labels "org.opencontainers.image.revision"}}' 2>/dev/null
	@docker inspect brandishbot:dev --format='Built:      {{index .Config.Labels "org.opencontainers.image.created"}}' 2>/dev/null
	@echo "================================"

docker-logs:
	@docker compose logs -f

push-staging:
	@echo "Pushing staging image..."
	@go run ./cmd/devtool push staging $$(git describe --tags --always --dirty)

push-production:
	@echo "Pushing production image..."
	@go run ./cmd/devtool push production $$(git describe --tags --always --dirty)

# Test database commands
test-integration:
	@echo "Running integration tests..."
	@go test ./internal/database/postgres -v -timeout=60s

test-staging:
	@echo "Running staging integration tests..."
	@echo "Target: $${API_URL:-http://localhost:8081}"
	@API_URL=$${API_URL:-http://localhost:8081} go test -tags=staging -v ./tests/staging

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

# Deployment commands
deploy-staging:
	@echo "Deploying to staging..."
	@go run ./cmd/devtool deploy staging $$(git describe --tags --always)

deploy-production:
	@echo "Deploying to production..."
	@go run ./cmd/devtool deploy production $$(git describe --tags --always)

rollback-staging:
	@echo "Rolling back staging..."
	@go run ./cmd/devtool rollback staging

rollback-production:
	@echo "Rolling back production..."
	@go run ./cmd/devtool rollback production

health-check-staging:
	@go run ./cmd/devtool health-check staging

health-check-prod:
	@go run ./cmd/devtool health-check production

# Staging reset and validation targets
reset-staging:
	@echo "🔄 Resetting staging environment..."
	@echo "Stopping staging containers..."
	@docker compose -f docker-compose.staging.yml down -v
	@echo "Starting fresh staging environment..."
	@docker compose -f docker-compose.staging.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@$(MAKE) health-check-staging
	@echo "✅ Staging reset complete"

seed-staging:
	@echo "🌱 Seeding staging database..."
	@docker compose -f docker-compose.staging.yml exec -T db psql -U $(DB_USER) -d $(DB_NAME) < scripts/setup_test_user.sql || echo "Note: Seed script may not exist"
	@echo "✅ Staging seeded (if seed scripts exist)"

validate-staging:
	@echo "🔍 Validating staging environment..."
	@API_URL=http://localhost:8081 $(MAKE) test-staging
	@echo "✅ Staging validation complete"

# Audit & Security targets
test-migrations:
	@go run ./cmd/devtool test-migrations

test-security:
	@go run ./cmd/devtool test-security

check-deps:
	@go run ./cmd/devtool check-deps

check-db:
	@go run ./cmd/devtool check-db


# Admin dashboard commands
admin-install:
	@echo "Installing admin dashboard dependencies..."
	@cd web/admin && npm ci
	@echo "✓ Admin dependencies installed"

admin-dev:
	@echo "Starting admin dashboard dev server..."
	@cd web/admin && npm run dev

admin-build:
	@echo "Building admin dashboard..."
	@cd web/admin && npm ci && npm run build
	@rm -rf internal/admin/dist
	@cp -r web/admin/dist internal/admin/dist
	@echo "✓ Admin dashboard built and copied to internal/admin/dist"

admin-clean:
	@echo "Cleaning admin dashboard artifacts..."
	@rm -rf web/admin/dist web/admin/node_modules
	@echo "✓ Admin artifacts cleaned"

# Mock generation
.PHONY: mocks clean-mocks
mocks:
	@echo "Generating mocks..."
	@$(MOCKERY)
	@echo "Mocks generated in mocks/ directory"

clean-mocks:
	@echo "Removing generated mocks..."
	@rm -rf mocks/
	@echo "✓ Removed mocks/ directory"

# Monitoring Commands
.PHONY: monitoring-up monitoring-down monitoring-restart monitoring-logs monitoring-status prometheus-reload

monitoring-up:
	@echo "Starting monitoring stack (Prometheus + Grafana)..."
	@docker-compose up -d prometheus grafana
	@echo "✓ Monitoring stack started"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Grafana:    http://localhost:3000 (admin/admin)"

monitoring-down:
	@echo "Stopping monitoring stack..."
	@docker-compose stop prometheus grafana
	@echo "✓ Monitoring stack stopped"

monitoring-restart:
	@echo "Restarting monitoring stack..."
	@docker-compose restart prometheus grafana
	@echo "✓ Monitoring stack restarted"

monitoring-logs:
	@echo "Showing monitoring logs (Ctrl+C to exit)..."
	@docker-compose logs -f prometheus grafana

monitoring-status:
	@echo "Checking monitoring stack health..."
	@echo -n "Prometheus: "
	@curl -s http://localhost:9090/-/healthy || echo "NOT HEALTHY"
	@echo -n "Grafana:    "
	@curl -s http://localhost:3000/api/health | grep -q '"database":"ok"' && echo "Healthy" || echo "NOT HEALTHY"

prometheus-reload:
	@echo "Reloading Prometheus configuration..."
	@curl -X POST http://localhost:9090/-/reload
	@echo "✓ Prometheus configuration reloaded"

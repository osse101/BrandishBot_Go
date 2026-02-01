.PHONY: help migrate-up migrate-down migrate-status migrate-create test build run clean docker-build docker-up docker-down deploy-staging deploy-production rollback-staging rollback-production health-check-staging health-check-prod install-hooks

# Tool paths
GOOSE   := go run github.com/pressly/goose/v3/cmd/goose
SWAG    := go run github.com/swaggo/swag/cmd/swag
LINT    := go run github.com/golangci/golangci-lint/cmd/golangci-lint
MOCKERY := go run github.com/vektra/mockery/v2
SQLC    := go run github.com/sqlc-dev/sqlc/cmd/sqlc

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
	@./scripts/unit_tests.sh

watch:
	@echo "Watching for changes to run unit tests..."
	@if command -v entr > /dev/null; then \
		find . -name "*.go" | entr -c ./scripts/unit_tests.sh; \
	else \
		echo "Error: 'entr' is not installed. Please install it to use this feature."; \
		exit 1; \
	fi

test-coverage:
	@echo "Generating coverage report..."
	@if [ ! -f logs/coverage.out ]; then \
		echo "Coverage profile not found. Running tests..."; \
		$(MAKE) test; \
	fi
	@go tool cover -html=logs/coverage.out -o logs/coverage.html
	@echo "Coverage report generated: logs/coverage.html"
	@./scripts/check_coverage.sh logs/coverage.out 0

test-coverage-check:
	@echo "Checking coverage threshold (80%)..."
	@if [ ! -f logs/coverage.out ]; then \
		echo "Coverage profile not found. Running tests..."; \
		$(MAKE) test; \
	fi
	@./scripts/check_coverage.sh logs/coverage.out 80

lint:
	@echo "Running linters..."
	@$(LINT) run ./...

lint-fix:
	@echo "Running linters with auto-fix..."
	@$(LINT) run --fix ./...

install-hooks:
	@echo "Installing git hooks..."
	@chmod +x scripts/pre-commit.sh
	@ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
	@echo "✓ Git hooks installed"

# Benchmark commands
.PHONY: bench bench-hot bench-save bench-baseline bench-compare bench-profile

bench:
	@echo "Running all benchmarks..."
	@go test -bench=. -benchmem -benchtime=2s ./...

bench-hot:
	@echo "Running hot path benchmarks..."
	@echo "  → Handler: HandleMessageHandler"
	@go test -bench=BenchmarkHandler_HandleMessage -benchmem -benchtime=2s ./internal/handler 2>/dev/null || echo "    (benchmark not yet implemented)"
	@echo "  → Service: HandleIncomingMessage"
	@go test -bench=BenchmarkService_HandleIncomingMessage -benchmem -benchtime=2s ./internal/user 2>/dev/null || echo "    (benchmark not yet implemented)"
	@echo "  → Service: AddItem"
	@go test -bench=BenchmarkService_AddItem -benchmem -benchtime=2s ./internal/user 2>/dev/null || echo "    (benchmark not yet implemented)"
	@echo "  → Utils: Inventory operations (existing)"
	@go test -bench=. -benchmem -benchtime=2s ./internal/utils

bench-save:
	@echo "Running benchmarks and saving results..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=2s ./... 2>&1 | tee benchmarks/results/$$(date +%Y%m%d-%H%M%S).txt
	@echo "✓ Results saved to benchmarks/results/"

bench-baseline:
	@echo "Setting benchmark baseline..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=2s ./... 2>&1 | tee benchmarks/results/baseline.txt
	@echo "✓ Baseline set: benchmarks/results/baseline.txt"

bench-compare:
	@if [ ! -f benchmarks/results/baseline.txt ]; then \
		echo "❌ Error: No baseline found. Run 'make bench-baseline' first."; \
		exit 1; \
	fi
	@echo "Running benchmarks and comparing to baseline..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=2s ./... > benchmarks/results/current.txt 2>&1 || true
	@if command -v benchstat > /dev/null 2>&1; then \
		benchstat benchmarks/results/baseline.txt benchmarks/results/current.txt; \
	else \
		echo ""; \
		echo "⚠️  benchstat not installed. Install with:"; \
		echo "   go install golang.org/x/perf/cmd/benchstat@latest"; \
		echo ""; \
		echo "Showing raw comparison:"; \
		echo "======================"; \
		echo "BASELINE:"; \
		grep "^Benchmark" benchmarks/results/baseline.txt | head -5; \
		echo ""; \
		echo "CURRENT:"; \
		grep "^Benchmark" benchmarks/results/current.txt | head -5; \
	fi

bench-profile:
	@echo "Profiling hot paths..."
	@mkdir -p benchmarks/profiles
	@echo "  → CPU profile (if benchmark exists)..."
	@go test -bench=BenchmarkHandler_HandleMessage -cpuprofile=benchmarks/profiles/cpu.prof ./internal/handler 2>/dev/null || \
		go test -bench=BenchmarkAddItems -cpuprofile=benchmarks/profiles/cpu.prof ./internal/utils
	@echo "  → Memory profile (if benchmark exists)..."
	@go test -bench=BenchmarkHandler_HandleMessage -memprofile=benchmarks/profiles/mem.prof -benchmem ./internal/handler 2>/dev/null || \
		go test -bench=BenchmarkAddItems -memprofile=benchmarks/profiles/mem.prof -benchmem ./internal/utils
	@echo "✓ Profiles saved to benchmarks/profiles/"
	@echo ""
	@echo "View CPU profile with:"
	@echo "  go tool pprof -http=:8080 benchmarks/profiles/cpu.prof"
	@echo "View memory profile with:"
	@echo "  go tool pprof -http=:8080 benchmarks/profiles/mem.prof"

# Build targets
build:
	@echo "Building all binaries to bin/..."
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	BUILD_TIME=$$(date -u '+%Y-%m-%d_%H:%M'); \
	GIT_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	LDFLAGS="-X github.com/osse101/BrandishBot_Go/internal/handler.Version=$$VERSION \
	         -X github.com/osse101/BrandishBot_Go/internal/handler.BuildTime=$$BUILD_TIME \
	         -X github.com/osse101/BrandishBot_Go/internal/handler.GitCommit=$$GIT_COMMIT"; \
	go build -ldflags "$$LDFLAGS" -o bin/app ./cmd/app; \
	go build -ldflags "$$LDFLAGS" -o bin/discord_bot ./cmd/discord
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
	@$(SWAG) init -g cmd/app/main.go --output ./docs/swagger
	@echo "Swagger docs updated: docs/swagger/"

generate:
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
	@./scripts/push_image.sh staging $$(git describe --tags --always --dirty)

push-production:
	@echo "Pushing production image..."
	@./scripts/push_image.sh production $$(git describe --tags --always --dirty)

# Test database commands
test-integration:
	@echo "Running integration tests..."
	@go test ./internal/database/postgres -v -timeout=60s

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

# Audit & Security targets
test-migrations:
	@chmod +x scripts/test_migrations.sh
	@./scripts/test_migrations.sh

test-security:
	@chmod +x scripts/test_security.sh
	@./scripts/test_security.sh

check-deps:
	@chmod +x scripts/check_deps.sh
	@./scripts/check_deps.sh

check-db:
	@chmod +x scripts/check_db.sh
	@./scripts/check_db.sh


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

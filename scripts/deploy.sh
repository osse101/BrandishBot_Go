#!/bin/bash
set -e

# BrandishBot Production Deployment Script
# Usage: ./scripts/deploy.sh <environment> <version>
# Example: ./scripts/deploy.sh staging v1.2.0-rc1

ENVIRONMENT="${1}"
VERSION="${2}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate arguments
if [[ -z "$ENVIRONMENT" ]] || [[ -z "$VERSION" ]]; then
    log_error "Usage: $0 <environment> <version>"
    log_info "Example: $0 staging v1.2.0-rc1"
    log_info "         $0 production v1.2.0"
    exit 1
fi

# Validate environment
if [[ "$ENVIRONMENT" != "staging" ]] && [[ "$ENVIRONMENT" != "production" ]]; then
    log_error "Environment must be 'staging' or 'production'"
    exit 1
fi

# Set compose file based on environment
if [[ "$ENVIRONMENT" == "staging" ]]; then
    COMPOSE_FILE="docker-compose.staging.yml"
elif [[ "$ENVIRONMENT" == "production" ]]; then
    COMPOSE_FILE="docker-compose.production.yml"
fi

log_info "=== BrandishBot Deployment ==="
log_info "Environment: $ENVIRONMENT"
log_info "Version: $VERSION"
log_info "Compose File: $COMPOSE_FILE"
echo ""

# Production confirmation
if [[ "$ENVIRONMENT" == "production" ]]; then
    log_warn "You are about to deploy to PRODUCTION"
    echo -n "Type 'yes' to continue: "
    read -r CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        log_error "Deployment cancelled"
        exit 1
    fi
fi

cd "$PROJECT_DIR"

# Load environment variables from .env if it exists
if [[ -f ".env" ]]; then
    log_info "Loading environment variables from .env"
    set -a  # Export all variables
    source .env
    set +a  # Stop auto-export
fi

# Step 1: Pre-deployment health check
log_info "Step 1/7: Pre-deployment health check"
if [[ -f "$SCRIPT_DIR/health-check.sh" ]]; then
    if ! bash "$SCRIPT_DIR/health-check.sh" "$ENVIRONMENT" 2>/dev/null; then
        log_warn "Pre-deployment health check failed (service may not be running yet)"
    else
        log_info "Current deployment is healthy"
    fi
else
    log_warn "health-check.sh not found, skipping pre-deployment health check"
fi

# Step 2: Database backup
log_info "Step 2/7: Creating database backup"
mkdir -p "$PROJECT_DIR/backups"
BACKUP_FILE="backups/backup_${ENVIRONMENT}_$(date +%Y%m%d_%H%M%S).sql"
if docker compose -f "$COMPOSE_FILE" ps db | grep -q "Up"; then
    DB_CONTAINER=$(docker compose -f "$COMPOSE_FILE" ps -q db)
    if [[ -n "$DB_CONTAINER" ]]; then
        docker exec "$DB_CONTAINER" pg_dump -U ${DB_USER:-brandishbot} -d ${DB_NAME:-brandishbot} > "$BACKUP_FILE" 2>/dev/null || true
        if [[ -f "$BACKUP_FILE" ]]; then
            log_info "Database backup created: $BACKUP_FILE"
        else
            log_warn "Failed to create database backup (database may not exist yet)"
        fi
    fi
else
    log_warn "Database not running, skipping backup"
fi

# Step 3: Build Docker image with version tag
log_info "Step 3/7: Building Docker image"
docker build \
    --build-arg VERSION="$VERSION" \
    -t "brandishbot:$VERSION" \
    -t "brandishbot:latest-$ENVIRONMENT" \
    -f Dockerfile \
    . || {
    log_error "Docker build failed"
    exit 1
}
log_info "Docker image built: brandishbot:$VERSION"

# Step 4: Deploy new containers
log_info "Step 4/7: Deploying new containers"
export DOCKER_IMAGE_TAG="$VERSION"
docker compose -f "$COMPOSE_FILE" up -d app discord || {
    log_error "Deployment failed"
    log_info "Attempting rollback..."
    docker compose -f "$COMPOSE_FILE" up -d --no-deps app discord
    exit 1
}
log_info "Containers deployed"

# Step 5: Wait for health checks
log_info "Step 5/7: Waiting for health checks (max 60 seconds)"
MAX_ATTEMPTS=30
ATTEMPT=0
HEALTHY=false

while [[ $ATTEMPT -lt $MAX_ATTEMPTS ]]; do
    ATTEMPT=$((ATTEMPT + 1))
    sleep 2
    
    if bash "$SCRIPT_DIR/health-check.sh" "$ENVIRONMENT" 2>/dev/null; then
        HEALTHY=true
        break
    fi
    
    echo -n "."
done
echo ""

if [[ "$HEALTHY" == "false" ]]; then
    log_error "Health check failed after 60 seconds"
    log_error "Deployment failed - manual intervention required"
    log_info "Check logs: docker compose -f $COMPOSE_FILE logs app"
    exit 1
fi

log_info "Health checks passed"

# Step 6: Run smoke tests
log_info "Step 6/7: Running smoke tests"
PORT=8080
if [[ "$ENVIRONMENT" == "staging" ]]; then
    PORT=8081
fi

# Test /healthz endpoint
if curl -sf "http://localhost:$PORT/healthz" > /dev/null; then
    log_info "✓ /healthz endpoint responding"
else
    log_error "✗ /healthz endpoint failed"
    exit 1
fi

# Test /progression/tree endpoint
if curl -sf "http://localhost:$PORT/progression/tree" > /dev/null; then
    log_info "✓ /progression/tree endpoint responding"
else
    log_warn "✗ /progression/tree endpoint failed (may be expected for fresh deployment)"
fi

# Step 7: Cleanup old images (keep last 5)
log_info "Step 7/7: Cleaning up old Docker images"
docker images "brandishbot" --format "{{.Tag}}" | grep -v "latest" | tail -n +6 | xargs -r -I {} docker rmi "brandishbot:{}" 2>/dev/null || true

echo ""
log_info "=== Deployment Complete ==="
log_info "Environment: $ENVIRONMENT"
log_info "Version: $VERSION"
log_info "Status: SUCCESS"
log_info ""
log_info "Next steps:"
log_info "  - Check logs: docker compose -f $COMPOSE_FILE logs -f app"
log_info "  - Run staging tests: STAGING_URL=http://localhost:$PORT make test-staging"
if [[ "$ENVIRONMENT" == "production" ]]; then
    log_info "  - Monitor for errors"
    log_info "  - If issues arise, rollback: ./scripts/rollback.sh production"
fi
echo ""

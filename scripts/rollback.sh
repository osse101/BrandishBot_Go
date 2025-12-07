#!/bin/bash
set -e

# BrandishBot Rollback Script
# Usage: ./scripts/rollback.sh <environment> [version]
# Example: ./scripts/rollback.sh production v1.1.0

ENVIRONMENT="${1}"
TARGET_VERSION="${2}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate environment
if [[ -z "$ENVIRONMENT" ]]; then
    log_error "Usage: $0 <environment> [version]"
    log_info "Example: $0 production v1.1.0"
    log_info "         $0 staging"
    exit 1
fi

if [[ "$ENVIRONMENT" != "staging" ]] && [[ "$ENVIRONMENT" != "production" ]]; then
    log_error "Environment must be 'staging' or 'production'"
    exit 1
fi

# Set compose file based on environment
if [[ "$ENVIRONMENT" == "staging" ]]; then
    COMPOSE_FILE="docker compose.staging.yml"
elif [[ "$ENVIRONMENT" == "production" ]]; then
    COMPOSE_FILE="docker compose.production.yml"
fi

cd "$PROJECT_DIR"

log_info "=== BrandishBot Rollback ==="
log_info "Environment: $ENVIRONMENT"
echo ""

# If no version specified, list available versions
if [[ -z "$TARGET_VERSION" ]]; then
    log_info "Available Docker images (last 10):"
    docker images "brandishbot" --format "table {{.Tag}}\t{{.CreatedAt}}" | head -n 11
    echo ""
    echo -n "Enter version to rollback to (or 'cancel' to abort): "
    read -r TARGET_VERSION
    
    if [[ "$TARGET_VERSION" == "cancel" ]] || [[ -z "$TARGET_VERSION" ]]; then
        log_error "Rollback cancelled"
        exit 1
    fi
fi

# Verify the target image exists
if ! docker images "brandishbot:$TARGET_VERSION" --format "{{.Tag}}" | grep -q "^$TARGET_VERSION$"; then
    log_error "Docker image brandishbot:$TARGET_VERSION not found"
    log_info "Available images:"
    docker images "brandishbot" --format "table {{.Tag}}\t{{.CreatedAt}}"
    exit 1
fi

# Production confirmation
if [[ "$ENVIRONMENT" == "production" ]]; then
    log_warn "You are about to rollback PRODUCTION to version $TARGET_VERSION"
    echo -n "Type 'yes' to continue: "
    read -r CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        log_error "Rollback cancelled"
        exit 1
    fi
fi

# Step 1: Stop current containers
log_info "Step 1/4: Stopping current containers"
docker compose -f "$COMPOSE_FILE" stop app discord

# Step 2: Rollback to target version
log_info "Step 2/4: Rolling back to version $TARGET_VERSION"
export DOCKER_IMAGE_TAG="$TARGET_VERSION"
docker compose -f "$COMPOSE_FILE" up -d --no-deps app discord

# Step 3: Wait for health checks
log_info "Step 3/4: Waiting for health checks (max 60 seconds)"
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
    log_error "Health check failed after rollback"
    log_error "Manual intervention required"
    log_info "Check logs: docker compose -f $COMPOSE_FILE logs app"
    exit 1
fi

log_info "Health checks passed"

# Step 4: Database rollback option
log_info "Step 4/4: Database rollback"
echo ""
log_warn "Do you need to restore the database from a backup?"
log_info "Available backups:"
ls -lth backups/backup_${ENVIRONMENT}_*.sql 2>/dev/null | head -n 5 || log_info "No backups found"
echo ""
echo -n "Enter backup filename to restore (or press Enter to skip): "
read -r BACKUP_FILE

if [[ -n "$BACKUP_FILE" ]] && [[ -f "$BACKUP_FILE" ]]; then
    log_warn "This will overwrite the current database!"
    echo -n "Type 'yes' to restore database from $BACKUP_FILE: "
    read -r CONFIRM_DB
    
    if [[ "$CONFIRM_DB" == "yes" ]]; then
        log_info "Restoring database from $BACKUP_FILE"
        DB_CONTAINER=$(docker compose -f "$COMPOSE_FILE" ps -q db)
        docker exec -i "$DB_CONTAINER" psql -U ${DB_USER:-brandishbot} -d ${DB_NAME:-brandishbot} < "$BACKUP_FILE"
        log_info "Database restored"
    else
        log_info "Database restore cancelled"
    fi
else
    log_info "Skipping database restore"
fi

echo ""
log_info "=== Rollback Complete ==="
log_info "Environment: $ENVIRONMENT"
log_info "Version: $TARGET_VERSION"
log_info "Status: SUCCESS"
echo ""

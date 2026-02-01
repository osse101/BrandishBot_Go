#!/usr/bin/env bash
set -euo pipefail

# scripts/deploy_remote.sh
# Deployment script for remote servers (pulls images instead of building)

ENVIRONMENT="${1}"
TAG="${2}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ENVIRONMENT="${1}"
TAG="${2}"
ACTION="${3:-deploy}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [[ -z "$ENVIRONMENT" ]]; then
    log_error "Usage: $0 <environment> [tag] [action]"
    log_info "Actions:"
    log_info "  deploy  - Pull, restart, and prune (Default)"
    log_info "  start   - Just start services"
    log_info "  stop    - Stop services (teardown)"
    log_info "  pull    - Just pull images"
    log_info "Example: $0 staging v1.0.0"
    log_info "Example: $0 production latest stop"
    exit 1
fi

# Load .env if present
if [[ -f .env ]]; then
    export $(grep -v '^#' .env | xargs)
fi

# Determine Configuration
if [[ "$ENVIRONMENT" == "staging" ]]; then
    COMPOSE_FILE="docker-compose.staging.yml"
    DEFAULT_TAG="latest-staging"
elif [[ "$ENVIRONMENT" == "production" ]]; then
    COMPOSE_FILE="docker-compose.production.yml"
    DEFAULT_TAG="latest-production"
else
    log_error "Environment must be 'staging' or 'production'"
    exit 1
fi

export DOCKER_IMAGE_TAG="${TAG:-$DEFAULT_TAG}"

log_info "=== BrandishBot Remote Deployment ==="
log_info "Environment: $ENVIRONMENT"
log_info "Image Tag: $DOCKER_IMAGE_TAG"
log_info "User: ${DOCKER_USER:-brandishbot}"

# Helper: startup command
startup() {
    log_info "Starting services..."
    # Ensure database is up first
    docker compose -f "$COMPOSE_FILE" up -d db
    # Wait for DB (optional, but good practice if not using healthchecks strict dependency)
    sleep 2
    # Start app and discord
    docker compose -f "$COMPOSE_FILE" up -d app discord
    log_info "Services started."
}

# Helper: teardown command
teardown() {
    log_info "Stopping services..."
    docker compose -f "$COMPOSE_FILE" down
    log_info "Services stopped."
}

# Helper: deploy command (default)
deploy() {
    # 1. Login (interactive if needed, or skip if already logged in)
    if ! docker system info | grep -q "Username"; then
        log_warn "Not logged in. Attempting docker login..."
        docker login
    fi

    # 2. Pull images
    log_info "Pulling images..."
    docker compose -f "$COMPOSE_FILE" pull app discord
    
    # 3. Restart services (rolling update style)
    startup
    
    # 4. Prune old images
    log_info "Cleaning up old images..."
    docker image prune -f
    
    	# 5. Check health
	if [[ -f "./scripts/health-check.sh" ]]; then
		log_info "Running health checks..."
		./scripts/health-check.sh "$ENVIRONMENT"
	fi

	# 6. Announce Deployment (Release Notes)
	if ! command -v jq &> /dev/null; then
		log_warn "jq not found. Skipping release notes announcement."
		return
	fi

	log_info "Generating release notes..."

	# Determine range (e.g., from last tag to current HEAD)
	# Assuming tags are v*
	LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
	
	if [[ -z "$LAST_TAG" ]]; then
		RANGE="HEAD~5..HEAD" # Fallback if no tags
		TITLE="Deployment Update"
	else
		RANGE="$LAST_TAG..HEAD"
		TITLE="Deployment Update ($LAST_TAG -> HEAD)"
	fi

	# Generate simpler log
	NOTES=$(git log --pretty=format:"• %s (%an)" "$RANGE" | head -n 20)
	
	if [[ -z "$NOTES" ]]; then
		NOTES="No new commits in this deployment."
	fi
	
	JSON_PAYLOAD=$(jq -n \
		--arg title "$TITLE" \
		--arg description "$NOTES" \
		--argjson color 65280 \
		'{title: $title, description: $description, color: $color}')

	# Determine Discord Bot Host
	# Inside docker network it might be 'discord', but this script runs on the host/runner.
	# If running on the same machine as docker, we need to know the port mapping.
	# Standard dev setup maps 8082:8082
	DISCORD_URL="http://localhost:8082/admin/announce"

	# If we are in a container/remote, we might need a different address, 
	# but assuming this script runs where docker CLI is available and ports are mapped.
	
	log_info "Sending release notes to Discord..."
	response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$DISCORD_URL" \
		-H "Content-Type: application/json" \
		-d "$JSON_PAYLOAD")

	if [[ "$response" == "200" ]]; then
		log_info "Release notes sent successfully."
	else
		log_warn "Failed to send release notes (Status: $response). Is the bot running and port 8082 exposed?"
	fi
}

# Main Execution Switch
case "$ACTION" in
    deploy)
        deploy
        ;;
    start)
        startup
        ;;
    stop|teardown)
        teardown
        ;;
    pull)
        log_info "Pulling images..."
        # 1. Login check
        if ! docker system info | grep -q "Username"; then
             log_warn "Not logged in. Attempting docker login..."
             docker login
        fi
        docker compose -f "$COMPOSE_FILE" pull app discord
        ;;
    *)
        log_error "Unknown action: $ACTION"
        log_info "Available actions: deploy, start, stop"
        exit 1
        ;;
esac

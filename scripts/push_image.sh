#!/bin/bash
set -e

# scripts/push_image.sh
# Builds and pushes Docker images to the configured registry

ENVIRONMENT="${1}"
VERSION="${2}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check arguments
if [[ -z "$ENVIRONMENT" ]] || [[ -z "$VERSION" ]]; then
    log_error "Usage: $0 <environment> <version>"
    exit 1
fi

# Load environment variables
if [[ -f "$PROJECT_DIR/.env" ]]; then
    source "$PROJECT_DIR/.env"
else
    log_warn ".env file not found"
fi

# Validate Registry Config
if [[ -z "$DOCKER_USER" ]]; then
    log_error "DOCKER_USER is not set in .env"
    log_info "Please set DOCKER_USER to your Docker Hub username or registry URL"
    exit 1
fi

IMAGE_NAME="${DOCKER_IMAGE_NAME:-brandishbot}"
FULL_IMAGE_NAME="$DOCKER_USER/$IMAGE_NAME"

log_info "=== Docker Image Push ==="
log_info "Environment: $ENVIRONMENT"
log_info "Version: $VERSION"
log_info "Image: $FULL_IMAGE_NAME"

# Check Docker Login
if ! docker system info | grep -q "Username"; then
    log_warn "Not logged into Docker Hub/Registry. Attempting login..."
    docker login
fi

cd "$PROJECT_DIR"

# Build Image
log_info "Building image..."
docker build \
    --build-arg VERSION="$VERSION" \
    -t "$FULL_IMAGE_NAME:$VERSION" \
    -t "$FULL_IMAGE_NAME:latest-$ENVIRONMENT" \
    -t "brandishbot:$VERSION" \
    -f Dockerfile \
    .

# Push Tags
log_info "Pushing tags to registry..."
docker push "$FULL_IMAGE_NAME:$VERSION"
docker push "$FULL_IMAGE_NAME:latest-$ENVIRONMENT"

log_info "✅ Successfully pushed:"
log_info "  - $FULL_IMAGE_NAME:$VERSION"
log_info "  - $FULL_IMAGE_NAME:latest-$ENVIRONMENT"

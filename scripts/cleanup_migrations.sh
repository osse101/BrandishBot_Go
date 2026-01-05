#!/bin/bash
# Clean up timestamp-based migrations from the goose_db_version table
# This resolves conflicts between timestamp and numbered migrations by removing
# the timestamp entries (version_id > 10000000) from the tracking table.
#
# Usage: ./scripts/cleanup_migrations.sh [container_name]

set -e

echo "================================================"
echo "Migration Cleanup Script"
echo "================================================"

# Load environment variables
if [ -f .env ]; then
    source .env
fi

# Determine container name
DB_CONTAINER="${1:-brandishbot_go-db-1}"

echo "Targeting DB Container: $DB_CONTAINER"

# Check if docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Error: docker command not found."
    exit 1
fi

# Verify container is running
if ! docker ps | grep -q "$DB_CONTAINER"; then
    echo "❌ Error: Container '$DB_CONTAINER' not found or not running."
    echo "Listing running containers:"
    docker ps --format "table {{.Names}}\t{{.Status}}"
    echo ""
    echo "Usage: ./scripts/cleanup_migrations.sh [container_name]"
    exit 1
fi

echo "Cleaning up timestamp migrations (ID > 10000000)..."

# Execute SQL via docker exec
# We expect DB_USER and DB_NAME to be set in .env, or use defaults
DB_USER="${DB_USER:-dev}"
DB_NAME="${DB_NAME:-app}"

echo "Using Database: $DB_NAME (User: $DB_USER)"

DELETE_CMD="DELETE FROM goose_db_version WHERE version_id > 10000000;"
SELECT_CMD="SELECT version_id, is_applied, tstamp FROM goose_db_version ORDER BY version_id;"

echo "Executing DELETE..."
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "$DELETE_CMD"

echo "Current Migration Status:"
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "$SELECT_CMD"

echo "================================================"
echo "✅ Cleanup complete."
echo "================================================"

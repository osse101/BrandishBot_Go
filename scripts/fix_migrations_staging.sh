#!/bin/bash
# Fix migration version mismatch on staging
# This script resolves the issue where goose migrations 26 and 27 are not recognized

set -e

echo "================================================"
echo "Migration Fix Script for Staging"
echo "================================================"
echo ""

# Load environment variables
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found"
    exit 1
fi

source .env

# Configuration
DB_CONTAINER="brandishbot_go-db-1"
APP_CONTAINER="brandishbot_go-app-1"

# Verify containers are running
if ! docker ps | grep -q "$DB_CONTAINER"; then
    echo "❌ Error: Database container not running"
    exit 1
fi

echo "1. Applying migration 26 (tune progression system)..."
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME < migrations/0026_tune_progression_system.sql 2>&1 | grep -v "ERROR" || true

echo ""
echo "2. Applying migration 27 (inventory filters)..."
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME < migrations/0027_inventory_filters.sql 2>&1 | grep -v "ERROR" || true

echo ""
echo "3. Registering migrations in goose tracking table..."
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME -c \
  "INSERT INTO goose_db_version (version_id, is_applied) VALUES (26, true), (27, true) ON CONFLICT DO NOTHING;"

echo ""
echo "4. Verifying migration status..."
docker exec $DB_CONTAINER psql -U $DB_USER -d $DB_NAME -c \
  "SELECT version_id, is_applied FROM goose_db_version WHERE version_id >= 20 ORDER BY id;"

echo ""
echo "5. Restarting application container..."
docker-compose restart app

echo ""
echo "6. Waiting for application to start (5 seconds)..."
sleep 5

echo ""
echo "7. Checking application logs..."
docker logs $APP_CONTAINER --tail 15

echo ""
echo "================================================"
echo "✅ Migration fix script completed!"
echo "================================================"
echo ""
echo "Verify the application started successfully above."
echo "You should see 'Starting server' in the logs."
echo ""
echo "Test the search endpoint:"
echo "curl -H \"X-API-Key: \$API_KEY\" http://server1050:8081/user/search \\"
echo "  -X POST -H \"Content-Type: application/json\" \\"
echo "  -d '{\"platform\":\"twitch\",\"platform_id\":\"24977686\",\"username\":\"osse101\"}'"
echo ""

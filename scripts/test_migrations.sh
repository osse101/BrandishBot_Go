#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing database migrations...${NC}"

# Configuration
export DATABASE_NAME="brandish_test_migrations"
export DATABASE_USER="${DB_USER:-postgres}"
export DATABASE_PASSWORD="${DB_PASSWORD:-postgres}"
export DATABASE_HOST="${DB_HOST:-localhost}"
export DATABASE_PORT="${DB_PORT:-5432}"
export DATABASE_URL="postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_NAME}?sslmode=disable"

# Check if goose is installed
if ! command -v goose &> /dev/null; then
    echo -e "${RED}Error: goose is not installed${NC}"
    echo "Install with: go install github.com/pressly/goose/v3/cmd/goose@latest"
    exit 1
fi

# Function to cleanup
cleanup() {
    echo -e "${YELLOW}Cleaning up test database...${NC}"
    psql -h "${DATABASE_HOST}" -p "${DATABASE_PORT}" -U "${DATABASE_USER}" -c "DROP DATABASE IF EXISTS ${DATABASE_NAME};" 2>/dev/null || true
}

# Setup cleanup trap
trap cleanup EXIT

# Create test database
echo -e "${YELLOW}Creating test database: ${DATABASE_NAME}${NC}"
psql -h "${DATABASE_HOST}" -p "${DATABASE_PORT}" -U "${DATABASE_USER}" -c "DROP DATABASE IF EXISTS ${DATABASE_NAME};" || true
psql -h "${DATABASE_HOST}" -p "${DATABASE_PORT}" -U "${DATABASE_USER}" -c "CREATE DATABASE ${DATABASE_NAME};"

# Test UP migrations
echo -e "${YELLOW}Testing UP migrations...${NC}"
goose -dir migrations postgres "${DATABASE_URL}" up
echo -e "${GREEN}✓ UP migrations completed${NC}"

# Get final migration count
UP_COUNT=$(goose -dir migrations postgres "${DATABASE_URL}" status | grep -c "Applied" || echo "0")
echo -e "Applied migrations: ${UP_COUNT}"

# Test DOWN migrations (all the way)
echo -e "${YELLOW}Testing DOWN migrations (all)...${NC}"
goose -dir migrations postgres "${DATABASE_URL}" down-to 0
echo -e "${GREEN}✓ DOWN migrations completed${NC}"

# Verify all migrations rolled back
REMAINING=$(goose -dir migrations postgres "${DATABASE_URL}" status | grep -c "Applied" || echo "0")
if [ "${REMAINING}" != "0" ]; then
    echo -e "${RED}✗ Error: ${REMAINING} migrations still applied after rollback${NC}"
    exit 1
fi
echo -e "${GREEN}✓ All migrations successfully rolled back${NC}"

# Test UP migrations again (idempotency)
echo -e "${YELLOW}Testing UP migrations again (idempotency)...${NC}"
goose -dir migrations postgres "${DATABASE_URL}" up
echo -e "${GREEN}✓ UP migrations completed again${NC}"

# Verify same count
FINAL_COUNT=$(goose -dir migrations postgres "${DATABASE_URL}" status | grep -c "Applied" || echo "0")
if [ "${UP_COUNT}" != "${FINAL_COUNT}" ]; then
    echo -e "${RED}✗ Error: Migration count mismatch (${UP_COUNT} vs ${FINAL_COUNT})${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All migration tests passed!${NC}"
echo -e "  - ${UP_COUNT} migrations tested"
echo -e "  - UP migrations: ✓"
echo -e "  - DOWN migrations: ✓"
echo -e "  - Idempotency: ✓"

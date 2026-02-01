#!/bin/bash
# scripts/init_dev.sh
# Initializes the development environment: checks deps, creates .env, starts DB, runs migrations.

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Initializing development environment...${NC}"

# 1. Check dependencies
echo "Checking dependencies..."
if [ -f "./scripts/check_deps.sh" ]; then
    ./scripts/check_deps.sh
else
    echo -e "${YELLOW}Warning: scripts/check_deps.sh not found. Skipping dependency check.${NC}"
fi

# 2. Setup .env file
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        echo "Creating .env file from .env.example..."
        cp .env.example .env

        # Update defaults for local dev
        # Use a temporary file for sed compatibility across Linux/macOS
        sed 's/^DB_PASSWORD=.*/DB_PASSWORD=pass/' .env > .env.tmp && mv .env.tmp .env

        # Add PORT if missing (it's not in .env.example)
        if ! grep -q "^PORT=" .env; then
            echo "" >> .env
            echo "PORT=8080" >> .env
        fi

        echo -e "${GREEN}✓ .env file created and configured.${NC}"
    else
        echo -e "${YELLOW}Warning: .env.example not found. Creating .env from scratch...${NC}"
        cat <<EOF > .env
DB_USER=dev
DB_PASSWORD=pass
DB_HOST=localhost
DB_PORT=5432
DB_NAME=app
LOG_LEVEL=INFO
PORT=8080
EOF
        echo -e "${GREEN}✓ .env file created from scratch.${NC}"
    fi
else
    echo "✓ .env file already exists."
fi

# Load .env to get DB config for wait check
set -a
source .env
set +a

# Determine Docker Compose command
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    DOCKER_COMPOSE="docker compose"
fi

# 3. Start Database
echo "Starting database via Docker Compose..."
$DOCKER_COMPOSE up -d db

# 4. Wait for Database
echo "Waiting for database to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if $DOCKER_COMPOSE exec -T db pg_isready -U ${DB_USER:-dev} -d ${DB_NAME:-app} > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Database is ready${NC}"
        break
    fi

    ATTEMPT=$((ATTEMPT + 1))
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo -e "${RED}Error: Database failed to start after 30 seconds${NC}"
        $DOCKER_COMPOSE logs db
        exit 1
    fi

    echo "Waiting for database... ($ATTEMPT/$MAX_ATTEMPTS)"
    sleep 1
done

# 5. Run Migrations
echo "Running migrations..."
make migrate-up

echo -e "${GREEN}🎉 Development environment initialized successfully!${NC}"
echo "You can now run the app with: make run"

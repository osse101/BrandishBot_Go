#!/bin/bash

# setup_env.sh

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting environment setup...${NC}"

# Check for .env file
if [ ! -f .env ]; then
    echo "Creating .env file from defaults..."
    cat <<EOF > .env
DB_USER=dev
DB_PASSWORD=pass
DB_HOST=localhost
DB_PORT=5432
DB_NAME=app
LOG_LEVEL=INFO
PORT=8080
EOF
    echo -e "${GREEN}.env file created.${NC}"
else
    echo ".env file already exists."
fi

# Check for Go
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go to run the setup script locally."
    exit 1
fi

# Check for Docker
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker to run the database container."
    exit 1
fi

# Ensure database container is running
if [ ! "$(docker ps -q -f name=pg)" ]; then
    if [ "$(docker ps -aq -f name=pg)" ]; then
        echo "Starting existing 'pg' container..."
        docker start pg
    else
        echo "Creating and starting 'pg' container..."
        docker run --name pg \
          -e POSTGRES_PASSWORD=pass \
          -e POSTGRES_USER=dev \
          -e POSTGRES_DB=app \
          -p 5432:5432 \
          -v pgdata:/var/lib/postgresql/data \
          -d postgres:15
    fi
    
    # Wait for DB to be ready
    echo "Waiting for database to be ready..."
    sleep 5
else
    echo "'pg' container is already running."
fi

# Run database setup/migrations
echo "Running database setup and migrations..."
go run cmd/setup/main.go

echo -e "${GREEN}Setup complete! You can now run the application with: go run cmd/app/main.go${NC}"

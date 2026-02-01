#!/usr/bin/env bash
# Check if Docker database is running and start if needed

set -euo pipefail

echo "Checking Docker database status..."

# Check if docker compose is available
if ! command -v docker compose &> /dev/null; then
    echo "Error: docker compose not found. Please install Docker Compose."
    exit 1
fi

# Check if db service is running
if docker compose ps db | grep -q "Up"; then
    echo "✓ Database is already running"
else
    echo "Starting database..."
    docker compose up -d db
    
    echo "Waiting for database to be ready..."
    sleep 3
    
    # Wait for database to accept connections
    MAX_ATTEMPTS=30
    ATTEMPT=0
    
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        if docker compose exec -T db pg_isready -U ${DB_USER:-dev} -d ${DB_NAME:-app} > /dev/null 2>&1; then
            echo "✓ Database is ready"
            break
        fi
        
        ATTEMPT=$((ATTEMPT + 1))
        if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
            echo "Error: Database failed to start after 30 seconds"
            docker compose logs db
            exit 1
        fi
        
        echo "Waiting for database... ($ATTEMPT/$MAX_ATTEMPTS)"
        sleep 1
    done
fi

echo "Database check complete"

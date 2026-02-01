#!/usr/bin/env bash
set -euo pipefail

# setup_env.sh
# Delegating to Makefile for consistency

echo "🚀 Starting environment setup via Makefile..."

if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
fi

make check-deps
make docker-up
echo "Waiting for database..."
sleep 5
make migrate-up
make generate

echo "✅ Setup complete!"

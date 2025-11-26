#!/bin/sh
set -e

echo "Waiting for database..."
# Wait for postgres to be ready
until pg_isready -h db -U $DB_USER; do
  echo "Database not ready, waiting..."
  sleep 2
done

echo "Running migrations..."
goose -dir migrations postgres "postgres://$DB_USER:$DB_PASSWORD@db:5432/$DB_NAME?sslmode=disable" up

echo "Starting application..."
exec ./brandishbot

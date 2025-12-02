#!/bin/sh
set -e

echo "Waiting for database..."
# Wait for postgres to be ready with timeout
MAX_RETRIES=30
RETRY_COUNT=0
until pg_isready -h db -U $DB_USER; do
  RETRY_COUNT=$((RETRY_COUNT + 1))
  if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
    echo "ERROR: Database failed to become ready after $MAX_RETRIES attempts"
    exit 1
  fi
  echo "Database not ready, waiting... ($RETRY_COUNT/$MAX_RETRIES)"
  sleep 2
done

echo "Database is ready!"

# Create backup before migrations (production safety)
if [ "${ENVIRONMENT}" = "production" ] || [ "${CREATE_BACKUP}" = "true" ]; then
  echo "Creating pre-migration backup..."
  BACKUP_FILE="/tmp/backup_$(date +%Y%m%d_%H%M%S).sql"
  pg_dump -h db -U $DB_USER -d $DB_NAME > $BACKUP_FILE 2>/dev/null || {
    echo "Warning: Could not create backup (database may be empty)"
  }
  if [ -f "$BACKUP_FILE" ]; then
    echo "Backup created: $BACKUP_FILE"
  fi
fi

# Run migrations with retry logic
echo "Running migrations..."
MAX_MIGRATION_RETRIES=3
MIGRATION_RETRY=0
MIGRATION_SUCCESS=false

while [ $MIGRATION_RETRY -lt $MAX_MIGRATION_RETRIES ]; do
  MIGRATION_RETRY=$((MIGRATION_RETRY + 1))
  
  if goose -dir migrations postgres "postgres://$DB_USER:$DB_PASSWORD@db:5432/$DB_NAME?sslmode=disable" up; then
    MIGRATION_SUCCESS=true
    break
  else
    echo "Migration attempt $MIGRATION_RETRY failed"
    if [ $MIGRATION_RETRY -lt $MAX_MIGRATION_RETRIES ]; then
      echo "Retrying in 5 seconds..."
      sleep 5
    fi
  fi
done

if [ "$MIGRATION_SUCCESS" = "false" ]; then
  echo "ERROR: Migrations failed after $MAX_MIGRATION_RETRIES attempts"
  exit 1
fi

echo "Migrations completed successfully"

echo "Starting application..."
exec ./brandishbot

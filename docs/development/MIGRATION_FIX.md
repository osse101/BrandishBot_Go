# Migration & Docker Fixes - Complete

## Summary
Successfully fixed migration issues and containerized the application.

## 1. Migration Fix (Combined Files)
**Problem**: `goose` (all recent versions) detects duplicate versions when `.up.sql` and `.down.sql` files exist in the same directory.
**Solution**: Combined separate files into single `.sql` files with goose markers.

**Format**:
```sql
-- +goose Up
CREATE TABLE ...;

-- +goose Down
DROP TABLE ...;
```

**Action Taken**:
- Ran `scripts/combine_migrations.sh` to merge all 15 migrations.
- Verified `goose status` works correctly.
- Verified `goose up` applies all migrations.

## 2. Docker Fixes
**Problem**: Docker container was missing `goose` and failing to connect to DB.
**Solution**:
- **Dockerfile**: Added `goose` installation and `docker-entrypoint.sh`.
- **Entrypoint**: Runs migrations automatically on startup.
- **Compose**: Added `DB_HOST=db` override to connect to database container.

## 3. Tool Path Fixes
**Problem**: `goose` not in PATH.
**Solution**: Updated `Makefile` to use `$(GOOSE)` variable which finds the binary automatically.

## How to Run

### Local Development
```bash
# Start DB
make docker-up

# Run migrations
./scripts/goose.sh up

# Run App
make run
```

### Docker Production
```bash
# Build & Start
docker-compose up --build

# Verify
curl -H "X-API-Key: <your-key>" http://localhost:8080/healthz
```

## Verification
✅ Migrations applied: 15/15
✅ App running in Docker
✅ Database connected
✅ Health check passing

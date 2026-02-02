# Issue: Recurring "database dev does not exist" Error

**Status**: RESOLVED
**Severity**: Low (does not affect functionality)
**Date**: 2025-12-21  
**Environment**: Development/Staging

## Symptom

PostgreSQL logs show recurring connection attempts to non-existent database "dev" every ~10 seconds:

```
db-1  | 2025-12-21 18:01:10.439 UTC [45] FATAL:  database "dev" does not exist
db-1  | 2025-12-21 18:01:20.510 UTC [54] FATAL:  database "dev" does not exist
db-1  | 2025-12-21 18:01:30.571 UTC [63] FATAL:  database "dev" does not exist
```

**Impact**: None - application functions normally. All successful health checks and migrations complete.

## Configuration

**Correct settings** (per `.env`):
- `DB_USER=dev`
- `DB_NAME=app`
- `DB_HOST=db` (in Docker) or `localhost` (local)
- `DB_PORT=5432`

All application code uses these values correctly.

## Investigation Results

### ✅ Verified Working
- App successfully connects to `app` database
- Migrations run successfully
- All healthchecks pass (`/healthz` returns 200)
- Discord bot functions normally
- No hardcoded "dev" database references in code

### ❌ Ruled Out
- **Deploy script**: Fixed to load `.env` properly (commit e1fdea1)
- **Hardcoded values**: No `DB_NAME=dev` or `database=dev` found in codebase
- **Discord bot**: Does not connect to database (API-only)
- **App containers**: All connections go to correct "app" database

### 🔍 Suspected Sources

Unknown process making connections every 10 seconds. Possibilities:
1. **Health monitoring tool** running in background
2. **IDE database plugin** with old connection settings
3. **Previous docker-compose network** with lingering connections
4. **System service** (pgAdmin, DBeaver, etc.) with incorrect config
5. **cron job or systemd timer** attempting database checks

## Attempts to Reproduce

```bash
# Check active connections
docker-compose exec db psql -U dev -d app -c \
  "SELECT datname, usename, application_name, client_addr FROM pg_stat_activity;"

# Result: All connections to "app" database (correct)
```

## Temporary Workaround

```bash
# Restart containers to clear any lingering processes
make docker-down
make docker-up
```

**Effect**: Error persists after restart, confirming it's from external source.

## Resolution

Identified `scripts/check_db.sh` using `pg_isready -U ${DB_USER:-dev}` without specifying the database name. `pg_isready` defaults to the username if `-d` is not specified. Since `DB_USER` defaults to `dev`, it was checking `dev` database.

**Fix**: Updated `scripts/check_db.sh` to include `-d ${DB_NAME:-app}`. Also updated `docs/deployment/PRODUCTION_STRATEGY.md` which had a similar incorrect example.

## References

- Deploy script fix: commit `e1fdea1`
- `.env.example` has correct defaults
- All Docker Compose configurations verified

## Notes

- Error frequency is **exactly 10 seconds** - strongly suggests automated polling
- Application is **fully functional** despite errors
- Priority: Low (cosmetic issue only)

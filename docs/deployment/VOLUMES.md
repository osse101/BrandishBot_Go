# Docker Volumes Guide

Volumes are used to persist data and share files between the host machine and Docker containers.

## 1. Database Persistence (Critical)
The database volume ensures your user data, inventory, and progression are saved even if the database container is destroyed.

**Current Configuration:**
```yaml
services:
  db:
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```
This uses a **named volume** managed by Docker.
- **Location**: `/var/lib/docker/volumes/brandishbot_go_pgdata/_data` (on Linux)
- **Backup**: Use `make db-export` to create a SQL dump.

---

## 2. Configuration (Optional)
The application loads loot tables from `configs/loot_tables.json` on startup.
To modify these tables without rebuilding the Docker image, you can mount the `configs` directory.

**How to Enable:**
Update `docker compose.yml`:
```yaml
services:
  app:
    volumes:
      - ./configs:/app/configs:ro  # Read-only mount
```

**Workflow:**
1. Edit `configs/loot_tables.json` on your host.
2. Restart the app to apply changes:
   ```bash
   docker compose restart app
   ```

---

## 3. Logs (Optional)
By default, we use the `json-file` logging driver which manages logs efficiently.
If you prefer to have log files written to your host machine (e.g., for external analysis tools), you can mount the logs directory.

**How to Enable:**
1. Update `.env` to set `LOG_DIR=/app/logs`.
2. Update `docker compose.yml`:
   ```yaml
   services:
     app:
       volumes:
         - ./logs:/app/logs
   ```

**Recommendation:** Stick to `docker compose logs` (default) unless you have a specific need for file-based logging.

---

## Summary of Data Locations

| Data Type | Persistence | Location | Backup Strategy |
|-----------|-------------|----------|-----------------|
| **Database** | ✅ Yes (Volume) | `pgdata` volume | `pg_dump` / `make db-export` |
| **Loot Tables** | ❌ No (Baked in) | `/app/configs` | Rebuild image OR mount volume |
| **Logs** | ⚠️ Rotated | Docker Daemon | `docker logs` |
| **Timeouts** | ❌ No (Memory) | RAM | None (Reset on restart) |

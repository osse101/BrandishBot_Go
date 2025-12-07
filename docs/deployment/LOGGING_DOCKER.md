# Docker Logging Guide

## Accessing Logs

### 1. View Real-time Logs (Follow)
To stream logs from all services:
```bash
docker compose logs -f
```

To stream logs from a specific service (e.g., `app` or `db`):
```bash
docker compose logs -f app
docker compose logs -f db
```

### 2. View Past Logs
To see the last 100 lines:
```bash
docker compose logs --tail=100 app
```

### 3. Log Rotation (Optimized)
We have configured log rotation in `docker compose.yml` to prevent logs from consuming all disk space.
- **Driver**: `json-file`
- **Max Size**: `10m` (10 Megabytes per file)
- **Max Files**: `3` (Keep last 3 files)

This ensures logs never exceed ~30MB per container.

## Application Logging
The application uses structured JSON logging in production (configured via `LOG_FORMAT=json` in `.env`).

**Example Output:**
```json
{"time":"2023-11-26T12:00:00Z","level":"INFO","msg":"Starting server","port":8080,"service":"brandish-bot"}
```

### Parsing Logs
You can pipe logs to `jq` for better readability:
```bash
docker compose logs -f app | grep "{" | jq .
```

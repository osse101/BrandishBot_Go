# Environment Guide: Development vs. Production

This guide explains how to run BrandishBot in different environments and the key differences between them.

## 1. Local Development Workstation

**Purpose**: Coding, testing, debugging, and running migrations manually.

### Prerequisites
Run `./scripts/check_deps.sh` to verify:
- Go 1.25+
- Docker & Docker Compose (for database)
- Make
- Goose (for migrations)

### Workflow
1. **Start Database**:
   ```bash
   make docker-up  # Starts Postgres in Docker
   ```

2. **Run Migrations**:
   ```bash
   ./scripts/goose.sh up
   ```

3. **Run Application**:
   ```bash
   make run
   # OR
   go run cmd/app/*.go
   ```

4. **Run Tests**:
   ```bash
   make test
   ```

### Key Features
- **Hot Reload**: Not enabled by default (requires `air`), but you can restart `go run` quickly.
- **Direct DB Access**: Database port `5432` is exposed to localhost.
- **Logs**: Output to stdout in text format (readable).
- **Debug**: Can attach debugger (Delve) to the process.

---

## 2. Dockerized Server (Production/Staging)

**Purpose**: Deployment, stability, isolation.

### Prerequisites
- Docker & Docker Compose ONLY (Go/Make/Goose NOT required on host)

### Workflow
1. **Deploy & Start**:
   ```bash
   docker-compose up -d --build
   ```
   *That's it!*

### How It Works
- **Containerized App**: The app runs inside a lightweight Alpine Linux container.
- **Auto-Migrations**: The container entrypoint (`scripts/docker-entrypoint.sh`) automatically runs migrations before starting the app.
- **Internal Networking**: App talks to DB via internal Docker network (`db:5432`).
- **Security**: Database port `5432` can be closed to the outside world (remove `ports` mapping in `docker-compose.yml` for production).

### Key Differences

| Feature | Local Dev | Dockerized Server |
|---------|-----------|-------------------|
| **Runtime** | Native Go binary on host | Alpine Linux Container |
| **Database** | Docker container (localhost:5432) | Docker container (db:5432) |
| **Migrations** | Manual (`goose up`) | Automatic (on startup) |
| **Logs** | Text format (stdout) | JSON/Text (configurable) |
| **Dependencies** | Go, Make, Goose required | Only Docker required |

### Troubleshooting Docker
```bash
# View logs
docker-compose logs -f app

# Check health
curl http://localhost:8080/healthz

# Shell into container
docker-compose exec app /bin/bash
```

# Database Configuration Notes

## Development Setup (Current Configuration)

The database is configured for **easy local development** with **security**:

```bash
# Start database
docker compose up -d db

# Or use the helper command
make check-db
```

### How It Works
- Database runs in Docker
- Port 5432 is bound to **localhost only** (`127.0.0.1:5432`)
- ✅ You can run `go run cmd/app/main.go` on your computer
- ✅ Database is **NOT** accessible from your local network
- ✅ Only your computer can connect to it

### Configuration
```env
DB_HOST=localhost
DB_PORT=5432  # This is still needed!
DB_USER=dev
DB_PASSWORD=your_secure_password_here
DB_NAME=app
```

> **Security Note**: The `127.0.0.1:5432:5432` binding in docker compose.yml means the database is only accessible from your computer (localhost), NOT from other devices on your network. This is secure AND convenient for development.

## Production/Docker Deployment

When running the entire application in Docker:

```bash
docker compose up -d
```

In this mode:
- Database is **only accessible via internal Docker network**
- No external port exposure
- Application connects to `db:5432` (Docker service name)
- Highest security - database completely isolated from network

### Configuration for Docker
```env
DB_HOST=db  # Use Docker service name instead of localhost
```

## Security Best Practices

1. **Never commit `.env`** - It's in `.gitignore`
2. **Use strong passwords** - Generate with `openssl rand -base64 32`
3. **Rotate credentials** - Change passwords periodically
4. **Limit access** - Database should only be accessible to application

## Troubleshooting

If you see "failed to connect to database":
1. Check if database is running: `docker compose ps db`
2. View database logs: `docker compose logs db`
3. Restart database: `docker compose restart db`
4. Or use the helper command: `make check-db`

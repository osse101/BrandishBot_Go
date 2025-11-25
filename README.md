# BrandishBot_Go

A high-performance game engine API for BrandishBot, built with Go. Provides inventory management, crafting, economy, and statistics tracking for live chatroom gaming experiences.

## Features

- **Inventory Management**: Add, remove, trade, and track items
- **Crafting System**: Recipe-based item crafting and upgrading
- **Economy**: Buy/sell items with dynamic pricing
- **Statistics**: User and system stats with leaderboards
- **Health Checks**: Production-ready liveness and readiness endpoints
- **API Documentation**: Interactive Swagger UI at `/swagger/`

## Quick Start

### Prerequisites
- Go 1.25+
- PostgreSQL 15+
- Docker & Docker Compose (recommended)

### Setup

1. **Clone and configure**:
```bash
cp .env.example .env
# Edit .env with your database credentials
```

2. **Start database**:
```bash
make docker-up
```

3. **Run migrations**:
```bash
make migrate-up
```

4. **Start the server**:
```bash
make run
# Server will start on http://localhost:8080
```

5. **View API documentation**:
Visit http://localhost:8080/swagger/index.html

## Development

### Makefile Commands

**Migrations**:
- `make migrate-up` - Run all pending migrations
- `make migrate-down` - Rollback last migration
- `make migrate-status` - Show migration status
- `make migrate-create NAME=<name>` - Create new migration

**Development**:
- `make test` - Run tests with coverage
- `make test-coverage` - Generate HTML coverage report
- `make build` - Build all binaries
- `make swagger` - Regenerate Swagger docs

**Docker**:
- `make docker-up` - Start services
- `make docker-down` - Stop services
- `make docker-logs` - View logs

### Project Structure

```
├── cmd/              # Entry points
│   ├── app/         # Main application
│   ├── setup/       # Database setup
│   └── debug/       # Debug tools
├── internal/        # Application code
│   ├── handler/     # HTTP handlers
│   ├── domain/      # Domain models
│   ├── user/        # User service
│   ├── crafting/    # Crafting service
│   ├── economy/     # Economy service
│   └── stats/       # Statistics service
├── migrations/      # SQL migrations
└── docs/            # Documentation & Swagger
```

## API Endpoints

### Health
- `GET /healthz` - Liveness check
- `GET /readyz` - Readiness check (DB connectivity)

### User
- `POST /user/register` - Register user
- `GET /user/inventory` - Get inventory
- `POST /user/item/add` - Add item
- `POST /user/item/use` - Use item

### Crafting
- `POST /user/item/upgrade` - Upgrade item
- `POST /user/item/disassemble` - Disassemble item
- `GET /recipes` - Get crafting recipes

### Economy
- `POST /user/item/buy` - Buy item
- `POST /user/item/sell` - Sell item
- `GET /prices` - Get market prices

### Stats
- `POST /stats/event` - Record event
- `GET /stats/user` - Get user stats
- `GET /stats/leaderboard` - Get leaderboard

*See `/swagger/` for complete API documentation with request/response examples.*

## Testing

```bash
# Run all tests
make test

# Generate coverage report
make test-coverage
# Open coverage.html in browser
```

## Contributing

See [AGENTS.md](./AGENTS.md) for development guidelines and architecture details.

## License

MIThread safety.

# BrandishBot_Go

A high-performance game engine API for BrandishBot, built with Go. Provides inventory management, crafting, economy, and statistics tracking for live chatroom gaming experiences.

## Features

- **Inventory Management**: Add, remove, trade, and track items
- **Crafting System**: Recipe-based item crafting and upgrading
- **Economy**: Buy/sell items with dynamic pricing
- **Statistics**: User and system stats with leaderboards
- **Discord Bot**: Full-featured Discord integration with slash commands
- **Health Checks**: Production-ready liveness and readiness endpoints
- **API Documentation**: Interactive Swagger UI at `/swagger/`

## Discord Bot 🤖

BrandishBot includes a native Discord bot with 21 slash commands providing complete feature access directly in Discord!

### Quick Start (Discord)

1. **Configure** (add to `.env`):
```bash
DISCORD_TOKEN=your_bot_token
DISCORD_APP_ID=your_app_id
API_URL=http://localhost:8080
```

2. **Start Discord Bot**:
```bash
# Local development
make build
make discord-run

# Docker (with core API)
make docker-up
```

3. **Use Commands**:
- `/info` - Get started
- `/profile` - View your profile
- `/search` - Find items
- `/inventory` - See your items
- `/help` for more!

### Discord Commands

**Economy**: `/buy`, `/sell`, `/prices`, `/give`  
**Inventory**: `/inventory`, `/search`, `/use`  
**Crafting**: `/upgrade`, `/disassemble`, `/recipes`  
**Gambling**: `/gamble`, `/join-gamble`  
**Stats**: `/stats`, `/leaderboard`  
**Progression**: `/vote`  
**Admin**: `/add-item`, `/remove-item`

See `/info commands` in Discord for full details.

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
- `make unit` - Run unit tests (fast)
- `make test-coverage` - Generate HTML coverage report
- `make build` - Build all binaries
- `make swagger` - Regenerate Swagger docs

**Discord Bot**:
- `make discord-run` - Run Discord bot locally
- `make discord-logs` - View Discord logs (Docker)
- `make docker-discord-build` - Build Discord image
- `make docker-discord-up` - Start Discord service
- `make docker-discord-restart` - Restart Discord bot

**Docker**:
- `make docker-up` - Start services
- `make docker-down` - Stop services
- `make docker-logs` - View logs

### Project Structure

```
├── cmd/              # Entry points
│   ├── app/         # Main application
│   ├── discord/     # Discord bot entry point
│   ├── setup/       # Database setup
│   ├── reset/       # Database reset utility
│   └── debug/       # Debug tools
├── internal/        # Application code
│   ├── handler/     # HTTP handlers
│   ├── domain/      # Domain models
│   ├── repository/  # Database interfaces
│   ├── database/    # SQLC and Postgres implementation
│   ├── user/        # User service
│   ├── economy/     # Economy service
│   ├── crafting/    # Crafting service
│   ├── harvest/     # Harvest (Farming) service
│   ├── progression/ # Progression tree service
│   ├── gamble/      # Gambling & Lootbox service
│   ├── job/         # Job & XP service
│   ├── stats/       # Statistics service
│   ├── discord/     # Discord bot implementation
│   └── cooldown/    # Cooldown service
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

## Event System

BrandishBot uses an asynchronous event-driven architecture for decoupled, reliable service communication:

📚 **Documentation:**
- **[Event Catalog](docs/events/EVENT_CATALOG.md)** - All 22+ event types with schemas and examples
- **[Architecture](docs/architecture/EVENT_SYSTEM.md)** - Event bus, ResilientPublisher, retry logic
- **[Developer Guide](docs/architecture/DEVELOPER_GUIDE.md)** - How to publish and subscribe to events

**Key Features:**
- 🔄 Automatic retry with exponential backoff (2s → 4s → 8s → 16s → 32s)
- 📝 Dead-letter logging for permanently failed events
- 🚫 Fire-and-forget: Domain operations never fail due to event errors
- 📊 Used for stats, notifications, audit logging, and cross-service communication

## Contributing

See [AGENTS.md](./AGENTS.md) for development guidelines and architecture details.

## License

MIThread safety.

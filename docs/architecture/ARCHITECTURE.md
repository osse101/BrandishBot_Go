# BrandishBot_Go Architecture

## Overview

BrandishBot_Go is a comprehensive backend gaming engine API that powers live chatroom gaming experiences across multiple streaming platforms (Twitch, YouTube, Discord). The system includes:

- **HTTP API Server** (port 8080) - RESTful API for game mechanics
- **Discord Bot** (port 8082) - Discord integration with slash commands
- **Real-time Events** - Server-Sent Events (SSE) for live updates
- **Background Workers** - Scheduled jobs and async processing

Built using Go with PostgreSQL for persistence, the system manages user progression, economy, crafting, gambling, jobs/XP, stats, and real-time engagement tracking.

## Technology Stack

- **Language**: Go 1.22+
- **Database**: PostgreSQL 15+
- **Database Driver**: pgx/v5
- **Query Generator**: SQLC
- **HTTP Server**: Standard library `net/http` with chi router
- **Discord**: discordgo library
- **WebSocket**: gorilla/websocket
- **Logging**: Zap (structured logging)
- **Metrics**: Prometheus
- **Migrations**: Goose
- **Testing**: Mockery, Testify, Testcontainers
- **Environment**: `.env` file configuration

## Architecture Pattern

The application follows a **layered event-driven architecture** with clear separation of concerns:

```
┌───────────────────────────────────────────────────────────────┐
│  Entry Points (HTTP Handlers, Discord Commands, SSE)          │  ← API/Bot Interface
├───────────────────────────────────────────────────────────────┤
│  Service Layer (Business Logic + Event Publishing)            │  ← Domain Logic
├───────────────────────────────────────────────────────────────┤
│  Repository Layer (Data Access Interfaces)                    │  ← Abstraction
├───────────────────────────────────────────────────────────────┤
│  Database Layer (PostgreSQL + SQLC Generated Queries)         │  ← Persistence
├───────────────────────────────────────────────────────────────┤
│  Event Bus (Pub/Sub) + Background Workers                     │  ← Async Processing
├───────────────────────────────────────────────────────────────┤
│  Real-Time Systems (SSE Hub, Streamer.bot Integration)        │  ← Live Updates
└───────────────────────────────────────────────────────────────┘
```

### Request Flow Pattern

Standard flow for all features:

```
Handler (internal/handler/)
  → Service (internal/*/service.go)
    → Repository (internal/repository/)
      → Postgres (internal/database/postgres/)
        → Event Bus (async notifications)
          → SSE Hub (real-time broadcast)
```

## Directory Structure

```
BrandishBot_Go/
├── cmd/
│   ├── app/                      # Main HTTP API server entry point
│   ├── discord/                  # Discord bot entry point
│   ├── setup/                    # Database setup utility
│   ├── debug/                    # Database inspection utility
│   ├── reset/                    # Database reset utility
│   └── gen-progression-keys/     # Progression tree key generator
├── internal/
│   ├── bootstrap/                # App initialization & DI
│   ├── config/                   # Configuration management
│   ├── database/                 # Database connection & queries
│   │   ├── generated/            # SQLC generated code
│   │   ├── postgres/             # Repository implementations
│   │   └── queries/              # SQLC SQL queries
│   ├── domain/                   # Domain models & entities
│   ├── handler/                  # HTTP request handlers
│   ├── server/                   # HTTP server & routing
│   ├── middleware/               # HTTP middleware
│   ├── event/                    # Event bus & publisher
│   ├── eventlog/                 # Event logging service
│   ├── sse/                      # Server-Sent Events hub
│   ├── user/                     # User service
│   ├── economy/                  # Economy system (buy/sell)
│   ├── crafting/                 # Crafting & disassembly
│   ├── progression/              # Progression tree & voting
│   ├── gamble/                   # Gamble sessions
│   ├── lootbox/                  # Loot table & drops
│   ├── job/                      # Jobs & XP system
│   ├── stats/                    # Stats & leaderboards
│   ├── cooldown/                 # Cooldown management
│   ├── linking/                  # Platform account linking
│   ├── naming/                   # Item name resolution
│   ├── scheduler/                # Background job scheduler
│   ├── worker/                   # Background workers
│   ├── streamerbot/              # Streamer.bot WebSocket client
│   ├── discord/                  # Discord bot commands
│   ├── repository/               # Repository interfaces
│   ├── logger/                   # Structured logging (Zap)
│   ├── metrics/                  # Prometheus metrics
│   ├── features/                 # Feature flags
│   └── utils/                    # Shared utilities
├── configs/
│   ├── items/                    # Item definitions & aliases
│   ├── recipes/                  # Crafting recipes
│   ├── loot_tables.json          # Loot table configuration
│   └── progression_tree.json     # Progression tree definition
├── migrations/                   # Goose SQL migrations
├── client/
│   └── csharp/                   # C# Streamer.bot client
├── docs/                         # Documentation
│   ├── architecture/             # Architecture docs & journals
│   ├── development/              # Dev guides & journals
│   ├── testing/                  # Testing guides
└── .env                          # Environment configuration
```

## Core Components

### 1. Application Bootstrap (`internal/bootstrap/`)

Handles application initialization and dependency injection:

- Database connection management
- Service initialization order
- Event bus setup
- Background worker startup
- Graceful shutdown orchestration

### 2. Configuration (`internal/config/`)

Environment-based configuration management:

- Database connection strings
- Server ports (API: 8080, Discord: 8082)
- Feature flags
- External service URLs (Streamer.bot)

### 3. Domain Layer (`internal/domain/`)

Core business entities and types:

- **User**: Multi-platform user accounts
- **Item**: In-game items with quality, type
- **Inventory**: JSONB-stored user inventories
- **Job**: Job definitions with XP requirements
- **ProgressionNode**: Tree nodes with costs, prerequisites, modifiers
- **Gamble**: Gamble session state
- **Events**: Typed event payloads

### 4. Event System (`internal/event/`)

**Event Bus** - Central pub/sub message broker:

- Topic-based subscription
- Resilient publishing with retry (exponential backoff)
- Event types: Engagement, Gamble, JobLevelUp, ProgressionCycle

**Key Event Types:**

- `EventTypeEngagement` - User engagement tracking
- `EventGambleStarted`, `EventGambleComplete` - Gamble lifecycle
- `EventTypeJobLevelUp` - Job progression
- `EventTypeProgressionCycleCompleted` - Voting cycle end
- `EventTypeProgressionTargetSet` - New unlock target
- `EventTypeProgressionVotingStarted` - Voting session start
- `EventTypeProgressionAllUnlocked` - All nodes unlocked

**Resilient Publisher** (`internal/event/resilient_publisher.go`):

- Retry pattern: 2s → 4s → 8s → 16s → 32s
- Circuit breaker for failed handlers
- Metrics tracking

### 5. Real-Time Systems

#### SSE Hub (`internal/sse/`)

Server-Sent Events for real-time browser updates:

- 100-message buffer per client
- 30-second keepalive
- Automatic cleanup on disconnect
- Event types: job.level_up, progression.\*, gamble.complete

**Event Integration** (`internal/sse/event_integration.go`):

- Subscribes to event bus
- Transforms events to SSE format
- Broadcasts to all connected clients

#### Streamer.bot Integration (`internal/streamerbot/`)

WebSocket client for Streamer.bot integration:

- Real-time event forwarding
- Automatic reconnection
- Event filtering and transformation
- Connection lifecycle management

### 6. Repository Layer (`internal/repository/`)

Data access interfaces with implementations in `internal/database/postgres/`:

- **UserRepository**: User accounts, inventory, timeout management
- **EconomyRepository**: Item prices, buy/sell transactions
- **CraftingRepository**: Recipe management, unlock tracking
- **ProgressionRepository**: Tree nodes, unlocks, voting sessions, engagement
- **GambleRepository**: Gamble sessions, participants, results
- **JobRepository**: Job definitions, user XP, level tracking
- **StatsRepository**: Event tracking, leaderboards, streaks
- **CooldownRepository**: Action cooldowns with check-then-lock pattern
- **LinkingRepository**: Platform account linking

**Pattern**: All repositories return domain models, handle transactions internally

### 7. Service Layer

Business logic with event publishing:

#### User System (`internal/user/`)

- User registration and platform linking
- Inventory management (add, remove, use items)
- Timeout enforcement
- User search

#### Economy System (`internal/economy/`)

- Dynamic pricing with job bonuses
- Buy/sell item transactions
- Price calculation based on base values

#### Crafting System (`internal/crafting/`)

- Item upgrades with masterwork chance (10%, 2x output)
- Item disassembly with perfect salvage (10%, 1.5x output)
- Recipe unlocking and management
- Job XP rewards for crafting actions

#### Progression System (`internal/progression/`)

- **Tree Management**: Load progression tree from JSON
- **Voting Sessions**: Parallel voting on multiple nodes
- **Vote Accumulation**: Unlock nodes during voting period
- **Cycle Management**: Complete voting cycles, start new sessions
- **Dynamic Prerequisites**: Runtime evaluation (nodes_unlocked_below_tier, total_nodes_unlocked)
- **Cost Calculation**: Tier-based scaling (baseCost × 1.30^tier)
- **Modifier Application**: Cached modifier effects (30-min TTL)
- **Engagement Tracking**: User contribution metrics
- **Admin Controls**: Freeze voting, force-end sessions

#### Gamble System (`internal/gamble/`)

- Gamble session creation and joining
- Worker-based async execution
- Quality-level multipliers (COMMON 1.0x to LEGENDARY 2.0x)
- Near-miss threshold (95%)
- Lootbox integration

#### Lootbox System (`internal/lootbox/`)

- Weighted random item selection
- Loot table configuration from JSON
- Quality level determination
- Item drop tracking

#### Job/XP System (`internal/job/`)

- Job definitions with XP curves
- XP awarding and level calculation
- Job bonus multipliers
- Level-up event publishing

#### Stats System (`internal/stats/`)

- User event tracking
- Streak calculation (daily engagement)
- Leaderboard generation
- System-wide statistics

#### Cooldown System (`internal/cooldown/`)

- Check-then-lock pattern (race-free)
- Configurable per-action cooldowns
- Transaction-based enforcement
- User-specific and global cooldowns

### 8. Handler Layer (`internal/handler/`)

HTTP request handlers organized by feature:

- **user.go**: Registration, inventory, timeout, search
- **economy.go**: Prices, buy, sell
- **crafting.go**: Upgrade, disassemble, recipes
- **progression.go**: Tree, voting, engagement, admin controls
- **gamble.go**: Start, join, retrieve sessions
- **job.go**: List jobs, user jobs, XP awards
- **stats.go**: User stats, leaderboards
- **sse.go**: SSE endpoint
- **health.go**: Health and readiness checks

### 9. Discord Bot (`internal/discord/`)

Discord integration with slash commands:

- **Bot Core**: Command registration, event handling
- **Commands**: Mirror API functionality (cmd\_\*.go files)
- **API Client**: HTTP client for API calls
- **Autocomplete**: Dynamic option completion
- **SSE Client**: Real-time event subscription

### 10. Scheduler & Workers (`internal/scheduler/`, `internal/worker/`)

Background job processing:

- **Scheduler**: Cron-based job scheduling
- **Gamble Worker**: Async gamble execution with queue
- **Jobs**: Progression cycle management, cleanup tasks

### 11. Observability

#### Logging (`internal/logger/`)

- Structured logging with Zap
- Correlation IDs for request tracing
- Log levels: DEBUG, INFO, WARN, ERROR
- JSON output for log aggregation

#### Metrics (`internal/metrics/`)

- Prometheus metric collection
- Application metrics (request counts, durations)
- Custom business metrics (gambles, crafts, votes)
- Endpoint: `/metrics`

#### Event Logging (`internal/eventlog/`)

- Event storage for audit trail
- Event replay capability
- Integration with event bus

### 12. Middleware (`internal/middleware/`)

HTTP middleware stack:

- CORS handling
- Request logging with duration
- Panic recovery
- Request ID injection

## Database Schema

### User & Platform System

```sql
platforms              users                user_platform_links
┌──────────┐          ┌──────────┐          ┌────────────┐
│platform_id│──┐   ┌──│user_id    │──┐   ┌──│user_id      │
│name       │  │   │  │username   │  │   │  │platform_id  │
└──────────┘  │   │  │created_at │  │   │  │platform_user│
              └───┼──│updated_at │  ├───┘  │timed_out_at │
                  └──│timed_out_at│  │     └────────────┘
                     └──────────┘  │
                                   │
                  ┌────────────────┘
                  │
                  │  user_inventory
                  │  ┌──────────────┐
                  └──│user_id        │
                     │inventory_data │ (JSONB)
                     │updated_at     │
                     └──────────────┘
```

### Items & Economy

```sql
items                  item_types            item_type_assignments
┌──────────┐          ┌──────────┐          ┌──────────┐
│item_id    │──┐   ┌──│item_type_id│──┐  ┌──│item_id    │
│item_name  │  │   │  │type_name   │  │  │  │item_type_id│
│description│  │   │  └──────────┘  ├──┘  └──┘
│base_value │  │   │                │
│rarity     │  │   │                │
│quality_level│  │   │                │
└──────────┘  └───┼────────────────┘
                  └─(many-to-many)
```

### Crafting System

```sql
crafting_recipes                disassemble_recipes
┌────────────────┐             ┌────────────────┐
│recipe_id       │             │recipe_id       │
│input_item_id   │             │input_item_id   │
│input_quantity  │             │output_item_id  │
│output_item_id  │             │output_quantity │
│output_quantity │             │unlock_type     │
│job_key         │             │unlock_threshold│
│xp_reward       │             └────────────────┘
│unlock_type     │
│unlock_threshold│
└────────────────┘

user_unlocked_recipes
┌────────────────┐
│user_id         │
│recipe_id       │
│unlocked_at     │
└────────────────┘
```

### Progression System

```sql
progression_nodes                progression_unlocks
┌────────────────────┐          ┌────────────────┐
│node_id             │──────┬───│node_id         │
│node_key            │      │   │unlocked_at     │
│node_type           │      │   │unlock_method   │
│tier                │      │   └────────────────┘
│size                │      │
│cost                │      │   progression_voting_sessions
│description         │      │   ┌────────────────────┐
│category            │      │   │session_id          │
│unlock_description  │      │   │tier                │
│modifier_type       │      │   │status (voting/frozen/completed)
│modifier_value      │      │   │target_node_id      │
│prerequisites       │      │   │votes_accumulated   │
└────────────────────┘      │   │cost_to_unlock      │
                            │   │started_at          │
progression_voting_options  │   │ended_at            │
┌────────────────────┐      │   └────────────────────┘
│session_id          │──────┤
│node_id             │──────┘   progression_voting_weights
│votes               │           ┌────────────────────┐
└────────────────────┘           │user_id             │
                                 │voting_weight       │
                                 │reason              │
                                 │created_at          │
                                 │expires_at          │
                                 └────────────────────┘

engagement_metrics
┌────────────────────┐
│user_id             │
│total_engagement    │
│votes_cast          │
│items_crafted       │
│gambles_participated│
│last_engagement_at  │
└────────────────────┘
```

### Gamble System

```sql
gambles                        gamble_participants
┌────────────────┐            ┌────────────────┐
│gamble_id       │────────────│gamble_id       │
│creator_id      │            │user_id         │
│lootbox_id      │            │wager_item_id   │
│status          │            │wager_quantity  │
│max_participants│            │joined_at       │
│created_at      │            └────────────────┘
│executed_at     │
└────────────────┘            gamble_opened_items
                              ┌────────────────┐
                              │gamble_id       │
                              │item_id         │
                              │quantity        │
                              │quality_level     │
                              │opened_at       │
                              └────────────────┘
```

### Jobs & XP System

```sql
jobs                           user_jobs
┌────────────────┐            ┌────────────────┐
│job_id          │────────────│user_id         │
│job_key         │            │job_id          │
│job_name        │            │xp              │
│description     │            │level           │
│max_level       │            │last_xp_at      │
└────────────────┘            └────────────────┘
```

### Stats & Leaderboards

```sql
stats_events                   events
┌────────────────┐            ┌────────────────┐
│event_id        │            │event_id        │
│user_id         │            │event_type      │
│event_type      │            │user_id         │
│created_at      │            │payload (JSONB) │
└────────────────┘            │created_at      │
                              └────────────────┘
```

### Loot Tables

```sql
loot_tables                    loot_table_items
┌────────────────┐            ┌────────────────┐
│loot_table_id   │────────────│loot_table_id   │
│loot_table_name │            │item_id         │
│description     │            │weight          │
└────────────────┘            │min_quantity    │
                              │max_quantity    │
                              └────────────────┘
```

### Cooldowns

```sql
cooldowns
┌────────────────┐
│user_id         │
│action_key      │
│expires_at      │
│created_at      │
└────────────────┘
```

### Key Features

- **UUID Primary Keys**: All user-related tables use UUID
- **JSONB Storage**: Flexible storage for inventory and event payloads
- **GIN Indexing**: Fast JSONB queries on inventory_data
- **Multi-Platform Links**: Users can link multiple streaming platforms
- **Voting Sessions**: Parallel voting with unlock accumulation
- **Dynamic Prerequisites**: Runtime-evaluated unlock requirements
- **Event Sourcing**: Full event log with payloads for audit trail
- **Engagement Tracking**: Per-user metrics for progression system
- **Check-Then-Lock**: Cooldown enforcement with row-level locking

## API Endpoints

### Health & Monitoring

- `GET /healthz` - Health check
- `GET /readyz` - Readiness check
- `GET /version` - Version information
- `GET /metrics` - Prometheus metrics

### User Management

- `POST /api/v1/user/register` - Register new user or link platform
- `GET /api/v1/user/inventory` - Get user inventory
- `GET /api/v1/user/inventory/:username` - Get inventory by username
- `PUT /api/v1/user/timeout` - Set user timeout
- `POST /api/v1/user/search` - Search users
- `POST /api/v1/user/item/add` - Add item to inventory
- `POST /api/v1/user/item/remove` - Remove item from inventory
- `POST /api/v1/user/item/use` - Use consumable item

### Economy

- `GET /api/v1/prices` - Get sellable item prices
- `GET /api/v1/prices/buy` - Get buyable item prices
- `POST /api/v1/economy/buy` - Buy item
- `POST /api/v1/economy/sell` - Sell item

### Crafting

- `POST /api/v1/user/item/upgrade` - Upgrade item (10% masterwork chance)
- `POST /api/v1/user/item/disassemble` - Disassemble item (10% perfect salvage)
- `GET /api/v1/crafting/recipes` - Get unlocked recipes

### Progression System

- `GET /api/v1/progression/tree` - Get full progression tree
- `GET /api/v1/progression/available` - Get available nodes to vote on
- `POST /api/v1/progression/vote` - Vote for node unlock
- `GET /api/v1/progression/status` - Get current voting status
- `POST /api/v1/progression/engagement` - Record engagement event
- `POST /api/v1/progression/engagement/:username` - Record engagement by username
- `GET /api/v1/progression/leaderboard` - Get engagement leaderboard
- `GET /api/v1/progression/session` - Get current voting session

**Admin Endpoints:**

- `POST /api/v1/progression/admin/freeze` - Freeze voting (prevents new votes)
- `POST /api/v1/progression/admin/force-end` - Force end session and publish unlocks
- `POST /api/v1/progression/admin/start` - Start new voting session
- `PUT /api/v1/progression/admin/weights` - Update user voting weights

### Gamble System

- `POST /api/v1/gamble/start` - Start gamble session
- `POST /api/v1/gamble/join` - Join gamble session
- `GET /api/v1/gamble/get` - Get gamble session details

### Jobs & XP

- `GET /api/v1/jobs` - Get all jobs
- `GET /api/v1/jobs/user` - Get user jobs with levels
- `POST /api/v1/jobs/award-xp` - Award XP to user
- `GET /api/v1/jobs/bonus` - Get job bonus multiplier
- `POST /api/v1/admin/jobs/xp` - Award XP (admin endpoint)

### Stats & Leaderboards

- `POST /api/v1/stats/event` - Record user event
- `GET /api/v1/stats/user` - Get user stats
- `GET /api/v1/stats/system` - Get system-wide stats
- `GET /api/v1/stats/leaderboard` - Get leaderboard

### Message Handling

- `POST /api/v1/message/handle` - Handle chat message
- `POST /api/v1/message/test` - Test message handling

### Admin

- `POST /api/v1/admin/reload-aliases` - Reload item aliases from config
- `GET /api/v1/admin/cache/stats` - Get cache statistics

### Real-Time Events

- `GET /api/v1/events` - Server-Sent Events stream

**SSE Event Types:**

- `job.level_up` - User leveled up in a job
- `progression.cycle.completed` - Voting cycle completed
- `progression.target.set` - New unlock target set
- `progression.voting_started` - New voting session started
- `progression.all_unlocked` - All nodes unlocked
- `gamble.complete` - Gamble session completed

### Documentation

- `/swagger/` - Swagger UI for API documentation

## Data Flow Examples

### Adding an Item

```
1. HTTP Request → Handler.HandleAddItem
2. Service.AddItem()
   ├─→ Repository.GetUserByUsername()
   ├─→ Repository.GetItemByName()
   ├─→ Repository.GetInventory()
   ├─→ Update inventory slots (in-memory)
   └─→ Repository.UpdateInventory()
3. HTTP Response ← Success/Error
```

### Progression Voting Flow

```
1. HTTP Request → Handler.HandleVote
2. ProgressionService.VoteForUnlock()
   ├─→ Repository.GetVotingSession()
   ├─→ Repository.RecordVote()
   ├─→ Check if node cost reached
   ├─→ Repository.AccumulateUnlock()
   ├─→ EventBus.Publish(ProgressionVoteRecorded)
   └─→ Cache.Invalidate(unlock cache)
3. HTTP Response ← Vote status
4. SSE Hub ← Broadcast vote update
5. Discord/Streamer.bot ← Real-time notification
```

### Crafting with Events

```
1. HTTP Request → Handler.HandleUpgrade
2. CraftingService.UpgradeItem()
   ├─→ Repository.GetRecipe()
   ├─→ Repository.GetUserInventory()
   ├─→ Check prerequisites (job level, unlocked recipes)
   ├─→ Roll for masterwork (10% chance, 2x output)
   ├─→ Transaction: Remove inputs, add outputs
   ├─→ JobService.AwardXP()
   │   └─→ EventBus.Publish(JobLevelUp) [if leveled up]
   ├─→ StatsService.RecordEvent(ItemCrafted)
   └─→ ProgressionService.RecordEngagement()
       └─→ EventBus.Publish(EngagementRecorded)
3. HTTP Response ← Crafting result
4. SSE Hub ← Broadcast job level up (if applicable)
```

### Gamble Execution Flow

```
1. HTTP Request → Handler.HandleStartGamble
2. GambleService.StartGamble()
   ├─→ Repository.CreateGambleSession()
   └─→ EventBus.Publish(GambleStarted)
3. Users join via HTTP → GambleService.JoinGamble()
4. Background Worker picks up gamble
5. GambleWorker.ExecuteGamble()
   ├─→ LootboxService.OpenLootbox() [for each participant]
   │   ├─→ Roll quality level
   │   ├─→ Select items from loot table
   │   └─→ Calculate multipliers
   ├─→ Determine winner (highest quality × near-miss logic)
   ├─→ Transaction: Distribute winnings
   ├─→ Repository.UpdateGambleStatus(completed)
   ├─→ EventBus.Publish(GambleComplete)
   └─→ StatsService.RecordEvent(GambleParticipated)
6. SSE Hub ← Broadcast gamble results
7. Discord/Streamer.bot ← Notification with results
```

### Event-Driven SSE Broadcast

```
1. Service publishes event → EventBus.Publish(event)
2. Event Bus routes to subscribers
3. SSE Event Integration receives event
4. Transform event to SSE format
5. SSE Hub broadcasts to all connected clients
6. Clients receive real-time update
```

### User Registration with Platform Linking

```
1. HTTP Request → Handler.HandleRegisterUser
2. UserService.RegisterUser()
   └─→ Repository.UpsertUser()
       ├─→ Transaction.Begin()
       ├─→ Insert/Update users table
       ├─→ Upsert platforms table
       ├─→ Upsert user_platform_links table
       └─→ Transaction.Commit()
3. HTTP Response ← User data with all linked platforms
```

## Configuration

Environment variables (`.env`):

**Database:**

- `DB_USER`: PostgreSQL username
- `DB_PASSWORD`: PostgreSQL password
- `DB_HOST`: Database host
- `DB_PORT`: Database port (default: 5433)
- `DB_NAME`: Database name (brandishbot)

**Server:**

- `PORT`: HTTP API server port (default: 8080)
- `DISCORD_PORT`: Discord bot port (default: 8082)

**Discord:**

- `DISCORD_TOKEN`: Discord bot token
- `DISCORD_GUILD_ID`: Discord server ID for slash commands

**Streamer.bot:**

- `STREAMERBOT_WS_URL`: WebSocket URL for Streamer.bot integration

**Feature Flags:**

- Various flags in `internal/features/` for enabling/disabling features

## Observability

### Logging (`internal/logger/`)

**Structured Logging with Zap:**

- JSON output for log aggregation
- Log levels: DEBUG, INFO, WARN, ERROR
- Correlation IDs for request tracing
- Context-aware logging

**Log Locations:**

- Console: Stdout for development
- File: `app.log` for persistent logs
- Request logging: HTTP middleware logs all requests with duration

**Log Fields:**

- `timestamp`: ISO8601 timestamp
- `level`: Log level
- `msg`: Log message
- `request_id`: Unique request identifier
- `user_id`: User context (when available)
- `duration_ms`: Request duration (middleware)

### Metrics (`internal/metrics/`)

**Prometheus Integration:**

- Endpoint: `GET /metrics`
- Request counts by endpoint and status code
- Request duration histograms
- Custom business metrics:
  - Gambles started/completed
  - Items crafted
  - Votes cast
  - Job level ups
  - Events published/received

### Event Logging (`internal/eventlog/`)

**Event Audit Trail:**

- All events stored in `events` table
- JSONB payload storage
- Queryable by event type, user, time range
- Supports event replay for debugging

## Utilities

### Setup (`cmd/setup/`)

### Setup (`cmd/setup/`)

Initializes database schema using SQL migrations from `migrations/`.

### Debug (`cmd/debug/`)

Dumps database contents for inspection:

- Platforms
- Users
- User-Platform Links
- Inventory
- Items
- Item Types
- Item Assignments

## Design Decisions

1. **JSONB for Inventory**: Chosen for flexibility and performance with sparse data
2. **Normalized User-Platform Links**: Supports multiple platforms per user
3. **Repository Pattern**: Decouples data access from business logic
4. **Interface-Based Services**: Enables testing and extensibility
5. **Incremental Migrations**: Uses SQL migration files for version control

## Future Considerations

Based on `AGENTS.md`:

- **Event-Driven Architecture**: Planned integration with event broker
- **Stats Service**: Will consume inventory events
- **Class Service**: Will handle XP and ability calculations

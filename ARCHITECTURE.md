# BrandishBot_Go Architecture

## Overview

BrandishBot_Go is a backend service for Streamerbot that manages user accounts, inventory, and items across multiple streaming platforms (Twitch, YouTube, Discord). The application is built using Go with a PostgreSQL database for persistence.

## Technology Stack

- **Language**: Go 1.x
- **Database**: PostgreSQL 15+
- **Database Driver**: pgx/v5
- **HTTP Server**: Standard library `net/http`
- **Environment**: `.env` file configuration

## Architecture Pattern

The application follows a **layered architecture** with clear separation of concerns:

```
┌─────────────────────────────────────┐
│         HTTP Handlers               │  ← Entry points
├─────────────────────────────────────┤
│         Service Layer               │  ← Business logic
├─────────────────────────────────────┤
│       Repository Layer              │  ← Data access
├─────────────────────────────────────┤
│         PostgreSQL                  │  ← Persistence
└─────────────────────────────────────┘
```

## Directory Structure

```
BrandishBot_Go/
├── cmd/
│   ├── app/           # Main application entry point
│   ├── setup/         # Database setup utility
│   └── debug/         # Database inspection utility
├── internal/
│   ├── database/      # Database connection
│   │   └── postgres/  # PostgreSQL repository implementations
│   ├── domain/        # Domain models (User, Item, Inventory)
│   ├── handler/       # HTTP request handlers
│   ├── server/        # HTTP server configuration
│   └── user/          # User service and interfaces
├── migrations/        # SQL migration files
└── .env              # Environment configuration
```

## Core Components

### 1. Domain Layer (`internal/domain/`)

Defines the core business entities:

- **User**: Represents a user with multi-platform support
  - Fields: `ID`, `Username`, `TwitchID`, `YoutubeID`, `DiscordID`, timestamps
  
- **Item**: Represents an in-game item
  - Fields: `ID`, `Name`, `Description`, `BaseValue`
  
- **Inventory**: User's collection of items stored as JSONB
  - Fields: `Slots` (array of `InventorySlot`)
  
- **InventorySlot**: Individual item entry
  - Fields: `ItemID`, `Quantity`

### 2. Repository Layer (`internal/database/postgres/`)

Implements data access patterns:

- **UserRepository**: Manages user and inventory data
  - `UpsertUser()`: Create or update user with platform links
  - `GetUserByPlatformID()`: Retrieve user by platform-specific ID
  - `GetUserByUsername()`: Retrieve user by username
  - `GetInventory()`: Fetch user's inventory from JSONB
  - `UpdateInventory()`: Save inventory changes
  - `GetItemByName()`: Retrieve item by name

### 3. Service Layer (`internal/user/`)

Contains business logic:

- **UserService**: Orchestrates user operations
  - `RegisterUser()`: User registration logic
  - `HandleIncomingMessage()`: Message handling and user creation
  - `FindUserByPlatformID()`: User lookup
  - `AddItem()`: Add items to user inventory

### 4. Handler Layer (`internal/handler/`)

HTTP request handlers:

- **inventory.go**: `HandleAddItem` - Add items to inventory
- **user.go**: `HandleRegisterUser` - Register/link user accounts
- **test.go**: `HandleTest` - Test endpoint for verification
- **message.go**: `HandleMessageHandler` - Process incoming messages

### 5. Server (`internal/server/`)

HTTP server setup with:
- Route registration
- Logging middleware
- Graceful shutdown support

## Database Schema

### Normalized User Structure

```sql
platforms              users                user_platform_links
┌──────────┐          ┌──────────┐          ┌────────────┐  
│platform_id│──┐   ┌──│user_id    │──┐   ┌──│user_id      │
│name       │  │   │  │username   │  │   │  │platform_id  │
└──────────┘  │   │  │created_at │  │   │  │platform_user│
              └───┼──│updated_at │  ├───┘  └────────────┘
                  └──└──────────┘  │
                                   │
                                   │
                  ┌────────────────┘
                  │
                  │  user_inventory
                  │  ┌──────────────┐
                  └──│user_id        │
                     │inventory_data │ (JSONB)
                     └──────────────┘
```

### Items System

```sql
items                  item_types            item_type_assignments
┌──────────┐          ┌──────────┐          ┌──────────┐
│item_id    │──┐   ┌──│item_type_id│──┐  ┌──│item_id    │
│item_name  │  │   │  │type_name   │  │  │  │item_type_id│
│description│  │   │  └──────────┘  ├──┘  └──┘
│base_value │  │   │                │
└──────────┘  └───┼────────────────┘
                  └─(many-to-many)
```

### Key Features

- **UUID Primary Keys**: All tables use UUID for user_id
- **JSONB Inventory**: Flexible inventory storage with GIN indexing
- **Multi-Platform Links**: Users can link multiple platform accounts
- **Item Types**: Items can have multiple types (consumable, upgradeable)

## API Endpoints

### POST /user/register
Register or link a user account.

**Request:**
```json
{
  "username": "string",
  "known_platform": "string",
  "known_platform_id": "string",
  "new_platform": "string",
  "new_platform_id": "string"
}
```

### POST /user/item/add
Add items to a user's inventory.

**Request:**
```json
{
  "username": "string",
  "platform": "string",
  "item_name": "string",
  "quantity": number
}
```

### POST /test
Test endpoint for verification.

**Request:**
```json
{
  "username": "string",
  "platform": "string",
  "platform_id": "string"
}
```

### POST /message/handle
Process incoming messages from streaming platforms.

## Data Flow Examples

### Adding an Item

```
1. HTTP Request → HandleAddItem
2. Service.AddItem()
   ├─→ GetUserByUsername()
   ├─→ GetItemByName()
   ├─→ GetInventory()
   ├─→ Update inventory slots
   └─→ UpdateInventory()
3. HTTP Response ← Success/Error
```

### User Registration

```
1. HTTP Request → HandleRegisterUser
2. Service.RegisterUser()
   └─→ UpsertUser()
       ├─→ Insert/Update users table
       ├─→ Upsert platforms table
       └─→ Upsert user_platform_links
3. HTTP Response ← User data
```

## Configuration

Environment variables (`.env`):
- `DB_USER`: PostgreSQL username
- `DB_PASSWORD`: PostgreSQL password
- `DB_HOST`: Database host
- `DB_PORT`: Database port (default: 5433)
- `DB_NAME`: Database name (brandishbot)

## Logging

- **File Logging**: All logs written to `app.log`
- **Console Logging**: Simultaneous output to stdout
- **Request Logging**: Middleware logs all HTTP requests with duration

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

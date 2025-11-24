# GoLang Migration Plan

This document outlines the proposed file structure, database design, and event-driven architecture for migrating the Streamer.bot C# project to Go.

## 1. File Structure

The proposed file structure follows the standard Go project layout:

```
/
|-- cmd/
|   `-- app/
|       `-- main.go
|-- internal/
|   |-- config/
|   |   `-- config.go
|   |-- database/
|   |   |-- postgres/
|   |   |   |-- inventory.go
|   |   |   |-- stats.go
|   |   |   `-- users.go
|   |   `-- database.go
|   |-- event/
|   |   |-- broker.go
|   |   `-- handler.go
|   |-- inventory/
|   |   `-- service.go
|   |-- stats/
|   |   `-- service.go
|   `-- user/
|       `-- service.go
|-- migrations/
|   |-- 0001_initial_schema.up.sql
|   `-- 0001_initial_schema.down.sql
|-- pkg/
|-- go.mod
|-- .env.example
`-- README.md
```

### Directory Breakdown:

*   **/cmd/app**: Contains the main application entry point.
*   **/internal/config**: Manages application configuration, loaded from environment variables or a config file.
*   **/internal/database**: Handles all database interactions. The `postgres` sub-package will contain the SQL queries for each domain.
*   **/internal/event**: Implements an event-driven architecture for decoupling services.
*   **/internal/inventory**: Contains the business logic for managing user inventories.
*   **/internal/stats**: Contains the business logic for managing user statistics.
*   **/internal/user**: Manages user-related data.
*   **/migrations**: Stores SQL migration files for creating and updating the database schema.
*   **/pkg**: For shared libraries (can be empty initially).

## 2. Database Schema

A relational database (e.g., PostgreSQL) is recommended to replace Streamer.bot's global variables. The following tables are proposed as a starting point:

### `users`

| Column | Type | Constraints |
| --- | --- | --- |
| `id` | `VARCHAR(255)` | Primary Key |
| `username` | `VARCHAR(255)` | Not Null |
| `created_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |
| `updated_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |

### `inventories`

| Column | Type | Constraints |
| --- | --- | --- |
| `user_id` | `VARCHAR(255)` | Foreign Key to `users(id)` |
| `item_name` | `VARCHAR(255)` | |
| `quantity` | `INT` | Not Null, Default `0` |
| `type` | `VARCHAR(50)` | e.g., 'item', 'material', 'active_item' |
| `created_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |
| `updated_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |

*Primary Key: (`user_id`, `item_name`)*

### `stats`

| Column | Type | Constraints |
| --- | --- | --- |
| `user_id` | `VARCHAR(255)` | Foreign Key to `users(id)` |
| `stat_name` | `VARCHAR(255)` | |
| `stat_value`| `VARCHAR(255)` | Not Null |
| `created_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |
| `updated_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |

*Primary Key: (`user_id`, `stat_name`)*

### `active_items`

| Column | Type | Constraints |
| --- | --- | --- |
| `id` | `SERIAL` | Primary Key |
| `user_id` | `VARCHAR(255)` | Foreign Key to `users(id)` |
| `item_name` | `VARCHAR(255)` | |
| `state` | `JSONB` | Not Null |
| `created_at`| `TIMESTAMPTZ` | Not Null, Default `NOW()` |

## 3. Event-Driven Architecture

An in-process event bus using Go channels is recommended for communication between services. This will decouple the domains and improve modularity.

### Components:

*   **Event Broker**: A central message bus that receives events and forwards them to registered handlers.
*   **Events**: Simple structs representing significant actions (e.g., `UserJoined`, `ItemAdded`, `StatIncremented`).
*   **Event Handlers**: Functions that subscribe to specific events and execute business logic in response.

### Example Flow:

1.  A user gives an item to another user.
2.  The `inventory` service validates the transaction and updates the database.
3.  The `inventory` service publishes an `ItemGivenEvent` to the event broker.
4.  The `stats` service, subscribed to this event, receives it and increments the "items_given" stat for the giver and "items_received" for the receiver.

This architecture replaces the need for direct calls between different parts of the code (like the `CPH.ExecuteMethod` calls in the C# project) and provides a clean, scalable way to manage data flow.

### Entry Point

The entry point to this system would be a set of exported functions or a REST/gRPC API that receives commands from the Streamer.bot environment (or any other client). These commands would then be translated into events or service calls within the Go application.

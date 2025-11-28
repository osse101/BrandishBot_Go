# Architectural Recommendations: Event System & Background Workers

**Date:** 2025-11-28
**Status:** Proposed

## Executive Summary

Based on the roadmap (Phase 2: Progression, Crafting; Phase 4: Real-time Events) and the current codebase analysis, we strongly recommend introducing both an **Internal Event System** and a **Background Worker Service**.

Currently, side effects (like engagement tracking) are handled via ad-hoc `go func()` calls. This approach is difficult to test, monitor, and scale. As the application grows to include Crafting (timers), Achievements (event-driven), and Discord Integration (async), a structured approach is required.

## 1. Internal Event System (Pub/Sub)

### Why do we need it?
The application is moving towards complex interactions where one action triggers multiple independent side effects.
*   **Example**: User crafts an item.
    *   *Core Logic*: Deduct materials, add item.
    *   *Side Effect 1*: Grant XP (Progression).
    *   *Side Effect 2*: Check "Master Crafter" achievement.
    *   *Side Effect 3*: Log analytic event.
    *   *Side Effect 4*: Notify Discord channel (if rare item).

Without an event system, the `CraftItem` handler becomes bloated with calls to `ProgressionService`, `AchievementService`, `DiscordService`, etc.

### Recommendation
Implement a synchronous (or hybrid) **Event Bus**.

*   **Structure**:
    *   `EventBus`: Central registry for subscriptions.
    *   `Events`: Typed structs (e.g., `ItemCraftedEvent`, `UserLeveledUpEvent`).
    *   `Handlers`: Functions that listen for specific events.
*   **Benefits**:
    *   **Decoupling**: The Inventory service doesn't need to know about Achievements. It just publishes `ItemCrafted`.
    *   **Testability**: You can test the Inventory service by asserting it published an event, without mocking every downstream service.
    *   **Extensibility**: Adding a new feature (e.g., "Guild Points") requires no changes to existing Inventory code, just a new Event Listener.

## 2. Background Worker Service

### Why do we need it?
1.  **Reliability**: Current `go func()` calls are unbounded. If 1000 requests come in, 1000 goroutines spawn. If the server crashes, those tasks are lost.
2.  **Timers**: The roadmap mentions "Time-based crafting". We need a system to "finish crafting" after X minutes.
3.  **Heavy Lifting**: Sending Discord messages or calculating leaderboards shouldn't block the HTTP API response.

### Recommendation
Implement a **Worker Pool** with a **Task Queue**.

*   **Phase 1 (Simple)**: In-memory worker pool.
    *   A fixed number of workers (e.g., 5) reading from a buffered channel.
    *   Prevents resource exhaustion.
*   **Phase 2 (Robust)**: Persistent Queue (e.g., Postgres-backed).
    *   Libraries like `riverqueue/river` or `hibiken/asynq` (Redis) or `sqlc` based queues.
    *   Ensures tasks survive server restarts (critical for long timers like "Crafting takes 4 hours").

### Proposed Use Cases
*   **Job: ProcessEngagement**: Move the current `engagement.go` logic here.
*   **Job: CompleteCrafting**: Scheduled task that runs when a craft finishes.
*   **Job: DiscordNotify**: Rate-limited sender for Discord messages.

## 3. Other Recommendations

### Configuration Management
*   **Viper**: As suggested in the roadmap, moving to `viper` allows for hot-reloading config and better structure as the app grows.

### Dependency Injection
*   As the number of services grows (EventBus, WorkerPool, Services), manual wiring in `main.go` might get verbose. Consider a simple DI pattern or library (like `google/wire` or just clean constructor injection) to keep `main.go` readable.

## Implementation Plan (Draft)

If approved, we will proceed with:

1.  **Create `internal/event`**: Simple Event Bus implementation.
2.  **Refactor `engagement.go`**: Use the Event Bus to publish `EngagementEvent` instead of calling service directly.
3.  **Create `internal/worker`**: Simple worker pool for processing these events asynchronously if needed.

# 🛠️ AGENTS & SERVICES Overview

This document describes the core agents, services, and communication patterns within the system. The architecture relies heavily on an **Event-Driven Architecture (EDA)** where services communicate asynchronously via an Event Broker.

## Core Communication Pattern: Event-Driven Architecture (EDA)

The system utilizes an **Event Broker** as the central message bus. Services publish **Events** (simple struct messages) to the broker, and other services act as **Event Handlers** by subscribing to relevant events and executing their business logic.

This pattern ensures **decoupling** between services, allowing them to operate independently and scale separately.

| Component | Type | Primary Function | Communication |
| :--- | :--- | :--- | :--- |
| **Main Application** | REST API | Handles user requests, authentication, and core transactional logic (e.g., Inventory updates). | **Inbound:** REST/HTTP |
| **Event Broker** | Message Bus | Receives, queues, and broadcasts Events to all registered Handlers. | Internal (Go interfaces/package) |
| **Inventory Service** | Event Publisher/Handler | Manages item ownership, validates transactions, and publishes inventory-related events. | REST (via Main App), Events (Outbound) |
| **Stats Service** | Event Handler | Listens for key events to update user statistics (e.g., counts of actions taken). | Events (Inbound) |
| **Class Service** | Service/Logic | Allocates experience points (XP) and computes the effects and power levels of in-game classes/abilities. | REST (via Main App/Other Services), Events (Inbound/Outbound) |

---

## 🔄 Detailed Agent Flows

### 1. The Transactional Flow (REST + Event Publishing)

The main application handles synchronous, critical updates using traditional **REST** calls, followed immediately by an event publication.

| Step | Agent | Action | Communication |
| :--- | :--- | :--- | :--- |
| 1. | **User/Client** | Initiates item transfer. | REST (Main Application) |
| 2. | **Main Application** | Calls `Inventory Service` to execute transfer logic. | REST |
| 3. | **Inventory Service** | **Updates Database** & publishes event. | **Publishes `ItemGivenEvent`** |

### 2. The Asynchronous Reaction Flow (Event Handling)

This flow illustrates how decoupled agents react to events without direct knowledge of the publisher.

| Step | Event | Publisher | Handlers | Handler Action |
| :--- | :--- | :--- | :--- | :--- |
| 1. | `ItemGivenEvent` | **Inventory Service** | **Stats Service** | Increments `items_given` and `items_received` counts in the database. |
| 2. | `UserJoinedEvent` | *Example Publisher* | **Class Service** | Allocates initial XP/starting class to the new user. |
| 3. | `ItemUsedEvent` | **Inventory Service** | **Class Service** | Computes if item use grants bonus XP or affects class abilities. |

---

## 🏗️ Go Implementation Notes

### Event Broker

The `Event Broker` should be implemented as a lightweight Go package or interface within the project, likely utilizing **concurrent maps** or **channels** to manage handler subscriptions and safely dispatch events to all subscribers.

```go
// EventBroker Interface Sketch
type EventBroker interface {
    // Publish sends an event to all subscribed handlers
    Publish(event Event)
    // Subscribe registers a handler function for a specific event type
    Subscribe(eventType string, handler func(event Event))
}
```

## Future Considerations

As this system grows, we will incrementally add bounded contexts:

- **Stats Service** – Listens to inventory events and recomputes user statistics asynchronously.
- **Class Service** – Listens to XP-gain events and triggers ability unlocks or level-ups.

Each service will consume an event stream published by the core inventory system, keeping them loosely coupled yet consistent.

---

## 🤖 AI Agent Best Practices

### Process Management & Cleanup

When testing the application, AI agents should follow these cleanup practices:

**Starting Background Processes:**

```powershell
# Background commands return a command ID
go run cmd/app/main.go
# Returns: Background command ID: abc123-def456-...
```

**Tracking Command IDs:**

- **ALWAYS** store the command ID returned from `run_command` when starting background processes
- Use this ID for targeted cleanup instead of searching by port/name
- Track IDs in a list throughout the session

**Cleanup Process:**

```powershell
# ✅ CORRECT: Use the tracked command ID
send_command_input(CommandId: "abc123-def456-...", Terminate: true)

# ❌ AVOID: Searching for processes by port (unreliable, can kill wrong processes)
# Get-NetTCPConnection -LocalPort 8080 | ... | Stop-Process
```

**End-of-Session Cleanup:**

- **MANDATORY**: Terminate ALL background processes at the end of testing
- Track all started command IDs during the session
- Clean up in reverse order (newest to oldest)
- Verify cleanup success with `command_status` tool

**Example Workflow:**

```powershell
1. Start server → Track: cmd_id_server
2. Run tests
3. Start debug script → Track: cmd_id_debug
4. Verify results
5. Cleanup: send_command_input(cmd_id_debug, Terminate=true)
6. Cleanup: send_command_input(cmd_id_server, Terminate=true)
7. Verify: command_status for both IDs shows "DONE"
```

**Why This Matters:**

- Prevents resource leaks
- Avoids port conflicts in future sessions
- Ensures clean state for next agent interaction
- More reliable than port-based process killing

**Key principle**: Log errors at the boundary where they occur, with full context.

### 6. Verify Fixes

After implementing a fix:

1. **Rebuild** the application
2. **Re-run** the reproduction script
3. **Check logs** for successful execution
4. **Verify** expected behavior matches actual outcome

### 7. Document Findings

Update relevant documentation:

- Fix schema mismatches in migration files or code
- Add comments explaining non-obvious error handling
- Update ARCHITECTURE.md if design assumptions were wrong

---

**Real Example Summary (lootbox1):**

| Attempt | Error Discovered | Fix Applied |
|---------|------------------|-------------|
| 1 | SQL column `p.platform_name` doesn't exist | Changed to `p.name` in queries |
| 2 | Logic error: "user not found" treated as fatal | Changed to check error message and continue |
| 3 | Missing platform data: "no rows in result set" | Identified need to seed platforms table |

**Result**: DEBUG logs provided complete visibility into the data flow, enabling rapid root cause identification.

---

## 🔒 Concurrency Best Practices

When working with this Go application, follow these guidelines to ensure thread-safety and prevent race conditions.

### Database Transactions for Atomic Operations

**Always use transactions when updating multiple related resources:**

```go
// ✅ CORRECT: Use transactions for atomic multi-resource updates
func (s *service) GiveItem(ctx context.Context, ownerUsername, receiverUsername, itemName string, quantity int) error {
    // Begin transaction
    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx) // Always defer rollback
    
    // Get both inventories within transaction
    ownerInv, _ := tx.GetInventory(ctx, ownerID)
    receiverInv, _ := tx.GetInventory(ctx, receiverID)
    
    // Modify both inventories
    // ... update logic ...
    
    // Update both within transaction
    tx.UpdateInventory(ctx, ownerID, *ownerInv)
    tx.UpdateInventory(ctx, receiverID, *receiverInv)
    
    // Commit - both succeed or both fail
    return tx.Commit(ctx)
}

// ❌ WRONG: Separate updates can leave inconsistent state
func (s *service) GiveItem(ctx context.Context, ownerUsername, receiverUsername, itemName string, quantity int) error {
    // Update owner
    s.repo.UpdateInventory(ctx, ownerID, *ownerInv)
    
    // ⚠️ If this fails, owner already lost items!
    s.repo.UpdateInventory(ctx, receiverID, *receiverInv)
}
```

When to use transactions:

- Transferring items between users (GiveItem)
- Any operation modifying multiple database rows
- Operations where partial completion would be invalid

**Repository Interface Pattern**
The repository.Tx interface lives in its own package to avoid circular dependencies:

internal/
  ├── repository/      # Transaction interface
  │   └── tx.go
  ├── user/           # Service layer (uses repository.Tx)
  │   └── service.go
  └── database/       # Implementation (implements repository.Tx)
      └── postgres/
          └── user.go

**Concurrency-Safe Patterns Already in Use**
These patterns are already thread-safe and don't need additional synchronization:

- HTTP Handlers - Each request runs in its own goroutine (built into net/http)
- Database Connection Pool - pgxpool.Pool is thread-safe
- Context Propagation - context.Context is immutable and safe for concurrent use
- Logger - log/slog is goroutine-safe
- Stateless Services - Services don't hold mutable state between requests

**When to Add Synchronization**
Protect shared mutable state with sync.RWMutex:

```go
type service struct {
    repo         Repository
    itemHandlers map[string]ItemEffectHandler
    mu           sync.RWMutex  // Protects itemHandlers
}

// Read operations use RLock (multiple readers allowed)
func (s *service) getHandler(name string) (ItemEffectHandler, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    handler, ok := s.itemHandlers[name]
    return handler, ok
}

// Write operations use Lock (exclusive access)
func (s *service) registerHandler(name string, handler ItemEffectHandler) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.itemHandlers[name] = handler
}
```

Current state: The itemHandlers map is write-once during
NewService()
, so no mutex is needed yet. Add one if dynamic registration is implemented.

**Testing for Race Conditions**
Before deploying changes that affect concurrency:

```powershell
# Run tests with race detector (Linux/Mac/Windows amd64)
go test -race ./...

# Build with race detector for debugging
go build -race cmd/app/main.go

# Run specific concurrent tests
go test -race -run TestGiveItem ./internal/user/...
```

Note: Race detector is not available on Windows/386. Test on Linux or Windows/amd64 for full verification.

### Common Pitfalls to Avoid

1. Loop Variable Capture in Goroutines

```go
// ❌ WRONG: All goroutines reference the same loop variable
for _, item := range items {
    go func() {
        process(item) // All goroutines see the last item!
    }()
}

// ✅ CORRECT: Pass as parameter or shadow
for _, item := range items {
    item := item // Shadow the loop variable
    go func() {
        process(item)
    }()
}
```

1. Closing Channels

```go
// ✅ CORRECT: Only sender closes channels
ch := make(chan int)
go func() {
    defer close(ch) // Sender closes
    for i := 0; i < 10; i++ {
        ch <- i
    }
}()

for val := range ch { // Receiver reads
    process(val)
}
```

1. Context Cancellation

```go
// ✅ CORRECT: Always check context cancellation in long operations
func (s *service) processLongOperation(ctx context.Context) error {
    for i := 0; i < 1000; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err() // Respect cancellation
        default:
            // Continue processing
        }
        
        // ... do work ...
    }
    return nil
}
```

### Graceful Shutdown

The application includes graceful shutdown to handle concurrent requests cleanly:

```go
// Server runs in goroutine
func() {
    srv.Start()
}()

// Wait for signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Shutdown with timeout for in-flight requests
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.Stop(shutdownCtx)
```

### Benefits

- In-flight HTTP requests complete (up to 30 seconds)
- Database connections close cleanly
- No orphaned goroutines or resource leaks

### Performance Considerations

Database Connection Pool Settings ([database/database.go](internal/database/database.go)):

```go
config.MaxConns = 10        // Maximum concurrent connections
config.MinConns = 2         // Minimum pooled connections
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
```

Adjust based on your deployment:

- Low traffic: MaxConns: 5-10
- High traffic: MaxConns: 20-50 (don't exceed PostgreSQL max_connections)
- CPU-bound: MaxConns ≈ NumCPU
- I/O-bound: MaxConns ≈ NumCPU * 2

### Summary Checklist

When adding new concurrent features:

- Use transactions for multi-resource updates
- Avoid shared mutable state (prefer stateless design)
- If shared state is needed, protect with sync.RWMutex
- Always defer Unlock() or Rollback() calls
- Pass contexts through the call chain
- Test with -race flag on supported platforms
- Handle graceful shutdown for new goroutines
- Document concurrency assumptions in code comments

### Testing and Debugging

Test Output Management
To keep the workspace clean and prevent console buffer issues on Windows, verbose test outputs should be redirected to files within the Output/ directory.

- Naming Convention: test_output_\<description\>.txt
- Location: Output/ (e.g., Output/test_output_final.txt)
- Usage: Redirect stdout/stderr to these files when running verbose tests (e.g., go test -v ./... > Output/test_output.txt 2>&1), then inspect the file content to diagnose failures.
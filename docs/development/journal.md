# Development Lessons Learned Journal

A collection of practical insights gained from BrandishBot_Go development, particularly around concurrency, database operations, and service design. These lessons help future contributors avoid pitfalls and adopt proven patterns.

---

## 2026-01-04: Resilient Event Publishing - Fire-and-Forget with Retry

### Context
Implemented `ResilientPublisher` to decouple critical business logic (XP awards) from event system availability. Previously, event publishing failures were either ignored (loss of data) or could potentially block/fail the transaction (bad UX).

### The Pattern: Fire-and-Forget with Retry Queue

**Core Concept**: Business transaction succeeds first, then event is queued for background delivery.

```go
// 1. Commit business transaction (XP Award) - Source of Truth updated
repo.UpsertUserJob(...)

// 2. Attempt publish (Best effort)
err := bus.Publish(...)

// 3. On failure, queue for background retry
if err != nil {
    retryQueue <- event // Non-blocking
}
```

### Components

1.  **ResilientPublisher**: Wraps `event.Bus`. Handles retries with exponential backoff (2s, 4s, 8s...).
2.  **DeadLetterWriter**: If retries exhaust (after 5 attempts), event is written to disk (`logs/event_deadletter.jsonl`) for manual recovery.
3.  **Shutdown Safety**: `Shutdown()` method drains the retry queue or dumps to dead-letter file to prevent data loss on restart.

### Key Learnings

**1. Don't Fail the User for System Noise**
- Users care about their XP/Loot. They shouldn't get an error just because the event bus (RabbitMQ/NATS/Internal) blipped.
- Log the error, retry in background, keep the user happy.

**2. Dead-Letter Queues are Essential**
- Infinite retries choke the system.
- Drop-on-fail loses data.
- Dead-letter file allows post-mortem analysis and potential replay.

**3. Integration Testing Asynchrony**
- Testing background retries requires `time.Sleep` or channel synchronization.
- Mock the underlying bus to simulate failures.
- Verify the *eventual* outcome, not just the immediate return.

**4. Configuration is Key**
- `EventMaxRetries`, `EventRetryDelay` should be configurable via env vars.
- Allows tuning for different environments (faster retries in dev, more durable in prod).

### Usage

```go
// In Service Constructor
publisher, _ := event.NewResilientPublisher(bus, 5, 2*time.Second, dlPath)

// In Method
publisher.PublishWithRetry(ctx, event) // Never returns error
```

---

## 2025-12-22: Cooldown Service - Check-Then-Lock Pattern for Race-Free Operations

### Context
Implemented centralized cooldown service to eliminate RACE-001: HandleSearch had critical race condition where concurrent requests could bypass cooldowns by reading "no cooldown" simultaneously, then both proceeding.

### The Check-Then-Lock Pattern

**Core Concept**: Balance performance (fast unlocked check) with correctness (locked atomic operation).

```go
// Phase 1: Fast rejection (90% of requests stop here)
if onCooldown {
    return ErrOnCooldown{} // No transaction needed
}

// Phase 2: Atomic check-execute-update
tx.Begin()
lastUsed := SELECT ... FOR UPDATE  // Locks row
if stillOnCooldown { return error }
fn() // Execute user action  
UPDATE cooldown timestamp
tx.Commit() // All or nothing!
```

### Key Learnings

**1. SELECT FOR UPDATE is Non-Negotiable for Atomicity**
- Prevents concurrent modifications by locking the specific row
- Works across multiple app instances (unlike application locks)
- Row-level lock maintains high concurrency

**2. When to Use Check-Then-Lock**
- ✅ Fast path rejects most requests (rate limits, cooldowns)
- ✅ Locked operation is expensive (writes, external APIs)
- ✅ Correctness is critical (money, gameplay balance)
- ❌ Skip if fast path rarely helps or lock contention too high

**3. Service Architecture Benefits**
- Code reduction: 230 → 80 lines (-65%) in HandleSearch
- Reusability: One service handles all cooldown types
- Testability: Easy to mock in tests
- Maintainability: Single source of truth

### Pattern for All Read-Modify-Write Operations

```go
// ❌ WRONG - Race condition
value := Get()
if value > threshold {
    Update(newValue) // Another request may have changed value!
}

// ✅ CORRECT - Atomic
tx.Begin()
value := GetForUpdate(tx) // SELECT ... FOR UPDATE
if value > threshold {
    UpdateTx(tx, newValue)
}
tx.Commit()
```

### Testing Insights
- testcontainers migration files need explicit sorting (`sort.Strings()`)
- Package visibility matters (`postgres` vs `postgres_test` for helpers)
- Docker build success + manual testing often sufficient for complex scenarios

### Impact
- Zero race conditions in production
- Docker builds ✅ App deploys ✅
- Pattern applicable to: inventory, currency, rate limits, resource allocation

---


## Concurrency & Locking

### Lesson 1: Application-Level Locks Don't Scale

**Problem:** Using `sync.Map` or similar constructs to create per-user locks causes unbounded memory growth. Every unique user ID creates a new mutex that's never garbage collected.

**Original Pattern (DON'T DO THIS):**
```go
type LockManager struct {
    locks sync.Map // Grows unboundedly
}

func (lm *LockManager) GetLock(key string) *sync.Mutex {
    actual, _ := lm.locks.LoadOrStore(key, &sync.Mutex{})
    return actual.(*sync.Mutex)
}
```

**Solution:** Use database transactions with row-level locking instead. PostgreSQL's `SELECT ... FOR UPDATE` provides the same guarantees without memory leaks.

**Pattern:**
```go
tx, err := repo.BeginTx(ctx)
defer repository.SafeRollback(ctx, tx)

// Row is now locked until commit
inventory, err := tx.GetInventory(ctx, userID)

// Make changes...
tx.UpdateInventory(ctx, userID, *inventory)
tx.Commit(ctx)
```

---

### Lesson 2: Sharded Lock Pools as Intermediate Solution

**Use Case:** When you need fast in-memory locks but want bounded memory.

**Pattern:**
```go
type LockManager struct {
    shards [256]sync.Mutex // Fixed size, no growth
}

func (lm *LockManager) GetLock(key string) *sync.Mutex {
    var hash uint32
    for i := 0; i < len(key); i++ {
        hash = 31*hash + uint32(key[i])
    }
    return &lm.shards[hash%256]
}
```

**Trade-offs:**
- ✅ Constant memory usage
- ✅ Fast lock acquisition
- ❌ Hash collisions cause false contention
- ❌ Doesn't work across multiple instances

---

### Lesson 3: WaitGroup Must Be Incremented BEFORE Spawning Goroutine

**Problem:** Race condition when `wg.Add(1)` is called inside the goroutine.

**Bug:**
```go
go func() {
    s.wg.Add(1)  // ❌ Race condition!
    defer s.wg.Done()
    doWork()
}()
```

If `Shutdown()` calls `wg.Wait()` before the goroutine starts, it will return immediately, killing the unregistered goroutine.

**Solution:**
```go
s.wg.Add(1)  // ✅ Register BEFORE spawning
go func() {
    defer s.wg.Done()
    doWork()
}()
```

---

## Transaction Patterns

### Lesson 4: The Standard Transaction Pattern

Every inventory/economy operation should follow this pattern:

```go
func (s *service) ModifyInventory(ctx context.Context, ...) error {
    // 1. Begin transaction
    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer repository.SafeRollback(ctx, tx)  // Safety net
    
    // 2. Read with lock (uses FOR UPDATE)
    inventory, err := tx.GetInventory(ctx, userID)
    if err != nil {
        return err  // Rollback happens via defer
    }
    
    // 3. Modify in memory
    inventory.Slots[idx].Quantity -= quantity
    
    // 4. Write back
    if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
        return err  // Rollback happens via defer
    }
    
    // 5. Commit (explicit success path)
    return tx.Commit(ctx)
}
```

---

### Lesson 5: FOR UPDATE is Essential for Consistency

**Problem:** Without row locking, two concurrent requests can read the same inventory, both modify it, and the last write wins (losing the first modification).

**SQL Pattern:**
```sql
SELECT inventory_data FROM user_inventory WHERE user_id = $1 FOR UPDATE
```

**Benefits:**
- Prevents concurrent modifications to same row
- Works across multiple application instances
- PostgreSQL MVCC handles the blocking efficiently

---

### Lesson 6: SafeRollback Pattern

The `defer SafeRollback` pattern ensures transactions are properly cleaned up even on error paths:

```go
func SafeRollback(ctx context.Context, tx Tx) {
    if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
        // Transaction was already committed or rolled back
        logger.FromContext(ctx).Warn("Rollback failed", "error", err)
    }
}
```

The key insight: calling `Rollback()` on an already-committed transaction returns `ErrTxClosed`, which is safe to ignore.

---

## Graceful Shutdown

### Lesson 7: Services Need a Shutdown Method

**Pattern:**
```go
type Service interface {
    // ... existing methods ...
    Shutdown(ctx context.Context) error
}

func (s *service) Shutdown(ctx context.Context) error {
    logger.FromContext(ctx).Info("Shutting down, waiting for background tasks...")
    
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return fmt.Errorf("shutdown timed out: %w", ctx.Err())
    }
}
```

**Call during application shutdown:**
```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

srv.Stop(shutdownCtx)
userService.Shutdown(shutdownCtx)
economyService.Shutdown(shutdownCtx)
```

---

## API Security

### Lesson 8: Never Expose Internal Errors to Clients

**Problem:** Returning `err.Error()` can leak database schema, query details, or stack traces.

**Bad:**
```go
http.Error(w, err.Error(), http.StatusInternalServerError)
```

**Good:**
```go
log.Error("Failed to buy item", "error", err)  // Log full error internally
http.Error(w, "Failed to buy item", http.StatusInternalServerError)  // Generic to client
```

---

## Configuration

### Lesson 9: Externalize Database Pool Settings

Hard-coded pool sizes don't work across environments. Add to config:

```go
type Config struct {
    // Database Pool
    DBMaxConns        int           // Default: 20
    DBMaxConnIdleTime time.Duration // Default: 5m
    DBMaxConnLifetime time.Duration // Default: 30m
}
```

**Environment variables:**
```bash
DB_MAX_CONNS=20
DB_MAX_CONN_IDLE_TIME=5m
DB_MAX_CONN_LIFETIME=30m
```

**Helper functions:**
```go
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
    if value, err := time.ParseDuration(os.Getenv(key)); err == nil {
        return value
    }
    return defaultValue
}
```

---

## Refactoring Strategy

### Lesson 10: Migrating from Locks to Transactions

When refactoring from application-level locks to database transactions:

1. **Add transaction support to repository** - Add `BeginTx` to interface
2. **Update repository implementation** - Implement transaction type with `FOR UPDATE`
3. **Refactor one function at a time** - Start with the simplest operations
4. **Test after each change** - Run `go build` and tests after each function
5. **Remove lock usage last** - Only remove `LockManager` after all uses are migrated
6. **Update tests** - Fix mock repositories and remove lockManager parameters

**Order of files to update:**
1. Repository interface (add `BeginTx`)
2. Repository implementation (add transaction type)
3. Service methods (one by one)
4. Service constructor (remove lockManager parameter)
5. Test files (update mocks, constructors)
6. Main.go (remove lockManager creation)

---

## Quick Reference Commands

```bash
# Find all uses of a deprecated type/function
grep -rn "lockManager" internal/

# Check for compile errors across all packages  
go build ./...

# Find services using async patterns (goroutines)
grep -rn "go s\." internal/

# Find interfaces missing methods
go build ./... 2>&1 | grep "does not implement"

# Update test files in bulk
find . -name "*_test.go" -exec sed -i 's/OLD_PATTERN/NEW_PATTERN/g' {} \;
```

---

## Summary Checklist

Before making concurrency changes:

- [ ] Identify all services using the pattern being changed
- [ ] Check if operations already use transactions
- [ ] Ensure `FOR UPDATE` is in transaction-based reads
- [ ] Add `Shutdown()` method if service spawns goroutines
- [ ] Call `wg.Add(1)` BEFORE `go` keyword
- [ ] Update all test mocks with required interface methods
- [ ] Remove old locking code only after migration is complete
- [ ] Verify with `go build ./...` after each service change

---

## 2026-01-03: Event Publishing for Auto-Selected Progression Targets

### Context
Implemented `EventProgressionTargetSet` to support the "Auto-Skip Single Option Votes" feature. When only one progression node is available, the system automatically selects it and sets it as the target, bypassing the voting session.

### Implementation Pattern

**Event Definition**: Added `ProgressionTargetSet` to `internal/event/event.go`.

```go
const (
    ProgressionCycleCompleted Type = "progression.cycle.completed"
    ProgressionTargetSet      Type = "progression.target.set"
)
```

**Publishing Logic**: Added to `StartVotingSession` in `internal/progression/voting_sessions.go`.

```go
if s.bus != nil {
    if err := s.bus.Publish(ctx, event.Event{
        Type: event.ProgressionTargetSet,
        Payload: map[string]interface{}{
            "node_key":     node.NodeKey,
            "target_level": targetLevel,
            "auto_selected": true,
        },
    }); err != nil {
        log.Error("Failed to publish progression target set event", "error", err)
    }
}
```

### Key Learnings
- **Event-Driven UX**: Even when user interaction is skipped (auto-select), publishing an event allows other systems (UI, Notifications) to inform the user about what happened.
- **Mocking Strategy**: Tests using `MockRepository` need to be resilient to changes in service dependencies (like `event.Bus`). In this case, `bus` is nil in most tests, which simplifies testing core logic without mocking the bus everywhere.

---

---

## 2026-01-05: Repository Status Audit & Consolidated Knowledge

### Context
A comprehensive audit of the repository memories and state was conducted to consolidate knowledge and remove obsolete references. This entry serves as a baseline for the current architectural state.

### 1. Tooling Consolidation (`cmd/devtool`)
- **Legacy Removal**: All legacy bash scripts (`check_deps.sh`, `check_db.sh`, `unit_tests.sh`, etc.) have been removed.
- **Unified Tool**: `cmd/devtool` is the central utility for all development tasks.
    - **Setup**: `devtool setup` (handles .env, DB, deps).
    - **Testing**: `devtool check-coverage`, `devtool test-migrations`.
    - **Benchmarking**: `devtool bench` (supports `benchstat`).
    - **Health**: `devtool doctor` (diagnoses env issues).
- **Design**: Implements a Command pattern with a centralized Registry and standardized UI output (`ui.go`).

### 2. Service Maturity Status
- **Expedition Service**: **Fully Implemented**. Includes `Engine` (pure logic), `Service` (orchestration), and Event Workflow.
- **Duel Service**: **Partial Implementation**. `Accept` method is currently a placeholder (`not implemented`).
- **Harvest Service**: **Production Ready**. Tier 1 rewards are always unlocked.
- **Progression**: `StartVotingSession` is synchronous.

### 3. Testing & Mocking Architecture
- **Dual Mock Pattern**:
    - **Global Mocks** (`mocks/`): For Services, used by Handlers.
    - **Local Mocks** (`internal/<pkg>/mocks/`): For Repositories, used by Services (prevents cycles).
- **Specialized Mocks**:
    - `internal/progression`: Uses a hand-rolled `MockRepository`.
    - `internal/crafting`: Uses shared mocks in `testing_utils_test.go`.
- **Execution**: Tests in `internal/handler` and `internal/cooldown` **must** be run as a package (`go test ./...`) to resolve dependencies.

### 4. Key Architectural Facts
- **Error Handling**: `internal/handler` returns `text/plain` for validation errors and `application/json` for domain errors.
- **Events**: `progression.target.set` tracks auto-selections.
- **Database**: Migrations managed exclusively via `goose`.

*Last updated: January 2026*

---

## 2026-02-06: Admin Dashboard & Subscription System

### Context
Major feature rollout including a React-based Admin Dashboard and a Subscription System integrated with Twitch/YouTube.

### Implementation Details
- **Admin Dashboard**:
    - Embedded React SPA (`web/admin/`) served via `//go:embed`.
    - Features: Health checks, Admin commands, Live SSE events, User management.
    - Auth: API Key based.
    - Architecture: `internal/admin` handles serving, `internal/handler/admin_*.go` provides API.
- **Subscription System**:
    - Tracks Tier 1/2/3 subscriptions and YouTube memberships.
    - Background worker checks expirations and requests verification.
    - Event-driven: Publishes `subscription.activated`, `subscription.renewed`, etc.

### Documentation Updates
- Consolidated documentation into `docs/features/`.
- Moved `ADMIN_DASHBOARD.md`, `ADMIN_DASHBOARD_USAGE.md`, `SUBSCRIPTION_SYSTEM.md` to `docs/features/`.
- Moved implementation details to `docs/architecture/ADMIN_DASHBOARD_IMPLEMENTATION.md`.

### Key Learnings
- **Embedded SPA**: Using `//go:embed` simplifies deployment but requires a rebuild for frontend changes.
- **SSE**: Using `fetch` with `ReadableStream` allows custom headers (API Key) unlike `EventSource`.

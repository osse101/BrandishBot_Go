# Architecture Lessons Learned Journal

A collection of architectural insights from BrandishBot_Go development, focusing on patterns that promote scalability, maintainability, and multi-instance deployments.

---

## Database Transaction Architecture

### Lesson 1: Database Transactions > Application Locks

**Key Insight:** When running multiple application instances against a shared database, application-level locks (mutexes) provide no protection.

```
┌─────────────┐    ┌─────────────┐
│ Instance A  │    │ Instance B  │
│ mutex.Lock()│    │ mutex.Lock()│ ← Different mutexes!
│     ║       │    │     ║       │
│ Read(user1) │    │ Read(user1) │ ← Both read same data
│ Modify      │    │ Modify      │
│ Write       │    │ Write       │ ← Last write wins (DATA LOSS)
└─────────────┘    └─────────────┘
```

**Solution:** Use `SELECT ... FOR UPDATE` within database transactions:

```
┌─────────────┐    ┌─────────────┐
│ Instance A  │    │ Instance B  │
│ BEGIN TX    │    │ BEGIN TX    │
│ SELECT FOR  │    │ SELECT FOR  │
│   UPDATE    │    │   UPDATE    │ ← BLOCKS until A commits
│ Modify      │    │   (waits)   │
│ COMMIT      │    │     ↓       │
│     ↓       │    │ Now proceeds│
└─────────────┘    └─────────────┘
```

---

### Lesson 2: Transaction Isolation Levels Matter

PostgreSQL's default `READ COMMITTED` is sufficient for most operations when combined with `FOR UPDATE`:

| Isolation Level | Phantom Reads | Non-Repeatable Reads | Use Case |
|-----------------|---------------|---------------------|----------|
| Read Uncommitted | Yes | Yes | Never use |
| Read Committed | Yes | No | Default, good for most ops |
| Repeatable Read | No | No | Complex multi-row reads |
| Serializable | No | No | Financial systems |

**For inventory operations:** `READ COMMITTED` + `FOR UPDATE` is optimal:
- Prevents lost updates
- Minimal locking overhead
- Good throughput

---

## Service Architecture

### Lesson 3: Repository Interface with Transaction Support

The repository interface should always support transactions:

```go
type Repository interface {
    // Standard operations (use internal connection)
    GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
    UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error
    
    // Transaction support
    BeginTx(ctx context.Context) (Tx, error)
}

type Tx interface {
    GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
    UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

**Key Design Principle:** The `Tx` interface mirrors the read/write methods but operates within a transaction context with row-level locking.

---

### Lesson 4: Service Lifecycle Management

Services that spawn background goroutines need explicit lifecycle methods:

```go
type Service interface {
    // Business operations
    DoOperation(ctx context.Context, ...) error
    
    // Lifecycle
    Shutdown(ctx context.Context) error
}
```

**Architecture Decision:** Use `sync.WaitGroup` to track active background tasks:

```
┌─────────────────────────────────────────┐
│               Service                    │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐ │
│  │ wg      │  │ repo    │  │ config  │ │
│  └────┬────┘  └─────────┘  └─────────┘ │
│       │                                  │
│  Tracks background tasks:               │
│  - XP awards                            │
│  - Event publishing                     │
│  - Async notifications                  │
└─────────────────────────────────────────┘
```

---

### Lesson 5: Dual-Layer Concurrency is Redundant

**Anti-Pattern:** Using both application locks AND database transactions.

```go
// DON'T DO THIS
lock := s.lockManager.GetLock(userID)
lock.Lock()
defer lock.Unlock()

tx, _ := s.repo.BeginTx(ctx)
// ... operations ...
```

This provides no additional safety but adds:
- Memory overhead (lock storage)
- CPU overhead (lock contention)
- Complexity (two failure modes)

**Correct Pattern:** Database transactions alone are sufficient.

---

## Configuration Architecture

### Lesson 6: Externalize All Pool/Connection Settings

Hard-coded values prevent environment-specific tuning:

```go
// DON'T
config.MaxConns = 10  // Works in dev, fails in prod

// DO
type Config struct {
    DBMaxConns        int           `env:"DB_MAX_CONNS" default:"20"`
    DBMaxConnIdleTime time.Duration `env:"DB_MAX_CONN_IDLE_TIME" default:"5m"`
    DBMaxConnLifetime time.Duration `env:"DB_MAX_CONN_LIFETIME" default:"30m"`
}
```

**Recommended Defaults:**

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| MaxConns | 10 | 20 | 50-100 |
| MaxConnIdleTime | 30m | 10m | 5m |
| MaxConnLifetime | 1h | 30m | 30m |

---

## Error Handling Architecture

### Lesson 7: Error Boundary at API Layer

Internal errors should never cross the API boundary:

```
┌─────────────────────────────────────────────────────────────┐
│                       API Handler                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   Error Boundary                      │   │
│  │  Internal Error → Log Details → Return Generic Msg   │   │
│  └─────────────────────────────────────────────────────┘   │
│                            ↓                                 │
│  Client sees: "Failed to buy item"                          │
│  Logs contain: "pq: violates foreign key constraint..."     │
└─────────────────────────────────────────────────────────────┘
```

**Pattern:**
```go
func HandleBuyItem(w http.ResponseWriter, r *http.Request) {
    result, err := service.BuyItem(ctx, ...)
    if err != nil {
        log.Error("Failed to buy item", "error", err, "user", username)
        http.Error(w, "Failed to buy item", http.StatusInternalServerError)
        return
    }
    // Success response...
}
```

---

## Scaling Considerations

### Lesson 8: Database is the Synchronization Point

For multi-instance deployments, the database serves as the single source of truth and synchronization:

```
     ┌──────────┐  ┌──────────┐  ┌──────────┐
     │Instance 1│  │Instance 2│  │Instance 3│
     └────┬─────┘  └────┬─────┘  └────┬─────┘
          │             │             │
          └──────────┬──┴─────────────┘
                     │
              ┌──────┴──────┐
              │  PostgreSQL │ ← Synchronization point
              │   (FOR UPDATE)
              └─────────────┘
```

**Implications:**
- Application can be stateless (no shared memory between instances)
- Horizontal scaling is straightforward
- Database connection pool sizing becomes critical

---

### Lesson 9: Background Tasks Need Graceful Handling

When scaling horizontally, consider:

1. **Fire-and-forget tasks** (e.g., XP awards) → Use `WaitGroup` and `Shutdown()`
2. **Scheduled tasks** (e.g., cleanup jobs) → Consider single-leader election or distributed scheduler
3. **Event processing** → Consider message queue (not just in-memory bus)

**Current Architecture (suitable for single-digit instances):**
```
Instance 1: [Job Scheduler] [Event Bus] [HTTP Server]
Instance 2: [Job Scheduler] [Event Bus] [HTTP Server]
```

**Future Architecture (for high scale):**
```
┌────────────────┐     ┌────────────────┐
│ HTTP Instances │────▶│  Message Queue │────▶ Workers
└────────────────┘     └────────────────┘
```

---

## Quick Reference

```bash
# Check for services with background goroutines
grep -rn "go s\." internal/

# Find services missing Shutdown method
grep -L "Shutdown" internal/*/service.go

# Check transaction usage
grep -rn "BeginTx\|FOR UPDATE" internal/

# Find redundant locking (anti-pattern)
grep -B5 -A5 "BeginTx" internal/ | grep -A5 -B5 "Lock()"
```

---

## Architecture Decision Records (ADRs)

| Date | Decision | Rationale |
|------|----------|-----------|
| Dec 2024 | Remove LockManager | Memory leaks, doesn't work multi-instance |
| Dec 2024 | Add FOR UPDATE | Row-level locking for consistency |
| Dec 2024 | Add Service.Shutdown() | Graceful shutdown for background tasks |
| Dec 2024 | Externalize pool config | Environment-specific tuning |

---

*Last updated: December 2024*

## Mock Generation Architecture

### Lesson 10: Dual Mock Pattern for Testing Flexibility

**Key Insight:** Different test scenarios benefit from different mock approaches. Generated mocks provide type safety and behavior verification, while stateful fakes enable integration-style testing.

**Pattern Implemented:**
```
internal/<package>/
├── repository.go           # Wrapper or local interface
├── fake_repository.go      # Optional: Stateful fake
├── mocks/
│   └── mock_repository.go  # Generated by mockery
```

**Why In-Package over Root Mocks:**
- Co-location: Mock lives with interface definition
- Ownership: Clear package boundaries
- Scalability: Better for large codebases (27+ packages)
- Navigation: Easier IDE discovery

**Configuration (`.mockery.yaml`):**
```yaml
internal/<pkg>:
  interfaces:
    Repository:
      config:
        dir: "{{.InterfaceDir}}/mocks"
        outpkg: "mocks"
        filename: "mock_repository.go"
        with-expecter: true
```

**Wrapper Interface Pattern** (for external types):
```go
// internal/user/repository.go
type Repository interface {
    repository.User  // Embed external interface
}
```

This allows mockery to generate mocks locally even when interface is defined elsewhere, avoiding import cycles.

**Benefits:**
- Zero maintenance: `make mocks` regenerates on interface changes
- Type safety: Compiler catches breaking changes
- Dual approach: Generated mocks + stateful fakes where needed
- ROI: Positive after ~6 interface changes

**Trade-offs:**
- Wrapper overhead: ~60 lines for 7 packages (negligible)
- Initial setup: 4 hours for 8 packages
- Learning curve: Team needs to understand dual approach

**Metrics (8 packages migrated):**
- Generated mocks: 6,800+ lines
- Wrapper interfaces: 60 lines total
- Duplicates removed: 11 files (~200KB)
- Estimated time saved: 30+ hours over project lifetime

---

## Architecture Decision Records (ADRs)

| Date | Decision | Rationale |
|------|----------|-----------|
| Dec 2024 | Remove LockManager | Memory leaks, doesn't work multi-instance |
| Dec 2024 | Add FOR UPDATE | Row-level locking for consistency |
| Dec 2024 | Add Service.Shutdown() | Graceful shutdown for background tasks |
| Dec 2024 | Externalize pool config | Environment-specific tuning |
| Jan 2026 | Dual Mock Pattern | Type safety + testing flexibility |
| Jan 2026 | In-package mock generation | Co-location + scalability |

---

*Last updated: January 2026*

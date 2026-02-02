# Issue: User Service Future Enhancements

**Status:** Backlog
**Priority:** Low
**Created:** 2026-01-16
**Component:** `internal/user`

## Overview

Post-refactor enhancement opportunities for the user service that are **out of scope** for the current improvement work but should be considered for future iterations.

## Enhancement Proposals

### 1. Operation Timeouts

**Problem:** Long-running inventory operations can block indefinitely.

**Solution:** Add context timeouts to operations:

```go
func (s *service) AddItem(ctx context.Context, ...) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    return s.withUserOp(ctx, ...)
}
```

**Benefits:**
- Prevents resource exhaustion
- Better failure detection
- Improved user experience (fail fast)

**Estimated Effort:** 2-3 hours

---

### 2. Distributed Tracing

**Problem:** Difficult to debug slow operations across service boundaries.

**Solution:** Add OpenTelemetry spans for each operation phase:

```go
func withUserOpResult[T any](...) (T, error) {
    ctx, span := tracer.Start(ctx, operationName)
    defer span.End()

    span.SetAttributes(
        attribute.String("platform", params.platform),
        attribute.String("username", params.username),
    )
    // ... rest of operation
}
```

**Benefits:**
- End-to-end operation visibility
- Performance bottleneck identification
- Correlation with external services (database, cache)

**Estimated Effort:** 4-6 hours
**Dependencies:** OpenTelemetry library integration

---

### 3. Operation Metrics Collection

**Problem:** No visibility into operation success rates, latencies, or patterns.

**Solution:** Add Prometheus metrics:

```go
var (
    operationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "user_inventory_operation_duration_seconds",
            Help: "Duration of inventory operations",
        },
        []string{"operation", "platform", "status"},
    )

    operationTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "user_inventory_operation_total",
            Help: "Total inventory operations",
        },
        []string{"operation", "platform", "status"},
    )
)

func withUserOpResult[T any](...) (T, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        status := "success"
        if err != nil {
            status = "error"
        }
        operationDuration.WithLabelValues(operationName, params.platform, status).Observe(duration)
        operationTotal.WithLabelValues(operationName, params.platform, status).Inc()
    }()
    // ... rest of operation
}
```

**Benefits:**
- Real-time operation monitoring
- Alerting on high error rates
- Capacity planning data
- User behavior insights

**Estimated Effort:** 3-4 hours
**Dependencies:** Prometheus metrics registry (already exists)

---

### 4. Strategy Pattern for Lookup Modes

**Problem:** As more lookup modes are added, if-else branching becomes unwieldy.

**Solution:** Implement strategy pattern:

```go
type lookupStrategy interface {
    validate(params inventoryOperationParams) error
    lookup(ctx context.Context, s *service, params inventoryOperationParams) (*domain.User, error)
    logPrefix() string
}

type platformIDStrategy struct{}
func (platformIDStrategy) validate(params inventoryOperationParams) error { ... }
func (platformIDStrategy) lookup(ctx context.Context, s *service, params inventoryOperationParams) (*domain.User, error) { ... }

type usernameStrategy struct{}
func (usernameStrategy) validate(params inventoryOperationParams) error { ... }
func (usernameStrategy) lookup(ctx context.Context, s *service, params inventoryOperationParams) (*domain.User, error) { ... }

var strategies = map[userLookupMode]lookupStrategy{
    lookupByPlatformID: platformIDStrategy{},
    lookupByUsername:   usernameStrategy{},
}
```

**Benefits:**
- Eliminates all mode-based branching
- Easy to add new lookup modes
- Each strategy is independently testable
- Cleaner separation of concerns

**When to implement:** When a 3rd lookup mode is needed (e.g., by email, by UUID)

**Estimated Effort:** 6-8 hours

---

### 5. inventoryOperationParams Builder Pattern

**Problem:** Creating `inventoryOperationParams` structs is verbose and error-prone.

**Solution:** Add fluent builder:

```go
type OperationParamsBuilder struct {
    params inventoryOperationParams
}

func NewOperationParams() *OperationParamsBuilder {
    return &OperationParamsBuilder{}
}

func (b *OperationParamsBuilder) WithPlatform(platform string) *OperationParamsBuilder {
    b.params.platform = platform
    return b
}

func (b *OperationParamsBuilder) WithPlatformID(platformID string) *OperationParamsBuilder {
    b.params.platformID = platformID
    return b
}

// ... other setters

func (b *OperationParamsBuilder) Build() inventoryOperationParams {
    return b.params
}

// Usage:
params := NewOperationParams().
    WithPlatform("twitch").
    WithPlatformID("12345").
    WithUsername("user").
    WithItem("lootbox").
    WithQuantity(1).
    Build()
```

**Benefits:**
- Self-documenting API
- Compile-time safety
- Optional parameter handling
- Reduces boilerplate

**When to implement:** When `inventoryOperationParams` grows beyond 6-7 fields

**Estimated Effort:** 3-4 hours

---

### 6. Rate Limiting per User

**Problem:** Users can spam inventory operations.

**Solution:** Add per-user rate limiting:

```go
func (s *service) withUserOp(...) error {
    // Check rate limit before operation
    if err := s.rateLimiter.Allow(ctx, params.username, operationName); err != nil {
        log.Warn("Rate limit exceeded", "username", params.username, "operation", operationName)
        return fmt.Errorf("too many requests, please try again later")
    }
    // ... rest of operation
}
```

**Benefits:**
- Prevents abuse
- Protects database from overload
- Fair resource distribution

**Estimated Effort:** 4-6 hours
**Dependencies:** Rate limiter service (e.g., Redis-based or in-memory)

---

### 7. Audit Trail for Inventory Changes

**Problem:** No historical record of inventory modifications.

**Solution:** Add audit logging:

```go
type InventoryAudit struct {
    UserID      string
    Operation   string
    ItemName    string
    Quantity    int
    Timestamp   time.Time
    Platform    string
    Metadata    map[string]interface{}
}

func (s *service) withUserOp(...) error {
    // ... execute operation
    if err == nil {
        s.auditLogger.Record(ctx, InventoryAudit{
            UserID:    user.ID,
            Operation: operationName,
            ItemName:  params.itemName,
            Quantity:  params.quantity,
            Timestamp: time.Now(),
            Platform:  params.platform,
        })
    }
    return err
}
```

**Benefits:**
- Fraud detection
- User support investigations
- Compliance requirements
- Data analytics

**Estimated Effort:** 6-8 hours
**Dependencies:** Audit log storage (database table or event stream)

---

## Prioritization Criteria

Implement enhancements when:
1. **Timeouts:** Operation latencies exceed 1s regularly
2. **Tracing:** Debugging slow operations becomes frequent
3. **Metrics:** Need production visibility for SLAs
4. **Strategy Pattern:** Adding a 3rd lookup mode
5. **Builder Pattern:** `inventoryOperationParams` grows to 7+ fields
6. **Rate Limiting:** Observing abuse or spam patterns
7. **Audit Trail:** Compliance or support requirements emerge

## Dependencies

- OpenTelemetry SDK (for tracing)
- Rate limiter service (for rate limiting)
- Audit log storage (for audit trail)

## Related Issues

- Current refactor: `docs/issues/user-service-refactor-improvement.md`
- Architecture journal: `docs/architecture/journal.md`
- Development patterns: `docs/development/journal.md`

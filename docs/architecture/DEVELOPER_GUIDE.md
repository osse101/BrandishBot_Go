# Event System Developer Guide

This guide explains how to work with events in BrandishBot.

## Table of Contents

1. [Publishing Events](#publishing-events)
2. [Subscribing to Events](#subscribing-to-events)
3. [Event Naming Conventions](#event-naming-conventions)
4. [Testing Event Handlers](#testing-event-handlers)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)

---

## Publishing Events

### Step 1: Define the Event Type

Add your event type to `internal/domain/stats.go`:

```go
const (
    // ... existing events ...
    EventMyNewFeature EventType = "my_new_feature"
)
```

**Naming Convention:** Use `snake_case`, be descriptive, use present tense verbs.

### Step 2: Publish Using ResilientPublisher

For **critical events** (affecting user state, rewards, progression):

```go
// In your service struct
type service struct {
    publisher *event.ResilientPublisher
    // ... other fields ...
}

// In your method
func (s *service) DoSomething(ctx context.Context, userID string) error {
    // ... domain logic ...
    
    // Publish event (fire-and-forget, never fails)
    s.publisher.PublishWithRetry(ctx, event.Event{
        Type: event.Type(domain.EventMyNewFeature),
        Payload: map[string]interface{}{
            "user_id": userID,
            "timestamp": time.Now(),
            "details": "whatever you need",
        },
        Metadata: map[string]interface{}{
            "source": "api",  // optional context
        },
    })
    
    return nil  // Domain operation succeeds even if event fails
}
```

For **non-critical events** (statistics, monitoring):

```go
// Can use direct Event Bus
if err := s.eventBus.Publish(ctx, event.Event{...}); err != nil {
    log.Warn("Event publish failed", "error", err)
    // Continue anyway
}
```

### Step 3: Document the Event

Add your event to [`docs/events/EVENT_CATALOG.md`](../events/EVENT_CATALOG.md) using the template.

---

## Subscribing to Events

### Step 1: Create Event Handler

```go
// internal/myservice/event_handler.go
package myservice

import (
    "context"
    "github.com/osse101/BrandishBot_Go/internal/event"
    "github.com/osse101/BrandishBot_Go/internal/domain"
    "github.com/osse101/BrandishBot_Go/internal/logger"
)

type EventHandler struct {
    service Service
}

func NewEventHandler(service Service) *EventHandler {
    return &EventHandler{service: service}
}

func (h *EventHandler) HandleMyEvent(ctx context.Context, e event.Event) error {
    log := logger.FromContext(ctx)
    
    // Extract payload
    payload, ok := e.Payload.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid payload type")
    }
    
    userID, ok := payload["user_id"].(string)
    if !ok {
        return fmt.Errorf("missing user_id in payload")
    }
    
    // Process event
    log.Info("Processing event", "user_id", userID)
    return h.service.ProcessEvent(ctx, userID)
}

// Register subscribes all handlers to the event bus
func (h *EventHandler) Register(bus event.Bus) {
    bus.Subscribe(event.Type(domain.EventMyNewFeature), h.HandleMyEvent)
}
```

### Step 2: Register in main.go

```go
// cmd/app/main.go
func main() {
    // ... initialization ...
    
    // Create service
    myService := myservice.NewService(repo)
    
    // Register event handler
    myEventHandler := myservice.NewEventHandler(myService)
    myEventHandler.Register(eventBus)
    
    // ... rest of setup ...
}
```

---

## Event Naming Conventions

### Event Type Names

**Format:** `category_action` in `snake_case`

**Good:**
- `job_level_up` - Clear action
- `item_transferred` - Descriptive
- `search_critical_success` - Specific outcome

**Bad:**
- `JobLevelUp` - Wrong case
- `job-level-up` - Wrong separator
- `lvl_up` - Too abbreviated
- `level_up_happened` - Redundant tense

### Event Categories

- **User:** `user_registered`, `user_banned`
- **Inventory:** `item_added`, `item_removed`, `item_used`
- **Economy:** `item_sold`, `item_bought`
- **Progression:** `job_level_up`, `achievement_unlocked`
- **Engagement:** `daily_streak`, `message_received`
- **Gambling:** `gamble_won`, `gamble_near_miss`
- **Crafting:** `crafting_success`, `crafting_critical_success`

### Payload Field Names

Use `snake_case` for consistency:
```go
Payload: map[string]interface{}{
    "user_id": "...",        // Good
    "new_level": 5,           // Good
    "NewLevel": 5,            // Bad - wrong case
    "new-level": 5,           // Bad - wrong separator
}
```

---

## Testing Event Handlers

### Unit Test with Mock Bus

```go
func TestMyEventHandler(t *testing.T) {
    // Create mock bus
    mockBus := &MockBus{}
    
    // Create handler
    service := myservice.NewService(mockRepo)
    handler := myservice.NewEventHandler(service)
    
    // Test event handling
    err := handler.HandleMyEvent(context.Background(), event.Event{
        Type: event.Type(domain.EventMyNewFeature),
        Payload: map[string]interface{}{
            "user_id": "test123",
        },
    })
    
    assert.NoError(t, err)
    // Assert side effects...
}
```

### Integration Test with ResilientPublisher

```go
func TestPublishWithRetry(t *testing.T) {
    // Create temp dead-letter file
    tmpFile := t.TempDir() + "/deadletter.jsonl"
    
    // Create real event bus
    bus := event.NewMemoryBus()
    
    // Create resilient publisher with short retry delay for testing
    publisher, err := event.NewResilientPublisher(
        bus, 
        3,                       // 3 retries
        100*time.Millisecond,    // 100ms base delay
        tmpFile,
    )
    require.NoError(t, err)
    defer publisher.Shutdown(context.Background())
    
    // Subscribe to event
    var received bool
    bus.Subscribe(event.Type(domain.EventMyNewFeature), func(ctx context.Context, e event.Event) error {
        received = true
        return nil
    })
    
    // Publish event
    publisher.PublishWithRetry(context.Background(), event.Event{
        Type: event.Type(domain.EventMyNewFeature),
        Payload: map[string]interface{}{"test": "data"},
    })
    
    // Wait for async processing
    time.Sleep(50 * time.Millisecond)
    
    assert.True(t, received)
}
```

### Test Event Handler Errors

```go
func TestEventHandlerError(t *testing.T) {
    bus := event.NewMemoryBus()
    
    // Subscribe handler that fails
    bus.Subscribe(event.Type(domain.EventMyNewFeature), func(ctx context.Context, e event.Event) error {
        return fmt.Errorf("intentional failure")
    })
    
    // Publish should return error (not using ResilientPublisher)
    err := bus.Publish(context.Background(), event.Event{
        Type: event.Type(domain.EventMyNewFeature),
        Payload: map[string]interface{}{},
    })
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "intentional failure")
}
```

---

## Best Practices

### ��️ DO: Use ResilientPublisher for Critical Events

```go
// ✅ Good - User state change, must be reliable
s.publisher.PublishWithRetry(ctx, event.Event{
    Type: event.Type(domain.EventJobLevelUp),
    Payload: map[string]interface{}{
        "user_id": userID,
        "new_level": level,
    },
})
```

### ❌ DON'T: Let Event Failures Break Domain Operations

```go
// ❌ Bad - Domain operation fails if event fails
if err := s.eventBus.Publish(ctx, event); err != nil {
    return fmt.Errorf("failed to level up: %w", err)  // WRONG!
}

// ✅ Good - Fire and forget
s.publisher.PublishWithRetry(ctx, event)
return nil  // Domain operation succeeds regardless
```

### ✅ DO: Include Timestamps

```go
Payload: map[string]interface{}{
    "user_id": userID,
    "timestamp": time.Now().Format(time.RFC3339),
}
```

### ✅ DO: Validate Payload in Handlers

```go
func (h *EventHandler) HandleEvent(ctx context.Context, e event.Event) error {
    payload, ok := e.Payload.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid payload type")
    }
    
    userID, ok := payload["user_id"].(string)
    if !ok || userID == "" {
        return fmt.Errorf("missing or invalid user_id")
    }
    
    // ... process event ...
}
```

### ✅ DO: Log Event Processing

```go
func (h *EventHandler) HandleEvent(ctx context.Context, e event.Event) error {
    log := logger.FromContext(ctx)
    log.Info("Processing event", "type", e.Type)
    
    // ... process ...
    
    log.Debug("Event processed successfully", "type", e.Type)
    return nil
}
```

### ❌ DON'T: Publish Events in Loops Without Rate Limiting

```go
// ❌ Bad - Can overwhelm retry queue
for _, user := range users {
    s.publisher.PublishWithRetry(ctx, event.Event{...})
}

// ✅ Better - Batch or rate limit
for i, user := range users {
    s.publisher.PublishWithRetry(ctx, event.Event{...})
    if i%100 == 0 {
        time.Sleep(10 * time.Millisecond)  // Rate limit
    }
}
```

### ✅ DO: Use Metadata for Context

```go
Payload: map[string]interface{}{
    "user_id": userID,
    "item_id": itemID,
},
Metadata: map[string]interface{}{
    "source": "api",           // Where the event came from
    "request_id": requestID,   // For tracing
    "version": "v1",           // Event schema version
}
```

---

## Troubleshooting

### Event Not Received by Subscriber

**Check:**
1. Is subscriber registered in `main.go`?
2. Is event type spelled correctly?
3. Is event actually being published? (add log before publish)
4. Is subscriber handler returning an error? (check logs)

**Debug:**
```go
// Add logging to subscriber
func (h *EventHandler) HandleEvent(ctx context.Context, e event.Event) error {
    log := logger.FromContext(ctx)
    log.Info("Event received", "type", e.Type, "payload", e.Payload)
    // ...
}
```

### Events in Dead-Letter Log

**Investigation:**
```bash
# View dead-letter log
cat logs/event_deadletter.jsonl | jq
```

**Common causes:**
- Subscriber handler has a bug (check `last_error`)
- Event payload is malformed
- External dependency (Discord, DB) is down

**Recovery:**
- Fix the bug in subscriber
- Optionally implement manual replay tool (future)

### Retry Queue Overflow

**Symptom:** Events skip retries and go straight to dead-letter

**Cause:** Retry queue buffer (1000 events) is full

**Solutions:**
1. Fix failing subscribers (reduce error rate)
2. Increase queue buffer size in `resilient_publisher.go`
3. Add backpressure/rate limiting to publishers

---

## Examples

### Complete Example: Adding Achievement Unlocked Event

**1. Define event in `internal/domain/stats.go`:**
```go
const EventAchievementUnlocked EventType = "achievement_unlocked"
```

**2. Publish from achievement service:**
```go
func (s *achievementService) UnlockAchievement(ctx context.Context, userID, achievementID string) error {
    // ... unlock logic ...
    
    s.publisher.PublishWithRetry(ctx, event.Event{
        Type: event.Type(domain.EventAchievementUnlocked),
        Payload: map[string]interface{}{
            "user_id": userID,
            "achievement_id": achievementID,
            "unlocked_at": time.Now().Format(time.RFC3339),
        },
    })
    
    return nil
}
```

**3. Create Discord subscriber:**
```go
// internal/discord/achievement_handler.go
func (h *AchievementHandler) HandleAchievementUnlocked(ctx context.Context, e event.Event) error {
    payload := e.Payload.(map[string]interface{})
    userID := payload["user_id"].(string)
    achievementID := payload["achievement_id"].(string)
    
    return h.discord.SendMessage(userID, fmt.Sprintf(
        "🏆 Achievement Unlocked: %s!", 
        achievementID,
    ))
}

func (h *AchievementHandler) Register(bus event.Bus) {
    bus.Subscribe(
        event.Type(domain.EventAchievementUnlocked),
        h.HandleAchievementUnlocked,
    )
}
```

**4. Register in `cmd/app/main.go`:**
```go
// Register achievement handler
achievementHandler := discord.NewAchievementHandler(discordBot)
achievementHandler.Register(eventBus)
```

**5. Document in [`docs/events/EVENT_CATALOG.md`](../events/EVENT_CATALOG.md)**

---

## Related Documentation

- [Event Catalog](../events/EVENT_CATALOG.md) - All event types and schemas
- [Architecture](EVENT_SYSTEM.md) - Event system architecture and design
- [Contributing Guide](../../CONTRIBUTING.md) - General development guidelines

---

## Summary

**Event Publishing:**
1. Define event type in `internal/domain/stats.go`
2. Use `ResilientPublisher.PublishWithRetry()` for critical events
3. Fire-and-forget pattern - never fail domain operations

**Event Subscribing:**
1. Create handler function with signature `func(context.Context, event.Event) error`
2. Register handler in `main.go` or event handler module
3. Validate payload and log processing

**Remember:** Events are for notifications, not critical data flow. Use them to decouple services, not to replace direct function calls.

---

## Mock Testing Framework

### Overview

BrandishBot uses a **dual mock pattern** for testing flexibility:

1. **Generated Mocks** - Type-safe, auto-generated mocks for behavior verification
2. **Stateful Fakes** - Manual in-memory implementations for integration-style tests

### When to Use Each

| Test Type | Use Generated Mock | Use Stateful Fake |
|-----------|-------------------|-------------------|
| Unit test with expectations | ✅ | ❌ |
| Integration-style test | ❌ | ✅ |
| Test error handling | ✅ | ⚠️ |
| Test complex workflows | ⚠️ | ✅ |
| Cross-package testing | ✅ | ❌ |

### Package Structure

```
internal/<package>/
├── repository.go           # Interface definition (or wrapper)
├── fake_repository.go      # Optional: Stateful fake
├── mocks/
│   └── mock_repository.go  # Generated by mockery
└── *_test.go               # Tests using either mock type
```

### Using Generated Mocks

**Example: Unit Test with Behavior Verification**

```go
package user_test

import (
    "testing"
    "github.com/osse101/BrandishBot_Go/internal/user"
    "github.com/osse101/BrandishBot_Go/internal/user/mocks"
)

func TestService_GetUser(t *testing.T) {
    // Create mock
    mockRepo := mocks.NewMockRepository(t)
    
    // Set expectations
    mockRepo.On("GetUserByID", mock.Anything, "alice").
        Return(&domain.User{Username: "alice"}, nil)
    
    // Create service with mock
    svc := user.NewService(mockRepo)
    
    // Test
    user, err := svc.GetUser(ctx, "alice")
    
    // Verify
    require.NoError(t, err)
    assert.Equal(t, "alice", user.Username)
    mockRepo.AssertExpectations(t)  // Verifies On() calls
}
```

**Benefits:**
- Type-safe: Compiler catches interface changes
- Behavior verification: Ensures methods are called with expected args
- Auto-generated: `make mocks` regenerates on interface changes

### Using Stateful Fakes  

**Example: Integration Test with State Manipulation**

```go
package user

import (
    "testing"
)

func TestService_ComplexWorkflow(t *testing.T) {
    // Create fake with initial state
    fake := NewFakeRepository()
    fake.users["alice"] = &domain.User{
        Username: "alice",
        Money: 1000,
    }
    
    // Create service with fake
    svc := NewService(fake)
    
    // Test workflow
    err := svc.BuyItem(ctx, "alice", "sword")
    require.NoError(t, err)
    
    // Verify state changes
    user := fake.users["alice"]
    assert.Equal(t, 900, user.Money)  // Spent 100
    assert.Contains(t, fake.inventories["alice"].Slots, "sword")
}
```

**Benefits:**
- Stateful: Can verify complex state transitions
- Realistic: Simulates actual repository behavior
- Flexible: Direct state manipulation for edge cases

### Adding Mocks to New Packages

**Step 1: Create Wrapper Interface (if needed)**

If your package uses an external interface (e.g., `repository.User`):

```go
// internal/mypackage/repository.go
package mypackage

import "github.com/osse101/BrandishBot_Go/internal/repository"

// Repository wraps external interface for local mock generation
type Repository interface {
    repository.MyPackage  // Embed external interface
}
```

**Step 2: Configure Mockery**

Add to `.mockery.yaml`:

```yaml
github.com/osse101/BrandishBot_Go/internal/mypackage:
  interfaces:
    Repository:
      config:
        dir: "{{.InterfaceDir}}/mocks"
        outpkg: "mocks"
        filename: "mock_repository.go"
        with-expecter: true
```

**Step 3: Generate Mocks**

```bash
make mocks
```

This creates `internal/mypackage/mocks/mock_repository.go`.

**Step 4: Use in Tests**

```go
import "github.com/osse101/BrandishBot_Go/internal/mypackage/mocks"

mockRepo := mocks.NewMockRepository(t)
```

### Common Patterns

**Pattern 1: Test Error Handling**

```go
// Using generated mock
mockRepo.On("GetUserByID", mock.Anything, "nobody").
    Return(nil, errors.New("user not found"))

user, err := svc.GetUser(ctx, "nobody")
assert.Error(t, err)
assert.Nil(t, user)
```

**Pattern 2: Test Concurrent Operations**

```go
// Using stateful fake
fake := NewFakeRepository()
fake.users["alice"] = &domain.User{Money: 1000}

// Run concurrent operations
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        svc.SpendMoney(ctx, "alice", 10)
    }()
}
wg.Wait()

// Verify final state
assert.Equal(t, 900, fake.users["alice"].Money)
```

**Pattern 3: Test Method Called Count**

```go
// Using generated mock with expecter
mockRepo.EXPECT().
    GetUserByID(mock.Anything, "alice").
    Return(user, nil).
    Times(3)  // Expect exactly 3 calls

// ... code that calls GetUserByID 3 times ...

mockRepo.AssertExpectations(t)
```

### Maintenance

**Regenerating Mocks:**
```bash
# Regenerate all mocks
make mocks

# This should be run after:
# - Changing an interface
# - Adding new interface methods
# - Removing interface methods
```

**Mock Files:**
- ✅ Auto-generated: `internal/*/mocks/mock_repository.go`
- ❌ Never edit manually (changes will be overwritten)

**Stateful Fakes:**
- ⚠️ Manual maintenance required
- Update when interface changes
- Consider: Do you really need it? (Generated mocks often sufficient)

### Best Practices

✅ **DO:**
- Use generated mocks for most tests
- Add stateful fakes only when needed
- Call `make mocks` after interface changes
- Use `mock.Anything` for context parameters
- Check expectations with `Assert Expectations(t)`

❌ **DON'T:**
- Edit generated mocks manually
- Create stateful fakes unless necessary
- Use stateful fakes for simple unit tests
- Forget to regenerate after interface changes

### Troubleshooting

**Problem: Mock not found**
```
undefined: mocks.NewMockRepository
```
**Solution:** Run `make mocks` to generate

**Problem: Mock has wrong methods**
```
mockRepo.UnknownMethod undefined
```
**Solution:** Interface changed, run `make mocks`

**Problem: Import cycle**
```
import cycle not allowed
```
**Solution:** Use wrapper interface pattern (see Step 1 above)

---


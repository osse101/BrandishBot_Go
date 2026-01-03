# Resilient Event Publishing with Retry Logic

**Priority:** HIGH  
**Complexity:** 8/10  
**Estimated Effort:** 4-6 hours  
**Created:** 2026-01-03

## Problem

Currently, when job XP awards trigger level-ups, events are published synchronously to the event bus. If event publishing fails (e.g., event bus unavailable, subscriber errors), there's no retry mechanism. This is problematic for high-cooldown actions like searches (30 minutes) because:

1. XP award succeeds and commits to database
2. Event publish fails
3. User gets XP but no level-up notification
4. No way to recover the missed event

**Impact:** Users are frustrated when they level up but receive no feedback, especially after long cooldowns.

## Current Implementation

```go
// internal/job/service.go
if leveledUp {
    s.eventBus.Publish(ctx, event.Event{ ... })  // No error handling!
}
```

## Proposed Solution

Implement **fire-and-forget with retry queue** strategy:
- XP awards always succeed (already committed to DB)
- Failed events go to async retry queue
- Background worker retries with exponential backoff
- After max retries (5), log to dead-letter file

**Key Design Decision:** Log and retry (don't fail XP awards)

## Implementation

### 1. Resilient Publisher

Create [`internal/event/resilient_publisher.go`](file:///home/osse1/projects/BrandishBot_Go/internal/event/resilient_publisher.go):

```go
package event

import (
    "context"
    "sync"
    "time"
    "github.com/osse101/BrandishBot_Go/internal/logger"
)

type ResilientPublisher struct {
    bus          Bus
    retryQueue   chan Event
    maxRetries   int           // 5
    retryDelay   time.Duration // 2s base
    wg           sync.WaitGroup
    shutdown     chan struct{}
    deadLetter   *DeadLetterWriter
}

func NewResilientPublisher(bus Bus, maxRetries int, retryDelay time.Duration, deadLetterPath string) (*ResilientPublisher, error) {
    dl, err := NewDeadLetterWriter(deadLetterPath)
    if err != nil {
        return nil, err
    }
    
    rp := &ResilientPublisher{
        bus:        bus,
        retryQueue: make(chan Event, 1000), // Buffer 1000 events
        maxRetries: maxRetries,
        retryDelay: retryDelay,
        shutdown:   make(chan struct{}),
        deadLetter: dl,
    }
    
    // Start background retry worker
    rp.wg.Add(1)
    go rp.retryWorker()
    
    return rp, nil
}

// PublishWithRetry attempts to publish, queues for retry on failure
func (rp *ResilientPublisher) PublishWithRetry(ctx context.Context, event Event) {
    if err := rp.bus.Publish(ctx, event); err != nil {
        log := logger.FromContext(ctx)
        log.Warn("Event publish failed, queuing for retry",
            "event_type", event.Type,
            "error", err)
        
        // Non-blocking send to retry queue
        select {
        case rp.retryQueue <- event:
            // Queued successfully
        default:
            log.Error("Retry queue full, event dropped to dead-letter",
                "event_type", event.Type)
            rp.deadLetter.Write(event)
        }
    }
}

func (rp *ResilientPublisher) retryWorker() {
    defer rp.wg.Done()
    
    for {
        select {
        case event := <-rp.retryQueue:
            rp.retryEvent(event, 1)
        case <-rp.shutdown:
            return
        }
    }
}

func (rp *ResilientPublisher) retryEvent(event Event, attempt int) {
    if attempt > rp.maxRetries {
        logger.Default().Error("Event retry exhausted, writing to dead-letter",
            "event_type", event.Type,
            "attempts", attempt)
        rp.deadLetter.Write(event)
        return
    }
    
    // Exponential backoff: 2s, 4s, 8s, 16s, 32s
    time.Sleep(rp.retryDelay * time.Duration(1<<(attempt-1)))
    
    ctx := context.Background()
    if err := rp.bus.Publish(ctx, event); err != nil {
        logger.Default().Warn("Event retry failed",
            "event_type", event.Type,
            "attempt", attempt,
            "error", err)
        
        // Schedule next retry
        go rp.retryEvent(event, attempt+1)
    } else {
        logger.Default().Info("Event retry succeeded",
            "event_type", event.Type,
            "attempt", attempt)
    }
}

func (rp *ResilientPublisher) Shutdown(ctx context.Context) error {
    close(rp.shutdown)
    rp.wg.Wait()
    close(rp.retryQueue)
    return rp.deadLetter.Close()
}
```

### 2. Dead-Letter Writer

Create [`internal/event/deadletter.go`](file:///home/osse1/projects/BrandishBot_Go/internal/event/deadletter.go):

```go
package event

import (
    "encoding/json"
    "os"
    "sync"
    "time"
)

type DeadLetterWriter struct {
    file *os.File
    mu   sync.Mutex
}

func NewDeadLetterWriter(path string) (*DeadLetterWriter, error) {
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    return &DeadLetterWriter{file: f}, nil
}

func (dlw *DeadLetterWriter) Write(event Event) error {
    dlw.mu.Lock()
    defer dlw.mu.Unlock()
    
    entry := map[string]interface{}{
        "timestamp": time.Now(),
        "event":     event,
    }
    
    data, _ := json.Marshal(entry)
    _, err := dlw.file.Write(append(data, '\n'))
    return err
}

func (dlw *DeadLetterWriter) Close() error {
    return dlw.file.Close()
}
```

### 3. Update Job Service

Modify [`internal/job/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/job/service.go):

```go
type Service struct {
    repo      Repository
    prog      progression.Service
    stats     stats.Service
    eventBus  event.Bus
    publisher *event.ResilientPublisher  // NEW
}

func NewService(repo Repository, prog progression.Service, stats stats.Service, bus event.Bus, deadLetterPath string) (*Service, error) {
    publisher, err := event.NewResilientPublisher(bus, 5, 2*time.Second, deadLetterPath)
    if err != nil {
        return nil, err
    }
    
    return &Service{
        repo:      repo,
        prog:      prog,
        stats:     stats,
        eventBus:  bus,
        publisher: publisher,
    }, nil
}

// In AwardXP:
if leveledUp {
    event := event.Event{
        Type: event.Type(domain.EventJobLevelUp),
        Payload: map[string]interface{}{
            "user_id":   userID,
            "job_key":   jobKey,
            "old_level": oldLevel,
            "new_level": newLevel,
            "timestamp": time.Now(),
        },
    }
    
    // Fire and forget - never fails XP award
    s.publisher.PublishWithRetry(ctx, event)
}
```

## Configuration

Add to `.env.example`:

```bash
# Event System Configuration
EVENT_MAX_RETRIES=5
EVENT_RETRY_DELAY_SECONDS=2
EVENT_DEADLETTER_PATH=./data/deadletter.jsonl
```

## Implementation Checklist

- [ ] Create `internal/event/resilient_publisher.go`
- [ ] Create `internal/event/deadletter.go`
- [ ] Update `internal/job/service.go` to use ResilientPublisher
- [ ] Update `cmd/app/main.go` to wire up ResilientPublisher
- [ ] Add configuration env vars to `.env.example`
- [ ] Create unit tests: `internal/event/resilient_publisher_test.go`
  - [ ] Test successful publish (no retry)
  - [ ] Test failed publish → retry → success
  - [ ] Test retry exhaustion → dead letter
  - [ ] Test retry queue overflow
- [ ] Create integration test: `internal/job/resilient_events_test.go`
  - [ ] Mock event bus that fails N times then succeeds
  - [ ] Award XP, verify level up committed to DB
  - [ ] Verify event eventually published after retries
- [ ] Manual testing on staging:
  - [ ] Break event bus temporarily
  - [ ] Award XP (should succeed)
  - [ ] Check logs for retry attempts
  - [ ] Restart event bus
  - [ ] Verify event delivered

## Affected Files

- [NEW] `internal/event/resilient_publisher.go`
- [NEW] `internal/event/deadletter.go`
- [NEW] `internal/event/resilient_publisher_test.go`
- [MODIFY] [`internal/job/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/job/service.go)
- [MODIFY] [`cmd/app/main.go`](file:///home/osse1/projects/BrandishBot_Go/cmd/app/main.go)
- [NEW] `internal/job/resilient_events_test.go`
- [MODIFY] `.env.example`

## Success Criteria

- ✅ XP awards never fail due to event publish errors
- ✅ Failed events retry up to 5 times with exponential backoff (2s, 4s, 8s, 16s, 32s)
- ✅ Dead-letter logging for permanently failed events
- ✅ Graceful shutdown waits for pending retries
- ✅ Zero data loss (all level-ups either published or logged)
- ✅ Comprehensive test coverage (>85%)

## Monitoring

Check dead-letter log periodically:
```bash
tail -f ./data/deadletter.jsonl
```

If events are accumulating, investigate:
- Event bus health
- Subscriber errors  
- Network issues

## Related Issues

- Code review: [code_review.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/code_review.md)
- Implementation plan: [implementation_plan.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/implementation_plan.md)
- test-resilient-publisher.md [test-resilient-publisher.md](file:///home/osse1/projects/BrandishBot_Go/docs/issues/test-resilient-publisher.md)

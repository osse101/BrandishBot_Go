# Test Resilient Event Publisher

**Status:** OPEN  
**Priority:** Medium  
**Estimated Effort:** 2-3 hours  
**Related Issues:** [resilient-event-publishing.md](./resolved/resilient-event-publishing.md)

## Context

The `ResilientPublisher` was implemented in Issue #2 to provide fire-and-forget event publishing with retry logic and dead-letter logging. While the integration with job service is tested, the `ResilientPublisher` itself lacks dedicated unit tests to verify its retry behavior, queue management, and failure handling.

## Problem Statement

The `ResilientPublisher` has non-trivial behavior that should be thoroughly tested:

1. **Retry Logic**: Events should retry with exponential backoff (2s, 4s, 8s, 16s, 32s)
2. **Dead-Letter Handling**: Events that fail after max retries should be logged to dead-letter file
3. **Queue Overflow**: When retry queue is full, events should go directly to dead-letter
4. **Graceful Shutdown**: Pending retries should be drained during shutdown
5. **Success Paths**: Events that succeed on first or subsequent attempts should work correctly

Currently, only integration-level testing exists via `TestAwardXP_PublishesEventOnLevelUp`.

## Proposed Solution

Create comprehensive unit tests in `internal/event/resilient_publisher_test.go` covering:

### Test Cases

#### 1. Successful Publish (No Retry)
- **Given**: Event published through ResilientPublisher
- **When**: Underlying bus.Publish succeeds on first attempt
- **Then**: Event is published, no retry queued, no dead-letter entry

#### 2. Failed Publish → Retry → Success
- **Given**: Event published through ResilientPublisher
- **When**: First publish fails, second attempt succeeds
- **Then**: 
  - Event is queued for retry
  - Retry happens after delay (~2s)
  - Event eventually succeeds
  - No dead-letter entry

#### 3. Retry Exhaustion → Dead Letter
- **Given**: ResilientPublisher with maxRetries=3
- **When**: Event fails all 3 retry attempts
- **Then**:
  - Event retries with exponential backoff
  - After 3rd failure, event written to dead-letter
  - Dead-letter file contains JSON entry with event details

#### 4. Retry Queue Overflow
- **Given**: ResilientPublisher with full retry queue (1000 events)
- **When**: Another event fails to publish
- **Then**:
  - Event immediately written to dead-letter (queue full)
  - Warning logged about queue overflow

#### 5. Graceful Shutdown
- **Given**: ResilientPublisher with pending retries in queue
- **When**: Shutdown is called
- **Then**:
  - Worker drains queue
  - Each pending event gets one final publish attempt
  - Failed events written to dead-letter
  - Shutdown completes within timeout

#### 6. Exponential Backoff Timing
- **Given**: Event fails multiple times
- **When**: Retries are scheduled
- **Then**:
  - Delays follow exponential pattern: 2s, 4s, 8s, 16s, 32s
  - Delays are approximately correct (±100ms tolerance for test timing)

### Implementation Approach

```go
// Use mock event bus to simulate failures
type mockBus struct {
    failCount int
    calls     []Event
}

// Test with actual ResilientPublisher and temp dead-letter file
func TestResilientPublisher_BasicRetry(t *testing.T) {
    tmpFile := t.TempDir() + "/deadletter.jsonl"
    
    // Mock bus that fails once, then succeeds
    bus := &mockBus{failCount: 1}
    
    rp, err := event.NewResilientPublisher(bus, 3, 100*time.Millisecond, tmpFile)
    require.NoError(t, err)
    defer rp.Shutdown(context.Background())
    
    // Publish event
    rp.PublishWithRetry(context.Background(), testEvent)
    
    // Wait for retry
    time.Sleep(200 * time.Millisecond)
    
    // Assert event was published after retry
    assert.Equal(t, 2, len(bus.calls)) // 1 failed + 1 retry
    
    // No dead-letter entry
    content, _ := os.ReadFile(tmpFile)
    assert.Empty(t, content)
}
```

### Additional Considerations

- **Use Short Delays in Tests**: Use 100ms base delay instead of 2s for faster tests
- **Mock Time**: Consider using time mocking for precise backoff testing
- **Concurrency**: Test with concurrent PublishWithRetry calls
- **Dead-Letter Format**: Verify JSONL format is parsable

## Success Criteria

- [ ] All 6 test cases implemented and passing
- [ ] Code coverage for `resilient_publisher.go` > 80%
- [ ] Tests run in < 5 seconds total
- [ ] Dead-letter file format validated (parsable JSON lines)
- [ ] Documentation added to test file explaining test strategy

## Future Enhancements

- Integration test simulating real event bus failures
- Dead-letter replay tool (manual recovery of failed events)
- Metrics/monitoring for retry queue depth and dead-letter rate
- Configurable retry strategies (linear, exponential, custom)

## References

- Implementation: `internal/event/resilient_publisher.go`
- Integration test: `internal/job/event_test.go:TestAwardXP_PublishesEventOnLevelUp`
- Configuration: `.env.example` (EVENT_MAX_RETRIES, EVENT_RETRY_DELAY, EVENT_DEADLETTER_PATH)

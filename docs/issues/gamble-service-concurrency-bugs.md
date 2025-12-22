# Bug Report: Gamble Service Concurrency & Security Issues

**Date**: 2025-12-22  
**Severity**: CRITICAL  
**Component**: `internal/gamble/service.go`  
**Reporter**: QA Specialist  
**Status**: Open

---

## Executive Summary

The Gamble Service contains **7 critical concurrency bugs** that can lead to:
- Race conditions causing duplicate gambles
- Inventory duplication exploits
- Data corruption
- Partial transaction failures

**Estimated Impact**: Production-blocking  
**Recommended Action**: Fix before deploying to production

---

## Critical Bug #1: Race Condition - Multiple Active Gambles 🔴

### Severity: 10/10 (CRITICAL)

### Description
Two users can simultaneously create gambles because the active gamble check is not atomic.

### Location
[`service.go:103-109`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L103-L109)

### Vulnerable Code
```go
// Check for active gamble
active, err := s.repo.GetActiveGamble(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to check active gamble: %w", err)
}
if active != nil {
    return nil, fmt.Errorf("a gamble is already active")
}
// Transaction begins AFTER the check
tx, err := s.repo.BeginTx(ctx)
```

### Reproduction Steps
1. User A calls `StartGamble` at time T
2. User B calls `StartGamble` at time T+1ms
3. Both check active gamble → both see `nil`
4. Both proceed to create gamble
5. **Result**: Two active gambles exist

### Test Case
```go
func TestStartGamble_Concurrent_RaceCondition(t *testing.T) {
    repo := new(MockRepository)
    s := NewService(repo, nil, new(MockLootboxService), nil, time.Minute, nil)
    
    ctx := context.Background()
    user1 := &domain.User{ID: "user1"}
    user2 := &domain.User{ID: "user2"}
    bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
    
    // Both see no active gamble
    repo.On("GetActiveGamble", ctx).Return(nil, nil)
    
    // Setup mocks for successful flow
    repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user1, nil)
    repo.On("GetUserByPlatformID", ctx, "twitch", "456").Return(user2, nil)
    
    tx1, tx2 := new(MockTx), new(MockTx)
    inventory := &domain.Inventory{Slots: []domain.InventorySlot{{ItemID: 1, Quantity: 5}}}
    
    repo.On("BeginTx", ctx).Return(tx1, nil).Once()
    repo.On("BeginTx", ctx).Return(tx2, nil).Once()
    tx1.On("GetInventory", ctx, "user1").Return(inventory, nil)
    tx2.On("GetInventory", ctx, "user2").Return(inventory, nil)
    tx1.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
    tx2.On("UpdateInventory", ctx, "user2", mock.Anything).Return(nil)
    tx1.On("Commit", ctx).Return(nil)
    tx2.On("Commit", ctx).Return(nil)
    tx1.On("Rollback", ctx).Return(nil).Maybe()
    tx2.On("Rollback", ctx).Return(nil).Maybe()
    repo.On("CreateGamble", ctx, mock.Anything).Return(nil)
    repo.On("JoinGamble", ctx, mock.Anything).Return(nil)
    
    // Launch concurrent StartGamble calls
    var wg sync.WaitGroup
    results := make(chan error, 2)
    
    wg.Add(2)
    go func() {
        defer wg.Done()
        _, err := s.StartGamble(ctx, "twitch", "123", "user1", bets)
        results <- err
    }()
    go func() {
        defer wg.Done()
        _, err := s.StartGamble(ctx, "twitch", "456", "user2", bets)
        results <- err
    }()
    
    wg.Wait()
    close(results)
    
    var successCount int
    for err := range results {
        if err == nil {
            successCount++
        }
    }
    
    // EXPECTED: Only 1 success
    // ACTUAL: Both succeed (BUG!)
    assert.Equal(t, 1, successCount)
}
```

### Fix Recommendation
```go
// Option 1: Database constraint
ALTER TABLE gambles ADD CONSTRAINT single_active_gamble 
    EXCLUDE USING gist (state WITH =) WHERE (state IN ('Joining', 'Opening'));

// Option 2: Pessimistic locking
tx, err := s.repo.BeginTx(ctx)
if err != nil {
    return nil, err
}
defer repository.SafeRollback(ctx, tx)

// Lock before check
active, err := tx.GetActiveGambleForUpdate(ctx) // SELECT ... FOR UPDATE
if active != nil {
    return nil, fmt.Errorf("a gamble is already active")
}
// Continue with creation...
```

---

## Critical Bug #2: Same User Can Join Multiple Times 🔴

### Severity: 9/10 (EXPLOIT)

### Description
No validation prevents a user from joining the same gamble multiple times, allowing them to gain unfair advantage.

### Location
[`service.go:194-271`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L194-L271)

### Exploit Scenario
1. Attacker starts gamble with 1 lootbox
2. Attacker joins same gamble 10 more times with 1 lootbox each
3. Attacker has 11/12 of items in pot (91% win chance if one other user joins)
4. Attacker wins and receives all items

### Test Case
```go
func TestJoinGamble_SameUserTwice_ShouldReject(t *testing.T) {
    repo := new(MockRepository)
    s := NewService(repo, nil, new(MockLootboxService), nil, time.Minute, nil)
    
    ctx := context.Background()
    gambleID := uuid.New()
    user := &domain.User{ID: "user1"}
    bets := []domain.LootboxBet{{ItemID: 1, Quantity: 1}}
    
    // Gamble already has this user
    gamble := &domain.Gamble{
        ID:           gambleID,
        State:        domain.GambleStateJoining,
        JoinDeadline: time.Now().Add(time.Minute),
        Participants: []domain.Participant{
            {UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
        },
    }
    
    repo.On("GetUserByPlatformID", ctx, "twitch", "123").Return(user, nil)
    repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
    
    // Try to join again - should fail
    err := s.JoinGamble(ctx, gambleID, "twitch", "123", "user1", bets)
    
    // EXPECTED: Error "already joined"
    // ACTUAL: No error, user joins twice (BUG!)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "already joined")
}
```

### Fix Recommendation
```go
// In JoinGamble, before transaction:
for _, p := range gamble.Participants {
    if p.UserID == user.ID {
        return fmt.Errorf("user has already joined this gamble")
    }
}

// Also add database unique constraint:
ALTER TABLE gamble_participants 
    ADD CONSTRAINT unique_user_per_gamble UNIQUE (gamble_id, user_id);
```

---

## Critical Bug #3: Inventory Duplication Exploit 🔴

### Severity: 9/10 (DATA CORRUPTION)

### Description
Same user joining from concurrent requests can bypass inventory deduction due to lack of row-level locking.

### Location
[`service.go:234-250`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L234-L250)

### Vulnerable Code
```go
// Get Inventory (no locking!)
inventory, err := tx.GetInventory(ctx, user.ID)
if err != nil {
    return fmt.Errorf("failed to get inventory: %w", err)
}

// Consume Bets
for _, bet := range bets {
    if err := consumeItem(inventory, bet.ItemID, bet.Quantity); err != nil {
        return fmt.Errorf("failed to consume bet: %w", err)
    }
}

// Update Inventory
if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
    return fmt.Errorf("failed to update inventory: %w", err)
}
```

### Scenario
1. User has 10 lootboxes
2. Request A: Read inventory → 10 boxes
3. Request B: Read inventory → 10 boxes (A hasn't committed)
4. Request A: Consume 5 → Write 5
5. Request B: Consume 5 → Write 5
6. **Result**: User joined twice but only spent 5 lootboxes (should be 10)

### Fix Recommendation
```go
// Use SELECT FOR UPDATE
inventory, err := tx.GetInventoryForUpdate(ctx, user.ID)

// Or use optimistic locking with version field
type Inventory struct {
    UserID  string
    Slots   []InventorySlot
    Version int64 // Add version field
}

// Update with version check
UPDATE inventory 
SET slots = $1, version = version + 1
WHERE user_id = $2 AND version = $3
```

---

## Critical Bug #4: ExecuteGamble Not Idempotent 🔴

### Severity: 8/10 (DUPLICATE PROCESSING)

### Description
State transition from `Joining` → `Opening` is not atomic, allowing duplicate execution.

### Location
[`service.go:294-301`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L294-L301)

### Vulnerable Code
```go
// Check state (not atomic with update!)
if gamble.State != domain.GambleStateJoining {
    return nil, fmt.Errorf("gamble is not in joining state")
}

// Separate operation - race window!
if err := s.repo.UpdateGambleState(ctx, id, domain.GambleStateOpening); err != nil {
    return nil, fmt.Errorf("failed to update state to opening: %w", err)
}
```

### Test Case
```go
func TestExecuteGamble_Concurrent_Idempotent(t *testing.T) {
    repo := new(MockRepository)
    lootboxSvc := new(MockLootboxService)
    s := NewService(repo, nil, lootboxSvc, nil, time.Minute, nil)
    
    ctx := context.Background()
    gambleID := uuid.New()
    
    gamble := &domain.Gamble{
        ID:    gambleID,
        State: domain.GambleStateJoining,
        Participants: []domain.Participant{
            {UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemID: 1, Quantity: 1}}},
        },
    }
    
    // Setup mocks to allow both executions
    repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)
    repo.On("UpdateGambleState", ctx, gambleID, domain.GambleStateOpening).Return(nil)
    
    lootboxItem := &domain.Item{ID: 1, InternalName: "box1"}
    drops := []lootbox.DroppedItem{{ItemID: 10, Quantity: 5, Value: 100}}
    repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)
    lootboxSvc.On("OpenLootbox", ctx, "box1", 1).Return(drops, nil)
    repo.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
    
    tx := new(MockTx)
    repo.On("BeginTx", ctx).Return(tx, nil)
    tx.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil)
    tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
    tx.On("Commit", ctx).Return(nil)
    tx.On("Rollback", ctx).Return(nil).Maybe()
    repo.On("CompleteGamble", ctx, mock.Anything).Return(nil)
    
    // Execute concurrently
    var wg sync.WaitGroup
    results := make(chan error, 2)
    
    wg.Add(2)
    go func() {
        defer wg.Done()
        _, err := s.ExecuteGamble(ctx, gambleID)
        results <- err
    }()
    go func() {
        defer wg.Done()
        _, err := s.ExecuteGamble(ctx, gambleID)
        results <- err
    }()
    
    wg.Wait()
    close(results)
    
    var successCount int
    for err := range results {
        if err == nil {
            successCount++
        }
    }
    
    // EXPECTED: 1 execution
    // ACTUAL: Both execute (BUG!)
    assert.Equal(t, 1, successCount)
    repo.AssertNumberOfCalls(t, "SaveOpenedItems", 1) // Should be called once
}
```

### Fix Recommendation
```go
// Compare-and-swap approach
func (r *repository) UpdateGambleStateIfMatches(
    ctx context.Context, 
    id uuid.UUID, 
    expectedState, newState domain.GambleState,
) (rowsAffected int64, error) {
    result := r.db.Exec(`
        UPDATE gambles 
        SET state = $1 
        WHERE id = $2 AND state = $3
    `, newState, id, expectedState)
    return result.RowsAffected(), result.Error
}

// Use in service:
rowsAffected, err := s.repo.UpdateGambleStateIfMatches(
    ctx, id, domain.GambleStateJoining, domain.GambleStateOpening,
)
if rowsAffected == 0 {
    return nil, fmt.Errorf("gamble already executed")
}
```

---

## Critical Bug #5: Missing Transaction Scope 🔴

### Severity: 8/10 (PARTIAL FAILURES)

### Description
`ExecuteGamble` performs multiple database operations outside a single transaction, risking partial failures.

### Location
[`service.go:274-511`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L274-L511)

### Operations NOT in Transaction
- Line 299: `UpdateGambleState` (separate call)
- Line 378: `SaveOpenedItems` (separate call)  
- Line 447-491: Winner inventory update (separate transaction)
- Line 502: `CompleteGamble` (separate call)

### Failure Scenario
1. State updated to `Opening` ✅
2. Items opened and saved ✅
3. Winner inventory update **FAILS** ❌ (disk full, constraint violation, etc.)
4. `CompleteGamble` succeeds ✅
5. **Result**: Gamble marked complete, but winner never got items!

### Impact
- Permanent item loss for winner
- No rollback mechanism
- Manual database intervention required

### Fix Recommendation
```go
func (s *service) ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error) {
    // Single transaction for all operations
    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer repository.SafeRollback(ctx, tx)
    
    // All operations use tx
    if err := tx.UpdateGambleState(ctx, id, domain.GambleStateOpening); err != nil {
        return nil, err
    }
    
    // ... open lootboxes ...
    
    if err := tx.SaveOpenedItems(ctx, allOpenedItems); err != nil {
        return nil, err
    }
    
    if err := tx.UpdateInventory(ctx, winnerID, *inv); err != nil {
        return nil, err
    }
    
    if err := tx.CompleteGamble(ctx, result); err != nil {
        return nil, err
    }
    
    // Commit all or rollback all
    if err := tx.Commit(ctx); err != nil {
        return nil, err
    }
    
    return result, nil
}
```

---

## Additional Critical Issues

### Bug #6: Missing Deadline Enforcement
**Location**: [`service.go:274`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L274)  
**Severity**: 7/10

`ExecuteGamble` doesn't check if `JoinDeadline` has passed. Admin could execute early, preventing users from joining.

**Fix**:
```go
if time.Now().Before(gamble.JoinDeadline) {
    return nil, fmt.Errorf("cannot execute before deadline")
}
```

### Bug #7: consumeItem Slice Mutation
**Location**: [`service.go:532`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L532)  
**Severity**: 7/10

Removing items during slice iteration can cause index issues when consuming multiple items.

**Test Case**:
```go
func TestConsumeItem_MultipleItemsRemoval(t *testing.T) {
    inventory := &domain.Inventory{
        Slots: []domain.InventorySlot{
            {ItemID: 1, Quantity: 5},
            {ItemID: 2, Quantity: 3},
            {ItemID: 3, Quantity: 2},
        },
    }
    
    consumeItem(inventory, 1, 5) // Removes slot
    consumeItem(inventory, 2, 3) // Removes slot
    consumeItem(inventory, 3, 2) // Removes slot
    
    // EXPECTED: Empty inventory
    assert.Empty(t, inventory.Slots)
}
```

### Bug #8: No Lootbox Type Validation
**Location**: [`service.go:134`](file:///home/osse1/projects/BrandishBot_Go/internal/gamble/service.go#L134)  
**Severity**: 7/10

Users can bet non-lootbox items (swords, armor, etc.).

**Fix**:
```go
for _, bet := range bets {
    item, err := s.repo.GetItemByID(ctx, bet.ItemID)
    if item.Type != domain.ItemTypeLootbox {
        return nil, fmt.Errorf("item %d is not a lootbox", bet.ItemID)
    }
}
```

---

## Recommended Action Plan

1. **Immediate** (before production):
   - Fix race condition in `StartGamble` (add database constraint)
   - Prevent duplicate user joins (validation + constraint)
   - Fix `ExecuteGamble` idempotency (CAS operation)

2. **High Priority** (this sprint):
   - Wrap `ExecuteGamble` in single transaction
   - Add row-level locking for inventory
   - Validate lootbox item types

3. **Medium Priority** (next sprint):
   - Fix `consumeItem` slice mutation
   - Add deadline enforcement
   - Implement gamble expiration/cleanup

4. **Testing** (continuous):
   - Add concurrency tests to CI/CD
   - Load test with 100+ concurrent users
   - Chaos engineering for transaction failures

---

## Test File Location
See complete test suite: `/home/osse1/.gemini/antigravity/brain/e09da51e-04ff-4f54-8f3d-c33d70769a64/gamble_critical_tests_example.go`

## Full Audit Report
See detailed audit: `/home/osse1/.gemini/antigravity/brain/e09da51e-04ff-4f54-8f3d-c33d70769a64/gamble_service_audit.md`

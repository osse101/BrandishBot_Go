# HandleSearch Cooldown Race Condition

**Issue ID:** `RACE-001`  
**Severity:** Medium  
**Component:** User Service - Search Feature  
**Status:** Open  
**Created:** 2025-12-22

---

## Summary

The `HandleSearch` method has a race condition where concurrent requests from the same user can bypass cooldown checks, allowing users to receive duplicate rewards by rapid-firing search commands.

---

## Problem Description

### Current Implementation

```go
func (s *service) HandleSearch(ctx, platform, platformID, username) (string, error) {
    user := getUserOrRegister(...)
    
    // 1. Check cooldown (unlocked read)
    lastUsed := repo.GetLastCooldown(user.ID, "search")
    if onCooldown(lastUsed) {
        return "cooldown active"
    }
    
    // 2. Process search and maybe add reward
    if roll <= threshold {
        tx := repo.BeginTx()
        inventory := tx.GetInventory(user.ID)
        // ... add reward ...
        tx.UpdateInventory(user.ID, inventory)
        tx.Commit()
    }
    
    // 3. Update cooldown (OUTSIDE transaction)
    repo.UpdateCooldown(user.ID, "search", now)
}
```

### The Race Condition

**Timeline:**
```
T1: User sends search request #1
T2: User sends search request #2 (within cooldown period)

Request #1                          Request #2
─────────────────────────────────────────────────────────────
GetLastCooldown() → nil             
(no cooldown yet)                   GetLastCooldown() → nil
                                    (still no cooldown!)
Process search → success            
Add reward to inventory             Process search → success
                                    Add reward to inventory
UpdateCooldown(now)                 
                                    UpdateCooldown(now)

Result: User gets 2x rewards! ❌
```

### Impact

- **Exploit Potential:** Users can bypass 5-minute cooldown by rapid-firing requests
- **Economy Impact:** Duplicate lootbox rewards inflate economy
- **Frequency:** Low under normal usage, high if discovered
- **Detection:** Hard to distinguish from network latency

---

## Root Cause

The cooldown check (step 1) and cooldown update (step 3) are **not atomic**. Between checking and updating, another request can slip through.

---

## Proposed Solutions

### Solution A: Full Transactional Wrapper ✅ Most Correct

**Wrap entire search execution in transaction:**

```go
func (s *service) HandleSearch(...) (string, error) {
    user := getUserOrRegister(...)
    
    // PHASE 1: Cheap check (fast rejection)
    lastUsed := repo.GetLastCooldown(user.ID, "search")  // Unlocked
    if onCooldown(lastUsed) {
        return "cooldown active"  // ~90% of requests exit here
    }
    
    // PHASE 2: Transaction with locking
    tx := repo.BeginTx()
    defer tx.Rollback()
    
    // Locked recheck (catches race conditions)
    lastUsed := repo.GetLastCooldownForUpdate(tx, user.ID, "search")  // FOR UPDATE
    if onCooldown(lastUsed) {
        return "cooldown active"  // Race detected, reject
    }
    
    // Process search (within transaction)
    if roll <= threshold {
        inventory := tx.GetInventory(user.ID)
        // ... add reward ...
        tx.UpdateInventory(user.ID, inventory)
    }
    
    // Update cooldown (within transaction)
    tx.UpdateCooldownTx(user.ID, "search", now)
    
    tx.Commit()  // All or nothing!
}
```

**Pros:**
- ✅ Completely eliminates race condition
- ✅ ACID guarantees (atomic cooldown + reward)
- ✅ Industry-standard "check-then-lock" pattern

**Cons:**
- ⚠️ Requires significant refactoring (~100 lines)
- ⚠️ Need to handle all search outcomes in transaction
- ⚠️ More complex error handling

**Performance:**
- Cheap check rejects ~90% of requests (1ms)
- Only off-cooldown requests pay transaction cost (5-10ms)
- Net improvement due to early rejection

---

### Solution B: Simplified (Success-Only Locking) ⚡ Pragmatic

**Only wrap SUCCESS case:**

```go
if roll <= threshold {
    tx := repo.BeginTx()
    
    // Locked cooldown check BEFORE adding reward
    lastUsed := repo.GetLastCooldownForUpdate(tx, user.ID, "search")
    if onCooldown(lastUsed) {
        return "cooldown active"
    }
    
    // Add reward
    inventory := tx.GetInventory(user.ID)
    // ... add lootbox ...
    tx.UpdateInventory(user.ID, inventory)
    
    // Update cooldown
    tx.UpdateCooldownTx(user.ID, "search", now)
    
    tx.Commit()
}
// Failures still update cooldown outside transaction (acceptable)
```

**Pros:**
- ✅ Prevents critical exploit (item duplication)
- ✅ Smaller refactoring (~30 lines)
- ✅ Low risk

**Cons:**
- ⚠️ Still allows cooldown bypass on failures (minor)

---

### Solution C: Redis-Based Cooldowns 🚀 Scalable

**Use Redis for atomic cooldown tracking:**

```go
key := fmt.Sprintf("cooldown:search:%s", user.ID)
if redis.Exists(key) {
    return "cooldown active"
}

// Process search...

// Set cooldown atomically
redis.SetEX(key, domain.SearchCooldownDuration, "1")
```

**Pros:**
- ✅ Naturally atomic (SETNX operation)
- ✅ Better performance than DB locks
- ✅ Easier to scale horizontally

**Cons:**
- ⚠️ Requires Redis infrastructure
- ⚠️ Data split between DB and Redis
- ⚠️ Bigger architectural change

---

## Recommended Approach

**Option A (Full Transactional)** is recommended for correctness and robustness. The "check-then-lock" pattern is industry-standard for this exact problem.

**Implementation Notes:**

1. Already have required repository methods:
   - `GetLastCooldownForUpdate(tx, userID, action)` ✅
   - `UpdateCooldownTx(tx, userID, action, time)` ✅

2. Testing requirements:
   - Add `TestHandleSearch_ConcurrentRequests_Integration`
   - Spawn goroutines, verify only one succeeds
   - Similar to existing `TestConcurrentAddItem_Integration`

3. Performance validation:
   - Measure before/after latency
   - Verify cheap check still rejects fast

---

## Testing Strategy

### Integration Test

```go
func TestHandleSearch_ConcurrentCooldownBypass_Integration(t *testing.T) {
    // Setup user
    user := createTestUser()
    
    // Fire 10 concurrent searches
    results := make(chan string, 10)
    for i := 0; i < 10; i++ {
        go func() {
            msg, _ := service.HandleSearch(ctx, "twitch", user.ID, "testuser")
            results <- msg
        }()
    }
    
    // Collect results
    successes := 0
    for i := 0; i < 10; i++ {
        msg := <-results
        if strings.Contains(msg, "found") {
            successes++
        }
    }
    
    // Only ONE should succeed despite 10 concurrent requests
    assert.Equal(t, 1, successes)
}
```

---

## References

- Transaction audit: `/home/osse1/.gemini/antigravity/brain/.../transaction_audit.md`
- Similar fix: `MergeUsersInTransaction` (atomic user merge)
- Pattern: Check-then-lock (optimistic concurrency control)

---

## Related Issues

- All methods using cooldowns should be audited similarly
- Consider extracting cooldown logic to dedicated service

---

## Timeline Estimate

- **Solution A:** 2-3 hours (refactoring + testing)
- **Solution B:** 1 hour (focused fix)
- **Solution C:** 4-6 hours (infrastructure + migration)

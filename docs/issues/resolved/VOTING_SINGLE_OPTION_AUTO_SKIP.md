RESOLVED

# Auto-Skip Single Option Votes

## Issue

When only one node is available for voting (e.g., `progression_system` at fresh start), the system still creates a voting session with 1 option and waits for votes/admin intervention.

This creates unnecessary friction:
- Users can only vote for one thing
- Admin must manually end a pointless vote
- Delays progression unnecessarily

## Current Behavior

```
Fresh Start:
  ↓
Admin: /progression start-voting
  ↓
GetAvailableUnlocks() → [progression_system] (only 1 node available)
  ↓
CreateVotingSession with 1 option
  ↓
Users vote (but there's only 1 choice!)
  ↓
Admin: /progression end-voting
  ↓
progression_system becomes target
```

## Expected Behavior

```
Fresh Start:
  ↓
Admin: /progression start-voting (or automatic on first contribution)
  ↓
GetAvailableUnlocks() → [progression_system] (only 1 node)
  ↓
IF options.length == 1:
  Skip voting, immediately set as target
  Log: "Only one option available, auto-selected: progression_system"
ELSE:
  Create voting session normally
```

## Implementation

### Location
`internal/progression/voting_sessions.go` - `StartVotingSession()`

### Pseudo-code

```go
func (s *service) StartVotingSession(ctx context.Context, unlockedNodeID *int) error {
    // ... existing code to get available nodes ...
    
    available, err := s.GetAvailableUnlocks(ctx)
    if err != nil {
        return err
    }
    
    if len(available) == 0 {
        return fmt.Errorf("no nodes available")
    }
    
    // NEW: Auto-skip if only one option
    if len(available) == 1 {
        log.Info("Only one option available, auto-selecting",
            "nodeKey", available[0].NodeKey)
        
        // Set as unlock target immediately
        progress, _ := s.repo.GetActiveUnlockProgress(ctx)
        if progress != nil {
            err := s.repo.SetUnlockTarget(ctx, progress.ID, available[0].ID, 1)
            if err != nil {
                return err
            }
            
            // Cache the unlock cost
            s.mu.Lock()
            s.cachedTargetCost = available[0].UnlockCost
            s.cachedProgressID = progress.ID
            s.mu.Unlock()
        }
        
        return nil // Skip voting session creation
    }
    
    // Continue with normal voting session...
}
```

## Edge Cases

1. **Bootstrap with 0 contributions**: Auto-select shouldn't unlock immediately, just set target
2. **Already enough contributions**: If contributions >= cost, unlock should trigger naturally on next contribution
3. **Event publishing**: Publish a "TargetSet" event even without voting.

## Testing

- [ ] Test fresh start with 1 available node
- [ ] Test with 2+ available nodes (normal voting)
- [ ] Test auto-select doesn't trigger unlock prematurely
- [ ] Test cache is populated correctly
- [ ] Test event publishing (if implemented)

## Priority

**Medium** - UX improvement, not critical but reduces admin burden

## Related

- Tree design issue: `progression_system` should probably not be a node, just default unlocked
- Or: Move to JSON config where we can mark nodes as "start_unlocked"

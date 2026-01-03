RESOLVED

# Refactor Username-Based Inventory Methods

**Priority:** HIGH  
**Complexity:** 7/10  
**Estimated Effort:** 3-4 hours  
**Created:** 2026-01-03

## Problem

Username-based inventory methods (`AddItemByUsername`, `RemoveItemByUsername`, `UseItemByUsername`, `GetInventoryByUsername`, `GiveItemByUsername`) contain ~500 lines of duplicated code with their platform ID counterparts. The only difference is the user lookup method:
- Platform ID methods: Use `getUserOrRegister()` (auto-registration)
- Username methods: Use `GetUserByPlatformUsername()` (lookup only)

The transaction logic for inventory operations is identical between both variants, violating DRY principles.

## Example of Duplication

```go
// AddItem (by platform ID)
func (s *service) AddItem(...) error {
    user, err := s.getUserOrRegister(...)  // Only difference
    // ... 40 lines of identical transaction logic ...
}

// AddItemByUsername
func (s *service) AddItemByUsername(...) error {
    user, err := s.repo.GetUserByPlatformUsername(...)  // Only difference
    // ... 40 lines of identical transaction logic ...
}
```

This pattern repeats across 5 method pairs.

## Proposed Solution

Extract common transaction logic into internal helper methods, then refactor public methods to call helpers.

### Step 1: Create Internal Helpers

Add to [`internal/user/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go):

```go
// Internal helper: Add item to a user's inventory (no user lookup)
func (s *service) addItemToUserInternal(ctx context.Context, user *domain.User, itemName string, quantity int) error {
    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer repository.SafeRollback(ctx, tx)
    
    item, err := s.getItemByNameCached(ctx, itemName)
    if err != nil {
        return err
    }
    if item == nil {
        return fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
    }
    
    inventory, err := tx.GetInventory(ctx, user.ID)
    if err != nil {
        return fmt.Errorf("failed to get inventory: %w", err)
    }
    
    // Add item logic
    found := false
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            inventory.Slots[i].Quantity += quantity
            found = true
            break
        }
    }
    if !found {
        inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity})
    }
    
    if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
        return fmt.Errorf("failed to update inventory: %w", err)
    }
    
    return tx.Commit(ctx)
}
```

Create similar helpers:
- `removeItemFromUserInternal(user, itemName, quantity)`
- `useItemInternal(user, itemName, quantity, targetUsername)`
- `getInventoryInternal(user, filter)`

### Step 2: Refactor Public Methods

```go
// Platform ID version (with auto-registration)
func (s *service) AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error {
    user, err := s.getUserOrRegister(ctx, platform, platformID, username)
    if err != nil {
        return err
    }
    return s.addItemToUserInternal(ctx, user, itemName, quantity)
}

// Username version (lookup only, no registration)
func (s *service) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
    user, err := s.repo.GetUserByPlatformUsername(ctx, platform, username)
    if err != nil {
        return err
    }
    return s.addItemToUserInternal(ctx, user, itemName, quantity)
}
```

## Implementation Checklist

- [ ] Create `addItemToUserInternal` helper
- [ ] Create `removeItemFromUserInternal` helper
- [ ] Create `useItemInternal` helper
- [ ] Create `getInventoryInternal` helper
- [ ] Refactor `AddItem` and `AddItemByUsername` to use helper
- [ ] Refactor `RemoveItem` and `RemoveItemByUsername` to use helper
- [ ] Refactor `UseItem` and `UseItemByUsername` to use helper
- [ ] Refactor `GetInventory` and `GetInventoryByUsername` to use helper
- [ ] Verify `GiveItem` and `GiveItemByUsername` already share `executeGiveItemTx` ✅
- [ ] Run all tests: `go test ./internal/user/... -v`
- [ ] Run integration tests: `go test ./internal/database/postgres/... -v -tags=integration`
- [ ] Run benchmarks to verify no performance regression: `go test ./internal/user/... -bench=. -benchmem`
- [ ] Update documentation if needed

## Affected Files

- [`internal/user/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go) - Main refactoring (~400-500 lines reduction)
- [`internal/user/service_test.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service_test.go) - Verify no regressions
- [`internal/user/username_methods_test.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/username_methods_test.go) - Verify no regressions
- [`internal/database/postgres/user_service_integration_test.go`](file:///home/osse1/projects/BrandishBot_Go/internal/database/postgres/user_service_integration_test.go) - Verify no regressions

## Success Criteria

- ✅ All tests pass
- ✅ ~400-500 lines of code removed
- ✅ No performance degradation in benchmarks
- ✅ Transaction logic centralized in 4-5 internal helpers
- ✅ Public methods are thin wrappers (5-10 lines each)

## Benefits

1. **Maintainability:** Bugs fixed once instead of twice
2. **Consistency:** Guaranteed identical behavior between variants
3. **Readability:** Public methods clearly show their purpose (lookup vs auto-register)
4. **Testability:** Can test transaction logic independently of user lookup

## Related Issues

- Code review: [code_review.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/code_review.md)
- Implementation plan: [implementation_plan.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/implementation_plan.md)

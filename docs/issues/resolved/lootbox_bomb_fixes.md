# Lootbox Grouping and Bomb Usage Message Fix

**Status**: RESOLVED
**Priority**: Low
**Category**: Bug Fix / Polish
**Resolved**: 2026-04-04 (Commit `b53e095`)

## Problem

1. **Junkboxes:** When opening junkboxes, money drops with different flavor names were not being grouped together properly in the output message, leading to messy formatting. They should be merged under a single generic "money" display name before aggregation.
2. **Bomb Usage Message:** The message displayed when setting a bomb was outdated ("Waiting for a crowd...").

These items were tracked in `docs/issues/todo.txt`.

## Solution

1. Modified `aggregateDropsAndUpdateInventory` in `internal/itemhandler/lootbox.go` to explicitly check if an item is `domain.ItemMoney`. If so, it fetches the display name without appending the quality level, ensuring all money drops group under a single generic name in the display map.
2. Updated the return message in `handleBomb` inside `internal/itemhandler/bomb.go` to be more descriptive: "It will detonate when the crowd gathers...".

## Related Files
- `internal/itemhandler/lootbox.go`
- `internal/itemhandler/bomb.go`
- `docs/issues/todo.txt` (items removed)
# Bomb Handler Refactor

## Issue
Bomb handler was originally part of weapon.go and weapon handlers had out of date text.

## Resolution
The Bomb Handler was moved out of `internal/itemhandler/weapon.go` to its own dedicated file `internal/itemhandler/bomb.go`.

The Lootbox drop function was also renamed from `ProcessLootbox` to `HandleLootbox` and `ProcessLootboxDrops` to `HandleLootboxDrops` for consistency in naming conventions within `itemhandler`.

The deploy script was updated, and user platform handling was simplified.

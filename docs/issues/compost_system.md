# Issue: Compost System Implementation

## Description

The Compost system, intended for recycling items into gems/resources, is currently incomplete. The service interface is defined, but core methods contain placeholders.

## Status

**Implementation Status: In Progress**

- **Service**: `internal/compost/service.go` exists.
- **Deposit**: Contains TODOs for inventory validation and removal.
- **Harvest**: Returns `not implemented`.
- **Status**: Seemingly implemented (`GetStatus`).

## Missing Components

1.  **Inventory Consumption**: `Deposit` method needs to actually remove items from user inventory.
2.  **Validation**: Ensure user owns the items.
3.  **Reward Logic**: `Harvest` needs to calculate rewards (Gems?) based on item rarity and quantity.
4.  **Ready Logic**: `Deposit` sets a hardcoded 24h timer. This should likely be dynamic based on item type/rarity.

## Location

- `internal/compost/service.go`

## Action Items

- [ ] Implement `Deposit` logic (inventory check/remove).
- [ ] Implement `Harvest` logic (reward calculation, database update).
- [ ] Implement dynamic composting times (if required).
- [ ] Verify `GetStatus` accuracy.

## Market balance event

As items are sold and bought, track the quantities of individual items being exchanged. When the net quantity of transactions passes a threshold, emit an event either increasing or decreasing the market value of the item. TBD if this is permanent.

## Status Update (2026-01-30)

**Status**: Open (Not Implemented).

- `internal/economy/service.go` implements basic buy/sell logic with fixed price ratios (modified by progression bonuses).
- No logic exists for tracking net transaction quantities or adjusting market values based on supply/demand events.

## Status Update (2026-02-15)

**Status**: Open (Not Implemented).

- Verified that `internal/economy/service.go` still relies on fixed base prices and progression modifiers.
- No dynamic market balancing logic found.

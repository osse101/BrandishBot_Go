# Issue: Placeholder Implementation of Major Gameplay Systems

## Description

The Duel and Expedition systems are currently functionally blocked by "not implemented" placeholders in core service methods. While the infrastructure (migrations, routes, basic service structure) is present, the actual game logic for resolving these events is missing.

### 1. Duel System Incomplete

The `internal/duel/service.go` method `Accept` is a placeholder.

- **Impact**: Users can challenge each other and have their currency/items deducted, but the duel can never be accepted or resolved. This leads to stuck game states and lost user currency.
- **Root Cause**: Missing coin-flip/dice-roll logic and winner awarding logic.
- **Location**: `internal/duel/service.go:90`.

### 2. Expedition System Incomplete

The `internal/expedition/service.go` method `ExecuteExpedition` is a placeholder.

- **Impact**: Campaigns/Expeditions can be started and joined, but they can never complete or award rewards.
- **Root Cause**: Missing loot table resolution and participant reward distribution logic.
- **Location**: `internal/expedition/service.go:153`.

## Proposed Solution

- Implement the resolution logic for Duels including random winner selection and reward distribution.
- Implement the Expedition execution logic, integrating with the `LootboxService` and handling multi-participant reward splits.
- Add background workers or scheduler jobs to handle timeouts/expirations for both systems (ref: `gamble_worker.go`).

# Issue: Placeholder Implementation of Major Gameplay Systems

## Description

The Duel and Compost systems were functionally blocked by "not implemented" placeholders in core service methods. This tracking issue documents the resolution status of these systems.

### 1. Duel System Incomplete

The `internal/duel/service.go` method `Accept` is a placeholder.

- **Impact**: Users can challenge each other and have their currency/items deducted, but the duel can never be accepted or resolved. This leads to stuck game states and lost user currency.
- **Root Cause**: Missing coin-flip/dice-roll logic and winner awarding logic.
- **Location**: `internal/duel/service.go:90`.

### 2. Compost System Incomplete (Resolved)

Previously incomplete, the Compost system (`internal/compost/`) is now fully implemented.

### 3. Expedition System (Resolved)

Previously incomplete, the Expedition system (`internal/expedition/service.go`) is now fully implemented with `ExecuteExpedition` logic and background workers.

## Proposed Solution

- Implement the resolution logic for Duels including random winner selection and reward distribution.
- ~~Implement the Compost system for item recycling.~~ (Done)
- ~~Add background workers or scheduler jobs to handle timeouts/expirations for Expeditions.~~ (Done)

## Status Update (2026-01-30)

Verified that `internal/duel/service.go` (`Accept`) and `internal/expedition/service.go` (`ExecuteExpedition`) still return "not implemented" errors. The issue persists.

## Status Update (2026-02-06)

- **Expeditions**: `ExecuteExpedition` is now implemented. Issue resolved for Expeditions.
- **Duels**: Still incomplete.
- **Compost**: Identified as incomplete. Added to this tracking issue.

## Status Update (2026-02-15)

- **Compost**: Fully implemented (`internal/compost/service.go`, `deposit.go`, `harvest.go`, `engine.go`). Issue resolved for Compost.
- **Duels**: Still incomplete (`Accept` returns "not implemented").

## Status Update (2026-02-28)

- **Compost**: Resolved. Verified implementation in `internal/compost/` including service lifecycle, engine logic, and database integration.
- **Expeditions**: Resolved. Verified implementation in `internal/expedition/` including encounter engine, skills, and background worker (`internal/worker/expedition_worker.go`).
- **Duels**: PENDING. `Accept` method still returns "not implemented". This is the last remaining item in this tracking issue.

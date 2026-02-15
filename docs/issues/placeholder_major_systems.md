# Issue: Placeholder Implementation of Major Gameplay Systems

## Description

The Duel and Compost systems are currently functionally blocked by "not implemented" placeholders in core service methods.

### 1. Duel System Incomplete

The `internal/duel/service.go` method `Accept` is a placeholder.

- **Impact**: Users can challenge each other and have their currency/items deducted, but the duel can never be accepted or resolved. This leads to stuck game states and lost user currency.
- **Root Cause**: Missing coin-flip/dice-roll logic and winner awarding logic.
- **Location**: `internal/duel/service.go:90`.

### 2. Compost System Incomplete

The `internal/compost/service.go` method `Harvest` and parts of `Deposit` are placeholders.

- **Impact**: Users cannot recycle items or claim rewards.
- **Root Cause**: Missing implementation logic.
- **Location**: `internal/compost/service.go`.

### 3. Expedition System (Resolved)

Previously incomplete, the Expedition system (`internal/expedition/service.go`) is now fully implemented with `ExecuteExpedition` logic and background workers.

## Proposed Solution

- Implement the resolution logic for Duels including random winner selection and reward distribution.
- Implement the Compost system for item recycling.
- Add background workers or scheduler jobs to handle timeouts/expirations for Duels.

## Status Update (2026-01-30)

Verified that `internal/duel/service.go` (`Accept`) and `internal/expedition/service.go` (`ExecuteExpedition`) still return "not implemented" errors. The issue persists.

## Status Update (2026-02-06)

- **Expeditions**: `ExecuteExpedition` is now implemented. Issue resolved for Expeditions.
- **Duels**: Still incomplete.
- **Compost**: Identified as incomplete. Added to this tracking issue.

# Missing Discord Command Registrations

This document tracks features that have backend implementation (Service/Handler layers) but lack corresponding Discord slash command registrations.

## Duels (PVP)

- **Status**: Backend Service Interface defined (`internal/duel/service.go`).
  - `Challenge` method is implemented.
  - `Accept` method returns "not implemented".
- **Missing**: `/duel` command group (challenge, accept, decline).
- **Action Required**:
  1. Implement `Accept` logic in `internal/duel/service.go` (e.g., coin flip, dice roll).
  2. Create `internal/discord/cmd_duel.go` to handle interactions.
  3. Register commands in `internal/discord/commands.go`.

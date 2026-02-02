# Issue: Missing Discord Command Registrations

## Description

Several core gameplay features implemented in the API are not currently accessible via the Discord bot because their command factories are not registered in the main entry point.

### 1. Duel System Commands

Commands for initiating, accepting, and declining duels are missing.

- **Missing Commands**: `/duel`, `/accept`, `/decline`.
- **Impact**: The duel system is entirely inaccessible to Discord users despite being a major advertised feature.
- **Location**: `cmd/discord/main.go`, `internal/discord/commands.go`.

### 2. Expedition System Commands

Commands for the newly implemented expedition system have not been added to the bot registry.

- **Missing Commands**: `/expedition start`, `/expedition join`, `/expedition status`.
- **Impact**: Users cannot participate in group expeditions via Discord.
- **Location**: `cmd/discord/main.go`.

## Proposed Solution

- Create `internal/discord/cmd_duel.go` and `internal/discord/cmd_expedition.go` (if they don't exist) with appropriate command factories.
- Register these factories in `getCommandFactories` within `cmd/discord/main.go`.
- Ensure autocomplete is implemented for expedition types and user targets.

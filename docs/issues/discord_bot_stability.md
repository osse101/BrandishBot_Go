# Issue: Discord Bot Graceful Shutdown and Stability

## Description

The Discord bot component lacks proper graceful shutdown handling for its background tasks and SSE client connection.

### 1. Fire-and-Forget Background Tasks

The `StartDailyCommitChecker` launches a ticker in a goroutine that is not tracked by any synchronization primitive.

- **Impact**: When the bot stops, this goroutine might be killed while in the middle of a GitHub API request or while formatting a message, potentially leading to incomplete logs or unexpected behavior on restart.
- **Location**: `internal/discord/bot.go:154`.

### 2. SSE Client Termination

The SSE client used for real-time notifications (`sseClient`) is stopped, but the connection handling logic doesn't fully guarantee all in-flight event processing is finished before exiting.

- **Impact**: Critical game events (like job level-ups) might be missed in Discord notifications during a bot restart.
- **Location**: `internal/discord/bot.go:110`.

## Proposed Solution

- Implement a `context.Context` based shutdown for all background tickers in the bot.
- Add a `sync.WaitGroup` to the `Bot` struct to track all background goroutines.
- Update `Stop()` to cancel the context and wait for the `WaitGroup` before closing the Discord session.

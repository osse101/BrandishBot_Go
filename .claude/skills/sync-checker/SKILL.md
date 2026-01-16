# BrandishBot Client Sync Checker
Assist in maintaining synchronization between Go API handlers, the Discord client, and the C# Streamer.bot client.

## Triggering
- When modifying files in `internal/handler/`
- When modifying `internal/discord/client.go`
- When modifying `client/csharp/BrandishBotClient.cs`

## Rules
- Every new endpoint must have a corresponding method in all three clients.
- Always check `docs/CLIENT_WRAPPER_CHECKLIST.md` after changes.
- Ensure the JSON tags in Go structs match the property names in C# classes.

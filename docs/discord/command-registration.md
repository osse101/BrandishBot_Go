# Discord Command Registration

## Smart Registration System

The Discord bot now uses intelligent command registration that avoids unnecessary API calls.

### How It Works

**On Startup:**
1. Fetches currently registered commands from Discord
2. Compares them with local command definitions
3. **Only updates if changes are detected**
4. Skips registration if commands are identical

### Benefits

✅ **Avoids Rate Limits** - Discord has rate limits on command registration  
✅ **Faster Startup** - Skips unnecessary API calls  
✅ **Safer Deployments** - No risk of accidentally clearing commands  
✅ **Smart Updates** - Detects changes in name, description, options, choices

### Force Update

When you **add or modify commands**, set the environment variable:

```bash
DISCORD_FORCE_COMMAND_UPDATE=true
```

This forces a full command refresh on next startup.

### Example Usage

**Normal startup (commands unchanged):**
```bash
./bin/discord_bot
# Output: Commands unchanged, skipping registration (count: 21)
```

**After adding new commands:**
```bash
DISCORD_FORCE_COMMAND_UPDATE=true ./bin/discord_bot
# Output: Force update enabled - replacing all commands (count: 21)
```

**After modifying existing commands:**
```bash
# No flag needed - auto-detected!
./bin/discord_bot
# Output: Commands changed, updating... (existing: 21, desired: 21)
```

### What's Compared

The system compares:
- Command name
- Command description  
- Number of options
- Option types, names, descriptions, required status
- Choice values (for dropdowns)

### Docker/.env Configuration

Add to `.env`:
```bash
DISCORD_FORCE_COMMAND_UPDATE=false  # Default
```

Set to `true` temporarily when deploying new commands, then change back to `false`.

## Troubleshooting

**Commands not updating?**
→ Set `DISCORD_FORCE_COMMAND_UPDATE=true` once

**Rate limit errors?**
→ Don't force update on every restart (the new system prevents this)

**Commands deleted accidentally?**
→ Run with force flag to re-register all commands

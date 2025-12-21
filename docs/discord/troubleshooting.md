# Discord Bot Troubleshooting Guide

Common issues and solutions for BrandishBot Discord integration.

## Bot Shows Offline

### Symptoms
- Bot appears offline in Discord
- No response to commands
- Logs show connection errors

### Solutions

**1. Check Bot Token**
```bash
# Verify token in .env
cat .env | grep DISCORD_TOKEN
```
- Token should be ~70 characters
- No spaces or quotes around token
- If exposed, regenerate at https://discord.com/developers

**2. Verify Bot is Running**
```bash
# Check if process is running
make discord-logs

# Should see: "Bot is ready"
```

**3. Check API Connection**
```bash
curl http://localhost:8080/healthz
# Should return: {"status":"healthy"}
```

**4. Restart Bot**
```bash
make docker-discord-restart
```

---

## Commands Not Appearing

### Symptoms
- Typing `/` shows no BrandishBot commands
- "Application did not respond" error
- Old commands still showing

### Solutions

**1. Force Command Registration**
```bash
# Add to .env
DISCORD_FORCE_COMMAND_UPDATE=true

# Restart bot
make docker-discord-restart

# Remove flag after
DISCORD_FORCE_COMMAND_UPDATE=false
```

**2. Wait for Sync**
Discord can take 1-2 minutes to propagate commands.
- Restart Discord app
- Try in different channel

**3. Check Logs**
```bash
make discord-logs | grep "Commands"
# Should see: "Commands registered successfully"
```

**4. Verify Permissions**
Bot needs `applications.commands` scope
- Re-invite bot if missing

---

## Permission Errors

### Symptoms
- "Missing Permissions" errors
- Bot can't send messages
- Embed/attachment failures

### Solutions

**1. Check Bot Permissions**
In Server Settings > Roles:
- Bot role has "Send Messages"
- Bot role has "Embed Links"
- Bot role has "Use Slash Commands"

**2. Channel Overrides**
Check channel-specific permissions:
- Right-click channel > Edit Channel
- Permissions > Bot role
- Ensure not denied

**3. Role Hierarchy**
Bot role must be above roles it manages:
- Server Settings > Roles
- Drag bot role higher

**4. Re-invite Bot**
Generate new invite with correct permissions:
https://discord.com/developers/applications

---

## API Connection Failures

### Symptoms
- "Error connecting to game server"
- Commands timeout
- Health check fails

### Solutions

**1. Check API Status**
```bash
curl http://localhost:8080/healthz
```

**2. Check Docker Network**
```bash
docker-compose ps
# Both 'app' and 'discord' should be 'Up'
```

**3. Verify API_URL**
```bash
# In docker-compose.yml
environment:
  - API_URL=http://app:8080  # Not localhost!
```

**4. Check Logs**
```bash
make discord-logs | grep "API"
# Look for connection errors
```

---

## Health Check Failures

### Symptoms
- Docker marks container unhealthy
- Container restarts frequently

### Solutions

**1. Check Health Endpoint**
```bash
# Inside container
docker-compose exec discord wget -O- http://localhost:8082/health
```

**2. Verify Port**
```bash
# In .env
DISCORD_WEBHOOK_PORT=8082
```

**3. Check Health Status**
```bash
curl http://localhost:8082/health
```

Expected response:
```json
{
  "status": "healthy",
  "uptime": "1h23m45s",
  "connected": true,
  "commands_received": 42,
  "api_reachable": true
}
```

---

## Rate Limit Errors

### Symptoms
- "Rate limited" in logs
- Commands delayed
- 429 errors

### Solutions

**1. Reduce Command Frequency**
Discord limits:
- 50 commands/second global
- 5 commands/second per user

**2. Check Retry Logic**
Bot automatically retries with backoff.
Check logs for retry attempts.

**3. Spread Out Registrations**
Don't force-update commands repeatedly.

---

## Database Connection Issues

### Symptoms
- "User not found" errors
- Data not persisting
- Registration failures

### Solutions

**1. Check API Database**
```bash
# Core API should connect to DB
make docker-logs | grep "database"
```

**2. Run Migrations**
```bash
make migrate-up
```

**3. Verify PostgreSQL**
```bash
docker-compose ps db
# Should show 'Up'
```

---

## Command-Specific Issues

### `/search` Not Working
- Check cooldowns (30s default)
- Verify user registered
- Check item probabilities

### `/gamble` Timing Out
- Ensure multiple users joined
- Wait for timer to expire
- Check lootbox availability

### `/upgrade` Fails
- Verify recipe exists (`/recipes`)
- Check materials in inventory
- Ensure sufficient quantity

### Admin Commands Fail
- Verify user has Administrator permission
- Check bot has admin role
- Verify API_KEY is correct

---

## Debugging Steps

### 1. Enable Verbose Logging
```bash
# Set in .env
LOG_LEVEL=DEBUG
```

### 2. Check All Services
```bash
docker-compose ps
# All should show 'Up (healthy)'
```

### 3. Inspect Logs
```bash
# Discord bot logs
make discord-logs

# API logs
docker-compose logs app

# Database logs
docker-compose logs db
```

### 4. Test Health
```bash
# Discord health
curl http://localhost:8082/health

# API health
curl http://localhost:8080/healthz

# Database ready
curl http://localhost:8080/readyz
```

### 5. Restart Services
```bash
# Restart everything
make docker-down
make docker-up

# Or just Discord
make docker-discord-restart
```

---

## Common Error Messages

### "Application did not respond"
- API is down or unreachable
- Command took > 3 seconds
- Check API_URL configuration

### "Unknown interaction"
- Command registry out of sync
- Force update commands
- Wait for Discord to sync

### "Missing Access"
- Bot lacks channel permissions
- Check role permissions
- Verify bot can see channel

### "Invalid Form Body"
- Malformed API request
- Check command parameters
- Verify item names

---

## Getting Help

### Before Asking
1. Check this guide
2. Review logs
3. Test health endpoints
4. Verify configuration

### Information to Provide
- Discord bot logs
- API logs
- Health check output
- Steps to reproduce

### Resources
- Setup Guide: `docs/discord/setup.md`
- Command Reference: `/info commands` in Discord
- API Docs: http://localhost:8080/swagger/

## Still Stuck?

File an issue with:
- Full error message
- Relevant logs
- Configuration (sanitized)
- Steps to reproduce

# Discord Bot Setup Guide

Complete guide to creating and configuring your Discord bot for BrandishBot.

## Prerequisites

- Discord account
- Server with admin permissions (or create your own)
- BrandishBot API running

## Step 1: Create Discord Application

1. **Go to Discord Developer Portal**
   - Visit https://discord.com/developers/applications
   - Click "New Application"
   - Name: "BrandishBot" (or your preference)
   - Click "Create"

2. **Configure Application**
   - Copy the **Application ID** (needed for `.env`)
   - Add description and icon (optional)

## Step 2: Create Bot User

1. **Navigate to Bot Section**
   - Click "Bot" in left sidebar
   - Click "Add Bot"
   - Confirm "Yes, do it!"

2. **Configure Bot Settings**
   - **Username**: BrandishBot
   - **Icon**: Upload an avatar
   - **Public Bot**: OFF (recommended for private use)
   - **Requires OAuth2 Code Grant**: OFF

3. **Bot Permissions**
   Enable these under "Privileged Gateway Intents":
   - ✅ **Presence Intent** - See user status
   - ✅ **Server Members Intent** - Access member list
   - ✅ **Message Content Intent** - Read message content

4. **Copy Bot Token**
   - Click "Reset Token"
   - **IMPORTANT**: Copy the token immediately!
   - Store securely - you can't see it again
   - This is your `DISCORD_TOKEN` for `.env`

## Step 3: Generate Bot Invite URL

1. **Go to OAuth2 > URL Generator**

2. **Select Scopes**
   - ✅ `bot`
   - ✅ `applications.commands`

3. **Select Bot Permissions**
   - ✅ Send Messages
   - ✅ Send Messages in Threads
   - ✅ Embed Links
   - ✅ Attach Files
   - ✅ Read Message History
   - ✅ Use Slash Commands
   - ✅ Add Reactions

4. **Copy Generated URL**
   - Bottom of page shows invite link
   - Copy this URL

## Step 4: Invite Bot to Server

1. **Open Invite URL** in browser
2. **Select Server** from dropdown
3. **Authorize** the bot
4. **Complete Captcha** if prompted

Your bot will now appear offline in the server!

## Step 5: Configure Environment

1. **Update `.env` file**:

```bash
# Discord Configuration
DISCORD_TOKEN=your_bot_token_here
DISCORD_APP_ID=your_application_id_here
DISCORD_DEV_CHANNEL_ID=optional_channel_for_logs

# Optional: Force command registration on first run
DISCORD_FORCE_COMMAND_UPDATE=true
```

2. **Get Channel IDs** (optional):
   - Enable Developer Mode: Settings > Advanced > Developer Mode
   - Right-click channel > Copy ID
   - Use for `DISCORD_DEV_CHANNEL_ID`

## Step 6: Start the Bot

**Local Development:**
```bash
make build
make discord-run
```

**Docker:**
```bash
make docker-up
```

**Check logs:**
```bash
make discord-logs
```

## Step 7: Register Commands

On first run, set:
```bash
DISCORD_FORCE_COMMAND_UPDATE=true
```

This registers all 21 slash commands with Discord.

**Verify in Discord:**
- Type `/` in any channel
- You should see BrandishBot commands!

## Step 8: Test Commands

Try these commands:
- `/ping` - Check bot status
- `/profile` - View your profile  
- `/info` - Get help
- `/search` - Find items

## Security Best Practices

### Token Security
- ✅ Never commit `.env` to git
- ✅ Use `.env.example` for templates
- ✅ Rotate token if exposed
- ✅ Restrict bot permissions

### Server Security
- ✅ Keep bot private (disable public)
- ✅ Only invite to trusted servers
- ✅ Monitor bot usage
- ✅ Set up admin-only channels

### API Key
Generate secure API key:
```bash
openssl rand -hex 32
```

Add to `.env`:
```bash
API_KEY=your_generated_key_here
```

## Troubleshooting

### Bot Shows Offline
- Check `DISCORD_TOKEN` is correct
- Verify bot is running: `make discord-logs`
- Check API is accessible: `curl http://localhost:8080/healthz`

### Commands Not Appearing
- Run with `DISCORD_FORCE_COMMAND_UPDATE=true`
- Wait 1-2 minutes for Discord to sync
- Restart Discord app

### Permission Errors
- Verify bot has required permissions
- Check role hierarchy (bot role should be high)
- Re-invite bot with updated permissions

### Connection Issues
- Check internet connection
- Verify Discord isn't down: https://discordstatus.com
- Check firewall rules

## Advanced Configuration

### Multiple Servers
The bot can join multiple servers. Commands work in all.

### Custom Prefix
Currently slash commands only. Prefix commands not implemented.

### Rate Limits
Discord rate limits apply:
- 50 commands/second global
- 5 commands/second per user

Bot includes retry logic for resilience.

## Next Steps

- Read `/info` in Discord for feature help
- Check `docs/discord/command-registration.md` for deployment
- Review main `README.md` for API details

## Support

- Check troubleshooting guide
- Review logs with `make discord-logs`
- GitHub Issues: [your repo]

Your bot is ready! 🚀

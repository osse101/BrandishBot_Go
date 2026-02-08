# Streamerbot API Integration for Info Commands

This guide shows how to configure Streamerbot to serve `!info` commands by calling the BrandishBot API.

## Overview

Streamerbot will make HTTP requests to:

```
GET http://localhost:8080/api/v1/info?platform=twitch&feature={topic}
```

The API returns JSON with platform-specific formatted content from the YAML files.

## Streamerbot Action Configuration

### Basic !info Command

**Command**: `!info [topic]`

**Sub-Actions**:

1. **Set Variables**
   - Type: `Core > Logic > Set Argument`
   - Variable: `%topic%`
   - Value: `%rawInput%` (or "overview" if empty)

2. **Call API**
   - Type: `Core > Network > Fetch URL`
   - URL: `http://localhost:8080/api/v1/info?platform=twitch&feature=%topic%`
   - Method: `GET`
   - Headers:
     - `X-API-Key: <your-api-key>`
   - Variable Name: `%response%`

3. **Parse JSON Response**
   - Type: `Core > Logic > Parse JSON`
   - JSON String: `%response%`
   - JSON Path: `$.description`
   - Variable Name: `%infoText%`

4. **Send to Chat**
   - Type: `Twitch > Bot Account > Send Message to Channel`
   - Message: `%infoText%`

### Handling Empty Input

To default to "overview" when no topic is provided:

```
Sub-Action 1: Check if rawInput is empty
- Type: Core > Logic > If/Else
- Condition: %rawInput% is empty
- Then: Set %topic% = "overview"
- Else: Set %topic% = %rawInput%
```

### Error Handling

Add a sub-action after the API call:

```
Sub-Action: Check for errors
- Type: Core > Logic > If/Else
- Condition: %response% contains "error"
- Then: Send Message "Topic not found. Try !info commands for a list."
- Else: Continue to parse and send
```

## Available Topics

Users can request info on:

- `!info` or `!info overview` - Bot introduction
- `!info farming` - Farming system
- `!info economy` - Economy and trading
- `!info crafting` - Crafting system
- `!info inventory` - Inventory management
- `!info gamble` - Gambling mechanics
- `!info expeditions` - Expeditions
- `!info quests` - Quests
- `!info jobs` - Job system
- `!info progression` - Progression voting
- `!info stats` - Stats and leaderboards
- `!info commands` - Command list

### Subtopics (hierarchical)

For features with subtopics (like farming):

```
!info farming         → Main farming info with "More: !info harvest, !info compost"
!info farming harvest → Specific harvest info
!info farming compost → Specific compost info
```

API call for subtopics:

```
URL: http://localhost:8080/api/v1/info?platform=twitch&feature=farming&topic=harvest
```

Streamerbot can parse `!info farming harvest` as:

- `feature=farming`
- `topic=harvest`

## API Authentication

### Option 1: API Key (Recommended)

Add header to Fetch URL action:

```
X-API-Key: <your-api-key-from-env>
```

### Option 2: Trusted Proxy

If Streamerbot runs on trusted server, configure server with `TRUSTED_PROXIES`:

```bash
TRUSTED_PROXIES=127.0.0.1,::1
```

Then no API key needed for local requests.

## Example Response

**Request**: `GET /api/v1/info?platform=twitch&feature=farming`

**Response**:

```json
{
  "platform": "twitch",
  "feature": "farming",
  "description": "Farming: Passive rewards! Wait (1hr-1wk) then !harvest. Rewards increase with time. Spoilage > 2wks! 5hr+ wait = Farmer XP. More: !info farming harvest, !info farming compost",
  "link": ""
}
```

## Testing

Test the API endpoint directly:

```bash
curl -H "X-API-Key: your-key" \
  "http://localhost:8080/api/v1/info?platform=twitch&feature=farming"
```

## Advantages of API Integration

✅ **Single Source of Truth**: YAML files in `configs/info/`
✅ **Automatic Updates**: Edit YAML, restart API, Streamerbot gets new content
✅ **Platform Consistency**: Same data source for Discord, Twitch, HTTP
✅ **Topic Hierarchy**: Support for subtopics like `farming` → `harvest`
✅ **No Duplication**: One description per platform in YAML

## Updating Content

To change info content:

1. Edit YAML file: `configs/info/farming.yaml`
2. Update the `streamerbot:` description field
3. Restart the API server
4. Streamerbot automatically gets new content on next `!info` call

No need to touch Streamerbot configuration!

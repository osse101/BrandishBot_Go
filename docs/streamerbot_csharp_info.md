# Streamerbot C# Client - Info Command Setup

This guide shows how to set up the `!info` command in Streamerbot using the BrandishBot C# client.

## Quick Start

### 1. Add C# Files to Streamerbot

Add these files to your Streamerbot action as C# code references:

- `BrandishBotClient.cs` (base client)
- `BrandishBotClient.Core.cs` (core methods including GetInfo)
- `Models.cs` (data models including InfoResponse)
- `ResponseFormatter.cs` (formatters including FormatInfo)
- `BrandishBotWrapper.cs` (Streamerbot wrapper)

### 2. Configure Global Variables

In Streamerbot, set these persisted global variables:

- `ServerBaseURL` = `http://127.0.0.1:8080` (or your API address)
- `ServerApiKey` = `your-api-key-here`

### 3. Create !info Command

**Command**: `!info`  
**Permission**: Everyone

**Sub-Actions**:

1. **Execute C# Code**
   - Method: `GetInfo`
   - Code: (reference all the BrandishBot files)

2. **Send Message to Channel**
   - Message: `%response%`

## Usage Examples

Users can now request info using:

```
!info                    → General overview
!info farming            → Farming feature
!info harvest            → Harvest topic (automatically found under farming)
!info crafting           → Crafting system
!info commands           → Full command list
!info compost            → Compost topic (automatically found under farming)
```

## How It Works

1. **Platform Auto-Detection**: Uses `%userType%` from Streamerbot context (twitch/youtube)
2. **API Call**: Calls `/api/v1/info?platform=twitch&feature=harvest`
3. **Smart Lookup**: Backend searches feature names first, then topic names if not found
4. **Response Parsing**: Extracts `description` field which contains compact text optimized for chat
5. **Output**: Sets `%response%` variable for chat message

## Method Signature

```csharp
public bool GetInfo()
{
    // Automatically uses userType (platform) from context
    // Reads input0 as name (optional) - can be feature OR topic
    // Backend determines whether it's a feature or topic
    // Returns formatted description in %response% variable
}
```

## Error Handling

The method gracefully handles:

- **Feature not found**: "Feature not found: {feature}"
- **Topic not found**: "Topic not found: {feature}/{topic}"
- **API errors**: Logs warning and returns user-friendly error message
- **Missing platform**: Logs error and returns false

## Available Features

Based on the YAML files in `configs/info/`:

| Feature     | Description                          |
| ----------- | ------------------------------------ |
| overview    | Bot introduction and getting started |
| farming     | Passive rewards and harvest system   |
| economy     | Buy/sell items and trading           |
| crafting    | Upgrade and disassemble items        |
| inventory   | Item management                      |
| gamble      | Betting mechanics                    |
| expeditions | Cooperative multiplayer              |
| quests      | Weekly challenges                    |
| jobs        | RPG job leveling                     |
| progression | Community voting                     |
| stats       | Leaderboards and tracking            |
| commands    | Full command reference               |

## Advanced: Topic Hierarchy

Some features support sub-topics. Example with farming:

```
!info farming          → Main farming info + "More: !info farming harvest"
!info farming harvest  → Specific harvest mechanics
!info farming compost  → Compost system details
```

**How to add**: Pass second argument as topic:

- input0 = "farming"
- input1 = "harvest"

## Testing

Test the setup:

1. **In Streamerbot Chat**:

   ```
   !info farming
   ```

2. **Expected Response**:

   ```
   Farming: Passive rewards! Wait (1hr-1wk) then !harvest. Rewards increase with time.
   Spoilage > 2wks! 5hr+ wait = Farmer XP. More: !info farming harvest, !info farming compost
   ```

3. **Check Logs**: Streamerbot logs should show:
   ```
   [BrandishBot] GetInfo completed successfully
   ```

## Troubleshooting

**"GetInfo: Missing userType"**: Ensure command is triggered from chat (not manual test)

**"Feature not found"**: Check that YAML file exists in `configs/info/`

**API connection failed**: Verify `ServerBaseURL` and check API is running

**Empty response**: Check `ServerApiKey` is correct and API authentication is configured

## Content Updates

To update info content:

1. Edit YAML file: `configs/info/{feature}.yaml`
2. Update `streamerbot:` description field
3. Restart API server
4. Changes take effect immediately (no Streamerbot restart needed)

Example:

```yaml
# configs/info/farming.yaml
streamerbot:
  description: "NEW TEXT HERE"
```

This keeps content centralized and consistent across all platforms!

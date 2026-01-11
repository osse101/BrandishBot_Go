# BrandishBot C# Client for streamer.bot

## Quick Start

### 1. Initialize the Client (Once at Startup)

```csharp
// Initialize the singleton once at application/action startup
BrandishBotClient.Initialize(
    baseUrl: "http://localhost:8080",  // Your API URL
    apiKey: "your-api-key-here"
);

// Then use BrandishBotClient.Instance everywhere
```

### 2. Common Usage Examples

#### Twitch Chat Integration

```csharp
// Handle a Twitch chat message
var response = await BrandishBotClient.Instance.HandleMessage(
    platform: Platform.Twitch,
    platformId: "%user.id%",           // streamer.bot variable
    username: "%user.name%",           // streamer.bot variable
    message: "%rawInput%",             // The message text
    isModerator: %user.isModerator%,   // Mod status
    isSubscriber: %user.isSubscriber%  // Sub status
);
```

#### YouTube Chat Integration

```csharp
// Handle a YouTube chat message
var response = await BrandishBotClient.Instance.HandleMessage(
    platform: Platform.YouTube,
    platformId: "%user.id%",
    username: "%user.name%",
    message: "%rawInput%",
    isModerator: false,
    isSubscriber: false
);
```

#### Get User Inventory

```csharp
var inventory = await BrandishBotClient.Instance.GetInventory(
    platform: Platform.Twitch,
    platformId: "%user.id%"
);
```

#### Open a Lootbox

```csharp
var result = await BrandishBotClient.Instance.UseItem(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    itemName: ItemName.Lootbox,  // Use string constants
    quantity: 1
);
```

#### Buy an Item

```csharp
var result = await BrandishBotClient.Instance.BuyItem(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    itemName: ItemName.Junkbox,
    quantity: 3
);
```

#### Start a Gamble

```csharp
var gamble = await BrandishBotClient.Instance.StartGamble(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    itemName: ItemName.Lootbox,
    quantity: 2
);
```

#### Join a Gamble

```csharp
var result = await BrandishBotClient.Instance.JoinGamble(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    gambleId: "gamble-uuid-here",
    itemName: ItemName.Lootbox,
    quantity: 2
);
```

#### Check Leaderboard

```csharp
var leaderboard = await BrandishBotClient.Instance.GetLeaderboard(
    metric: "engagement_score",
    limit: 10
);
```

#### Vote for Progression

```csharp
var result = await BrandishBotClient.Instance.VoteForNode(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    nodeKey: "feature_gamble"
);
```

## streamer.bot Setup

### Execute Code Action

1. In streamer.bot, create a new action
2. Add "Execute Code" sub-action
3. Select C# as the language
4. Copy/paste the BrandishBotClient class
5. Use the examples above in your action code

### Example Action: Process Chat Message

**Important for streamer.bot users:** Each action runs in isolation, so you need to initialize in every action. The Initialize() method is safe to call multiple times.

```csharp
using System;
using System.Threading.Tasks;

public class CPHInline
{
    public async Task<bool> Execute()
    {
        // Call Initialize in EVERY action - it's safe to call multiple times
        BrandishBotClient.Initialize(
            "http://localhost:8080",
            "your-api-key"
        );
        
        try
        {
            var response = await BrandishBotClient.Instance.HandleMessage(
                Platform.Twitch,
                CPH.GetGlobalVar<string>("userId"),
                CPH.GetGlobalVar<string>("userName"),
                CPH.GetGlobalVar<string>("message"),
                CPH.GetGlobalVar<bool>("isModerator"),
                CPH.GetGlobalVar<bool>("isSubscriber")
            );
            
            CPH.SendMessage(response);  // Send result to chat
            return true;
        }
        catch (Exception ex)
        {
            CPH.LogError($"API Error: {ex.Message}");
            return false;
        }
    }
}
```

**Pro Tip:** Store your API URL and key in streamer.bot global variables for easy updates:

```csharp
BrandishBotClient.Initialize(
    CPH.GetGlobalVar<string>("BrandishBotUrl", true),
    CPH.GetGlobalVar<string>("BrandishBotApiKey", true)
);
```

## Environment Variables

Set these in your `.env` file on the server:

```bash
GAMBLE_JOIN_DURATION_MINUTES=2  # Time to join gambles
API_KEY=your-secret-api-key     # Required for auth
```

## Response Formats

All endpoints return JSON. Common patterns:

**Success:**

```json
{
  "message": "Item purchased successfully",
  "inventory": {...}
}
```

**Error:**

```json
{
  "error": "Insufficient funds"
}
```

## Constants Reference

### Platform Constants

```csharp
Platform.Twitch   = "twitch"
Platform.YouTube  = "youtube"
Platform.Discord  = "discord"
```

### Event Type Constants

```csharp
EventType.Message    = "message"
EventType.Follow     = "follow"
EventType.Subscribe  = "subscribe"
EventType.Raid       = "raid"
EventType.Bits       = "bits"
```

### Item Name Constants (REQUIRED)

**All item operations now use string names, not numeric IDs:**

```csharp
// These are the command names users type in chat
ItemName.Money    = "money"    // Coins
ItemName.Junkbox  = "junkbox"  // Tier 0 - Rusty Lootbox
ItemName.Lootbox  = "lootbox"  // Tier 1 - Basic Lootbox
ItemName.Goldbox  = "goldbox"  // Tier 2 - Golden Lootbox
ItemName.Missile  = "missile"  // Ray Gun / Blaster
```

### Item Public Name Constants

```csharp
// These are the command names users type in chat
ItemName.Money    = "money"    // Coins
ItemName.Junkbox  = "junkbox"  // Tier 0 - Rusty Lootbox
ItemName.Lootbox  = "lootbox"  // Tier 1 - Basic Lootbox
ItemName.Goldbox  = "goldbox"  // Tier 2 - Golden Lootbox
ItemName.Missile  = "missile"  // Ray Gun / Blaster
```

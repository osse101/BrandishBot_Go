# BrandishBot C# Client for streamer.bot

## Quick Start

### 1. Initialize the Client
```csharp
var client = new BrandishBotClient(
    baseUrl: "http://localhost:8080",  // Your API URL
    apiKey: "your-api-key-here"
);
```

### 2. Common Usage Examples

#### Twitch Chat Integration
```csharp
// Handle a Twitch chat message
var response = await client.HandleMessage(
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
var response = await client.HandleMessage(
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
var inventory = await client.GetInventory(
    platform: Platform.Twitch,
    platformId: "%user.id%"
);
```

#### Open a Lootbox
```csharp
var result = await client.UseItem(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    itemId: ItemId.Lootbox1,  // Use constants
    quantity: 1
);
```

#### Buy an Item
```csharp
var result = await client.BuyItem(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    itemId: ItemId.Lootbox0,
    quantity: 3
);
```

#### Start a Gamble
```csharp
var gamble = await client.StartGamble(
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    lootboxItemId: ItemId.Lootbox1,
    quantity: 2  // Betting 2 lootbox1s
);
```

#### Join a Gamble
```csharp
var result = await client.JoinGamble(
    gambleId: "gamble-uuid-here",  // From active gamble
    platform: Platform.Twitch,
    platformId: "%user.id%",
    username: "%user.name%",
    lootboxItemId: ItemId.Lootbox1,
    quantity: 2
);
```

#### Check Leaderboard
```csharp
var leaderboard = await client.GetLeaderboard(
    metric: "engagement_score",
    limit: 10
);
```

#### Vote for Progression
```csharp
var result = await client.VoteForNode(
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
```csharp
using System;
using System.Threading.Tasks;

public class CPHInline
{
    public async Task<bool> Execute()
    {
        var client = new BrandishBotClient(
            "http://localhost:8080",
            "your-api-key"
        );
        
        try
        {
            var response = await client.HandleMessage(
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

## Platform Constants Reference

```csharp
Platform.Twitch   = "twitch"
Platform.YouTube  = "youtube"
Platform.Discord  = "discord"

EventType.Message    = "message"
EventType.Follow     = "follow"
EventType.Subscribe  = "subscribe"
EventType.Raid       = "raid"
EventType.Bits       = "bits"

ItemId.Money      = 1
ItemId.Lootbox0   = 2
ItemId.Lootbox1   = 3
ItemId.Lootbox2   = 4
ItemId.Blaster    = 5
```

# Streamer.bot Integration Checklist

Quick reference for setting up Streamer.bot actions that connect to the BrandishBot C# client wrapper.

## Initial Setup

### 1. Install C# Client
- [ ] Copy `BrandishBotClient.cs` to Streamer.bot import directory
- [ ] Import the file into Streamer.bot
- [ ] Verify Newtonsoft.Json is available (built into Streamer.bot)

### 2. Create Initialization Action
**Action Name**: `BrandishBot - Initialize`  
**Trigger**: Manual / On Startup

```csharp
// Initialize the BrandishBot client (run once at startup)
BrandishBotClient.Initialize("http://localhost:8080", "YOUR_API_KEY_HERE");

if (BrandishBotClient.IsInitialized)
{
    CPH.SendMessage("✅ BrandishBot connected!");
}
else
{
    CPH.SendMessage("❌ BrandishBot failed to initialize!");
}
```

**Variables to Set**:
- `apiUrl` = `http://localhost:8080` (or your server URL)
- `apiKey` = Your API key from `.env` file

---

## Core User Commands

### 3. Search Command
**Action Name**: `!search`  
**Trigger**: Twitch Chat Command `!search`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var result = await client.Search(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString()
);

CPH.SendMessage(result);
```

**Arguments Used**: `userId`, `userName`

---

### 4. Inventory Command
**Action Name**: `!inventory`  
**Trigger**: Twitch Chat Command `!inventory`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var inventory = await client.GetInventory(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    null  // No filter
);

// Parse and display inventory
CPH.SendMessage($"@{args["userName"]} inventory: {inventory}");
```

**Optional**: Add filter parameter: `"lootbox"`, `"resource"`, etc.

---

### 5. Open Lootbox Command
**Action Name**: `!open <item> [quantity]`  
**Trigger**: Twitch Chat Command `!open`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

string itemName = args["rawInput"].ToString().Trim();
int quantity = 1;  // Default

// Parse quantity if provided (e.g., "!open lootbox 5")
string[] parts = itemName.Split(' ');
if (parts.Length > 1)
{
    itemName = parts[0];
    int.TryParse(parts[1], out quantity);
}

var result = await client.UseItem(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    itemName,
    quantity
);

CPH.SendMessage(result);
```

**Arguments Used**: `userId`, `userName`, `rawInput`

---

### 6. Buy Item Command
**Action Name**: `!buy <item> [quantity]`  
**Trigger**: Twitch Chat Command `!buy`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

string itemName = args["rawInput"].ToString().Trim();
int quantity = 1;

string[] parts = itemName.Split(' ');
if (parts.Length > 1)
{
    itemName = parts[0];
    int.TryParse(parts[1], out quantity);
}

var result = await client.BuyItem(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    itemName,
    quantity
);

CPH.SendMessage(result);
```

---

### 7. Sell Item Command
**Action Name**: `!sell <item> [quantity]`  
**Trigger**: Twitch Chat Command `!sell`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

string itemName = args["rawInput"].ToString().Trim();
int quantity = 1;

string[] parts = itemName.Split(' ');
if (parts.Length > 1)
{
    itemName = parts[0];
    int.TryParse(parts[1], out quantity);
}

var result = await client.SellItem(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    itemName,
    quantity
);

CPH.SendMessage(result);
```

---

### 8. Prices Command
**Action Name**: `!prices`  
**Trigger**: Twitch Chat Command `!prices`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var prices = await client.GetSellPrices();
CPH.SendMessage(prices);
```

---

## Progression Commands

### 9. Progression Tree Command
**Action Name**: `!tree` or `!progression`  
**Trigger**: Twitch Chat Command `!tree`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var tree = await client.GetProgressionTree();
// Consider parsing JSON and formatting nicely
CPH.SendMessage("Check out the progression tree: [link to visualization]");
```

---

### 10. Vote Command
**Action Name**: `!vote <node_key>`  
**Trigger**: Twitch Chat Command `!vote`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

string nodeKey = args["rawInput"].ToString().Trim();

var result = await client.VoteForNode(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    nodeKey
);

CPH.SendMessage(result);
```

---

### 11. Voting Session Command
**Action Name**: `!voting` or `!session`  
**Trigger**: Twitch Chat Command `!voting`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var session = await client.GetVotingSession();
// Parse JSON and display options
CPH.SendMessage("Current voting options: " + session);
```

---

### 12. Unlock Progress Command
**Action Name**: `!progress`  
**Trigger**: Twitch Chat Command `!progress`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var progress = await client.GetUnlockProgress();
CPH.SendMessage(progress);
```

---

## Gamble Commands

### 13. Start Gamble Command
**Action Name**: `!gamble <item> [quantity]`  
**Trigger**: Twitch Chat Command `!gamble`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

string itemName = args["rawInput"].ToString().Trim();
int quantity = 1;

string[] parts = itemName.Split(' ');
if (parts.Length > 1)
{
    itemName = parts[0];
    int.TryParse(parts[1], out quantity);
}

var result = await client.StartGamble(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    itemName,
    quantity
);

CPH.SendMessage(result);
```

---

### 14. Join Gamble Command
**Action Name**: `!join <item> [quantity]`  
**Trigger**: Twitch Chat Command `!join`

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

// Get active gamble ID first
var activeGamble = await client.GetActiveGamble();
// Parse JSON to extract gamble_id

string itemName = args["rawInput"].ToString().Trim();
int quantity = 1;

string[] parts = itemName.Split(' ');
if (parts.Length > 1)
{
    itemName = parts[0];
    int.TryParse(parts[1], out quantity);
}

var result = await client.JoinGamble(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    "GAMBLE_ID_FROM_ACTIVE",  // Parse from activeGamble JSON
    itemName,
    quantity
);

CPH.SendMessage(result);
```

---

## Engagement Tracking (Background)

### 15. Message Tracker
**Action Name**: `Track Message`  
**Trigger**: Twitch First Words, Twitch Chat Message (all messages)

```csharp
var client = BrandishBotClient.Instance;
if (client == null) return;  // Silently fail if not initialized

// Use all-in-one message handler
var result = await client.HandleMessage(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    args["userName"].ToString(),
    args["message"].ToString()
);

// Optionally send result if it contains a reward
if (!string.IsNullOrEmpty(result) && result.Contains("found"))
{
    CPH.SendMessage(result);
}
```

**Purpose**: Tracks engagement and auto-gives rewards for item mentions

---

### 16. Follow Tracker
**Action Name**: `Track Follow`  
**Trigger**: Twitch Follow

```csharp
var client = BrandishBotClient.Instance;
if (client == null) return;

await client.RecordEvent(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    BrandishBot.Client.EventType.Follow
);

// Add contribution points
await client.AdminAddContribution(5);  // 5 points for a follow
```

---

### 17. Subscribe Tracker
**Action Name**: `Track Subscribe`  
**Trigger**: Twitch Sub, Twitch ReSub

```csharp
var client = BrandishBotClient.Instance;
if (client == null) return;

await client.RecordEvent(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    BrandishBot.Client.EventType.Subscribe
);

// Add contribution points
await client.AdminAddContribution(10);  // 10 points for a sub
```

---

### 18. Raid Tracker
**Action Name**: `Track Raid`  
**Trigger**: Twitch Raid

```csharp
var client = BrandishBotClient.Instance;
if (client == null) return;

int viewers = int.Parse(args["viewers"].ToString());

await client.RecordEvent(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),
    BrandishBot.Client.EventType.Raid,
    new { viewer_count = viewers }
);

// Add contribution points based on raid size
int contribution = viewers / 2;  // 1 point per 2 viewers
await client.AdminAddContribution(contribution);
```

---

## Admin Commands (Streamer Only)

### 19. Give Item Command
**Action Name**: `!giveitem @user <item> [quantity]`  
**Trigger**: Twitch Chat Command `!giveitem` (Moderator/Broadcaster only)

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

// Parse: !giveitem @username itemname quantity
string input = args["rawInput"].ToString().Trim();
string[] parts = input.Split(' ');

if (parts.Length < 2)
{
    CPH.SendMessage("Usage: !giveitem @user itemname [quantity]");
    return;
}

string targetUser = parts[0].TrimStart('@');
string itemName = parts[1];
int quantity = parts.Length > 2 ? int.Parse(parts[2]) : 1;

// Get target user ID (Streamer.bot provides this via TwitchGetUserId action)
string targetUserId = CPH.TwitchGetUserId(targetUser);

var result = await client.GiveItem(
    BrandishBot.Client.Platform.Twitch,
    args["userId"].ToString(),  // From user
    BrandishBot.Client.Platform.Twitch,
    targetUserId,  // To user
    targetUser,
    itemName,
    quantity
);

CPH.SendMessage(result);
```

---

### 20. Add Item Command (Admin)
**Action Name**: `!additem @user <item> <quantity>`  
**Trigger**: Twitch Chat Command `!additem` (Broadcaster only)

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

string input = args["rawInput"].ToString().Trim();
string[] parts = input.Split(' ');

string targetUser = parts[0].TrimStart('@');
string itemName = parts[1];
int quantity = int.Parse(parts[2]);

string targetUserId = CPH.TwitchGetUserId(targetUser);

var result = await client.AddItem(
    BrandishBot.Client.Platform.Twitch,
    targetUserId,
    itemName,
    quantity
);

CPH.SendMessage(result);
```

---

### 21. Instant Unlock (Admin)
**Action Name**: `!instantunlock`  
**Trigger**: Twitch Chat Command `!instantunlock` (Broadcaster only)

```csharp
var client = BrandishBotClient.Instance;
if (client == null) { CPH.SendMessage("BrandishBot not initialized!"); return; }

var result = await client.AdminInstantUnlock();
CPH.SendMessage(result);
```

---

## Summary Checklist

### Essential Actions (Minimum Viable Integration)
- [ ] Initialize action (run on startup)
- [ ] !search command
- [ ] !inventory command
- [ ] !open command
- [ ] Message tracker (background engagement)

### Recommended Actions
- [ ] !prices command
- [ ] !buy / !sell commands
- [ ] !vote command
- [ ] !progress command
- [ ] Follow/Sub/Raid trackers

### Optional Actions
- [ ] Gamble commands
- [ ] Admin commands (!giveitem, !additem)
- [ ] Progression tree visualization

---

## Testing Checklist

1. [ ] **Test initialization** - Run initialize action, verify connection
2. [ ] **Test search** - Use !search, verify response
3. [ ] **Test inventory** - Use !inventory, verify items shown
4. [ ] **Test lootbox** - Use !open lootbox, verify rewards
5. [ ] **Test engagement** - Send messages, verify auto-rewards
6. [ ] **Test voting** - Use !vote, verify vote recorded
7. [ ] **Test errors** - Try invalid commands, verify graceful error handling

---

## Troubleshooting

### "BrandishBot not initialized!"
- Check that Initialize action ran successfully
- Verify API URL and API Key are correct
- Check that backend is running (`http://localhost:8080/healthz`)

### "Error: 401 Unauthorized"
- API Key is incorrect
- Check API_KEY in backend `.env` file matches client

### "Error: 500 Server Error"
- Backend crashed or has a bug
- Check backend logs: `docker compose -f docker-compose.production.yml logs -f app`

### Commands not responding
- Check that action triggers are set correctly
- Verify arguments (`userId`, `userName`, `rawInput`) are being passed
- Enable debug logging in Streamer.bot

---

## Advanced: Custom Response Formatting

Example of parsing JSON response for better formatting:

```csharp
// Get inventory and format nicely
var inventoryJson = await client.GetInventory(...);
var inventory = JsonConvert.DeserializeObject<InventoryResponse>(inventoryJson);

string formatted = $"@{userName} inventory: ";
foreach (var item in inventory.Items)
{
    formatted += $"{item.Name} x{item.Quantity}, ";
}

CPH.SendMessage(formatted.TrimEnd(',', ' '));
```

---

## Platform-Specific Notes

### Twitch Arguments Available
- `userId` - User's Twitch ID
- `userName` - User's Twitch username
- `message` - Full message text
- `rawInput` - Command arguments (text after !command)
- `isModerator` - Is user a mod?
- `isBroadcaster` - Is user the broadcaster?

### YouTube Integration
Replace `Platform.Twitch` with `Platform.YouTube` and use YouTube-specific arguments.

---

## Quick Start

1. Import `BrandishBotClient.cs` into Streamer.bot
2. Create "Initialize" action with startup trigger
3. Create "!search" command action
4. Create "Track Message" action for engagement
5. Test with `!search` in chat
6. Add more commands as needed

---

## Related Files
- [C# Client Source](file:///home/osse1/projects/BrandishBot_Go/client/csharp/BrandishBotClient.cs)
- [API Endpoint Reference](file:///home/osse1/projects/BrandishBot_Go/docs/CLIENT_WRAPPER_CHECKLIST.md)

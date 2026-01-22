# Client Wrapper Binding Checklist

Complete checklist for implementing client wrappers (C#, TypeScript, Python, etc.) for BrandishBot API.

## Status Legend

- ✅ **Implemented** - Already bound in C# client
- ⚠️ **Partially Implemented** - Exists but may need updates 
- ❌ **Not Implemented** - Needs to be added
- 🔒 **Admin Only** - Requires admin/streamer permissions

---

## Core Configuration

- [x] **Initialize Client**
  - Base URL configuration
  - API Key authentication
  - HTTP client with retry logic
  - Timeout configuration (recommended: 10-30s)

---

## 1. User Management

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/user/register` | POST | ✅ | `RegisterUser` | Auto-register user on first interaction |
| `/user/timeout` | GET | ✅ | `GetUserTimeout` | Check if user is timed out |

### Parameters
- **RegisterUser**: `platform`, `platform_id`, `username`
- **GetUserTimeout**: `username`

---

## 2. Inventory & Items

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/user/inventory` | GET | ✅ | `GetInventory` | Get user's inventory with optional filter |
| `/user/item/add` | POST | ✅ 🔒 | `AddItem` | Add items (admin/streamer only) |
| `/user/item/remove` | POST | ✅ 🔒 | `RemoveItem` | Remove items (admin/streamer only) |
| `/user/item/give` | POST | ✅ | `GiveItem` | Transfer item between users |
| `/user/item/use` | POST | ✅ | `UseItem` | Use item (lootboxes, etc.) |
| `/user/search` | POST | ✅ | `Search` | Search for items (daily cooldown) |

### Parameters
- **GetInventory**: `platform`, `platform_id`, `username`, `filter?` (optional: "resource", "lootbox", etc.)
- **AddItem/RemoveItem**: `platform`, `platform_id`, `username`, `item_name`, `quantity`
- **GiveItem**: `from_platform`, `from_platform_id`, `to_platform`, `to_platform_id`, `to_username`, `item_name`, `quantity`
- **UseItem**: `platform`, `platform_id`, `username`, `item_name`, `quantity`, `target_user?` (optional)
- **Search**: `platform`, `platform_id`, `username`

---

## 3. Economy

| Endpoint | Method | C# Status | Binding Name | Description  |
|----------|--------|-----------|--------------|-------------|
| `/user/item/buy` | POST | ✅ | `BuyItem` | Purchase item from shop |
| `/user/item/sell` | POST | ✅ | `SellItem` | Sell item for currency |
| `/prices` | GET | ✅ | `GetSellPrices` | Get current sell prices |
| `/prices/buy` | GET | ✅ | `GetBuyPrices` | Get current buy prices |

### Parameters
- **BuyItem/SellItem**: `platform`, `platform_id`, `username`, `item_name`, `quantity`

---

## 4. Crafting System

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/user/item/upgrade` | POST | ⚠️ | `UpgradeItem` | Craft item upgrade |
| `/user/item/disassemble` | POST | ✅ | `DisassembleItem` | Break down item for materials |
| `/recipes` | GET | ✅ | `GetRecipes` | Get available recipes |
| `/recipes/unlocked` | GET | ❌ | `GetUnlockedRecipes` | Get user's unlocked recipes |

###Parameters
- **UpgradeItem**: `platform`, `platform_id`, `username`, `item` (string, not recipe_id)
  - ⚠️ C# client uses `recipe_id` (int) - **NEEDS UPDATE to use `item` (string)**
- **DisassembleItem**: `platform`, `platform_id`, `username`, `item_name`, `quantity`
- **GetUnlockedRecipes**: `platform`, `platform_id`, `username`

> [!WARNING]
> **Breaking Change in UpgradeItem**
> 
> The C# client currently uses `recipe_id` (integer), but the API now expects `item` (string - item name).
> Update C# method signature to match Discord client.

---

## 5. Gamble System

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/gamble/start` | POST | ✅ | `StartGamble` | Start new gamble session |
| `/gamble/join` | POST | ✅ | `JoinGamble` | Join existing gamble |
| `/gamble/get` | GET | ✅ | `GetActiveGamble` | Get active gamble details |

### Parameters
- **StartGamble**: `platform`, `platform_id`, `username`, `bets` (array of {`item_name`, `quantity`})
- **JoinGamble**: `platform`, `platform_id`, `username`, `id` (query param), `bets`

---

## 6. Stats & Leaderboards

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/stats/event` | POST | ✅ | `RecordEvent` | Track user event (message, follow, etc.) |
| `/stats/user` | GET | ✅ | `GetUserStats` | Get user statistics |
| `/stats/system` | GET | ✅ | `GetSystemStats` | Get system-wide stats |
| `/stats/leaderboard` | GET | ✅ | `GetLeaderboard` | Get leaderboard |

### Parameters
- **RecordEvent**: `platform`, `platform_id`, `event_type`, `metadata?`
- **GetUserStats**: `platform`, `platform_id`, `period?` (optional: "daily", "weekly", "monthly", "all")
- **GetLeaderboard**: `metric?` (optional: "engagement_score"), `limit?` (default: 10)

---

## 7. Progression System

### User Actions

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/progression/tree` | GET | ✅ | `GetProgressionTree` | Get full progression tree |
| `/progression/available` | GET | ✅ | `GetAvailableNodes` | Get unlockable nodes |
| `/progression/status` | GET | ✅ | `GetProgressionStatus` | Get progression status |
| `/progression/vote` | POST | ✅ | `VoteForNode` | Vote for node unlock |
| `/progression/session` | GET | ✅ | `GetVotingSession` | Get current voting session |
| `/progression/unlock-progress` | GET | ✅ | `GetUnlockProgress` | Get unlock progress % |
| `/progression/engagement` | GET | ✅ | `GetUserEngagement` | Get user contribution points |
| `/progression/leaderboard` | GET | ✅ | `GetContributionLeaderboard` | Get contribution leaderboard |

### Admin Actions

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/progression/admin/unlock` | POST | ✅ 🔒 | `AdminUnlockNode` | Force unlock node |
| `/progression/admin/relock` | POST | ✅ 🔒 | `AdminRelockNode` | Re-lock node |
| `/progression/admin/instant-unlock` | POST | ✅ 🔒 | `AdminInstantUnlock` | Unlock vote leader immediately |
| `/progression/admin/start-voting` | POST | ✅ 🔒 | `AdminStartVoting` | Start new voting session |
| `/progression/admin/end-voting` | POST | ✅ 🔒 | `AdminEndVoting` | End current voting session |
| `/progression/admin/reset` | POST | ✅ 🔒 | `AdminResetProgression` | Reset entire progression tree |
| `/progression/admin/contribution` | POST | ✅ 🔒 | `AdminAddContribution` | Add contribution points |

### Parameters
- **VoteForNode**: `platform`, `platform_id`, `username`, `node_key`
- **GetUserEngagement**: `user_id`
- **AdminUnlockNode/RelockNode**: `node_key`, `level`
- **AdminResetProgression**: `reset_by`, `reason`, `preserve_user_progression` (bool)
- **AdminAddContribution**: `amount`

---

## 8. Jobs System

| Endpoint | Method | C# Status | Binding Name  | Description |
|----------|--------|-----------|--------------|-------------|
| `/jobs` | GET | ✅ | `GetAllJobs` | Get all available jobs |
| `/jobs/user` | GET | ✅ | `GetUserJobs` | Get user's job progress |
| `/jobs/award-xp` | POST | ✅ 🔒 | `AwardJobXP` | Award XP (admin/streamer) |
| `/jobs/bonus` | GET | ✅ | `GetJobBonus` | Get active job bonus |

### Parameters
- **GetUserJobs**: `platform`, `platform_id`
- **AwardJobXP**: `platform`, `platform_id`, `username`, `job_name`, `xp_amount`
- **GetJobBonus**: `user_id`, `job_key`, `bonus_type`

---

## 9. Account Linking

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/link/initiate` | POST | ✅ | `InitiateLinking` | Start account linking |
| `/link/claim` | POST | ✅ | `ClaimLinkingCode` | Claim linking code |
| `/link/confirm` | POST | ✅ | `ConfirmLinking` | Confirm linking |
| `/link/unlink` | POST | ✅ | `UnlinkAccounts` | Unlink accounts |
| `/link/status` | GET | ✅ | `GetLinkingStatus` | Get linking status |

### Parameters
- **InitiateLinking**: `platform`, `platform_id`, `username`
- **ClaimLinkingCode**: `platform`, `platform_id`, `username`, `code`
- **ConfirmLinking/UnlinkAccounts**: `platform`, `platform_id`
- **GetLinkingStatus**: `platform`, `platform_id`

---

## 10. Message Handler (Convenience)

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/message/handle` | POST | ✅ | `HandleMessage` | All-in-one message processor |

### Parameters
- **HandleMessage**: `platform`, `platform_id`, `username`, `message`

> [!TIP]
> **Use for Chat Integration**
> 
> This endpoint handles engagement tracking, command detection, and rewards in a single call.
> Perfect for Twitch/YouTube chat integrations.

---

## 11. Admin Utilities

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/admin/reload-aliases` | POST | ✅ 🔒 | `ReloadAliases` | Reload item name aliases |
| `/test` | POST | ✅ | `Test` | Debug test endpoint |

---

## 12. Health Checks

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/healthz` | GET | ✅ | `HealthCheck` | Basic health check |
| `/readyz` | GET | ✅ | `ReadyCheck` | Ready check (includes DB) |

---

## 13. Real-time Events (SSE)

| Endpoint | Method | C# Status | Binding Name | Description |
|----------|--------|-----------|--------------|-------------|
| `/events` | GET (SSE) | ✅ | `BrandishBotSSE` | Server-Sent Events stream |

### SSE Event Types
- `job.level_up` - User leveled up a job
- `progression.voting_started` - New voting session started
- `progression.cycle_completed` - Node unlocked + new voting session

### Parameters
- **Connect**: `types` (optional query param) - comma-separated list of event types to filter

### C# Usage Example
```csharp
var sseClient = new BrandishBotSSE("http://localhost:8080", "api-key", new[] {
    SSEEventType.JobLevelUp,
    SSEEventType.VotingStarted,
    SSEEventType.CycleCompleted
});

sseClient.OnJobLevelUp += (sender, evt) => {
    var payload = evt.GetPayload<JobLevelUpPayload>();
    Console.WriteLine($"{payload.JobKey} leveled up to {payload.NewLevel}!");
};

sseClient.OnVotingStarted += (sender, evt) => {
    var payload = evt.GetPayload<VotingStartedPayload>();
    Console.WriteLine($"New voting session with {payload.Options.Length} options");
};

sseClient.Start(); // Non-blocking, runs in background with auto-reconnect
```

### Discord Bot Configuration
Set `DISCORD_NOTIFICATION_CHANNEL_ID` environment variable to enable SSE notifications in Discord.

---

## Implementation Checklist by Client

### C# Client (`BrandishBotClient.cs`)

- [ ] **Update `UpgradeItem` signature** to use string `item` instead of int `recipe_id`
  ```csharp
  // OLD (incorrect)
  public async Task<string> UpgradeItem(..., int recipeId)
  
  // NEW (correct)
  public async Task<string> UpgradeItem(..., string itemName, int quantity)
  ```

- [ ] **Add `GetUnlockedRecipes` method**
  ```csharp
  public async Task<string> GetUnlockedRecipes(string platform, string platformId, string username)
  {
      var query = BuildQuery(
          "platform=" + platform,
          "platform_id=" + platformId,
          "username=" + username
      );
      return await GetAsync("/recipes/unlocked" + query);
  }
  ```

- [ ] **Test all endpoints** against latest API
- [ ] **Update documentation** with correct signatures

### Discord Client (Go - `internal/discord/client.go`)

- ✅ All methods implemented and up-to-date
- ✅ Using string-based item names
- ✅ Retry logic implemented

---

## Common Parameters Reference

### Platform Values
- `"twitch"` - Twitch platform
- `"youtube"` - YouTube platform
- `"discord"` - Discord platform

### Item Names (Public Names)
- `"money"` - Currency
- `"junkbox"` - Tier 0 lootbox
- `"lootbox"` - Tier 1 lootbox
- `"goldbox"` - Tier 2 lootbox
- `"missile"` - Blaster/Ray Gun

### Event Types
- `"message"` - Chat message
- `"follow"` - New follower
- `"subscribe"` - Subscription
- `"raid"` - Raid
- `"bits"` - Bits/cheers
- `"gift"` - Gift subscription

### Filter Types (for GetInventory)
- `"resource"` - Resource items only
- `"lootbox"` - Lootbox items only
- `"consumable"` - Consumable items
- `"upgrade"` - Upgrade materials

---

## Error Handling Recommendations

### HTTP Status Codes
- `200` - Success
- `201` - Created (e.g., new user registered)
- `400` - Bad request (invalid parameters)
- `401` - Unauthorized (missing/invalid API key)
- `404` - Not found (item, user, etc.)
- `429` - Rate limited (cooldown active)
- `500` - Server error (retry recommended)

### Error Response Format
```json
{
  "error": "Human-readable error message"
}
```

### Recommended Retry Logic
- Retry on `5xx` errors (server issues)
- **DO NOT** retry on `4xx` errors (client errors)
- Exponential backoff: 500ms, 1s, 2s
- Max 3 retries

---

## Testing Checklist

### Unit Tests
- [ ] Serialize/deserialize request bodies correctly
- [ ] Query string building works
- [ ] Headers set correctly (X-API-Key)
- [ ] Retry logic functions properly

### Integration Tests
- [ ] Can register user
- [ ] Can search for items
- [ ] Can buy/sell items
- [ ] Can use lootboxes
- [ ] Can vote for progression
- [ ] Error handling works correctly

### End-to-End Tests
- [ ] Full user journey (register → search → open lootbox → sell items)
- [ ] Progression voting cycle
- [ ] Account linking flow

---

## API Versioning Notes

> [!NOTE]
> The API does not currently use versioning (`/v1/` prefix).
> 
> If versioning is added in the future, update base URL to include version:
> - Old: `http://localhost:8080`
> - New: `http://localhost:8080/v1`

---

## Rate Limiting

| Endpoint | Cooldown | Notes |
|----------|----------|-------|
| `/user/search` | 10 minutes | Per user, reduced after 6 daily searches |
| `/progression/vote` | Once per session | Can't change vote |
| `/gamble/*` | Join window only | 2 minutes to join after start |

Handle `429 Too Many Requests` gracefully with user-friendly messages.

---

## Security Considerations

1. **Never log API keys** - Sensitive credentials
2. **Use HTTPS in production** - Encrypt traffic
3. **Validate user input** - Prevent injection
4. **Rate limit client-side** - Don't spam API
5. **Handle errors gracefully** - Don't expose internals

---

## Streamer.bot Specific Notes

### Initialization Pattern
```csharp
// In a "Load" action (runs once at startup)
BrandishBotClient.Initialize("http://localhost:8080", "your-api-key-here");

// In any other action
if (!BrandishBotClient.IsInitialized) 
{
    CPH.SendMessage("BrandishBot not initialized!");
    return;
}

var client = BrandishBotClient.Instance;
var result = await client.Search("twitch", userId, userName);
```

### Singleton Benefits
- ✅ One HTTP client reused across all actions
- ✅ Connection pooling and performance
- ✅ Consistent configuration
- ❌ Static state persists between action runs (feature, not bug)

---

## Quick Start Example

```csharp
// 1. Initialize (once, at startup)
BrandishBotClient.Initialize("http://localhost:8080", "your-api-key");

var client = BrandishBotClient.Instance;

// 2. Handle chat message (Twitch integration)
var result = await client.HandleMessage(
    Platform.Twitch, 
    userId, 
    username, 
    "!search"
);
CPH.SendMessage(result);

// 3. Get inventory
var inventory = await client.GetInventory(Platform.Twitch, userId);

// 4. Vote for progression
var voteResult = await client.VoteForNode(
    Platform.Twitch, 
    userId, 
    "upgrade_contribution_boost"
);
```

---

## Summary

| Category | Total Endpoints | C# Implemented | Needs Update |
|----------|----------------|----------------|--------------|
| User Management | 2 | 2 (100%) | 0 |
| Inventory & Items | 6 | 6 (100%) | 0 |
| Economy | 4 | 4 (100%) | 0 |
| Crafting | 4 | 3 (75%) | 1 (GetUnlockedRecipes) |
| Gamble | 3 | 3 (100%) | 0 |
| Stats | 4 | 4 (100%) | 0 |
| Progression (User) | 8 | 8 (100%) | 0 |
| Progression (Admin) | 7 | 7 (100%) | 0 |
| Jobs | 4 | 4 (100%) | 0 |
| Account Linking | 5 | 5 (100%) | 0 |
| Message Handler | 1 | 1 (100%) | 0 |
| Admin Utils | 2 | 2 (100%) | 0 |
| Health Checks | 2 | 2 (100%) | 0 |
| Real-time Events (SSE) | 1 | 1 (100%) | 0 |
| **TOTAL** | **53** | **52 (98%)** | **1** |

### Action Items
1. ⚠️ Update `UpgradeItem` to use string item name instead of int recipe_id
2. ❌ Add `GetUnlockedRecipes` method
3. ✅ All other endpoints properly bound

---

## Related Documentation
- [Production Deployment Strategy](file:///home/osse1/projects/BrandishBot_Go/docs/deployment/PRODUCTION_STRATEGY.md)
- [API Routes Reference](file:///home/osse1/projects/BrandishBot_Go/cmd/app/main.go)
- [C# Client Source](file:///home/osse1/projects/BrandishBot_Go/client/csharp/BrandishBotClient.cs)
- [Discord Client Reference](file:///home/osse1/projects/BrandishBot_Go/internal/discord/client.go)

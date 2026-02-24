# API Coverage Reference

> **Last Updated:** Feb 2026
> **Maintainer:** Development Team  
> **Purpose:** Master reference for maintaining API consistency across Discord, Server, and C# Client

## Quick Reference

**Current Coverage**: 97% Streamer.bot, 100% Discord, 100% API endpoints

This document is the **single source of truth** for keeping these three systems synchronized:

1. **API Endpoints** - [`internal/server/server.go`](../../internal/server/server.go)
2. **Discord Commands** - [`internal/discord/cmd_*.go`](../../internal/discord/)
3. **C# Client/Wrapper** - [`client/csharp/`](../../client/csharp/)

---

## How to Use This Document

### When Adding a New Feature

1. **Add API endpoint** to `server.go` first
2. **Add row** to the appropriate table below
3. **Implement Discord command** in `cmd_*.go` file
4. **Add C# client method** to `BrandishBotClient.cs` (or partial)
5. **Add C# wrapper** (if needed) to `BrandishBotWrapper.cs`
6. **Update checkmarks** in this document

### Legend

- ✅ = Implemented
- ❌ = Not implemented
- 🔒 = Admin-only
- 🎯 = Public endpoint (no auth required)

---

## API Endpoints by Route Group

### Health & Version

| API Endpoint      | Discord | C# Client | C# Wrapper | Notes           |
| ----------------- | ------- | --------- | ---------- | --------------- |
| `GET /healthz` 🎯 | —       | ✅        | ✅         | Liveness probe  |
| `GET /readyz` 🎯  | —       | ✅        | ✅         | Readiness probe |
| `GET /version` 🎯 | —       | ✅        | ✅         | Version info    |
| `GET /metrics` 🎯 | —       | ❌        | ❌         | Prometheus only |

### Info (`/api/v1/info`)

| API Endpoint      | Discord | C# Client | C# Wrapper | Notes           |
| ----------------- | ------- | --------- | ---------- | --------------- |
| `GET /info` 🎯    | `/info` | ✅        | ✅         | System info     |

### User Management (`/api/v1/user`)

| API Endpoint                      | Discord          | C# Client | C# Wrapper | Notes             |
| --------------------------------- | ---------------- | --------- | ---------- | ----------------- |
| `POST /user/register`             | Auto             | ✅        | ✅         | Auto-registration |
| `GET /user/timeout`               | `/check-timeout` | ✅        | ✅         | Timeout status    |
| `PUT /user/timeout`               | `/timeout`       | ✅        | ✅         | Set timeout       |
| `GET /user/inventory`             | `/inventory`     | ✅        | ✅         | With filters      |
| `GET /user/inventory-by-username` | —                | ✅        | Auto       | Username lookup   |
| `POST /user/search`               | `/search`        | ✅        | ✅         | Find items        |

### Items (`/api/v1/user/item`)

| API Endpoint                    | Discord              | C# Client | C# Wrapper | Notes          |
| ------------------------------- | -------------------- | --------- | ---------- | -------------- |
| `POST /item/add`                | —                    | ✅        | ❌         | Admin          |
| `POST /item/add-by-username`    | `/admin-add-item` 🔒 | ✅        | ✅         | Admin          |
| `POST /item/remove`             | —                    | ✅        | ❌         | Admin          |
| `POST /item/remove-by-username` | `/admin-remove-item` 🔒| ✅      | ✅         | Admin          |
| `POST /item/give`               | `/give`              | ✅        | ✅         | Transfer items |
| `POST /item/sell`               | `/sell`              | ✅        | ✅         | Sell items     |
| `POST /item/buy`                | `/buy`               | ✅        | ✅         | Buy from shop  |
| `POST /item/use`                | `/use`               | ✅        | ✅         | Use consumable |
| `POST /item/upgrade`            | `/upgrade`           | ✅        | ✅         | Craft upgrade  |
| `POST /item/disassemble`        | `/disassemble`       | ✅        | ✅         | Break down     |

### Economy & Crafting

| API Endpoint      | Discord        | C# Client | C# Wrapper | Notes       |
| ----------------- | -------------- | --------- | ---------- | ----------- |
| `GET /recipes`    | `/recipes`     | ✅        | ✅         | All recipes |
| `GET /prices`     | `/prices-sell` | ✅        | ✅         | Sell prices |
| `GET /prices/buy` | `/prices`      | ✅        | ✅         | Buy prices  |

### Gambling & Slots

| API Endpoint         | Discord         | C# Client | C# Wrapper | Notes         |
| -------------------- | --------------- | --------- | ---------- | ------------- |
| `POST /gamble/start` | `/gamble-start` | ✅        | ✅         | Start session |
| `POST /gamble/join`  | `/gamble-join`  | ✅        | ✅         | Join session  |
| `GET /gamble/get`    | —               | ✅        | ✅         | View active   |
| `GET /gamble/active` | —               | ✅        | ✅         | Get active    |
| `POST /slots/spin`   | `/slots`        | ✅        | ✅         | Play slots    |

### Expeditions (`/api/v1/expedition`)

| API Endpoint           | Discord              | C# Client | C# Wrapper | Notes             |
| ---------------------- | -------------------- | --------- | ---------- | ----------------- |
| `POST /expedition/start` | `/explore`         | ✅        | ✅         | Start expedition  |
| `POST /expedition/join`  | `/explore`         | ✅        | ✅         | Join expedition   |
| `GET /expedition/get`    | —                  | ✅        | ✅         | Get details       |
| `GET /expedition/active` | —                  | ✅        | ✅         | Get active        |
| `GET /expedition/journal`| `/expedition-journal`| ✅      | ✅         | Get journal       |
| `GET /expedition/status` | —                  | ✅        | ✅         | System status     |

### Farming (`/api/v1/harvest`, `/api/v1/compost`)

| API Endpoint             | Discord             | C# Client | C# Wrapper | Notes             |
| ------------------------ | ------------------- | --------- | ---------- | ----------------- |
| `POST /harvest`          | `/harvest`          | ✅        | ✅         | Harvest crops     |
| `POST /compost/deposit`  | `/compost-deposit`  | ✅        | ✅         | Add to compost    |
| `POST /compost/harvest`  | `/compost-harvest`  | ✅        | ✅         | Harvest compost   |
| `GET /compost/status`    | `/compost-status`   | ✅        | ✅         | Compost status    |

### Stats (`/api/v1/stats`)

| API Endpoint             | Discord        | C# Client | C# Wrapper | Notes        |
| ------------------------ | -------------- | --------- | ---------- | ------------ |
| `POST /stats/event`      | —              | ✅        | ❌         | Background   |
| `GET /stats/user`        | `/stats`       | ✅        | ✅         | User stats   |
| `GET /stats/system`      | —              | ✅        | ✅         | System stats |
| `GET /stats/leaderboard` | `/leaderboard` | ✅        | ✅         | Rankings     |

### Jobs (`/api/v1/jobs`)

| API Endpoint          | Discord      | C# Client | C# Wrapper | Notes         |
| --------------------- | ------------ | --------- | ---------- | ------------- |
| `GET /jobs`           | —            | ✅        | ✅         | All jobs      |
| `GET /jobs/user`      | —            | ✅        | ✅         | User progress |
| `POST /jobs/award-xp` | —            | ✅        | ✅         | Award XP      |
| `GET /jobs/bonus`     | `/job-bonus` | ✅        | ✅         | Job bonuses   |

### Quests (`/api/v1/quests`)

| API Endpoint             | Discord             | C# Client | C# Wrapper | Notes             |
| ------------------------ | ------------------- | --------- | ---------- | ----------------- |
| `GET /quests/active`     | `/quests`           | ✅        | ✅         | Active quests     |
| `GET /quests/progress`   | `/quests`           | ✅        | ✅         | User progress     |
| `POST /quests/claim`     | `/claimquest`       | ✅        | ✅         | Claim reward      |

### Progression (`/api/v1/progression`)

| API Endpoint                       | Discord            | C# Client | C# Wrapper | Notes          |
| ---------------------------------- | ------------------ | --------- | ---------- | -------------- |
| `GET /progression/tree`            | —                  | ✅        | ✅         | Full tree      |
| `GET /progression/available`       | —                  | ✅        | ✅         | Unlockable     |
| `POST /progression/vote`           | `/vote`            | ✅        | ✅         | Vote for node  |
| `GET /progression/status`          | —                  | ✅        | ✅         | Global status  |
| `GET /progression/engagement`      | `/engagement`      | ✅        | ✅         | Contributions  |
| `GET /progression/engagement-by-username`| —            | ✅        | ✅         | Lookup contrib |
| `GET /progression/leaderboard`     | —                  | ✅        | ✅         | Rankings       |
| `GET /progression/session`         | `/voting-session`  | ✅        | ✅         | Voting session |
| `GET /progression/unlock-progress` | `/unlock-progress` | ✅        | ✅         | Progress       |
| `GET /progression/estimate/{nodeKey}`| —                | ✅        | ✅         | Cost estimate  |

### Progression Admin (`/api/v1/progression/admin`) 🔒

| API Endpoint                 | Discord                  | C# Client | C# Wrapper | Notes          |
| ---------------------------- | ------------------------ | --------- | ---------- | -------------- |
| `POST /admin/unlock`         | `/admin-unlock`          | ✅        | ✅         | Force unlock   |
| `POST /admin/unlock-all`     | —                        | ✅        | ✅         | Unlock all     |
| `POST /admin/relock`         | `/admin-relock`          | ✅        | ✅         | Force relock   |
| `POST /admin/instant-unlock` | `/admin-instant-resolve` | ✅        | ✅         | Instant unlock |
| `POST /admin/start-voting`   | `/admin-start-voting`    | ✅        | ✅         | Start voting   |
| `POST /admin/end-voting`     | `/admin-end-voting`      | ✅        | ✅         | End voting     |
| `POST /admin/force-end-voting`| —                       | ✅        | ✅         | Force end      |
| `POST /admin/reset`          | `/admin-reset-tree`      | ✅        | ✅         | Reset tree     |
| `POST /admin/contribution`   | `/admin-contribution`    | ✅        | ✅         | Add points     |

### Account Linking (`/api/v1/link`)

| API Endpoint          | Discord              | C# Client | C# Wrapper | Notes         |
| --------------------- | -------------------- | --------- | ---------- | ------------- |
| `POST /link/initiate` | `/link`              | ✅        | ✅         | Generate code |
| `POST /link/claim`    | `/link [token]`      | ✅        | ✅         | Claim code    |
| `POST /link/confirm`  | `/link confirm:true` | ✅        | ✅         | Confirm link  |
| `POST /link/unlink`   | `/unlink`            | ✅        | ✅         | Unlink        |
| `GET /link/status`    | —                    | ✅        | ✅         | Link status   |

### Prediction (`/api/v1/prediction`)

| API Endpoint             | Discord             | C# Client | C# Wrapper | Notes             |
| ------------------------ | ------------------- | --------- | ---------- | ----------------- |
| `POST /prediction`       | —                   | ✅        | ✅         | Process outcome   |

### Subscriptions (`/api/v1/subscriptions`)

| API Endpoint                 | Discord | C# Client | C# Wrapper | Notes               |
| ---------------------------- | ------- | --------- | ---------- | ------------------- |
| `POST /subscriptions/event`  | —       | ✅        | ❌         | Webhook handler     |
| `GET /subscriptions/user`    | —       | ✅        | ❌         | User subscription   |

### Events (`/api/v1/events`)

| API Endpoint      | Discord | C# Client | C# Wrapper | Notes           |
| ----------------- | ------- | --------- | ---------- | --------------- |
| `GET /events`     | —       | ✅        | ❌         | SSE Stream      |

### Admin Utilities (`/api/v1/admin`) 🔒

| API Endpoint                             | Discord                 | C# Client | C# Wrapper | Notes          |
| ---------------------------------------- | ----------------------- | --------- | ---------- | -------------- |
| `POST /admin/reload-aliases`             | —                       | ✅        | ✅         | Reload aliases |
| `POST /admin/job/award-xp`               | `/admin-award-xp`       | ✅        | ✅         | Admin XP       |
| `POST /admin/job/reset-daily-xp`         | `/admin-reset-daily`    | ✅        | ✅         | Manual reset   |
| `GET /admin/job/reset-status`            | `/admin-reset-status`   | ✅        | ✅         | Reset status   |
| `POST /admin/progression/reload-weights` | `/admin-reload-weights` | ✅        | ✅         | Reload cache   |
| `GET /admin/cache/stats`                 | `/admin-cache-stats`    | ✅        | ✅         | Cache stats    |
| `GET /admin/metrics`                     | `/admin-metrics`        | ✅        | ✅         | Metrics        |
| `POST /admin/sse/broadcast`              | —                       | ✅        | ✅         | Broadcast msg  |
| `GET /admin/users/lookup`                | `/admin-user`           | ✅        | ✅         | User info      |
| `GET /admin/users/recent`                | `/admin-users-recent`   | ✅        | ✅         | Recent users   |
| `GET /admin/users/active`                | `/admin-users-active`   | ✅        | ✅         | Active chat    |
| `GET /admin/items`                       | (Autocomplete)          | ✅        | ✅         | Item list      |
| `GET /admin/jobs`                        | (Autocomplete)          | ✅        | ✅         | Job list       |
| `GET /admin/events`                      | `/admin-events`         | ✅        | ✅         | System events  |
| `POST /admin/timeout/clear`              | —                       | ✅        | ✅         | Clear timeout  |
| `GET /admin/simulate/capabilities`       | `/admin-simulation`     | ✅        | ✅         | Sim capability |
| `GET /admin/simulate/scenarios`          | `/admin-simulation`     | ✅        | ✅         | Sim scenarios  |
| `POST /admin/simulate/run`               | `/admin-simulation`     | ✅        | ✅         | Run sim        |
| `POST /admin/simulate/run-custom`        | —                       | ✅        | ✅         | Run custom     |
| `GET /admin/simulate/scenario`           | —                       | ✅        | ✅         | Get scenario   |

### Other

| API Endpoint           | Discord | C# Client | C# Wrapper | Notes        |
| ---------------------- | ------- | --------- | ---------- | ------------ |
| `POST /message/handle` | —       | ✅        | ✅         | Chat handler |
| `POST /test`           | —       | ✅        | ✅         | Debug        |

---

## Coverage Statistics

| System                 | Total | Complete | Missing | %    |
| ---------------------- | ----- | -------- | ------- | ---- |
| **API Endpoints**      | 128   | 128      | 0       | 100% |
| **Discord Commands**   | 60    | 60       | 0       | 100% |
| **C# Client Methods**  | 94    | 94       | 0       | 100% |
| **C# Wrapper Methods** | 67    | 65       | 2       | 97%  |

### Missing Items

**C# Wrapper** (Low Priority):

- `RecordEvent()` - Internal background tracking
- `GiveItemByUsername()` - Backend not implemented

---

## Implementation Guide

### Adding New Endpoint

```go
// 1. server.go
r.Post("/api/v1/feature/action", handler.HandleAction(service))

// 2. handler/feature.go
func HandleAction(service Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation
    }
}
```

### Adding Discord Command

```go
// internal/discord/cmd_feature.go
func FeatureCommand() (*discordgo.ApplicationCommand, CommandHandler) {
    cmd := &discordgo.ApplicationCommand{
        Name:        "feature",
        Description: "Do something",
    }
    handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
        // Call client.FeatureAction()
    }
    return cmd, handler
}
```

### Adding C# Client Method

```csharp
// BrandishBotClient.cs
public async Task<string> FeatureAction(params...)
{
    return await PostJsonAsync("/api/v1/feature/action", new { ... });
}
```

### Adding C# Wrapper (Optional)

```csharp
// BrandishBotWrapper.cs
public bool FeatureAction()
{
    EnsureInitialized();
    // Extract args, call client, set response
}
```

---

## Maintenance

**Update this document when:**

- Adding new API endpoints
- Adding Discord commands
- Adding C# client methods
- Deprecating features
- Changing endpoint paths

**Verification Commands:**

```bash
# Count API endpoints
grep -E "r\.(Post|Get|Put|Delete|Patch)" internal/server/server.go | wc -l

# Count Discord commands (explicit + helper created)
expr $(grep "&discordgo.ApplicationCommand{" internal/discord/cmd_*.go | wc -l) + $(grep "CreateItemQuantityCommand" internal/discord/cmd_*.go | wc -l)

# Count C# client methods
grep "public async Task" client/csharp/*.cs | wc -l

# Count C# wrapper methods
grep "public bool" client/csharp/BrandishBotWrapper.cs | wc -l
```

---

**Document Version**: 1.3
**Last Review**: Feb 2026

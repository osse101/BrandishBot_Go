# Event Catalog

This document catalogs all events in the BrandishBot event system, their schemas, and usage.

## Quick Reference

| Event Type | Category | Source | When Emitted |
|------------|----------|--------|--------------|
| `job_level_up` | Progression | Job Service | User's job level increases |
| `progression.cycle.completed` | Progression | Progression Service | Community vote cycle completes |
| `user_registered` | User | User Service | New user registers |
| `item_added` | Inventory | User Service | Item added to inventory |
| `item_removed` | Inventory | User Service | Item removed from inventory |
| `item_used` | Inventory | User Service | Consumable item used |
| `item_sold` | Economy | Economy Service | Item sold to shop |
| `item_bought` | Economy | Economy Service | Item purchased from shop |
| `item_transferred` | Inventory | User Service | Item transferred to another user |
| `message_received` | Chat | Handler | User sends message |
| `search` | Activity | User Service | User performs search action |
| `search_near_miss` | Activity | User Service | Search almost succeeded |
| `search_critical_fail` | Activity | User Service | Search critically fails |
| `search_critical_success` | Activity | User Service | Search critically succeeds |
| `gamble_near_miss` | Gambling | Gamble Service | Gamble almost won |
| `gamble_tie_break_lost` | Gambling | Gamble Service | Lost tie-breaker in gamble |
| `gamble_critical_fail` | Gambling | Gamble Service | Gamble critically fails |
| `daily_streak` | Engagement | Stats Service | User maintains daily streak |
| `crafting_critical_success` | Crafting | Crafting Service | Crafting critically succeeds |
| `crafting_perfect_salvage` | Crafting | Crafting Service | Perfect salvage while disassembling |
| `lootbox_jackpot` | Lootbox | Lootbox Service | Lootbox jackpot won |
| `lootbox_big_win` | Lootbox | Lootbox Service | Big win from lootbox |

---

## Event Schemas

### job_level_up

**Emitted when:** A user's job level increases  
**Source:** `internal/job/service.go`  
**Published via:** ResilientPublisher (fire-and-forget with retry)

**Payload Schema:**
```json
{
  "user_id": "string (UUID)",
  "job_key": "string (explorer|blacksmith)",  
  "new_level": "integer",
  "old_level": "integer"
}
```

**Metadata:**
```json
{
  "source": "string (e.g. 'search', 'crafting')"
}
```

**Subscribers:**
- Stats Service: Records level-up statistics
- Event Log: Persists event for audit trail
- Metrics Collector: Tracks level-up metrics

**Example:**
```go
// Publishing
eventType := event.Type(domain.EventJobLevelUp)
publisher.PublishWithRetry(ctx, event.Event{
    Type: eventType,
    Payload: map[string]interface{}{
        "user_id":   "user123",
        "job_key":   "explorer",
        "new_level": 5,
        "old_level": 4,
    },
    Metadata: map[string]interface{}{
        "source": "search",
    },
})
```

---

### progression.cycle.completed

**Emitted when:** A community progression vote cycle completes  
**Source:** `internal/progression/service.go`  
**Published via:** Event Bus (direct)

**Payload Schema:**
```json
{
  "cycle_id": "integer",
  "winner_node_id": "integer",
  "vote_count": "integer",
  "timestamp": "RFC3339 string"
}
```

**Subscribers:**
- Progression Notifier: Sends notifications to Discord and Streamer.bot
- Event Log: Persists event for audit trail
- Metrics Collector: Tracks cycle metrics

---

### progression.target.set

**Emitted when:** A new progression target is selected (either automatically or by admin) without a voting session
**Source:** `internal/progression/voting_sessions.go`

**Payload Schema:**
```json
{
  "node_key": "string",
  "target_level": "integer",
  "auto_selected": "boolean",
  "session_id": "integer"
}
```

**Subscribers:**
- Progression Notifier: Updates UI/Stream overlay to show new target
- Event Log: Persists event for audit trail

---

### user_registered

**Emitted when:** A new user registers in the system  
**Source:** `internal/user/service.go`

**Payload Schema:**
```json
{
  "user_id": "string (UUID)",
  "platform": "string (twitch|youtube|discord)",
  "platform_user_id": "string",
  "username": "string",
  "timestamp": "RFC3339 string"
}
```

**Subscribers:**
- Stats Service: Records registration event
- Event Log: Persists for audit

---

### item_added

**Emitted when:** Item(s) added to user's inventory  
**Source:** `internal/user/service.go`

**Payload Schema:**
```json
{
  "user_id": "string",
  "item_name": "string",
  "quantity": "integer",
  "source": "string (e.g. 'search', 'purchase', 'craft')"
}
```

---

### item_removed

**Emitted when:** Item(s) removed from user's inventory  
**Source:** `internal/user/service.go`

**Payload Schema:**
```json
{
  "user_id": "string",
  "item_name": "string",
  "quantity": "integer",
  "reason": "string (e.g. 'sold', 'used', 'transferred')"
}
```

---

### item_used

**Emitted when:** User uses a consumable item  
**Source:** `internal/user/service.go`

**Payload Schema:**
```json
{
  "user_id": "string",
  "item_name": "string",
  "quantity": "integer",
  "effect": "string (description of what happened)",
  "target_user_id": "string (optional, for items affecting others)"
}
```

---

### search / search_* Events

**Source:** `internal/user/service.go`

**Common Payload:**
```json
{
  "user_id": "string",
  "outcome": "string (normal|near_miss|critical_fail|critical_success)",
  "item_found": "string (optional)",
  "quantity": "integer (optional)",
  "roll": "integer (dice roll result)"
}
```

**Event Types:**
- `search`: Normal search attempt
- `search_near_miss`: Almost found something rare
- `search_critical_fail`: Critically failed search
- `search_critical_success`: Found extra loot

---

### gamble_* Events

**Source:** `internal/gamble/service.go`

**Common Payload:**
```json
{
  "user_id": "string",
  "gamble_id": "integer",
  "outcome": "string",
  "participant_count": "integer"
}
```

**Event Types:**
- `gamble_near_miss`: Almost won
- `gamble_tie_break_lost`: Lost tie-breaker
- `gamble_critical_fail`: Spectacularly failed

---

### crafting_* Events

**Source:** `internal/crafting/service.go`

**Payload:**
```json
{
  "user_id": "string",
  "recipe_id": "integer",
  "item_crafted": "string",
  "materials_used": "array of strings"
}
```

**Event Types:**
- `crafting_critical_success`: Extra output or bonus
- `crafting_perfect_salvage`: Recovered all materials perfectly

---

### lootbox_* Events

**Source:** `internal/lootbox/service.go`

**Payload:**
```json
{
  "user_id": "string",
  "lootbox_tier": "integer",
  "items_received": "array of {item: string, quantity: integer}",
  "total_value": "integer"
}
```

**Event Types:**
- `lootbox_jackpot`: Hit the jackpot
- `lootbox_big_win`: Significant win

---

## Event Template

When adding a new event, use this template:

```markdown
### event_name

**Emitted when:** [Trigger condition]  
**Source:** [File path]  
**Published via:** [Event Bus | ResilientPublisher]

**Payload Schema:**
```json
{
  "field1": "type and description",
  "field2": "type and description"
}
```

**Metadata:** (if applicable)
```json
{
  "meta_field": "type and description"
}
```

**Subscribers:**
- Service 1: Purpose
- Service 2: Purpose

**Example:**
```go
// Publishing code example
```
```

---

## Notes

### Critical Events (use ResilientPublisher)

The following events should ALWAYS use `ResilientPublisher.PublishWithRetry` to ensure delivery:
- `job_level_up` - Critical for user progression
- Any event that affects user state or rewards

### Fire-and-Forget Pattern

All event publishing follows the fire-and-forget pattern:
- **Domain operations never fail** due to event publishing errors
- Events retry asynchronously with exponential backoff
- Failed events (after retries) logged to dead-letter file
- See [Architecture Documentation](../architecture/event_system.md) for details

### Monitoring

Dead-letter events should be monitored:
```bash
tail -f logs/event_deadletter.jsonl
```

Each line is a JSON object with:
- `timestamp`: When the event finally failed
- `event`: The original event object
- `attempts`: Number of retry attempts
- `last_error`: Final error message

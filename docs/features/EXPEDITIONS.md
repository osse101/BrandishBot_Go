# Expedition System

The expedition system is a cooperative multiplayer adventure feature. Players form parties and embark on multi-turn expeditions where encounters are generated procedurally from a JSON configuration file. Each turn involves a random encounter, a probability-based skill check tied to the party's job levels, and a narrative outcome. The expedition ends when the party reaches turn 50 (win), hits 100 fatigue (exhaustion), or all members are knocked out.

---

## Table of Contents

- [Gameplay Flow](#gameplay-flow)
- [Architecture Summary](#architecture-summary)
- [Skill System](#skill-system)
- [Encounter Engine](#encounter-engine)
- [encounters.json Reference](#encountersjson-reference)
- [API Endpoints](#api-endpoints)
- [SSE Events](#sse-events)
- [Discord Commands](#discord-commands)
- [Reward Distribution](#reward-distribution)

---

## Gameplay Flow

1. A user starts an expedition (`/explore` or `POST /api/v1/expedition/start`)
2. A **3-minute join window** opens; other users can join the party
3. After the deadline, the **expedition worker** triggers execution automatically
4. The engine runs up to **50 turns**, generating encounters and narratives
5. Journal entries are saved to the DB and streamed via SSE in real time
6. Rewards are distributed to all participants
7. A **15-minute global cooldown** begins before the next expedition can start

### End Conditions

| Condition | Result |
|-----------|--------|
| Turn 50 reached | **Win** — conscious members get bonus rewards |
| Fatigue reaches 100 | **Exhaustion** — expedition ends early |
| All members knocked out | **Total KO** — expedition ends early |

---

## Architecture Summary

```
Discord / Streamer.bot
         |
    SSE Events (expedition.started, expedition.turn, expedition.completed)
         |
   SSE Hub ← Event Bus ← Expedition Service
                              |
                     Expedition Engine (pure logic)
                              |
                     encounters.json (config)
```

### Layer Breakdown

| Layer | Location | Responsibility |
|-------|----------|----------------|
| Domain types | `internal/domain/expedition.go` | State, turn, result, reward, journal structs |
| Config loader | `internal/expedition/config.go` | Loads and validates `encounters.json` |
| Engine | `internal/expedition/engine.go` | Pure-logic turn loop, no DB dependencies |
| Skills | `internal/expedition/skills.go` | Probability-based skill checks using job levels |
| Encounters | `internal/expedition/encounters.go` | Weighted encounter/outcome selection with progressive inversion |
| Scaling | `internal/expedition/scaling.go` | Party-size scaling for KO/revive effects |
| Journal | `internal/expedition/journal.go` | 3-part narrative generation from templates |
| Journal format | `internal/expedition/journal_format.go` | Plain text and structured output formatting |
| Service | `internal/expedition/service.go` | Orchestrates execution, rewards, DB persistence, events |
| Repository | `internal/repository/expedition.go` | Interface for DB operations |
| Postgres impl | `internal/database/postgres/expedition.go` | SQLC-backed repository implementation |
| Worker | `internal/worker/expedition_worker.go` | Timer-based execution scheduler |
| Handler | `internal/handler/expedition.go` | HTTP API handlers |
| Discord commands | `internal/discord/cmd_expedition.go` | `/explore` and `/expedition-journal` |
| Discord SSE | `internal/discord/sse_handlers.go` | Real-time Discord channel notifications |

### Key Design Decisions

- **Pure engine**: The `Engine` struct has zero DB or service dependencies. It takes a config and party, runs the turn loop in memory, and returns a result. This makes it fully unit-testable.
- **Timer-based worker**: Follows the same pattern as the gamble worker. Subscribes to `ExpeditionStarted` events, schedules a `time.AfterFunc` for the join deadline, and calls `ExecuteExpedition` when the timer fires.
- **CAS state transition**: `ExecuteExpedition` uses `UpdateExpeditionStateIfMatches` (compare-and-swap) to transition from `Recruiting` to `InProgress`, preventing duplicate execution.
- **Global cooldown**: The cooldown is keyed on `"global"` + `"expedition"`, not per-user. Only one expedition can run at a time across the entire system.

---

## Skill System

Six expedition skills map 1:1 to six jobs:

| Skill | Job |
|-------|-----|
| Fortitude | Blacksmith |
| Perception | Explorer |
| Survival | Farmer |
| Cunning | Gambler |
| Persuasion | Merchant |
| Knowledge | Scholar |

### Probability-Based Skill Check

Every party member has all 6 jobs at varying levels. The skill check algorithm:

1. Find `maxJobLevel` — the highest individual job level across **all** members and **all** jobs (minimum 1)
2. For the required skill, look up the corresponding job (e.g., Fortitude -> blacksmith)
3. For each **conscious** member, compute `contribution = member.JobLevels[jobKey] / maxJobLevel`
4. If the member has a **temporary skill** matching the check, add `+0.3` (configurable via `temp_skill_bonus`)
5. Build cumulative probability segments: each member gets a segment proportional to their contribution
6. Roll `r` in `[0, 1.0)`:
   - If `r < total`: **Pass**. Walk segments to find which member acted.
   - If `r >= total`: **Fail**. The member with the highest contribution is selected for narrative.

**Example**: Skill = Fortitude (blacksmith), maxJobLevel = 15

```
MemberA: blacksmith=10 -> 10/15 = 0.667
MemberB: blacksmith=3  ->  3/15 = 0.200
MemberC: blacksmith=0  ->  0/15 = 0.000
Total = 0.867

Roll segments: [0, 0.667) -> A passes | [0.667, 0.867) -> B passes | [0.867, 1.0) -> FAIL
```

### Debuffs and Temporary Skills

- **Debuff**: A debuffed member who is selected as the acting member automatically fails the skill check. The debuff is then cleared.
- **Temporary skill**: Grants `+0.3` contribution bonus to a specific skill. Consumed after the check is attempted (regardless of pass/fail).

---

## Encounter Engine

### Turn Loop (`engine.go`)

Each turn follows this sequence:

1. Add base fatigue (`base_fatigue_per_turn`)
2. Roll an encounter type (weighted random with progressive inversion)
3. Roll an outcome category (positive/neutral/negative, weighted random with dynamic modifiers)
4. Pick a random skill from the encounter's skill list
5. Resolve skill check against conscious party members
6. Handle debuff (force fail if acting member is debuffed, clear debuff)
7. Consume temporary skill if applicable
8. Apply effects (fatigue, purse, rewards, KOs, revives, debuffs, temp skills, weight shifts)
9. Build 3-part narrative from config templates
10. Check end conditions (fatigue >= max, all KO'd, turn >= max)

Turn 0 is an intro narrative selected randomly from `intro_narratives` with no encounter or skill check.

### Progressive Weight Inversion

Encounter selection uses progressive inversion so rare encounters become more likely as the expedition progresses:

```
progress = turn / maxTurns
invertedWeight = (1/baseWeight) / sum(1/baseWeight for all encounters)
effectiveWeight = baseWeight * (1 - progress) + invertedWeight * progress
```

At turn 1, weights match the configured base weights. At turn 50, weights are fully inverted — previously rare encounters (boss fights, treasure) become common, while previously common encounters (explore, travel) become rare.

### Party Size Scaling

KO and revive counts from effects scale with the initial party size:

```
scaledCount = baseCount * ceil(partySize / party_scale_divisor)
```

With the default `party_scale_divisor` of 3: a solo player sees base counts, a party of 3 sees 1x, a party of 6 sees 2x, etc.

---

## encounters.json Reference

**Location**: `configs/expedition/encounters.json`

The configuration file defines all encounter definitions, engine settings, and narrative templates. It is loaded at server startup and validated before the engine can run.

### Top-Level Structure

```json
{
  "version": "1.0",
  "settings": { ... },
  "intro_narratives": [ ... ],
  "encounters": { ... }
}
```

### Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `base_fatigue_per_turn` | int | 5 | Fatigue added at the start of every turn |
| `max_fatigue` | int | 100 | Expedition ends when fatigue reaches this value |
| `max_turns` | int | 50 | Expedition wins if this many turns are completed |
| `starting_purse` | int | 200 | Initial money pool shared by the party |
| `skill_check_bonus_money` | int | 50 | Bonus money added to purse on non-negative outcomes |
| `xp_formula_divisor` | int | 4 | XP per member = `ceil(partySize / divisor) + 1` |
| `leader_bonus_reward` | string | `"lootbox_tier2"` | Extra item awarded to the expedition leader |
| `win_bonus_reward` | string | `"xp_rarecandy"` | Extra item for conscious members on win |
| `win_bonus_money` | int | 500 | Extra money for conscious members on win |
| `party_scale_divisor` | int | 3 | Divisor for party-size scaling of KO/revive counts |
| `temp_skill_bonus` | float | 0.3 | Contribution bonus from temporary skills |

### Intro Narratives

An array of scene-setting strings used for turn 0. One is selected randomly per expedition.

```json
"intro_narratives": [
  "The expedition lands amid still lakes and a temple of old.",
  "The party sets out into a vast canyon carved by ancient rivers.",
  "Dense fog parts to reveal crumbling walls and overgrown paths."
]
```

### Encounter Definitions

Each encounter is keyed by its identifier (e.g., `"explore"`, `"combat_boss"`) and defines:

```json
{
  "display_name": "Exploration",
  "base_weight": 0.18,
  "skills": ["perception", "survival"],
  "min_party": 1,
  "outcomes": {
    "positive": { ... },
    "neutral": { ... },
    "negative": { ... }
  }
}
```

| Field | Description |
|-------|-------------|
| `display_name` | Human-readable name |
| `base_weight` | Base selection probability (all weights should sum to ~1.0 across encounters) |
| `skills` | Skills that can be checked during this encounter (one is picked randomly per turn) |
| `min_party` | Minimum conscious party members required; encounter is skipped if below threshold |
| `outcomes` | Maps of `"positive"`, `"neutral"`, `"negative"` to outcome definitions |

### 12 Encounter Types

| Key | Display Name | Skills | Min Party | Base Weight |
|-----|-------------|--------|-----------|-------------|
| `explore` | Exploration | Perception, Survival | 1 | 0.18 |
| `travel` | Travel | Survival, Fortitude | 1 | 0.15 |
| `combat_skirmish` | Skirmish | Fortitude, Cunning | 1 | 0.12 |
| `camp` | Camp | Knowledge, Survival | 1 | 0.10 |
| `hazard` | Hazard | Survival, Fortitude | 1 | 0.08 |
| `encounter` | Social Encounter | Persuasion, Cunning | 1 | 0.07 |
| `discovery` | Discovery | Perception, Knowledge | 1 | 0.07 |
| `combat_elite` | Elite Battle | Fortitude, Knowledge | 2 | 0.06 |
| `treasure` | Treasure | Perception, Cunning | 1 | 0.05 |
| `mystic` | Mystic Event | Knowledge, Persuasion | 1 | 0.05 |
| `drama` | Party Drama | Persuasion, Cunning | 1 | 0.04 |
| `combat_boss` | Boss Fight | Fortitude, Cunning, Persuasion | 3 | 0.03 |

### Outcome Definitions

Each outcome category has a `weight` (probability within the encounter) and two branches:

```json
{
  "weight": 0.30,
  "skill_pass": {
    "effects": { ... },
    "narratives": [ ... ]
  },
  "skill_fail": {
    "effects": { ... },
    "narratives": [ ... ]
  }
}
```

Outcome weights within an encounter must sum to ~1.0 (validated on load).

### Effects

Effects are applied based on whether the skill check passed or failed:

| Field | Type | Description |
|-------|------|-------------|
| `fatigue_delta` | int | Change to fatigue (can be negative for recovery) |
| `purse_delta` | int | Change to money pool (can be negative) |
| `reward` | string | Item key added to the reward pool (empty = none) |
| `ko_scale` | int | Base KO count (scaled by party size) |
| `revive_scale` | int | Base revive count (scaled by party size) |
| `debuff_primary` | bool | If true, the acting member is debuffed |
| `temp_skill` | string | Grants a temporary skill to the acting member |
| `shift_weights` | float | Shifts outcome weights toward positive for future turns |

### Narratives

Each narrative is a 3-part template:

```json
{
  "surprise": "A glint of metal behind fallen stone",
  "action": "{{primary}} pries open the cache",
  "outcome": "Supplies recovered in good condition"
}
```

Parts are joined with `. ` separators. Available placeholders:

| Placeholder | Replaced With |
|-------------|---------------|
| `{{primary}}` | The acting member's username |
| `{{secondary}}` | A random other conscious member's username (or "a companion" if none) |

Multiple narratives can be defined per outcome; one is selected randomly each turn.

### Config Validation

The config loader (`LoadEncounterConfig`) validates:

- At least one intro narrative exists
- At least one encounter exists
- `max_turns`, `max_fatigue`, and `party_scale_divisor` are positive
- Every encounter has at least one skill defined
- Every encounter has outcomes defined
- Outcome weights within each encounter sum to ~1.0 (tolerance: 0.01)
- Every outcome has both `skill_pass` and `skill_fail` with at least one narrative each

---

## API Endpoints

All expedition endpoints are under `/api/v1/expedition`. They require API key authentication via the `X-API-Key` header.

### Start Expedition

```
POST /api/v1/expedition/start
```

**Request body:**
```json
{
  "platform": "twitch",
  "platform_id": "123456",
  "username": "leader_name",
  "expedition_type": "standard"
}
```

**Response (201):**
```json
{
  "message": "Expedition started! Others can join.",
  "expedition_id": "a1b2c3d4-...",
  "join_deadline": "2026-02-05 14:03:00"
}
```

**Error cases:**
- 400: Missing/invalid fields
- 409: Expedition already active, or on cooldown
- 403: Expedition feature not unlocked (progression system)

### Join Expedition

```
POST /api/v1/expedition/join?id=<expedition_id>
```

**Request body:**
```json
{
  "platform": "twitch",
  "platform_id": "789012",
  "username": "member_name"
}
```

**Response (200):**
```json
{
  "message": "Joined expedition!"
}
```

### Get Expedition Details

```
GET /api/v1/expedition/get?id=<expedition_id>
```

**Response (200):**
```json
{
  "expedition": {
    "id": "a1b2c3d4-...",
    "initiator_id": "...",
    "expedition_type": "standard",
    "state": "Completed",
    "created_at": "...",
    "join_deadline": "..."
  },
  "participants": [
    {
      "user_id": "...",
      "username": "leader_name",
      "is_leader": true,
      "final_money": 650,
      "final_xp": 2,
      "final_items": ["lootbox_tier2", "lootbox_tier1"]
    }
  ]
}
```

### Get Active Expedition

```
GET /api/v1/expedition/active
```

Returns the currently active expedition (Recruiting or InProgress), or `null` if none.

### Get Expedition Status

```
GET /api/v1/expedition/status
```

**Response (200):**
```json
{
  "has_active": false,
  "active_details": null,
  "cooldown_expires": "2026-02-05T14:30:00Z",
  "on_cooldown": true
}
```

### Get Expedition Journal

```
GET /api/v1/expedition/journal?id=<expedition_id>
```

**Response (200):**
```json
[
  {
    "turn_number": 0,
    "encounter_type": "",
    "outcome": "",
    "narrative": "The expedition lands amid still lakes and a temple of old.",
    "fatigue": 0,
    "purse": 200
  },
  {
    "turn_number": 1,
    "encounter_type": "explore",
    "outcome": "neutral",
    "skill_checked": "perception",
    "skill_passed": true,
    "primary_member": "leader_name",
    "narrative": "The path splits in three directions. leader_name examines each route. The correct path is found without delay.",
    "fatigue": 5,
    "purse": 250
  }
]
```

---

## SSE Events

The expedition system publishes three event types through the SSE hub:

### `expedition.started`

Published when a new expedition begins recruiting.

```json
{
  "expedition_id": "a1b2c3d4-...",
  "initiator": "leader_name",
  "join_deadline": "2026-02-05T14:03:00Z"
}
```

### `expedition.turn`

Published for every turn during execution. The Discord bot shows every 5th turn plus the intro to avoid channel spam.

```json
{
  "expedition_id": "a1b2c3d4-...",
  "turn_number": 5,
  "narrative": "A rustling in the undergrowth. leader_name draws their blade. The threat retreats into shadow.",
  "fatigue": 30,
  "purse": 350
}
```

### `expedition.completed`

Published when the expedition finishes.

```json
{
  "expedition_id": "a1b2c3d4-...",
  "total_turns": 50,
  "won": true,
  "all_ko": false,
  "rewards": [
    {
      "user_id": "...",
      "username": "leader_name",
      "money": 650,
      "items": ["lootbox_tier2", "lootbox_tier1", "xp_rarecandy"],
      "xp": 2,
      "is_leader": true
    }
  ]
}
```

---

## Discord Commands

### `/explore`

Multi-purpose command that checks the current expedition state and acts accordingly:

| State | Behavior |
|-------|----------|
| No active expedition, no cooldown | Starts a new expedition (user becomes leader) |
| Active expedition in Recruiting | Joins the existing expedition |
| Active expedition in InProgress | Shows current status |
| Cooldown active | Shows time remaining |

### `/expedition-journal`

Displays the journal for a completed expedition. Takes an `expedition_id` option. Shows up to 10 journal entries per embed page.

---

## Reward Distribution

After the engine completes, rewards are calculated and distributed:

1. **Purse division**: The accumulated purse is divided among all participants with ~20% random variance
2. **Item pool**: Items collected during the expedition are randomly assigned to participants
3. **Leader bonus**: The expedition initiator receives an extra `lootbox_tier2`
4. **Win bonus**: If the expedition reached turn 50, all conscious members receive `xp_rarecandy` and 500 extra money
5. **Job XP**: Each participant receives `ceil(partySize / 4) + 1` XP to every job
6. **Engagement**: Each participant earns 3 progression engagement points

Rewards are persisted to the database and added to user inventories via the user service. Job XP is awarded via the job service, which may trigger level-up events.

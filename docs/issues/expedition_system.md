# Expedition System - Reference Documentation

> Source: `~/projects/Streamerbot/CustomCommands/Expedition.cs`
> Purpose: Documentation for future migration to BrandishBot_Go

---

## Overview

The Expedition system is a **multi-player cooperative adventure game** where chat users join parties, explore procedurally-generated events, and earn rewards. It integrates with the class system for skill checks and provides narrative journal output.

---

## Core Mechanics

### Expedition States
```
Inactive → Prepare → Active → Cleanup → Inactive
```

1. **Inactive**: No expedition running, waiting for queue trigger
2. **Prepare**: Join phase - users can `!explore` to join the party
3. **Active**: Expedition loop running, processing events
4. **Cleanup**: Distributing rewards and XP

### Party System
- **Leader**: First person in queue, receives guaranteed `lootbox2`
- **Members**: Join during prepare phase via `!explore` command
- **Party states**: Active (conscious) vs Out (knocked out)
- **Player properties**:
  - `IsConscious` - Can participate in events
  - `IsDebuffed` - Next skill check auto-fails, then clears
  - `TemporarySkills` - Earned during expedition, consumed on use
  - `prizeMoney` - Accumulated shards
  - `prizeItems` - Accumulated item rewards

### Fatigue System
- Starts at 0, expedition ends when fatigue >= 100
- Base +5 fatigue per turn
- Events can increase/decrease fatigue
- Turn limit: 50 turns max

### Purse System
- Shared party currency pool (starts at 2000)
- Events can add/subtract from purse
- Distributed with variance at end to all participants

---

## Event System

### Event Types (12 total)

| Event | Base Weight | Description |
|-------|-------------|-------------|
| Explore | 0.18 | Exploration, can find loot |
| Travel | 0.12 | Movement, resource management |
| Combat_Skirmish | 0.22 | Light combat, 1 potential casualty |
| Combat_Elite | 0.07 | Medium combat, guarantees 1 casualty |
| Combat_BossLike | 0.00 | Heavy combat, requires 3+ party members |
| Camp | 0.10 | Rest, can restore knocked out members |
| Hazard | 0.03 | Environmental danger |
| Discovery | 0.08 | Find something valuable |
| Encounter | 0.10 | Meet NPC (merchant, etc.) |
| Treasure | 0.03 | Loot opportunity |
| Mystic | 0.005 | Magical event |
| Drama | 0.025 | Party conflict/resolution |

### Weight Progression
Weights dynamically adjust based on turn progress:
- Early game: Higher weights for common events (Explore, Combat_Skirmish)
- Late game: Weights invert, making rare events more common

### Outcome System
Each event has three possible outcomes:
- **Positive (pos)**: Good result, rewards
- **Delayed (del)**: Neutral/mixed result
- **Negative (neg)**: Bad result, casualties/losses

Outcome weights per event type (example):
```
Explore:     pos=0.30, del=0.55, neg=0.15
Hazard:      pos=0.20, del=0.25, neg=0.55
Treasure:    pos=0.60, del=0.20, neg=0.20
Combat_Boss: pos=0.35, del=0.25, neg=0.40
```

### Outcome Weight Modifiers
- Persisted globally, adjusted based on results
- Positive outcomes shift weights toward more positive
- Negative outcomes shift weights toward more negative
- Creates dynamic difficulty balancing

---

## Skill System

### Skills (9 total)
```
Perception, Investigation, Survival, Athletics, Stealth,
Intimidation, Medicine, Insight, Persuasion
```

### Event-to-Skill Mapping
Each event type has 3 associated skills:
```go
Explore:         [Perception, Survival, Investigation]
Travel:          [Survival, Athletics, Perception]
Combat_Skirmish: [Athletics, Stealth, Intimidation]
Combat_Elite:    [Athletics, Medicine, Insight]
Combat_BossLike: [Intimidation, Insight, Perception]
Camp:            [Medicine, Survival, Insight]
Hazard:          [Survival, Athletics, Perception]
Discovery:       [Investigation, Insight, Stealth]
Encounter:       [Persuasion, Insight, Intimidation]
Treasure:        [Investigation, Stealth, Persuasion]
Mystic:          [Medicine, Persuasion, Insight]
Drama:           [Stealth, Persuasion, Athletics]
```

### Class-to-Skill Mapping
Each class provides 3 skills:
```go
Denizen:    [Perception, Survival, Insight]
Medic:      [Medicine, Insight, Survival]
Looter:     [Investigation, Stealth, Perception]
Criminal:   [Stealth, Intimidation, Investigation]
Lawman:     [Insight, Intimidation, Perception]
Blacksmith: [Athletics, Medicine, Investigation]
Broker:     [Persuasion, Insight, Investigation]
Farmer:     [Survival, Perception, Medicine]
Antagonist: [Intimidation, Stealth, Athletics]
```

### Skill Check Flow
1. Event selected, random skill chosen from event's skill pool
2. Find party member with matching skill (via class or temporary skills)
3. If found: `passSkillCheck = true`, member gets +50 bonus money
4. If debuffed: Clear debuff, but fail the check anyway
5. Temporary skills consumed on use

---

## Event Handlers (Detailed)

### Explore
- **pos + skill**: Shift outcome weights positive, reward `lootbox1`
- **pos**: Reward `lootbox1`
- **del + skill**: -10 fatigue
- **neg - skill**: Knock out secondary member

### Travel
- **pos + skill**: Primary gains temporary skill
- **pos**: -10 fatigue
- **del + skill**: -10 fatigue
- **neg - skill**: Knock out secondary member

### Camp
- **pos + skill**: Restore random knocked out member
- **pos**: -10 fatigue
- **del + skill**: Primary gains temporary skill
- **neg - skill**: Primary gets debuffed

### Hazard
- **pos + skill**: Reward `lootbox2`
- **pos - skill**: +5 fatigue
- **del + skill**: Primary gains temporary skill
- **del - skill**: Knock out secondary member
- **neg - skill**: Knock out primary AND secondary

### Discovery
- **pos + skill**: Restore knocked out member
- **pos**: Reward `lootbox1`
- **del + skill**: Primary gains temporary skill
- **neg**: Knock out secondary, debuff primary

### Encounter (Merchant NPC)
- **pos + skill**: Extra `lootbox2` bonus
- **pos**: Spend 200 purse, get `lootbox1`
- **del + skill**: Primary gains temporary skill
- **neg - skill**: Knock out secondary
- **neg**: Lose 1 random reward

### Treasure
- **pos + skill**: Bonus `lootbox2`
- **pos**: +300 purse (shard reward)
- **del + skill**: +300 purse
- **neg - skill**: Knock out secondary
- **neg**: -500 purse

### Mystic
- **pos + skill**: Restore knocked out member
- **pos**: -10 fatigue
- **del + skill**: -10 fatigue, -500 purse
- **neg - skill**: Knock out secondary, get `lootbox1`
- **neg**: -500 purse

### Drama
- **pos + skill**: (TODO: permanent buff)
- **pos**: -10 fatigue, leader takes center stage
- **del - skill**: +5 fatigue
- **neg - skill**: Knock out primary AND secondary
- **neg**: +10 fatigue

### Combat_Skirmish
- Reward: `lootbox1`
- **-skill**: Knock out secondary

### Combat_Elite
- Requires 2+ party members (else party wipe)
- Reward: `lootbox2`
- Always knocks out primary
- **-skill**: Also knock out secondary

### Combat_BossLike
- Requires 3+ party members (else party wipe)
- Reward: `lootbox3`
- Always knocks out primary AND secondary
- **-skill**: Also knock out skilled member

---

## Rewards

### Reward Distribution
1. **Purse**: Divided with variance among all party members
2. **Items**: Randomly distributed to party members
3. **Leader Bonus**: Guaranteed `lootbox2`
4. **Completion Bonus**: If reach turn 50, active members get `rarecandy` + 500 money

### Reward Tokens
- `shard` → +300 to purse (not actual item)
- `lootbox1`, `lootbox2`, `lootbox3` → Added to reward pool
- `rarecandy` → Completion bonus

### XP Distribution
- All party members receive class XP
- Amount: `partySize / 4 + 1` XP per member

---

## Journal System

- Events rendered to text journal
- Saved to `Output/ExpeditionJournal.txt`
- Includes loot distribution at end
- Displayed via `ShowExpeditionJournal` action
- Playback timing: 1 second per turn

---

## Commands

| Command | Description |
|---------|-------------|
| `!explore` | Join active expedition (during Prepare phase) |
| `!party` | View current party members |
| Toggle (admin) | Enable/disable expedition system |

---

## Global State

### Persisted Variables
- `AllowExpeditions` - Boolean toggle
- `ExpeditionReadyToStart` - Flag for timer system
- `ExpeditionCount` - Queue of expedition leaders
- `ExpeditionOutcomeModifiers` - JSON dictionary of weight adjustments

### Runtime State
- `expState` - Current expedition phase
- `ExpeditionLeader` - Party leader Player object
- `expeditionParty` - All party members
- `expeditionFinalParty` - Members who completed expedition
- `expeditionPurse` - Shared currency pool
- `expeditionRewards` - Item reward pool
- `outcomeWeightModifiers` - Dynamic outcome weights

---

## Integration Points

### Class System
- Uses `ClassType` enum for skill mapping
- Awards class XP at expedition end
- Class determines available skills

### Inventory System
- Calls `Inventory.AddItemSet` to distribute rewards
- Items: shards, lootbox1, lootbox2, lootbox3, rarecandy

### Stats System
- Tracks `ExpeditionsJoined` stat per user

### Timer System
- `ExpeditionTimerExpire` - Checks if expedition can start
- `ExpeditionJoinTimer` - Join phase duration

---

## Migration Considerations for BrandishBot_Go

### Required Systems
1. **Class System** - Need class-to-skill mapping (new feature)
2. **Skill System** - 9 skills with event associations
3. **Party System** - Multi-user session management
4. **Journal System** - Event narrative generation
5. **Timer/Queue System** - Expedition scheduling

### Progression Integration
Each item reward should be a progression node:
- `item_lootbox1`, `item_lootbox2`, `item_lootbox3`
- `item_rarecandy`

Feature node:
- `feature_expedition` (currently exists in tree at Tier 4)

### Database Schema Additions
- `expeditions` - Expedition sessions
- `expedition_participants` - Party members per expedition
- `expedition_events` - Event log per expedition
- `expedition_rewards` - Reward distribution records

### API Endpoints
- `POST /api/v1/expedition/join` - Join expedition
- `GET /api/v1/expedition/party` - View party
- `GET /api/v1/expedition/status` - Current expedition status
- `GET /api/v1/expedition/journal` - Get journal output
- `POST /api/v1/expedition/admin/toggle` - Enable/disable

### Discord Commands
- `/explore` - Join expedition
- `/party` - View party
- `/expedition-status` - View current status

## Status Update (2026-01-30)

**Implementation Status: In Progress**

- **Database**: Tables `expeditions` and `expedition_participants` exist (via migration `0012_add_expeditions.sql`).
- **Service**: `internal/expedition/service.go` exists, but `ExecuteExpedition` is a placeholder (`fmt.Errorf("not implemented")`).
- **Discord Integration**: Expedition commands (`/explore`, etc.) are not registered in `cmd/discord/main.go`.

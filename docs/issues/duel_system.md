# Duel System Feature Specification

## Overview

The Duel System allows users to challenge each other to 1v1 duels with wagered currency. The system integrates with Twitch predictions and applies timeouts to losers for entertainment value.

## Commands

### `!duel <target> <wager>`
Initiates a duel challenge against another user.

**Parameters:**
- `target` - Username of the person being challenged
- `wager` - Amount of shards to bet

**Requirements:**
- Challenger must own at least 1 `stick` item
- Challenger must have at least `wager` shards
- No other duel can be active at the time

**Flow:**
1. Validate parameters and item requirements
2. Deduct 1 stick and wager shards from challenger
3. Create Twitch prediction (if predictions not already active)
4. Store duel state with 2-minute acceptance window
5. Wait for target to accept/decline

### `!accept`
Accepts a pending duel challenge (target user only).

**Requirements:**
- Must be the target of the active duel
- Must own at least 1 `stick` item
- Must have at least the wagered amount of shards

**Flow:**
1. Validate caller is the duel target
2. Deduct 1 stick and wager shards from target
3. Mark duel as accepted
4. Trigger duel start sequence (KirbyDuelStart action)

### `!decline`
Declines a pending duel challenge (target or challenger can use).

**Flow:**
1. Validate caller is involved in the duel
2. Refund challenger's stick and shards
3. Mark duel as declined

## Duel Resolution

When the duel starts (after acceptance or timeout):

### If Accepted:
1. Play visual sequence (KirbyDuelEnd action, 4 second wait)
2. Roll random 0-99 (50/50 odds)
3. Winner receives `wager * 2` shards
4. Loser receives 60-second timeout with reason "You've been CLUDGED."
5. Resolve Twitch prediction with winning outcome
6. Clean up duel state

### If Not Accepted (Timeout):
1. Refund challenger's stick and shards
2. Cancel Twitch prediction
3. Clean up duel state

## Special Cases

### Self-Duel
If a user duels themselves:
- If they have a stick: Remove it, message "A stick falls between your legs and you fall to the ground. You lose the duel."
- If no stick: Message "You trip, fall in the mud, and die of cringe. Honestly, the duel was lost long before that."
- 60-second timeout with reason "Recovering from the shame."

### Joey (NPC) Duel
Special handler for dueling "joey":
- Plays random taunting quote from preset list
- Triggers DuelJoey action
- Times out challenger for `wager % 300` seconds
- User always loses to Joey

**Joey Quotes:**
- "Hope ya brought a helmet - this duel's gonna wreck ya!"
- "I may not be a genius, but I'm a genius at winning!"
- "You're about to get wheeled over!"
- "Step aside, amateurs - this duel's got a champ now!"
- "I don't need luck. I got heart... and way better cards."
- "Don't take it personal when I win. Actually - yeah, take it real personal."
- "When I win - and I will - you can blame it on your bad life choices."
- "Hope you like second place - 'cause that's your new address."
- "You brought a stick to this? Cute. I brought destiny."
- "Losing to me builds character. You're welcome in advance."

## Data Model

### Duel State (stored in global var)
```json
{
  "duelId": "guid",
  "duelChallengerId": "twitch_user_id",
  "duelTargetId": "twitch_user_id",
  "duelWager": "100",
  "duelChallengerName": "username",
  "duelTargetName": "username",
  "duelStatus": "pending|accepted|declined",
  "duelStartTime": "UTC datetime",
  "duelEndTime": "UTC datetime (start + 2 min)"
}
```

### Global Variables
- `isActiveDuel` (bool) - Whether a duel is currently pending/active
- `duel` (Dictionary) - The duel state object
- `isDuelPredictionActive` (bool) - Whether a Twitch prediction is running

## Item Requirements

| Item | Purpose |
|------|---------|
| `stick` | Required to participate (1 per participant, consumed) |
| `shard` | Currency for wagering |

## Integration Points

### Twitch Features
- **Predictions**: Created when duel starts with options [challenger, target], 120 second duration
- **Timeouts**: Applied to loser (60 sec) and self-duelers (60 sec)
- **Reply**: Joey quotes reply to the original message

### External Actions (Streamer.bot)
- `DuelJoey` - Triggered when challenging Joey
- `KirbyDuelStart` - Triggered when duel is accepted
- `KirbyDuelEnd` - Triggered when duel resolves

### Inventory System
- `GetInventoryItemAmount` - Check stick/shard counts
- `AddItem` - Award winnings, refunds
- `RemoveItem` - Deduct stakes

### Timeout System
- `TimeoutUser` - Apply timeout to loser

## Implementation Notes

### Odds
- Standard duel: 50/50 random roll
- Joey duel: Always lose

### Economy
- Entry cost: 1 stick + wager shards (per participant)
- Prize pool: wager * 2 shards (winner takes all)
- Net effect: 2 sticks removed from economy per duel

### Timing
- Acceptance window: 2 minutes
- Prediction duration: 120 seconds
- Post-accept visual delay: 4 seconds
- Loser timeout: 60 seconds
- Self-duel timeout: 60 seconds
- Joey duel timeout: wager % 300 seconds

## Migration Considerations for Go Implementation

### API Endpoints Needed
- `POST /api/v1/duel/initiate` - Start a duel challenge
- `POST /api/v1/duel/accept` - Accept pending duel
- `POST /api/v1/duel/decline` - Decline pending duel
- `GET /api/v1/duel/active` - Get current active duel (if any)
- `POST /api/v1/duel/resolve` - Execute duel resolution (internal/worker)

### Database Tables
```sql
CREATE TABLE duels (
    id UUID PRIMARY KEY,
    challenger_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    wager INTEGER NOT NULL,
    status TEXT NOT NULL, -- pending, accepted, declined, completed, expired
    winner_id TEXT,
    prediction_id TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ
);
```

### Service Dependencies
- UserService (inventory operations)
- TwitchService (predictions, timeouts)
- EventBus (duel events for stats/progression)

### Worker Considerations
- Need background worker to handle duel expiration
- Similar pattern to existing gamble_worker.go
- Check for expired pending duels and refund

### Discord Integration
- `/duel <target> <wager>` command
- `/accept` command
- `/decline` command
- Embed responses with duel status

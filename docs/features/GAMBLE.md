# Gamble System

The Gamble system allows players to wager lootboxes in a high-stakes, winner-takes-all game. Participants contribute lootboxes to a pool, and the player whose opened items have the highest total value wins everything.

## Core Mechanics

### 1. Initiation (`/gamble start`)
- A player starts a gamble by wagering one or more lootboxes.
- This creates a new active gamble session with a **Join Deadline** (configurable, typically 2 minutes).
- Only one gamble can be active at a time.

### 2. Joining (`/gamble join`)
- Other players can join the active gamble before the deadline.
- **Requirement**: Joiners must match the initiator's wager exactly (same lootbox types and quantities).
- The system automatically validates inventory and locks the wagered items.

### 3. Execution
- Once the deadline passes, the system executes the gamble.
- **Opening**: All wagered lootboxes are opened for all participants.
- **Value Calculation**: The total value of items found is calculated for each participant.
  - Value = Item Base Value * Quantity
  - Progression modifiers (e.g., `gamble_win_bonus`) may apply.

### 4. Winner Determination
- The participant with the **highest total value** of found items is declared the winner.
- **Tie-Breaker**: If multiple players tie for the highest value, a random winner is selected among them.
- **Winner Takes All**: The winner receives **ALL** items found by all participants.
- Losers receive nothing (except XP).

## Statistics & Tracking

The system tracks several special events for stats and achievements:

- **Critical Failure**: When a player's total value is significantly lower than the average (defined by `CriticalFailThreshold`).
- **Near Miss**: When a loser's score is very close to the winner's score (defined by `NearMissThreshold`).
- **Tie Break Lost**: When a player ties for the win but loses the random tie-breaker.

## Job Integration: Gambler

Participating in gambles awards XP to the **Gambler** job.

- **Participation XP**: Awarded for every lootbox wagered (`GamblerXPPerLootbox`).
- **Win Bonus**: Substantial XP bonus for winning (`GamblerWinBonus`).
- **Leveled Up**: Leveling up the Gambler job can unlock perks and bonuses.

## API Endpoints

### Start Gamble
```http
POST /api/v1/gamble/start
```
**Body**:
```json
{
  "platform": "twitch",
  "platform_id": "12345",
  "username": "initiator",
  "bets": [
    { "item_name": "lootbox_tier1", "quantity": 1 }
  ]
}
```

### Join Gamble
```http
POST /api/v1/gamble/join
```
**Body**:
```json
{
  "gamble_id": "uuid-string",
  "platform": "twitch",
  "platform_id": "67890",
  "username": "joiner"
}
```

### Get Active Gamble
```http
GET /api/v1/gamble/active
```
Returns the currently active gamble session (if any).

### Get Gamble Details
```http
GET /api/v1/gamble/get?id=uuid-string
```
Returns details of a specific gamble, including participants, state, and results.

## Implementation Details

- **Service**: `internal/gamble/service.go`
- **Worker**: `internal/worker/gamble_worker.go` (Handles automatic execution after deadline)
- **Database**: `gambles`, `gamble_participants`, `gamble_opened_items` tables.

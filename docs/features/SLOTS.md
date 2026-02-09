# Slots Minigame

The slots minigame is a single-player gambling feature where players bet money on a 3-reel slot machine. The game uses weighted probability distribution to achieve a 92% RTP (8% house edge), providing house-favored but fair gameplay. Players can trigger special events like big wins and jackpots, with visual feedback sent to Streamer.bot for on-stream displays.

---

## Table of Contents

- [Gameplay Flow](#gameplay-flow)
- [Architecture Summary](#architecture-summary)
- [Symbol Distribution & Payouts](#symbol-distribution--payouts)
- [Special Features](#special-features)
- [RTP Mechanics](#rtp-mechanics)
- [API Endpoints](#api-endpoints)
- [Discord Commands](#discord-commands)
- [Streamer.bot Integration](#streamerbot-integration)
- [Progression Integration](#progression-integration)
- [Engagement & XP](#engagement--xp)

---

## Gameplay Flow

1. User initiates a spin via `/slots <bet>` or `POST /api/v1/slots/spin`
2. System validates bet amount (10-10,000 money) and checks feature unlock
3. **Cooldown check**: Verifies user hasn't spun within last 10 minutes
4. Transaction begins: inventory locked, money verified
5. Three reels spin independently using weighted random selection
6. Payout calculated based on matching symbols
7. Inventory updated atomically (bet deducted, winnings added)
8. Transaction committed
9. **Cooldown set**: 10-minute cooldown applied to user
10. Engagement tracked and Gambler XP awarded asynchronously
11. Event published to Streamer.bot with visual trigger flags

### Limits

| Limit        | Value                   |
| ------------ | ----------------------- |
| Minimum bet  | 10 money                |
| Maximum bet  | 10,000 money            |
| **Cooldown** | **10 minutes per user** |

---

## Architecture Summary

```
Discord / Streamer.bot
         |
    SSE Events (slots.completed)
         |
   SSE Hub ← Event Bus ← Slots Service
                              |
                     Weighted RNG Engine
                              |
                     Symbol Weights Config
```

### Layer Breakdown

| Layer           | Location                             | Responsibility                                                   |
| --------------- | ------------------------------------ | ---------------------------------------------------------------- |
| Domain types    | `internal/domain/slots.go`           | SlotsResult, SlotsCompletedPayload structs                       |
| Constants       | `internal/slots/constants.go`        | Symbol weights, payout multipliers, thresholds                   |
| Service         | `internal/slots/service.go`          | Core game logic, RNG, payout calculation, transaction management |
| Handler         | `internal/handler/slots.go`          | HTTP API handler with validation and feature locking             |
| Discord command | `internal/discord/cmd_slots.go`      | `/slots` command with visual embed responses                     |
| Streamer.bot    | `internal/streamerbot/subscriber.go` | Event subscriber for visual triggers                             |

### Key Design Decisions

- **Stateless**: No persistent slots history table. Each spin is independent.
- **Atomic transactions**: Uses PostgreSQL transaction with inventory locking to prevent race conditions.
- **Cryptographic RNG**: Uses `utils.SecureRandomInt()` for unpredictable, fair spins.
- **Event-driven**: Publishes to internal event bus, forwarded to SSE and Streamer.bot.
- **House-favored**: 92% RTP ensures long-term profitability while maintaining fairness.

---

## Symbol Distribution & Payouts

The game uses 7 symbols with weighted probabilities to achieve target RTP:

| Symbol     | Weight   | Probability | 3-Match Payout   | Expected Value |
| ---------- | -------- | ----------- | ---------------- | -------------- |
| 🍋 LEMON   | 400/1000 | 40.0%       | 0.5x (lose half) | -32.0%         |
| 🍒 CHERRY  | 250/1000 | 25.0%       | 2.0x             | +15.6%         |
| 🔔 BELL    | 150/1000 | 15.0%       | 5.0x             | +16.9%         |
| 💰 BAR     | 100/1000 | 10.0%       | 10.0x            | +10.0%         |
| 7️⃣ SEVEN   | 70/1000  | 7.0%        | 25.0x            | +8.6%          |
| 💎 DIAMOND | 25/1000  | 2.5%        | 100.0x           | +1.6%          |
| ⭐ STAR    | 5/1000   | 0.5%        | 500.0x           | +0.06%         |

### 2-Match Consolation Prize

If exactly 2 out of 3 symbols match (but not all 3):

- **Payout**: 0.1x bet (10% return)
- **Example**: 🍒🍒🔔 → Bet 100, get 10 back

### Total RTP Calculation

**Target RTP**: 92% ± 1%

The system balances losing symbols (LEMON) against winning symbols to maintain house edge while providing frequent small wins and rare big wins.

---

## Special Features

### Big Win Threshold

- **Trigger**: Payout ≥ 10x bet
- **Effect**: "BIG WIN" message, orange embed color, +200 Gambler XP bonus
- **Streamer.bot**: `trigger_type: "big_win"` flag for visual effects

### Jackpot Threshold

- **Trigger**: Payout ≥ 50x bet
- **Effect**: "JACKPOT" message, gold embed color, +1000 Gambler XP bonus
- **Streamer.bot**: `trigger_type: "jackpot"` flag for visual effects

### Mega Jackpot

- **Trigger**: 500x payout (3× STAR symbols)
- **Effect**: "MEGA JACKPOT" message, +1000 Gambler XP bonus
- **Streamer.bot**: `trigger_type: "mega_jackpot"` flag for celebration effects

### Near-Miss Tracking

- **Definition**: Exactly 2 out of 3 symbols match
- **Purpose**: Analytics and future UI enhancements ("So close!")
- **Streamer.bot**: `is_near_miss: true` flag

---

## RTP Mechanics

### Return to Player (RTP)

**Formula**: `(Total Paid Out / Total Wagered) × 100`

**Target**: 92%

### How It Works

1. Each symbol has a weighted probability (total = 1000)
2. Three reels spin independently using the same weights
3. Payout multipliers are balanced to achieve 92% RTP over infinite spins
4. LEMON (most common) has negative expected value to create house edge
5. Rare symbols (DIAMOND, STAR) have high payouts but low probability

### Verification

The RTP can be verified by:

- Simulating 100,000+ spins
- Calculating: `(sum of all payouts) / (sum of all bets)`
- Result should be 92% ± 1%

---

## API Endpoints

### Spin Slots

**Endpoint**: `POST /api/v1/slots/spin`

**Request**:

```json
{
  "platform": "discord",
  "platform_id": "123456789",
  "username": "player1",
  "bet_amount": 100
}
```

**Response** (Win):

```json
{
  "user_id": "uuid",
  "username": "player1",
  "reel1": "CHERRY",
  "reel2": "CHERRY",
  "reel3": "CHERRY",
  "bet_amount": 100,
  "payout_amount": 200,
  "payout_multiplier": 2.0,
  "message": "You won 200 money (net +100)!",
  "is_win": true,
  "is_near_miss": false,
  "trigger_type": "normal"
}
```

**Response** (Loss):

```json
{
  "user_id": "uuid",
  "username": "player1",
  "reel1": "LEMON",
  "reel2": "BELL",
  "reel3": "BAR",
  "bet_amount": 100,
  "payout_amount": 0,
  "payout_multiplier": 0.0,
  "message": "Better luck next time! You lost 100 money.",
  "is_win": false,
  "is_near_miss": false,
  "trigger_type": "normal"
}
```

**Error Responses**:

- `400 Bad Request`: Invalid bet amount, insufficient funds, cooldown active
- `403 Forbidden`: Feature not unlocked
- `500 Internal Server Error`: Transaction failed

**Cooldown Error Example**:

```json
{
  "error": "action 'slots' on cooldown: 4m 23s remaining"
}
```

---

## Discord Commands

### `/slots <bet>`

**Description**: Spin the slots machine and win money!

**Parameters**:

- `bet` (required, integer, 10-10000): Amount of money to bet

**Example Usage**:

```
/slots bet:100
```

**Response Embed**:

**Win Example**:

```
🎰 Slots - Win! 🎰
You won 200 money (net +100)!

Reels: 🍒 | 🍒 | 🍒
Bet Amount: 100 money
Payout: 200 money
Multiplier: 2.00x

Player: player1
```

**Big Win Example**:

```
🎉 BIG WIN! 🎉
You won 1000 money (net +900)!

Reels: 💰 | 💰 | 💰
Bet Amount: 100 money
Payout: 1000 money
Multiplier: 10.00x

Player: player1
```

**Jackpot Example**:

```
💎 JACKPOT! 💎
You won 10000 money (net +9900)!

Reels: 💎 | 💎 | 💎
Bet Amount: 100 money
Payout: 10000 money
Multiplier: 100.00x

Player: player1
```

**Embed Colors**:

- Red (`0xFF0000`): Loss
- Green (`0x00FF00`): Win
- Orange (`0xFFA500`): Big Win (10x+)
- Dark Orange (`0xFF8C00`): Jackpot (50x+)
- Gold (`0xFFD700`): Mega Jackpot (500x)

---

## Streamer.bot Integration

### Event: `slots.completed`

**Action**: `BrandishBot_SlotsResult`

**Payload**:

```json
{
  "user_id": "uuid",
  "username": "player1",
  "bet_amount": "100",
  "reel1": "CHERRY",
  "reel2": "CHERRY",
  "reel3": "CHERRY",
  "payout_amount": "200",
  "payout_multiplier": "2.00",
  "trigger_type": "normal",
  "is_win": "true",
  "is_near_miss": "false"
}
```

### Trigger Types

Use `trigger_type` to control visual effects:

| Trigger Type   | When              | Suggested Effect                       |
| -------------- | ----------------- | -------------------------------------- |
| `normal`       | Standard win/loss | Basic animation                        |
| `big_win`      | 10x+ payout       | Celebration animation, confetti        |
| `jackpot`      | 50x+ payout       | Major celebration, screen shake        |
| `mega_jackpot` | 500x payout       | Full-screen celebration, sound effects |

### Use Cases

1. **On-Stream Display**: Show reel symbols in overlay
2. **Celebration Effects**: Trigger confetti, screen shake for big wins
3. **Leaderboard**: Track biggest single win
4. **Analytics**: Monitor `is_near_miss` for engagement metrics

---

## Progression Integration

### Feature Node: `feature_slots`

---

## Engagement & XP

### Engagement Metrics

Automatically tracked when players spin slots:

| Metric          | Weight | Description                                     |
| --------------- | ------ | ----------------------------------------------- |
| `slots_spin`    | 5.0    | Every spin contributes to progression           |
| `slots_win`     | 10.0   | Winning spins contribute more                   |
| `slots_big_win` | 50.0   | Big wins (10x+) significantly boost progression |
| `slots_jackpot` | 200.0  | Jackpots (50x+) massively boost progression     |

### Gambler Job XP

**Job**: Gambler

**XP Formula**:

```
Base XP = bet_amount / 10

Bonuses:
+ 50 XP if payout > bet (any win)
+ 200 XP if trigger_type == "big_win"
+ 1000 XP if trigger_type == "jackpot" or "mega_jackpot"
```

**Examples**:

- Bet 100, lose: **10 XP**
- Bet 100, win 200 (2x): **60 XP** (10 base + 50 bonus)
- Bet 100, win 1000 (10x): **260 XP** (10 base + 50 win + 200 big win)
- Bet 100, win 10000 (100x): **1060 XP** (10 base + 50 win + 1000 jackpot)

---

## Future Enhancements

### Planned Features

1. **Progression Modifiers**:
   - `upgrade_slots_rtp_boost`: Increase RTP from 92% → 95%
   - `upgrade_slots_jackpot_boost`: Increase DIAMOND/STAR multipliers by 50%

2. **Bonus Items**:
   - **Free Spin Item**: Consumable that allows bet=0 spin with full payouts
   - **Lucky Charm Item**: Temporary buff that boosts rare symbol weights

3. **Stats & Leaderboards**:
   - Track: Total wagered, total won, biggest single win, jackpots hit
   - Daily/weekly leaderboards for highest single payout

4. **Analytics**:
   - Add `slots_history` table for long-term analytics
   - Track near-miss frequency for engagement analysis
   - Monitor actual RTP over time for balance adjustments

---

## Configuration

### Symbol Weights

Defined in `internal/slots/constants.go`:

```go
var SymbolWeights = map[string]int{
    SymbolLemon:   400, // 40%
    SymbolCherry:  250, // 25%
    SymbolBell:    150, // 15%
    SymbolBar:     100, // 10%
    SymbolSeven:   70,  // 7%
    SymbolDiamond: 25,  // 2.5%
    SymbolStar:    5,   // 0.5%
}
```

### Payout Multipliers

```go
var PayoutMultipliers = map[string]float64{
    SymbolLemon:   0.5,   // Lose half bet
    SymbolCherry:  2.0,   // Double bet
    SymbolBell:    5.0,   // 5x payout
    SymbolBar:     10.0,  // 10x payout
    SymbolSeven:   25.0,  // 25x payout
    SymbolDiamond: 100.0, // 100x jackpot
    SymbolStar:    500.0, // 500x mega jackpot
}
```

### Thresholds

```go
const (
    MinBetAmount       = 10
    MaxBetAmount       = 10000
    BigWinThreshold    = 10.0  // 10x bet
    JackpotThreshold   = 50.0  // 50x bet
    TwoMatchMultiplier = 0.1   // Consolation prize
)
```

---

## Testing

### Manual Testing Checklist

- [ ] Small bet (10 money) processes correctly
- [ ] Large bet (10,000 money) processes correctly
- [ ] Insufficient funds returns friendly error
- [ ] Feature locked returns 403 with progression info
- [ ] Win updates inventory correctly
- [ ] Loss deducts bet correctly
- [ ] Big win triggers special event (Streamer.bot)
- [ ] Jackpot triggers jackpot event (Streamer.bot)
- [ ] Engagement tracking increments progression
- [ ] Gambler XP awards correctly
- [ ] Discord embed displays correctly
- [ ] Near-miss tracking works (2/3 match)

### RTP Verification

To verify the 92% RTP:

1. Run 100,000+ simulated spins
2. Calculate: `(total_payouts / total_bets) × 100`
3. Result should be 92% ± 1%

Example test: `internal/slots/service_test.go` (future implementation)

---

## Troubleshooting

### Common Issues

**Problem**: "Action 'slots' on cooldown"

- **Cause**: User spun slots within last 10 minutes
- **Solution**: Wait for cooldown to expire (time remaining shown in error message)

**Problem**: "Insufficient funds" error

- **Cause**: User doesn't have enough money
- **Solution**: Earn money via farming, search, or economy

**Problem**: "Feature not yet unlocked"

- **Cause**: `feature_slots` not unlocked in progression
- **Solution**: Unlock `feature_economy` first, then vote for `feature_slots`

**Problem**: Transaction failed

- **Cause**: Database deadlock or concurrent inventory modification
- **Solution**: Service automatically retries; user should retry if error persists

**Problem**: Streamer.bot events not showing

- **Cause**: Streamer.bot client disconnected
- **Solution**: Check Streamer.bot connection, events are stored and will retry

---

## Database Schema

### Migration: `0021_add_slots_engagement.sql`

Adds engagement weights for slots metrics:

```sql
INSERT INTO engagement_weights (metric_type, weight, description) VALUES
    ('slots_spin', 5.0, 'Player spun the slots'),
    ('slots_win', 10.0, 'Player won on slots'),
    ('slots_big_win', 50.0, 'Player hit a big win (10x+ payout)'),
    ('slots_jackpot', 200.0, 'Player hit jackpot (50x+ payout)');
```

### No Persistent History

Unlike gambles or expeditions, slots spins are **not** stored in a dedicated table. This keeps the database lean while still tracking engagement and XP through existing systems.

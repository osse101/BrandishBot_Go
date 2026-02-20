# Trap & Item Interaction System

The Trap & Item Interaction System allows users to use various items from their inventory to interact with other chat participants, primarily through timeouts and protective measures.

## Core Mechanics

### Traps (`trap`)

A stealthy way to catch an unsuspecting user.

- **Command**: `/use trap <target>`
- **Effect**: Places a hidden trap on the specified user. The next time that user sends a message in chat, the trap triggers, timing them out for 60 seconds.
- **Consumption**: Consumes 1 Trap.
- **Constraint**: A user can only have one active trap on them at a time.
- **Self-Trap**: If you target someone who already has a trap, you might accidentally step on it yourself!

### Mines (`mine`)

A chaotic explosive that targets random active chatters.

- **Command**: `/use mine`
- **Effect**: Automatically selects a random active chatter (someone who has messaged recently). Places a hidden mine on them that acts like a trap.
- **Risk**: If no other users are active, or if you are just unlucky, you might drop the mine on your own foot!
- **Consumption**: Consumes 1 Mine.

### Blasters & Missiles

Direct offensive tools to time out users immediately.

- **Command**: `/use <item_name> <target>`
- **Effect**: Instantly times out the target user.
- **Duration**:
  - `blaster`: 60 seconds
  - `missile`: 60 seconds
  - `this`: 101 seconds
  - `deez`: 202 seconds
  - `big_blaster`: 600 seconds (10 minutes)
  - `huge_blaster`: 6000 seconds (100 minutes)
- **Quality Bonus**: Higher quality items increase the duration (see [Quality Modifiers](#quality-modifiers)).

### Area of Effect Weapons (`tnt`, `grenade`)

Explosives that hit multiple or random targets.

- **TNT**:
  - **Command**: `/use tnt`
  - **Effect**: Hits 5-9 random active chatters with a 60-second timeout.
- **Grenade**:
  - **Command**: `/use grenade`
  - **Effect**: Hits 1 random active chatter with a 60-second timeout.

### Defensive Items (`shield`, `mirror_shield`)

Protection against offensive items.

- **Shield**:
  - **Command**: `/use shield [quantity]`
  - **Effect**: Blocks the next incoming timeout or trap trigger. Each shield protects against one attack.
- **Mirror Shield**:
  - **Command**: `/use mirror_shield [quantity]`
  - **Effect**: Reflects the timeout back to the attacker.

### Support Items (`revive_small`, `revive_medium`, `revive_large`)

Help a timed-out user return to chat sooner.

- **Command**: `/use <item_name> <target>`
- **Effect**: Reduces the target's current timeout duration.
- **Base Reduction**:
  - `revive_small`: 60 seconds
  - `revive_medium`: 600 seconds
  - `revive_large`: 6000 seconds
- **Quality Bonus**: Higher quality items increase the reduction amount (see [Quality Modifiers](#quality-modifiers)).

### Utility Items

- **Rare Candy** (`rarecandy`):
  - **Command**: `/use rarecandy <job_name>`
  - **Effect**: Grants 500 XP to the specified job (e.g., `blacksmith`, `merchant`).
- **Shovel** (`shovel`):
  - **Command**: `/use shovel`
  - **Effect**: Digs up resources. Guarantees 2 Sticks per use.
- **Stick** (`stick`):
  - **Command**: `/use stick`
  - **Effect**: A humble stick. Mostly used for crafting or as a joke weapon.

## Quality Modifiers

The quality of the item used (Common, Uncommon, Rare, etc.) directly affects the power of the item.

- **Weapons/Traps**: Increases the timeout duration.
- **Revives**: Increases the timeout reduction amount.

| Quality Level | Duration Adjustment |
| :------------ | :------------------ |
| **Legendary** | +40 Seconds         |
| **Epic**      | +30 Seconds         |
| **Rare**      | +20 Seconds         |
| **Uncommon**  | +10 Seconds         |
| **Common**    | +0 Seconds          |
| **Poor**      | -10 Seconds         |
| **Junk**      | -20 Seconds         |
| **Cursed**    | -30 Seconds         |

_Example: A **Legendary Blaster** (60s base) will timeout a user for **100s** (60s + 40s)._
_Example: A **Rare Revive Small** (60s base) will reduce a timeout by **80s** (60s + 20s)._

## Implementation Details

The item logic is handled in `internal/user/item_handlers.go` and executed via the **User Service** (`internal/user/service.go`).

- **Timeout System**: The User Service manages timeouts directly in-memory (`internal/user/timeout.go`). Timeouts accumulate if multiple are applied.
- **Active Chatter Tracking**: The system tracks users who have recently messaged (`internal/user/active_chatter_tracker.go`) to determine valid targets for random-target items (Mines, TNT).

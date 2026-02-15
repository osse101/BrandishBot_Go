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

### Blasters (`blaster`, `big_blaster`, `huge_blaster`)
Direct offensive tools to time out users immediately.

- **Command**: `/use <item_name> <target>`
- **Effect**: Instantly times out the target user.
- **Duration**:
  - `blaster`: 60 seconds
  - `big_blaster`: 600 seconds (10 minutes)
  - `huge_blaster`: 6000 seconds (100 minutes)
- **Quality Bonus**: Higher quality items (Uncommon, Rare, etc.) have increased duration.

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
- **Reduction**:
  - `revive_small`: 60 seconds
  - `revive_medium`: 600 seconds
  - `revive_large`: 6000 seconds

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

## Implementation Details

The item logic is handled in `internal/user/item_handlers.go`. It interacts with the `TimeoutService` (Discord/Twitch integration) and the `Inventory` system.

- **Active Chatter Tracking**: The system tracks users who have recently messaged to determine valid targets for random-target items (Mines, TNT).
- **Quality Modifiers**: Item quality (Common, Uncommon, Rare, Epic, Legendary) scales the effect duration or power.

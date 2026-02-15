# Roadmap & Next Steps Suggestions

Here are 10 suggested next steps for BrandishBot_Go.
Items 1-3 are high-priority tasks retained from the previous analysis.
Items 4-10 are **new creative feature ideas** designed to increase player engagement and depth, separate from standard technical maintenance.

## Immediate Priorities (Kept)

### 1. Finish Duel System Implementation

**Type:** Core Feature | **Priority:** Critical
The Duel system is partially implemented but non-functional. The database tables exist, but the `Accept` logic is a placeholder and commands are unregistered.

- **Action:** Implement game logic in `internal/duel/service.go` (winner selection, payouts).
- **Action:** Register `/duel`, `/accept`, `/decline` commands.
- **Why:** Existing code is dead weight without this; it unlocks a major social interaction loop.

### 2. Implement Leaderboards

**Type:** Engagement | **Priority:** High
While `GetLeaderboard` exists in the Stats service, it needs to be exposed meaningfully to users to drive competition.

- **Action:** Create specific leaderboard commands for **Top Wealth**, **Highest Job Levels**, and **Most Duels Won**.
- **Action:** Add a "Weekly Champion" automated event that rewards the top player of the week.
- **Why:** Social competition is a primary driver for chat-based games.

### 3. Admin Dashboard Enhancements

**Type:** Tooling | **Priority:** Medium
The Admin Dashboard (`web/admin`) exists but needs expansion to support live-ops.

- **Action:** Add a **"Game Master"** panel to manually trigger events (e.g., spawn a Lootbox, start a Vote).
- **Action:** Add visual graphs for economy health (inflation monitoring).
- **Why:** Allows admins to keep the game alive and balanced without restarting the server or editing DBs.

## New Creative Feature Ideas

### 4. "Community Construction" Events

**Type:** Cooperative Gameplay
Instead of just personal progression, introduce global goals that the entire server contributes to.

- **Concept:** A "Grand Forge" or "Statue" that requires 10,000 Wood and 5,000 Stone to build.
- **Mechanic:** Users use a `/contribute` command to donate items.
- **Reward:** Upon completion, the entire server gets a **global buff** (e.g., +10% XP or +5% Luck) for 24 hours.
- **Why:** Fosters cooperation and drains excess resources from the economy.

### 5. Job Specializations (Sub-Classes)

**Type:** Progression Depth
Expand the current Job system (`internal/job`) to allow specialization at higher levels.

- **Concept:** Upon reaching Level 10 in a Job (e.g., Miner), the player must choose a path: **"Prospector"** (Higher Gem chance) or **"Excavator"** (Double Ore yield).
- **Mechanic:** Permanent flag on the user_jobs table. Unlock specific passive recipes or bonuses based on choice.
- **Why:** Adds "build variety" and trade-offs, encouraging players to trade with others who chose differently.

### 6. Item Set Bonuses

**Type:** Collection Mechanic
Encourage players to collect thematic sets of items rather than just the "best" one.

- **Concept:** Equipping or holding a full set of "Fisherman's Gear" (Hat, Rod, Boots) grants a bonus effect.
- **Mechanic:** `internal/stats` or `internal/character` check that applies a modifier when all specific items are in inventory.
- **Effect:** Passive bonus (e.g., "Fishing cooldown reduced by 10%").
- **Why:** Increases the value of lower-tier items if they are part of a desirable set.

### 7. The "Bounty Board"

**Type:** Player-Driven Economy
A system for players to request items they are too lazy to farm.

- **Concept:** Player A needs "50 Iron". They list a Bounty: "Buying 50 Iron for 500 Gold".
- **Mechanic:** Player B sees the bounty, fulfills it with `/fulfill [id]`, and the transaction happens automatically.
- **Difference from Market:** This is a **request** contract, whereas the Market is usually **sell** listings.
- **Why:** Connects wealthy players (who have gold but no time) with farmers (who have time but need gold).

### 8. Visual "Slot Machine" Minigame

**Type:** Instant Gratification / Sink
A text-based visual minigame for quick gambling interaction in chat.

- **Concept:** A command `/slots [amount]` that displays a 3x3 grid of emojis.
- **Mechanic:**
  🍒 | 🍒 | 🍋
  🔔 | 💎 | 🍒
  🍇 | 🍇 | 🍇
- **Logic:** Server generates the grid. Horizontal/Diagonal matches pay out multipliers.
- **Why:** Highly visual, "streamable" content that acts as a fun currency sink.

### 9. "Prestige" / Rebirth System

**Type:** Long-term Retention
An endgame mechanic for players who have "maxed out".

- **Concept:** Allow players to reset their Job Levels and Inventory back to zero.
- **Reward:** Gain 1 **"Soul Shard"** (or similar currency) that can be spent on permanent, account-wide upgrades (e.g., "Permanent +5% XP", "Reduced Cooldowns").
- **Why:** Solves the "I have finished the game" problem by creating a cyclical loop.

### 10. NPC Reputation System

**Type:** Lore / PVE
Introduce NPCs (like "Joey" from the Duel issues) that players can build reputation with.

- **Concept:** Gift items to NPCs to raise "Friendship".
- **Mechanic:** `/gift [NPC] [Item]`. Different NPCs like different items.
- **Reward:** At high friendship, NPCs send player exclusive gifts or unlock secret shops.
- **Why:** Gives "useless" items a purpose (gifts) and adds flavor/world-building to the bot.

# Player Commands & Mechanics Guide

> **Note**: This documentation is based on the backend logic found in `internal/gamble`, `internal/user`, `internal/economy`, `internal/expedition`, `internal/harvest`, and `internal/quest`.

## # Gamble

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `/gamble start <item> [qty]` | Starts a new High-Stakes Gamble. | **<item>** (xQty) |
| `/gamble join <id> <item> [qty]` | Joins the active Gamble. | **<item>** (xQty) |

> **Warning**: This is a "Winner Takes All" system. If you lose, you get nothing.

### 2. The Shout
Feeling lucky? Use `/gamble start lootbox1 1` to risk it all for GLORY!

### 3. The Helper
*   **Winner Takes All**: The highest total value of opened items wins EVERY lootbox bet by all participants.
*   **Tie-Breaks**: Resolved by RNG (and tears). Losers get a "Tie-Break Lost" record.
*   **Near Miss**: Score 95% of the winner? We track that pain as a "Near Miss".
*   **XP**: Awards **Gambler XP** per lootbox bet (plus a **Win Bonus**).

---

## # Economy & Items

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `/search` | Scavenge for items. 80% chance to find **Lootbox0**. | 30m Cooldown |
| `/buy <item> [qty]` | Buy items from the shop. | **Money** (Base Value) |
| `/sell <item> [qty]` | Sell items for cash. | **Item** |
| `/give <target> <item> [qty]` | Transfer items to another player. | **Item** |
| `/use <item> [qty] [target]` | Use an item (e.g., blaster, trap). | **Item** |
| `/use trap <target>` | Place a hidden trap on a user. | **Trap** |
| `/use mine` | Plant a mine on a random active user. | **Mine** |
| `/inventory` | View your item collection. | None |
| `/recipes` | View crafting recipes. | None |
| `/upgrade <recipe-id>` | Craft an item upgrade. | **Materials** |
| `/disassemble <item> [qty]` | Break down items for materials. | **Item** |

### 2. The Shout
Need cash? `/search` for loot or `/sell` your junk!
Silence the haters! `/use blaster @troll` to timeout them for 60s!

### 3. The Helper
See [docs/features/TRAPS.md](../features/TRAPS.md) for full details on item interactions.

*   **Lootbox0** (Cost: 10): Common. Contains **Money** (1-10).
*   **Lootbox1** (Cost: 50): Basic. Contains **Money** (10-100) or **Lootbox0** (50%).
*   **Lootbox2** (Cost: 100): Rare. Contains **Money** (100-500) or **Lootbox1** (50%). Has a 10% chance for a **Blaster**.
*   **Blaster**: Use with `/use blaster <target>`. Awards **Merchant XP** when buying/selling.
*   **Trap**: Use with `/use trap <target>`. Waits for the target to chat, then times them out.
*   **Mine**: Use with `/use mine`. Plants a trap that hits a random active chatter (or yourself!).

---

## # Expeditions

See [docs/features/EXPEDITIONS.md](../features/EXPEDITIONS.md).

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `/explore` | Start or join a cooperative expedition. | Global Cooldown (15m) |
| `/expedition-journal [id]` | View the journal of a completed expedition. | None |

### 2. The Shout
Gather your party! Use `/explore` to venture into the unknown and earn rare rewards!

### 3. The Helper
*   **Cooperative**: Work together with other players to survive 50 turns.
*   **Skills**: Your job levels (Blacksmith, Explorer, etc.) determine your success in encounters.
*   **Rewards**: Win to earn money, XP, and rare lootboxes. Even losing grants some rewards!
*   **Cooldown**: Expeditions have a global cooldown, so coordinate with your server!

---

## # Quests & Farming

See [docs/features/FARMING.md](../features/FARMING.md) and [docs/features/WEEKLY_QUESTS.md](../features/WEEKLY_QUESTS.md).

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `/quests` | View your weekly quests. | None |
| `/claimquest <id>` | Claim rewards for a completed quest. | None |
| `/harvest` | Harvest accumulated rewards from your farm. | 1h Minimum |

### 2. The Shout
Check `/quests` every week for big payouts! Don't forget to `/harvest` your crops!

### 3. The Helper
*   **Weekly Quests**: Reset every week. Complete them for Money and Merchant XP.
*   **Farming**: Your farm accumulates rewards passively over time.
*   **Harvest**: Rewards improve the longer you wait (up to 1 week).
*   **Spoilage**: If you wait longer than **2 weeks (336 hours)**, your crops spoil and you lose the main rewards!

---

## # Progression & Jobs

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `/vote <feature>` | Vote for the next feature to unlock. | None |
| `/job-progress` | Check your current job levels and XP. | None |
| `/stats` | View your overall statistics. | None |
| `/leaderboard` | View top players. | None |

### 2. The Shout
Want new features? Use `/vote` to make it happen! Level up your jobs to get stronger!

### 3. The Helper
*   **Community Driven**: Features unlock when the **Engagement Score** hits the target.
*   **Jobs**: Level up jobs like **Blacksmith**, **Explorer**, **Farmer**, **Gambler**, **Merchant**, and **Scholar** by performing related actions.
*   **Benefits**: Higher job levels improve your odds in Expeditions and other activities.

---

## # Account Linking

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `/link` | Link your Discord account to Twitch/YouTube. | None |
| `/unlink` | Unlink a platform from your account. | None |

### 2. The Helper
*   **Cross-Platform**: Link accounts to share inventory and stats across platforms.
*   **Subscriptions**: Linked accounts allow you to benefit from Twitch Subscriptions / YouTube Memberships in-game.

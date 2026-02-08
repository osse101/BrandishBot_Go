# Player Commands & Mechanics Guide

> **Note**: This documentation is auto-generated based on the backend logic found in `internal/gamble`, `internal/user`, `internal/economy`, and `configs/loot_tables.json`.

## # Gamble

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `!gamble start <item> [qty]` | Starts a new High-Stakes Gamble. | **<item>** (xQty) |
| `!gamble join <id> <item> [qty]` | Joins the active Gamble. | **<item>** (xQty) |

> **Warning**: This is a "Winner Takes All" system. If you lose, you get nothing.

### 2. Twitch Chat (The Shout)
Feeling lucky? Type `!gamble start lootbox1 1` to risk it all for GLORY!

### 3. Discord Chat (The Helper)
*   **Winner Takes All**: The highest total value of opened items wins EVERY lootbox bet by all participants.
*   **Tie-Breaks**: Resolved by RNG (and tears). Losers get a "Tie-Break Lost" record.
*   **Near Miss**: Score 95% of the winner? We track that pain as a "Near Miss".
*   **XP**: Awards **Gambler XP** per lootbox bet (plus a **Win Bonus**).

---

## # Economy & Items

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `!search` | Scavenge for items. 80% chance to find **Lootbox0**. | 30m Cooldown |
| `!buy <item> [qty]` | Buy items from the shop. | **Money** (Base Value) |
| `!sell <item> [qty]` | Sell items for cash. | **Item** |
| `!give <target> <item> [qty]` | Transfer items to another player. | **Item** |
| `!use blaster <target>` | Time out a user for 60 seconds. | 1x **Blaster** |
| `!use trap <target>` | Plant a hidden trap. Explodes when target speaks. | 1x **Trap** |

### 2. Twitch Chat (The Shout)
Need cash? `!search` for loot or `!sell` your junk!
Silence the haters! `!use blaster @troll` to timeout them for 60s!

### 3. Discord Chat (The Helper)
*   **Lootbox0** (Cost: 10): Common. Contains **Money** (1-10).
*   **Lootbox1** (Cost: 50): Basic. Contains **Money** (10-100) or **Lootbox0** (50%).
*   **Lootbox2** (Cost: 100): Rare. Contains **Money** (100-500) or **Lootbox1** (50%). Has a 10% chance for a **Blaster**.
*   **Blaster**: Use with `!use blaster <target>`. Awards **Merchant XP** when buying/selling.
*   **Trap**: Use with `!use trap <target>`. Waits for the target to chat, then times them out.

---

## # Progression

### 1. The Gist Entry (The Manual)

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| `!vote <feature>` | Vote for the next feature to unlock. | None |

### 2. Twitch Chat (The Shout)
Want new features? Type `!vote gamble` to make it happen!

### 3. Discord Chat (The Helper)
*   **Community Driven**: Features unlock when the **Engagement Score** hits the target.
*   **Contribute**: Chat, gamble, and play to increase the score.
*   **Unlocks**: Voting determines which feature (e.g., Gamble, Duel) unlocks next.

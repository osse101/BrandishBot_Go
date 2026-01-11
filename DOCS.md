# WikiBot Documentation 📝

## Tier 1: The Gist (The Manual)

# Core Commands

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| **!search** | Scavenge for items and XP. First search of the day is guaranteed success. | 30m Cooldown (Diminishing returns after 6/day) |
| **!inventory** | View your current stash of items. | Free |
| **!use <item> [target]** | Use an item (e.g., open a lootbox). | Consumes 1 Item |
| **!give <target> <item> <qty>** | Transfer items to another player. | Free |
| **!sell <item> <qty>** | Sell items for currency (Requires Feature Unlock). | Free |
| **!buy <item> <qty>** | Buy items with currency (Requires Feature Unlock). | Cost: Variable |

# Gambling & Economy

| Command | Description | Cost/Cooldown |
| :--- | :--- | :--- |
| **!gamble start** | Start a new gambling lobby with lootbox bets. | Requires Feature Unlock |
| **!gamble join** | Join an existing gambling lobby. | Bet Amount |

# Lootbox Contents & Odds

| Item Name | Contents | Odds |
| :--- | :--- | :--- |
| **Lootbox Tier 0** | Money (1-10) | 100% |
| **Lootbox Tier 1** | Money (10-100)<br>Lootbox Tier 0 | 100%<br>50% |
| **Lootbox Tier 2** | Money (100-500)<br>Lootbox Tier 1<br>**Blaster** | 100%<br>50%<br>10% |

# Mechanics Breakdown

> **Diminishing Returns**: To discourage spam, your `!search` effectiveness drops significantly after **6 searches per day**.
> *   **Success Rate**: Drops to 10%
> *   **XP Gain**: Drops to 10%

> **First Search Bonus**: Your first `!search` after the daily reset is a **Guaranteed Success**.

---

## Tier 2: Twitch Chat (The Shout)

*   **Search**: "!search is your daily bread. Don't forget your 6 daily runs for max efficiency!"
*   **Lootboxes**: "Feeling lucky? Open a Tier 2 box for a 10% shot at the **Blaster**!"
*   **Gamble**: "Risk it all! Use !gamble start to put your lootboxes on the line."

---

## Tier 3: Discord Chat (The Helper)

### Search Strategy 🔍
*   **Daily Cap**: You get **6 high-efficiency searches** per day. After that, success rates tank to 10%.
*   **First Run**: Your first search is always a win. Use it!
*   **XP**: Searching awards **Explorer Job XP**.

### Loot & Economy 💰
*   **Tier Up**: Tier 1 boxes have a 50% chance to drop a Tier 0 box. Tier 2 boxes can drop Tier 1s. It's boxes all the way down.
*   **The Jackpot**: The **Blaster** weapon is only found in **Lootbox Tier 2** (10% drop rate).
*   **Trading**: Use `!give` to trade with friends or `!sell` to cash out if you have the feature unlocked.

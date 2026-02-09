# Farming & Harvest System

The Farming system allows players to passively accumulate resources over time. By letting your "crops" grow, you can harvest increasingly valuable rewards.

## Core Mechanics

### Harvesting (`/harvest`)
- **Command**: `/harvest`
- **Requirement**: Must have the `feature_farming` progression node unlocked.
- **Minimum Wait**: 1 Hour.
- **Maximum Wait (Spoilage)**: 336 Hours (2 Weeks).

### Time & Tiers
The longer you wait between harvests, the more rewards you accumulate. You receive **all rewards** from every tier you have surpassed.

| Tier | Time Reached | Added Rewards | Cumulative Total |
| :--- | :--- | :--- | :--- |
| **1** | 2 Hours | 2 Money | 2 Money |
| **2** | 5 Hours | 10 Money | 12 Money |
| **3** | 12 Hours | 5 Money, 1 Stick | 17 Money, 1 Stick |
| **4** | 24 Hours | 5 Money, 2 Sticks | 22 Money, 3 Sticks |
| **5** | 48 Hours | 10 Money, 1 Lootbox0 | 32 Money, 3 Sticks, 1 Lootbox0 |
| **6** | 72 Hours | 10 Money, 2 Lootbox0 | 42 Money, 3 Sticks, 3 Lootbox0 |
| **7** | 90 Hours | 5 Money, 5 Sticks | 47 Money, 8 Sticks, 3 Lootbox0 |
| **8** | 110 Hours | 15 Money, 1 Lootbox1 | 62 Money, 8 Sticks, 3 Lootbox0, 1 Lootbox1 |
| **9** | 130 Hours | 15 Money, 1 Lootbox1 | 77 Money, 8 Sticks, 3 Lootbox0, 2 Lootbox1 |
| **10** | 168 Hours (1 Wk) | 20 Money, 1 Lootbox2 | 97 Money, 8 Sticks, 3 Lootbox0, 2 Lootbox1, 1 Lootbox2 |

> **Note**: Some items (like Sticks and Lootboxes) may require specific progression unlocks to be received. If you haven't unlocked the item, you won't get it, but you'll still get the money.

### Farmer XP
- **Unlock**: Starts accumulating after **5 hours**.
- **Rate**: 8 XP per hour.
- **Award**: Granted automatically upon harvest.

### Spoilage
If you wait longer than **336 hours (2 weeks)**, your crops will spoil!
- **Penalty**: You lose the standard accumulated rewards.
- **Salvage**: You receive a consolation prize of **1 Lootbox1** and **3 Sticks**.

## Implementation Details
The system is implemented in `internal/harvest/`.
- **Service**: `internal/harvest/service.go`
- **Tiers**: `internal/harvest/reward_tiers.go`
- **Persistence**: Harvest state is stored in the database, tracking `last_harvested_at`.

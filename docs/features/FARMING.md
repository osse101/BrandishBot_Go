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

### Technical Implementation

The harvest service employs robust concurrency patterns to ensure reliability:
- **Graceful Shutdown**: The service exposes a `Shutdown(ctx)` method and uses a `sync.WaitGroup` to ensure all background operations (like XP awarding) complete before the application stops.
- **Asynchronous XP Awarding**: Farmer XP is awarded asynchronously to prevent blocking the harvest transaction.
- **Context Management**: The asynchronous XP task uses `context.WithoutCancel` (Go 1.21+) to detach from the request context, ensuring the award process completes even if the user cancels the HTTP request immediately after the transaction commits.
- **Transaction Safety**: The harvest operation runs within a database transaction, ensuring the harvest timestamp is only updated if the rewards are successfully added to the inventory.

## Compost System

The Compost system allows players to recycle unwanted items into useful resources.

### Core Mechanics

- **Unlock**: Requires the `feature_compost` progression node.
- **Capacity**: Default bin holds **5 items**.
- **Efficiency**: Converts items at **50% value efficiency** (Default).
- **Output**: Produces the highest-value item possible of the **dominant input type**.

### How It Works

1.  **Deposit**: Add items with the `compostable` tag to your bin.
    -   *Command*: `/use compost <item> <amount>` (or via UI)
2.  **Process**: The bin processes items over time.
    -   **Warmup**: 1 Hour fixed time.
    -   **Per Item**: +30 Minutes per item.
    -   *Example*: 5 items take 1h + (5 * 30m) = 3.5 Hours.
3.  **Harvest**: Collect the result once ready.
    -   *Command*: `/harvest compost` (or via UI)
    -   *XP*: Awards XP equal to 10% of input value.

### Output Calculation

The system calculates the **Total Input Value** based on item base values and quality.
The **Output Value** is 50% of the Input Value.

The system determines the **Dominant Type** (e.g., Organic, Gem, Metal) based on what you deposited. It then rewards you with the most valuable item of that type that fits within the Output Value.

> **Example**:
> You deposit 10 Common Herbs (Value 100 each, Type: Organic).
> Total Input: 1000. Output Value: 500.
> Dominant Type: Organic.
> The system looks for Organic items worth <= 500.
> If "Premium Fertilizer" is worth 250, you receive 2 of them.

### Sludge (Spoilage)

If you leave your finished compost in the bin for too long (**1 Week** after finishing), it turns into **Sludge**.
-   **Reward**: `compost_sludge` (Quantity = Input Value / 10).
-   **Value**: Significantly less than a proper harvest.

## Implementation Details

The compost system uses a "Garbage In, Value Out" engine in `internal/compost/engine.go`.
-   **Service**: `internal/compost/service.go` handles transactions and validation.
-   **Engine**: Pure logic for calculating ready times and outputs.
-   **Events**: Publishes `compost.harvested` for stats and notifications.

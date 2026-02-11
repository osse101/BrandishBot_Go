# Economy System

The Economy system drives player progression by allowing the exchange of currency (Money) for items and resources. It integrates tightly with Jobs, Quests, and Progression systems.

## Core Mechanics

### Buying Items (`/buy`)
- **Requirements**:
  - The item must be **Buyable** (configured in database).
  - The player must have **Sufficient Funds** (Money).
  - The item must be **Unlocked** (via Progression Tree).
- **Price Calculation**:
  - `Base Price` = Configured item value.
  - **Weekly Sale Discount**: If `feature_weekly_discount` is unlocked, items in the weekly sale category receive a discount (e.g., 20% off Weapons).
- **XP Reward**: Buying items awards **Merchant XP** (`Cost / 10`).
- **Quest Tracking**: Purchases count towards relevant Weekly Quests.

### Selling Items (`/sell`)
- **Requirements**:
  - The player must have the item in their **Inventory**.
  - The item must be **Sellable** (configured in database).
- **Price Calculation**:
  - `Base Sell Price` = `Item Base Value * SellPriceRatio` (Default: 0.5).
  - **Economy Bonus**: If unlocked, the `economy_bonus` progression node increases the sell price multiplier.
- **XP Reward**: Selling items awards **Merchant XP** (`Value / 10`).
- **Quest Tracking**: Sales count towards relevant Weekly Quests.

## Weekly Sales
- **Rotation**: Discounts rotate weekly between item categories (e.g., Weapon -> Armor -> Consumable).
- **Schedule**: Defined in `configs/economy/weekly_sales.json`.
- **Eligibility**: Requires `feature_weekly_discount` to be unlocked.

## API Endpoints

### Get Sell Prices
```http
GET /api/v1/prices
```
Returns a list of all sellable items and their current sell prices (including bonuses).

### Get Buy Prices
```http
GET /api/v1/prices/buy
```
Returns a list of all buyable items and their current buy prices (including discounts).

### Buy Item
```http
POST /api/v1/user/item/buy
```
**Body**:
```json
{
  "platform": "twitch",
  "platform_id": "12345",
  "username": "buyer",
  "item_name": "wood_sword",
  "quantity": 1
}
```

### Sell Item
```http
POST /api/v1/user/item/sell
```
**Body**:
```json
{
  "platform": "twitch",
  "platform_id": "12345",
  "username": "seller",
  "item_name": "wood_sword",
  "quantity": 1
}
```

## Implementation Details

- **Service**: `internal/economy/service.go`
- **Repository**: `internal/repository/economy.go`
- **Configuration**: `configs/economy/weekly_sales.json`
- **Integration**:
  - `JobService` (Award Merchant XP)
  - `QuestService` (Track Quest Progress)
  - `ProgressionService` (Check Unlocks & Bonuses)

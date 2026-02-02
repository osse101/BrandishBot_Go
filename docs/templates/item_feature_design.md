# Item Feature Design Template: [Item Name]

## 1. Overview

Briefly describe the item and its role in the game.

- **Internal Name:** `item_example`
- **Public Name:** `example`
- **Display Name:** `A Mysterious Example`
- **Tier:** [0-4]
- **Tags:** `[consumable, tradeable, upgradeable, disassembleable, sellable, buyable]`

## 2. Utility & Mechanics

Detailed explanation of what happens when the item is used.

### Behavior

- [ ] Active Use (via `!use [item]`)
- [ ] Passive Effect
- [ ] Currency / Material

### API / Handler Input Args

What arguments does the handler expect in the `args` map?

- `target_username`: (string) The user being targeted (if applicable)
- `job_name`: (string) The job to apply effects to (if applicable)
- ...

### Handler Implementation

- **Existing Handler:** [e.g., `weapon`, `lootbox`, `revive`, `shield`, `rarecandy`]
- **New Handler Needed:** Yes/No (If yes, describe the logic)
- **Similar To:** [Existing item/handler this is based on]

## 3. Interactions

### Shine Interaction

How does the `ShineLevel` (COMMON, UNCOMMON, RARE, EPIC, LEGENDARY) affect the item?

- **Duration Multiplier:** (e.g., EPIC +50% duration)
- **Effect Potency:** (e.g., RARE +20% damage)
- **Drop Quality:** (e.g., Higher shine lootbox = better drop tables)
- **Visuals:** Any special messages or "juice" for high shine?

### Upgrade / Crafting Interaction

How does the item participate in the crafting system?

- **Is Upgradeable:** Yes/No
  - **Recipe Cost:** What materials are needed to "upgrade" to this item?
  - **Success Rewards:** Masterwork multiplier (e.g., 2x output on critical success).
- **Is Disassembleable:** Yes/No
  - **Scrap Output:** What materials are returned?
  - **Perfect Salvage:** Multiplier for perfect salvage events.

## 4. Systems Integration

Which existing systems are used, or what new systems are required?

- [ ] **Progression:** Gated behind a node? (e.g., `feature_farming`)
- [ ] **Job System:** Does it grant XP? Which job?
- [ ] **Stats/Events:** What events are recorded? (e.g., `EventLootboxJackpot`)
- [ ] **Cooldown:** Does it have a global or per-user cooldown?

## 5. Scope of Work

Identify high-level changes across the codebase.

### Repositories Affected

- `configs/items/items.json`: Definition and handler config.
- `internal/domain/item.go`: Any new domain constants.
- `internal/user/item_handlers.go`: Logic for `!use`.
- `internal/lootbox/service.go`: If it's a new type of lootbox.
- `internal/naming/constants.go`: Public name and display name mappings.
- `migrations/`: Any new database tables or schema changes.

### Similar Handlers

List existing handlers that can be referenced for implementation:

- `handleWeapon`: Reference for targeting logic.
- `handleLootbox`: Reference for drop processing.
- `handleShield`: Reference for buff/status application.

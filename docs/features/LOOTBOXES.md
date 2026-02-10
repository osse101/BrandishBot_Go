# Lootbox & Item Quality System

The Lootbox system provides players with randomized rewards, featuring a tiered structure and a quality mechanic that replaces the legacy "Shine" system.

## Lootbox Tiers

Lootboxes come in four distinct tiers, each with increasing rarity and potential rewards:

1.  **Lootbox Tier 0 (Junkbox)**: Contains basic resources and common items. Often found in low-level activities.
2.  **Lootbox Tier 1 (Common)**: Standard lootbox with a balanced mix of items.
3.  **Lootbox Tier 2 (Rare)**: Contains higher-quality items and rare crafting materials.
4.  **Lootbox Tier 3 (Diamond)**: The highest tier, guaranteeing at least one high-quality item and a chance for legendary drops.

## Item Quality System

Every item dropped from a lootbox has a "Quality" level, which affects its value and effectiveness. This system replaces the previous "Shine" mechanic.

### Quality Levels

| Quality Level | Multiplier | Description |
| :--- | :--- | :--- |
| **Common** | 1.0x | Standard quality. |
| **Uncommon** | 1.1x | Slightly better than standard. |
| **Rare** | 1.25x | High quality with a noticeable value boost. |
| **Epic** | 1.5x | Exceptional quality, significantly more valuable. |
| **Legendary** | 2.0x | The pinnacle of item quality. Double value. |
| **Poor** | 0.8x | Below average quality. Reduced value. |
| **Junk** | 0.6x | Very low quality. Significantly reduced value. |
| **Cursed** | 0.4x | The worst possible quality. Minimal value. |

### Drop Mechanics

When opening a lootbox:
1.  **Guaranteed Drops**: Some items are guaranteed based on the box tier.
2.  **Chance Drops**: Other items have a probability of dropping.
3.  **Quality Roll**: Each dropped item rolls for quality. Higher tier boxes increase the chance of better quality.
    *   **Currency Exception**: Currency items (e.g., Money) always drop as **Common** quality, but their *quantity* is multiplied by the quality multiplier rolled.

## Usage

Items can be opened using the `/use` command:
- `/use lootbox` (opens a Tier 1 box)
- `/use lootbox_tier2`
- `/use lootbox_tier3`
- `/use junkbox`

## Implementation Details

The system is implemented in `internal/lootbox/service.go`.
- **Configuration**: Loot tables are defined in `configs/loot_tables.json`.
- **Quality Logic**: `internal/utils/quality.go` and `internal/domain/item.go`.

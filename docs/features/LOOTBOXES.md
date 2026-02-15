# Lootbox & Item Quality System

The lootbox system provides players with randomized rewards through a tiered structure and quality mechanic. As of v2, drops use a **pool-based** model with a gatekeeper roll separating item drops from consolation money.

## Lootbox Tiers

| Tier | Public Name | Item Drop Rate | Consolation Money | Description |
|---|---|---|---|---|
| **Tier 0** | Junkbox | 30% | 1–10 | Basic resources and common items |
| **Tier 1** | Lootbox | 50% | 10–100 | Balanced mix of utility and weapons |
| **Tier 2** | Goldbox | 65% | 100–500 | Higher-quality weapons and rare items |
| **Tier 3** | Diamondbox | 80% | 500–2500 | Premium weapons, magical items, and tier-2 containers |

## Drop Pipeline (v2)

Each box opening runs a 3-stage pipeline:

```
Stage 1 — Gatekeeper
  roll < item_drop_rate  →  Stage 2 (item path)
  roll ≥ item_drop_rate  →  Consolation money (skip to next open)

Stage 2 — Pool selection
  Weighted random pick among the lootbox's pool references

Stage 3 — Item selection
  Weighted random pick among the chosen pool's items
```

Multiple opens of the same box (e.g. opening 5 at once) are aggregated — duplicate items sum their quantities before the quality roll.

### Consolation Money

When the gatekeeper roll fails the player receives money instead of an item. The amount is drawn from a jitter formula:

```
base   = rnd() × (money_max − money_min) + money_min
jitter = 1 + (rnd() − 0.5) × (1 − item_drop_rate)
amount = max(1, round(base × jitter))
```

The jitter term widens the spread as `item_drop_rate` decreases (lower-tier boxes have more variable consolation prizes). Consolation money is always **Common** quality and bypasses the quality roll.

## Named Pools

Pools are shared across boxes and defined in `configs/loot_tables.json`. An entry can reference a specific item (`item_name`) or expand to all items of a content type (`item_type`):

| Pool | Contents |
|---|---|
| `pool_utility` | Shovel, Video Filter, Stick, Scrap |
| `pool_weapons_basic` | This, Grenade, Blaster |
| `pool_weapons_premium` | Deez, Mirror Shield, Huge Blaster |
| `pool_explosives` | All `explosive`-type items (auto-expanded) |
| `pool_defense` | Shield, Mirror Shield |
| `pool_healing` | Revive Small |
| `pool_magical` | Rare Candy |
| `pool_containers_t0/t1/t2` | Lootbox Tier 0/1/2 (nested containers) |

Type expansion happens once at startup: `{"item_type": "explosive", "weight": 25}` inserts one weighted entry per matching item (each gets weight 25 independently).

## Item Quality System

Every item from the pool path rolls for quality. The roll is shifted by the box's own quality level:

| Quality | Multiplier | Roll threshold (Common box) |
|---|---|---|
| Legendary | 2.0x | ≤ 1% |
| Epic | 1.5x | ≤ 5% |
| Rare | 1.25x | ≤ 15% |
| Uncommon | 1.1x | ≤ 30% |
| Common | 1.0x | ≤ 70% |
| Poor | 0.8x | ≤ 85% |
| Junk | 0.6x | ≤ 95% |
| Cursed | 0.4x | > 95% |

Higher-tier boxes shift all thresholds upward (+3% per quality tier above Common), making rare outcomes more likely. Junk/Poor boxes shift them downward.

**Critical Upgrade** (progression-locked): 1% chance to upgrade the rolled quality by one tier.

**Currency exception**: Currency items (e.g. Money) always receive Common quality, but their *quantity* is multiplied by the quality multiplier instead of their value.

## Usage

```
/use junkbox          → opens Tier 0
/use lootbox          → opens Tier 1
/use goldbox          → opens Tier 2
/use diamondbox       → opens Tier 3
```

## Implementation

| Layer | Location |
|---|---|
| Service | `internal/lootbox/service.go` |
| Cache builder & pipeline | `internal/lootbox/processor.go` |
| Quality rolls & currency logic | `internal/lootbox/converter.go`, `quality.go` |
| Config | `configs/loot_tables.json` |
| Schema | `configs/schemas/loot_tables.schema.json` |

The cache (`map[string]*FlattenedLootbox`) is built once at startup via `GetAllItems` and is read-only during operation — no mutex required for concurrent opens.

### Orphan Tracking

At startup, any item in the database that is not referenced by any pool (excluding Money) emits a `WARN` log: `"Item not referenced in any pool (orphaned)"`. Items intentionally excluded from pools (e.g. `item_script`, `compost_sludge`) will appear in this log by design.

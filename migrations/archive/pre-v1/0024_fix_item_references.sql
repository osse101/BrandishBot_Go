-- +goose Up
-- Migration: Fix Item References to Use Internal Names
-- Description: Updates all item references in disassemble and type assignment tables
--              to use the new internal_name format (lootbox_tier0, lootbox_tier1, lootbox_tier2)
--              instead of the old item_name format (lootbox0, lootbox1, lootbox2)

-- Note: This migration assumes migration 0022_add_item_naming.sql has already run
-- and that the items table now uses internal_name column

-- Since the disassemble feature migration (0013) ran BEFORE the naming migration (0022),
-- it used the old 'item_name' column which no longer exists.
-- The data it inserted is still valid, but we need to ensure consistency going forward.

-- The tables are using item_id foreign keys, so the references are still valid.
-- However, for clarity and future migrations, we document the correct internal names here.

-- Verify current state (this is informational)
-- Expected: lootbox_tier1 should be marked as disassembleable
-- Expected: Disassemble recipe should exist for lootbox_tier1 -> lootbox_tier0

-- Ensure lootbox_tier1 is marked as disassembleable (using new internal_name)
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, it.item_type_id
FROM items i, item_types it
WHERE i.internal_name = 'lootbox_tier1' AND it.type_name = 'disassembleable'
ON CONFLICT DO NOTHING;

-- Ensure disassemble recipe exists for lootbox_tier1
INSERT INTO disassemble_recipes (source_item_id, quantity_consumed)
SELECT item_id, 1
FROM items
WHERE internal_name = 'lootbox_tier1'
ON CONFLICT (source_item_id) DO NOTHING;

-- Ensure output for the disassemble recipe (lootbox_tier0)
INSERT INTO disassemble_outputs (recipe_id, item_id, quantity)
SELECT dr.recipe_id, i.item_id, 1
FROM disassemble_recipes dr
JOIN items source ON dr.source_item_id = source.item_id
JOIN items i ON i.internal_name = 'lootbox_tier0'
WHERE source.internal_name = 'lootbox_tier1'
ON CONFLICT (recipe_id, item_id) DO NOTHING;

-- Ensure association between lootbox_tier0->lootbox_tier1 upgrade and lootbox_tier1->lootbox_tier0 disassemble
INSERT INTO recipe_associations (upgrade_recipe_id, disassemble_recipe_id)
SELECT cr.recipe_id, dr.recipe_id
FROM crafting_recipes cr
JOIN items upgrade_target ON cr.target_item_id = upgrade_target.item_id
JOIN disassemble_recipes dr ON dr.source_item_id = upgrade_target.item_id
WHERE upgrade_target.internal_name = 'lootbox_tier1'
ON CONFLICT (upgrade_recipe_id, disassemble_recipe_id) DO NOTHING;

-- +goose Down
-- This migration is idempotent and uses ON CONFLICT DO NOTHING
-- Rollback is not necessary as it doesn't delete or modify existing data

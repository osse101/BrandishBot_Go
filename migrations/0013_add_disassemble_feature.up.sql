-- Migration: Add Disassemble Feature
-- Description: Creates tables for disassemble recipes and adds the example lootbox1 -> lootbox0 recipe

-- Create disassemble_recipes table
CREATE TABLE IF NOT EXISTS disassemble_recipes (
    recipe_id SERIAL PRIMARY KEY,
    source_item_id INTEGER NOT NULL REFERENCES items(item_id),
    quantity_consumed INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_source_item UNIQUE (source_item_id)
);

-- Create disassemble_outputs table
CREATE TABLE IF NOT EXISTS disassemble_outputs (
    output_id SERIAL PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES disassemble_recipes(recipe_id) ON DELETE CASCADE,
    item_id INTEGER NOT NULL REFERENCES items(item_id),
    quantity INTEGER NOT NULL,
    CONSTRAINT unique_recipe_output UNIQUE (recipe_id, item_id)
);

-- Create recipe_associations table to link upgrade and disassemble recipes
CREATE TABLE IF NOT EXISTS recipe_associations (
    association_id SERIAL PRIMARY KEY,
    upgrade_recipe_id INTEGER NOT NULL REFERENCES crafting_recipes(recipe_id) ON DELETE CASCADE,
    disassemble_recipe_id INTEGER NOT NULL REFERENCES disassemble_recipes(recipe_id) ON DELETE CASCADE,
    CONSTRAINT unique_association UNIQUE (upgrade_recipe_id, disassemble_recipe_id)
);

-- Add 'disassembleable' item type
INSERT INTO item_types (type_name) 
VALUES ('disassembleable')
ON CONFLICT (type_name) DO NOTHING;

-- Mark lootbox1 as disassembleable
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, it.item_type_id
FROM items i, item_types it
WHERE i.item_name = 'lootbox1' AND it.type_name = 'disassembleable'
ON CONFLICT DO NOTHING;

-- Create disassemble recipe for lootbox1 -> lootbox0
INSERT INTO disassemble_recipes (source_item_id, quantity_consumed)
SELECT item_id, 1
FROM items
WHERE item_name = 'lootbox1'
ON CONFLICT (source_item_id) DO NOTHING;

-- Add output for the disassemble recipe (lootbox0)
INSERT INTO disassemble_outputs (recipe_id, item_id, quantity)
SELECT dr.recipe_id, i.item_id, 1
FROM disassemble_recipes dr
JOIN items source ON dr.source_item_id = source.item_id
JOIN items i ON i.item_name = 'lootbox0'
WHERE source.item_name = 'lootbox1'
ON CONFLICT (recipe_id, item_id) DO NOTHING;

-- Create association between lootbox0->lootbox1 upgrade and lootbox1->lootbox0 disassemble
-- First, find the upgrade recipe for lootbox1 (target_item_id)
-- Then link it to the disassemble recipe for lootbox1 (source_item_id)
INSERT INTO recipe_associations (upgrade_recipe_id, disassemble_recipe_id)
SELECT cr.recipe_id, dr.recipe_id
FROM crafting_recipes cr
JOIN items upgrade_target ON cr.target_item_id = upgrade_target.item_id
JOIN disassemble_recipes dr ON dr.source_item_id = upgrade_target.item_id
WHERE upgrade_target.item_name = 'lootbox1'
ON CONFLICT (upgrade_recipe_id, disassemble_recipe_id) DO NOTHING;

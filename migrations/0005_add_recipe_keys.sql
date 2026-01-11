-- +goose Up
-- Add recipe_key columns and update constraints for JSON-based recipe configuration

-- Add recipe_key to crafting_recipes
ALTER TABLE public.crafting_recipes 
    ADD COLUMN recipe_key VARCHAR(255);

-- Populate recipe_key for existing recipes (use target item's internal name as temporary key)
UPDATE public.crafting_recipes cr
SET recipe_key = i.internal_name
FROM public.items i
WHERE cr.target_item_id = i.item_id;

-- Make recipe_key NOT NULL after populating
ALTER TABLE public.crafting_recipes 
    ALTER COLUMN recipe_key SET NOT NULL;

-- Drop old unique constraint on target_item_id
ALTER TABLE public.crafting_recipes 
    DROP CONSTRAINT IF EXISTS crafting_recipes_target_item_id_key;

-- Add unique constraint on recipe_key
ALTER TABLE public.crafting_recipes 
    ADD CONSTRAINT crafting_recipes_recipe_key_key UNIQUE (recipe_key);

-- Add recipe_key to disassemble_recipes
ALTER TABLE public.disassemble_recipes 
    ADD COLUMN recipe_key VARCHAR(255);

-- Populate recipe_key for existing recipes (use source item's internal name as temporary key)
UPDATE public.disassemble_recipes dr
SET recipe_key = i.internal_name
FROM public.items i
WHERE dr.source_item_id = i.item_id;

-- Make recipe_key NOT NULL after populating
ALTER TABLE public.disassemble_recipes 
    ALTER COLUMN recipe_key SET NOT NULL;

-- Drop old unique constraint on source_item_id
ALTER TABLE public.disassemble_recipes 
    DROP CONSTRAINT IF EXISTS unique_source_item;

-- Add unique constraint on recipe_key
ALTER TABLE public.disassemble_recipes 
    ADD CONSTRAINT disassemble_recipes_recipe_key_key UNIQUE (recipe_key);

-- +goose Down
-- Restore original constraints and remove recipe_key columns

-- Restore unique constraint on target_item_id
ALTER TABLE public.crafting_recipes 
    ADD CONSTRAINT crafting_recipes_target_item_id_key UNIQUE (target_item_id);

-- Drop recipe_key constraint and column
ALTER TABLE public.crafting_recipes 
    DROP CONSTRAINT IF EXISTS crafting_recipes_recipe_key_key;
ALTER TABLE public.crafting_recipes 
    DROP COLUMN recipe_key;

-- Restore unique constraint on source_item_id
ALTER TABLE public.disassemble_recipes 
    ADD CONSTRAINT unique_source_item UNIQUE (source_item_id);

-- Drop recipe_key constraint and column
ALTER TABLE public.disassemble_recipes 
    DROP CONSTRAINT IF EXISTS disassemble_recipes_recipe_key_key;
ALTER TABLE public.disassemble_recipes 
    DROP COLUMN recipe_key;

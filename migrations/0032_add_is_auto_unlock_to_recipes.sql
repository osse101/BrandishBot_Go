-- +goose Up
-- Add is_auto_unlock column to crafting_recipes
ALTER TABLE public.crafting_recipes 
ADD COLUMN is_auto_unlock BOOLEAN NOT NULL DEFAULT FALSE;

-- Mark lootbox_tier0 as auto-unlocked
UPDATE public.crafting_recipes 
SET is_auto_unlock = TRUE 
WHERE recipe_key = 'lootbox_tier0';

-- +goose Down
-- Remove is_auto_unlock column
ALTER TABLE public.crafting_recipes 
DROP COLUMN is_auto_unlock;

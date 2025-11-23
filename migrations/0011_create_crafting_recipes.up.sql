-- Create crafting_recipes table
CREATE TABLE IF NOT EXISTS crafting_recipes (
    recipe_id SERIAL PRIMARY KEY,
    target_item_id INTEGER NOT NULL REFERENCES items(item_id) ON DELETE CASCADE,
    base_cost JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(target_item_id)
);

-- Create recipe_unlocks table to track which users have unlocked which recipes
CREATE TABLE IF NOT EXISTS recipe_unlocks (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    recipe_id INTEGER NOT NULL REFERENCES crafting_recipes(recipe_id) ON DELETE CASCADE,
    unlocked_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, recipe_id)
);

-- Index for fast recipe lookups by target item
CREATE INDEX IF NOT EXISTS idx_recipes_target_item ON crafting_recipes(target_item_id);

-- Index for fast unlock checks
CREATE INDEX IF NOT EXISTS idx_recipe_unlocks_user ON recipe_unlocks(user_id);

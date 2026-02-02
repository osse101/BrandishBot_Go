-- +goose Up
-- Add internal_name and public_name columns for item naming system

-- Step 1: Rename item_name to internal_name
ALTER TABLE items RENAME COLUMN item_name TO internal_name;

-- Step 2: Add public_name column
ALTER TABLE items ADD COLUMN public_name VARCHAR(100);

-- Step 3: Add handler column for linking to item handler functions
ALTER TABLE items ADD COLUMN handler VARCHAR(50);

-- Step 4: Add default_display column for fallback display name
ALTER TABLE items ADD COLUMN default_display VARCHAR(255);

-- Step 5: Create unique index on public_name
CREATE UNIQUE INDEX idx_items_public_name ON items(public_name) WHERE public_name IS NOT NULL;

-- Step 6: Update existing items with new naming scheme
UPDATE items SET 
    public_name = 'money',
    default_display = 'Coins',
    handler = NULL
WHERE internal_name = 'money';

UPDATE items SET 
    internal_name = 'lootbox_tier0',
    public_name = 'junkbox',
    default_display = 'Rusty Lootbox',
    handler = 'lootbox'
WHERE internal_name = 'lootbox0';

UPDATE items SET 
    internal_name = 'lootbox_tier1',
    public_name = 'lootbox',
    default_display = 'Basic Lootbox',
    handler = 'lootbox'
WHERE internal_name = 'lootbox1';

UPDATE items SET 
    internal_name = 'lootbox_tier2',
    public_name = 'goldbox',
    default_display = 'Golden Lootbox',
    handler = 'lootbox'
WHERE internal_name = 'lootbox2';

UPDATE items SET 
    internal_name = 'weapon_blaster',
    public_name = 'missile',
    default_display = 'Ray Gun',
    handler = 'blaster'
WHERE internal_name = 'blaster';

-- +goose Down
-- Revert item naming changes

-- Remove new columns
DROP INDEX IF EXISTS idx_items_public_name;
ALTER TABLE items DROP COLUMN IF EXISTS default_display;
ALTER TABLE items DROP COLUMN IF EXISTS handler;
ALTER TABLE items DROP COLUMN IF EXISTS public_name;

-- Rename back to item_name
ALTER TABLE items RENAME COLUMN internal_name TO item_name;

-- Revert item names to original
UPDATE items SET item_name = 'lootbox0' WHERE item_name = 'lootbox_tier0';
UPDATE items SET item_name = 'lootbox1' WHERE item_name = 'lootbox_tier1';
UPDATE items SET item_name = 'lootbox2' WHERE item_name = 'lootbox_tier2';
UPDATE items SET item_name = 'blaster' WHERE item_name = 'weapon_blaster';

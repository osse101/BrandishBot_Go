-- Seed test recipe: 1 lootbox0 upgrades to 1 lootbox1
-- Get item IDs first
DO $$
DECLARE
    lootbox0_id INT;
    lootbox1_id INT;
BEGIN
    -- Get item IDs
    SELECT item_id INTO lootbox0_id FROM items WHERE item_name = 'lootbox0';
    SELECT item_id INTO lootbox1_id FROM items WHERE item_name = 'lootbox1';
    
    -- Insert recipe: requires 1 lootbox0 to create 1 lootbox1
    INSERT INTO crafting_recipes (target_item_id, base_cost)
    VALUES (
        lootbox1_id,
        jsonb_build_array(
            jsonb_build_object('item_id', lootbox0_id, 'quantity', 1)
        )
    )
    ON CONFLICT (target_item_id) DO NOTHING;
    
    RAISE NOTICE 'Recipe created: 1 lootbox0 -> 1 lootbox1';
END $$;

-- Create test user and setup for upgrade testing
DO $$
DECLARE
    test_user_id UUID;
    recipe_id_val INT;
    lootbox0_id INT;
BEGIN
    -- Create test user if not exists
    INSERT INTO users (username, created_at, updated_at)
    VALUES ('testuser', NOW(), NOW())
    ON CONFLICT DO NOTHING
    RETURNING user_id INTO test_user_id;
    
    -- If user already exists, get their ID
    IF test_user_id IS NULL THEN
        SELECT user_id INTO test_user_id FROM users WHERE username = 'testuser';
    END IF;
    
    -- Get the recipe ID for lootbox1
    SELECT recipe_id INTO recipe_id_val FROM crafting_recipes 
    WHERE target_item_id = (SELECT item_id FROM items WHERE item_name = 'lootbox1');
    
    -- Unlock the recipe for test user
    INSERT INTO recipe_unlocks (user_id, recipe_id, unlocked_at)
    VALUES (test_user_id, recipe_id_val, NOW())
    ON CONFLICT DO NOTHING;
    
    -- Get lootbox0 item ID
    SELECT item_id INTO lootbox0_id FROM items WHERE item_name = 'lootbox0';
    
    -- Give user 10 lootbox0 items
    INSERT INTO user_inventory (user_id, inventory_data)
    VALUES (
        test_user_id,
        jsonb_build_object(
            'slots', jsonb_build_array(
                jsonb_build_object('item_id', lootbox0_id, 'quantity', 10)
            )
        )
    )
    ON CONFLICT (user_id) DO UPDATE
    SET inventory_data = jsonb_build_object(
        'slots', jsonb_build_array(
            jsonb_build_object('item_id', lootbox0_id, 'quantity', 10)
        )
    );
    
    RAISE NOTICE 'Test user setup complete:';
    RAISE NOTICE '  User ID: %', test_user_id;
    RAISE NOTICE '  Recipe unlocked: %', recipe_id_val;
    RAISE NOTICE '  Inventory: 10x lootbox0';
END $$;

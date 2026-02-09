-- +goose Up
-- Migration to consolidate currency items to COMMON quality
-- This fixes existing inventories with multiple currency stacks

-- Update user_inventory JSONB to merge all money slots into COMMON quality
UPDATE user_inventory
SET inventory_data = (
    SELECT jsonb_build_object(
        'slots',
        -- Combine non-money slots and consolidated money slot
        COALESCE(
            (
                -- Get all non-money slots unchanged
                SELECT jsonb_agg(slot)
                FROM jsonb_array_elements(inventory_data->'slots') AS slot
                WHERE (slot->>'item_id')::int != (SELECT item_id FROM items WHERE internal_name = 'money')
            ),
            '[]'::jsonb
        ) ||
        -- Add single money slot with total quantity if money exists
        CASE
            WHEN EXISTS (
                SELECT 1
                FROM jsonb_array_elements(inventory_data->'slots') AS slot
                WHERE (slot->>'item_id')::int = (SELECT item_id FROM items WHERE internal_name = 'money')
            )
            THEN jsonb_build_array(
                jsonb_build_object(
                    'item_id', (SELECT item_id FROM items WHERE internal_name = 'money'),
                    'quantity', (
                        SELECT COALESCE(SUM((slot->>'quantity')::int), 0)
                        FROM jsonb_array_elements(inventory_data->'slots') AS slot
                        WHERE (slot->>'item_id')::int = (SELECT item_id FROM items WHERE internal_name = 'money')
                    ),
                    'quality_level', 'COMMON'
                )
            )
            ELSE '[]'::jsonb
        END
    )
)
WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements(inventory_data->'slots') AS slot
    WHERE (slot->>'item_id')::int = (SELECT item_id FROM items WHERE internal_name = 'money')
);

-- +goose Down
-- Rollback not supported as this is a data consolidation
-- Users would need to restore from backup if rollback is required

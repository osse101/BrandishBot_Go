-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_traps RENAME COLUMN shine_level TO quality_level;

-- Update JSONB inventory data: rename 'shine' key to 'quality' in each object in the 'slots' array
UPDATE user_inventory
SET inventory_data = jsonb_set(
    inventory_data,
    '{slots}',
    COALESCE(
        (
            SELECT jsonb_agg(
                CASE 
                    WHEN slot ? 'shine' THEN (slot - 'shine') || jsonb_build_object('quality', slot->'shine')
                    ELSE slot
                END
            )
            FROM jsonb_array_elements(inventory_data->'slots') AS slot
        ),
        '[]'::jsonb
    )
)
WHERE inventory_data ? 'slots';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_traps RENAME COLUMN quality_level TO shine_level;

-- Revert JSONB inventory data: rename 'quality' key to 'shine' in each object in the 'slots' array
UPDATE user_inventory
SET inventory_data = jsonb_set(
    inventory_data,
    '{slots}',
    COALESCE(
        (
            SELECT jsonb_agg(
                CASE 
                    WHEN slot ? 'quality' THEN (slot - 'quality') || jsonb_build_object('shine', slot->'quality')
                    ELSE slot
                END
            )
            FROM jsonb_array_elements(inventory_data->'slots') AS slot
        ),
        '[]'::jsonb
    )
)
WHERE inventory_data ? 'slots';
-- +goose StatementEnd

-- +goose Up
-- Add currency item type
INSERT INTO item_types (type_name) VALUES ('currency') ON CONFLICT DO NOTHING;

-- Add money item
INSERT INTO items (item_name, item_description, base_value) 
VALUES ('money', 'Currency', 1) 
ON CONFLICT DO NOTHING;

-- Assign currency type to money
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name = 'money' AND t.type_name = 'currency'
ON CONFLICT DO NOTHING;

-- +goose Down
-- Remove currency type assignment from money
DELETE FROM item_type_assignments 
WHERE item_type_id IN (
    SELECT item_type_id FROM item_types WHERE type_name = 'currency'
);

-- Remove money item
DELETE FROM items WHERE item_name = 'money';

-- Remove currency type
DELETE FROM item_types WHERE type_name = 'currency';

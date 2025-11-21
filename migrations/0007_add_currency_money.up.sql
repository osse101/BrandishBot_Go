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

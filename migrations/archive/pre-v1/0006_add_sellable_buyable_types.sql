-- +goose Up
-- Add sellable and buyable item types
INSERT INTO item_types (type_name) VALUES ('sellable'), ('buyable') ON CONFLICT DO NOTHING;

-- Assign types to lootboxes
-- lootbox0: buyable only
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name = 'lootbox0' AND t.type_name = 'buyable'
ON CONFLICT DO NOTHING;

-- lootbox1: both sellable and buyable
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name = 'lootbox1' AND t.type_name IN ('sellable', 'buyable')
ON CONFLICT DO NOTHING;

-- lootbox2: sellable only
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name = 'lootbox2' AND t.type_name = 'sellable'
ON CONFLICT DO NOTHING;

-- +goose Down
-- Remove sellable/buyable type assignments
DELETE FROM item_type_assignments 
WHERE item_type_id IN (
    SELECT item_type_id FROM item_types WHERE type_name IN ('sellable', 'buyable')
);

-- Remove sellable/buyable types
DELETE FROM item_types WHERE type_name IN ('sellable', 'buyable');

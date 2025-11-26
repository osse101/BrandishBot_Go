-- +goose Up
INSERT INTO items (item_name, item_description, base_value)
VALUES ('blaster', 'So anyway, I started blasting', 10)
ON CONFLICT (item_name) DO NOTHING;

INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name = 'blaster' AND t.type_name = 'consumable'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM item_type_assignments
WHERE item_id IN (SELECT item_id FROM items WHERE item_name = 'blaster');

DELETE FROM items WHERE item_name = 'blaster';

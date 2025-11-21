-- Seed Item Types
INSERT INTO item_types (type_name) VALUES ('consumable'), ('upgradeable') ON CONFLICT DO NOTHING;

-- Seed Items (Lootboxes)
INSERT INTO items (item_name, item_description, base_value) VALUES 
('lootbox0', 'Junk Lootbox', 10),
('lootbox1', 'Basic Lootbox', 50),
('lootbox2', 'Good Lootbox', 100)
ON CONFLICT DO NOTHING;

-- Assign Types to Items
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name IN ('lootbox0', 'lootbox1', 'lootbox2')
  AND t.type_name IN ('consumable', 'upgradeable')
ON CONFLICT DO NOTHING;

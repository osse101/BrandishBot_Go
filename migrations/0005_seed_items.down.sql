-- Remove seed data
DELETE FROM item_type_assignments WHERE item_id IN (
    SELECT item_id FROM items WHERE item_name IN ('lootbox0', 'lootbox1', 'lootbox2')
);
DELETE FROM items WHERE item_name IN ('lootbox0', 'lootbox1', 'lootbox2');
DELETE FROM item_types WHERE type_name IN ('consumable', 'upgradeable');

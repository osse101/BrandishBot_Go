DELETE FROM item_type_assignments
WHERE item_id IN (SELECT item_id FROM items WHERE item_name = 'blaster');

DELETE FROM items WHERE item_name = 'blaster';

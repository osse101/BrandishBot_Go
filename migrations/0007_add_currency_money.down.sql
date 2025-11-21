-- Remove currency type assignment from money
DELETE FROM item_type_assignments 
WHERE item_type_id IN (
    SELECT item_type_id FROM item_types WHERE type_name = 'currency'
);

-- Remove money item
DELETE FROM items WHERE item_name = 'money';

-- Remove currency type
DELETE FROM item_types WHERE type_name = 'currency';

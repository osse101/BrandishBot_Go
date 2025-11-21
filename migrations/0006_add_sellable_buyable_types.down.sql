-- Remove sellable/buyable type assignments
DELETE FROM item_type_assignments 
WHERE item_type_id IN (
    SELECT item_type_id FROM item_types WHERE type_name IN ('sellable', 'buyable')
);

-- Remove sellable/buyable types
DELETE FROM item_types WHERE type_name IN ('sellable', 'buyable');

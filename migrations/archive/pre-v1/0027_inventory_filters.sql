-- +goose Up
-- Remove inventory slot upgrades (refunds are not handled automatically here, assuming dev/reset environment or manual compensation if needed)
DELETE FROM progression_nodes WHERE node_key = 'upgrade_inventory_slot';

-- Add new item types
INSERT INTO item_types (type_name) VALUES 
('upgradable'), 
('consumable'), 
('material'),
('weapon')
ON CONFLICT DO NOTHING;

-- Assign types to items
-- lootbox0 (Rusty Lockbox): upgradable, consumable
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.internal_name = 'lootbox0' AND t.type_name IN ('upgradable', 'consumable')
ON CONFLICT DO NOTHING;

-- lootbox1 (Standard Lockbox): upgradable, consumable
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.internal_name = 'lootbox1' AND t.type_name IN ('upgradable', 'consumable')
ON CONFLICT DO NOTHING;

-- lootbox2 (Gold Lockbox): consumable (maybe not upgradable if it's top tier?) -> Let's say just consumable for now.
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.internal_name = 'lootbox2' AND t.type_name = 'consumable'
ON CONFLICT DO NOTHING;

-- weapon_blaster: weapon
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.internal_name = 'weapon_blaster' AND t.type_name = 'weapon'
ON CONFLICT DO NOTHING;

-- Add Progression Nodes for Filters
-- Filter: Upgrade (under feature_upgrade)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'feature_filter_upgrade', 'feature', 'Inventory Filter: Upgrade', 'Unlock the /inventory filter:upgrade command', id, 1, 50, 60
FROM progression_nodes WHERE node_key = 'feature_upgrade'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'feature_filter_upgrade');

-- Filter: Sellable (under feature_sell)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'feature_filter_sellable', 'feature', 'Inventory Filter: Sellable', 'Unlock the /inventory filter:sellable command', id, 1, 50, 60
FROM progression_nodes WHERE node_key = 'feature_sell'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'feature_filter_sellable');

-- Filter: Consumable (under feature_economy - generic start)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'feature_filter_consumable', 'feature', 'Inventory Filter: Consumable', 'Unlock the /inventory filter:consumable command', id, 1, 50, 41
FROM progression_nodes WHERE node_key = 'feature_economy'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'feature_filter_consumable');


-- +goose Down
-- WARNING: This rollback will DELETE progression nodes that users may have unlocked
-- If any users have unlocked these filter features, their unlock records will become orphaned
-- Consider archiving to a backup table instead in production environments

-- Remove user unlocks for filter features (prevents orphaned records)
DELETE FROM user_progression_unlocks WHERE node_id IN (
    SELECT id FROM progression_nodes WHERE node_key IN ('feature_filter_upgrade', 'feature_filter_sellable', 'feature_filter_consumable')
);

-- Remove nodes
DELETE FROM progression_nodes WHERE node_key IN ('feature_filter_upgrade', 'feature_filter_sellable', 'feature_filter_consumable');

-- Remove item type assignments (specific ones added above)
-- Hard to be precise without IDs, but safe to leave types if they remain unused.

-- Restore slot upgrade (optional, but good for reversibility)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'upgrade_inventory_slot', 'upgrade', 'Extra Inventory Slot', 'Increase inventory capacity by 1', id, 10, 2500, 41
FROM progression_nodes WHERE node_key = 'feature_economy'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'upgrade_inventory_slot');

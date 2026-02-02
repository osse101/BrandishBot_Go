-- +goose Up
-- Update engagement weights to reflect desired progression pace
UPDATE engagement_weights SET weight = 1.0 WHERE metric_type = 'message';
UPDATE engagement_weights SET weight = 25.0 WHERE metric_type = 'search_performed';
UPDATE engagement_weights SET weight = 200.0 WHERE metric_type = 'item_crafted'; -- Significant progress for crafting
UPDATE engagement_weights SET weight = 50.0 WHERE metric_type = 'item_sold';
UPDATE engagement_weights SET weight = 50.0 WHERE metric_type = 'item_bought';
UPDATE engagement_weights SET weight = 50.0 WHERE metric_type = 'item_disassembled';
UPDATE engagement_weights SET weight = 10.0 WHERE metric_type = 'item_used';

-- Ensure all metrics exist
INSERT INTO engagement_weights (metric_type, weight, description) 
VALUES ('search_performed', 25.0, 'Performed a search') ON CONFLICT (metric_type) DO UPDATE SET weight = 25.0;
INSERT INTO engagement_weights (metric_type, weight, description) 
VALUES ('item_crafted', 200.0, 'Crafted an item') ON CONFLICT (metric_type) DO UPDATE SET weight = 200.0;
INSERT INTO engagement_weights (metric_type, weight, description) 
VALUES ('item_sold', 50.0, 'Sold an item') ON CONFLICT (metric_type) DO UPDATE SET weight = 50.0;
INSERT INTO engagement_weights (metric_type, weight, description) 
VALUES ('item_bought', 50.0, 'Bought an item') ON CONFLICT (metric_type) DO UPDATE SET weight = 50.0;
INSERT INTO engagement_weights (metric_type, weight, description) 
VALUES ('item_disassembled', 50.0, 'Disassembled an item') ON CONFLICT (metric_type) DO UPDATE SET weight = 50.0;
INSERT INTO engagement_weights (metric_type, weight, description) 
VALUES ('item_used', 10.0, 'Used an item') ON CONFLICT (metric_type) DO UPDATE SET weight = 10.0;

-- Update existing node costs to create better curve (Scaled for small community of ~5-10 active users)
-- Basic features (Week 1 target)
UPDATE progression_nodes SET unlock_cost = 2500 WHERE node_key = 'item_money';
UPDATE progression_nodes SET unlock_cost = 2500 WHERE node_key = 'item_lootbox0';
UPDATE progression_nodes SET unlock_cost = 4000 WHERE node_key = 'feature_search';
UPDATE progression_nodes SET unlock_cost = 4000 WHERE node_key = 'feature_disassemble';

-- Mid-tier features
UPDATE progression_nodes SET unlock_cost = 7500 WHERE node_key = 'feature_economy';
UPDATE progression_nodes SET unlock_cost = 7500 WHERE node_key = 'feature_upgrade';
UPDATE progression_nodes SET unlock_cost = 6000 WHERE node_key = 'feature_buy';
UPDATE progression_nodes SET unlock_cost = 6000 WHERE node_key = 'feature_sell';

-- Advanced features
UPDATE progression_nodes SET unlock_cost = 15000 WHERE node_key = 'feature_gamble';
UPDATE progression_nodes SET unlock_cost = 20000 WHERE node_key = 'feature_duel';
UPDATE progression_nodes SET unlock_cost = 20000 WHERE node_key = 'feature_expedition';

-- Add "Increased Value of Contributions" node
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'upgrade_contribution_boost', 'upgrade', 'Contribution Boost', 'Increase contribution points from all sources by 50%', id, 1, 50000, 100
FROM progression_nodes WHERE node_key = 'feature_economy'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'upgrade_contribution_boost');

-- Add Job Unlocks (Large features)
-- Blacksmith (requires Upgrade)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'job_blacksmith', 'feature', 'Unlock Job: Blacksmith', 'Unlock the Blacksmith job path', id, 1, 25000, 50
FROM progression_nodes WHERE node_key = 'feature_upgrade'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'job_blacksmith');

UPDATE progression_nodes SET unlock_cost = 25000 WHERE node_key = 'job_blacksmith';

-- Explorer (requires Search)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'job_explorer', 'feature', 'Unlock Job: Explorer', 'Unlock the Explorer job path', id, 1, 25000, 51
FROM progression_nodes WHERE node_key = 'feature_search'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'job_explorer');

UPDATE progression_nodes SET unlock_cost = 25000 WHERE node_key = 'job_explorer';

-- Merchant (requires Economy)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'job_merchant', 'feature', 'Unlock Job: Merchant', 'Unlock the Merchant job path', id, 1, 25000, 52
FROM progression_nodes WHERE node_key = 'feature_economy'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'job_merchant');

UPDATE progression_nodes SET unlock_cost = 25000 WHERE node_key = 'job_merchant';

-- Gambler (requires Gamble)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'job_gambler', 'feature', 'Unlock Job: Gambler', 'Unlock the Gambler job path', id, 1, 30000, 53
FROM progression_nodes WHERE node_key = 'feature_gamble'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'job_gambler');

UPDATE progression_nodes SET unlock_cost = 30000 WHERE node_key = 'job_gambler';

-- Farmer (requires Search)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'job_farmer', 'feature', 'Unlock Job: Farmer', 'Unlock the Farmer job path', id, 1, 30000, 54
FROM progression_nodes WHERE node_key = 'feature_search'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'job_farmer');

UPDATE progression_nodes SET unlock_cost = 30000 WHERE node_key = 'job_farmer';

-- Add Inventory Slot Upgrades (Small unlocks)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
SELECT 'upgrade_inventory_slot', 'upgrade', 'Extra Inventory Slot', 'Increase inventory capacity by 1', id, 10, 2500, 41
FROM progression_nodes WHERE node_key = 'feature_economy'
AND NOT EXISTS (SELECT 1 FROM progression_nodes WHERE node_key = 'upgrade_inventory_slot');

UPDATE progression_nodes SET unlock_cost = 2500 WHERE node_key = 'upgrade_inventory_slot';

-- +goose Down
DELETE FROM progression_nodes WHERE node_key IN (
  'upgrade_contribution_boost', 
  'job_blacksmith', 'job_explorer', 'job_merchant', 'job_gambler', 'job_farmer',
  'upgrade_inventory_slot'
);

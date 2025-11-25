-- +goose Up
-- Seed initial progression tree

-- Root: Unlock progression system itself (auto-unlocked, cost 0)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order) VALUES
('progression_system', 'feature', 'Progression System', 'Unlock the community progression system', NULL, 1, 0, 0);

-- Tier 1: First choices after root (low cost - 500)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order) VALUES
('item_money', 'item', 'Money', 'Unlock the money item for trading', (SELECT id FROM progression_nodes WHERE node_key = 'progression_system'), 1, 500, 1),
('item_lootbox0', 'item', 'Basic Lootbox', 'Unlock lootbox0 (basic lootbox)', (SELECT id FROM progression_nodes WHERE node_key = 'progression_system'), 1, 500, 2);

-- Tier 2: Medium features (cost 1000-1500)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order) VALUES
('feature_economy', 'feature', 'Economy System', 'Unlock buy/sell marketplace features', (SELECT id FROM progression_nodes WHERE node_key = 'item_money'), 1, 1500, 10),
('feature_upgrade', 'feature', 'Upgrade Items', 'Unlock item upgrading/crafting system', (SELECT id FROM progression_nodes WHERE node_key = 'item_lootbox0'), 1, 1500, 20),
('feature_disassemble', 'feature', 'Disassemble Items', 'Unlock item disassembly for materials', (SELECT id FROM progression_nodes WHERE node_key = 'item_lootbox0'), 1, 1000, 21),
('feature_search', 'feature', 'Search System', 'Unlock item search and filtering', (SELECT id FROM progression_nodes WHERE node_key = 'item_lootbox0'), 1, 1000, 23);

-- Tier 3: Sub-features and items (cost 800-1200)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order) VALUES
('feature_buy', 'feature', 'Buy Items', 'Unlock item purchasing from market', (SELECT id FROM progression_nodes WHERE node_key = 'feature_economy'), 1, 800, 11),
('feature_sell', 'feature', 'Sell Items', 'Unlock item selling to market', (SELECT id FROM progression_nodes WHERE node_key = 'feature_economy'), 1, 800, 12),
('item_lootbox1', 'item', 'Advanced Lootbox', 'Unlock lootbox1 with better rewards', (SELECT id FROM progression_nodes WHERE node_key = 'feature_upgrade'), 1, 1200, 22);

-- Tier 4: Advanced features (cost 2500-3000)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order) VALUES
('feature_gamble', 'feature', 'Gambling', 'Unlock gambling mechanics (high risk/reward)', (SELECT id FROM progression_nodes WHERE node_key = 'feature_economy'), 1, 2500, 30),
('feature_duel', 'feature', 'Dueling', 'Unlock player vs player combat duels', (SELECT id FROM progression_nodes WHERE node_key = 'item_lootbox1'), 1, 3000, 31),
('feature_expedition', 'feature', 'Expeditions', 'Unlock expedition/adventure system', (SELECT id FROM progression_nodes WHERE node_key = 'feature_search'), 1, 3000, 32);

-- Incremental upgrade example (5 levels, consistent cost per level)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order) VALUES
('upgrade_cooldown_reduction', 'upgrade', 'Cooldown Reduction', 'Reduce command cooldowns (unlockable 5 times)', (SELECT id FROM progression_nodes WHERE node_key = 'feature_economy'), 5, 1500, 40);

-- Auto-unlock root node
INSERT INTO progression_unlocks (node_id, current_level, unlocked_by, engagement_score) VALUES
((SELECT id FROM progression_nodes WHERE node_key = 'progression_system'), 1, 'auto', 0);

-- +goose Down
DELETE FROM progression_unlocks WHERE unlocked_by = 'auto';
DELETE FROM progression_nodes WHERE node_key IN (
    'progression_system', 'item_money', 'item_lootbox0', 'feature_economy',
    'feature_upgrade', 'feature_disassemble', 'feature_search', 'feature_buy',
    'feature_sell', 'item_lootbox1', 'feature_gamble', 'feature_duel',
    'feature_expedition', 'upgrade_cooldown_reduction'
);

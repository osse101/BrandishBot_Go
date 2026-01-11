-- +goose Up
INSERT INTO progression_nodes (node_key, node_type, display_name, description, tier, size, category, unlock_cost, max_level, sort_order)
VALUES ('item_money', 'item', 'Money', 'Currency', 1, 'small', 'item', 100, 1, 10)
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM progression_nodes WHERE node_key = 'item_money';

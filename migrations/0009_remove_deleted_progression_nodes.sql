-- +goose Up
-- Migration: Remove deleted progression nodes from v0.9 cleanup
-- These nodes were removed from configs/progression_tree.json

-- Clear winning_option_id from sessions referencing these nodes
UPDATE progression_voting_sessions SET winning_option_id = NULL
WHERE winning_option_id IN (
    SELECT pvo.id FROM progression_voting_options pvo
    JOIN progression_nodes pn ON pvo.node_id = pn.id
    WHERE pn.node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Remove user votes referencing voting options for these nodes
DELETE FROM user_votes WHERE option_id IN (
    SELECT pvo.id FROM progression_voting_options pvo
    JOIN progression_nodes pn ON pvo.node_id = pn.id
    WHERE pn.node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Remove voting options for these nodes
DELETE FROM progression_voting_options WHERE node_id IN (
    SELECT id FROM progression_nodes WHERE node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Clear unlock progress references to these nodes
UPDATE progression_unlock_progress SET node_id = NULL
WHERE node_id IN (
    SELECT id FROM progression_nodes WHERE node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Remove unlock records for these nodes
DELETE FROM progression_unlocks WHERE node_id IN (
    SELECT id FROM progression_nodes WHERE node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Remove prerequisite relationships pointing to these nodes
DELETE FROM progression_prerequisites WHERE prerequisite_node_id IN (
    SELECT id FROM progression_nodes WHERE node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Remove prerequisite relationships where these nodes depend on something
DELETE FROM progression_prerequisites WHERE node_id IN (
    SELECT id FROM progression_nodes WHERE node_key IN (
        'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
        'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
        'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
    )
);

-- Finally, remove the nodes themselves
DELETE FROM progression_nodes WHERE node_key IN (
    'feature_buy', 'feature_sell', 'item_bigmissile', 'item_deez',
    'item_mine', 'item_revivem', 'item_revivel', 'item_trap',
    'upgrade_job_daily_cap', 'upgrade_search_cooldown_reduction'
);

-- +goose Down
-- Cannot restore deleted nodes - this is a one-way migration
-- To rollback, restore from backup or re-sync from progression_tree.json

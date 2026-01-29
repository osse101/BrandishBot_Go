-- name: EnsureInventoryRow :exec
INSERT INTO user_inventory (user_id, inventory_data)
VALUES ($1, $2)
ON CONFLICT (user_id) DO NOTHING;

-- name: GetInventoryForUpdate :one
SELECT inventory_data FROM user_inventory WHERE user_id = $1 FOR UPDATE;

-- name: GetInventory :one
SELECT inventory_data FROM user_inventory WHERE user_id = $1;

-- name: UpdateInventory :exec
INSERT INTO user_inventory (user_id, inventory_data)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET inventory_data = EXCLUDED.inventory_data;

-- name: CreateUser :one
INSERT INTO users (username, created_at, updated_at)
VALUES ($1, NOW(), NOW())
RETURNING user_id;

-- name: UpdateUser :exec
UPDATE users 
SET username = $1, updated_at = NOW()
WHERE user_id = $2;

-- name: GetPlatformID :one
SELECT platform_id FROM platforms WHERE name = $1;

-- name: UpsertUserPlatformLink :exec
INSERT INTO user_platform_links (user_id, platform_id, platform_user_id)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, platform_id) DO UPDATE
SET platform_user_id = EXCLUDED.platform_user_id;

-- name: GetUserByPlatformID :one
SELECT u.user_id, u.username
FROM users u
JOIN user_platform_links upl ON u.user_id = upl.user_id
JOIN platforms p ON upl.platform_id = p.platform_id
WHERE p.name = $1 AND upl.platform_user_id = $2;

-- name: GetUserPlatformLinks :many
SELECT p.name, upl.platform_user_id
FROM user_platform_links upl
JOIN platforms p ON upl.platform_id = p.platform_id
WHERE upl.user_id = $1;

-- name: GetUserByPlatformUsername :one
SELECT u.user_id, u.username
FROM users u
JOIN user_platform_links upl ON u.user_id = upl.user_id
JOIN platforms p ON upl.platform_id = p.platform_id
WHERE LOWER(u.username) = LOWER($1)
AND p.name = $2;

-- name: GetItemByName :one
SELECT 
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
WHERE i.internal_name = $1
GROUP BY i.item_id;

-- name: GetItemByPublicName :one
SELECT 
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
WHERE i.public_name = $1
GROUP BY i.item_id;

-- name: GetItemsByIDs :many
SELECT 
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
WHERE i.item_id = ANY($1::int[])
GROUP BY i.item_id;

-- name: GetItemsByNames :many
SELECT 
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
WHERE i.internal_name = ANY($1::text[])
GROUP BY i.item_id;

-- name: GetItemByID :one
SELECT 
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
WHERE i.item_id = $1
GROUP BY i.item_id;

-- name: GetAllItems :many
SELECT 
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
GROUP BY i.item_id
ORDER BY i.item_id;

-- name: GetSellablePrices :many
SELECT DISTINCT i.internal_name, i.public_name, i.base_value
FROM items i
INNER JOIN item_type_assignments ita ON i.item_id = ita.item_id
INNER JOIN item_types it ON ita.item_type_id = it.item_type_id
WHERE it.type_name = 'sellable' AND i.public_name IS NOT NULL
ORDER BY i.public_name;

-- name: IsItemBuyable :one
SELECT EXISTS (
    SELECT 1
    FROM items i
    JOIN item_type_assignments ita ON i.item_id = ita.item_id
    JOIN item_types it ON ita.item_type_id = it.item_type_id
    WHERE i.internal_name = $1 AND it.type_name = 'buyable'
);

-- name: GetRecipeByTargetItemID :one
SELECT recipe_id, target_item_id, base_cost, created_at FROM crafting_recipes WHERE target_item_id = $1;

-- name: IsRecipeUnlocked :one
SELECT EXISTS (SELECT 1 FROM recipe_unlocks WHERE user_id = $1 AND recipe_id = $2);

-- name: UnlockRecipe :exec
INSERT INTO recipe_unlocks (user_id, recipe_id, unlocked_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id, recipe_id) DO NOTHING;

-- name: GetUnlockedRecipesForUser :many
SELECT i.internal_name AS item_name, r.target_item_id AS item_id
FROM crafting_recipes r
JOIN recipe_unlocks ru ON r.recipe_id = ru.recipe_id
JOIN items i ON r.target_item_id = i.item_id
WHERE ru.user_id = $1
ORDER BY i.internal_name;

-- name: GetAllRecipes :many
SELECT i.internal_name AS item_name, r.target_item_id AS item_id, i.item_description
FROM crafting_recipes r
JOIN items i ON r.target_item_id = i.item_id
ORDER BY i.internal_name;

-- name: GetDisassembleRecipeBySourceItemID :one
SELECT recipe_id, source_item_id, quantity_consumed, created_at
FROM disassemble_recipes
WHERE source_item_id = $1;

-- name: GetDisassembleOutputs :many
SELECT item_id, quantity
FROM disassemble_outputs
WHERE recipe_id = $1
ORDER BY item_id;

-- name: GetAssociatedUpgradeRecipeID :one
SELECT upgrade_recipe_id
FROM recipe_associations
WHERE disassemble_recipe_id = $1;

-- name: GetLastCooldown :one
SELECT last_used_at
FROM user_cooldowns
WHERE user_id = $1 AND action_name = $2;

-- name: GetLastCooldownForUpdate :one
SELECT last_used_at
FROM user_cooldowns
WHERE user_id = $1 AND action_name = $2
FOR UPDATE;

-- name: UpdateCooldown :exec
INSERT INTO user_cooldowns (user_id, action_name, last_used_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, action_name) DO UPDATE
SET last_used_at = EXCLUDED.last_used_at;

-- name: GetBuyablePrices :many
SELECT DISTINCT i.internal_name, i.public_name, i.base_value
FROM items i
INNER JOIN item_type_assignments ita ON i.item_id = ita.item_id
INNER JOIN item_types it ON ita.item_type_id = it.item_type_id
WHERE it.type_name = 'buyable' AND i.public_name IS NOT NULL
ORDER BY i.public_name;

-- name: GetUserByID :one
SELECT user_id, username, created_at, updated_at FROM users WHERE user_id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE user_id = $1;

-- name: DeleteInventory :exec
DELETE FROM user_inventory WHERE user_id = $1;

-- name: DeleteUserPlatformLink :exec
DELETE FROM user_platform_links 
WHERE user_id = $1 
AND platform_id = (SELECT platform_id FROM platforms WHERE name = $2);

-- name: UpdateUserTimestamp :exec
UPDATE users SET updated_at = NOW() WHERE user_id = $1;

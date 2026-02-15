-- Item Repository Queries

-- name: GetItemByInternalName :one
SELECT
    i.item_id, i.internal_name, i.public_name, i.default_display, i.item_description, i.base_value, i.handler,
    i.content_type,
    COALESCE(array_agg(t.type_name) FILTER (WHERE t.type_name IS NOT NULL), '{}')::text[] as types
FROM items i
LEFT JOIN item_type_assignments ita ON i.item_id = ita.item_id
LEFT JOIN item_types t ON ita.item_type_id = t.item_type_id
WHERE i.internal_name = $1
GROUP BY i.item_id;

-- name: InsertItem :one
INSERT INTO items (internal_name, public_name, default_display, item_description, base_value, handler, content_type)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING item_id;

-- name: UpdateItem :exec
UPDATE items
SET public_name = $1, default_display = $2, item_description = $3, base_value = $4, handler = $5, content_type = $6
WHERE item_id = $7;

-- name: GetAllItemTypes :many
SELECT item_type_id, type_name FROM item_types ORDER BY type_name;

-- name: InsertItemType :one
INSERT INTO item_types (type_name)
VALUES ($1)
ON CONFLICT (type_name) DO UPDATE SET type_name = EXCLUDED.type_name
RETURNING item_type_id;

-- name: ClearItemTags :exec
DELETE FROM item_type_assignments WHERE item_id = $1;

-- name: AssignItemTag :exec
INSERT INTO item_type_assignments (item_id, item_type_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetSyncMetadata :one
SELECT config_name, last_sync_time, file_hash, file_mod_time
FROM config_sync_metadata
WHERE config_name = $1;

-- name: UpsertSyncMetadata :exec
INSERT INTO config_sync_metadata (config_name, last_sync_time, file_hash, file_mod_time)
VALUES ($1, $2, $3, $4)
ON CONFLICT (config_name) DO UPDATE
SET last_sync_time = EXCLUDED.last_sync_time,
    file_hash = EXCLUDED.file_hash,
    file_mod_time = EXCLUDED.file_mod_time;

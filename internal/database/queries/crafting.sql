-- Crafting Recipe Repository Queries

-- name: GetAllCraftingRecipes :many
SELECT recipe_id, recipe_key, target_item_id, base_cost, created_at
FROM crafting_recipes
ORDER BY recipe_id;

-- name: GetCraftingRecipeByKey :one
SELECT recipe_id, recipe_key, target_item_id, base_cost, created_at
FROM crafting_recipes
WHERE recipe_key = $1;

-- name: InsertCraftingRecipe :one
INSERT INTO crafting_recipes (recipe_key, target_item_id, base_cost)
VALUES ($1, $2, $3)
RETURNING recipe_id;

-- name: UpdateCraftingRecipe :exec
UPDATE crafting_recipes
SET recipe_key = $1, target_item_id = $2, base_cost = $3
WHERE recipe_id = $4;

-- name: GetAllDisassembleRecipes :many
SELECT recipe_id, recipe_key, source_item_id, quantity_consumed, created_at
FROM disassemble_recipes
ORDER BY recipe_id;

-- name: GetDisassembleRecipeByKey :one
SELECT recipe_id, recipe_key, source_item_id, quantity_consumed, created_at
FROM disassemble_recipes
WHERE recipe_key = $1;

-- name: InsertDisassembleRecipe :one
INSERT INTO disassemble_recipes (recipe_key, source_item_id, quantity_consumed)
VALUES ($1, $2, $3)
RETURNING recipe_id;

-- name: UpdateDisassembleRecipe :exec
UPDATE disassemble_recipes
SET recipe_key = $1, source_item_id = $2, quantity_consumed = $3
WHERE recipe_id = $4;

-- name: ClearDisassembleOutputs :exec
DELETE FROM disassemble_outputs
WHERE recipe_id = $1;

-- name: InsertDisassembleOutput :exec
INSERT INTO disassemble_outputs (recipe_id, item_id, quantity)
VALUES ($1, $2, $3);

-- name: UpsertRecipeAssociation :exec
INSERT INTO recipe_associations (upgrade_recipe_id, disassemble_recipe_id)
VALUES ($1, $2)
ON CONFLICT (upgrade_recipe_id, disassemble_recipe_id) DO NOTHING;

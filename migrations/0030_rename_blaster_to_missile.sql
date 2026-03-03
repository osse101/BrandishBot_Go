-- +goose Up
UPDATE items SET internal_name = 'weapon_missile' WHERE internal_name = 'weapon_blaster';
UPDATE items SET internal_name = 'weapon_hugemissile' WHERE internal_name = 'weapon_hugeblaster';

-- +goose Down
UPDATE items SET internal_name = 'weapon_blaster' WHERE internal_name = 'weapon_missile';
UPDATE items SET internal_name = 'weapon_hugeblaster' WHERE internal_name = 'weapon_hugemissile';

-- +goose Up
ALTER TABLE public.user_platform_links ADD COLUMN platform_username character varying(255);

-- +goose Down
ALTER TABLE public.user_platform_links DROP COLUMN platform_username;

## 2024-05-23 - PixelPusher Initialization
**Learning:** Found that `migrations/0022_add_item_naming.sql` maps internal IDs to display names.
**Action:** Use `DefaultDisplay` field for naming generated assets (e.g., `RustyLootbox.png` instead of `lootbox0.png`).

## 2024-05-23 - Job Definitions
**Learning:** `migrations/0019_create_job_tables.sql` contains job display names and associated features.
**Action:** Generate job icons based on these names (e.g., `Blacksmith.png`, `Explorer.png`).

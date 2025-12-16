# LevelUp Journal

## Critical Learnings

- **Feedback Loop Missing**: Players have no feedback for daily engagement/consistency. Implementing "Daily Login Streak" to address this.
- **Service/Repo Boundaries**: Adding logic to `RecordUserEvent` allows capturing all activities as "login" events without needing a dedicated "Login" event, but requires careful recursion handling (excluding the streak event itself).
- **Visual Rarity Feedback**: Added `ShineLevel` to lootbox drops and gamble results. This allows the frontend to trigger specific visual effects (Common vs Legendary shine) based on backend probability logic, without the client knowing the probability tables.
- **Gamified RNG**: Implemented a Value Multiplier linked to Shine Level (e.g., Legendary = 2.0x Value). This ensures that "Critical Shine" events aren't just cosmetic but actively boost the player's score in competitive contexts like Gambles.

# Patch Notes (Week of April 3, 2026)

## System Updates & Bug Fixes

- **Daily Resets**: Added a startup check to automatically trigger any daily resets that might have been missed while the bot was offline.
- **Cooldowns**: Fixed a bug where user-specific progression bonuses were not properly reducing cooldowns (like the search cooldown reduction) because the user ID wasn't being passed correctly.
- **Lootboxes**: Fixed an issue where money drops from junkboxes had messy formatting. Money drops are now grouped under a single generic name before being displayed.
- **Items**: Updated the description message when deploying a Bomb.
- **Behind the Scenes**: Replaced the global `http.Client` with a shared singleton in devtools, and added explicit timeouts to the Discord health checks to prevent silent hanging.

# Patch Notes (Week of March 20, 2026)

## New Features & Gameplay Updates

- **Bomb Item**: Watch out! A new "Bomb" item has been added to shake things up.
- **Expeditions**: Joining expeditions is now smoother—you no longer need an ID! We've also added better gating to keep things balanced.
- **Jobs & Rewards**: We've tweaked XP gains and made it so your harvest spoil rewards now scale nicely with your job level.
- **Display Improvements**: We polished the look of Gamble payouts and the Job Progress command so they are easier to read.

## System Updates & Bug Fixes

- **General Fixes**: Resolved an annoying bug where junkbox recipes wrongly showed as locked. We also fixed issues with engagement tracking, target selection, and the video filter.
- **Progression**: Progression nodes are now smarter and will react automatically if item costs change.
- **Behind the Scenes**: A huge week for internal polish! We streamlined our code across several systems (like Quests, Slots, and Predictions), expanded our testing, and supercharged our developer tools to make future updates even faster.

# Patch Notes (Week of March 13, 2026)

## New Features

- **Duels**: The groundwork for challenging other players to Duels has been laid!
- **Gamble Updates**: Gamble events now display the winner's username, making it easier to see who won big.
- **Weapon 'This'**: The `weapon_this` item now correctly targets and times out the user who uses it.
- **Search Regions**: Added new Search Regions to expand your exploration options.

## System Updates

- **Progression Tiers**: Introduced progression tiers and improved how items unlock as the community progresses.
- **Item Display**: Fixed how certain items are displayed and added new aliases to make commands easier.

## Bug Fixes

- **General fixes**: Resolved issues with the Disassemble feature, Gamble commands, and Discord job commands.
- **Account Linking**: Fixed bugs related to linking your accounts.

# Patch Notes (Week of March 6, 2026)

## New Features

- **Item Unlocking System**: We've updated the progression system to correctly handle item unlocking, making sure you don't get access to new items until the community unlocks them.
- **Video Filters**: Added fun new video filter items to customize your experience!

## System Updates

- **Developer Tools**: Improved the internal developer tools by consolidating mock generation, making it safer and faster to test the bot across different platforms.
- **Behind the Scenes**: Refactored the internal Slots minigame code and loading utilities to keep the bot running smoothly and efficiently.

# Patch Notes (Week of Feb 23, 2026)

## Developer Experience

- **Devtool Upgrade**: `cmd/devtool` now supports strict flags, a `-watch` mode for rapid feedback loops using `fsnotify`, and a `-smart` mode to only run tests for changed packages.
- **Admin Dashboard Tech Stack**: The admin dashboard has been modernized with React 19, Vite, TypeScript, and Tailwind CSS.

## System Updates

- **Code Refactoring**: Active Chatter Tracking logic has been split into semantic files (`active_chatter_*.go`) for better maintainability.
- **Job System**: `bonus_config` table introduced to unify progression and job bonuses.
- **Documentation**: Updated `DEVTOOL.md`, `ADMIN_DASHBOARD_USAGE.md`, `JOBS.md`, and `TRAPS.md` to reflect recent changes.

# Patch Notes (Week of Feb 20, 2026)

## New Features

- **Chat Interaction**: We've added a new "String Finder" system! The bot can now detect specific keywords in chat (like "Bapanada") and trigger fun responses.
- **Timeout Stacking**: Timeouts are now smarter. If you get hit by multiple traps or weapons, the duration will stack instead of resetting.

## System Updates

- **Admin Tools**: Admins now have the ability to instantly clear user timeouts from the dashboard.
- **Active Chatter Tracking**: We've improved how active chatters are tracked to provide better engagement insights.
- **Lootbox Logic**: The loot distribution engine has been refined for better performance and consistency.
- **Job System**: Job definitions have been seeded into the database, laying the groundwork for upcoming job-specific bonuses.

## Bug Fixes

- **General**: Various code cleanups and database optimizations to keep things running smoothly.
- **Documentation**: Updated documentation for the new Chat Interaction system.

# Patch Notes (Week of Feb 13, 2026)

## New Features

- **Admin Dashboard**: Manage users, view system health, and execute commands via a new web interface.
- **Slots Minigame**: Test your luck with the new Slots minigame! Spin to win big rewards.
- **Subscriptions**: Added support for Twitch and YouTube memberships with exclusive benefits.

## System Updates

- **Item Quality**: The "Shine" system has been officially renamed to "Quality" for clarity.
- **Database Optimization**: Major backend updates to improve stability and performance.
- **Inventory Stacking**: Improved how items stack in your inventory.

# Patch Notes (Week of Feb 6, 2026)

## New Features

- **Expeditions**: Send your characters on expeditions to gather resources! (Unlockable via progression).
- **Quests**: New quest system to challenge your skills.
- **Farming**: Get your hands dirty with the new Farming and Compost systems.
- **New Job**: The **Scholar** job is now available.
- **Traps & Mines**: Watch your step! Trap items have been added, and Mines now behave like traps.
- **Predictions**: Participate in prediction events with new engagement features.

## Gameplay Updates

- **Lootboxes**: Now feature rarity levels—get better loot!
- **Weapons**: All weapons can now have "Quality" visual effects.
- **Progression**: Reduced the cost of unlock nodes and balanced Job XP gain.
- **Voting**: New users are automatically registered when voting. The previous vote winner is now displayed.

## Bug Fixes

- **Crafting**: Fixed a pesky bug where crafting would fail if your materials were split in your inventory.
- **Cooldowns**: Actions that fail will no longer consume your cooldowns.
- **General**: Fixed a crash when buying free items and improved overall system stability.

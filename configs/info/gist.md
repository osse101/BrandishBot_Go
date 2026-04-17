Twitch: https://www.twitch.tv/osse101
Youtube: https://www.youtube.com/@osse_101
Discord: [https://discord.gg/sVfcvRPR](https://discord.gg/rmfN56RXGb)

# Table of Contents

1. [Timeouts](#timeouts)
2. [Mines](#mines)
3. [Input Sabotage](#input-sabotage)
4. [Shuffler](#shuffler)
5. [TextToSpeech](#TestToSpeech)
6. [Commands](#commands)
7. [Gambling Game](#gambling-game)
8. [Duels](#duels)
9. [Expeditions](#expeditions)
10. [Use Item](#use-item)
11. [Item Store](#item-store)
12. [Upgrade Item](#upgrade-item)
13. [Convert Item](#convert-item)
14. [Classes](#classes)
15. [Statistics](#statistics)
16. [Stream Control](#stream-control)
17. [Discord Bot](#discord-bot)
18. [Unsorted](#unsorted)

---

# Timeouts

TODO: 🚧 Use !toggleO to join or leave the digging game.🚧

**Timeout a specific user**

| Item                 | Description                                                      | Cheer    |
| -------------------- | ---------------------------------------------------------------- | -------- |
| `missile <user>`     | 60s timeout.                                                     | cheer100 |
| `hugemissile <user>` | 600s timeout                                                     |          |
| `grenade`            | Timeout a random chatter                                         |          |
| `tnt`                | Timeout random chatters                                          |          |
| `revives <user>`     | Revives a user from a timeout                                    |          |
| `mine`               | Timeout a random user with a delay for 60s                       |          |
| `trap <user>`        | Timeout a user for 60s with a delay                              |          |
| `bomb`               | Timeout a group with a delay for 60s                             |          |
| `shield`             | Prevent the next timeout on you                                  |          |
| `mirrorshield`       | Prevent the next timeout on you and timeout the user who used it |          |

---

# Input Sabotage

**Send a controller input to my game**
**_Not always Active_**

| Command           | Description                                                            |
| ----------------- | ---------------------------------------------------------------------- |
| `cheer5 <button>` | Send a single input. Point redeemable. Input item.                     |
| `cheer10 S`       | Send a Start input. Point redeemable. Input item.                      |
| `cheer200`        | Enable unlimited input presses for 1 minute.                           |
| `cheer600`        | Enable unlimited input presses for 5 minutes.                          |
| `!input <button>` | Send a single input on a short cooldown.                               |
| `<button>`        | Automatically consume input items with single character chat messages. |

**Available buttons**:

| Button Name | Command |
| ----------- | ------- |
| A           | `a`     |
| B           | `b`     |
| X           | `x`     |
| Y           | `y`     |
| Left        | `l`     |
| Up          | `u`     |
| Right       | `r`     |
| Down        | `d`     |
| L           | `L`     |
| R           | `R`     |
| Select      | `s`     |
| Start       | `S`     |

**_Start is more expensive_**

---

# Shuffler

**Swap the current active game**
**_Not always Active_**

| Command     | Description                            |
| ----------- | -------------------------------------- |
| `cheer50`   | **"Swap"** point redeem. Swap item.    |
| `!swap`     | Free swap on a 30s CD.                 |
| `!use swap` | Consume a swap item to perform a swap. |

---

# TextToSpeech

| Command    | Description                                      |
| ---------- | ------------------------------------------------ |
| `cheer100` | Default Brian voice will read your message.      |
| `cheer200` | Specify which AI voice to read your message.     |
| `!tts`     | [Full instructions](https://tts.monster/osse101) |

---

# Commands

| Command                             | Description                                                                             |
| ----------------------------------- | --------------------------------------------------------------------------------------- |
| `!info <topic>`                     | Get usage information about the bot.                                                    |
| `!inventory <user>` `!items` `!inv` | Get a list of all items you currently own. Can be used to view another users inventory. |
| `!use <item>`                       | Use an item. Add **parameters** as needed by the item.                                  |
| `!use lootbox`                      | Randomly selects a lootbox to use.                                                      |
| `!give <user> <item> <amount>`      | Give items to another user.                                                             |
| `!upgrade <item>` `!up`             | Trade many of 1 item to receive a similar but better item.                              |
| `!disassemble <item>` `!down`       | Trade 1 item to recieve many worse items.                                               |
| `!dig` `!search`                    | Attempt to find an item while avoiding any mines.                                       |
| `!addFeature <Description>`         | Add an item to my TODO list.                                                            |
| `!readFeature [feature id]`         | Read an item to my TODO list.                                                           |
| `!submit <Suggestion>`              | Submit a game or seed to my backlog.                                                    |
| `!getTimeout <target>` `!gett`      | Get the remaining timeout on a user.                                                    |
| `!song`                             | Get the current rainwave song.                                                          |

---

# Gambling Game

**Community lootbox showdown**

| Command                      | Description                                                                                                       |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `!gamble <lootbox> [amount]` | Spend the listed lootboxes to open a gambling game lobby for chat. Everyone wagers the same item type and amount. |
| `!gamble`                    | Join the active gambling game before the join timer ends by matching the host's wager.                            |
| `!info  gamble`              | Post a quick reminder in chat explaining how the gambling game works.                                             |

When the timer expires, every participant's lootboxes are opened. The highest total shard value wins the entire pot.

---

# Duels

**Challenge another chatter to a wagered duel.**

| Command               | Description                                                                                                                         |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `!duel <user>`        | Challenge someone. Both players must have a Stick and enough shards to cover the wager. The bot creates a prediction when possible. |
| _(challenged player)_ | Accept or decline with `!duel` or `!duel decline`. Declining refunds the challenger’s stake.                                        |

When a duel starts, each duelist puts in 100 shards and their Stick. The winner receives double the wagered shards while the loser eats a 60 second timeout. If the duel isn’t accepted in time, everyone gets their items back automatically.【F:CustomCommands/Duel.cs†L19-L376】

---

# Expeditions

**Group adventures that appear when an expedition is queued.**

- A Leader starts an Expedition that anyone is free to join with !explore or !join.
- The party' composition will decide how successful the expedition will be.
- A expedition journal will describe what troubles the party faced and treasures found

---

# Use Item

**Use an item from your inventory. Multiple may be used at one time.**

| Item                         | Description                                                      |
| ---------------------------- | ---------------------------------------------------------------- |
| `missile <target>`           | Timeout the target for 60s                                       |
| `hugemissile <target>`       | Timeout the target for 600s                                      |
| `ReviveS <target>`           | Reduce timeout the target for 60s                                |
| `Shovel`                     | Destroy the Shovel to gain a Stick                               |
| `Stick`                      | Destroy the Stick                                                |
| `Shield`                     | Prevents the next timeout                                        |
| `MirrorShield`               | Prevents the next timeout and reflects it to another random user |
| `Mine`                       | Place a Mine                                                     |
| `Trap <target>`              | Place a Trap on target user.                                     |
| `Tnt`                        | Time out a random number of users in chat.                       |
| `Lootbox[x]`                 | Get a chance at recieving another item.                          |
| `fx <effectName>`            | Enable a webcam effect for 60 seconds.                           |
| `rarecandy <class> [amount]` | Gain 1 Level in Class.                                           |

---

## Item Store

**Shards**

Shards are the currency used in the store. They are commonly acquired through lootboxes or point redeems.
`!buy  <item> <amount>` -- Buy items from the shop.
`!sell <item> <amount>` -- Sell items for shards. Items sell at 1/10th the value.
`!sell <item> <amount> <value>` -- Post items for sale for the listed price. Items are taken immediately but sold when a buyer is found.

| Item       | Value |
| ---------- | ----- |
| Blaster    | 1000  |
| BigBlaster | 5000  |
| Mine       | 75    |
| Trap       | 750   |
| Input      | 10    |
| Swap       | 100   |
| Poll       | 200   |
| Fx         | 300   |
| Stick      | 100   |
| Shield     | 100   |
| Shovel     | 200   |
| Lootbox0   | 100   |
| Lootbox1   | 500   |
| Lootbox2   | 1000  |
| This       | 101   |

---

## Upgrade Item

**Upgrade items from your inventory**

| Item         | Quantity | New Item      |
| ------------ | -------- | ------------- |
| `Blaster`    | 10       | `BigBlaster`  |
| `BigBlaster` | 10       | `HugeBlaster` |
| `Mine`       | 10       | `Trap`        |
| `Trap`       | 10       | `Tnt`         |
| `Stick`      | 10       | `Shield`      |
| `Input`      | 10       | `Swap`        |
| `LootBox1`   | 10       | `LootBox2`    |

---

## Disassemble Item

**Downgrade items from your inventory**

- `!disassemble <item> [amount]` — Disassemble items in bulk. Add an amount if you only want to convert part of your stack.

| Item | Quantity | New Item |
| ---- | -------- | -------- |

---

## Jobs

Gain experience by performing common actions. Experience becomes levels and levels provide passive bonuses when performing those actions in the future.

| Job        | Description                          | Bonus                                      | Special Bonus                    |
| ---------- | ------------------------------------ | ------------------------------------------ | -------------------------------- |
| Denizen    | Here for the minefield               | Enables interactions with the digging game | None                             |
| Medic      | Specializes in healing others        | Stronger revives                           | Chance of free revive            |
| Looter     | Specializes in Digging               | Reduced chance of hitting a mine           | Increased lootbox chance         |
| Criminal   | Specializes in random chaos          | Increased timeout of mines                 | Larger TNTs                      |
| Lawman     | Specializes in blasting others       | Increased blaster timeout                  | Chance of free blaster           |
| Blacksmith | Specializes in Crafting and Upgrades | Chance of a bonus upgrade                  | Reduced base cost                |
| Broker     | Specializes in trading               | Improved shop prices                       | ??                               |
| Farmer     | Specializes in farming               | ??                                         | ??                               |
| Antagonist | Specializes in getting timed out     | Reduced timeout time                       | Gain shards from being timed out |

| Command             | Description                               |
| ------------------- | ----------------------------------------- |
| `!getClass` `!getc` | Get user's job levels and specialization. |

---

## Statistics

**Add a username to get information about that user.**

| Stat     | Description                           |
| -------- | ------------------------------------- |
| `!spent` | Total number of channel points spent. |

---

## Stream Control

**Commands for Subs and Mods**

**_Toggle OBS items with `!obs <element>`_**
| Element | Description |
|-------------|-------------|
| `alerts` | Sub renewal messages, raid messages, TTS |
| `bgm` | Background music (rainwave.cc) |
| `cam`, `cam fix` | Webcam |
| `chat` | Chat on stream |
| `gamesound` | Gamesound |
| `slideshow` | The map rando strat playlist slideshow |
| `knock` | Twitch Knock Knock |
| `tracker` | EmoTracker |
| `incident` | The "Time since last incident" counter |
| `cat` | Cat Videos |

| Command              | Description                    |
| -------------------- | ------------------------------ |
| `!brb`               | Toggles twitch channel clips   |
| `!game <name>`       | Change the game name on twitch |
| `!title <new title>` | Change the stream title        |

**_Temporarily activate webcam filters with `!use fx <effectName>`_**

> `frameskip`, `matrix`, `pixelate`, `glitch`, `bw`, `perspective`, `rainbow`, `sick`, `vhs`, `thermal`, `page`, `zoom`, `outline`, `bloom`, `gameboy`

**Plugins**
| Name & Link | Description |
|-----------------------------------------------------------------------------|--------------------------------------------------------------|
| [OBS ShaderFilter](https://obsproject.com/forum/resources/obs-shaderfilter.1736/) | Apply custom GLSL shaders as filters to sources and scenes. |
| [Retro Effects](https://obsproject.com/forum/resources/retro-effects.1972/) | Add retro-style visual effects (CRT, pixelation, scanlines). |

---

## Discord Bot

Usernames must match the user's twitch name.

| Command            | Description              |
| ------------------ | ------------------------ |
| `/inventory @user` | Get the user's inventory |

---

## Unsorted

| Action         | Description              |
| -------------- | ------------------------ |
| `cheer25`      | Buy a shovel.            |
| `cheerX shard` | Gain X number of shards. |

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

| Item                       | Description                                           |
| -------------------------- | ----------------------------------------------------- |
| `missile <target>`         | Timeout target for 60s.                               |
| `hugemissile <target>`     | Timeout target for 100 minutes.                       |
| `grenade`                  | Randomly timeout a user for 60s.                      |
| `tnt`                      | Randomly timeout a group.                             |
| `mine`                     | Delayed timeout on a random chatter.                  |
| `trap <target>`            | Delayed timeout on a targetted chatter.               |
| `bomb`                     | Delayed timeout on a group.                           |
| `revives <target>`         | Reduce timeout by 60s.                                |
| `stick`                    | A simple wooden stick.                                |
| `shovel`                   | Destroy the Shovel. (Commonly used for digging)       |
| `shield`                   | Prevents the next weapon attack.                      |
| `mirrorshield`             | Prevents the next weapon attack and reflects it back. |
| `lootbox[0-3]`             | Open for random items and currency.                   |
| `rarecandy <job> [amount]` | Instantly grants 500 XP to the selected Job.          |
| `filter <effect>`          | Apply a webcam filter for 60 seconds.                 |

---

## Item Store

**Shards**

Shards are the currency used in the store. They are commonly acquired through lootboxes or point redeems.
`!buy  <item> <amount>` -- Buy items from the shop.
`!sell <item> <amount>` -- Sell items for shards. Items sell at 1/10th the value.
`!sell <item> <amount> <value>` -- Post items for sale for the listed price. Items are taken immediately but sold when a buyer is found.

| Item       | Value |
| ---------- | ----- |
| Shovel     | 100   |
| Stick      | 100   |
| Shield     | 1000  |
| Missile    | 1000  |
| This       | 101   |
| Deez       | 1001  |
| Revives    | 1000  |
| Mine       | 250   |
| Grenade    | 500   |
| Junkbox    | 100   |
| Lootbox    | 500   |
| Goldbox    | 2500  |
| Diamondbox | 10000 |
| Scrap      | 10    |
| Filter     | 200   |

---

## Upgrade Item

**Upgrade items from your inventory**

`!upgrade <item> [amount]`

| Item      | Quantity | New Item     | Job Level |
| --------- | -------- | ------------ | --------- |
| `Junkbox` | 5        | `Lootbox`    | 1         |
| `Lootbox` | 3        | `Goldbox`    | 5         |
| `Goldbox` | 3        | `Diamondbox` | 10        |
| `This`    | 10       | `Deez`       | 10        |
| `Mine`    | 10       | `Trap`       | 5         |
| `Trap`    | 1        | `TNT`        | 15        |
| `Stick`   | 10       | `Shield`     | 10        |

---

## Disassemble Item

**Downgrade items to salvage materials**

`!disassemble <item> [amount]`

| Item         | Yield                 |
| ------------ | --------------------- |
| `Lootbox`    | 1 Junkbox + 25 Money  |
| `Goldbox`    | 1 Lootbox + 125 Money |
| `Diamondbox` | 1 Goldbox + 625 Money |
| `Deez`       | 2 This + 125 Money    |
| `Trap`       | 2 Mine + 62 Money     |
| `TNT`        | 1 Trap + 625 Money    |
| `Shield`     | 2 Stick + 125 Money   |

---

## Jobs

Gain experience by performing common actions. Experience becomes levels and levels provide passive bonuses when performing those actions in the future.

| Job        | Description                        | Bonus                                      | Special Bonus                     |
| ---------- | ---------------------------------- | ------------------------------------------ | --------------------------------- |
| Blacksmith | Masters of crafting and upgrades   | Increased crafting success rate (+10%/lvl) | Chance of a bonus upgrade         |
| Explorer   | Scouts who find extra rewards      | Increased search quality (+10%/lvl)        | Increased search yield (+10%/lvl) |
| Merchant   | Traders who get better deals       | Improved shop prices (+5%/lvl)             | Better sell values (+5%/lvl)      |
| Gambler    | High rollers who win bigger prizes | Increased gamble winnings (+5%/lvl)        | Higher win chance                 |
| Farmer     | Patient cultivators of crops       | Increased farming yield (+10%/lvl)         | Improved harvest tier (+0.2/lvl)  |
| Scholar    | Contributors to community progress | Increased contribution power (+10%/lvl)    | Job XP multiplier (+10%/lvl)      |

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

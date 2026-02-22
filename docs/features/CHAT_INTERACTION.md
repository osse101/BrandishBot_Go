# Chat Interaction & Moderation

This document details the systems that monitor and react to user chat messages, including keyword detection and timeout enforcement.

## String Finder System

The **String Finder** (`internal/user/string_finder.go`) is a utility that scans incoming chat messages for specific keywords or patterns to trigger automated responses or easter eggs.

### Mechanics

- **Pattern Matching**: Uses regex with word boundaries to find exact matches.
- **Priority**: Rules have priority levels. If multiple patterns match, the one with the highest priority wins.
- **Greedy Matching**: Sorts patterns by length (descending) to ensure longer phrases are matched before their substrings (e.g., "superman" is matched before "super").
- **Case Insensitive**: All matching is case-insensitive.

### Default Triggers

If no external configuration is provided, the system defaults to the following triggers:

| Pattern    | Code  | Priority |
| :--------- | :---- | :------- |
| `Bapanada` | `OBS` | 10       |
| `gary`     | `OBS` | 10       |
| `shedinja` | `OBS` | 10       |

### Configuration

Rules can be loaded from a JSON configuration file. Each rule consists of:

- `pattern`: The string to search for.
- `code`: An internal code used by the bot to determine the response type.
- `priority`: Integer priority level.

---

## Timeout System

The **Timeout System** (`internal/user/timeout.go`) is responsible for temporarily restricting a user's ability to chat or interact with the bot.

### Core Logic

- **In-Memory**: Timeouts are stored in memory and **will be lost if the application restarts**.
- **Accumulation**: If a user is already timed out and receives another timeout (e.g., from a second trap or weapon), the new duration is **added** to the remaining time.
  - _Example_: User has 30s remaining. Hit by Blaster (60s). New timeout = 90s.
- **Platform Agnostic**: The system is designed to support multiple platforms (Twitch, Discord, YouTube), keyed by `platform:username`.

### Interaction with Items

Timeouts are the primary effect of offensive items in the [Trap & Item System](./TRAPS.md).

- **Weapons**: Apply immediate timeouts (60s - 100 minutes).
- **Traps**: Apply a 60s timeout when the victim sends a message.
- **Revives**: Reduce the remaining timeout duration.

### Admin Controls

Admins can manage timeouts via the **Admin Dashboard** or API:

- **Clear Timeout**: Instantly removes a user's timeout.
- **View Timeout**: Check the remaining duration for a user.

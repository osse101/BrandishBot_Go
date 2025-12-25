# Feature Request: Comprehensive QA & Admin Tooling

## Summary
Implement a comprehensive suite of Admin and QA tools exposed through both Discord and Streamer.bot (via C# Wrapper). These tools will provide visibility into and control over the Progression, Economy, and Game Loop systems, addressing current testing bottlenecks.

## Reasoning & Motivation
Currently, "QA" and testing of the Progression system is extremely difficult because:
1.  **Opaque State**: Testers cannot easily see the current state of the progression tree, voting sessions, or unlock progress without direct database access.
2.  **RNG Dependency**: Testing the "Unlock" flow relies on waiting for voting sessions (24h cycles) or getting lucky with RNG selection, making it impossible to test specific feature unlock paths deterministically.
3.  **Black Box Logic**: Clients (Discord/Streamer.bot) are unaware of the internal logic, making integration testing guesswork.
4.  **Admin Unfriendly**: Existing tools are limited to basic item manipulation, leaving the core game loop (Progression) untestable.

By exposing the **already existing** backend Admin endpoints to the client layers, we can significantly accelerate testing, debugging, and content verification.

## Proposed Features

### 1. Discord Admin Command Suite
*Goal: Quick, manual verification and state inspection.*

-   **`/admin-tree-status`**:
    -   Displays the full progression tree.
    -   **Critical**: Includes "Node Key" and "Description" for each node, enabling admins to know *what* to unlock.
    -   Visual indication of Locked/Unlocked state.
-   **`/admin-unlock [node_key] [level]`**:
    -   Force-unlocks a specific node.
    -   Bypasses voting/cost requirements.
-   **`/admin-relock [node_key] [level]`**:
    -   Re-locks a node.
    -   Essential for re-testing usage flows without wiping the entire database.
-   **`/admin-instant-resolve`**:
    -   Force-ends the current voting session immediately.
    -   Selects the current winner (or random if no votes).
    -   Triggers all "Unlock" events (Notifications, etc.).
-   **`/admin-reset-tree`**:
    -   Full progression wipe (with safety confirmation).

### 2. Streamer.bot Integration (C# Wrapper)
*Goal: Automation, Integration Testing, and Streamer Control.*

Update `BrandishBotWrapper.cs` to include Admin methods corresponding to the backend endpoints:
-   `AdminUnlockNode(string nodeKey, int level)`
-   `AdminRelockNode(string nodeKey, int level)`
-   `AdminInstantUnlock()`
-   `AdminResetTree()`
-   `AdminStartVoting()`
-   `AdminEndVoting()`

**Use Cases**:
-   Stream Deck button to "Force Vote End" during a stream for pacing.
-   Automated test scripts in Streamer.bot to verify "Unlock -> Sound Effect" pipelines.

### 3. Data Visibility Enhancements
-   Ensure all "Status" endpoints return rich metadata (Display Name, Internal Description, Requirements) so usage is self-documenting.

## Impact
-   **QA**: Reduces time-to-test for progression features from "Days" (waiting for cycles) to "Seconds".
-   **Dev**: Allows verifying complex dependency chains (e.g., "If I unlock X, does Y become available?") instantly.
-   **Ops**: Provides streamers with "Panic Buttons" (Reset, Force End) to manage live events.

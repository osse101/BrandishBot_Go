# Progression System Guidance

This document provides a deep dive into the Progression System of BrandishBot. It explains how features are gated, unlocked, and modified through community engagement.

## 🌟 Core Concept

The Progression System gamifies the development and feature rollout of the bot. Instead of features being immediately available, they exist as **Nodes** in a **Progression Tree**.

1.  **Engagement**: Users perform actions (chatting, commands, crafting) to earn **Contribution Points**.
2.  **Voting**: Points accumulate into a community pool. When enough points are gathered, a **Voting Session** begins.
3.  **Unlocking**: Users vote on available nodes. The winner is unlocked, making the feature/item available to everyone.
4.  **Modifiers**: Some nodes ("Upgrades") don't unlock new features but _modify_ existing values (e.g., +10% XP).

---

## 🌳 The Progression Tree

The tree is the central data structure. It is defined in code/configuration and synced to the database.

### Source of Truth

The tree is defined in **`configs/progression_tree.json`**. This JSON file is the **absolute source of truth**.

- **Do not manually insert nodes into the database.**
- **Do not modify node properties in the database.**
- Always edit the JSON file. The system will sync changes on startup.

### Node Structure

A node consists of:

- **Key**: Unique identifier (e.g., `feature_economy`, `item_money`).
- **Type**:
  - `feature`: Unlocks functionality (e.g., Market).
  - `item`: Unlocks an item type.
  - `job`: Unlocks a job.
  - `upgrade`: Modifies a value.
- **Tier**: Non negative integer. Determines base unlock cost.
- **Size**: `small`, `medium`, `large`. Multiplier for unlock cost.
- **Prerequisites**: List of keys that must be unlocked _first_.
  - Supports **Dynamic Prerequisites**: e.g., `"-nodes_unlocked_below_tier:1:4"` (Requires 4 Tier 1 nodes).
- **ModifierConfig**: (For `upgrade` type) Defines what value changes.

### The Tree Loader (`tree_loader.go`)

The `TreeLoader` component is responsible for:

1.  **Validation**: Ensures no cycles, valid keys, valid prerequisites.
2.  **Idempotent Sync**:
    - **Inserts** new nodes.
    - **Updates** existing nodes (name, description, costs).
    - **Calculates Costs**: Uses Tier + Size to auto-calculate required engagement points.
    - **Auto-Unlock**: Nodes marked `auto_unlock: true` are instantly unlocked if not already.

### Generating Keys (`keys.go`)

The file `internal/progression/keys.go` contains Go constants for all node keys.

- **Auto-Generated**: This file is generated from the JSON config.
- **Usage**: Use these constants (e.g., `progression.FeatureEconomy`) instead of raw strings in your code.
- **Command**: Run `make generate` after modifying the JSON to update this file.

---

## 🔗 Node Interactions

### 1. Prerequisites (The Graph)

Nodes form a Directed Acyclic Graph (DAG).

- **Static Dependencies**: `A requires B`. B must be unlocked for A to become available for voting.
- **Dynamic Dependencies**: `A requires X nodes of Tier Y`. Allows flexible gating without strict paths.

### 2. Gating Features

Code checks the tree to see if a feature is allowed.

```go
// Check if "Economy" feature is unlocked
if unlocked, _ := progressionService.IsFeatureUnlocked(ctx, "feature_economy"); !unlocked {
    return errors.New("feature locked")
}
```

### 3. Modifiers (Upgrades)

Upgrades allow dynamic scaling of game values without code deployment.

**Example**: `job_xp_multiplier`

- Code uses `GetModifiedValue("job_xp_multiplier", baseValue)`.
- If `upgrade_job_xp_1` is unlocked, the system applies the modifier defined in the JSON.
- Supported logic: `multiplicative`, `additive`, `linear`.

---

## 🔄 The Engagement Loop

1.  **User Action**: User sends a message.
2.  **RecordEngagement**: Handler calls `progression.RecordEngagement`.
3.  **Weighting**: Action type ("message") is looked up in `engagement_weights`. Score is calculated.
4.  **Pool**: Score is added to the "Next Unlock" pool.
5.  **Threshold**: When Pool >= Cost the active node is unlocked. The voting session ends to set the next active node. The next voting session begins.

---

## 🛠️ Developer Workflow

### Adding a New Gated Feature

1.  **Define Node**: Add entry to `configs/progression_tree.json`.
    - Set `key` to match your feature (snake_case).
2.  **Generate Keys**: Run `make generate` to update `internal/progression/keys.go`.
3.  **Implement Feature**: Write your code, handlers, services.
4.  **Add Gate Check**: In your handler/service, add `IsFeatureUnlocked(progression.FeatureYourKey)`.
5.  **Test**: Restart app (triggers Loader sync). Use `admin-unlock` to test the unlocked state.

### Adding a Modifiable Value

1.  **Identify Value**: E.g., "Daily limit".
2.  **Instrument Code**: Replace constant with `progressionService.GetModifiedValue("daily_limit_bonus", default)`.
3.  **Define Upgrade Node**: Add `upgrade` node to JSON.
    - `modifier_config`: `{ "feature_key": "daily_limit_bonus", "modifier_type": "linear", "per_level_value": 5 }`.

---

## 📚 Reference

- **Config**: `configs/progression_tree.json`
- **Loader**: `internal/progression/tree_loader.go`
- **Service**: `internal/progression/service.go`
- **Domain**: `internal/domain/progression.go`

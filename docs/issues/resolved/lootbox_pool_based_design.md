This design moves the loot system from a dynamic filtering model to a **Static Pool Composition** model. It prioritizes mathematical stability and performance by using pre-defined item sets and a multi-stage roll process.

---

## 1. Overview

The system utilizes **Pool Sets** to decouple item definitions from lootbox logic. This ensures that adding new items or themed variants does not inadvertently shift the "No Item" probability or break the economy of existing boxes.

## 2. Data Architecture

### 2.1 Schema Definition

- **Items:** Core metadata (ID, Value, Tags).
- **Pools:** Named collections of `Item_IDs` with associated internal weights.
- **Lootbox Recipes:** Definitions that map to one or more Pools and define the `ItemDropRate` (the "Gatekeeper" roll).

### 2.2 Ingestion & Validation

Upon service start (or JSON-to-Postgres sync), the system must execute an **Orphan Tracking Script**.

- **Requirement:** Any item existing in the `items.json` that is not referenced in at least one `pool.json` must be logged as a `Warning`.
- **Strict Mode:** If an item is marked as `Active` but is orphaned, the system should fail to start to prevent "ghost content" that players can never obtain.

---

## 3. The Loot Pipeline

When a player triggers a box opening, the system follows a three-stage execution:

1. **Stage 1: The Gatekeeper (Drop Chance)**

- Perform a roll against the box’s `ItemDropRate`.
- If failed, return `FixedMoney` only.

2. **Stage 2: Pool Selection**

- If multiple pools exist (e.g., 80% Junk, 20% Fire), roll to select a single pool.

3. **Stage 3: Item Selection**

- Perform a weighted roll within the selected pool to determine the specific item.

---

## 4. Performance & Caching

To minimize Postgres overhead during high-concurrency "mass openings," the system requires a **Lootbox Final Table Cache**.

### 4.1 Caching Requirement

- **The Cache:** Store the "Flattened" version of a lootbox (all items across all its pools with their calculated absolute probabilities) in memory (e.g., Redis or a Go Map).

### 4.2 Invalidation Logic

Since the "available pool" for a player changes based on their progression, the cache must be invalidated when a new **Unlock Tier** or **Progression Milestone** is reached.

- **Handler:** The Loot Service listens for this event and purges the cached tables for that specific player, forcing a re-calculation on the next box opening.

---

## 5. Summary Table of Requirements

| Feature         | Requirement                     | Reason                                                               |
| --------------- | ------------------------------- | -------------------------------------------------------------------- |
| **Integrity**   | Orphan Tracking Script          | Prevents unreachable content and configuration debt.                 |
| **Stability**   | Gatekeeper Roll (Success Check) | Decouples "Item Quantity" from "Item Variety."                       |
| **Performance** | Final Table Caching             | Reduces DB load by avoiding redundant table flattening.              |
| **Flexibility** | Named Pool Sets                 | Allows for themed side-grades (e.g., FireJunk) without code changes. |

---

### Comparison of Roll Logic

### Next Step

Would you like me to draft the **Go Structs** for the cached table entries, or perhaps the **Postgres SQL** for the `pool_memberships` join table? I can also provide a sample of the **Orphan Tracking** logic if you're ready to implement the ingestion script.

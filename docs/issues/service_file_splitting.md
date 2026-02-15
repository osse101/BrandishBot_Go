# Service File Splitting — Packages to Refactor

> **Reference commit:** `9e6cc661` — `internal/harvest/service.go` (389 lines) was split into 6 files by domain concern.
> **Pattern:** Keep `service.go` for interface + struct + constructor + `Shutdown`. Move method implementations to domain-specific files.

## Priority Criteria

| Factor                      | Weight                                                       |
| --------------------------- | ------------------------------------------------------------ |
| File size (lines)           | Primary — larger files benefit most                          |
| Number of distinct concerns | High — clear groupings make splitting easy                   |
| Already-split siblings      | Medium — packages with some splitting done are half-finished |
| Test impact                 | Low — pure code-move refactors don't change test behavior    |

---

## 🔴 Priority 1 — High Impact (>1000 lines, clear domain splits)

### 1. `internal/progression/service.go` — 1,427 lines

**The single largest file in the codebase.** Already has some splitting (`voting_sessions.go`, `cache.go`, `unlock_cache.go`, etc.) but the main `service.go` is still enormous.

| Proposed File              | Functions to Move                                                                                                                                                                                                                                              | Lines (est.) |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `tree.go`                  | `GetProgressionTree`, `GetAvailableUnlocks`, `isNodeAvailable`, `checkStaticPrereqs`, `checkDynamicPrereqs`, `GetAvailableUnlocksWithFutureTarget`, `getNodesDependentOn`, `checkDynamicPrerequisite`, `GetNode`, `GetRequiredNodes`, `checkAllNodesUnlocked`  | ~300         |
| `unlock.go`                | `IsFeatureUnlocked`, `IsItemUnlocked`, `IsNodeUnlocked`, `AreItemsUnlocked`                                                                                                                                                                                    | ~100         |
| `engagement.go`            | `RecordEngagement`, `GetEngagementScore`, `calculateScholarBonus`, `GetUserEngagement`, `GetUserEngagementByUsername`, `GetContributionLeaderboard`, `GetEngagementVelocity`, `EstimateUnlockTime`, `getCachedWeight`, `cacheWeights`, `InvalidateWeightCache` | ~350         |
| `admin.go`                 | `AdminUnlock`, `AdminUnlockAll`, `AdminRelock`, `AdminFreezeVoting`, `AdminStartVoting`, `ResetProgressionTree`, `ForceInstantUnlock`, `CheckAndUnlockCriteria`                                                                                                | ~200         |
| `voting.go`                | `VoteForUnlock`, `resolveUserByPlatform`, `validateVotingSession`, `enrichSessionWithEstimates`, `GetActiveVotingSession`, `GetMostRecentVotingSession`                                                                                                        | ~120         |
| `modifiers.go`             | `GetModifiedValue`, `GetModifierForFeature`, `GetAllModifiersForFeature`                                                                                                                                                                                       | ~90          |
| `events.go`                | `handleNodeUnlocked`, `handleNodeRelocked`                                                                                                                                                                                                                     | ~40          |
| `status.go`                | `GetProgressionStatus`                                                                                                                                                                                                                                         | ~50          |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `Shutdown`, `InvalidateUnlockCacheForTest`                                                                                                                                                                                    | ~170         |

**Notes:**

- `voting_sessions.go` (954 lines) already exists and handles the session lifecycle — the `voting.go` proposed above contains only the service-layer voting methods that call into it. Consider whether to merge or keep them separate.
- The existing `modifiers.go` (44 lines) contains types; the proposed move adds the service methods that use those types.

---

### 2. `internal/user/service.go` — 1,349 lines

Already has some splitting (`item_handlers.go`, `linking.go`, `search_helpers.go`, `cache.go`, etc.) but `service.go` is still very large.

| Proposed File              | Functions to Move                                                                                                                                                                                                                                      | Lines (est.) |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------ |
| `inventory.go`             | `addItemToUserInternal`, `removeItemFromUserInternal`, `AddItemByUsername`, `RemoveItemByUsername`, `AddItems`, `GiveItem`, `executeGiveItemTx`, `GetInventory`, `GetInventoryByUsername`, `getInventoryInternal`, `ensureItemsInCache`, `addItemToTx` | ~450         |
| `use_item.go`              | `useItemInternal`, `UseItem`, `validateItem`, `resolveItemName`                                                                                                                                                                                        | ~120         |
| `timeout.go`               | `AddTimeout`, `ClearTimeout`, `GetTimeoutPlatform`, `ReduceTimeoutPlatform`, `TimeoutUser`, `GetTimeout`, `ReduceTimeout`, `timeoutKey`                                                                                                                | ~200         |
| `search.go`                | `HandleSearch`, `executeSearch`, `calculateSearchParameters`, `processSearchSuccess`, `processSearchFailure`                                                                                                                                           | ~150         |
| `registration.go`          | `RegisterUser`, `UpdateUser`, `FindUserByPlatformID`, `HandleIncomingMessage`, `getUserOrRegister`, `setPlatformID`, `getPlatformKeysFromUser`                                                                                                         | ~150         |
| `shield.go`                | `ApplyShield`                                                                                                                                                                                                                                          | ~20          |
| `trap.go`                  | `triggerTrap`                                                                                                                                                                                                                                          | ~50          |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `Shutdown`, `GetCacheStats`, `GetActiveChatters`, `loadCacheConfig`                                                                                                                                                   | ~200         |

**Notes:**

- `search_helpers.go` (197 lines) already exists with search loot tables — `search.go` would contain the service-layer search orchestration. Consider whether to merge them.
- `item_handlers.go` (822 lines) handles use-item dispatch and is already separate, which is good.

---

## 🟡 Priority 2 — Medium Impact (600–1000 lines, moderate complexity)

### 3. `internal/gamble/service.go` — 735 lines

Currently only has `constants.go` and `repository.go` alongside it. Moderate complexity with clear lifecycle phases.

| Proposed File              | Functions to Move                                                                                                                                                                                                   | Lines (est.) |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `start.go`                 | `StartGamble`, `validateGambleBets`, `resolveLootboxBet`, `resolveItemName`, `publishGambleStartedEvent`                                                                                                            | ~150         |
| `join.go`                  | `JoinGamble`, `executeGambleJoinTx`                                                                                                                                                                                 | ~100         |
| `execute.go`               | `ExecuteGamble`, `validateGambleExecution`, `transitionToOpeningState`, `processParticipantOutcomes`, `processParticipantLootbox`, `createParticipantSummary`, `publishGambleCompletedEvent`, `formatLootboxResult` | ~300         |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `GetGamble`, `GetActiveGamble`                                                                                                                                                     | ~150         |

---

### 4. `internal/crafting/service.go` — 729 lines

Already has `recipe_loader.go`, `constants.go`, and `events.go`. The remaining service logic splits cleanly along upgrade vs. disassemble.

| Proposed File              | Functions to Move                                                                                                                                                 | Lines (est.) |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `upgrade.go`               | `UpgradeItem`, `executeUpgradeTx`, `resolveItemName`, `validateUpgradeRequest`, `calculateUpgradeQuantity`, `calculateMasterworkBonus`, `calculateCraftsmanBonus` | ~300         |
| `disassemble.go`           | `DisassembleItem`, `validateDisassembleRequest`, `executeSingleDisassembleTx`, `executeBatchDisassembleTx`, / `executeDisassembleTx`, `calculatePerfectSalvage`   | ~200         |
| `recipes.go`               | `GetRecipe`, `GetUnlockedRecipes`, `GetAllRecipes`                                                                                                                | ~100         |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `Shutdown`                                                                                                                       | ~100         |

---

### 5. `internal/economy/service.go` — 640 lines

Only has `constants.go` alongside it. Has two distinct flows: buy and sell.

| Proposed File              | Functions to Move                                                                                                                | Lines (est.) |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `sell.go`                  | `SellItem`, `getSellEntities`, `calculateSellPrice`, `executeSellTransaction`, `finalizeSale`                                    | ~200         |
| `buy.go`                   | `BuyItem`, `getBuyEntities`, `calculatePurchaseDetails`, `finalizePurchase`                                                      | ~150         |
| `prices.go`                | `GetSellablePrices`, `GetBuyablePrices`, `loadWeeklySales`, `getCurrentWeeklySale`, `applyWeeklySaleDiscount`, `resolveItemName` | ~180         |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `Shutdown`                                                                                      | ~100         |

---

### 6. `internal/job/service.go` — 596 lines

Has `event_handler.go` (503 lines) and `constants.go`. The service itself has clear seams.

| Proposed File              | Functions to Move                                                                                                          | Lines (est.) |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `xp.go`                    | `AwardXP`, `AwardXPByPlatform`, `calculateXPAmount`, `awardXPInternal`, `CalculateLevel`, `GetXPForLevel`, `GetXPProgress` | ~200         |
| `bonuses.go`               | `GetJobBonus`, `GetJobLevel`, `getJobBonusData`                                                                            | ~80          |
| `queries.go`               | `GetUserJobs`, `GetUserJobsByPlatform`, `GetPrimaryJob`, `GetAllJobs`, `GetUserByPlatformID`                               | ~100         |
| `daily_reset.go`           | `ResetDailyJobXP`, `GetDailyResetStatus`                                                                                   | ~60          |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `Shutdown`                                                                                | ~100         |

**Note:** `event_handler.go` at 503 lines could also benefit from splitting if it handles many distinct event types.

---

## 🟢 Priority 3 — Lower Impact (400–600 lines, but still beneficial)

### 7. `internal/compost/service.go` — 457 lines

Already has `engine.go` and `constants.go`. Could split deposit vs. harvest flows.

| Proposed File              | Functions to Move                                                                                                                  | Lines (est.) |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| `deposit.go`               | `Deposit`, `checkBinCapacity`, `executeDepositTransaction`, `resolveDepositItems`, `updateBinWithDeposits`, `checkDepositPossible` | ~180         |
| `harvest.go`               | `Harvest`, `idleHarvestResult`, `resolveLazyBinStatus`, `compostingHarvestResult`, `processHarvestItems`, `awardHarvestXP`         | ~170         |
| `user.go`                  | `validateFeature`, `getUserAndBin`                                                                                                 | ~30          |
| `service.go` _(remaining)_ | Interface, struct, `NewService`, `Shutdown`, `formatDuration`                                                                      | ~80          |

---

### 8. `internal/expedition/service.go` — 427 lines

Already well-split (`engine.go`, `encounters.go`, `journal.go`, `skills.go`, `config.go`). Could still benefit from pulling out orchestration steps.

| Proposed File              | Functions to Move                                       | Lines (est.) |
| -------------------------- | ------------------------------------------------------- | ------------ |
| `start.go`                 | Functions related to starting an expedition             | ~100         |
| `resolve.go`               | Functions related to resolving/completing an expedition | ~150         |
| `service.go` _(remaining)_ | Interface, struct, `NewService`                         | ~170         |

---

## ⚪ Not Recommended for Splitting

| Package            | File                          | Lines                                                                                               | Reason |
| ------------------ | ----------------------------- | --------------------------------------------------------------------------------------------------- | ------ |
| `internal/discord` | `client.go` (1,395 lines)     | Already well-organized with `cmd_*.go` splits; `client.go` is mostly boilerplate setup/registration |
| `internal/discord` | `sse_handlers.go` (606 lines) | Each handler is a ~20 line method; structural, not complex                                          |
| `internal/handler` | `progression.go` (759 lines)  | HTTP handlers are shallow; splitting buys little navigability                                       |
| `internal/handler` | `inventory.go` (573 lines)    | Same — shallow handler methods                                                                      |
| `internal/stats`   | `service.go` (366 lines)      | Manageable size                                                                                     |
| `internal/slots`   | `service.go` (356 lines)      | Manageable size                                                                                     |

---

## Implementation Guidelines

1. **Pure code moves only** — No logic changes, no renames, no refactors. Each PR should compile and pass tests identically.
2. **Verify with:** `go build ./internal/<pkg>/...` and `go test ./internal/<pkg>/...`
3. **One package per PR** to keep diffs reviewable.
4. **Keep `service.go` as the anchor** — Always retains: interface definition, struct definition, constructor (`NewService`), `Shutdown`, compile-time interface check.
5. **File naming convention** — Use domain nouns (`inventory.go`, `engagement.go`) not verbs (`handling.go`) or generic names (`helpers.go`).
6. **Don't split test files** — Test files can stay as-is since Go doesn't require test files to mirror source files.

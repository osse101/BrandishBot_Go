# Upgrade Nodes Implementation Status

This document tracks the minimal support implementation for progression upgrade nodes.

## ✅ Completed - Minimal Support Added

### Tier 1

#### upgrade_progression_basic
- **Status**: ✅ Minimal support added
- **Location**: `internal/progression/service.go:536-552`
- **Implementation**: GetModifiedValue call added to RecordEngagement
- **TODO Comments**: Added for testing (lines 541-543)
- **Tests**: Stub created at `internal/progression/upgrades_test.go`

### Tier 2

#### upgrade_gamble_win_bonus
- **Status**: ✅ Already implemented
- **Location**: `internal/gamble/service.go:418`
- **Implementation**: GetModifiedValue already in use
- **Tests**: Stub created at `internal/gamble/upgrades_test.go` for verification

#### upgrade_crafting_1
- **Status**: ✅ Minimal support added
- **Location**: `internal/crafting/service.go:320-335, 722-740`
- **Implementation**:
  - GetModifiedValue call added to calculateUpgradeOutput (masterwork)
  - GetModifiedValue call added to calculatePerfectSalvage
  - ProgressionService interface added but NOT wired (nil)
- **Wiring TODO**: Service constructor needs progression service parameter
- **Tests**: Stub created at `internal/crafting/upgrades_test.go`

#### upgrade_economy_1
- **Status**: ⚠️ Partial - TODO comments only
- **Location**: `internal/economy/service.go:203-206`
- **Implementation**: TODO comments added, refactoring needed
- **Blocker**: calculateSellPrice doesn't have context parameter
- **Next Steps**:
  1. Create calculateSellPriceWithModifier(ctx, service, baseValue)
  2. Update callers to pass context
- **Tests**: Stub created at `internal/economy/upgrades_test.go`

#### upgrade_exploration_1
- **Status**: ⏸️ Blocked - Needs search/exploration service
- **Location**: N/A - Service doesn't exist
- **Implementation**: Skipped for now
- **Next Steps**: Implement search/exploration service first

#### upgrade_farming_1
- **Status**: ⏸️ Blocked - Needs farming service
- **Location**: N/A - Service doesn't exist
- **Implementation**: Skipped for now
- **Next Steps**: Implement farming service first

### Tier 3

#### upgrade_progression_two
- **Status**: ✅ Minimal support added (stacking)
- **Location**: Same as upgrade_progression_basic
- **Implementation**: Uses same GetModifiedValue call, stacks multiplicatively
- **TODO Comments**: Stacking test needed
- **Tests**: Stub created at `internal/progression/upgrades_test.go`

#### upgrade_job_xp_multiplier
- **Status**: ✅ Already implemented
- **Location**: `internal/job/service.go:512`
- **Implementation**: GetModifiedValue already in use
- **Tests**: Stub created at `internal/job/upgrades_test.go` for verification

#### upgrade_job_level_cap
- **Status**: ✅ Minimal support added
- **Location**: `internal/job/service.go:532-543`
- **Implementation**: GetModifiedValue call added to getMaxJobLevel
- **TODO Comments**: Added for linear modifier testing
- **Tests**: Stub created at `internal/job/upgrades_test.go`

### Tier 4

#### upgrade_progression_three
- **Status**: ✅ Minimal support added (triple stacking)
- **Location**: Same as upgrade_progression_basic
- **Implementation**: Uses same GetModifiedValue call, stacks multiplicatively
- **TODO Comments**: Triple stacking test needed
- **Tests**: Stub created at `internal/progression/upgrades_test.go`

---

## 📝 Summary

### By Status
- ✅ **Already Implemented**: 2 (upgrade_gamble_win_bonus, upgrade_job_xp_multiplier)
- ✅ **Minimal Support Added**: 5 (progression_basic, progression_two, progression_three, crafting_1, job_level_cap)
- ⚠️ **Partial (TODO only)**: 1 (economy_1 - needs refactoring)
- ⏸️ **Blocked**: 2 (exploration_1, farming_1 - need services)

### Total: 10 Upgrade Nodes
- **7 / 10** have minimal support or better
- **3 / 10** require additional work (1 refactoring, 2 new services)

---

## 🔧 Remaining Work

### High Priority
1. **upgrade_economy_1**: Refactor calculateSellPrice to accept context
   - Create calculateSellPriceWithModifier
   - Update all callers to use context-aware version
   - Wire progression service dependency

2. **upgrade_crafting_1**: Wire ProgressionService in NewService
   - Update `cmd/app/main.go` to pass progression service
   - Update constructor signature
   - Update all test files

### Medium Priority
3. **All Upgrades**: Write actual tests (currently stubs with t.Skip)
   - Test modifier application at each level
   - Test stacking behavior (progression upgrades)
   - Test fallback when progression service unavailable
   - Integration tests with real services

### Low Priority (Future)
4. **upgrade_exploration_1**: Implement search/exploration service
5. **upgrade_farming_1**: Implement farming service

---

## 🧪 Test File Locations

All test stubs created with TODO comments and t.Skip():

- `internal/progression/upgrades_test.go` - Progression rate upgrades (basic, two, three)
- `internal/crafting/upgrades_test.go` - Crafting success rate
- `internal/economy/upgrades_test.go` - Economy bonus
- `internal/gamble/upgrades_test.go` - Gamble win bonus
- `internal/job/upgrades_test.go` - Job XP and level cap

---

## 🎯 Next Steps for Full Implementation

1. **Wire crafting progression service** (1-2 hours)
   - Update NewService constructor
   - Update main.go initialization
   - Update existing tests

2. **Refactor economy for context** (2-3 hours)
   - Create context-aware price calculation
   - Update callers
   - Test modifier application

3. **Write actual tests** (1 day per service)
   - Remove t.Skip() from test stubs
   - Implement test logic per TODO comments
   - Verify modifier application
   - Test stacking (progression upgrades)

4. **Manual testing** (1-2 hours)
   - Admin unlock upgrades to various levels
   - Verify modifiers apply in-game
   - Test with real users

---

## 📋 Checklist for "Full" Implementation

- [ ] Wire crafting ProgressionService dependency
- [ ] Refactor economy calculateSellPrice for context
- [ ] Implement progression rate tests (basic, two, three)
- [ ] Implement crafting success rate tests
- [ ] Implement economy bonus tests
- [ ] Verify gamble win bonus tests
- [ ] Verify job XP multiplier tests
- [ ] Implement job level cap tests
- [ ] Manual testing of all upgrades
- [ ] Document any design decisions (buy price behavior, etc.)

---

## ✨ What This Minimal Implementation Provides

1. **Code structure** in place for all 7 implementable upgrades
2. **GetModifiedValue calls** at correct locations with fallbacks
3. **TODO comments** marking exact implementation points
4. **Test stubs** documenting what needs testing
5. **No breaking changes** - all services still work without modifiers
6. **Clear path forward** for completing implementation

The codebase is now ready for:
- Admin testing by unlocking upgrades and observing effects
- Incremental completion of each upgrade's tests
- Easy location of implementation points via TODO(upgrade_*) comments

# Quick Grep Guide for Upgrade TODOs

Find all upgrade-related TODOs quickly:

## Find All Upgrade TODOs

```bash
# All upgrade TODOs across the codebase
grep -rn "TODO(upgrade_" internal/

# By specific upgrade
grep -rn "TODO(upgrade_progression_basic)" internal/
grep -rn "TODO(upgrade_progression_two)" internal/
grep -rn "TODO(upgrade_progression_three)" internal/
grep -rn "TODO(upgrade_gamble_win_bonus)" internal/
grep -rn "TODO(upgrade_crafting_1)" internal/
grep -rn "TODO(upgrade_economy_1)" internal/
grep -rn "TODO(upgrade_job_xp_multiplier)" internal/
grep -rn "TODO(upgrade_job_level_cap)" internal/
```

## Find Modifier Application Points

```bash
# All GetModifiedValue calls
grep -rn "GetModifiedValue" internal/

# Specific feature keys
grep -rn "progression_rate" internal/
grep -rn "gamble_win_bonus" internal/
grep -rn "crafting_success_rate" internal/
grep -rn "economy_bonus" internal/
grep -rn "job_xp_multiplier" internal/
grep -rn "job_level_cap" internal/
```

## Find Test Stubs

```bash
# All upgrade test files
find internal/ -name "upgrades_test.go"

# Skipped tests (stubs)
grep -rn "t.Skip.*TODO" internal/

# By service
grep -n "TODO" internal/progression/upgrades_test.go
grep -n "TODO" internal/crafting/upgrades_test.go
grep -n "TODO" internal/economy/upgrades_test.go
grep -n "TODO" internal/gamble/upgrades_test.go
grep -n "TODO" internal/job/upgrades_test.go
```

## Implementation Status

```bash
# Count TODOs by upgrade node
grep -r "TODO(upgrade_progression_basic)" internal/ | wc -l
grep -r "TODO(upgrade_crafting_1)" internal/ | wc -l
grep -r "TODO(upgrade_economy_1)" internal/ | wc -l
# ... etc
```

## Quick Reference: File Locations

### Implementation Files
- `internal/progression/service.go:536-552` - Progression rate modifier
- `internal/crafting/service.go:320-335, 722-740` - Crafting success modifiers
- `internal/economy/service.go:203-206` - Economy bonus (TODO only)
- `internal/job/service.go:512, 532-543` - Job XP and level cap
- `internal/gamble/service.go:418` - Gamble win bonus (already implemented)

### Test Stub Files
- `internal/progression/upgrades_test.go` - 5 test stubs
- `internal/crafting/upgrades_test.go` - 4 test stubs
- `internal/economy/upgrades_test.go` - 5 test stubs
- `internal/job/upgrades_test.go` - 4 test stubs
- `internal/gamble/upgrades_test.go` - 6 test stubs

### Documentation
- `docs/issues/progression_nodes/upgrades.md` - Full requirements
- `docs/issues/progression_nodes/upgrades_implementation_status.md` - Status tracking

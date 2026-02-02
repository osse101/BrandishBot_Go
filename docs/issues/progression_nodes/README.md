# Progression Node Implementation Tasks

This directory contains implementation tasks for the 36 unimplemented progression nodes in BrandishBot.

## Overview

**Total Nodes**: 46
**Implemented (with feature gates)**: 10 (22%)
**Pending Implementation**: 36 (78%)

### Node Distribution by Type

| Type | Total | Implemented | Pending |
|------|-------|-------------|---------|
| Features | 11 | 8 | 3 |
| Items | 19 | 2 | 17 |
| Jobs | 6 | 0 | 6 |
| Upgrades | 10 | 0 | 10 |

### Files in This Directory

- **items.md** - 17 item unlock implementation tasks
- **jobs.md** - 6 job unlock implementation tasks
- **upgrades.md** - 10 upgrade modifier implementation tasks
- **features.md** - 3 feature gate implementation tasks

## How to Use These Task Files

Each task file contains a checklist of nodes to implement, organized by type. Every node follows this format:

```markdown
## [Node Name] (`node_key`)

**Type**: [item|job|upgrade|feature] | **Tier**: X | **Size**: [small|medium|large]

**Prerequisites**: [comma-separated list]

**Implementation Checklist**:
- [ ] Add feature gate check in [handler/service file]
- [ ] Update [relevant service method] to check unlock status
- [ ] Add tests in [test file]
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "...", "level": 1}'`
- [ ] Test locked behavior (should return 403)
- [ ] Test unlocked behavior (should work normally)

**Files to Modify**:
- `[primary file]` - [what to change]
- `[test file]` - [what to test]

**Acceptance Criteria**:
- ✓ Locked state returns 403 with clear error message
- ✓ Unlocked state allows full functionality
- ✓ Tests cover both locked and unlocked cases
```

## General Implementation Patterns

### For Items

Items need unlock checks in:
- **Inventory handlers** - `internal/handler/user.go`
- **Item service** - `internal/item/service.go`
- **Economy handlers** - `internal/handler/prices.go` (for buyable items)

**Pattern**:
```go
// Check if item is unlocked
unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.ItemXXX)
if err != nil {
    return nil, fmt.Errorf("failed to check unlock status: %w", err)
}
if !unlocked {
    return nil, ErrItemLocked
}
```

### For Jobs

Jobs need unlock checks in:
- **Job service** - `internal/job/service.go`
- **Job handlers** - `internal/handler/job.go`

**Pattern**:
```go
// In job activation/XP award methods
unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.JobXXX)
if err != nil {
    return fmt.Errorf("failed to check job unlock: %w", err)
}
if !unlocked {
    return ErrJobLocked
}
```

### For Upgrades

Upgrades need modifier application in relevant services. The modifier system is already implemented via `GetModifiedValue()`:

**Pattern**:
```go
// Apply modifier to base value
modifiedValue := s.progressionService.GetModifiedValue(ctx, "feature_key", baseValue)
```

**Implementation steps**:
1. Identify where the base value is used
2. Wrap it with `GetModifiedValue()`
3. Add tests for modified vs unmodified values

### For Features

Features need feature gate checks in handlers:

**Pattern**:
```go
// Check feature unlock
unlocked, err := s.progressionService.IsNodeUnlocked(ctx, progression.FeatureXXX)
if err != nil {
    return nil, fmt.Errorf("failed to check feature unlock: %w", err)
}
if !unlocked {
    return nil, ErrFeatureNotUnlocked
}
```

## Testing Guidelines

### Unit Tests

For each implemented node, add tests covering:

1. **Locked state** - Should return 403/error
2. **Unlocked state** - Should work normally
3. **Modifier effects** (for upgrades) - Should apply correctly

**Example test structure**:
```go
func TestItemXXX_Locked(t *testing.T) {
    // Setup with item locked
    // Attempt to use item
    // Assert error returned
}

func TestItemXXX_Unlocked(t *testing.T) {
    // Setup with item unlocked
    // Attempt to use item
    // Assert success
}
```

### Manual Testing

Use the progression admin endpoints to test:

```bash
# Unlock a node
curl -X POST http://localhost:8080/api/v1/progression/admin/unlock \
  -H "Content-Type: application/json" \
  -d '{"node_key": "item_shovel", "level": 1}'

# Check unlock status
curl http://localhost:8080/api/v1/progression/tree
```

## Implementation Priority

Recommended order:

1. **Features** (3 tasks) - Foundation for other systems (duels, compost, expeditions)
2. **Jobs** (6 tasks) - Enable XP progression
3. **Items** (17 tasks) - Core gameplay items
4. **Upgrades** (10 tasks) - Enhancement layer

## Related Documentation

- **Progression System**: `docs/PROGRESSION_API.md`
- **Architecture**: `docs/ARCHITECTURE.md`
- **Development Guide**: `docs/development/FEATURE_DEVELOPMENT_GUIDE.md`
- **CLAUDE.md**: Project-wide context and patterns

## Current Implementation Status

### Implemented (10 nodes with feature gates)

**Features** (8):
1. `progression_system` (auto-unlocked)
2. `feature_economy`
3. `feature_search`
4. `feature_upgrade`
5. `feature_disassemble`
6. `feature_gamble`
7. `feature_farming`
8. `feature_events`

**Items** (2):
9. `item_money`
10. `item_lootbox0`

### Pending Implementation (36 nodes)

See individual task files for complete breakdown.

## Notes

- All nodes in `configs/progression_tree.json` are already configured with costs, tiers, and prerequisites
- The progression voting/unlock system is fully operational
- Node keys are auto-generated in `internal/progression/keys.go`
- Each task is independent and can be implemented in any order (respecting dependencies)
- Estimated effort: 1-4 hours per node depending on complexity

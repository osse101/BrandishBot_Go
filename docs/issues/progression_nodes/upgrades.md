# Upgrade Implementation Tasks

10 upgrade nodes requiring modifier application implementation.

All upgrades use the modifier system via `GetModifiedValue()` in the progression service.

---

## Tier 1 Upgrades

### Progression Upgrade Basic (`upgrade_progression_basic`)

**Type**: upgrade | **Tier**: 1 | **Size**: small | **Max Level**: 5

**Prerequisites**: progression_system, -total_nodes_unlocked:5

**Modifier Config**:
- **feature_key**: `progression_rate`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Identify where progression rate is calculated
- [ ] Wrap base progression rate with `GetModifiedValue(ctx, "progression_rate", baseRate)`
- [ ] Add tests for modified progression rate at levels 1-5
- [ ] Verify modifier stacks with upgrade_progression_two and upgrade_progression_three
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedRate = baseRate * (1.0 + 0.1 * level)`

**Files to Modify**:
- `internal/progression/service.go` - Apply modifier to engagement contribution calculations
- `internal/progression/service_test.go` - Add modifier tests

**Acceptance Criteria**:
- ✓ Level 1: 10% faster progression (1.1x multiplier)
- ✓ Level 5: 50% faster progression (1.5x multiplier)
- ✓ Modifier applies to engagement point contributions
- ✓ Tests verify modifier at each level

**Effect**: Community unlocks nodes faster as they level up this upgrade.

---

## Tier 2 Upgrades

### Gambling Bonus (`upgrade_gamble_win_bonus`)

**Type**: upgrade | **Tier**: 2 | **Size**: small | **Max Level**: 5

**Prerequisites**: feature_gamble

**Modifier Config**:
- **feature_key**: `gamble_win_bonus`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.05 (5% per level)

**Implementation Checklist**:
- [ ] Find gamble winnings calculation in `internal/gamble/service.go`
- [ ] Wrap winnings with `GetModifiedValue(ctx, "gamble_win_bonus", baseWinnings)`
- [ ] Add tests for modified winnings at levels 1-5
- [ ] Verify modifier applies to all gamble types
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedWinnings = baseWinnings * (1.0 + 0.05 * level)`

**Files to Modify**:
- `internal/gamble/service.go` - Apply modifier in ExecuteGamble when calculating winnings
- `internal/gamble/service_test.go` - Add modifier tests

**Acceptance Criteria**:
- ✓ Level 1: 5% bonus on winnings (1.05x multiplier)
- ✓ Level 5: 25% bonus on winnings (1.25x multiplier)
- ✓ Modifier applies to all gamble winnings
- ✓ Tests verify modifier at each level

**Effect**: Gamble winners receive more items/currency as this upgrade levels up.

---

### Exploration Upgrade (`upgrade_exploration_1`)

**Type**: upgrade | **Tier**: 2 | **Size**: small | **Max Level**: 5

**Prerequisites**: item_shovel

**Modifier Config**:
- **feature_key**: `search_quality`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Identify search/exploration quality calculation (TBD - search service)
- [ ] Wrap search quality with `GetModifiedValue(ctx, "search_quality", baseQuality)`
- [ ] Add tests for improved search results at levels 1-5
- [ ] Verify modifier improves item find rate or quality
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedQuality = baseQuality * (1.0 + 0.1 * level)`

**Files to Modify**:
- TBD - Search/exploration service (needs implementation)
- TBD - Search service tests

**Acceptance Criteria**:
- ✓ Level 1: 10% better search quality (1.1x multiplier)
- ✓ Level 5: 50% better search quality (1.5x multiplier)
- ✓ Modifier improves item find rate or rarity
- ✓ Tests verify modifier at each level

**Effect**: Search and expedition activities yield better results.

**Note**: Requires search/exploration service implementation first.

---

### Crafting Upgrade (`upgrade_crafting_1`)

**Type**: upgrade | **Tier**: 2 | **Size**: small | **Max Level**: 5

**Prerequisites**: item_scrap

**Modifier Config**:
- **feature_key**: `crafting_success_rate`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Find crafting success rate in `internal/crafting/service.go`
- [ ] Wrap success rate with `GetModifiedValue(ctx, "crafting_success_rate", baseRate)`
- [ ] Add tests for improved success rate at levels 1-5
- [ ] Verify modifier applies to both upgrade and disassemble
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedRate = baseRate * (1.0 + 0.1 * level)`

**Files to Modify**:
- `internal/crafting/service.go` - Apply modifier to masterwork chance and success rates
- `internal/crafting/service_test.go` - Add modifier tests

**Acceptance Criteria**:
- ✓ Level 1: 10% better crafting success (1.1x multiplier)
- ✓ Level 5: 50% better crafting success (1.5x multiplier)
- ✓ Modifier applies to masterwork chance and salvage rate
- ✓ Tests verify modifier at each level

**Effect**: Higher chance of masterwork upgrades and perfect salvages.

---

### Farming Upgrade (`upgrade_farming_1`)

**Type**: upgrade | **Tier**: 2 | **Size**: small | **Max Level**: 5

**Prerequisites**: item_stick

**Modifier Config**:
- **feature_key**: `farming_yield`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Identify farming yield calculation (TBD - farming service)
- [ ] Wrap yield with `GetModifiedValue(ctx, "farming_yield", baseYield)`
- [ ] Add tests for improved farming yield at levels 1-5
- [ ] Verify modifier applies to all farming activities
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedYield = baseYield * (1.0 + 0.1 * level)`

**Files to Modify**:
- TBD - Farming service (needs implementation)
- TBD - Farming service tests

**Acceptance Criteria**:
- ✓ Level 1: 10% better farming yield (1.1x multiplier)
- ✓ Level 5: 50% better farming yield (1.5x multiplier)
- ✓ Modifier increases crop/resource output
- ✓ Tests verify modifier at each level

**Effect**: Farming activities produce more resources.

**Note**: Requires farming service implementation first.

---

### Economy Upgrade (`upgrade_economy_1`)

**Type**: upgrade | **Tier**: 2 | **Size**: small | **Max Level**: 5

**Prerequisites**: item_script

**Modifier Config**:
- **feature_key**: `economy_bonus`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.05 (5% per level)

**Implementation Checklist**:
- [ ] Find economy transaction values in `internal/economy/service.go`
- [ ] Wrap buy/sell prices with `GetModifiedValue(ctx, "economy_bonus", basePrice)`
- [ ] Add tests for better prices at levels 1-5
- [ ] Verify modifier applies to both buying and selling
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedBonus = baseBonus * (1.0 + 0.05 * level)`

**Files to Modify**:
- `internal/economy/service.go` - Apply modifier to buy/sell calculations
- `internal/economy/service_test.go` - Add modifier tests

**Acceptance Criteria**:
- ✓ Level 1: 5% economy bonus (1.05x multiplier)
- ✓ Level 5: 25% economy bonus (1.25x multiplier)
- ✓ Modifier improves buy prices or sell prices
- ✓ Tests verify modifier at each level

**Effect**: Better prices when buying/selling items.

---

## Tier 3 Upgrades

### Job XP Boost (`upgrade_job_xp_multiplier`)

**Type**: upgrade | **Tier**: 3 | **Size**: small | **Max Level**: 5

**Prerequisites**: job_scholar, -nodes_unlocked_below_tier:2:15

**Modifier Config**:
- **feature_key**: `job_xp_multiplier`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Find job XP award in `internal/job/service.go`
- [ ] Wrap XP amount with `GetModifiedValue(ctx, "job_xp_multiplier", baseXP)`
- [ ] Add tests for increased XP at levels 1-5
- [ ] Verify modifier applies to all jobs
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedXP = baseXP * (1.0 + 0.1 * level)`

**Files to Modify**:
- `internal/job/service.go` - Apply modifier in AwardXP
- `internal/job/service_test.go` - Add modifier tests

**Acceptance Criteria**:
- ✓ Level 1: 10% more XP (1.1x multiplier)
- ✓ Level 5: 50% more XP (1.5x multiplier)
- ✓ Modifier applies to all job XP gains
- ✓ Tests verify modifier at each level

**Effect**: All jobs level up faster.

**Note**: Already implemented - verify it's working correctly.

---

### Progression Upgrade 2 (`upgrade_progression_two`)

**Type**: upgrade | **Tier**: 3 | **Size**: small | **Max Level**: 5

**Prerequisites**: job_scholar

**Modifier Config**:
- **feature_key**: `progression_rate`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Verify modifier stacks with upgrade_progression_basic
- [ ] Test combined effect at different levels
- [ ] Verify formula: `totalMultiplier = (1.0 + 0.1 * level1) * (1.0 + 0.1 * level2)`
- [ ] Add tests for stacking modifiers
- [ ] Test with both upgrades at max level

**Files to Modify**:
- `internal/progression/service.go` - Already implemented, verify stacking
- `internal/progression/service_test.go` - Add stacking tests

**Acceptance Criteria**:
- ✓ Stacks multiplicatively with upgrade_progression_basic
- ✓ Level 5 + Level 5 basic = 2.25x total multiplier (1.5 * 1.5)
- ✓ Tests verify stacking at various levels

**Effect**: Further accelerates progression rate.

---

### Job Level Boost (`upgrade_job_level_cap`)

**Type**: upgrade | **Tier**: 3 | **Size**: large | **Max Level**: 3

**Prerequisites**: job_scholar

**Modifier Config**:
- **feature_key**: `job_level_cap`
- **modifier_type**: `linear`
- **base_value**: 0
- **per_level_value**: 10 (additive)

**Implementation Checklist**:
- [ ] Find job level cap in `internal/job/service.go`
- [ ] Wrap cap with `GetModifiedValue(ctx, "job_level_cap", baseCapmin/max)`
- [ ] Add tests for increased cap at levels 1-3
- [ ] Verify both min and max level caps increase
- [ ] Test with admin unlock at different levels
- [ ] Verify formula: `modifiedCap = baseCap + (10 * level)`

**Files to Modify**:
- `internal/job/service.go` - Apply modifier to level cap checks
- `internal/job/service_test.go` - Add modifier tests

**Acceptance Criteria**:
- ✓ Level 1: +10 to min/max job levels
- ✓ Level 3: +30 to min/max job levels
- ✓ Modifier applies to all jobs
- ✓ Tests verify modifier at each level

**Effect**: Jobs can reach higher levels.

---

## Tier 4 Upgrades

### Progression Upgrade 3 (`upgrade_progression_three`)

**Type**: upgrade | **Tier**: 4 | **Size**: small | **Max Level**: 5

**Prerequisites**: upgrade_progression_two

**Modifier Config**:
- **feature_key**: `progression_rate`
- **modifier_type**: `multiplicative`
- **base_value**: 1.0
- **per_level_value**: 0.1 (10% per level)

**Implementation Checklist**:
- [ ] Verify modifier stacks with upgrade_progression_basic and upgrade_progression_two
- [ ] Test triple stacking at different levels
- [ ] Verify formula: `total = (1.0 + 0.1*L1) * (1.0 + 0.1*L2) * (1.0 + 0.1*L3)`
- [ ] Add tests for triple stacking
- [ ] Test with all three upgrades at max level

**Files to Modify**:
- `internal/progression/service.go` - Already implemented, verify triple stacking
- `internal/progression/service_test.go` - Add triple stacking tests

**Acceptance Criteria**:
- ✓ Stacks with both previous progression upgrades
- ✓ Level 5 + Level 5 tier 2 + Level 5 tier 1 = 3.375x total (1.5 * 1.5 * 1.5)
- ✓ Tests verify triple stacking

**Effect**: Maximum progression rate acceleration for endgame.

---

## Implementation Pattern

All upgrades use the same modifier application pattern:

```go
// In relevant service
func (s *Service) CalculateValue(ctx context.Context, baseValue float64) float64 {
    // Apply modifier
    modifiedValue := s.progressionService.GetModifiedValue(ctx, "feature_key", baseValue)
    return modifiedValue
}
```

### For Multiplicative Modifiers

```go
// GetModifiedValue internally calculates:
// return baseValue * (modifierConfig.BaseValue + modifierConfig.PerLevelValue * currentLevel)
```

### For Linear Modifiers

```go
// GetModifiedValue internally calculates:
// return baseValue + (modifierConfig.PerLevelValue * currentLevel)
```

## Testing Pattern

```go
func TestUpgradeXXX_ModifierApplication(t *testing.T) {
    // Setup with upgrade at level 1
    baseValue := 100.0
    expectedValue := baseValue * 1.1 // 10% boost

    result := service.CalculateValue(ctx, baseValue)
    assert.Equal(t, expectedValue, result)
}

func TestUpgradeXXX_ModifierStacking(t *testing.T) {
    // Setup with multiple related upgrades
    // Verify they stack correctly (multiplicative or additive)
}
```

## Modifier Stacking Rules

### Progression Rate Upgrades
All three progression upgrades stack **multiplicatively**:
- Basic (Tier 1): 1.0 + 0.1*L1
- Tier 2: 1.0 + 0.1*L2
- Tier 3: 1.0 + 0.1*L3
- **Total**: (1.0 + 0.1*L1) × (1.0 + 0.1*L2) × (1.0 + 0.1*L3)

**Max Effect**: 1.5 × 1.5 × 1.5 = **3.375x** progression rate

### Other Modifiers
Each modifier affects its own feature key independently - no cross-stacking.

## Priority

Recommended implementation order:
1. **upgrade_progression_basic** - Core mechanic
2. **upgrade_job_xp_multiplier** - Already implemented, verify
3. **upgrade_gamble_win_bonus** - Gamble service exists
4. **upgrade_crafting_1** - Crafting service exists
5. **upgrade_economy_1** - Economy service exists
6. **upgrade_progression_two** - Test stacking with basic
7. **upgrade_job_level_cap** - Linear modifier example
8. **upgrade_progression_three** - Test triple stacking
9. **upgrade_exploration_1** - Needs search service
10. **upgrade_farming_1** - Needs farming service

## Notes

- Modifier system is already implemented in `internal/progression/service.go`
- Use `GetModifiedValue(ctx, featureKey, baseValue)` to apply modifiers
- Modifiers are cached for 30 minutes for performance
- Multiple modifiers with same feature_key stack multiplicatively
- Test at various levels, not just max level

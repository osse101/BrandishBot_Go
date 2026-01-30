# Item Implementation Tasks

19 item nodes requiring unlock gate implementation.

## Tier 1 Items

### Missile (`weapon_missile`)

**Type**: item | **Tier**: 1 | **Size**: medium

**Prerequisites**: item_money

**Implementation Checklist**:
- [ ] Add feature gate check in weapon/targeting system
- [ ] Update item use handler to check unlock status
- [ ] Add tests for locked/unlocked states
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "weapon_missile", "level": 1}'`
- [ ] Test locked behavior (should return 403)
- [ ] Test unlocked behavior (weapon works normally)

**Files to Modify**:
- `internal/handler/user.go` - Add unlock check in item use handler
- `internal/user/service.go` - Check unlock before allowing missile use
- `internal/user/service_test.go` - Add locked/unlocked tests

**Acceptance Criteria**:
- ✓ Locked state returns 403 with error "Missile not unlocked"
- ✓ Unlocked state allows missile usage
- ✓ Tests cover both states

---

### Video Filter (`item_video_filter`)

**Type**: item | **Tier**: 1 | **Size**: medium

**Prerequisites**: item_money, -total_nodes_unlocked:5

**Implementation Checklist**:
- [ ] Add feature gate check in item handler
- [ ] Update item service to check unlock status
- [ ] Add tests for locked/unlocked states
- [ ] Verify with admin unlock
- [ ] Test locked behavior (should return 403)
- [ ] Test unlocked behavior (filter works)

**Files to Modify**:
- `internal/handler/user.go` - Add unlock check
- `internal/item/service.go` - Verify unlock before use
- `internal/item/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Locked state returns 403 error
- ✓ Unlocked state allows video filter application
- ✓ Tests verify both locked and unlocked states

---

### Decent Lootbox (`item_lootbox1`)

**Type**: item | **Tier**: 1 | **Size**: medium

**Prerequisites**: item_lootbox0

**Implementation Checklist**:
- [ ] Add unlock check in lootbox opening handler
- [ ] Update lootbox service to verify unlock
- [ ] Add tests for locked/unlocked lootbox1
- [ ] Verify with admin unlock
- [ ] Test locked behavior (should fail with 403)
- [ ] Test unlocked behavior (can open lootbox1)

**Files to Modify**:
- `internal/lootbox/service.go` - Add unlock verification in OpenLootbox
- `internal/lootbox/service_test.go` - Add locked/unlocked tests
- `internal/handler/gamble.go` - Verify unlock before opening

**Acceptance Criteria**:
- ✓ Cannot open lootbox1 when locked
- ✓ Can open lootbox1 when unlocked
- ✓ Error message clearly indicates unlock required

---

### Stick (`item_stick`)

**Type**: item | **Tier**: 1 | **Size**: medium

**Prerequisites**: feature_farming

**Implementation Checklist**:
- [ ] Add unlock check in inventory handler
- [ ] Update item service to verify unlock
- [ ] Add tests for acquiring/using stick when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior
- [ ] Test unlocked behavior

**Files to Modify**:
- `internal/handler/user.go` - Add unlock check for stick acquisition
- `internal/item/service.go` - Verify unlock
- `internal/item/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Cannot acquire stick when locked
- ✓ Can acquire/use stick when unlocked
- ✓ Tests cover both states

---

## Tier 2 Items

### Shovel (`item_shovel`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_search

**Implementation Checklist**:
- [ ] Add unlock check in search/exploration handler
- [ ] Update search service to verify shovel unlock
- [ ] Add tests for shovel-dependent search features
- [ ] Verify with admin unlock
- [ ] Test locked behavior (search limited without shovel)
- [ ] Test unlocked behavior (full search capability)

**Files to Modify**:
- `internal/handler/user.go` - Add shovel unlock check
- `internal/item/service.go` - Verify unlock before use
- `internal/item/service_test.go` - Add locked/unlocked tests

**Acceptance Criteria**:
- ✓ Search features limited when shovel locked
- ✓ Full search capability when shovel unlocked
- ✓ Clear error messaging

---

### Grenade (`item_grenade`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: weapon_missile

**Implementation Checklist**:
- [ ] Add unlock check in weapon handler
- [ ] Update item use to verify grenade unlock
- [ ] Add tests for grenade usage when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot use grenade)
- [ ] Test unlocked behavior (random timeout works)

**Files to Modify**:
- `internal/handler/user.go` - Add grenade unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Cannot use grenade when locked
- ✓ Grenade random timeout works when unlocked
- ✓ Tests cover both states

---

### Scrap (`item_scrap`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_upgrade

**Implementation Checklist**:
- [ ] Add unlock check in crafting/disassemble handlers
- [ ] Update crafting service to verify scrap unlock
- [ ] Add tests for scrap acquisition/use when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (no scrap drops)
- [ ] Test unlocked behavior (scrap available)

**Files to Modify**:
- `internal/crafting/service.go` - Check scrap unlock in DisassembleItem
- `internal/crafting/service_test.go` - Add locked/unlocked tests
- `internal/handler/disassemble.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Scrap not available when locked
- ✓ Scrap drops from disassembly when unlocked
- ✓ Tests verify unlock gating

---

### Script (`item_script`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_economy

**Implementation Checklist**:
- [ ] Add unlock check in economy handlers
- [ ] Update economy service to verify script unlock
- [ ] Add tests for script currency when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (script not available)
- [ ] Test unlocked behavior (script currency active)

**Files to Modify**:
- `internal/economy/service.go` - Add script unlock check
- `internal/economy/service_test.go` - Add tests
- `internal/handler/prices.go` - Verify unlock for script transactions

**Acceptance Criteria**:
- ✓ Script currency unavailable when locked
- ✓ Script transactions work when unlocked
- ✓ Clear error messaging

---

### Good Lootbox (`item_lootbox2`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: item_lootbox1, job_gambler

**Implementation Checklist**:
- [ ] Add unlock check in lootbox handler
- [ ] Update lootbox service to verify lootbox2 unlock
- [ ] Add tests for opening lootbox2 when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot open)
- [ ] Test unlocked behavior (rare items available)

**Files to Modify**:
- `internal/lootbox/service.go` - Add lootbox2 unlock check
- `internal/lootbox/service_test.go` - Add tests
- `internal/handler/gamble.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot open lootbox2 when locked
- ✓ Can open lootbox2 with rare items when unlocked
- ✓ Tests cover unlock gating

---

### Shield (`item_shield`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: weapon_missile

**Implementation Checklist**:
- [ ] Add unlock check in item handler
- [ ] Update shield blocking logic to verify unlock
- [ ] Add tests for shield defense when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (no shield protection)
- [ ] Test unlocked behavior (blocks next attack)

**Files to Modify**:
- `internal/handler/user.go` - Add shield unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Shield unavailable when locked
- ✓ Shield blocks attacks when unlocked
- ✓ Tests verify unlock gating

---

### Rare Candy (`xp_rarecandy`)

**Type**: item | **Tier**: 2 | **Size**: medium

**Prerequisites**: upgrade_progression_basic

**Implementation Checklist**:
- [ ] Add unlock check in job XP handler
- [ ] Update job service to verify rare candy unlock
- [ ] Add tests for instant XP when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot use)
- [ ] Test unlocked behavior (instant XP grant)

**Files to Modify**:
- `internal/job/service.go` - Add rare candy unlock check in AwardXP
- `internal/job/service_test.go` - Add tests
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Rare candy unavailable when locked
- ✓ Instant XP grant works when unlocked
- ✓ Tests cover unlock gating

---

## Tier 3 Items

### This (`item_this`)

**Type**: item | **Tier**: 3 | **Size**: medium

**Prerequisites**: item_grenade

**Implementation Checklist**:
- [ ] Add unlock check in weapon handler
- [ ] Update item use to verify "this" unlock
- [ ] Add tests for 101s timeout when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot use)
- [ ] Test unlocked behavior (101s timeout)

**Files to Modify**:
- `internal/handler/user.go` - Add "this" unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Cannot use "this" when locked
- ✓ 101s timeout works when unlocked
- ✓ Tests verify unlock gating

---

### Small Revive (`item_revives`)

**Type**: item | **Tier**: 3 | **Size**: medium

**Prerequisites**: item_shield

**Implementation Checklist**:
- [ ] Add unlock check in timeout handler
- [ ] Update user service to verify revive unlock
- [ ] Add tests for timeout reduction when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (no revive available)
- [ ] Test unlocked behavior (60s timeout reduction)

**Files to Modify**:
- `internal/handler/user.go` - Add revive unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Revive unavailable when locked
- ✓ 60s timeout reduction works when unlocked
- ✓ Tests cover unlock gating

---

### Shiny Lootbox (`item_lootbox3`)

**Type**: item | **Tier**: 3 | **Size**: medium

**Prerequisites**: item_lootbox2

**Implementation Checklist**:
- [ ] Add unlock check in lootbox handler
- [ ] Update lootbox service to verify lootbox3 unlock
- [ ] Add tests for epic/legendary drops when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot open)
- [ ] Test unlocked behavior (legendary chance)

**Files to Modify**:
- `internal/lootbox/service.go` - Add lootbox3 unlock check
- `internal/lootbox/service_test.go` - Add tests
- `internal/handler/gamble.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot open lootbox3 when locked
- ✓ Epic/legendary drops available when unlocked
- ✓ Tests verify unlock gating

---

## Tier 4 Items

### TNT (`item_tnt`)

**Type**: item | **Tier**: 4 | **Size**: medium

**Prerequisites**: item_this, -nodes_unlocked_below_tier:2:20

**Implementation Checklist**:
- [ ] Add unlock check in weapon handler
- [ ] Update item use to verify TNT unlock
- [ ] Add tests for massive explosive effect when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot use)
- [ ] Test unlocked behavior (massive effect)

**Files to Modify**:
- `internal/handler/user.go` - Add TNT unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ TNT unavailable when locked
- ✓ Massive explosive effect when unlocked
- ✓ Tests verify unlock gating

---

### Huge Missile (`item_hugemissile`)

**Type**: item | **Tier**: 4 | **Size**: medium

**Prerequisites**: weapon_missile, -nodes_unlocked_below_tier:2:20

**Implementation Checklist**:
- [ ] Add unlock check in weapon handler
- [ ] Update item use to verify huge missile unlock
- [ ] Add tests for 100-minute timeout when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot use)
- [ ] Test unlocked behavior (100min timeout)

**Files to Modify**:
- `internal/handler/user.go` - Add huge missile unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Huge missile unavailable when locked
- ✓ 100-minute timeout works when unlocked
- ✓ Tests verify unlock gating

---

### Mirror Shield (`weapon_mirror`)

**Type**: item | **Tier**: 4 | **Size**: medium

**Prerequisites**: item_shield, item_revives

**Implementation Checklist**:
- [ ] Add unlock check in weapon/defense handler
- [ ] Update shield logic to verify mirror shield unlock
- [ ] Add tests for attack reflection when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (no reflection)
- [ ] Test unlocked behavior (attacks reflected)

**Files to Modify**:
- `internal/handler/user.go` - Add mirror shield unlock check
- `internal/user/service.go` - Verify unlock before use
- `internal/user/service_test.go` - Add tests

**Acceptance Criteria**:
- ✓ Mirror shield unavailable when locked
- ✓ Attack reflection works when unlocked
- ✓ Tests verify unlock gating

---

## Implementation Notes

### Common Pattern for Items

All item implementations follow this pattern:

```go
// In item service or handler
func (s *Service) UseItem(ctx context.Context, userID, itemKey string) error {
    // Check if item is unlocked
    nodeKey := progression.ItemKeyToNodeKey(itemKey)
    unlocked, err := s.progressionService.IsNodeUnlocked(ctx, nodeKey)
    if err != nil {
        return fmt.Errorf("failed to check unlock status: %w", err)
    }
    if !unlocked {
        return ErrItemLocked
    }

    // Proceed with normal item use logic
    // ...
}
```

### Testing Pattern

```go
func TestItemXXX_Locked(t *testing.T) {
    // Setup mock progression service returning false for IsNodeUnlocked
    // Attempt to use item
    // Assert error is ErrItemLocked
}

func TestItemXXX_Unlocked(t *testing.T) {
    // Setup mock progression service returning true for IsNodeUnlocked
    // Attempt to use item
    // Assert success
}
```

## Priority

Recommended implementation order:
1. Basic weapons (missile, grenade)
2. Lootboxes (lootbox1, lootbox2, lootbox3)
3. Defense items (shield, mirror shield, revives)
4. Utility items (shovel, stick, scrap, script)
5. Advanced weapons (this, TNT, huge missile)

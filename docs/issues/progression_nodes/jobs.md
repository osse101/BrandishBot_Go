# Job Implementation Tasks

6 job nodes requiring unlock gate implementation.

All jobs are Tier 2, medium size.

---

## Blacksmith Job (`job_blacksmith`)

**Type**: job | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_upgrade

**Description**: Job for Upgrading and Disassembling Items

**Implementation Checklist**:
- [ ] Add unlock check in job service for blacksmith activation
- [ ] Update crafting XP award to verify blacksmith unlock
- [ ] Add tests for blacksmith job when locked/unlocked
- [ ] Verify with admin unlock: `curl -X POST .../admin/unlock -d '{"node_key": "job_blacksmith", "level": 1}'`
- [ ] Test locked behavior (cannot activate job)
- [ ] Test unlocked behavior (can earn blacksmith XP from crafting)

**Files to Modify**:
- `internal/job/service.go` - Add blacksmith unlock check in GetUserJobs/AwardXP
- `internal/job/service_test.go` - Add locked/unlocked tests
- `internal/crafting/service.go` - Check blacksmith unlock before awarding XP
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot activate blacksmith job when locked
- ✓ Can earn XP from upgrading/disassembling when unlocked
- ✓ Error message: "Blacksmith job not unlocked"
- ✓ Tests cover both locked and unlocked states

---

## Explorer Job (`job_explorer`)

**Type**: job | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_search

**Description**: Job for exploring to find items

**Implementation Checklist**:
- [ ] Add unlock check in job service for explorer activation
- [ ] Update search/exploration to award explorer XP when unlocked
- [ ] Add tests for explorer job when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot activate job)
- [ ] Test unlocked behavior (earn XP from searching)

**Files to Modify**:
- `internal/job/service.go` - Add explorer unlock check
- `internal/job/service_test.go` - Add tests
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot activate explorer job when locked
- ✓ Earn XP from search activities when unlocked
- ✓ Error message: "Explorer job not unlocked"
- ✓ Tests cover unlock gating

---

## Gambler Job (`job_gambler`)

**Type**: job | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_gamble

**Description**: Job for playing games of chance

**Implementation Checklist**:
- [ ] Add unlock check in job service for gambler activation
- [ ] Update gamble service to award gambler XP when unlocked
- [ ] Add tests for gambler job when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot activate job)
- [ ] Test unlocked behavior (earn XP from gambling)

**Files to Modify**:
- `internal/job/service.go` - Add gambler unlock check
- `internal/job/service_test.go` - Add tests
- `internal/gamble/service.go` - Check gambler unlock before awarding XP
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot activate gambler job when locked
- ✓ Earn XP from gamble participation when unlocked
- ✓ Error message: "Gambler job not unlocked"
- ✓ Tests verify unlock gating

---

## Farmer Job (`job_farmer`)

**Type**: job | **Tier**: 2 | **Size**: medium

**Prerequisites**: feature_farming

**Description**: Unlock Farmer Job - earn XP through farming activities and crop management

**Implementation Checklist**:
- [ ] Add unlock check in job service for farmer activation
- [ ] Update farming service to award farmer XP when unlocked
- [ ] Add tests for farmer job when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot activate job)
- [ ] Test unlocked behavior (earn XP from farming)

**Files to Modify**:
- `internal/job/service.go` - Add farmer unlock check
- `internal/job/service_test.go` - Add tests
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot activate farmer job when locked
- ✓ Earn XP from farming activities when unlocked
- ✓ Error message: "Farmer job not unlocked"
- ✓ Tests cover unlock gating

---

## Merchant Job (`job_merchant`)

**Type**: job | **Tier**: 2 | **Size**: large

**Prerequisites**: feature_economy

**Description**: Unlock Merchant Job - earn XP through buying and selling items

**Implementation Checklist**:
- [ ] Add unlock check in job service for merchant activation
- [ ] Update economy service to award merchant XP when unlocked
- [ ] Add tests for merchant job when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot activate job)
- [ ] Test unlocked behavior (earn XP from buy/sell transactions)

**Files to Modify**:
- `internal/job/service.go` - Add merchant unlock check
- `internal/job/service_test.go` - Add tests
- `internal/economy/service.go` - Check merchant unlock before awarding XP
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot activate merchant job when locked
- ✓ Earn XP from economy transactions when unlocked
- ✓ Error message: "Merchant job not unlocked"
- ✓ Tests verify unlock gating

---

## Scholar Job (`job_scholar`)

**Type**: job | **Tier**: 2 | **Size**: medium

**Prerequisites**: upgrade_progression_basic

**Description**: Job for using the progression system

**Implementation Checklist**:
- [ ] Add unlock check in job service for scholar activation
- [ ] Update progression service to award scholar XP when unlocked
- [ ] Add tests for scholar job when locked/unlocked
- [ ] Verify with admin unlock
- [ ] Test locked behavior (cannot activate job)
- [ ] Test unlocked behavior (earn XP from progression engagement)

**Files to Modify**:
- `internal/job/service.go` - Add scholar unlock check
- `internal/job/service_test.go` - Add tests
- `internal/progression/service.go` - Check scholar unlock before awarding XP
- `internal/handler/job.go` - Verify unlock

**Acceptance Criteria**:
- ✓ Cannot activate scholar job when locked
- ✓ Earn XP from voting/progression activity when unlocked
- ✓ Error message: "Scholar job not unlocked"
- ✓ Tests verify unlock gating

---

## Implementation Pattern

All jobs follow this pattern:

### In Job Service

```go
// GetUserJobs - check if job is unlocked before returning it
func (s *Service) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error) {
    allJobs, err := s.repo.GetAllJobs(ctx)
    if err != nil {
        return nil, err
    }

    var availableJobs []domain.Job
    for _, job := range allJobs {
        // Check if job is unlocked
        nodeKey := progression.JobKeyToNodeKey(job.Key)
        unlocked, err := s.progressionService.IsNodeUnlocked(ctx, nodeKey)
        if err != nil {
            return nil, fmt.Errorf("failed to check job unlock: %w", err)
        }
        if unlocked {
            availableJobs = append(availableJobs, job)
        }
    }

    // Return user's progress in available jobs
    return s.repo.GetUserJobs(ctx, userID)
}

// AwardXP - check if job is unlocked before awarding XP
func (s *Service) AwardXP(ctx context.Context, userID, jobKey string, xp int) error {
    nodeKey := progression.JobKeyToNodeKey(jobKey)
    unlocked, err := s.progressionService.IsNodeUnlocked(ctx, nodeKey)
    if err != nil {
        return fmt.Errorf("failed to check job unlock: %w", err)
    }
    if !unlocked {
        return ErrJobLocked
    }

    // Proceed with XP award
    return s.repo.AwardXP(ctx, userID, jobKey, xp)
}
```

### Testing Pattern

```go
func TestJobXXX_Locked(t *testing.T) {
    // Setup mock progression service returning false for IsNodeUnlocked
    // Attempt to activate job or award XP
    // Assert error is ErrJobLocked
}

func TestJobXXX_Unlocked(t *testing.T) {
    // Setup mock progression service returning true for IsNodeUnlocked
    // Attempt to activate job or award XP
    // Assert success
}
```

## XP Award Integration

Each job needs XP awarded from its associated activity:

| Job | Activity | Service | Method |
|-----|----------|---------|--------|
| Blacksmith | Upgrading/Disassembling | `crafting.Service` | `UpgradeItem`, `DisassembleItem` |
| Explorer | Searching | TBD (search service) | Search operations |
| Gambler | Gambling | `gamble.Service` | `ExecuteGamble` |
| Farmer | Farming | TBD (farming service) | Farming operations |
| Merchant | Buy/Sell | `economy.Service` | `BuyItem`, `SellItem` |
| Scholar | Progression | `progression.Service` | `VoteForUnlock`, `RecordEngagement` |

## Priority

Recommended implementation order:
1. **Scholar** - Progression engagement (core mechanic)
2. **Gambler** - Already have gamble service
3. **Blacksmith** - Already have crafting service
4. **Merchant** - Already have economy service
5. **Explorer** - Need search service implementation
6. **Farmer** - Need farming service implementation

## Notes

- Jobs are currently auto-unlocked in development for testing
- Once job unlock gates are implemented, users must unlock jobs before earning XP
- Job level caps and XP multipliers are affected by upgrade nodes
- All jobs share the same base XP formula but different activities

package harvest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/mocks"
)

// MockResilientPublisher implements ResilientPublisher for testing
type MockResilientPublisher struct {
	mock.Mock
}

func (m *MockResilientPublisher) PublishWithRetry(ctx context.Context, evt event.Event) {
	m.Called(ctx, evt)
}

func TestHarvest_Success(t *testing.T) {
	// Setup Mocks
	mockRepo := new(mocks.MockRepositoryHarvestRepository)
	mockUserRepo := new(mocks.MockRepositoryUser)
	mockProgressionSvc := new(mocks.MockProgressionService)
	mockJobSvc := new(mocks.MockJobService)
	mockPublisher := new(MockResilientPublisher)
	mockTx := new(mocks.MockRepositoryHarvestTx)

	svc := NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, mockPublisher)
	ctx := context.Background()

	user := &domain.User{ID: "user1"}
	now := time.Now()
	lastHarvested := now.Add(-5 * time.Hour) // 5 hours ago (Tier 2)

	// Expectations
	mockUserRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	mockProgressionSvc.On("IsFeatureUnlocked", ctx, "feature_farming").Return(true, nil)

	// Harvest State logic
	mockRepo.On("GetHarvestState", ctx, user.ID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	// performHarvestTransaction -> BeginTx
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	defer mockTx.AssertExpectations(t)

	// Inside Tx:
	// 1. GetHarvestStateWithLock
	mockTx.On("GetHarvestStateWithLock", ctx, user.ID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	// 2. getBonusMultipliers -> GetModifiedValue calls
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureHarvestYield, 1.0).Return(1.0, nil)
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureGrowthSpeed, 1.0).Return(1.0, nil)

	// 3. calculateHarvestRewards -> GetModifiedValue for spoil and tier
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureSpoilExtension, 0.0).Return(0.0, nil)
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureHarvestTier, 3.0).Return(9.0, nil) // Max tier available

	// 4. calculateRewards: 5 hours (Tier 2) -> 12 money (no unlocks needed)

	// 5. fireAsyncEvents -> PublishWithRetry
	mockPublisher.On("PublishWithRetry", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
		return evt.Type == domain.EventTypeHarvestCompleted
	})).Return()

	// 6. applyHarvestRewards -> GetInventory, GetItemsByNames, UpdateInventory
	inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)

	moneyItem := domain.Item{ID: 1, InternalName: domain.ItemMoney}
	mockUserRepo.On("GetItemsByNames", ctx, []string{domain.ItemMoney}).Return([]domain.Item{moneyItem}, nil)

	mockTx.On("UpdateInventory", ctx, user.ID, mock.MatchedBy(func(inv domain.Inventory) bool {
		return len(inv.Slots) == 1 && inv.Slots[0].ItemID == 1 && inv.Slots[0].Quantity == 12
	})).Return(nil)

	// 7. UpdateHarvestState
	mockTx.On("UpdateHarvestState", ctx, user.ID, mock.Anything).Return(nil)

	// 8. Commit
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil).Maybe()

	// Execute
	resp, err := svc.Harvest(ctx, domain.PlatformTwitch, "123", "testuser")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 12, resp.ItemsGained[domain.ItemMoney])
	assert.InDelta(t, 5.0, resp.HoursSinceHarvest, 0.1)
}

func TestHarvest_Spoiled(t *testing.T) {
	// Setup Mocks
	mockRepo := new(mocks.MockRepositoryHarvestRepository)
	mockUserRepo := new(mocks.MockRepositoryUser)
	mockProgressionSvc := new(mocks.MockProgressionService)
	mockJobSvc := new(mocks.MockJobService)
	mockPublisher := new(MockResilientPublisher)
	mockTx := new(mocks.MockRepositoryHarvestTx)

	svc := NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, mockPublisher)
	ctx := context.Background()

	user := &domain.User{ID: "user1"}
	now := time.Now()
	// Spoiled threshold is 336 hours. Let's say 400 hours.
	lastHarvested := now.Add(-400 * time.Hour)

	// Expectations
	mockUserRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	mockProgressionSvc.On("IsFeatureUnlocked", ctx, "feature_farming").Return(true, nil)
	mockRepo.On("GetHarvestState", ctx, user.ID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	defer mockTx.AssertExpectations(t)

	mockTx.On("GetHarvestStateWithLock", ctx, user.ID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureHarvestYield, 1.0).Return(1.0, nil)
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureGrowthSpeed, 1.0).Return(1.0, nil)
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureSpoilExtension, 0.0).Return(0.0, nil)
	mockProgressionSvc.On("GetModifiedValue", ctx, user.ID, featureHarvestTier, 3.0).Return(9.0, nil)

	// Spoiled logic returns fixed rewards: 1 Lootbox1, 3 Sticks.
	mockPublisher.On("PublishWithRetry", mock.Anything, mock.Anything).Return()

	inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)

	// Expect lookup for Lootbox1 and Stick
	lbItem := domain.Item{ID: 2, InternalName: domain.ItemLootbox1}
	stickItem := domain.Item{ID: 3, InternalName: domain.ItemStick}
	mockUserRepo.On("GetItemsByNames", ctx, mock.MatchedBy(func(names []string) bool {
		return len(names) == 2 // Order might vary
	})).Return([]domain.Item{lbItem, stickItem}, nil)

	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("UpdateHarvestState", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil).Maybe()

	// Execute
	resp, err := svc.Harvest(ctx, domain.PlatformTwitch, "123", "testuser")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, resp.ItemsGained[domain.ItemLootbox1])
	assert.Equal(t, 3, resp.ItemsGained[domain.ItemStick])
	assert.Contains(t, resp.Message, "spoiled")
}

func TestHarvest_TooSoon(t *testing.T) {
	mockRepo := new(mocks.MockRepositoryHarvestRepository)
	mockUserRepo := new(mocks.MockRepositoryUser)
	mockProgressionSvc := new(mocks.MockProgressionService)
	mockJobSvc := new(mocks.MockJobService)
	mockPublisher := new(MockResilientPublisher)
	mockTx := new(mocks.MockRepositoryHarvestTx)

	svc := NewService(mockRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, mockPublisher)
	ctx := context.Background()

	user := &domain.User{ID: "user1"}
	// 30 mins ago (too soon, min is 1 hour)
	lastHarvested := time.Now().Add(-30 * time.Minute)

	mockUserRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "123").Return(user, nil)
	mockProgressionSvc.On("IsFeatureUnlocked", ctx, "feature_farming").Return(true, nil)
	mockRepo.On("GetHarvestState", ctx, user.ID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)

	mockTx.On("GetHarvestStateWithLock", ctx, user.ID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)
	mockTx.On("Rollback", ctx).Return(nil).Maybe()

	_, err := svc.Harvest(ctx, domain.PlatformTwitch, "123", "testuser")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrHarvestTooSoon))
}

func TestCalculateRewards(t *testing.T) {
	tests := []struct {
		name            string
		hoursElapsed    float64
		unlockedItems   map[string]bool
		limitIndex      int
		expectedReward  map[string]int
		yieldMultiplier float64
	}{
		{
			name:         "Less than 2 hours - no tier reached",
			hoursElapsed: 1.5,
			limitIndex:   9,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{}, // No tier reached
		},
		{
			name:         "Exactly 2 hours - Tier 1",
			hoursElapsed: 2.0,
			limitIndex:   9,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 2,
			},
		},
		{
			name:         "5 hours - Tier 1 + 2",
			hoursElapsed: 5.0,
			limitIndex:   9,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 12, // 2 + 10
			},
		},
		{
			name:         "24 hours - All stick tiers, stick unlocked",
			hoursElapsed: 24.0,
			limitIndex:   9,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: false,
				itemLootbox2: false,
			},
			expectedReward: map[string]int{
				itemMoney: 22, // 2 + 10 + 5 + 5
				itemStick: 3,  // 1 + 2
			},
		},
		{
			name:         "24 hours - stick NOT unlocked",
			hoursElapsed: 24.0,
			limitIndex:   9,
			unlockedItems: map[string]bool{
				itemStick:    false,
				itemLootbox1: false,
				itemLootbox2: false,
			},
			expectedReward: map[string]int{
				itemMoney: 22, // 2 + 10 + 5 + 5 (money from stick tiers still counts)
			},
		},
		{
			name:         "Tier Limit Applied (Limit at Tier 1 -> 2 hours)",
			hoursElapsed: 168.0, // Enough for max tier
			limitIndex:   0,     // But limited to Tier 0 (2 hours)
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 2, // Only Tier 1 reward
			},
		},
		{
			name:         "Tier Limit Applied (Limit at Tier 2 -> 5 hours)",
			hoursElapsed: 168.0,
			limitIndex:   1, // Limited to Tier 1 (5 hours)
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 12, // Tier 1 (2) + Tier 2 (10)
			},
		},
		{
			name:         "Yield Bonus - 1.5x multiplier",
			hoursElapsed: 5.0, // Tier 1 + 2 (12 money)
			limitIndex:   9,
			unlockedItems: map[string]bool{
				itemStick:    true,
				itemLootbox1: true,
				itemLootbox2: true,
			},
			expectedReward: map[string]int{
				itemMoney: 18, // 12 * 1.5 = 18
			},
			yieldMultiplier: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockProgressionSvc := new(mocks.MockProgressionService)

			// Setup IsItemUnlocked expectations
			for itemName, unlocked := range tt.unlockedItems {
				mockProgressionSvc.On("IsItemUnlocked", mock.Anything, itemName).Return(unlocked, nil).Maybe()
			}

			// Create service
			svc := &service{
				progressionSvc: mockProgressionSvc,
			}

			// Execute
			multiplier := 1.0
			if tt.yieldMultiplier > 0 {
				multiplier = tt.yieldMultiplier
			}
			rewards := svc.calculateRewards(context.Background(), tt.hoursElapsed, multiplier, tt.limitIndex)

			// Assert
			assert.Equal(t, tt.expectedReward, rewards)

			mockProgressionSvc.AssertExpectations(t)
		})
	}
}

func TestRewardTiers(t *testing.T) {
	tiers := getRewardTiers()

	// Verify tiers are ordered by MaxHours
	for i := 1; i < len(tiers); i++ {
		assert.Greater(t, tiers[i].MaxHours, tiers[i-1].MaxHours,
			"Tiers must be ordered by MaxHours")
	}

	// Verify tier structure
	assert.Equal(t, 10, len(tiers), "Should have 10 tiers")

	// Verify first tier
	assert.Equal(t, 2.0, tiers[0].MaxHours)
	assert.Equal(t, 2, tiers[0].Items[itemMoney])
	assert.Empty(t, tiers[0].RequiresUnlock)

	// Verify last tier
	assert.Equal(t, 168.0, tiers[9].MaxHours)
	assert.Equal(t, 20, tiers[9].Items[itemMoney])
	assert.Equal(t, 1, tiers[9].Items[itemLootbox2])
	assert.True(t, tiers[9].RequiresUnlock[itemLootbox2])
}

func TestMinHarvestInterval(t *testing.T) {
	assert.Equal(t, 1.0, minHarvestInterval, "Minimum harvest interval should be 1 hour")
}

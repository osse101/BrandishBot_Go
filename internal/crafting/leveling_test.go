package crafting

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
)

// MockJobService for testing XP awards
type MockJobService struct {
	AwardXPCalled bool
	AwardXPResult *domain.XPAwardResult
	AwardXPErr    error
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	m.AwardXPCalled = true
	return m.AwardXPResult, m.AwardXPErr
}

func TestUpgradeItem_LevelUpFeedback(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	mockStats := &MockStatsService{}
	mockJob := &MockJobService{}

	// Create service with mock job service
	svc := NewService(repo, mockJob, mockStats).(*service)
	svc.rnd = func() float64 { return 1.0 } // Force no masterwork

	// Setup mock job service to return a level up
	mockJob.AwardXPResult = &domain.XPAwardResult{
		JobKey:    "blacksmith",
		XPGained:  100,
		NewXP:     1000,
		NewLevel:  5,
		LeveledUp: true,
	}

	ctx := context.Background()

	// Give alice 2 lootbox0
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 1, Quantity: 2})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Upgrade
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	assert.NoError(t, err)

	// Verify leveling feedback
	assert.True(t, result.LeveledUp, "Should report level up")
	assert.Equal(t, 5, result.NewLevel, "Should report new level")
	assert.Equal(t, 100, result.XPGained, "Should report XP gained")
	assert.True(t, mockJob.AwardXPCalled, "Job service should be called")
}

func TestUpgradeItem_EpiphanyFeedback(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	mockStats := &MockStatsService{}
	mockJob := &MockJobService{}

	svc := NewService(repo, mockJob, mockStats).(*service)
	svc.rnd = func() float64 { return 1.0 }

	// Setup mock job service to return MORE XP than base (Epiphany)
	// Base XP is typically 10 * quantity. Let's say we craft 2 items = 20 XP.
	// We return 40 XP to simulate Epiphany.
	mockJob.AwardXPResult = &domain.XPAwardResult{
		JobKey:    "blacksmith",
		XPGained:  40, // Double normal
		NewXP:     1000,
		NewLevel:  4,
		LeveledUp: false,
	}

	ctx := context.Background()

	// Give alice 2 lootbox0
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 1, Quantity: 2})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Upgrade
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	assert.NoError(t, err)

	// Verify epiphany feedback
	assert.True(t, result.IsEpiphany, "Should report epiphany when XP > base")
	assert.Equal(t, 40, result.XPGained)
}

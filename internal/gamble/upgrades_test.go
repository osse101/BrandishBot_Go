package gamble

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

// This file contains test stubs for gamble upgrade node modifier application.
// See docs/issues/progression_nodes/upgrades.md for implementation details.

func TestUpgradeGambleWinBonus_ExistingImplementation(t *testing.T) {
	ts := setupService(nil, true)
	ctx := context.Background()
	gambleID := uuid.New()

	// Setup gamble with 1 participant
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	ts.repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	// Mock transaction
	tx := new(MockTx)
	ts.repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	// Mock item resolution
	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	ts.namingResolver.On("ResolvePublicName", domain.ItemLootbox1).Return("", false)
	ts.repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	ts.repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	// Mock lootbox drop (value 100)
	drops := []lootbox.DroppedItem{{ItemID: 10, Quantity: 1, Value: 100}}
	ts.lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops, nil)

	// Mock Progression Service: 1.25x bonus (100 -> 125)
	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(100)).Return(float64(125), nil)

	// Mock remaining calls
	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	ts.resilientPub.On("PublishWithRetry", ctx, mock.Anything).Return()

	result, err := ts.svc.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, int64(125), result.TotalValue)
	assert.Equal(t, int64(125), result.Items[0].Value) // Individual item value should be updated
	ts.progressionSvc.AssertExpectations(t)
}

func TestUpgradeGambleWinBonus_AllGambleTypes(t *testing.T) {
	ts := setupService(nil, true)
	ctx := context.Background()
	gambleID := uuid.New()

	// Setup gamble with mixed bets (simulated by drops)
	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 2}}},
		},
	}

	ts.repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	tx := new(MockTx)
	ts.repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	ts.namingResolver.On("ResolvePublicName", domain.ItemLootbox1).Return("", false)
	ts.repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	ts.repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	// Mock lootbox drops: one currency (value 100), one item (value 200)
	// OpenLootbox called once for qty 2, returns 2 items
	drops := []lootbox.DroppedItem{
		{ItemID: 10, Quantity: 1, Value: 100},
		{ItemID: 11, Quantity: 1, Value: 200},
	}
	ts.lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 2, mock.Anything).Return(drops, nil)

	// Mock Progression Service
	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(100)).Return(float64(110), nil) // 1.1x
	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(200)).Return(float64(220), nil) // 1.1x

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	ts.resilientPub.On("PublishWithRetry", ctx, mock.Anything).Return()

	result, err := ts.svc.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, int64(330), result.TotalValue) // 110 + 220
	ts.progressionSvc.AssertExpectations(t)
}

func TestUpgradeGambleWinBonus_MultipleParticipants(t *testing.T) {
	ts := setupService(nil, true)
	ctx := context.Background()
	gambleID := uuid.New()

	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
			{UserID: "user2", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	ts.repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	tx := new(MockTx)
	ts.repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	ts.namingResolver.On("ResolvePublicName", domain.ItemLootbox1).Return("", false)
	ts.repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	ts.repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	// User1 gets 100, User2 gets 200. Winner is User2.
	drops := []lootbox.DroppedItem{{ItemID: 10, Quantity: 1, Value: 100}}
	ts.lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops, nil).Twice()

	// Mock Progression Service - called for BOTH participants during calculation
	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(100)).Return(float64(150), nil).Twice()

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, mock.Anything).Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, mock.Anything, mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	ts.resilientPub.On("PublishWithRetry", ctx, mock.Anything).Return()

	result, err := ts.svc.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, int64(300), result.TotalValue) // 150 + 150
	ts.progressionSvc.AssertExpectations(t)
}

func TestUpgradeGambleWinBonus_ModifierFailureFallback(t *testing.T) {
	ts := setupService(nil, true)
	ctx := context.Background()
	gambleID := uuid.New()

	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "user1", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	ts.repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	tx := new(MockTx)
	ts.repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	ts.namingResolver.On("ResolvePublicName", domain.ItemLootbox1).Return("", false)
	ts.repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	ts.repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	drops := []lootbox.DroppedItem{{ItemID: 10, Quantity: 1, Value: 100}}
	ts.lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops, nil)

	// Mock Progression Service Failure
	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(100)).Return(float64(0), assert.AnError)

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, "user1").Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, "user1", mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()
	ts.resilientPub.On("PublishWithRetry", ctx, mock.Anything).Return()

	result, err := ts.svc.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, int64(100), result.TotalValue) // Should fallback to base value
	ts.progressionSvc.AssertExpectations(t)
}

func TestUpgradeGambleWinBonus_IntegrationTest(t *testing.T) {
	TestUpgradeGambleWinBonus_ExistingImplementation(t)
}

func TestUpgradeGambleWinBonus_NearMissInteraction(t *testing.T) {
	ts := setupService(nil, true)
	ctx := context.Background()
	gambleID := uuid.New()

	gamble := &domain.Gamble{
		ID:    gambleID,
		State: domain.GambleStateJoining,
		Participants: []domain.Participant{
			{UserID: "winner", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
			{UserID: "loser", LootboxBets: []domain.LootboxBet{{ItemName: domain.ItemLootbox1, Quantity: 1}}},
		},
	}

	ts.repo.On("GetGamble", ctx, gambleID).Return(gamble, nil)

	tx := new(MockTx)
	ts.repo.On("BeginGambleTx", ctx).Return(tx, nil)
	tx.On("UpdateGambleStateIfMatches", ctx, gambleID, domain.GambleStateJoining, domain.GambleStateOpening).Return(int64(1), nil)

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	ts.namingResolver.On("ResolvePublicName", domain.ItemLootbox1).Return("", false)
	ts.repo.On("GetItemByName", ctx, domain.ItemLootbox1).Return(lootboxItem, nil)
	ts.repo.On("GetItemByID", ctx, 1).Return(lootboxItem, nil)

	dropsWinner := []lootbox.DroppedItem{{ItemID: 10, Quantity: 1, Value: 100}}
	dropsLoser := []lootbox.DroppedItem{{ItemID: 10, Quantity: 1, Value: 95}}

	ts.lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(dropsWinner, nil).Once()
	ts.lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(dropsLoser, nil).Once()

	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(100)).Return(float64(125), nil)
	ts.progressionSvc.On("GetModifiedValue", ctx, ProgressionFeatureGambleWinBonus, float64(95)).Return(float64(118), nil)

	tx.On("SaveOpenedItems", ctx, mock.Anything).Return(nil)
	tx.On("GetInventory", ctx, "winner").Return(&domain.Inventory{}, nil)
	tx.On("UpdateInventory", ctx, "winner", mock.Anything).Return(nil)
	tx.On("CompleteGamble", ctx, mock.Anything).Return(nil)
	tx.On("Commit", ctx).Return(nil)
	tx.On("Rollback", ctx).Return(nil).Maybe()

	ts.resilientPub.On("PublishWithRetry", ctx, mock.MatchedBy(func(e event.Event) bool {
		payload, ok := e.Payload.(domain.GambleCompletedPayloadV2)
		if !ok {
			return false
		}
		for _, p := range payload.Participants {
			if p.UserID == "loser" {
				return p.IsNearMiss
			}
		}
		return false
	})).Return()

	result, err := ts.svc.ExecuteGamble(ctx, gambleID)

	assert.NoError(t, err)
	assert.Equal(t, "winner", result.WinnerID)
	ts.progressionSvc.AssertExpectations(t)
	ts.resilientPub.AssertExpectations(t)
}

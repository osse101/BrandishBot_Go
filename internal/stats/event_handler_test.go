package stats_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestEventHandler_HandleItemSold(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.ItemSoldPayload{
		UserID:       "user-1",
		ItemName:     "stick",
		ItemCategory: "material",
		Quantity:     5,
		TotalValue:   50,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemSold),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventItemSold, mock.MatchedBy(func(v map[string]interface{}) bool {
		return v["item_name"] == "stick" && v["quantity"] == 5 && v["total_value"] == 50 && v["category"] == "material"
	})).Return(nil)

	err := handler.HandleItemSold(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemBought(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.ItemBoughtPayload{
		UserID:       "user-1",
		ItemName:     "shovel",
		ItemCategory: "utility",
		Quantity:     1,
		TotalValue:   100,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemBought),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventItemBought, mock.MatchedBy(func(v map[string]interface{}) bool {
		return v["item_name"] == "shovel" && v["quantity"] == 1 && v["total_value"] == 100 && v["category"] == "utility"
	})).Return(nil)

	err := handler.HandleItemBought(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemAdded(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.ItemAddedPayload{
		UserID:   "user-1",
		ItemName: "stick",
		Quantity: 5,
		Source:   "search",
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemAdded),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventItemAdded, payload).Return(nil)

	err := handler.HandleItemAdded(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemRemoved(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.ItemRemovedPayload{
		UserID:   "user-1",
		ItemName: "stick",
		Quantity: 5,
		Source:   "crafting",
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemRemoved),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventItemRemoved, payload).Return(nil)

	err := handler.HandleItemRemoved(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemUsed(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.ItemUsedPayload{
		UserID:   "user-1",
		ItemName: "shield",
		Quantity: 1,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemUsed),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventItemUsed, payload).Return(nil)

	err := handler.HandleItemUsed(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemTransferred(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.ItemTransferredPayload{
		FromUserID: "user-1",
		ToUserID:   "user-2",
		ItemName:   "stick",
		Quantity:   5,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemTransferred),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventItemTransferred, payload).Return(nil)
	mockSvc.On("RecordUserEvent", ctx, "user-2", domain.StatsEventItemTransferred, payload).Return(nil)

	err := handler.HandleItemTransferred(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleSearchPerformed(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.SearchPerformedPayload{
		UserID:         "user-1",
		IsCritical:     true,
		IsNearMiss:     false,
		IsCriticalFail: false,
		IsFirstDaily:   true,
		XPAmount:       10,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeSearchPerformed),
		Payload: payload,
	}

	metadata := domain.SearchMetadata{
		IsCritical:   true,
		IsNearMiss:   false,
		IsCritFail:   false,
		IsFirstDaily: true,
		XPAmount:     10,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventSearch, metadata).Return(nil)
	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventSearchCriticalSuccess, metadata).Return(nil)

	err := handler.HandleSearchPerformed(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleJobLevelUp(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := event.JobLevelUpPayloadV1{
		UserID:   "user-1",
		JobKey:   "job_miner",
		NewLevel: 5,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeJobLevelUp),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventTypeJobLevelUp, payload).Return(nil)

	err := handler.HandleJobLevelUp(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleJobXPCritical(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := event.JobXPCriticalPayloadV1{
		UserID:  "user-1",
		JobKey:  "job_miner",
		BaseXP:  10,
		BonusXP: 5,
		Source:  "epiphany",
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeJobXPCritical),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventType(domain.EventTypeJobXPCritical), payload).Return(nil)

	err := handler.HandleJobXPCritical(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandlePredictionParticipated(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.PredictionParticipantPayload{
		UserID:   "user-1",
		Username: "TestUser",
		IsWinner: true,
		Platform: "twitch",
		XP:       50,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypePredictionParticipated),
		Payload: payload,
	}

	metadata := domain.PredictionMetadata{
		Username: "TestUser",
		IsWinner: true,
		Platform: "twitch",
		XP:       50,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventType("prediction_participation"), metadata).Return(nil)

	err := handler.HandlePredictionParticipated(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleSlotsCompleted(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.SlotsCompletedPayload{
		UserID:           "user-1",
		BetAmount:        10,
		PayoutAmount:     20,
		PayoutMultiplier: 2.0,
		IsWin:            true,
		IsNearMiss:       false,
		TriggerType:      "mega_jackpot",
		Reel1:            "apple",
		Reel2:            "apple",
		Reel3:            "apple",
	}

	evt := event.Event{
		Type:    event.Type(domain.EventSlotsCompleted),
		Payload: payload,
	}

	metadata := domain.SlotsMetadata{
		BetAmount:        10,
		PayoutAmount:     20,
		PayoutMultiplier: 2.0,
		NetProfit:        10,
		IsWin:            true,
		IsNearMiss:       false,
		TriggerType:      "mega_jackpot",
		Reel1:            "apple",
		Reel2:            "apple",
		Reel3:            "apple",
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventTypeSlotsSpin, metadata).Return(nil)
	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventTypeSlotsWin, metadata).Return(nil)
	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventTypeSlotsMegaJackpot, metadata).Return(nil)

	err := handler.HandleSlotsCompleted(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleGambleCompleted(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := domain.GambleCompletedPayloadV2{
		GambleID:   "gamble-1",
		TotalValue: 100,
		Participants: []domain.GambleParticipantOutcome{
			{
				UserID:     "user-1",
				Score:      95,
				IsNearMiss: true,
			},
			{
				UserID:     "user-2",
				Score:      0,
				IsCritFail: true,
			},
			{
				UserID:         "user-3",
				Score:          100,
				IsTieBreakLost: true,
			},
		},
	}

	evt := event.Event{
		Type:    event.Type(domain.EventGambleCompleted),
		Payload: payload,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.StatsEventGambleNearMiss, domain.GambleMetadata{
		GambleID:    "gamble-1",
		Score:       95,
		WinnerScore: 100,
	}).Return(nil)

	mockSvc.On("RecordUserEvent", ctx, "user-2", domain.StatsEventGambleCriticalFail, domain.GambleMetadata{
		GambleID: "gamble-1",
		Score:    0,
	}).Return(nil)

	mockSvc.On("RecordUserEvent", ctx, "user-3", domain.StatsEventGambleTieBreakLost, domain.GambleMetadata{
		GambleID: "gamble-1",
		Score:    100,
	}).Return(nil)

	err := handler.HandleGambleCompleted(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemUpgraded(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := crafting.ItemUpgradedPayload{
		UserID:        "user-1",
		ItemName:      "sword",
		Quantity:      1,
		IsMasterwork:  true,
		BonusQuantity: 1,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemUpgraded),
		Payload: payload,
	}

	metadata := domain.CraftingMetadata{
		ItemName:         "sword",
		OriginalQuantity: 1,
		MasterworkCount:  1,
		BonusQuantity:    1,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventTypeCraftingCriticalSuccess, metadata).Return(nil)

	err := handler.HandleItemUpgraded(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemDisassembled(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := crafting.ItemDisassembledPayload{
		UserID:              "user-1",
		ItemName:            "sword",
		Quantity:            1,
		IsPerfectSalvage:    true,
		PerfectSalvageCount: 1,
		Multiplier:          2.0,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemDisassembled),
		Payload: payload,
	}

	metadata := domain.CraftingMetadata{
		ItemName:     "sword",
		Quantity:     1,
		PerfectCount: 1,
		Multiplier:   2.0,
	}

	mockSvc.On("RecordUserEvent", ctx, "user-1", domain.EventTypeCraftingPerfectSalvage, metadata).Return(nil)

	err := handler.HandleItemDisassembled(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleItemSold_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemSold),
		Payload: "invalid payload",
	}

	err := handler.HandleItemSold(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemBought_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemBought),
		Payload: "invalid payload",
	}

	err := handler.HandleItemBought(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemAdded_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemAdded),
		Payload: "invalid payload",
	}

	err := handler.HandleItemAdded(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemRemoved_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemRemoved),
		Payload: "invalid payload",
	}

	err := handler.HandleItemRemoved(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemTransferred_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemTransferred),
		Payload: "invalid payload",
	}

	err := handler.HandleItemTransferred(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemUsed_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemUsed),
		Payload: "invalid payload",
	}

	err := handler.HandleItemUsed(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleSearchPerformed_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeSearchPerformed),
		Payload: "invalid payload",
	}

	err := handler.HandleSearchPerformed(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleJobLevelUp_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeJobLevelUp),
		Payload: "invalid payload",
	}

	err := handler.HandleJobLevelUp(ctx, evt)
	require.NoError(t, err) // Don't fail on type mismatch
}

func TestEventHandler_HandleJobLevelUp_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := event.JobLevelUpPayloadV1{
		UserID: "",
	}
	evt := event.Event{
		Type:    event.Type(domain.EventTypeJobLevelUp),
		Payload: payload,
	}

	err := handler.HandleJobLevelUp(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandleJobXPCritical_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeJobXPCritical),
		Payload: "invalid payload",
	}

	err := handler.HandleJobXPCritical(ctx, evt)
	require.NoError(t, err) // Don't fail on type mismatch
}

func TestEventHandler_HandleJobXPCritical_EmptyUserID(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	payload := event.JobXPCriticalPayloadV1{
		UserID: "",
	}
	evt := event.Event{
		Type:    event.Type(domain.EventTypeJobXPCritical),
		Payload: payload,
	}

	err := handler.HandleJobXPCritical(ctx, evt)
	require.NoError(t, err)
}

func TestEventHandler_HandlePredictionParticipated_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypePredictionParticipated),
		Payload: "invalid payload",
	}

	err := handler.HandlePredictionParticipated(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleSlotsCompleted_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventSlotsCompleted),
		Payload: "invalid payload",
	}

	err := handler.HandleSlotsCompleted(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleGambleCompleted_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventGambleCompleted),
		Payload: "invalid payload",
	}

	err := handler.HandleGambleCompleted(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemUpgraded_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemUpgraded),
		Payload: "invalid payload",
	}

	err := handler.HandleItemUpgraded(ctx, evt)
	require.Error(t, err)
}

func TestEventHandler_HandleItemDisassembled_Error(t *testing.T) {
	ctx := context.Background()
	mockSvc := mocks.NewMockStatsService(t)
	handler := stats.NewEventHandler(mockSvc)

	evt := event.Event{
		Type:    event.Type(domain.EventTypeItemDisassembled),
		Payload: "invalid payload",
	}

	err := handler.HandleItemDisassembled(ctx, evt)
	require.Error(t, err)
}

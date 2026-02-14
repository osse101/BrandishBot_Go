package job

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventHandler handles events related to jobs
type EventHandler struct {
	service Service
}

// NewEventHandler creates a new job event handler
func NewEventHandler(service Service) *EventHandler {
	return &EventHandler{
		service: service,
	}
}

// Register subscribes the handler to relevant events
func (h *EventHandler) Register(bus event.Bus) {
	// Crafting events
	bus.Subscribe(event.Type(domain.EventTypeItemUpgraded), h.HandleItemUpgraded)
	bus.Subscribe(event.Type(domain.EventTypeItemDisassembled), h.HandleItemDisassembled)

	// Slots events
	bus.Subscribe(event.Type(domain.EventSlotsCompleted), h.HandleSlotsCompleted)

	// Harvest / Compost / Expedition events
	bus.Subscribe(event.Type(domain.EventTypeHarvestCompleted), h.HandleHarvestCompleted)
	bus.Subscribe(event.Type(domain.EventTypeCompostHarvested), h.HandleCompostHarvested)
	bus.Subscribe(event.Type(domain.EventTypeExpeditionRewarded), h.HandleExpeditionRewarded)

	// Gamble events
	bus.Subscribe(event.Type(domain.EventTypeGambleParticipated), h.HandleGambleParticipated)
	bus.Subscribe(event.Type(domain.EventGambleCompleted), h.HandleGambleCompleted)

	// Economy events
	bus.Subscribe(event.Type(domain.EventTypeItemSold), h.HandleItemSold)
	bus.Subscribe(event.Type(domain.EventTypeItemBought), h.HandleItemBought)

	// Search events
	bus.Subscribe(event.Type(domain.EventTypeSearchPerformed), h.HandleSearchPerformed)

	// Engagement events (Scholar XP)
	bus.Subscribe(event.Type(domain.EventTypeEngagement), h.HandleEngagement)

	// Quest events
	bus.Subscribe(event.Type(domain.EventTypeQuestClaimed), h.HandleQuestClaimed)

	// Prediction events
	bus.Subscribe(event.Type(domain.EventTypePredictionParticipated), h.HandlePredictionParticipated)

	// Item usage events (Rare Candy)
	bus.Subscribe(event.Type(domain.EventTypeItemUsed), h.HandleItemUsed)
}

// HandleItemUpgraded handles item upgrade events to award Blacksmith XP
func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[crafting.ItemUpgradedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	metadata := domain.JobXPMetadata{
		Source:       SourceUpgrade,
		ItemName:     payload.ItemName,
		Quantity:     payload.Quantity,
		IsMasterwork: payload.IsMasterwork,
	}

	return h.handleBlacksmithXP(ctx, payload.UserID, payload.Quantity, SourceUpgrade, metadata)
}

// HandleItemDisassembled handles item disassemble events to award Blacksmith XP
func (h *EventHandler) HandleItemDisassembled(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[crafting.ItemDisassembledPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item disassembled payload: %w", err)
	}

	metadata := domain.JobXPMetadata{
		Source:           SourceDisassemble,
		ItemName:         payload.ItemName,
		Quantity:         payload.Quantity,
		IsPerfectSalvage: payload.IsPerfectSalvage,
	}

	return h.handleBlacksmithXP(ctx, payload.UserID, payload.Quantity, SourceDisassemble, metadata)
}

func (h *EventHandler) handleBlacksmithXP(ctx context.Context, userID string, quantity int, source string, metadata domain.JobXPMetadata) error {
	totalXP := BlacksmithXPPerItem * quantity
	return h.awardXPAndLog(ctx, userID, JobKeyBlacksmith, totalXP, source, metadata, "blacksmithing")
}

// HandleSlotsCompleted handles slots completion events to award Gambler XP
func (h *EventHandler) HandleSlotsCompleted(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.SlotsCompletedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode slots completed payload: %w", err)
	}

	metadata := domain.JobXPMetadata{
		Source:       SourceSlots,
		BetAmount:    payload.BetAmount,
		PayoutAmount: payload.PayoutAmount,
		TriggerType:  payload.TriggerType,
	}

	// Base XP: betAmount / 10
	xp := metadata.BetAmount / 10
	// Win bonus
	if metadata.PayoutAmount > metadata.BetAmount {
		xp += GamblerWinBonus
	}
	// Jackpot bonus
	if metadata.TriggerType == "jackpot" || metadata.TriggerType == "mega_jackpot" {
		xp += 100
	}

	if xp <= 0 {
		return nil
	}

	return h.awardXPAndLog(ctx, payload.UserID, JobKeyGambler, xp, SourceSlots, metadata, "slots")
}

// HandleHarvestCompleted handles harvest completion events to award Farmer XP
func (h *EventHandler) HandleHarvestCompleted(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.HarvestCompletedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode harvest completed payload: %w", err)
	}

	if payload.XPAmount <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:       SourceHarvest,
		HoursElapsed: payload.HoursElapsed,
		HoursWaited:  payload.HoursElapsed, // Backwards compatibility for now
		Spoiled:      payload.Spoiled,
	}

	return h.awardXPAndLog(ctx, payload.UserID, JobKeyFarmer, payload.XPAmount, SourceHarvest, metadata, "harvest")
}

// HandleCompostHarvested handles compost harvest events to award Farmer XP
func (h *EventHandler) HandleCompostHarvested(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.CompostHarvestedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode compost harvested payload: %w", err)
	}

	if payload.XPAmount <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:     SourceCompostHarvest,
		InputValue: payload.InputValue,
		IsSludge:   payload.IsSludge,
	}

	return h.awardXPAndLog(ctx, payload.UserID, JobKeyFarmer, payload.XPAmount, SourceCompostHarvest, metadata, LogSourceCompost)
}

// HandleExpeditionRewarded handles expedition reward events to award job XP
func (h *EventHandler) HandleExpeditionRewarded(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.ExpeditionRewardedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode expedition rewarded payload: %w", err)
	}

	for jobKey, xpAmount := range payload.JobXP {
		if xpAmount <= 0 {
			continue
		}

		metadata := domain.JobXPMetadata{
			Source:       SourceExpedition,
			ExpeditionID: payload.ExpeditionID,
		}

		result, err := h.service.AwardXP(ctx, payload.UserID, jobKey, xpAmount, SourceExpedition, metadata)
		if err != nil {
			log.Warn("Failed to award XP for expedition", "error", err, "user_id", payload.UserID, "job", jobKey)
			continue
		}

		if result != nil && result.LeveledUp {
			log.Info("Job leveled up from expedition!", "user_id", payload.UserID, "job", jobKey, "new_level", result.NewLevel)
		}
	}

	return nil
}

// HandleGambleParticipated handles gamble participation events (start/join) to award Gambler XP
func (h *EventHandler) HandleGambleParticipated(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.GambleParticipatedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode gamble participated payload: %w", err)
	}

	xp := payload.LootboxCount * GamblerXPPerLootbox
	if xp <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:   payload.Source,
		GambleID: payload.GambleID,
		Quantity: payload.LootboxCount, // lootbox_count -> quantity
	}

	return h.awardXPAndLog(ctx, payload.UserID, JobKeyGambler, xp, payload.Source, metadata, "gambling")
}

// HandleGambleCompleted handles gamble completion events to award Gambler win XP
func (h *EventHandler) HandleGambleCompleted(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.GambleCompletedPayloadV2](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode gamble completed payload: %w", err)
	}

	for _, participant := range payload.Participants {
		if !participant.IsWinner {
			continue
		}

		xp := GamblerWinBonus
		metadata := domain.JobXPMetadata{
			Source:   SourceGambleWin,
			GambleID: payload.GambleID,
		}

		result, err := h.service.AwardXP(ctx, participant.UserID, JobKeyGambler, xp, SourceGambleWin, metadata)
		if err != nil {
			log.Warn("Failed to award Gambler win XP", "error", err, "user_id", participant.UserID)
			continue
		}

		if result != nil && result.LeveledUp {
			log.Info("Gambler leveled up from win!", "user_id", participant.UserID, "new_level", result.NewLevel)
		}
	}

	return nil
}

// HandleItemSold handles item sold events to award Merchant XP
func (h *EventHandler) HandleItemSold(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.ItemSoldPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item sold payload: %w", err)
	}

	return h.handleMerchantXP(ctx, payload.UserID, payload.ItemName, payload.TotalValue, SourceSell)
}

// HandleItemBought handles item bought events to award Merchant XP
func (h *EventHandler) HandleItemBought(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.ItemBoughtPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item bought payload: %w", err)
	}

	return h.handleMerchantXP(ctx, payload.UserID, payload.ItemName, payload.TotalValue, SourceBuy)
}

func (h *EventHandler) handleMerchantXP(ctx context.Context, userID, itemName string, totalValue int, action string) error {
	log := logger.FromContext(ctx)

	xp := int(math.Ceil(float64(totalValue) / MerchantXPValueDivisor))
	if xp <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:   action,
		ItemName: itemName,
		Value:    totalValue,
	}

	result, err := h.service.AwardXP(ctx, userID, JobKeyMerchant, xp, action, metadata)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to award Merchant XP for %s", action), "error", err, "user_id", userID)
		return nil
	}

	if result != nil && result.LeveledUp {
		log.Info(fmt.Sprintf("Merchant leveled up from %s!", action), "user_id", userID, "new_level", result.NewLevel)
	}

	return nil
}

// HandleSearchPerformed handles search performed events to award Explorer XP
func (h *EventHandler) HandleSearchPerformed(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.SearchPerformedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode search performed payload: %w", err)
	}

	if payload.XPAmount <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:       SourceSearch,
		IsCritical:   payload.IsCritical,
		IsNearMiss:   payload.IsNearMiss,
		IsFirstDaily: payload.IsFirstDaily,
	}

	return h.awardXPAndLog(ctx, payload.UserID, JobKeyExplorer, payload.XPAmount, SourceSearch, metadata, "search")
}

// HandleEngagement handles engagement events to award Scholar XP
func (h *EventHandler) HandleEngagement(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Engagement events use *domain.EngagementMetric payload
	metric, err := event.DecodePayload[domain.EngagementMetric](evt.Payload)
	if err != nil {
		// Log error if needed, but per previous logic we ignore it if it's strictly a decode issue
		return nil
	}

	if metric.UserID == "" {
		return nil
	}

	result, err := h.service.AwardXP(ctx, metric.UserID, JobKeyScholar, ScholarXPPerEngagement, SourceEngagement, domain.JobXPMetadata{
		Source:     SourceEngagement,
		MetricType: metric.MetricType,
	})
	if err != nil {
		if errors.Is(err, domain.ErrFeatureLocked) {
			return nil // Silently ignore if Scholar is not unlocked
		}
		log.Warn("Failed to award Scholar XP for engagement", "error", err, "user_id", metric.UserID)
		return nil
	}

	if result != nil && result.LeveledUp {
		log.Info("Scholar leveled up!", "user_id", metric.UserID, "new_level", result.NewLevel)
	}

	return nil
}

// HandleQuestClaimed handles quest claimed events to award Merchant XP for quest rewards
func (h *EventHandler) HandleQuestClaimed(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[domain.QuestClaimedPayloadV1](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode quest claimed payload: %w", err)
	}

	if payload.RewardXP <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:   SourceQuest,
		QuestKey: payload.QuestKey,
		QuestID:  payload.QuestID,
	}

	return h.awardXPAndLog(ctx, payload.UserID, JobKeyMerchant, payload.RewardXP, SourceQuest, metadata, LogSourceQuest)
}

func (h *EventHandler) awardXPAndLog(ctx context.Context, userID, jobKey string, xp int, source string, metadata domain.JobXPMetadata, logSource string) error {
	log := logger.FromContext(ctx)

	result, err := h.service.AwardXP(ctx, userID, jobKey, xp, source, metadata)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to award %s XP for %s", jobKey, source), "error", err, "user_id", userID)
		return nil
	}

	if result != nil && result.LeveledUp {
		log.Info(fmt.Sprintf("%s leveled up from %s!", jobKey, logSource), "user_id", userID, "new_level", result.NewLevel)
	}

	return nil
}

// HandlePredictionParticipated handles prediction participation events to award Gambler XP
func (h *EventHandler) HandlePredictionParticipated(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.PredictionParticipantPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode prediction participated payload: %w", err)
	}

	if payload.XP <= 0 {
		return nil
	}

	metadata := domain.JobXPMetadata{
		Source:   SourcePrediction,
		Username: payload.Username,
		IsWinner: payload.IsWinner,
		Platform: payload.Platform,
	}

	userID := payload.UserID
	if userID == "" {
		userID = payload.Username
	}

	result, err := h.service.AwardXP(ctx, userID, JobKeyGambler, payload.XP, SourcePrediction, metadata)
	if err != nil {
		if errors.Is(err, domain.ErrFeatureLocked) {
			return nil // Silently ignore if Gambler is not unlocked
		}
		log.Warn("Failed to award Gambler XP for prediction", "error", err, "user_id", userID)
		return nil
	}

	if result != nil && result.LeveledUp {
		log.Info("Gambler leveled up from prediction!", "user_id", userID, "new_level", result.NewLevel)
	}

	return nil
}

// HandleItemUsed handles item usage events, specifically awarding XP for Rare Candy
func (h *EventHandler) HandleItemUsed(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.ItemUsedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item used payload: %w", err)
	}

	// Currently handles Rare Candy XP awards
	if payload.ItemName == domain.ItemRareCandy {
		jobName, _ := payload.Metadata["job_name"].(string)
		source, ok := payload.Metadata[domain.MetadataKeySource].(string)
		if !ok {
			source = SourceRareCandy
		}

		// Extract XP amount safely from metadata (JSON numbers are float64)
		var xpTotal int
		if val, ok := payload.Metadata["xp_total"]; ok {
			switch v := val.(type) {
			case float64:
				xpTotal = int(v)
			case int:
				xpTotal = v
			case int64:
				xpTotal = int(v)
			}
		}

		metadata := domain.JobXPMetadata{
			Source:  source,
			JobName: jobName,
			XPTotal: xpTotal,
		}

		result, err := h.service.AwardXP(ctx, payload.UserID, jobName, xpTotal, source, metadata)
		if err != nil {
			log.Warn("Failed to award XP from item use", "error", err, "user_id", payload.UserID, "item", payload.ItemName)
			return nil
		}

		if result != nil && result.LeveledUp {
			log.Info("Job leveled up from item use!", "user_id", payload.UserID, "job", jobName, "new_level", result.NewLevel)
		}
	}

	return nil
}

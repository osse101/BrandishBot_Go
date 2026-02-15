package stats

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventHandler handles events related to stats
type EventHandler struct {
	service Service
}

// NewEventHandler creates a new stats event handler
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

	// Gamble events
	bus.Subscribe(event.Type(domain.EventGambleCompleted), h.HandleGambleCompleted)

	// Search events
	bus.Subscribe(event.Type(domain.EventTypeSearchPerformed), h.HandleSearchPerformed)

	// Job events (stats recording for level-up and epiphany)
	bus.Subscribe(event.Type(domain.EventTypeJobLevelUp), h.HandleJobLevelUp)
	bus.Subscribe(event.Type(domain.EventTypeJobXPCritical), h.HandleJobXPCritical)

	// Prediction events
	bus.Subscribe(event.Type(domain.EventTypePredictionParticipated), h.HandlePredictionParticipated)

	// Economy events
	bus.Subscribe(event.Type(domain.EventTypeItemSold), h.HandleItemSold)
	bus.Subscribe(event.Type(domain.EventTypeItemBought), h.HandleItemBought)
}

// HandleItemSold handles item sold events to record stats
func (h *EventHandler) HandleItemSold(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.ItemSoldPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item sold payload: %w", err)
	}

	metadata := map[string]interface{}{
		domain.MetadataKeyItemName: payload.ItemName,
		domain.MetadataKeyQuantity: payload.Quantity,
		"total_value":              payload.TotalValue,
		"category":                 payload.ItemCategory,
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.StatsEventItemSold, metadata); err != nil {
		log.Warn("Failed to record item sold stat", "error", err, "user_id", payload.UserID)
	}
	return nil
}

// HandleItemBought handles item bought events to record stats
func (h *EventHandler) HandleItemBought(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.ItemBoughtPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item bought payload: %w", err)
	}

	metadata := map[string]interface{}{
		domain.MetadataKeyItemName: payload.ItemName,
		domain.MetadataKeyQuantity: payload.Quantity,
		"total_value":              payload.TotalValue,
		"category":                 payload.ItemCategory,
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.StatsEventItemBought, metadata); err != nil {
		log.Warn("Failed to record item bought stat", "error", err, "user_id", payload.UserID)
	}
	return nil
}

// HandleItemUpgraded handles item upgrade events to record stats
func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[crafting.ItemUpgradedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	if payload.IsMasterwork {
		err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeCraftingCriticalSuccess, domain.CraftingMetadata{
			ItemName:         payload.ItemName,
			OriginalQuantity: payload.Quantity,
			MasterworkCount:  1,
			BonusQuantity:    payload.BonusQuantity,
		})
		if err != nil {
			log.Warn("Failed to record crafting critical success stat", "error", err, "user_id", payload.UserID)
		}
	}

	return nil
}

// HandleItemDisassembled handles item disassemble events to record stats
func (h *EventHandler) HandleItemDisassembled(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[crafting.ItemDisassembledPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item disassembled payload: %w", err)
	}

	if payload.IsPerfectSalvage {
		err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeCraftingPerfectSalvage, domain.CraftingMetadata{
			ItemName:     payload.ItemName,
			Quantity:     payload.Quantity,
			PerfectCount: payload.PerfectSalvageCount,
			Multiplier:   payload.Multiplier,
		})
		if err != nil {
			log.Warn("Failed to record crafting perfect salvage stat", "error", err, "user_id", payload.UserID)
		}
	}

	return nil
}

// HandleSlotsCompleted handles slots completion events to record stats
func (h *EventHandler) HandleSlotsCompleted(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.SlotsCompletedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode slots completed payload: %w", err)
	}

	metadata := domain.SlotsMetadata{
		BetAmount:        payload.BetAmount,
		PayoutAmount:     payload.PayoutAmount,
		PayoutMultiplier: payload.PayoutMultiplier,
		NetProfit:        payload.PayoutAmount - payload.BetAmount,
		IsWin:            payload.IsWin,
		IsNearMiss:       payload.IsNearMiss,
		TriggerType:      payload.TriggerType,
		Reel1:            payload.Reel1,
		Reel2:            payload.Reel2,
		Reel3:            payload.Reel3,
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeSlotsSpin, metadata); err != nil {
		log.Warn("Failed to record slots spin stats", "error", err)
	}

	if payload.IsWin {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeSlotsWin, metadata); err != nil {
			log.Warn("Failed to record slots win stats", "error", err)
		}
	}

	if payload.TriggerType == "mega_jackpot" {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeSlotsMegaJackpot, metadata); err != nil {
			log.Warn("Failed to record slots mega jackpot stats", "error", err)
		}
	}

	return nil
}

// HandleGambleCompleted handles gamble completion events to record per-participant stats
func (h *EventHandler) HandleGambleCompleted(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.GambleCompletedPayloadV2](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode gamble completed v2 payload: %w", err)
	}

	for _, p := range payload.Participants {
		if p.IsCritFail {
			_ = h.service.RecordUserEvent(ctx, p.UserID, domain.StatsEventGambleCriticalFail, domain.GambleMetadata{
				GambleID: payload.GambleID,
				Score:    p.Score,
			})
		}
		if p.IsTieBreakLost {
			_ = h.service.RecordUserEvent(ctx, p.UserID, domain.StatsEventGambleTieBreakLost, domain.GambleMetadata{
				GambleID: payload.GambleID,
				Score:    p.Score,
			})
		}
		if p.IsNearMiss {
			if err := h.service.RecordUserEvent(ctx, p.UserID, domain.StatsEventGambleNearMiss, domain.GambleMetadata{
				GambleID:    payload.GambleID,
				Score:       p.Score,
				WinnerScore: payload.TotalValue,
			}); err != nil {
				log.Warn("Failed to record gamble near miss stat", "error", err, "user_id", p.UserID)
			}
		}
	}

	return nil
}

// HandleSearchPerformed handles search performed events to record stats
func (h *EventHandler) HandleSearchPerformed(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.SearchPerformedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode search performed payload: %w", err)
	}

	metadata := domain.SearchMetadata{
		IsCritical:   payload.IsCritical,
		IsNearMiss:   payload.IsNearMiss,
		IsCritFail:   payload.IsCriticalFail,
		IsFirstDaily: payload.IsFirstDaily,
		XPAmount:     payload.XPAmount,
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.StatsEventSearch, metadata); err != nil {
		log.Warn("Failed to record search stat", "error", err, "user_id", payload.UserID)
	}

	if payload.IsCritical {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.StatsEventSearchCriticalSuccess, metadata); err != nil {
			log.Warn("Failed to record search critical success stat", "error", err, "user_id", payload.UserID)
		}
	}

	if payload.IsNearMiss {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.StatsEventSearchNearMiss, metadata); err != nil {
			log.Warn("Failed to record search near miss stat", "error", err, "user_id", payload.UserID)
		}
	}

	if payload.IsCriticalFail {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.StatsEventSearchCriticalFail, metadata); err != nil {
			log.Warn("Failed to record search critical fail stat", "error", err, "user_id", payload.UserID)
		}
	}

	return nil
}

// HandleJobLevelUp handles job level-up events to record stats
func (h *EventHandler) HandleJobLevelUp(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[event.JobLevelUpPayloadV1](evt.Payload)
	if err != nil {
		return nil // Don't fail on type mismatch
	}

	if payload.UserID == "" {
		return nil
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeJobLevelUp, payload); err != nil {
		log.Warn("Failed to record job level-up stat", "error", err, "user_id", payload.UserID)
	}

	return nil
}

// HandleJobXPCritical handles job XP critical (Epiphany) events to record stats
func (h *EventHandler) HandleJobXPCritical(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[event.JobXPCriticalPayloadV1](evt.Payload)
	if err != nil {
		return nil // Don't fail on type mismatch
	}

	if payload.UserID == "" {
		return nil
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventTypeJobXPCritical, payload); err != nil {
		log.Warn("Failed to record job XP critical stat", "error", err, "user_id", payload.UserID)
	}

	return nil
}

// HandlePredictionParticipated handles prediction participation events to record stats
func (h *EventHandler) HandlePredictionParticipated(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.PredictionParticipantPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode prediction participated payload: %w", err)
	}

	metadata := domain.PredictionMetadata{
		Username: payload.Username,
		IsWinner: payload.IsWinner,
		Platform: payload.Platform,
		XP:       payload.XP,
	}

	userID := payload.UserID
	if userID == "" {
		userID = payload.Username
	}

	if err := h.service.RecordUserEvent(ctx, userID, domain.EventType("prediction_participation"), metadata); err != nil {
		log.Warn("Failed to record prediction participated stat", "error", err, "user_id", userID)
	}

	return nil
}

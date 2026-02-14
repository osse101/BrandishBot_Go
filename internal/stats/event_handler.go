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
	bus.Subscribe(event.Type(domain.EventJobLevelUp), h.HandleJobLevelUp)
	bus.Subscribe(event.Type(domain.EventTypeJobXPCritical), h.HandleJobXPCritical)

	// Prediction events
	bus.Subscribe(event.Type(domain.EventTypePredictionParticipated), h.HandlePredictionParticipated)
}

// HandleItemUpgraded handles item upgrade events to record stats
func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[crafting.ItemUpgradedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	if payload.IsMasterwork {
		err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventCraftingCriticalSuccess, map[string]interface{}{
			"item_name":         payload.ItemName,
			"original_quantity": payload.Quantity,
			"masterwork_count":  1,
			"bonus_quantity":    payload.BonusQuantity,
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
		err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventCraftingPerfectSalvage, map[string]interface{}{
			"item_name":     payload.ItemName,
			"quantity":      payload.Quantity,
			"perfect_count": payload.PerfectSalvageCount,
			"multiplier":    payload.Multiplier,
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

	metadata := map[string]interface{}{
		"bet_amount":        payload.BetAmount,
		"payout_amount":     payload.PayoutAmount,
		"payout_multiplier": payload.PayoutMultiplier,
		"net_profit":        payload.PayoutAmount - payload.BetAmount,
		"is_win":            payload.IsWin,
		"is_near_miss":      payload.IsNearMiss,
		"trigger_type":      payload.TriggerType,
		"reel1":             payload.Reel1,
		"reel2":             payload.Reel2,
		"reel3":             payload.Reel3,
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSlotsSpin, metadata); err != nil {
		log.Warn("Failed to record slots spin stats", "error", err)
	}

	if payload.IsWin {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSlotsWin, metadata); err != nil {
			log.Warn("Failed to record slots win stats", "error", err)
		}
	}

	if payload.TriggerType == "mega_jackpot" {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSlotsMegaJackpot, metadata); err != nil {
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
			_ = h.service.RecordUserEvent(ctx, p.UserID, domain.EventGambleCriticalFail, map[string]interface{}{
				"gamble_id": payload.GambleID,
				"score":     p.Score,
			})
		}
		if p.IsTieBreakLost {
			_ = h.service.RecordUserEvent(ctx, p.UserID, domain.EventGambleTieBreakLost, map[string]interface{}{
				"gamble_id": payload.GambleID,
				"score":     p.Score,
			})
		}
		if p.IsNearMiss {
			if err := h.service.RecordUserEvent(ctx, p.UserID, domain.EventGambleNearMiss, map[string]interface{}{
				"gamble_id":    payload.GambleID,
				"score":        p.Score,
				"winner_score": payload.TotalValue,
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

	metadata := map[string]interface{}{
		"is_critical":    payload.IsCritical,
		"is_near_miss":   payload.IsNearMiss,
		"is_crit_fail":   payload.IsCriticalFail,
		"is_first_daily": payload.IsFirstDaily,
		"xp_amount":      payload.XPAmount,
	}

	if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSearch, metadata); err != nil {
		log.Warn("Failed to record search stat", "error", err, "user_id", payload.UserID)
	}

	if payload.IsCritical {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSearchCriticalSuccess, metadata); err != nil {
			log.Warn("Failed to record search critical success stat", "error", err, "user_id", payload.UserID)
		}
	}

	if payload.IsNearMiss {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSearchNearMiss, metadata); err != nil {
			log.Warn("Failed to record search near miss stat", "error", err, "user_id", payload.UserID)
		}
	}

	if payload.IsCriticalFail {
		if err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventSearchCriticalFail, metadata); err != nil {
			log.Warn("Failed to record search critical fail stat", "error", err, "user_id", payload.UserID)
		}
	}

	return nil
}

// HandleJobLevelUp handles job level-up events to record stats
func (h *EventHandler) HandleJobLevelUp(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	metadata, ok := evt.Payload.(map[string]interface{})
	if !ok {
		return nil // Don't fail on type mismatch
	}

	userID, _ := metadata["user_id"].(string)
	if userID == "" {
		return nil
	}

	if err := h.service.RecordUserEvent(ctx, userID, domain.EventJobLevelUp, metadata); err != nil {
		log.Warn("Failed to record job level-up stat", "error", err, "user_id", userID)
	}

	return nil
}

// HandleJobXPCritical handles job XP critical (Epiphany) events to record stats
func (h *EventHandler) HandleJobXPCritical(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Epiphany events use a map payload
	metadata, ok := evt.Payload.(map[string]interface{})
	if !ok {
		return nil // Don't fail on type mismatch
	}

	userID, _ := metadata["user_id"].(string)
	if userID == "" {
		return nil
	}

	if err := h.service.RecordUserEvent(ctx, userID, domain.EventJobXPCritical, metadata); err != nil {
		log.Warn("Failed to record job XP critical stat", "error", err, "user_id", userID)
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

	metadata := map[string]interface{}{
		"username":  payload.Username,
		"is_winner": payload.IsWinner,
		"platform":  payload.Platform,
		"xp":        payload.XP,
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

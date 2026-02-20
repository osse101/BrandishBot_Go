package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type UseItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
	TargetUser string `json:"target_user,omitempty" validate:"omitempty,max=100,excludesall=\x00\n\r\t"`
}

type UseItemResponse struct {
	Message string `json:"message"`
}

// mapItemToProgressionNode maps internal item names to progression node keys
func mapItemToProgressionNode(itemName string) string {
	mapping := map[string]string{
		// Weapons
		"weapon_missile":     progression.ItemWeaponMissile,
		"item_grenade":       progression.ItemGrenade,
		"explosive_tnt":      progression.ItemTnt,
		"weapon_hugeblaster": progression.ItemHugemissile,

		// Defense
		"item_shield":   progression.ItemShield,
		"weapon_mirror": progression.ItemWeaponMirror,

		// Recovery
		"revive_small": progression.ItemRevives,

		// Progression
		"xp_rarecandy": progression.ItemXpRarecandy,

		// Lootboxes
		"lootbox_tier0": progression.ItemLootbox0,
		"lootbox_tier1": progression.ItemLootbox1,
		"lootbox_tier2": progression.ItemLootbox2,
		"lootbox_tier3": progression.ItemLootbox3,

		// Utilities
		"item_shovel":       progression.ItemShovel,
		"item_stick":        progression.ItemStick,
		"item_video_filter": progression.ItemVideoFilter,

		// Passive items (economy checks these separately)
		"item_scrap":  progression.ItemScrap,
		"item_script": progression.ItemScript,
	}
	return mapping[itemName]
}

// HandleUseItem handles using an item
// @Summary Use item
// @Description Use an item from inventory
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body UseItemRequest true "Usage details"
// @Success 200 {object} UseItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Item locked by progression"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/item/use [post]
func HandleUseItem(svc user.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UseItemRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Use item"); err != nil {
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		log := logger.FromContext(r.Context())

		// Check if item is progression-locked
		if progressionSvc != nil {
			// Map item internal name to progression node key
			nodeKey := mapItemToProgressionNode(req.ItemName)
			if nodeKey != "" {
				if CheckFeatureLocked(w, r, progressionSvc, nodeKey) {
					return // CheckFeatureLocked already wrote 403 response
				}
			}
		}

		message, err := svc.UseItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity, req.TargetUser)
		if err != nil {
			log.Error("Failed to use item", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item used successfully",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity,
			"message", message)

		// Track engagement for item usage

		// Record contribution for item usage
		if err := progressionSvc.RecordEngagement(r.Context(), req.Username, "item_used", req.Quantity); err != nil {
			log.Error("Failed to record use engagement", "error", err)
			// Don't fail the request
		}
		if userID, err := svc.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
			middleware.TrackEngagementFromContext(
				middleware.WithUserID(r.Context(), userID),
				eventBus,
				"item_used",
				req.Quantity,
			)
		}

		// Publish item.used event
		if err := PublishEvent(r.Context(), eventBus, "item.used", map[string]interface{}{
			"user_id":  req.Username,
			"item":     req.ItemName,
			"quantity": req.Quantity,
			"target":   req.TargetUser,
			"result":   message,
		}); err != nil {
			_ = err // Error already logged in PublishEvent
		}

		respondJSON(w, http.StatusOK, UseItemResponse{
			Message: message,
		})
	}
}

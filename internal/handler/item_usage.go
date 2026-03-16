package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
	Quantity   int    `json:"quantity" validate:"omitempty,min=1,max=10000"`
	TargetUser string `json:"target_user,omitempty" validate:"omitempty,max=100,excludesall=\x00\n\r\t"`
}

type UseItemResponse struct {
	Message string `json:"message"`
}

var itemToProgressionNodeMap = map[string]string{
	// Weapons
	domain.ItemMissile:     progression.ItemWeaponMissile,
	domain.ItemGrenade:     progression.ItemGrenade,
	domain.ItemTNT:         progression.ItemTnt,
	domain.ItemHugeMissile: progression.ItemHugemissile,

	// Defense
	domain.ItemShield:       progression.ItemShield,
	domain.ItemMirrorShield: progression.ItemWeaponMirror,

	// Recovery
	domain.ItemReviveSmall: progression.ItemRevives,

	// Progression
	domain.ItemRareCandy: progression.ItemXpRarecandy,

	// Lootboxes
	domain.ItemLootbox0: progression.ItemLootbox0,
	domain.ItemLootbox1: progression.ItemLootbox1,
	domain.ItemLootbox2: progression.ItemLootbox2,
	domain.ItemLootbox3: progression.ItemLootbox3,

	// Utilities
	domain.ItemShovel:      progression.ItemShovel,
	domain.ItemStick:       progression.ItemStick,
	domain.ItemVideoFilter: progression.ItemVideoFilter,

	// Passive items (economy checks these separately)
	domain.ItemScrap:  progression.ItemScrap,
	domain.ItemScript: progression.ItemScript,
}

// mapItemToProgressionNode maps internal item names to progression node keys
func mapItemToProgressionNode(itemName string) string {
	return itemToProgressionNodeMap[itemName]
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
			// Resolve the name to internal name first for progression check
			internalName := req.ItemName
			if item, err := svc.GetItemByName(r.Context(), req.ItemName); err == nil && item != nil {
				internalName = item.InternalName
			}

			// Map item internal name to progression node key
			nodeKey := mapItemToProgressionNode(internalName)
			if nodeKey != "" {
				if CheckFeatureLocked(w, r, progressionSvc, nodeKey) {
					return // CheckFeatureLocked already wrote 403 response
				}
			}
		}

		message, err := svc.UseItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity, req.TargetUser)
		if err != nil {
			log.Error("Failed to use item", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		// Determine engagement points
		engagementPoints := req.Quantity
		switch req.ItemName {
		case domain.ItemLootbox0, domain.ItemLootbox1, domain.ItemLootbox2, domain.ItemLootbox3:
			engagementPoints = 0
		}

		// Attempt to resolve the correct UUID for metrics/events
		metricUserID := req.Username
		if userID, err := svc.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
			metricUserID = userID

			if engagementPoints > 0 {
				middleware.TrackEngagementFromContext(
					middleware.WithUserID(r.Context(), userID),
					eventBus,
					domain.MetricTypeItemUsed,
					engagementPoints,
				)
			}
		} else {
			log.Warn("Could not resolve UUID for item usage metrics, using username", "username", req.Username, "error", err)
		}

		log.Info("Item used successfully",
			"user_id", metricUserID,
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity,
			"message", message)

		RespondJSON(w, http.StatusOK, UseItemResponse{
			Message: message,
		})
	}
}

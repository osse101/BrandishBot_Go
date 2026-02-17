package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type GiveItemRequest struct {
	OwnerPlatform    string `json:"owner_platform" validate:"required,platform"`
	OwnerPlatformID  string `json:"owner_platform_id" validate:"required"`
	Owner            string `json:"owner" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ReceiverPlatform string `json:"receiver_platform" validate:"required,platform"`
	Receiver         string `json:"receiver" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName         string `json:"item_name" validate:"required,max=100"`
	Quantity         int    `json:"quantity" validate:"min=1,max=10000"`
}

// HandleGiveItem handles transferring items between users
// @Summary Give item to another user
// @Description Transfer an item from one user's inventory to another user.
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body GiveItemRequest true "Transfer details including owner and receiver info"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse "Invalid request or self-gifting attempt"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/user/item/give [post]
func HandleGiveItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GiveItemRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Give item"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		// Check for self-gifting (same platform and same username)
		if req.OwnerPlatform == req.ReceiverPlatform &&
			(req.Owner == req.Receiver || req.OwnerPlatformID == req.Receiver) {
			log.Info("Self-gifting attempt detected", "user", req.Owner)
			respondError(w, http.StatusBadRequest, "You can't give items to yourself! Nice try though.")
			return
		}

		if err := svc.GiveItem(r.Context(), req.OwnerPlatform, req.OwnerPlatformID, req.Owner, req.ReceiverPlatform, req.Receiver, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to give item", "error", err, "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item transferred successfully", "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgItemTransferredSuccess})
	}
}

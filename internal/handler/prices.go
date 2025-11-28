package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// HandleGetPrices handles getting item prices
// @Summary Get item prices
// @Description Get current sell prices for items
// @Tags economy
// @Produce json
// @Success 200 {array} domain.Item
// @Failure 500 {object} ErrorResponse
// @Router /prices [get]
func HandleGetPrices(svc economy.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		items, err := svc.GetSellablePrices(r.Context())
		if err != nil {
			log.Error("Failed to get sellable prices", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Sellable prices retrieved", "count", len(items))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, items)
	}
}

package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// HandleGetBuyPrices handles getting item buy prices
// @Summary Get item buy prices
// @Description Get current buy prices for items
// @Tags economy
// @Produce json
// @Success 200 {array} domain.Item
// @Failure 500 {object} ErrorResponse
// @Router /prices/buy [get]
func HandleGetBuyPrices(svc economy.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		items, err := svc.GetBuyablePrices(r.Context())
		if err != nil {
			log.Error("Failed to get buyable prices", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Buyable prices retrieved", "count", len(items))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, items)
	}
}

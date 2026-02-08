package handler

import (
	"context"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
// @Router /api/v1/prices [get]
func HandleGetPrices(svc economy.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleGetPricesInternal(w, r, svc.GetSellablePrices, "sellable")
	}
}

// HandleGetBuyPrices handles getting item buy prices
// @Summary Get item buy prices
// @Description Get current buy prices for items
// @Tags economy
// @Produce json
// @Success 200 {array} domain.Item
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/prices/buy [get]
func HandleGetBuyPrices(svc economy.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleGetPricesInternal(w, r, svc.GetBuyablePrices, "buyable")
	}
}

func handleGetPricesInternal(w http.ResponseWriter, r *http.Request, fetcher func(context.Context) ([]domain.Item, error), label string) {
	log := logger.FromContext(r.Context())

	items, err := fetcher(r.Context())
	if err != nil {
		log.Error("Failed to get "+label+" prices", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	log.Info(label+" prices retrieved", "count", len(items))

	respondJSON(w, http.StatusOK, items)
}

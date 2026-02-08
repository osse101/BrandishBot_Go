package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/prediction"
)

// PredictionHandlers handles prediction-related HTTP requests
type PredictionHandlers struct {
	service prediction.Service
}

// NewPredictionHandlers creates a new prediction handlers instance
func NewPredictionHandlers(service prediction.Service) *PredictionHandlers {
	return &PredictionHandlers{
		service: service,
	}
}

// HandleProcessOutcome processes a prediction outcome
// @Summary Process prediction outcome
// @Description Convert channel points to progression contribution and award XP to participants
// @Tags prediction
// @Accept json
// @Produce json
// @Param request body domain.PredictionOutcomeRequest true "Prediction outcome data"
// @Success 200 {object} domain.PredictionResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/prediction [post]
func (h *PredictionHandlers) HandleProcessOutcome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.PredictionOutcomeRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Process prediction outcome"); err != nil {
			return
		}

		result, err := h.service.ProcessOutcome(r.Context(), &req)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to process prediction outcome")
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

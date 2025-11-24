package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

type DisassembleItemRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
}

type DisassembleItemResponse struct {
	Outputs           map[string]int `json:"outputs"`
	QuantityProcessed int            `json:"quantity_processed"`
}

func HandleDisassembleItem(svc crafting.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		
		var req DisassembleItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode disassemble item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		log.Debug("Disassemble item request",
			"username", req.Username,
			"item", req.Item,
			"quantity", req.Quantity)

		if req.Username == "" || req.Item == "" || req.Quantity <= 0 {
			log.Warn("Invalid disassemble item request")
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		outputs, quantityProcessed, err := svc.DisassembleItem(r.Context(), req.Username, req.Platform, req.Item, req.Quantity)
		if err != nil {
			log.Error("Failed to disassemble item", "error", err, "username", req.Username, "item", req.Item)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Item disassembled successfully",
			"username", req.Username,
			"item", req.Item,
			"quantity_processed", quantityProcessed,
			"outputs", outputs)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DisassembleItemResponse{
			Outputs:           outputs,
			QuantityProcessed: quantityProcessed,
		})
	}
}

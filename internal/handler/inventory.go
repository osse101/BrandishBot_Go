package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/user"
)

type AddItemRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

func HandleAddItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		if err := svc.AddItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Item added successfully"))
	}
}

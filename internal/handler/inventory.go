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

type RemoveItemRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type RemoveItemResponse struct {
	Removed int `json:"removed"`
}

func HandleRemoveItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RemoveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		removed, err := svc.RemoveItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RemoveItemResponse{Removed: removed})
	}
}

type GiveItemRequest struct {
	Owner    string `json:"owner"`
	Receiver string `json:"receiver"`
	Platform string `json:"platform"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

func HandleGiveItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GiveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Owner == "" || req.Receiver == "" || req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		if err := svc.GiveItem(r.Context(), req.Owner, req.Receiver, req.Platform, req.ItemName, req.Quantity); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Item transferred successfully"))
	}
}

type SellItemRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type SellItemResponse struct {
	MoneyGained int `json:"money_gained"`
	ItemsSold   int `json:"items_sold"`
}

func HandleSellItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SellItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		moneyGained, itemsSold, err := svc.SellItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SellItemResponse{
			MoneyGained: moneyGained,
			ItemsSold:   itemsSold,
		})
	}
}

type BuyItemRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type BuyItemResponse struct {
	ItemsBought int `json:"items_bought"`
}

func HandleBuyItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BuyItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		bought, err := svc.BuyItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BuyItemResponse{
			ItemsBought: bought,
		})
	}
}

type UseItemRequest struct {
	Username       string `json:"username"`
	Platform       string `json:"platform"`
	ItemName       string `json:"item_name"`
	Quantity       int    `json:"quantity"`
	TargetUsername string `json:"target_username"`
}

type UseItemResponse struct {
	Message string `json:"message"`
}

func HandleUseItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UseItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		if req.Username == "" || req.ItemName == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		message, err := svc.UseItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity, req.TargetUsername)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UseItemResponse{
			Message: message,
		})
	}
}

type GetInventoryResponse struct {
	Items []user.UserInventoryItem `json:"items"`
}

func HandleGetInventory(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Missing username query parameter", http.StatusBadRequest)
			return
		}

		items, err := svc.GetInventory(r.Context(), username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GetInventoryResponse{
			Items: items,
		})
	}
}

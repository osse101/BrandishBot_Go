package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/logger"
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
		log := logger.FromContext(r.Context())
		
		var req AddItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode add item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		log.Debug("Add item request",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity)

		// Validate platform
		if req.Platform != "" {
			if err := ValidatePlatform(req.Platform); err != nil {
				log.Warn("Invalid platform", "platform", req.Platform)
				http.Error(w, "Invalid platform", http.StatusBadRequest)
				return
			}
		}

		// Validate username
		if err := ValidateUsername(req.Username); err != nil {
			log.Warn("Invalid username", "error", err)
			http.Error(w, "Invalid username", http.StatusBadRequest)
			return
		}

		// Validate item name
		if err := ValidateItemName(req.ItemName); err != nil {
			log.Warn("Invalid item name", "error", err)
			http.Error(w, "Invalid item name", http.StatusBadRequest)
			return
		}

		// Validate quantity
		if err := ValidateQuantity(req.Quantity); err != nil {
			log.Warn("Invalid quantity", "quantity", req.Quantity, "error", err)
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		if err := svc.AddItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to add item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, "Failed to add item", http.StatusInternalServerError) // Generic error
			return
		}
		
		log.Info("Item added successfully", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

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
		log := logger.FromContext(r.Context())
		
		var req RemoveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode remove item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		log.Debug("Remove item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			log.Warn("Invalid remove item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		removed, err := svc.RemoveItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to remove item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Item removed successfully", "username", req.Username, "item", req.ItemName, "removed", removed)

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
		log := logger.FromContext(r.Context())
		
		var req GiveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode give item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		log.Debug("Give item request",
			"owner", req.Owner,
			"receiver", req.Receiver,
			"item", req.ItemName,
			"quantity", req.Quantity)

		if req.Owner == "" || req.Receiver == "" || req.ItemName == "" || req.Quantity <= 0 {
			log.Warn("Invalid give item request")
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		if err := svc.GiveItem(r.Context(), req.Owner, req.Receiver, req.Platform, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to give item", "error", err, "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Item transferred successfully", "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName, "quantity", req.Quantity)

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

func HandleSellItem(svc economy.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		
		var req SellItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode sell item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		log.Debug("Sell item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			log.Warn("Invalid sell item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		moneyGained, itemsSold, err := svc.SellItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to sell item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Item sold successfully",
			"username", req.Username,
			"item", req.ItemName,
			"items_sold", itemsSold,
			"money_gained", moneyGained)

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

func HandleBuyItem(svc economy.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())
		
		var req BuyItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode buy item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		log.Debug("Buy item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		if req.Username == "" || req.ItemName == "" || req.Quantity <= 0 {
			log.Warn("Invalid buy item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		bought, err := svc.BuyItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to buy item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Item purchased successfully",
			"username", req.Username,
			"item", req.ItemName,
			"items_bought", bought)

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
		log := logger.FromContext(r.Context())
		
		var req UseItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode use item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}
		
		log.Debug("Use item request",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity,
			"target", req.TargetUsername)

		if req.Username == "" || req.ItemName == "" {
			log.Warn("Missing required fields", "username", req.Username, "item", req.ItemName)
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		message, err := svc.UseItem(r.Context(), req.Username, req.Platform, req.ItemName, req.Quantity, req.TargetUsername)
		if err != nil {
			log.Error("Failed to use item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Item used successfully",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity,
			"message", message)

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
		log := logger.FromContext(r.Context())
		
		username := r.URL.Query().Get("username")
		if username == "" {
			log.Warn("Missing username query parameter")
			http.Error(w, "Missing username query parameter", http.StatusBadRequest)
			return
		}
		
		log.Debug("Get inventory request", "username", username)

		items, err := svc.GetInventory(r.Context(), username)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "username", username)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		log.Info("Inventory retrieved", "username", username, "item_count", len(items))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GetInventoryResponse{
			Items: items,
		})
	}
}

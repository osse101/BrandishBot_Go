package domain

import "time"

// HarvestState represents a user's harvest state
type HarvestState struct {
	UserID          string    `json:"user_id"`
	LastHarvestedAt time.Time `json:"last_harvested_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// HarvestReward represents a tier of rewards based on time elapsed
type HarvestReward struct {
	MaxHours       float64           // Maximum hours for this tier
	Items          map[string]int    // Item name -> quantity
	RequiresUnlock map[string]bool   // Item name -> requires progression unlock
}

// HarvestResponse represents the API response for harvest operation
type HarvestResponse struct {
	ItemsGained          map[string]int `json:"items_gained"`
	HoursSinceHarvest    float64        `json:"hours_since_harvest"`
	NextHarvestAt        time.Time      `json:"next_harvest_at"`
	Message              string         `json:"message"`
}

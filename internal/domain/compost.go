package domain

import "time"

// CompostBinStatus represents the state of a compost bin
type CompostBinStatus string

const (
	CompostBinStatusIdle       CompostBinStatus = "idle"
	CompostBinStatusComposting CompostBinStatus = "composting"
	CompostBinStatusReady      CompostBinStatus = "ready"
	CompostBinStatusSludge     CompostBinStatus = "sludge"
)

// CompostBin represents a user's compost bin
type CompostBin struct {
	ID           string           `json:"id"`
	UserID       string           `json:"user_id"`
	Status       CompostBinStatus `json:"status"`
	Capacity     int              `json:"capacity"`
	Items        []CompostBinItem `json:"items"`
	ItemCount    int              `json:"item_count"`
	StartedAt    *time.Time       `json:"started_at,omitempty"`
	ReadyAt      *time.Time       `json:"ready_at,omitempty"`
	SludgeAt     *time.Time       `json:"sludge_at,omitempty"`
	InputValue   int              `json:"input_value"`
	DominantType string           `json:"dominant_type"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// CompostBinItem is a snapshot of an item deposited into the bin
type CompostBinItem struct {
	ItemID       int          `json:"item_id"`
	ItemName     string       `json:"item_name"`
	Quantity     int          `json:"quantity"`
	QualityLevel QualityLevel `json:"quality_level"`
	BaseValue    int          `json:"base_value"`
	ContentTypes []string     `json:"content_types"`
}

// CompostOutput holds the result of a harvest
type CompostOutput struct {
	Items      map[string]int `json:"items"`
	IsSludge   bool           `json:"is_sludge"`
	TotalValue int            `json:"total_value"`
	Message    string         `json:"message"`
}

// CompostStatusResponse is returned when bin is not ready to harvest
type CompostStatusResponse struct {
	Status    CompostBinStatus `json:"status"`
	Capacity  int              `json:"capacity"`
	ItemCount int              `json:"item_count"`
	Items     []CompostBinItem `json:"items"`
	ReadyAt   *time.Time       `json:"ready_at,omitempty"`
	SludgeAt  *time.Time       `json:"sludge_at,omitempty"`
	TimeLeft  string           `json:"time_left,omitempty"`
}

// HarvestResult is returned by Service.Harvest - either status info or actual output
type HarvestResult struct {
	Harvested bool                   `json:"harvested"`
	Output    *CompostOutput         `json:"output,omitempty"`
	Status    *CompostStatusResponse `json:"status,omitempty"`
}

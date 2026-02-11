package compost

import "time"

// Compost logic constants
const (
	WarmupDuration    = 1 * time.Hour
	PerItemDuration   = 30 * time.Minute
	SludgeTimeout     = 168 * time.Hour // 1 week
	DefaultCapacity   = 5
	DefaultMultiplier = 0.5
)

// User-facing messages
const (
	MsgBinEmpty        = "Bin is empty. Deposit items to start composting!"
	MsgReadyNow        = "ready now"
	MsgHarvestSludge   = "Your compost sat too long and turned to sludge!"
	MsgHarvestComplete = "Composting complete!"
	MsgHarvestFallback = "Composting complete! (converted to money)"
)

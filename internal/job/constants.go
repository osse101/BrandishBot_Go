package job

import "github.com/osse101/BrandishBot_Go/internal/domain"

// XP formula constants
const (
	// BaseXP is the base XP value used in level calculations
	BaseXP = 250.0

	// LevelExponent is the exponent used in the XP formula: XP = BaseXP * (Level ^ LevelExponent)
	LevelExponent = 1.3

	// MaxIterationLevel is the maximum level to iterate to when calculating levels
	MaxIterationLevel = 100
)

// Default system values
const (
	// DefaultMaxLevel is the default maximum job level when progression system isn't available
	DefaultMaxLevel = 10

	// DefaultXPMultiplier is the default XP multiplier when no boost is active
	DefaultXPMultiplier = 1.0

	// DefaultDailyCap is the base daily XP cap per job
	DefaultDailyCap = 500
)

// Job keys for referencing specific jobs
const (
	JobKeyBlacksmith = domain.JobKeyBlacksmith
	JobKeyExplorer   = domain.JobKeyExplorer
	JobKeyMerchant   = domain.JobKeyMerchant
	JobKeyGambler    = domain.JobKeyGambler
	JobKeyFarmer     = domain.JobKeyFarmer
	JobKeyScholar    = domain.JobKeyScholar
)

// XP award amounts for different actions
const (
	// Blacksmith XP awards
	BlacksmithXPPerItem = 200

	// Gambler XP awards
	GamblerXPPerLootbox = 50
	GamblerWinBonus     = 50

	// Explorer XP awards
	ExplorerXPPerItem = 45

	// Merchant XP awards (value-based)
	MerchantXPValueDivisor = 2.5 // XP = ceil(transactionValue / divisor)
	MerchantBonusPerLevel  = 0.5 // 0.5% price adjustment per level

	// Scholar XP awards
	ScholarXPPerEngagement = 2
)

// XP source types for tracking and special behavior
const (
	SourceEngagement     = "engagement"        // Engagement XP
	SourceSearch         = domain.ActionSearch // Search XP
	SourceRareCandy      = "rarecandy"         // Rare candy usage - bypasses daily cap
	SourceHarvest        = "harvest"           // Harvest XP - bypasses daily cap
	SourcePrediction     = "prediction"        // Prediction XP
	SourceQuest          = "quest"             // Quest XP
	SourceUpgrade        = "upgrade"           // Item upgrade XP
	SourceDisassemble    = "disassemble"       // Item disassemble XP
	SourceSlots          = "slots"             // Slots XP
	SourceCompostHarvest = "compost_harvest"   // Compost harvest XP
	SourceExpedition     = "expedition"        // Expedition XP
	SourceGambleWin      = "win"               // Gamble win XP
	SourceSell           = "sell"              // Item sell XP
	SourceBuy            = "buy"               // Item buy XP
)

// Log source constants for better tracking in logs
const (
	LogSourceCompost = "compost"
	LogSourceQuest   = "quest"
)

// Job Epiphany constants
const (
	EpiphanyChance     = 0.05 // 5% chance
	EpiphanyMultiplier = 2.0  // Double XP
)

// Info represents basic information about a job for display/autocomplete purposes
type Info struct {
	Key         string
	DisplayName string
}

// AllJobs is a list of all available jobs in the system
var AllJobs = []Info{
	{Key: JobKeyBlacksmith, DisplayName: "Blacksmith (Crafting)"},
	{Key: JobKeyExplorer, DisplayName: "Explorer (Exploration)"},
	{Key: JobKeyMerchant, DisplayName: "Merchant (Economy)"},
	{Key: JobKeyGambler, DisplayName: "Gambler (Gambling)"},
	{Key: JobKeyFarmer, DisplayName: "Farmer (Farming)"},
	{Key: JobKeyScholar, DisplayName: "Scholar (Community)"},
}

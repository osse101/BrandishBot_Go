package job

// XP formula constants
const (
	// BaseXP is the base XP value used in level calculations
	BaseXP = 100.0

	// LevelExponent is the exponent used in the XP formula: XP = BaseXP * (Level ^ LevelExponent)
	LevelExponent = 1.5

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
	JobKeyBlacksmith = "blacksmith"
	JobKeyExplorer   = "explorer"
	JobKeyMerchant   = "merchant"
	JobKeyGambler    = "gambler"
	JobKeyFarmer     = "farmer"
	JobKeyScholar    = "scholar"
)

// XP award amounts for different actions
const (
	// Blacksmith XP awards
	BlacksmithXPPerItem = 10

	// Gambler XP awards
	GamblerXPPerLootbox = 20
	GamblerWinBonus     = 50

	// Explorer XP awards
	ExplorerXPPerItem = 10

	// Merchant XP awards (value-based)
	MerchantXPValueDivisor = 10.0 // XP = ceil(transactionValue / divisor)
	MerchantBonusPerLevel  = 0.5  // 0.5% price adjustment per level

	// Scholar XP awards
	ScholarXPPerEngagement = 5
	ScholarBonusPerLevel   = 10.0 // 10% engagement value increase per level
)

// XP source types for tracking and special behavior
const (
	SourceRareCandy = "rarecandy" // Rare candy usage - bypasses daily cap
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
	{Key: JobKeyExplorer, DisplayName: "Explorer (Digging)"},
	{Key: JobKeyMerchant, DisplayName: "Merchant (Economy)"},
	{Key: JobKeyGambler, DisplayName: "Gambler (Gambling)"},
	{Key: JobKeyFarmer, DisplayName: "Farmer (TBD)"},
	{Key: JobKeyScholar, DisplayName: "Scholar (Community)"},
}

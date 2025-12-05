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
)

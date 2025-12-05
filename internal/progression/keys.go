package progression

// Feature and item keys used throughout the progression system
// These constants prevent typos and provide a single source of truth
const (
	// System
	FeatureProgressionSystem = "progression_system"

	// Items
	ItemMoney    = "item_money"
	ItemLootbox0 = "item_lootbox0"
	ItemLootbox1 = "item_lootbox1"

	// Core Features
	FeatureEconomy     = "feature_economy"
	FeatureUpgrade     = "feature_upgrade"
	FeatureDisassemble = "feature_disassemble"
	FeatureSearch      = "feature_search"

	// Economy Sub-features
	FeatureBuy  = "feature_buy"
	FeatureSell = "feature_sell"

	// Advanced Features
	FeatureGamble     = "feature_gamble"
	FeatureDuel       = "feature_duel"
	FeatureExpedition = "feature_expedition"

	// Job System
	FeatureJobsXP      = "feature_jobs_xp"
	JobBlacksmith      = "job_blacksmith"
	JobExplorer        = "job_explorer"
	JobMerchant        = "job_merchant"
	JobGambler         = "job_gambler"
	JobFarmer          = "job_farmer"
	JobScholar         = "job_scholar"

	// Upgrades
	UpgradeCooldownReduction = "upgrade_cooldown_reduction"
	UpgradeJobsXPBoost       = "upgrade_jobs_xp_boost"
)

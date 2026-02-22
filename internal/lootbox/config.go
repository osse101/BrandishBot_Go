package lootbox

// ============================================================================
// Config types — JSON → Go (v2 format)
// ============================================================================

// PoolItemDef is one entry in a pool. Exactly one of ItemName or ItemType must be set.
type PoolItemDef struct {
	ItemName string `json:"item_name,omitempty"`
	ItemType string `json:"item_type,omitempty"`
	Weight   int    `json:"weight"`
}

// PoolDef holds the items that make up a named pool.
type PoolDef struct {
	Items []PoolItemDef `json:"items"`
}

// PoolRef links a pool into a lootbox with a relative selection weight.
type PoolRef struct {
	PoolName string `json:"pool_name"`
	Weight   int    `json:"weight"`
}

// MoneyRange defines the consolation money range (inclusive).
type MoneyRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Def defines one lootbox type in the config.
type Def struct {
	ItemDropRate float64    `json:"item_drop_rate"` // gatekeeper probability [0,1]
	FixedMoney   MoneyRange `json:"fixed_money"`
	Pools        []PoolRef  `json:"pools"`
}

// LootTableConfig is the top-level v2 config structure.
type LootTableConfig struct {
	Version   string             `json:"version"`
	Pools     map[string]PoolDef `json:"pools"`
	Lootboxes map[string]Def     `json:"lootboxes"`
}

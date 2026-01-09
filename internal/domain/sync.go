package domain

import "time"

// SyncMetadata tracks the last sync of a JSON config file
type SyncMetadata struct {
	ConfigName   string    `json:"config_name" db:"config_name"`
	LastSyncTime time.Time `json:"last_sync_time" db:"last_sync_time"`
	FileHash     string    `json:"file_hash" db:"file_hash"`
	FileModTime  time.Time `json:"file_mod_time" db:"file_mod_time"`
}

// ItemType represents a tag/type that can be assigned to items
type ItemType struct {
	ID   int    `json:"id" db:"item_type_id"`
	Name string `json:"name" db:"type_name"`
}

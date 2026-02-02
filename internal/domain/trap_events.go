package domain

import "github.com/google/uuid"

// TrapPlacedData contains data for trap placement events
type TrapPlacedData struct {
	TrapID         uuid.UUID
	SetterID       uuid.UUID
	SetterUsername string
	TargetID       uuid.UUID
	TargetUsername string
	ShineLevel     ShineLevel // COMMON, UNCOMMON, RARE, EPIC, LEGENDARY, etc.
	TimeoutSeconds int
}

// TrapTriggeredData contains data for trap trigger events
type TrapTriggeredData struct {
	TrapID           uuid.UUID
	SetterID         uuid.UUID
	SetterUsername   string
	TargetID         uuid.UUID
	TargetUsername   string
	ShineLevel       ShineLevel // COMMON, UNCOMMON, RARE, EPIC, LEGENDARY, etc.
	TimeoutSeconds   int
	WasSelfTriggered bool
}

// ToMap converts TrapPlacedData to map for event publishing
func (d *TrapPlacedData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"trap_id":         d.TrapID.String(),
		"setter_id":       d.SetterID.String(),
		"setter_username": d.SetterUsername,
		"target_id":       d.TargetID.String(),
		"target_username": d.TargetUsername,
		"shine_level":     d.ShineLevel,
		"timeout_seconds": d.TimeoutSeconds,
	}
}

// ToMap converts TrapTriggeredData to map for event publishing
func (d *TrapTriggeredData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"trap_id":            d.TrapID.String(),
		"setter_id":          d.SetterID.String(),
		"setter_username":    d.SetterUsername,
		"target_id":          d.TargetID.String(),
		"target_username":    d.TargetUsername,
		"shine_level":        d.ShineLevel,
		"timeout_seconds":    d.TimeoutSeconds,
		"was_self_triggered": d.WasSelfTriggered,
	}
}

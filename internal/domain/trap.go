package domain

import (
	"time"

	"github.com/google/uuid"
)

// Trap represents a trap placed on a user
type Trap struct {
	ID             uuid.UUID
	SetterID       uuid.UUID
	TargetID       uuid.UUID
	ShineLevel     ShineLevel // COMMON, UNCOMMON, RARE, EPIC, LEGENDARY, JUNK, etc.
	TimeoutSeconds int
	PlacedAt       time.Time
	TriggeredAt    *time.Time
}

// IsTriggered returns true if the trap has been triggered
func (t *Trap) IsTriggered() bool {
	return t.TriggeredAt != nil
}

// CalculateTimeout returns the timeout duration in seconds based on shine level
// Base: 60s, with adjustments per shine level
// Junk: 40s, Common: 60s, Uncommon: 70s, Rare: 80s, Epic: 90s, Legendary: 100s
func (t *Trap) CalculateTimeout() int {
	baseTimeout := 60
	shineModifier := map[ShineLevel]int{
		ShineJunk:      -20, // 40s
		ShineCommon:    0,   // 60s (base)
		ShineUncommon:  10,  // 70s
		ShineRare:      20,  // 80s
		ShineEpic:      30,  // 90s
		ShineLegendary: 40,  // 100s
		ShinePoor:      -10, // 50s
		ShineCursed:    -30, // 30s
	}

	if modifier, ok := shineModifier[t.ShineLevel]; ok {
		return baseTimeout + modifier
	}
	return baseTimeout
}

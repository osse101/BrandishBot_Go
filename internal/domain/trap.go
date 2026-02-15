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
	QualityLevel   QualityLevel // COMMON, UNCOMMON, RARE, EPIC, LEGENDARY, JUNK, etc.
	TimeoutSeconds int
	PlacedAt       time.Time
	TriggeredAt    *time.Time
}

// IsTriggered returns true if the trap has been triggered
func (t *Trap) IsTriggered() bool {
	return t.TriggeredAt != nil
}

// CalculateTimeout returns the timeout duration in seconds based on quality level
// Base: 60s, with adjustments per quality level
// Junk: 40s, Common: 60s, Uncommon: 70s, Rare: 80s, Epic: 90s, Legendary: 100s
func (t *Trap) CalculateTimeout() int {
	baseTimeout := 60
	adjustment := int(t.QualityLevel.GetTimeoutAdjustment().Seconds())
	return baseTimeout + adjustment
}

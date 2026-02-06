package scenario

import (
	"time"
)

// Clock provides an abstraction for time operations
type Clock interface {
	// Now returns the current time
	Now() time.Time
	// Since returns the duration since the given time
	Since(t time.Time) time.Duration
	// Until returns the duration until the given time
	Until(t time.Time) time.Duration
}

// RealClock uses the actual system time
type RealClock struct{}

// NewRealClock creates a new RealClock instance
func NewRealClock() *RealClock {
	return &RealClock{}
}

// Now returns the current system time
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// Since returns the duration since the given time
func (c *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Until returns the duration until the given time
func (c *RealClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}

// SimulatedClock allows time manipulation for testing
type SimulatedClock struct {
	current time.Time
}

// NewSimulatedClock creates a new SimulatedClock starting at the given time
func NewSimulatedClock(start time.Time) *SimulatedClock {
	return &SimulatedClock{
		current: start,
	}
}

// NewSimulatedClockNow creates a new SimulatedClock starting at the current time
func NewSimulatedClockNow() *SimulatedClock {
	return &SimulatedClock{
		current: time.Now(),
	}
}

// Now returns the simulated current time
func (c *SimulatedClock) Now() time.Time {
	return c.current
}

// Since returns the duration since the given time
func (c *SimulatedClock) Since(t time.Time) time.Duration {
	return c.current.Sub(t)
}

// Until returns the duration until the given time
func (c *SimulatedClock) Until(t time.Time) time.Duration {
	return t.Sub(c.current)
}

// Advance moves the simulated time forward by the given duration
func (c *SimulatedClock) Advance(d time.Duration) {
	c.current = c.current.Add(d)
}

// AdvanceHours moves the simulated time forward by the given number of hours
func (c *SimulatedClock) AdvanceHours(hours float64) {
	c.current = c.current.Add(time.Duration(hours * float64(time.Hour)))
}

// Set sets the simulated time to a specific value
func (c *SimulatedClock) Set(t time.Time) {
	c.current = t
}

// SetToHoursAgo sets the simulated time to a number of hours ago from now
func (c *SimulatedClock) SetToHoursAgo(hours float64) {
	c.current = time.Now().Add(-time.Duration(hours * float64(time.Hour)))
}

// GetCurrent returns the current simulated time
func (c *SimulatedClock) GetCurrent() time.Time {
	return c.current
}

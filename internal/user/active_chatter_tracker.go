package user

import (
	"sync"
)

// ActiveChatterTracker tracks users who have recently sent messages
// and are eligible to be targeted by random weapons like grenades and TNT.
type ActiveChatterTracker struct {
	mu       sync.RWMutex
	chatters map[string]*ChatterInfo
	stopCh   chan struct{}
}

// NewActiveChatterTracker creates a new tracker and starts the cleanup goroutine
func NewActiveChatterTracker() *ActiveChatterTracker {
	tracker := &ActiveChatterTracker{
		chatters: make(map[string]*ChatterInfo),
		stopCh:   make(chan struct{}),
	}
	go tracker.cleanupLoop()
	return tracker
}

// Stop stops the cleanup goroutine (useful for testing and shutdown)
func (t *ActiveChatterTracker) Stop() {
	close(t.stopCh)
}

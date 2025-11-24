package concurrency

import (
	"sync"
)

// LockManager handles named locks
type LockManager struct {
	locks sync.Map
}

// NewLockManager creates a new LockManager
func NewLockManager() *LockManager {
	return &LockManager{}
}

// GetLock returns a mutex for the given key
func (lm *LockManager) GetLock(key string) *sync.Mutex {
	lock, _ := lm.locks.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

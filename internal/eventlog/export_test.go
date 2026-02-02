package eventlog

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/event"
)

// TestHooks provides access to private methods for testing.
// This file is only compiled during tests (due to _test.go suffix).
type TestHooks struct {
	svc *service
}

// NewTestHooks creates test hooks for the given service.
func NewTestHooks(s Service) *TestHooks {
	return &TestHooks{svc: s.(*service)}
}

// HandleEvent exposes the private handleEvent method for testing.
func (h *TestHooks) HandleEvent(ctx context.Context, evt event.Event) error {
	return h.svc.handleEvent(ctx, evt)
}

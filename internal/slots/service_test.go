package slots

import (
	"context"
	"testing"
	"time"
)

func TestServiceShutdown(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil, nil).(*service)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s.wg.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		s.wg.Done()
	}()

	err := s.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected no error on graceful shutdown, got %v", err)
	}
}

func TestServiceShutdown_Timeout(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil, nil).(*service)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	s.wg.Add(1)
	// We don't call Done(), forcing a timeout

	err := s.Shutdown(ctx)
	if err == nil {
		t.Errorf("expected timeout error on shutdown, got nil")
	}
}

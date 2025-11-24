package stats

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ThreadSafeMockRepository implements Repository interface for testing with thread safety
type ThreadSafeMockRepository struct {
	events           []domain.StatsEvent
	recordEventError error
	mu               sync.Mutex
}

func (m *ThreadSafeMockRepository) RecordEvent(ctx context.Context, event *domain.StatsEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.recordEventError != nil {
		return m.recordEventError
	}
	event.EventID = int64(len(m.events) + 1)
	m.events = append(m.events, *event)
	return nil
}

func (m *ThreadSafeMockRepository) GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var filtered []domain.StatsEvent
	for _, event := range m.events {
		if event.UserID == userID && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func (m *ThreadSafeMockRepository) GetEventsByType(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var filtered []domain.StatsEvent
	for _, event := range m.events {
		if event.EventType == eventType && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func (m *ThreadSafeMockRepository) GetTopUsers(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time, limit int) ([]domain.LeaderboardEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	counts := make(map[string]int)
	for _, event := range m.events {
		if event.EventType == eventType && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.UserID]++
		}
	}
	
	var entries []domain.LeaderboardEntry
	for userID, count := range counts {
		entries = append(entries, domain.LeaderboardEntry{
			UserID:    userID,
			Count:     count,
			EventType: string(eventType),
		})
	}
	
	return entries, nil
}

func (m *ThreadSafeMockRepository) GetEventCounts(ctx context.Context, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	counts := make(map[domain.EventType]int)
	for _, event := range m.events {
		if event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.EventType]++
		}
	}
	return counts, nil
}

func (m *ThreadSafeMockRepository) GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	counts := make(map[domain.EventType]int)
	for _, event := range m.events {
		if event.UserID == userID && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.EventType]++
		}
	}
	return counts, nil
}

func (m *ThreadSafeMockRepository) GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	count := 0
	for _, event := range m.events {
		if event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			count++
		}
	}
	return count, nil
}

func TestConcurrency_RecordUserEvent(t *testing.T) {
	// Use a thread-safe mock repo because we want to test the SERVICE concurrency,
	// not the mock repo's lack of thread safety.
	repo := &ThreadSafeMockRepository{}
	svc := NewService(repo)
	ctx := context.Background()

	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			err := svc.RecordUserEvent(ctx, "user-concurrent", domain.EventItemAdded, nil)
			if err != nil {
				t.Errorf("RecordUserEvent failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify total events
	count, err := repo.GetTotalEventCount(ctx, time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}

	if count != concurrency {
		t.Errorf("Expected %d events, got %d", concurrency, count)
	}
}

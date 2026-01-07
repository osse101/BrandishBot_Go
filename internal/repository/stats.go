package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Stats defines the interface for stats persistence
type Stats interface {
	RecordEvent(ctx context.Context, event *domain.StatsEvent) error
	GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error)
	GetEventsByType(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time) ([]domain.StatsEvent, error)
	GetUserEventsByType(ctx context.Context, userID string, eventType domain.EventType, limit int) ([]domain.StatsEvent, error)
	GetTopUsers(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time, limit int) ([]domain.LeaderboardEntry, error)
	GetEventCounts(ctx context.Context, startTime, endTime time.Time) (map[domain.EventType]int, error)
	GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error)
	GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error)
}

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/stats"
)

// StatsRepository implements the stats repository for PostgreSQL
type StatsRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

// NewStatsRepository creates a new StatsRepository
func NewStatsRepository(pool *pgxpool.Pool) stats.Repository {
	return &StatsRepository{
		pool: pool,
		q:    generated.New(pool),
	}
}

// RecordEvent inserts a new event into the stats_events table
func (r *StatsRepository) RecordEvent(ctx context.Context, event *domain.StatsEvent) error {
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	userUUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	
	// Prepare params
	params := generated.RecordEventParams{
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
		EventType: string(event.EventType),
		EventData: eventDataJSON,
	}
	
	if !event.CreatedAt.IsZero() {
		params.CreatedAt = pgtype.Timestamp{Time: event.CreatedAt, Valid: true}
	} else {
		params.CreatedAt = pgtype.Timestamp{Time: time.Now(), Valid: true}
	}

	result, err := r.q.RecordEvent(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	
	event.EventID = int64(result.EventID)
	event.CreatedAt = result.CreatedAt.Time
	
	return nil
}

// GetEventsByUser retrieves all events for a specific user within a time range
func (r *StatsRepository) GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetEventsByUser(ctx, generated.GetEventsByUserParams{
		UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	var events []domain.StatsEvent
	for _, row := range rows {
		var eventData map[string]interface{}
		if len(row.EventData) > 0 {
			if err := json.Unmarshal(row.EventData, &eventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}
		
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = [16]byte(row.UserID.Bytes)
		}

		events = append(events, domain.StatsEvent{
			EventID:   row.EventID,
			UserID:    uid.String(),
			EventType: domain.EventType(row.EventType),
			EventData: eventData,
			CreatedAt: row.CreatedAt.Time,
		})
	}

	return events, nil
}

// GetUserEventsByType retrieves events of a specific type for a specific user with a limit
func (r *StatsRepository) GetUserEventsByType(ctx context.Context, userID string, eventType domain.EventType, limit int) ([]domain.StatsEvent, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserEventsByType(ctx, generated.GetUserEventsByTypeParams{
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
		EventType: string(eventType),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query user events: %w", err)
	}

	var events []domain.StatsEvent
	for _, row := range rows {
		var eventData map[string]interface{}
		if len(row.EventData) > 0 {
			if err := json.Unmarshal(row.EventData, &eventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}
		
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = [16]byte(row.UserID.Bytes)
		}

		events = append(events, domain.StatsEvent{
			EventID:   row.EventID,
			UserID:    uid.String(),
			EventType: domain.EventType(row.EventType),
			EventData: eventData,
			CreatedAt: row.CreatedAt.Time,
		})
	}

	return events, nil
}

// GetEventsByType retrieves all events of a specific type within a time range
func (r *StatsRepository) GetEventsByType(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	rows, err := r.q.GetEventsByType(ctx, generated.GetEventsByTypeParams{
		EventType:   string(eventType),
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	var events []domain.StatsEvent
	for _, row := range rows {
		var eventData map[string]interface{}
		if len(row.EventData) > 0 {
			if err := json.Unmarshal(row.EventData, &eventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}
		
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = [16]byte(row.UserID.Bytes)
		}

		events = append(events, domain.StatsEvent{
			EventID:   row.EventID,
			UserID:    uid.String(),
			EventType: domain.EventType(row.EventType),
			EventData: eventData,
			CreatedAt: row.CreatedAt.Time,
		})
	}

	return events, nil
}

// GetTopUsers retrieves the most active users for a specific event type
func (r *StatsRepository) GetTopUsers(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time, limit int) ([]domain.LeaderboardEntry, error) {
	rows, err := r.q.GetTopUsers(ctx, generated.GetTopUsersParams{
		EventType:   string(eventType),
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}

	var entries []domain.LeaderboardEntry
	for _, row := range rows {
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = [16]byte(row.UserID.Bytes)
		}
		
		entries = append(entries, domain.LeaderboardEntry{
			UserID:    uid.String(),
			Username:  row.Username,
			Count:     int(row.EventCount),
			EventType: string(eventType),
		})
	}

	return entries, nil
}

// GetEventCounts retrieves event counts grouped by event type within a time range
func (r *StatsRepository) GetEventCounts(ctx context.Context, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	rows, err := r.q.GetEventCounts(ctx, generated.GetEventCountsParams{
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query event counts: %w", err)
	}

	counts := make(map[domain.EventType]int)
	for _, row := range rows {
		counts[domain.EventType(row.EventType)] = int(row.Count)
	}

	return counts, nil
}

// GetUserEventCounts retrieves event counts for a specific user grouped by event type
func (r *StatsRepository) GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserEventCounts(ctx, generated.GetUserEventCountsParams{
		UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query user event counts: %w", err)
	}

	counts := make(map[domain.EventType]int)
	for _, row := range rows {
		counts[domain.EventType(row.EventType)] = int(row.Count)
	}

	return counts, nil
}

// GetTotalEventCount retrieves the total number of events within a time range
func (r *StatsRepository) GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error) {
	count, err := r.q.GetTotalEventCount(ctx, generated.GetTotalEventCountParams{
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get total event count: %w", err)
	}

	return int(count), nil
}

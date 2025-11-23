package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// StatsRepository implements the stats repository for PostgreSQL
type StatsRepository struct {
	db *pgxpool.Pool
}

// NewStatsRepository creates a new StatsRepository
func NewStatsRepository(db *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{db: db}
}

// RecordEvent inserts a new event into the stats_events table
func (r *StatsRepository) RecordEvent(ctx context.Context, event *domain.StatsEvent) error {
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO stats_events (user_id, event_type, event_data, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING event_id, created_at
	`

	err = r.db.QueryRow(ctx, query,
		event.UserID,
		event.EventType,
		eventDataJSON,
		event.CreatedAt,
	).Scan(&event.EventID, &event.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// GetEventsByUser retrieves all events for a specific user within a time range
func (r *StatsRepository) GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	query := `
		SELECT event_id, user_id, event_type, event_data, created_at
		FROM stats_events
		WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []domain.StatsEvent
	for rows.Next() {
		var event domain.StatsEvent
		var eventDataJSON []byte

		err := rows.Scan(
			&event.EventID,
			&event.UserID,
			&event.EventType,
			&eventDataJSON,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(eventDataJSON) > 0 {
			if err := json.Unmarshal(eventDataJSON, &event.EventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetEventsByType retrieves all events of a specific type within a time range
func (r *StatsRepository) GetEventsByType(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	query := `
		SELECT event_id, user_id, event_type, event_data, created_at
		FROM stats_events
		WHERE event_type = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, eventType, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []domain.StatsEvent
	for rows.Next() {
		var event domain.StatsEvent
		var eventDataJSON []byte

		err := rows.Scan(
			&event.EventID,
			&event.UserID,
			&event.EventType,
			&eventDataJSON,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(eventDataJSON) > 0 {
			if err := json.Unmarshal(eventDataJSON, &event.EventData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetTopUsers retrieves the most active users for a specific event type
func (r *StatsRepository) GetTopUsers(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time, limit int) ([]domain.LeaderboardEntry, error) {
	query := `
		SELECT se.user_id, u.username, COUNT(*) as event_count
		FROM stats_events se
		JOIN users u ON se.user_id = u.user_id
		WHERE se.event_type = $1 AND se.created_at >= $2 AND se.created_at <= $3
		GROUP BY se.user_id, u.username
		ORDER BY event_count DESC
		LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, eventType, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	for rows.Next() {
		var entry domain.LeaderboardEntry
		err := rows.Scan(&entry.UserID, &entry.Username, &entry.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entry.EventType = string(eventType)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard: %w", err)
	}

	return entries, nil
}

// GetEventCounts retrieves event counts grouped by event type within a time range
func (r *StatsRepository) GetEventCounts(ctx context.Context, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	query := `
		SELECT event_type, COUNT(*) as count
		FROM stats_events
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY event_type
	`

	rows, err := r.db.Query(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[domain.EventType]int)
	for rows.Next() {
		var eventType domain.EventType
		var count int
		err := rows.Scan(&eventType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event count: %w", err)
		}
		counts[eventType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event counts: %w", err)
	}

	return counts, nil
}

// GetUserEventCounts retrieves event counts for a specific user grouped by event type
func (r *StatsRepository) GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	query := `
		SELECT event_type, COUNT(*) as count
		FROM stats_events
		WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3
		GROUP BY event_type
	`

	rows, err := r.db.Query(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query user event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[domain.EventType]int)
	for rows.Next() {
		var eventType domain.EventType
		var count int
		err := rows.Scan(&eventType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event count: %w", err)
		}
		counts[eventType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event counts: %w", err)
	}

	return counts, nil
}

// GetTotalEventCount retrieves the total number of events within a time range
func (r *StatsRepository) GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM stats_events
		WHERE created_at >= $1 AND created_at <= $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, startTime, endTime).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total event count: %w", err)
	}

	return count, nil
}

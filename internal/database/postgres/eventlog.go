package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
)

type eventLogRepository struct {
	db *pgxpool.Pool
}

// NewEventLogRepository creates a new PostgreSQL event log repository
func NewEventLogRepository(db *pgxpool.Pool) eventlog.Repository {
	return &eventLogRepository{db: db}
}

// LogEvent stores an event in the database
func (r *eventLogRepository) LogEvent(ctx context.Context, eventType string, userID *string, payload, metadata map[string]interface{}) error {
	query := `
		INSERT INTO events (event_type, user_id, payload, metadata)
		VALUES ($1, $2, $3, $4)
	`

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Exec(ctx, query, eventType, userID, payloadJSON, metadataJSON)
	return err
}

// GetEvents retrieves events based on filter criteria
func (r *eventLogRepository) GetEvents(ctx context.Context, filter eventlog.EventFilter) ([]eventlog.Event, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, event_type, user_id, payload, metadata, created_at
		FROM events
		WHERE 1=1`)

	args := []interface{}{}
	argNum := 1

	if filter.UserID != nil {
		fmt.Fprintf(&queryBuilder, " AND user_id = $%d", argNum)
		args = append(args, *filter.UserID)
		argNum++
	}

	if filter.EventType != nil {
		fmt.Fprintf(&queryBuilder, " AND event_type = $%d", argNum)
		args = append(args, *filter.EventType)
		argNum++
	}

	if filter.Since != nil {
		fmt.Fprintf(&queryBuilder, " AND created_at >= $%d", argNum)
		args = append(args, *filter.Since)
		argNum++
	}

	if filter.Until != nil {
		fmt.Fprintf(&queryBuilder, " AND created_at <= $%d", argNum)
		args = append(args, *filter.Until)
		argNum++
	}

	queryBuilder.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		fmt.Fprintf(&queryBuilder, " LIMIT $%d", argNum)
		args = append(args, filter.Limit)
	}

	rows, err := r.db.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

// GetEventsByUser retrieves events for a specific user
func (r *eventLogRepository) GetEventsByUser(ctx context.Context, userID string, limit int) ([]eventlog.Event, error) {
	query := `
		SELECT id, event_type, user_id, payload, metadata, created_at
		FROM events
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

// GetEventsByType retrieves events of a specific type
func (r *eventLogRepository) GetEventsByType(ctx context.Context, eventType string, limit int) ([]eventlog.Event, error) {
	query := `
		SELECT id, event_type, user_id, payload, metadata, created_at
		FROM events
		WHERE event_type = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, eventType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

// CleanupOldEvents removes events older than the specified number of days
func (r *eventLogRepository) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	query := `
		DELETE FROM events
		WHERE created_at < NOW() - INTERVAL '1 day' * $1
	`

	result, err := r.db.Exec(ctx, query, retentionDays)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// scanEvents scans rows into Event structs
func (r *eventLogRepository) scanEvents(rows pgx.Rows) ([]eventlog.Event, error) {
	var events []eventlog.Event

	for rows.Next() {
		var evt eventlog.Event
		var payloadJSON, metadataJSON []byte

		err := rows.Scan(
			&evt.ID,
			&evt.EventType,
			&evt.UserID,
			&payloadJSON,
			&metadataJSON,
			&evt.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal payload
		if err := json.Unmarshal(payloadJSON, &evt.Payload); err != nil {
			return nil, err
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &evt.Metadata); err != nil {
				return nil, err
			}
		}

		events = append(events, evt)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

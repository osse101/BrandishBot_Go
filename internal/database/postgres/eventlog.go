package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
)

type eventLogRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

// NewEventLogRepository creates a new PostgreSQL event log repository
func NewEventLogRepository(pool *pgxpool.Pool) eventlog.Repository {
	return &eventLogRepository{
		pool: pool,
		q:    generated.New(pool),
	}
}

// LogEvent stores an event in the database
func (r *eventLogRepository) LogEvent(ctx context.Context, eventType string, userID *string, payload, metadata map[string]interface{}) error {
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

	var uid pgtype.Text
	if userID != nil {
		uid = pgtype.Text{String: *userID, Valid: true}
	} else {
		uid = pgtype.Text{Valid: false}
	}

	return r.q.LogEvent(ctx, generated.LogEventParams{
		EventType: eventType,
		UserID:    uid,
		Payload:   payloadJSON,
		Metadata:  metadataJSON,
	})
}

// GetEvents retrieves events based on filter criteria
func (r *eventLogRepository) GetEvents(ctx context.Context, filter eventlog.EventFilter) ([]eventlog.Event, error) {
	// Prepare params
	params := generated.GetEventsParams{
		Limit: int32(filter.Limit),
	}

	if filter.UserID != nil {
		params.UserID = pgtype.Text{String: *filter.UserID, Valid: true}
	} else {
		params.UserID = pgtype.Text{Valid: false}
	}

	if filter.EventType != nil {
		params.EventType = pgtype.Text{String: *filter.EventType, Valid: true}
	} else {
		params.EventType = pgtype.Text{Valid: false}
	}

	if filter.Since != nil {
		params.Since = pgtype.Timestamptz{Time: *filter.Since, Valid: true}
	} else {
		params.Since = pgtype.Timestamptz{Valid: false}
	}

	if filter.Until != nil {
		params.Until = pgtype.Timestamptz{Time: *filter.Until, Valid: true}
	} else {
		params.Until = pgtype.Timestamptz{Valid: false}
	}

	rows, err := r.q.GetEvents(ctx, params)
	if err != nil {
		return nil, err
	}

	var events []eventlog.Event
	for _, row := range rows {
		evt, err := mapRowToEvent(row)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

// GetEventsByUser retrieves events for a specific user
func (r *eventLogRepository) GetEventsByUser(ctx context.Context, userID string, limit int) ([]eventlog.Event, error) {
	rows, err := r.q.GetLogEventsByUser(ctx, generated.GetLogEventsByUserParams{
		UserID: pgtype.Text{String: userID, Valid: true},
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var events []eventlog.Event
	for _, row := range rows {
		evt, err := mapRowToEvent(row)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

// GetEventsByType retrieves events of a specific type
func (r *eventLogRepository) GetEventsByType(ctx context.Context, eventType string, limit int) ([]eventlog.Event, error) {
	rows, err := r.q.GetLogEventsByType(ctx, generated.GetLogEventsByTypeParams{
		EventType: eventType,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var events []eventlog.Event
	for _, row := range rows {
		evt, err := mapRowToEvent(row)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

// CleanupOldEvents removes events older than the specified number of days
func (r *eventLogRepository) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	return r.q.CleanupOldEvents(ctx, int32(retentionDays))
}

func mapRowToEvent(row generated.Event) (eventlog.Event, error) {
	evt := eventlog.Event{
		ID:        row.ID,
		EventType: row.EventType,
		UserID:    nil, // Handle nullable ptr below
		CreatedAt: row.CreatedAt.Time,
	}

	if row.UserID.Valid {
		uid := row.UserID.String
		evt.UserID = &uid
	}

	if len(row.Payload) > 0 {
		if err := json.Unmarshal(row.Payload, &evt.Payload); err != nil {
			return evt, err
		}
	}

	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &evt.Metadata); err != nil {
			return evt, err
		}
	}

	return evt, nil
}

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
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// StatsRepository implements the stats repository for PostgreSQL
type StatsRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

// NewStatsRepository creates a new StatsRepository
func NewStatsRepository(pool *pgxpool.Pool) repository.Stats {
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

	userUUID, err := parseUserUUID(event.UserID)
	if err != nil {
		return err
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

	event.EventID = result.EventID
	event.CreatedAt = result.CreatedAt.Time

	return nil
}

// GetEventsByUser retrieves all events for a specific user within a time range
func (r *StatsRepository) GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	userUUID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.GetEventsByUser(ctx, generated.GetEventsByUserParams{
		UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: startTime, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	events := make([]domain.StatsEvent, 0, len(rows))
	for _, row := range rows {
		event, err := mapStatsEvent(row.EventID, row.UserID, row.EventType, row.EventData, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}

	return events, nil
}

// GetUserEventsByType retrieves events of a specific type for a specific user with a limit
func (r *StatsRepository) GetUserEventsByType(ctx context.Context, userID string, eventType domain.EventType, limit int) ([]domain.StatsEvent, error) {
	userUUID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.GetUserEventsByType(ctx, generated.GetUserEventsByTypeParams{
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
		EventType: string(eventType),
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query user events: %w", err)
	}

	events := make([]domain.StatsEvent, 0, len(rows))
	for _, row := range rows {
		event, err := mapStatsEvent(row.EventID, row.UserID, row.EventType, row.EventData, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
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

	events := make([]domain.StatsEvent, 0, len(rows))
	for _, row := range rows {
		event, err := mapStatsEvent(row.EventID, row.UserID, row.EventType, row.EventData, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}

	return events, nil
}

func mapStatsEvent(eventID int64, userID pgtype.UUID, eventType string, eventDataJSON []byte, createdAt pgtype.Timestamp) (*domain.StatsEvent, error) {
	var eventData map[string]interface{}
	if len(eventDataJSON) > 0 {
		if err := json.Unmarshal(eventDataJSON, &eventData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}
	}

	var uid uuid.UUID
	if userID.Valid {
		uid = userID.Bytes
	}

	return &domain.StatsEvent{
		EventID:   eventID,
		UserID:    uid.String(),
		EventType: domain.EventType(eventType),
		EventData: eventData,
		CreatedAt: createdAt.Time,
	}, nil
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

	entries := make([]domain.LeaderboardEntry, 0, len(rows))
	for _, row := range rows {
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = row.UserID.Bytes
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
	userUUID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
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

// GetUserSlotsStats retrieves aggregated slots statistics for a user
func (r *StatsRepository) GetUserSlotsStats(ctx context.Context, userID string, startTime, endTime time.Time) (*domain.SlotsStats, error) {
	userUUID, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}

	row, err := r.q.GetUserSlotsStats(ctx, generated.GetUserSlotsStatsParams{
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
		StartTime: pgtype.Timestamp{Time: startTime, Valid: true},
		EndTime:   pgtype.Timestamp{Time: endTime, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user slots stats: %w", err)
	}

	totalSpins := int(row.TotalSpins)
	totalWins := int(row.TotalWins)
	totalBet := convertToInt(row.TotalBet)
	totalPayout := convertToInt(row.TotalPayout)
	megaJackpotsHit := int(row.MegaJackpotsHit)
	biggestWin := convertToInt(row.BiggestWin)

	netProfit := totalPayout - totalBet
	winRate := 0.0
	if totalSpins > 0 {
		winRate = float64(totalWins) / float64(totalSpins) * 100
	}

	return &domain.SlotsStats{
		UserID:          userID,
		TotalSpins:      totalSpins,
		TotalWins:       totalWins,
		TotalBet:        totalBet,
		TotalPayout:     totalPayout,
		NetProfit:       netProfit,
		WinRate:         winRate,
		MegaJackpotsHit: megaJackpotsHit,
		BiggestWin:      biggestWin,
	}, nil
}

// GetSlotsLeaderboardByProfit retrieves top users by net profit
func (r *StatsRepository) GetSlotsLeaderboardByProfit(ctx context.Context, startTime, endTime time.Time, limit int) ([]domain.SlotsStats, error) {
	rows, err := r.q.GetSlotsLeaderboardByProfit(ctx, generated.GetSlotsLeaderboardByProfitParams{
		StartTime:   pgtype.Timestamp{Time: startTime, Valid: true},
		EndTime:     pgtype.Timestamp{Time: endTime, Valid: true},
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get slots leaderboard by profit: %w", err)
	}

	stats := make([]domain.SlotsStats, 0, len(rows))
	for _, row := range rows {
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = row.UserID.Bytes
		}

		netProfit := convertToInt(row.NetProfit)
		totalSpins := int(row.TotalSpins)

		stats = append(stats, domain.SlotsStats{
			UserID:     uid.String(),
			Username:   row.Username,
			TotalSpins: totalSpins,
			NetProfit:  netProfit,
		})
	}

	return stats, nil
}

// GetSlotsLeaderboardByWinRate retrieves top users by win rate (minimum spins required)
func (r *StatsRepository) GetSlotsLeaderboardByWinRate(ctx context.Context, startTime, endTime time.Time, minSpins, limit int) ([]domain.SlotsStats, error) {
	rows, err := r.q.GetSlotsLeaderboardByWinRate(ctx, generated.GetSlotsLeaderboardByWinRateParams{
		StartTime:   pgtype.Timestamp{Time: startTime, Valid: true},
		EndTime:     pgtype.Timestamp{Time: endTime, Valid: true},
		MinSpins:    int64(minSpins),
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get slots leaderboard by win rate: %w", err)
	}

	stats := make([]domain.SlotsStats, 0, len(rows))
	for _, row := range rows {
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = row.UserID.Bytes
		}

		totalSpins := int(row.TotalSpins)
		totalWins := int(row.TotalWins)
		winRate := convertToFloat(row.WinRate)

		stats = append(stats, domain.SlotsStats{
			UserID:     uid.String(),
			Username:   row.Username,
			TotalSpins: totalSpins,
			TotalWins:  totalWins,
			WinRate:    winRate,
		})
	}

	return stats, nil
}

// GetSlotsLeaderboardByMegaJackpots retrieves top users by mega jackpots hit
func (r *StatsRepository) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, startTime, endTime time.Time, limit int) ([]domain.SlotsStats, error) {
	rows, err := r.q.GetSlotsLeaderboardByMegaJackpots(ctx, generated.GetSlotsLeaderboardByMegaJackpotsParams{
		StartTime:   pgtype.Timestamp{Time: startTime, Valid: true},
		EndTime:     pgtype.Timestamp{Time: endTime, Valid: true},
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get slots leaderboard by mega jackpots: %w", err)
	}

	stats := make([]domain.SlotsStats, 0, len(rows))
	for _, row := range rows {
		var uid uuid.UUID
		if row.UserID.Valid {
			uid = row.UserID.Bytes
		}

		megaJackpotsHit := int(row.MegaJackpotsHit)

		stats = append(stats, domain.SlotsStats{
			UserID:          uid.String(),
			Username:        row.Username,
			MegaJackpotsHit: megaJackpotsHit,
		})
	}

	return stats, nil
}

// convertToInt converts interface{} to int, handling various numeric types
func convertToInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

// convertToFloat converts interface{} to float64, handling various numeric types
func convertToFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0.0
	}
}

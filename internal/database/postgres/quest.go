package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

type QuestRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

func NewQuestRepository(db *pgxpool.Pool) *QuestRepository {
	return &QuestRepository{
		db: db,
		q:  generated.New(db),
	}
}

// GetActiveQuests returns all active quests
func (r *QuestRepository) GetActiveQuests(ctx context.Context) ([]domain.Quest, error) {
	rows, err := r.q.GetActiveQuests(ctx)
	if err != nil {
		return nil, err
	}

	quests := make([]domain.Quest, len(rows))
	for i, row := range rows {
		quests[i] = mapQuestRow(row)
	}
	return quests, nil
}

// GetActiveQuestsForWeek returns active quests for a specific week
func (r *QuestRepository) GetActiveQuestsForWeek(ctx context.Context, year, weekNumber int) ([]domain.Quest, error) {
	rows, err := r.q.GetActiveQuestsForWeek(ctx, generated.GetActiveQuestsForWeekParams{
		Year:       int32(year),
		WeekNumber: int32(weekNumber),
	})
	if err != nil {
		return nil, err
	}

	quests := make([]domain.Quest, len(rows))
	for i, row := range rows {
		quests[i] = mapQuestRow(row)
	}
	return quests, nil
}

// CreateQuest creates a new quest from a template
func (r *QuestRepository) CreateQuest(ctx context.Context, template domain.QuestTemplate, year, weekNumber int) (*domain.Quest, error) {
	params := generated.CreateQuestParams{
		QuestKey:        template.QuestKey,
		QuestType:       template.QuestType,
		Description:     template.Description,
		BaseRequirement: int32(template.BaseRequirement),
		BaseRewardMoney: int32(template.BaseRewardMoney),
		BaseRewardXp:    int32(template.BaseRewardXp),
		Active:          true,
		WeekNumber:      int32(weekNumber),
		Year:            int32(year),
	}

	if template.TargetCategory != nil {
		params.TargetCategory = pgtype.Text{
			String: *template.TargetCategory,
			Valid:  true,
		}
	}

	if template.TargetRecipeKey != nil {
		params.TargetRecipeKey = pgtype.Text{
			String: *template.TargetRecipeKey,
			Valid:  true,
		}
	}

	row, err := r.q.CreateQuest(ctx, params)
	if err != nil {
		return nil, err
	}

	quest := mapQuestRow(row)
	return &quest, nil
}

// DeactivateAllQuests deactivates all active quests
func (r *QuestRepository) DeactivateAllQuests(ctx context.Context) error {
	return r.q.DeactivateAllQuests(ctx)
}

// GetUserQuestProgress gets all quest progress for a user
func (r *QuestRepository) GetUserQuestProgress(ctx context.Context, userID string) ([]domain.QuestProgress, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserQuestProgress(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	progress := make([]domain.QuestProgress, len(rows))
	for i, row := range rows {
		progress[i] = mapQuestProgressRow(row)
	}
	return progress, nil
}

// GetUserActiveQuestProgress gets active quest progress for a user
func (r *QuestRepository) GetUserActiveQuestProgress(ctx context.Context, userID string) ([]domain.QuestProgress, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserActiveQuestProgress(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	progress := make([]domain.QuestProgress, len(rows))
	for i, row := range rows {
		progress[i] = mapQuestProgressRowActive(row)
	}
	return progress, nil
}

// IncrementQuestProgress increments progress on a quest
func (r *QuestRepository) IncrementQuestProgress(ctx context.Context, userID string, questID int, incrementBy int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	return r.q.IncrementQuestProgress(ctx, generated.IncrementQuestProgressParams{
		UserID:          userUUID,
		QuestID:         int32(questID),
		ProgressCurrent: int32(incrementBy),
	})
}

// CompleteQuest marks a quest as completed
func (r *QuestRepository) CompleteQuest(ctx context.Context, userID string, questID int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	return r.q.CompleteQuest(ctx, generated.CompleteQuestParams{
		UserID:  userUUID,
		QuestID: int32(questID),
	})
}

// ClaimQuestReward marks a quest reward as claimed
func (r *QuestRepository) ClaimQuestReward(ctx context.Context, userID string, questID int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	return r.q.ClaimQuestReward(ctx, generated.ClaimQuestRewardParams{
		UserID:  userUUID,
		QuestID: int32(questID),
	})
}

// GetUnclaimedCompletedQuests gets all unclaimed but completed quests for a user
func (r *QuestRepository) GetUnclaimedCompletedQuests(ctx context.Context, userID string) ([]domain.QuestProgress, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUnclaimedCompletedQuests(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	progress := make([]domain.QuestProgress, len(rows))
	for i, row := range rows {
		progress[i] = mapQuestProgressRowUnclaimed(row)
	}
	return progress, nil
}

// ResetWeeklyQuests deletes progress for inactive quests
func (r *QuestRepository) ResetWeeklyQuests(ctx context.Context) (int64, error) {
	result, err := r.q.ResetInactiveQuestProgress(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetWeeklyQuestResetState gets the last weekly reset state
func (r *QuestRepository) GetWeeklyQuestResetState(ctx context.Context) (lastReset time.Time, weekNum, year int, err error) {
	row, err := r.q.GetWeeklyQuestResetState(ctx)
	if err != nil {
		return time.Time{}, 0, 0, err
	}

	return row.LastResetTime.Time, int(row.WeekNumber), int(row.Year), nil
}

// UpdateWeeklyQuestResetState updates the weekly reset state
func (r *QuestRepository) UpdateWeeklyQuestResetState(ctx context.Context, resetTime time.Time, weekNum, year, questsGenerated int, progressReset int64) error {
	return r.q.UpdateWeeklyQuestResetState(ctx, generated.UpdateWeeklyQuestResetStateParams{
		LastResetTime:   pgtype.Timestamptz{Time: resetTime, Valid: true},
		WeekNumber:      int32(weekNum),
		Year:            int32(year),
		QuestsGenerated: int32(questsGenerated),
		ProgressReset:   int32(progressReset),
	})
}

// Helper to map SQLC quest row to domain model
func mapQuestRow(row generated.Quest) domain.Quest {
	quest := domain.Quest{
		QuestID:         int(row.QuestID),
		QuestKey:        row.QuestKey,
		QuestType:       row.QuestType,
		Description:     row.Description,
		BaseRequirement: int(row.BaseRequirement),
		BaseRewardMoney: int(row.BaseRewardMoney),
		BaseRewardXp:    int(row.BaseRewardXp),
		Active:          row.Active,
		WeekNumber:      int(row.WeekNumber),
		Year:            int(row.Year),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.TargetCategory.Valid {
		quest.TargetCategory = &row.TargetCategory.String
	}

	if row.TargetRecipeKey.Valid {
		quest.TargetRecipeKey = &row.TargetRecipeKey.String
	}

	return quest
}

// Helper to map SQLC quest progress row to domain model
func mapQuestProgressRow(row generated.GetUserQuestProgressRow) domain.QuestProgress {
	progress := domain.QuestProgress{
		UserID:           row.UserID.String(),
		QuestID:          int(row.QuestID),
		ProgressCurrent:  int(row.ProgressCurrent),
		ProgressRequired: int(row.ProgressRequired),
		RewardMoney:      int(row.RewardMoney),
		RewardXp:         int(row.RewardXp),
		StartedAt:        row.StartedAt.Time,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		QuestKey:         row.QuestKey,
		QuestType:        row.QuestType,
		Description:      row.Description,
	}

	if row.CompletedAt.Valid {
		progress.CompletedAt = &row.CompletedAt.Time
	}
	if row.ClaimedAt.Valid {
		progress.ClaimedAt = &row.ClaimedAt.Time
	}
	if row.TargetCategory.Valid {
		progress.TargetCategory = &row.TargetCategory.String
	}
	if row.TargetRecipeKey.Valid {
		progress.TargetRecipeKey = &row.TargetRecipeKey.String
	}

	return progress
}

// Helper to map SQLC active quest progress row to domain model
func mapQuestProgressRowActive(row generated.GetUserActiveQuestProgressRow) domain.QuestProgress {
	progress := domain.QuestProgress{
		UserID:           row.UserID.String(),
		QuestID:          int(row.QuestID),
		ProgressCurrent:  int(row.ProgressCurrent),
		ProgressRequired: int(row.ProgressRequired),
		RewardMoney:      int(row.RewardMoney),
		RewardXp:         int(row.RewardXp),
		StartedAt:        row.StartedAt.Time,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		QuestKey:         row.QuestKey,
		QuestType:        row.QuestType,
		Description:      row.Description,
	}

	if row.CompletedAt.Valid {
		progress.CompletedAt = &row.CompletedAt.Time
	}
	if row.ClaimedAt.Valid {
		progress.ClaimedAt = &row.ClaimedAt.Time
	}
	if row.TargetCategory.Valid {
		progress.TargetCategory = &row.TargetCategory.String
	}
	if row.TargetRecipeKey.Valid {
		progress.TargetRecipeKey = &row.TargetRecipeKey.String
	}

	return progress
}

// Helper to map SQLC unclaimed quest progress row to domain model
func mapQuestProgressRowUnclaimed(row generated.GetUnclaimedCompletedQuestsRow) domain.QuestProgress {
	progress := domain.QuestProgress{
		UserID:           row.UserID.String(),
		QuestID:          int(row.QuestID),
		ProgressCurrent:  int(row.ProgressCurrent),
		ProgressRequired: int(row.ProgressRequired),
		RewardMoney:      int(row.RewardMoney),
		RewardXp:         int(row.RewardXp),
		StartedAt:        row.StartedAt.Time,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		QuestKey:         row.QuestKey,
		QuestType:        row.QuestType,
		Description:      row.Description,
	}

	if row.CompletedAt.Valid {
		progress.CompletedAt = &row.CompletedAt.Time
	}
	if row.ClaimedAt.Valid {
		progress.ClaimedAt = &row.ClaimedAt.Time
	}
	if row.TargetCategory.Valid {
		progress.TargetCategory = &row.TargetCategory.String
	}
	if row.TargetRecipeKey.Valid {
		progress.TargetRecipeKey = &row.TargetRecipeKey.String
	}

	return progress
}

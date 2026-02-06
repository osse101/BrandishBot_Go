package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

type QuestRepository interface {
	// Quest management
	GetActiveQuests(ctx context.Context) ([]domain.Quest, error)
	GetActiveQuestsForWeek(ctx context.Context, year, weekNumber int) ([]domain.Quest, error)
	CreateQuest(ctx context.Context, template domain.QuestTemplate, year, weekNumber int) (*domain.Quest, error)
	DeactivateAllQuests(ctx context.Context) error

	// User quest progress
	GetUserQuestProgress(ctx context.Context, userID string) ([]domain.QuestProgress, error)
	GetUserActiveQuestProgress(ctx context.Context, userID string) ([]domain.QuestProgress, error)
	IncrementQuestProgress(ctx context.Context, userID string, questID int, incrementBy int) error
	CompleteQuest(ctx context.Context, userID string, questID int) error
	GetUnclaimedCompletedQuests(ctx context.Context, userID string) ([]domain.QuestProgress, error)
	ClaimQuestReward(ctx context.Context, userID string, questID int) error

	// Weekly reset
	ResetWeeklyQuests(ctx context.Context) (int64, error)
	GetWeeklyQuestResetState(ctx context.Context) (lastReset time.Time, weekNum, year int, err error)
	UpdateWeeklyQuestResetState(ctx context.Context, resetTime time.Time, weekNum, year, questsGenerated int, progressReset int64) error
}

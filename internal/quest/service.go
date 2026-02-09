package quest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type Service interface {
	// Quest management
	GetActiveQuests(ctx context.Context) ([]domain.Quest, error)
	GetUserQuestProgress(ctx context.Context, userID string) ([]domain.QuestProgress, error)
	ClaimQuestReward(ctx context.Context, userID string, questID int) (money int, err error)

	// Progress tracking (called by handlers/services)
	OnItemBought(ctx context.Context, userID string, itemCategory string, quantity int) error
	OnItemSold(ctx context.Context, userID string, itemCategory string, quantity, moneyEarned int) error
	OnRecipeCrafted(ctx context.Context, userID string, recipeKey string, quantity int) error
	OnSearch(ctx context.Context, userID string) error

	// Weekly reset (called by worker)
	ResetWeeklyQuests(ctx context.Context) error
	GenerateWeeklyQuests(ctx context.Context, year, weekNumber int) error

	// Lifecycle
	Shutdown(ctx context.Context) error
}

type service struct {
	repo       repository.QuestRepository
	jobService job.Service
	publisher  *event.ResilientPublisher
	questPool  []domain.QuestTemplate
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

func NewService(
	repo repository.QuestRepository,
	jobService job.Service,
	publisher *event.ResilientPublisher,
) (Service, error) {
	s := &service{
		repo:       repo,
		jobService: jobService,
		publisher:  publisher,
	}

	// Load quest pool from config
	if err := s.loadQuestPool(); err != nil {
		return nil, fmt.Errorf("failed to load quest pool: %w", err)
	}

	return s, nil
}

// loadQuestPool loads quest templates from config
func (s *service) loadQuestPool() error {
	data, err := os.ReadFile(config.ConfigPathQuestPool)
	if err != nil {
		return err
	}

	var cfg domain.QuestPoolConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	s.mu.Lock()
	s.questPool = cfg.QuestPool
	s.mu.Unlock()

	return nil
}

// GetActiveQuests returns all active quests
func (s *service) GetActiveQuests(ctx context.Context) ([]domain.Quest, error) {
	return s.repo.GetActiveQuests(ctx)
}

// GetUserQuestProgress returns user's quest progress
func (s *service) GetUserQuestProgress(ctx context.Context, userID string) ([]domain.QuestProgress, error) {
	return s.repo.GetUserQuestProgress(ctx, userID)
}

// GenerateWeeklyQuests generates 3 random quests for the week
func (s *service) GenerateWeeklyQuests(ctx context.Context, year, weekNumber int) error {
	log := logger.FromContext(ctx)

	s.mu.RLock()
	poolSize := len(s.questPool)
	s.mu.RUnlock()

	if poolSize < 3 {
		return fmt.Errorf("quest pool has fewer than 3 templates")
	}

	// Use week number as seed for deterministic randomization
	seed := int64(year*100 + weekNumber)
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec

	// Shuffle pool and take first 3
	s.mu.RLock()
	poolCopy := make([]domain.QuestTemplate, len(s.questPool))
	copy(poolCopy, s.questPool)
	s.mu.RUnlock()

	rng.Shuffle(len(poolCopy), func(i, j int) {
		poolCopy[i], poolCopy[j] = poolCopy[j], poolCopy[i]
	})

	selectedQuests := poolCopy[:3]

	// Create quests in database
	for _, template := range selectedQuests {
		_, err := s.repo.CreateQuest(ctx, template, year, weekNumber)
		if err != nil {
			log.Error("Failed to create quest", "quest_key", template.QuestKey, "error", err)
			return err
		}
	}

	log.Info("Generated weekly quests", "year", year, "week", weekNumber, "count", 3)
	return nil
}

// ResetWeeklyQuests deactivates old quests and generates new ones
func (s *service) ResetWeeklyQuests(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Get current week
	now := time.Now().UTC()
	year, week := now.ISOWeek()

	// Deactivate all active quests
	if err := s.repo.DeactivateAllQuests(ctx); err != nil {
		return fmt.Errorf("failed to deactivate quests: %w", err)
	}

	// Delete progress for inactive quests
	result, err := s.repo.ResetWeeklyQuests(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset quest progress: %w", err)
	}

	// Generate new quests
	if err := s.GenerateWeeklyQuests(ctx, year, week); err != nil {
		return fmt.Errorf("failed to generate quests: %w", err)
	}

	// Update reset state
	if err := s.repo.UpdateWeeklyQuestResetState(ctx, now, week, year, 3, result); err != nil {
		log.Warn("Failed to update reset state", "error", err)
	}

	// Publish reset event
	if s.publisher != nil {
		evt := event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeWeeklyQuestReset),
			Payload: map[string]interface{}{
				"reset_time":       now,
				"week_number":      week,
				"year":             year,
				"quests_generated": 3,
				"progress_reset":   result,
			},
		}
		s.publisher.PublishWithRetry(ctx, evt)
	}

	log.Info("Weekly quest reset completed", "week", week, "year", year, "progress_reset", result)
	return nil
}

// OnItemBought handles quest progress when user buys items
func (s *service) OnItemBought(ctx context.Context, userID string, itemCategory string, quantity int) error {
	quests, err := s.repo.GetUserActiveQuestProgress(ctx, userID)
	if err != nil {
		return err
	}

	for _, qp := range quests {
		// Skip if already completed
		if qp.CompletedAt != nil {
			continue
		}

		// Only process buy_items quests
		if qp.QuestType != domain.QuestTypeBuyItems {
			continue
		}

		// Check if item category matches quest filter
		if !s.matchesCategoryFilter(itemCategory, qp.TargetCategory) {
			continue
		}

		// Increment progress
		if err := s.incrementAndCheckCompletion(ctx, userID, qp, quantity); err != nil {
			return err
		}
	}

	return nil
}

// OnItemSold handles quest progress when user sells items
func (s *service) OnItemSold(ctx context.Context, userID string, itemCategory string, quantity, moneyEarned int) error {
	quests, err := s.repo.GetUserActiveQuestProgress(ctx, userID)
	if err != nil {
		return err
	}

	for _, qp := range quests {
		if qp.CompletedAt != nil {
			continue
		}

		switch qp.QuestType {
		case domain.QuestTypeSellItems:
			if s.matchesCategoryFilter(itemCategory, qp.TargetCategory) {
				if err := s.incrementAndCheckCompletion(ctx, userID, qp, quantity); err != nil {
					return err
				}
			}

		case domain.QuestTypeEarnMoney:
			if err := s.incrementAndCheckCompletion(ctx, userID, qp, moneyEarned); err != nil {
				return err
			}
		}
	}

	return nil
}

// incrementAndCheckCompletion increments progress and auto-completes if threshold reached
func (s *service) incrementAndCheckCompletion(ctx context.Context, userID string, qp domain.QuestProgress, incrementBy int) error {
	log := logger.FromContext(ctx)

	newProgress := qp.ProgressCurrent + incrementBy
	if newProgress > qp.ProgressRequired {
		newProgress = qp.ProgressRequired
	}

	// Update progress
	if err := s.repo.IncrementQuestProgress(ctx, userID, qp.QuestID, incrementBy); err != nil {
		return err
	}

	// Auto-complete if threshold reached
	if newProgress >= qp.ProgressRequired {
		if err := s.repo.CompleteQuest(ctx, userID, qp.QuestID); err != nil {
			return err
		}

		log.Info("Quest auto-completed", "user_id", userID, "quest_id", qp.QuestID, "quest_key", qp.QuestKey)

		// Publish completion event
		if s.publisher != nil {
			evt := event.Event{
				Version: "1.0",
				Type:    event.Type(domain.EventTypeQuestCompleted),
				Payload: map[string]interface{}{
					"user_id":   userID,
					"quest_id":  qp.QuestID,
					"quest_key": qp.QuestKey,
				},
			}
			s.publisher.PublishWithRetry(ctx, evt)
		}
	}

	return nil
}

// matchesCategoryFilter checks if item category matches quest filter
func (s *service) matchesCategoryFilter(itemCategory string, filterCategory *string) bool {
	// nil filter means all items match
	if filterCategory == nil {
		return true
	}

	// Exact match (case-insensitive)
	return strings.EqualFold(itemCategory, *filterCategory)
}

// OnRecipeCrafted handles quest progress when user performs crafting
func (s *service) OnRecipeCrafted(ctx context.Context, userID string, recipeKey string, quantity int) error {
	quests, err := s.repo.GetUserActiveQuestProgress(ctx, userID)
	if err != nil {
		return err
	}

	for _, qp := range quests {
		if qp.CompletedAt != nil {
			continue
		}

		if qp.QuestType != domain.QuestTypeCraftRecipe {
			continue
		}

		// Check if recipe matches
		if qp.TargetRecipeKey == nil || !strings.EqualFold(recipeKey, *qp.TargetRecipeKey) {
			continue
		}

		if err := s.incrementAndCheckCompletion(ctx, userID, qp, quantity); err != nil {
			return err
		}
	}

	return nil
}

// OnSearch handles quest progress when user performs a search
func (s *service) OnSearch(ctx context.Context, userID string) error {
	quests, err := s.repo.GetUserActiveQuestProgress(ctx, userID)
	if err != nil {
		return err
	}

	for _, qp := range quests {
		if qp.CompletedAt != nil {
			continue
		}

		if qp.QuestType != domain.QuestTypePerformSearches {
			continue
		}

		if err := s.incrementAndCheckCompletion(ctx, userID, qp, 1); err != nil {
			return err
		}
	}

	return nil
}

// ClaimQuestReward claims completed quest reward
func (s *service) ClaimQuestReward(ctx context.Context, userID string, questID int) (money int, err error) {
	log := logger.FromContext(ctx)

	// Get quest progress
	quests, err := s.repo.GetUnclaimedCompletedQuests(ctx, userID)
	if err != nil {
		return 0, err
	}

	var targetQuest *domain.QuestProgress
	for _, q := range quests {
		if q.QuestID == questID {
			targetQuest = &q
			break
		}
	}

	if targetQuest == nil {
		return 0, fmt.Errorf("quest not found or not claimable")
	}

	// Mark as claimed
	if err := s.repo.ClaimQuestReward(ctx, userID, questID); err != nil {
		return 0, err
	}

	// Award Merchant XP asynchronously
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		bgCtx := context.Background()
		metadata := map[string]interface{}{
			"quest_id":  questID,
			"quest_key": targetQuest.QuestKey,
		}

		if _, err := s.jobService.AwardXP(bgCtx, userID, job.JobKeyMerchant, targetQuest.RewardXp, "quest_reward", metadata); err != nil {
			log.Error("Failed to award quest XP", "user_id", userID, "error", err)
		}
	}()

	log.Info("Quest reward claimed", "user_id", userID, "quest_id", questID, "money", targetQuest.RewardMoney, "xp", targetQuest.RewardXp)

	// Publish claim event
	if s.publisher != nil {
		evt := event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeQuestClaimed),
			Payload: map[string]interface{}{
				"user_id":      userID,
				"quest_id":     questID,
				"reward_money": targetQuest.RewardMoney,
				"reward_xp":    targetQuest.RewardXp,
			},
		}
		s.publisher.PublishWithRetry(ctx, evt)
	}

	return targetQuest.RewardMoney, nil
}

func (s *service) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

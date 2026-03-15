// Package search implements the search gameplay feature.
// Players search for loot with cooldown tracking, quality-based rewards,
// region-specific item drops, and progression integration.
package search

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// UserResolver resolves user identity from platform credentials.
type UserResolver interface {
	GetUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error)
}

// ItemLookup provides cached item metadata.
type ItemLookup interface {
	GetItemByName(ctx context.Context, name string) (*domain.Item, error)
	BuildPublicNameIndex() map[string]string
}

// RewardGranter adds items to a user's inventory transactionally.
type RewardGranter interface {
	GrantSearchReward(ctx context.Context, user *domain.User, quantity int, quality domain.QualityLevel) error
	GrantItemReward(ctx context.Context, user *domain.User, item *domain.Item, quantity int, quality domain.QualityLevel) error
}

// ProgressionService provides progression-based modifiers.
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, userID string, featureKey string, baseValue float64) (float64, error)
}

// Deps bundles all dependencies for the search service.
type Deps struct {
	UserResolver   UserResolver
	ItemLookup     ItemLookup
	RewardGranter  RewardGranter
	CooldownSvc    cooldown.Service
	StatsSvc       stats.Service
	JobSvc         job.Service
	ProgressionSvc ProgressionService
	Publisher      *event.ResilientPublisher
	Rnd            func() float64
	Regions        []Region
}

// Service defines the interface for the search gameplay feature.
type Service interface {
	HandleSearch(ctx context.Context, platform, platformID, username, itemHint string) (string, error)
}

// service implements the search gameplay feature.
type service struct {
	deps Deps
}

// New creates a new search service with the given dependencies.
func New(deps Deps) Service {
	return &service{deps: deps}
}

// searchParams holds computed parameters for a single search.
type searchParams struct {
	isFirstSearchDaily bool
	isDiminished       bool
	xpMultiplier       float64
	successThreshold   float64
	dailyCount         int
	streak             int
	region             *Region
}

// HandleSearch performs a search action for a user with cooldown tracking.
func (s *service) HandleSearch(ctx context.Context, platform, platformID, username, itemHint string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleSearch called", "platform", platform, "platformID", platformID, "username", username, "itemHint", itemHint)

	if username == "" || platform == "" {
		return "", domain.ErrInvalidInput
	}
	if platform != domain.PlatformTwitch && platform != domain.PlatformDiscord {
		return "", domain.ErrInvalidInput
	}

	user, err := s.deps.UserResolver.GetUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user or register", "error", err)
		return "", err
	}

	var resultMessage string
	err = s.deps.CooldownSvc.EnforceCooldown(ctx, user.ID, domain.ActionSearch, func() error {
		var err error
		resultMessage, err = s.executeSearch(ctx, user, itemHint)
		return err
	})

	if err != nil {
		return "", err
	}

	log.Info("Search completed", "username", username, "result", resultMessage)
	return resultMessage, nil
}

// executeSearch performs the actual search logic (called within cooldown enforcement).
func (s *service) executeSearch(ctx context.Context, user *domain.User, itemHint string) (string, error) {
	log := logger.FromContext(ctx)
	params := s.calculateSearchParameters(ctx, user)

	// Resolve search region based on explorer level and optional item hint
	if len(s.deps.Regions) > 0 {
		explorerLevel := 0
		if s.deps.JobSvc != nil {
			if level, err := s.deps.JobSvc.GetJobLevel(ctx, user.ID, domain.JobKeyExplorer); err == nil {
				explorerLevel = level
			} else {
				log.Warn("Failed to get explorer level for region resolution", "error", err)
			}
		}
		pubIndex := s.deps.ItemLookup.BuildPublicNameIndex()
		params.region = resolveRegion(s.deps.Regions, explorerLevel, itemHint, pubIndex)
		if params.region != nil {
			params.successThreshold += params.region.LootboxChanceModifier
			if params.successThreshold < 0.1 {
				params.successThreshold = 0.1
			}
			log.Debug("Search region resolved", "region", params.region.Name, "modifier", params.region.LootboxChanceModifier, "threshold", params.successThreshold)
		}
	}

	// Perform search roll
	roll := s.deps.Rnd()

	var resultMessage string
	isSuccess := roll <= params.successThreshold
	var isCritical, isNearMiss, isCritFail bool
	var itemName string
	var quantity int

	if isSuccess {
		var err error
		resultMessage, err = s.processSearchSuccess(ctx, user, roll, params)
		if err != nil {
			return "", err
		}
		isCritical = roll <= domain.SearchCriticalRate
		quantity = 1
		if isCritical {
			quantity = 2
		}
		itemName = domain.ItemLootbox0
	} else {
		failureType := determineSearchFailureType(roll, params.successThreshold)
		isNearMiss = failureType == searchFailureNearMiss
		isCritFail = failureType == searchFailureCritical
		resultMessage = s.processSearchFailure(roll, params.successThreshold, params)
	}

	xpAmount := int(float64(job.ExplorerXPPerItem) * params.xpMultiplier)
	if xpAmount < 1 {
		xpAmount = 1
	}

	if s.deps.Publisher != nil {
		s.deps.Publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeSearchPerformed),
			Payload: domain.SearchPerformedPayload{
				UserID:         user.ID,
				Success:        isSuccess,
				IsCritical:     isCritical,
				IsNearMiss:     isNearMiss,
				IsCriticalFail: isCritFail,
				XPAmount:       xpAmount,
				ItemName:       itemName,
				Quantity:       quantity,
				Timestamp:      time.Now().Unix(),
			},
		})
	}

	return resultMessage, nil
}

func (s *service) calculateSearchParameters(ctx context.Context, user *domain.User) searchParams {
	log := logger.FromContext(ctx)
	dailyCount := 0
	if s.deps.StatsSvc != nil {
		stats, err := s.deps.StatsSvc.GetUserStats(ctx, user.ID, domain.PeriodDaily)
		if err != nil {
			log.Warn("Failed to get search counts", "error", err)
		} else if stats != nil && stats.EventCounts != nil {
			dailyCount = stats.EventCounts[domain.StatsEventSearch]
		}
	}

	params := searchParams{
		isFirstSearchDaily: (dailyCount == 0),
		isDiminished:       (dailyCount >= domain.SearchDailyDiminishmentThreshold),
		xpMultiplier:       1.0,
		successThreshold:   domain.SearchSuccessRate,
		dailyCount:         dailyCount,
	}

	if params.isDiminished {
		params.xpMultiplier = domain.SearchDiminishedXPMultiplier
		log.Info("Diminished search returns applied", "username", user.Username, "dailyCount", dailyCount)
	}

	if params.isFirstSearchDaily && s.deps.StatsSvc != nil {
		streak, err := s.deps.StatsSvc.GetUserCurrentStreak(ctx, user.ID)
		if err != nil {
			log.Warn("Failed to get user streak", "error", err)
		} else {
			params.streak = streak
		}
	}

	return params
}

func (s *service) processSearchSuccess(ctx context.Context, user *domain.User, roll float64, params searchParams) (string, error) {
	isCritical := roll <= domain.SearchCriticalRate
	quantity := 1
	if isCritical {
		quantity = 2
	}

	// Determine if we grant a region item instead of a lootbox
	if params.region != nil && len(params.region.ItemDrops) > 0 {
		regionRoll := utils.RandomFloat()
		if regionRoll < domain.SearchRegionItemDropChance {
			return s.processRegionItemDrop(ctx, user, isCritical, quantity, params)
		}
	}

	// Default: grant lootbox
	qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
	if err := s.deps.RewardGranter.GrantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
		return "", err
	}

	item, err := s.deps.ItemLookup.GetItemByName(ctx, domain.ItemLootbox0)
	if err != nil {
		return "", fmt.Errorf("failed to get reward item: %w", err)
	}
	if item == nil {
		return "", domain.ErrItemNotFound
	}

	msg := s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params)
	if params.region != nil && params.region.RequiredExplorerLevel > 0 {
		msg += fmt.Sprintf(" [%s]", params.region.Name)
	}
	return msg, nil
}

func (s *service) processRegionItemDrop(ctx context.Context, user *domain.User, isCritical bool, quantity int, params searchParams) (string, error) {
	log := logger.FromContext(ctx)
	droppedItemName := rollRegionItemDrop(params.region.ItemDrops)
	if droppedItemName == "" {
		log.Warn("Region item drop roll returned empty, falling back to lootbox")
		qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
		if err := s.deps.RewardGranter.GrantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
			return "", err
		}
		item, err := s.deps.ItemLookup.GetItemByName(ctx, domain.ItemLootbox0)
		if err != nil {
			return "", fmt.Errorf("failed to get reward item: %w", err)
		}
		if item == nil {
			return "", domain.ErrItemNotFound
		}
		return s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params), nil
	}

	item, err := s.deps.ItemLookup.GetItemByName(ctx, droppedItemName)
	if err != nil || item == nil {
		log.Error("Failed to get region drop item, falling back to lootbox", "item", droppedItemName, "error", err)
		qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
		if err := s.deps.RewardGranter.GrantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
			return "", err
		}
		lbItem, err := s.deps.ItemLookup.GetItemByName(ctx, domain.ItemLootbox0)
		if err != nil {
			return "", fmt.Errorf("failed to get reward item: %w", err)
		}
		if lbItem == nil {
			return "", domain.ErrItemNotFound
		}
		return s.formatSearchSuccessMessage(ctx, user, lbItem, quantity, isCritical, params), nil
	}

	qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
	if err := s.deps.RewardGranter.GrantItemReward(ctx, user, item, quantity, qualityLevel); err != nil {
		return "", err
	}

	msg := s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params)
	if params.region != nil {
		msg += fmt.Sprintf(" [%s]", params.region.Name)
	}
	log.Info("Region item drop granted", "item", droppedItemName, "region", params.region.Name, "quantity", quantity)
	return msg, nil
}

func (s *service) processSearchFailure(roll float64, successThreshold float64, params searchParams) string {
	failureType := determineSearchFailureType(roll, successThreshold)
	resultMessage := formatSearchFailureMessage(failureType)

	if params.region != nil && params.region.RequiredExplorerLevel > 0 {
		resultMessage += fmt.Sprintf(" [%s]", params.region.Name)
	}

	return formatSearchFailureMessageWithMeta(resultMessage, params)
}

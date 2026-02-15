package crafting

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// RecipeInfo represents recipe information with lock status
type RecipeInfo struct {
	ItemName string              `json:"item_name"`
	Locked   bool                `json:"locked,omitempty"`
	BaseCost []domain.RecipeCost `json:"base_cost,omitempty"`
}

// Result contains the result of an upgrade operation
type Result struct {
	ItemName      string `json:"item_name"`
	Quantity      int    `json:"quantity"`
	IsMasterwork  bool   `json:"is_masterwork"`
	BonusQuantity int    `json:"bonus_quantity"`
}

// DisassembleResult contains the result of a disassemble operation
type DisassembleResult struct {
	Outputs           map[string]int `json:"outputs"`
	QuantityProcessed int            `json:"quantity_processed"`
	IsPerfectSalvage  bool           `json:"is_perfect_salvage"`
	Multiplier        float64        `json:"multiplier"`
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	PublishWithRetry(ctx context.Context, event event.Event)
}

// Service defines the interface for crafting operations
type Service interface {
	UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*Result, error)
	GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*RecipeInfo, error)
	GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]repository.UnlockedRecipeInfo, error)
	GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error)
	DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*DisassembleResult, error)
	Shutdown(ctx context.Context) error
}

// ProgressionService defines the interface for progression operations
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

// JobService defines the interface for checking job levels
type JobService interface {
	GetJobLevel(ctx context.Context, userID, jobKey string) (int, error)
}

// Crafting balance constants are defined in constants.go

type service struct {
	repo           repository.Crafting
	eventPublisher EventPublisher
	progressionSvc ProgressionService
	jobService     JobService      // For checking job level requirements
	namingResolver naming.Resolver // For resolving public names to internal names
	rnd            func() float64  // For rolling RNG (does not need to be cryptographically secure)
}

// NewService creates a new crafting service
func NewService(repo repository.Crafting, eventPublisher EventPublisher, namingResolver naming.Resolver, progressionSvc ProgressionService, jobService JobService) Service {
	return &service{
		repo:           repo,
		eventPublisher: eventPublisher,
		progressionSvc: progressionSvc,
		jobService:     jobService,
		namingResolver: namingResolver,
		rnd:            utils.RandomFloat,
	}
}

// Shutdown gracefully shuts down the crafting service by waiting for all async operations to complete
func (s *service) Shutdown(ctx context.Context) error {
	// No more async operations to wait for
	return nil
}

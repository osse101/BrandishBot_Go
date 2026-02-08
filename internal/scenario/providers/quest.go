package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/quest"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/scenario"
	"github.com/osse101/BrandishBot_Go/internal/scenario/capabilities"
)

const (
	FeatureQuest = "quest"

	// Quest-specific parameters
	ParamQuestID      = "quest_id"
	ParamCount        = "count"
	ParamItemCategory = "item_category"
	ParamQuantity     = "quantity"
	ParamMoneyEarned  = "money_earned"
	ParamRecipeKey    = "recipe_key"
)

// QuestProvider implements the scenario.Provider interface for the quest feature
type QuestProvider struct {
	db        *pgxpool.Pool
	questSvc  quest.Service
	questRepo repository.QuestRepository
	userRepo  repository.User
}

// NewQuestProvider creates a new quest scenario provider
func NewQuestProvider(
	db *pgxpool.Pool,
	questSvc quest.Service,
	questRepo repository.QuestRepository,
	userRepo repository.User,
) *QuestProvider {
	return &QuestProvider{
		db:        db,
		questSvc:  questSvc,
		questRepo: questRepo,
		userRepo:  userRepo,
	}
}

// Feature returns the feature name
func (p *QuestProvider) Feature() string {
	return FeatureQuest
}

// Capabilities returns the capabilities this provider supports
func (p *QuestProvider) Capabilities() []scenario.CapabilityType {
	return []scenario.CapabilityType{
		scenario.CapabilityEventInjector,
	}
}

// GetCapabilityInfo returns detailed capability information
func (p *QuestProvider) GetCapabilityInfo() []scenario.CapabilityInfo {
	return []scenario.CapabilityInfo{
		capabilities.EventInjectorCapabilityInfo(),
	}
}

// SupportsAction returns true if the provider supports the given action
func (p *QuestProvider) SupportsAction(action scenario.ActionType) bool {
	switch action {
	case scenario.ActionSetState,
		scenario.ActionInjectEvent,
		scenario.ActionInjectQuest,
		scenario.ActionTriggerSearch,
		scenario.ActionClaimReward:
		return true
	default:
		return false
	}
}

// PrebuiltScenarios returns the list of pre-built quest scenarios
func (p *QuestProvider) PrebuiltScenarios() []scenario.Scenario {
	return []scenario.Scenario{
		p.searchSpeedrunScenario(),
		p.questProgressionScenario(),
	}
}

// ExecuteStep executes a single step
func (p *QuestProvider) ExecuteStep(ctx context.Context, step scenario.Step, state *scenario.ExecutionState) (*scenario.StepResult, error) {
	result := scenario.NewStepResult(step.Name, 0, step.Action)

	switch step.Action {
	case scenario.ActionSetState:
		return p.executeSetState(ctx, step, state, result)
	case scenario.ActionTriggerSearch:
		return p.executeTriggerSearch(ctx, step, state, result)
	case scenario.ActionInjectEvent:
		return p.executeInjectEvent(ctx, step, state, result)
	case scenario.ActionClaimReward:
		return p.executeClaimReward(ctx, step, state, result)
	default:
		return nil, fmt.Errorf("%w: %s", scenario.ErrInvalidAction, step.Action)
	}
}

// executeSetState initializes the user for quest testing
func (p *QuestProvider) executeSetState(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	// Extract parameters
	username := getStringParam(step.Parameters, ParamUsername, "quest_test_user")
	platform := getStringParam(step.Parameters, ParamPlatform, DefaultPlatform)
	platformID := getStringParam(step.Parameters, "platform_id", "scenario_quest_user")

	// Get or create the user
	user, err := p.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Create the user
			newUser := &domain.User{
				Username:  username,
				DiscordID: platformID,
			}
			if err := p.userRepo.UpsertUser(ctx, newUser); err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
			user, err = p.userRepo.GetUserByPlatformID(ctx, platform, platformID)
			if err != nil {
				return nil, fmt.Errorf("failed to get created user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	}

	// Store user in state
	state.User = &scenario.SimulatedUser{
		UserID:     user.ID,
		Username:   user.Username,
		Platform:   platform,
		PlatformID: platformID,
	}

	// Get active quests
	activeQuests, err := p.questSvc.GetActiveQuests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active quests: %w", err)
	}

	// Get user's quest progress
	progress, err := p.questSvc.GetUserQuestProgress(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quest progress: %w", err)
	}

	result.AddOutput("user_id", user.ID)
	result.AddOutput("username", user.Username)
	result.AddOutput("active_quests", len(activeQuests))
	result.AddOutput("quest_progress", len(progress))

	// Store for later steps
	state.SetResult("active_quests", activeQuests)
	state.SetResult("user_progress", progress)

	return result, nil
}

// executeTriggerSearch triggers search events for quest progress
func (p *QuestProvider) executeTriggerSearch(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	if state.User == nil {
		return nil, scenario.ErrUserNotInitialized
	}

	params, err := capabilities.ParseSearchEventParams(step.Parameters)
	if err != nil {
		return nil, err
	}

	// Trigger search events
	successCount := 0
	for i := 0; i < params.Count; i++ {
		if err := p.questSvc.OnSearch(ctx, state.User.UserID); err != nil {
			result.AddOutput("error_at_search", i+1)
			result.AddOutput("error", err.Error())
			break
		}
		successCount++
	}

	result.AddOutput("searches_triggered", successCount)
	result.AddOutput("searches_requested", params.Count)

	// Get updated progress
	progress, err := p.questSvc.GetUserQuestProgress(ctx, state.User.UserID)
	if err == nil {
		result.AddOutput("updated_progress", progress)
		state.SetResult("user_progress", progress)
	}

	return result, nil
}

// executeInjectEvent injects a specific event type
func (p *QuestProvider) executeInjectEvent(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	if state.User == nil {
		return nil, scenario.ErrUserNotInitialized
	}

	params, err := capabilities.ParseEventInjectorParams(step.Parameters)
	if err != nil {
		return nil, err
	}

	successCount := 0

	for i := 0; i < params.Count; i++ {
		var injectErr error

		switch params.EventType {
		case "search":
			injectErr = p.questSvc.OnSearch(ctx, state.User.UserID)

		case "item_bought":
			category := getStringParam(params.Metadata, "category", "any")
			quantity := getIntParam(params.Metadata, "quantity", 1)
			injectErr = p.questSvc.OnItemBought(ctx, state.User.UserID, category, quantity)

		case "item_sold":
			category := getStringParam(params.Metadata, "category", "any")
			quantity := getIntParam(params.Metadata, "quantity", 1)
			money := getIntParam(params.Metadata, "money_earned", 100)
			injectErr = p.questSvc.OnItemSold(ctx, state.User.UserID, category, quantity, money)

		case "recipe_crafted":
			recipeKey := getStringParam(params.Metadata, "recipe_key", "")
			quantity := getIntParam(params.Metadata, "quantity", 1)
			if recipeKey == "" {
				return nil, scenario.NewParameterError("metadata.recipe_key", "required for recipe_crafted event")
			}
			injectErr = p.questSvc.OnRecipeCrafted(ctx, state.User.UserID, recipeKey, quantity)

		default:
			return nil, fmt.Errorf("%w: unknown event type %s", scenario.ErrInvalidParameter, params.EventType)
		}

		if injectErr != nil {
			result.AddOutput("error_at_event", i+1)
			result.AddOutput("error", injectErr.Error())
			break
		}
		successCount++
	}

	result.AddOutput("events_injected", successCount)
	result.AddOutput("event_type", params.EventType)
	result.AddOutput("events_requested", params.Count)

	// Get updated progress
	progress, err := p.questSvc.GetUserQuestProgress(ctx, state.User.UserID)
	if err == nil {
		result.AddOutput("updated_progress", progress)
		state.SetResult("user_progress", progress)
	}

	return result, nil
}

// executeClaimReward claims a quest reward
func (p *QuestProvider) executeClaimReward(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
	if state.User == nil {
		return nil, scenario.ErrUserNotInitialized
	}

	questID := getIntParam(step.Parameters, ParamQuestID, 0)
	if questID == 0 {
		// Try to get the first completed unclaimed quest
		progress, err := p.questSvc.GetUserQuestProgress(ctx, state.User.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get quest progress: %w", err)
		}

		for _, qp := range progress {
			if qp.CompletedAt != nil && qp.ClaimedAt == nil {
				questID = qp.QuestID
				break
			}
		}

		if questID == 0 {
			result.AddOutput("error", "no completed unclaimed quests found")
			return result, nil
		}
	}

	money, xp, err := p.questSvc.ClaimQuestReward(ctx, state.User.UserID, questID)
	if err != nil {
		result.AddOutput("error", err.Error())
		return result, nil
	}

	result.AddOutput("quest_id", questID)
	result.AddOutput("money_rewarded", money)
	result.AddOutput("xp_rewarded", xp)
	result.AddOutput("claimed", true)

	return result, nil
}

// Pre-built scenario definitions

func (p *QuestProvider) searchSpeedrunScenario() scenario.Scenario {
	return scenario.Scenario{
		ID:          "quest_search_speedrun",
		Name:        "Search Speedrun",
		Description: "Tests completing a search quest by rapidly triggering search events. Sets up user, triggers 10 searches, attempts to claim reward.",
		Feature:     FeatureQuest,
		Capabilities: []scenario.CapabilityType{
			scenario.CapabilityEventInjector,
		},
		Steps: []scenario.Step{
			{
				Name:        "initialize",
				Description: "Set up user for quest testing",
				Action:      scenario.ActionSetState,
				Parameters: map[string]interface{}{
					ParamUsername: "search_speedrunner",
					ParamPlatform: DefaultPlatform,
					"platform_id": "scenario_search_speedrunner",
				},
				Assertions: []scenario.Assertion{
					{
						Type: scenario.AssertNotEmpty,
						Path: "output.user_id",
					},
				},
			},
			{
				Name:        "trigger_searches",
				Description: "Trigger 10 search events to complete any search quest",
				Action:      scenario.ActionTriggerSearch,
				Parameters: map[string]interface{}{
					ParamCount: 10,
				},
				Assertions: []scenario.Assertion{
					{
						Type:  scenario.AssertEquals,
						Path:  "output.searches_triggered",
						Value: 10,
					},
				},
			},
			{
				Name:        "claim_reward",
				Description: "Attempt to claim any completed quest reward",
				Action:      scenario.ActionClaimReward,
				Parameters:  map[string]interface{}{},
				Assertions:  []scenario.Assertion{
					// No assertions - quest might not be available
				},
			},
		},
	}
}

func (p *QuestProvider) questProgressionScenario() scenario.Scenario {
	return scenario.Scenario{
		ID:          "quest_item_progression",
		Name:        "Item Quest Progression",
		Description: "Tests quest progress with item events (buying/selling). Injects various item events and checks progress.",
		Feature:     FeatureQuest,
		Capabilities: []scenario.CapabilityType{
			scenario.CapabilityEventInjector,
		},
		Steps: []scenario.Step{
			{
				Name:        "initialize",
				Description: "Set up user for quest testing",
				Action:      scenario.ActionSetState,
				Parameters: map[string]interface{}{
					ParamUsername: "quest_trader",
					ParamPlatform: DefaultPlatform,
					"platform_id": "scenario_quest_trader",
				},
				Assertions: []scenario.Assertion{
					{
						Type: scenario.AssertNotEmpty,
						Path: "output.user_id",
					},
				},
			},
			{
				Name:        "inject_buy_events",
				Description: "Inject 5 item bought events",
				Action:      scenario.ActionInjectEvent,
				Parameters: map[string]interface{}{
					"event_type": "item_bought",
					ParamCount:   5,
					"metadata": map[string]interface{}{
						"category": "any",
						"quantity": 1,
					},
				},
				Assertions: []scenario.Assertion{
					{
						Type:  scenario.AssertEquals,
						Path:  "output.events_injected",
						Value: 5,
					},
				},
			},
			{
				Name:        "inject_sell_events",
				Description: "Inject 5 item sold events",
				Action:      scenario.ActionInjectEvent,
				Parameters: map[string]interface{}{
					"event_type": "item_sold",
					ParamCount:   5,
					"metadata": map[string]interface{}{
						"category":     "any",
						"quantity":     1,
						"money_earned": 100,
					},
				},
				Assertions: []scenario.Assertion{
					{
						Type:  scenario.AssertEquals,
						Path:  "output.events_injected",
						Value: 5,
					},
				},
			},
		},
	}
}

// Helper functions

func getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultVal
}

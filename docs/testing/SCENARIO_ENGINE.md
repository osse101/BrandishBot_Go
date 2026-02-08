# Scenario Engine Usage Guide

The Scenario Engine is an admin testing framework that allows controlled manipulation of game state and time for feature testing without manual setup.

## Overview

The engine provides:
- **State Injection**: Initialize users, inventories, and feature-specific state
- **Temporal Warping**: Manipulate time-dependent features via direct DB timestamp updates
- **Event Injection**: Trigger events without real user actions
- **Assertions**: Validate expected outcomes at each step

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────────┐
│   Handler   │────▶│    Engine    │────▶│   Providers    │
│  /simulate  │     │  (executor)  │     │ (feature impl) │
└─────────────┘     └──────────────┘     └────────────────┘
                           │
                    ┌──────┴──────┐
                    │  Registry   │
                    │ (providers) │
                    └─────────────┘
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/simulate/capabilities` | List available capabilities |
| GET | `/api/v1/admin/simulate/scenarios` | List all pre-built scenarios |
| GET | `/api/v1/admin/simulate/scenario?id=X` | Get specific scenario details |
| POST | `/api/v1/admin/simulate/run` | Execute a pre-built scenario |
| POST | `/api/v1/admin/simulate/run-custom` | Execute a custom scenario |

## Running a Pre-built Scenario

```bash
# List available scenarios
curl http://localhost:8080/api/v1/admin/simulate/scenarios \
  -H "X-API-Key: your-key"

# Run the "Patient Farmer" harvest scenario
curl -X POST http://localhost:8080/api/v1/admin/simulate/run \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_id": "harvest_patient_farmer"
  }'
```

### Example Response

```json
{
  "scenario_id": "harvest_patient_farmer",
  "scenario_name": "The Patient Farmer",
  "success": true,
  "duration_ms": 45,
  "started_at": "2024-01-15T10:00:00Z",
  "completed_at": "2024-01-15T10:00:00Z",
  "steps": [
    {
      "step_name": "initialize",
      "step_index": 0,
      "action": "set_state",
      "success": true,
      "duration_ms": 12,
      "output": {
        "user_id": "abc-123",
        "username": "patient_farmer",
        "harvest_state_initialized": true
      },
      "assertions": [
        {"type": "not_empty", "path": "output.user_id", "passed": true}
      ]
    },
    {
      "step_name": "time_warp",
      "step_index": 1,
      "action": "time_warp",
      "success": true,
      "duration_ms": 8,
      "output": {
        "warped_hours": 168,
        "simulated_elapsed_hours": 168
      }
    },
    {
      "step_name": "execute_harvest",
      "step_index": 2,
      "action": "execute_harvest",
      "success": true,
      "duration_ms": 25,
      "output": {
        "items_gained": {"money": 97, "lootbox1": 3},
        "hours_since_harvest": 168.002
      }
    }
  ],
  "user": {
    "user_id": "abc-123",
    "username": "patient_farmer",
    "platform": "discord"
  }
}
```

## Running a Custom Scenario

```bash
curl -X POST http://localhost:8080/api/v1/admin/simulate/run-custom \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": {
      "id": "custom_harvest_test",
      "name": "Custom Harvest Test",
      "feature": "harvest",
      "steps": [
        {
          "name": "setup",
          "action": "set_state",
          "parameters": {
            "username": "custom_user",
            "platform": "discord",
            "platform_id": "custom_platform_id"
          }
        },
        {
          "name": "warp_48h",
          "action": "time_warp",
          "parameters": {"hours": 48},
          "assertions": [
            {"type": "equals", "path": "output.warped_hours", "value": 48}
          ]
        },
        {
          "name": "harvest",
          "action": "execute_harvest",
          "assertions": [
            {"type": "not_empty", "path": "output.items_gained"},
            {"type": "greater_than", "path": "output.hours_since_harvest", "value": 40}
          ]
        }
      ]
    }
  }'
```

## Available Scenarios

### Harvest Feature

| ID | Name | Description |
|----|------|-------------|
| `harvest_patient_farmer` | The Patient Farmer | Max tier (168h) harvest |
| `harvest_spoiled` | Spoiled Harvest | Tests spoilage after 336h |
| `harvest_first_time` | First Time Farmer | First harvest initialization |
| `harvest_quick` | Quick Harvest | Minimum (1h) harvest |

### Quest Feature

| ID | Name | Description |
|----|------|-------------|
| `quest_search_speedrun` | Search Speedrun | Trigger 10 searches, attempt claim |
| `quest_item_progression` | Item Quest Progression | Inject buy/sell events |

## Capabilities

### TimeWarp

Manipulates time-dependent features by updating database timestamps directly.

**Actions:**
- `time_warp` - Set last action time to N hours ago

**Parameters:**
- `hours` (required): Number of hours to simulate elapsed

### EventInjector

Triggers feature events without real user actions.

**Actions:**
- `inject_event` - Inject a specific event type
- `trigger_search` - Trigger search events (quest-specific)

**Parameters:**
- `event_type`: Type of event (`search`, `item_bought`, `item_sold`, `recipe_crafted`)
- `count`: Number of events to inject
- `metadata`: Additional event data

## Assertion Types

| Type | Description | Parameters |
|------|-------------|------------|
| `equals` | Exact value match | `value` |
| `greater_than` | Numeric comparison | `value` |
| `less_than` | Numeric comparison | `value` |
| `between` | Range check | `min`, `max` |
| `contains` | String substring | `value` |
| `not_empty` | Value exists and non-empty | - |
| `empty` | Value is empty/null | - |
| `true` | Boolean true | - |
| `false` | Boolean false | - |
| `error_contains` | Error message substring | `value` |

### Path Notation

Assertions use dot-notation paths:
- `output.items_gained` - Step output field
- `output.items_gained.money` - Nested field
- `state.harvest_result` - Execution state
- `user.username` - Simulated user field

## Writing a New Provider

### 1. Implement the Provider Interface

```go
package providers

import (
    "context"
    "github.com/osse101/BrandishBot_Go/internal/scenario"
)

type MyFeatureProvider struct {
    // Dependencies
    db         *pgxpool.Pool
    featureSvc myfeature.Service
}

func (p *MyFeatureProvider) Feature() string {
    return "myfeature"
}

func (p *MyFeatureProvider) Capabilities() []scenario.CapabilityType {
    return []scenario.CapabilityType{
        scenario.CapabilityTimeWarp,
        scenario.CapabilityEventInjector,
    }
}

func (p *MyFeatureProvider) SupportsAction(action scenario.ActionType) bool {
    switch action {
    case scenario.ActionSetState, scenario.ActionTimeWarp:
        return true
    }
    return false
}

func (p *MyFeatureProvider) ExecuteStep(ctx context.Context, step scenario.Step, state *scenario.ExecutionState) (*scenario.StepResult, error) {
    result := scenario.NewStepResult(step.Name, 0, step.Action)

    switch step.Action {
    case scenario.ActionSetState:
        // Initialize user/state
        return p.executeSetState(ctx, step, state, result)
    case scenario.ActionTimeWarp:
        // Manipulate timestamps
        return p.executeTimeWarp(ctx, step, state, result)
    default:
        return nil, scenario.ErrInvalidAction
    }
}

func (p *MyFeatureProvider) PrebuiltScenarios() []scenario.Scenario {
    return []scenario.Scenario{
        // Define pre-built scenarios here
    }
}
```

### 2. Register the Provider

In `cmd/app/main.go`:

```go
// Initialize provider
myProvider := providers.NewMyFeatureProvider(db, myFeatureSvc)
scenarioRegistry.Register(myProvider)
```

## Best Practices

1. **Unique Test Users**: Use unique `platform_id` values per scenario to avoid state pollution
2. **Cleanup**: Scenarios create test data - consider cleanup strategies for production
3. **Assertions**: Add assertions at each step to catch failures early
4. **Descriptive Names**: Use clear step names for debugging failed scenarios
5. **Error Output**: Always capture errors in step output for debugging

## Designing Complex Scenarios: Gamble Feature Example

The gamble feature demonstrates a complex multi-user scenario with:
- Multiple users participating
- Variable bet items (different lootbox types)
- State transitions: `joining` → `opening` → `completed`
- Inventory setup and consumption

### Gamble Provider Design

A gamble provider would require the `MultiUser` capability in addition to state management.

```go
// internal/scenario/providers/gamble.go
package providers

const FeatureGamble = "gamble"

// Custom action types for gamble
const (
    ActionCreateUser     scenario.ActionType = "create_user"
    ActionSetInventory   scenario.ActionType = "set_inventory"
    ActionStartGamble    scenario.ActionType = "start_gamble"
    ActionJoinGamble     scenario.ActionType = "join_gamble"
    ActionTimeWarpDeadline scenario.ActionType = "time_warp_deadline"
    ActionExecuteGamble  scenario.ActionType = "execute_gamble"
)

type GambleProvider struct {
    db         *pgxpool.Pool
    gambleSvc  gamble.Service
    userRepo   repository.User
    users      []*scenario.SimulatedUser  // Multi-user tracking
}

func (p *GambleProvider) Capabilities() []scenario.CapabilityType {
    return []scenario.CapabilityType{
        scenario.CapabilityTimeWarp,
        scenario.CapabilityMultiUser,  // Required for multi-user support
    }
}
```

### Pre-built Gamble Scenarios

#### 1. Two-Player Junkbox Gamble

```go
func (p *GambleProvider) twoPlayerJunkboxScenario() scenario.Scenario {
    return scenario.Scenario{
        ID:          "gamble_two_player_junkbox",
        Name:        "Two Player Junkbox Duel",
        Description: "Two users gamble with 3 junkboxes each. Tests full gamble lifecycle.",
        Feature:     FeatureGamble,
        Capabilities: []scenario.CapabilityType{
            scenario.CapabilityTimeWarp,
            scenario.CapabilityMultiUser,
        },
        Steps: []scenario.Step{
            // Step 1: Create first user with inventory
            {
                Name:        "create_initiator",
                Description: "Create initiator user with junkboxes",
                Action:      ActionCreateUser,
                Parameters: map[string]interface{}{
                    "username":    "gambler_1",
                    "platform":    "discord",
                    "platform_id": "scenario_gambler_1",
                    "inventory": map[string]int{
                        "lootbox_tier0": 5,  // 5 junkboxes
                    },
                },
                Assertions: []scenario.Assertion{
                    {Type: scenario.AssertNotEmpty, Path: "output.user_id"},
                    {Type: scenario.AssertEquals, Path: "output.inventory.lootbox_tier0", Value: 5},
                },
            },
            // Step 2: Create second user with inventory
            {
                Name:        "create_joiner",
                Description: "Create joiner user with junkboxes",
                Action:      ActionCreateUser,
                Parameters: map[string]interface{}{
                    "username":    "gambler_2",
                    "platform":    "discord",
                    "platform_id": "scenario_gambler_2",
                    "inventory": map[string]int{
                        "lootbox_tier0": 5,
                    },
                },
                Assertions: []scenario.Assertion{
                    {Type: scenario.AssertNotEmpty, Path: "output.user_id"},
                },
            },
            // Step 3: Start gamble as initiator
            {
                Name:        "start_gamble",
                Description: "Initiator starts gamble with 3 junkboxes",
                Action:      ActionStartGamble,
                Parameters: map[string]interface{}{
                    "user_index": 0,  // Use first created user
                    "bets": []map[string]interface{}{
                        {"item_name": "junkbox", "quantity": 3},
                    },
                },
                Assertions: []scenario.Assertion{
                    {Type: scenario.AssertNotEmpty, Path: "output.gamble_id"},
                    {Type: scenario.AssertEquals, Path: "output.state", Value: "joining"},
                },
            },
            // Step 4: Join gamble as second user
            {
                Name:        "join_gamble",
                Description: "Second user joins the gamble",
                Action:      ActionJoinGamble,
                Parameters: map[string]interface{}{
                    "user_index": 1,  // Use second created user
                },
                Assertions: []scenario.Assertion{
                    {Type: scenario.AssertEquals, Path: "output.participant_count", Value: 2},
                },
            },
            // Step 5: Warp past join deadline
            {
                Name:        "warp_past_deadline",
                Description: "Warp time past the join deadline",
                Action:      ActionTimeWarpDeadline,
                Parameters: map[string]interface{}{
                    "minutes_past": 5,  // 5 minutes past deadline
                },
            },
            // Step 6: Execute gamble
            {
                Name:        "execute_gamble",
                Description: "Execute the gamble and determine winner",
                Action:      ActionExecuteGamble,
                Parameters:  map[string]interface{}{},
                Assertions: []scenario.Assertion{
                    {Type: scenario.AssertNotEmpty, Path: "output.winner_id"},
                    {Type: scenario.AssertEquals, Path: "output.state", Value: "completed"},
                    {Type: scenario.AssertGreaterThan, Path: "output.total_value", Value: 0},
                },
            },
        },
    }
}
```

#### 2. Multi-Lootbox Variety Gamble

```go
func (p *GambleProvider) multiLootboxScenario() scenario.Scenario {
    return scenario.Scenario{
        ID:          "gamble_multi_lootbox",
        Name:        "Multi-Lootbox High Stakes",
        Description: "Three users gamble with a mix of lootbox types",
        Feature:     FeatureGamble,
        Steps: []scenario.Step{
            // Create 3 users with different lootbox types
            {
                Name:   "create_user_1",
                Action: ActionCreateUser,
                Parameters: map[string]interface{}{
                    "username":    "high_roller",
                    "platform_id": "scenario_high_roller",
                    "inventory": map[string]int{
                        "lootbox_tier0": 10,  // 10 junkboxes
                        "lootbox_tier1": 5,   // 5 decent lootboxes
                        "lootbox_tier2": 2,   // 2 good lootboxes
                    },
                },
            },
            {
                Name:   "create_user_2",
                Action: ActionCreateUser,
                Parameters: map[string]interface{}{
                    "username":    "cautious_player",
                    "platform_id": "scenario_cautious",
                    "inventory": map[string]int{
                        "lootbox_tier0": 10,
                        "lootbox_tier1": 5,
                        "lootbox_tier2": 2,
                    },
                },
            },
            {
                Name:   "create_user_3",
                Action: ActionCreateUser,
                Parameters: map[string]interface{}{
                    "username":    "risk_taker",
                    "platform_id": "scenario_risk_taker",
                    "inventory": map[string]int{
                        "lootbox_tier0": 10,
                        "lootbox_tier1": 5,
                        "lootbox_tier2": 2,
                    },
                },
            },
            // Start with mixed bets
            {
                Name:   "start_gamble",
                Action: ActionStartGamble,
                Parameters: map[string]interface{}{
                    "user_index": 0,
                    "bets": []map[string]interface{}{
                        {"item_name": "junkbox", "quantity": 2},
                        {"item_name": "lootbox1", "quantity": 1},  // Decent lootbox
                    },
                },
            },
            // Users 2 and 3 join
            {Name: "join_user_2", Action: ActionJoinGamble, Parameters: map[string]interface{}{"user_index": 1}},
            {Name: "join_user_3", Action: ActionJoinGamble, Parameters: map[string]interface{}{"user_index": 2}},
            // Execute
            {Name: "warp_deadline", Action: ActionTimeWarpDeadline, Parameters: map[string]interface{}{"minutes_past": 5}},
            {
                Name:   "execute",
                Action: ActionExecuteGamble,
                Assertions: []scenario.Assertion{
                    {Type: scenario.AssertEquals, Path: "output.participant_count", Value: 3},
                    {Type: scenario.AssertNotEmpty, Path: "output.winner_id"},
                },
            },
        },
    }
}
```

### Step Implementation Examples

#### CreateUser with Inventory

```go
func (p *GambleProvider) executeCreateUser(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
    username := getStringParam(step.Parameters, "username", "gamble_test_user")
    platform := getStringParam(step.Parameters, "platform", "discord")
    platformID := getStringParam(step.Parameters, "platform_id", fmt.Sprintf("scenario_%d", time.Now().UnixNano()))

    // Create or get user
    user, err := p.getOrCreateUser(ctx, platform, platformID, username)
    if err != nil {
        return nil, err
    }

    // Set up inventory if specified
    if inventory, ok := step.Parameters["inventory"].(map[string]interface{}); ok {
        if err := p.setUserInventory(ctx, user.ID, inventory); err != nil {
            return nil, err
        }
    }

    // Track user in multi-user state
    simUser := &scenario.SimulatedUser{
        UserID:     user.ID,
        Username:   username,
        Platform:   platform,
        PlatformID: platformID,
    }
    p.users = append(p.users, simUser)

    // Set as active user in state
    state.User = simUser
    state.SetResult("users", p.users)

    result.AddOutput("user_id", user.ID)
    result.AddOutput("user_index", len(p.users)-1)

    // Return inventory state
    inv, _ := p.userRepo.GetInventory(ctx, user.ID)
    invMap := make(map[string]int)
    for _, slot := range inv.Slots {
        item, _ := p.userRepo.GetItemByID(ctx, slot.ItemID)
        if item != nil {
            invMap[item.InternalName] = slot.Quantity
        }
    }
    result.AddOutput("inventory", invMap)

    return result, nil
}
```

#### StartGamble

```go
func (p *GambleProvider) executeStartGamble(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
    // Get user by index
    userIndex := getIntParam(step.Parameters, "user_index", 0)
    if userIndex >= len(p.users) {
        return nil, fmt.Errorf("user_index %d out of range", userIndex)
    }
    user := p.users[userIndex]

    // Parse bets
    betsRaw, ok := step.Parameters["bets"].([]interface{})
    if !ok {
        return nil, scenario.NewParameterError("bets", "required")
    }

    var bets []domain.LootboxBet
    for _, b := range betsRaw {
        betMap := b.(map[string]interface{})
        bets = append(bets, domain.LootboxBet{
            ItemName: betMap["item_name"].(string),
            Quantity: int(betMap["quantity"].(float64)),
        })
    }

    // Call real service
    gamble, err := p.gambleSvc.StartGamble(ctx, user.Platform, user.PlatformID, user.Username, bets)
    if err != nil {
        result.AddOutput("error", err.Error())
        return result, nil
    }

    // Store gamble ID for subsequent steps
    state.SetResult("gamble_id", gamble.ID.String())
    state.SetResult("gamble", gamble)

    result.AddOutput("gamble_id", gamble.ID.String())
    result.AddOutput("state", string(gamble.State))
    result.AddOutput("join_deadline", gamble.JoinDeadline.Format(time.RFC3339))

    return result, nil
}
```

#### TimeWarp for Gamble Deadline

```go
func (p *GambleProvider) executeTimeWarpDeadline(ctx context.Context, step scenario.Step, state *scenario.ExecutionState, result *scenario.StepResult) (*scenario.StepResult, error) {
    gambleIDStr, ok := state.GetResult("gamble_id")
    if !ok {
        return nil, fmt.Errorf("no active gamble in state")
    }

    gambleID, _ := uuid.Parse(gambleIDStr.(string))
    minutesPast := getIntParam(step.Parameters, "minutes_past", 5)

    // Calculate new deadline (in the past)
    newDeadline := time.Now().Add(-time.Duration(minutesPast) * time.Minute)

    // Direct DB update to manipulate join_deadline
    query := `UPDATE gambles SET join_deadline = $1 WHERE id = $2`
    _, err := p.db.Exec(ctx, query, newDeadline, gambleID)
    if err != nil {
        return nil, scenario.WrapDatabaseError("update gamble deadline", err)
    }

    result.AddOutput("new_deadline", newDeadline.Format(time.RFC3339))
    result.AddOutput("minutes_past", minutesPast)

    return result, nil
}
```

### Custom Scenario via API

```bash
curl -X POST http://localhost:8080/api/v1/admin/simulate/run-custom \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": {
      "id": "custom_gamble_test",
      "name": "Custom 2-Player Gamble",
      "feature": "gamble",
      "capabilities": ["time_warp", "multi_user"],
      "steps": [
        {
          "name": "create_player_1",
          "action": "create_user",
          "parameters": {
            "username": "alice",
            "platform_id": "alice_123",
            "inventory": {"lootbox_tier0": 10}
          }
        },
        {
          "name": "create_player_2",
          "action": "create_user",
          "parameters": {
            "username": "bob",
            "platform_id": "bob_456",
            "inventory": {"lootbox_tier0": 10}
          }
        },
        {
          "name": "start",
          "action": "start_gamble",
          "parameters": {
            "user_index": 0,
            "bets": [{"item_name": "junkbox", "quantity": 5}]
          }
        },
        {
          "name": "join",
          "action": "join_gamble",
          "parameters": {"user_index": 1}
        },
        {
          "name": "warp",
          "action": "time_warp_deadline",
          "parameters": {"minutes_past": 5}
        },
        {
          "name": "execute",
          "action": "execute_gamble",
          "assertions": [
            {"type": "not_empty", "path": "output.winner_id"},
            {"type": "greater_than", "path": "output.total_value", "value": 0}
          ]
        }
      ]
    }
  }'
```

### Key Design Patterns for Multi-User Scenarios

1. **User Index Tracking**: Store created users in an array, reference by index
2. **Shared State**: Store gamble ID and other shared state for subsequent steps
3. **Inventory Setup**: Set exact inventory contents before testing
4. **Time Manipulation**: Update deadline timestamps directly in DB
5. **Sequential Flow**: Steps execute in order, each building on previous state

## Debugging

Failed scenarios include detailed error information:

```json
{
  "success": false,
  "steps": [
    {
      "step_name": "execute_harvest",
      "success": false,
      "output": {
        "error": "harvest requires farming feature to be unlocked"
      },
      "assertions": [
        {
          "type": "not_empty",
          "path": "output.items_gained",
          "passed": false,
          "error": "path 'output.items_gained' not found"
        }
      ]
    }
  ]
}
```

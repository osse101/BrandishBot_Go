package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// MockAPIClient is a mock implementation of APIClient for testing
type MockAPIClient struct {
	RegisterUserFunc    func(string, string) (*domain.User, error)
	SearchFunc          func(string, string, string) (string, error)
	GetInventoryFunc    func(string, string, string) ([]user.InventoryItem, error)
	UseItemFunc         func(string, string, string, string, int) (string, error)
	BuyItemFunc         func(string, string, string, string, int) (string, error)
	SellItemFunc        func(string, string, string, string, int) (string, error)
	GetSellPricesFunc   func() (string, error)
	GetBuyPricesFunc    func() (string, error)
	GiveItemFunc        func(string, string, string, string, string, string, int) (string, error)
	UpgradeItemFunc     func(string, string, string, int) (string, error)
	DisassembleItemFunc func(string, string, string, string, int) (string, error)
	GetRecipesFunc      func() (string, error)
	GetLeaderboardFunc  func(string, int) (string, error)
	GetUserStatsFunc    func(string, string) (string, error)
	AddItemFunc         func(string, string, string, int) (string, error)
	RemoveItemFunc      func(string, string, string, int) (string, error)
	StartGambleFunc     func(string, string, string, string, int) (string, error)
	JoinGambleFunc      func(string, string, string, string, string, int) (string, error)
	VoteForNodeFunc     func(string, string, string, string) (string, error)
}

func (m *MockAPIClient) RegisterUser(username, discordID string) (*domain.User, error) {
	if m.RegisterUserFunc != nil {
		return m.RegisterUserFunc(username, discordID)
	}
	return &domain.User{ID: "test-user-id", Username: username}, nil
}

func (m *MockAPIClient) Search(platform, platformID, username string) (string, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(platform, platformID, username)
	}
	return "Found 10 money!", nil
}

func (m *MockAPIClient) GetInventory(platform, platformID, username string) ([]user.InventoryItem, error) {
	if m.GetInventoryFunc != nil {
		return m.GetInventoryFunc(platform, platformID, username)
	}
	return []user.InventoryItem{}, nil
}

func (m *MockAPIClient) UseItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	if m.UseItemFunc != nil {
		return m.UseItemFunc(platform, platformID, username, itemName, quantity)
	}
	return "Used item successfully", nil
}

func (m *MockAPIClient) BuyItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	if m.BuyItemFunc != nil {
		return m.BuyItemFunc(platform, platformID, username, itemName, quantity)
	}
	return "Purchased 1x " + itemName, nil
}

func (m *MockAPIClient) SellItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	if m.SellItemFunc != nil {
		return m.SellItemFunc(platform, platformID, username, itemName, quantity)
	}
	return "Sold 1x " + itemName, nil
}

func (m *MockAPIClient) GetSellPrices() (string, error) {
	if m.GetSellPricesFunc != nil {
		return m.GetSellPricesFunc()
	}
	return "lootbox0: 10 coins", nil
}

func (m *MockAPIClient) GetBuyPrices() (string, error) {
	if m.GetBuyPricesFunc != nil {
		return m.GetBuyPricesFunc()
	}
	return "lootbox0: 15 coins", nil
}

func (m *MockAPIClient) GiveItem(fromPlatform, fromPlatformID, toPlatform, toPlatformID, toUsername, itemName string, quantity int) (string, error) {
	if m.GiveItemFunc != nil {
		return m.GiveItemFunc(fromPlatform, fromPlatformID, toPlatform, toPlatformID, toUsername, itemName, quantity)
	}
	return "Gave item successfully", nil
}

func (m *MockAPIClient) UpgradeItem(platform, platformID, username string, recipeID int) (string, error) {
	if m.UpgradeItemFunc != nil {
		return m.UpgradeItemFunc(platform, platformID, username, recipeID)
	}
	return "Crafted item successfully", nil
}

func (m *MockAPIClient) DisassembleItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	if m.DisassembleItemFunc != nil {
		return m.DisassembleItemFunc(platform, platformID, username, itemName, quantity)
	}
	return "Disassembled item successfully", nil
}

func (m *MockAPIClient) GetRecipes() (string, error) {
	if m.GetRecipesFunc != nil {
		return m.GetRecipesFunc()
	}
	return "Recipe 1: Basic Sword", nil
}

func (m *MockAPIClient) GetLeaderboard(metric string, limit int) (string, error) {
	if m.GetLeaderboardFunc != nil {
		return m.GetLeaderboardFunc(metric, limit)
	}
	return "1. Player1 - 1000 points", nil
}

func (m *MockAPIClient) GetUserStats(platform, platformID string) (string, error) {
	if m.GetUserStatsFunc != nil {
		return m.GetUserStatsFunc(platform, platformID)
	}
	return "Total events: 42", nil
}

func (m *MockAPIClient) AddItem(platform, platformID, itemName string, quantity int) (string, error) {
	if m.AddItemFunc != nil {
		return m.AddItemFunc(platform, platformID, itemName, quantity)
	}
	return "Added items successfully", nil
}

func (m *MockAPIClient) RemoveItem(platform, platformID, itemName string, quantity int) (string, error) {
	if m.RemoveItemFunc != nil {
		return m.RemoveItemFunc(platform, platformID, itemName, quantity)
	}
	return "Removed items successfully", nil
}

func (m *MockAPIClient) StartGamble(platform, platformID, username, itemName string, quantity int) (string, error) {
	if m.StartGambleFunc != nil {
		return m.StartGambleFunc(platform, platformID, username, itemName, quantity)
	}
	return "Gamble started!", nil
}

func (m *MockAPIClient) JoinGamble(platform, platformID, username, gambleID, itemName string, quantity int) (string, error) {
	if m.JoinGambleFunc != nil {
		return m.JoinGambleFunc(platform, platformID, username, gambleID, itemName, quantity)
	}
	return "Joined gamble!", nil
}

func (m *MockAPIClient) VoteForNode(platform, platformID, username, nodeKey string) (string, error) {
	if m.VoteForNodeFunc != nil {
		return m.VoteForNodeFunc(platform, platformID, username, nodeKey)
	}
	return "Vote recorded!", nil
}

func (m *MockAPIClient) AdminUnlockNode(nodeKey string, level int) (string, error) {
	return "Node unlocked", nil
}

// Helper to create test interaction
func createTestInteraction(commandName string, options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name:    commandName,
				Options: options,
			},
			User: &discordgo.User{
				ID:       "test-user-123",
				Username: "TestUser",
			},
		},
	}
}

// TestCommandRegistry tests the command registry
func TestCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	cmd := &discordgo.ApplicationCommand{
		Name:        "test",
		Description: "Test command",
	}

	handlerCalled := false
	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handlerCalled = true
	}

	registry.Register(cmd, handler)

	if registry.Commands["test"] == nil {
		t.Error("Command not registered")
	}

	if registry.Handlers["test"] == nil {
		t.Error("Handler not registered")
	}

	// Test handle
	registry.Handle(nil, createTestInteraction("test", nil), nil)

	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

// TestRecordCommand tests command tracking
func TestRecordCommand(t *testing.T) {
	// Reset counter
	commandCounter = 0

	RecordCommand()
	RecordCommand()
	RecordCommand()

	if commandCounter != 3 {
		t.Errorf("Expected 3 commands, got %d", commandCounter)
	}
}

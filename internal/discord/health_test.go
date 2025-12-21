package discord

import (
	"errors"
	"testing"
)

// TestHealthStatus tests the health check endpoint
func TestHealthStatus(t *testing.T) {
	// Reset counters
	commandCounter = 0

	// Record some commands
	RecordCommand()
	RecordCommand()

	status := HealthStatus{
		Status:           "healthy",
		Uptime:           "1h",
		Connected:        true,
		CommandsReceived: 2,
		APIReachable:     true,
	}

	if status.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", status.Status)
	}

	if status.CommandsReceived != 2 {
		t.Errorf("Expected 2 commands, got %d", status.CommandsReceived)
	}
}

// TestMockAPIClient tests the mock client
func TestMockAPIClient(t *testing.T) {
	mock := &MockAPIClient{}

	// Test default implementations
	user, err := mock.RegisterUser("test", "123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if user.Username != "test" {
		t.Errorf("Expected username 'test', got %s", user.Username)
	}

	// Test custom function
	mock.SearchFunc = func(platform, platformID, username string) (string, error) {
		return "Custom search result", nil
	}

	result, err := mock.Search("discord", "123", "test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "Custom search result" {
		t.Errorf("Expected custom result, got %s", result)
	}

	// Test error handling
	mock.BuyItemFunc = func(platform, platformID, username, itemName string, quantity int) (string, error) {
		return "", errors.New("insufficient funds")
	}

	_, err = mock.BuyItem("discord", "123", "test", "sword", 1)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "insufficient funds" {
		t.Errorf("Expected 'insufficient funds', got %s", err.Error())
	}
}

// TestEconomyCommands tests economy-related mocks
func TestEconomyCommands(t *testing.T) {
	mock := &MockAPIClient{}

	tests := []struct {
		name     string
		testFunc func() (string, error)
		want     string
	}{
		{
			name: "Buy Item",
			testFunc: func() (string, error) {
				return mock.BuyItem("discord", "123", "test", "lootbox0", 1)
			},
			want: "Purchased 1x lootbox0",
		},
		{
			name: "Sell Item",
			testFunc: func() (string, error) {
				return mock.SellItem("discord", "123", "test", "lootbox0", 1)
			},
			want: "Sold 1x lootbox0",
		},
		{
			name: "Get Sell Prices",
			testFunc: func() (string, error) {
				return mock.GetSellPrices()
			},
			want: "lootbox0: 10 coins",
		},
		{
			name: "Get Buy Prices",
			testFunc: func() (string, error) {
				return mock.GetBuyPrices()
			},
			want: "lootbox0: 15 coins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.testFunc()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Want %s, got %s", tt.want, got)
			}
		})
	}
}

// TestCraftingCommands tests crafting-related mocks
func TestCraftingCommands(t *testing.T) {
	mock := &MockAPIClient{}

	tests := []struct {
		name     string
		testFunc func() (string, error)
		wantErr  bool
	}{
		{
			name: "Upgrade Item",
			testFunc: func() (string, error) {
				return mock.UpgradeItem("discord", "123", "test", 1)
			},
			wantErr: false,
		},
		{
			name: "Disassemble Item",
			testFunc: func() (string, error) {
				return mock.DisassembleItem("discord", "123", "test", "sword", 1)
			},
			wantErr: false,
		},
		{
			name: "Get Recipes",
			testFunc: func() (string, error) {
				return mock.GetRecipes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.testFunc()
			if (err != nil) != tt.wantErr {
				t.Errorf("Want error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && result == "" {
				t.Error("Expected non-empty result")
			}
		})
	}
}

// TestStatsCommands tests stats-related mocks
func TestStatsCommands(t *testing.T) {
	mock := &MockAPIClient{}

	// Test leaderboard
	result, err := mock.GetLeaderboard("money", 10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty leaderboard")
	}

	// Test user stats
	result, err = mock.GetUserStats("discord", "123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty stats")
	}
}

// TestAdminCommands tests admin-related mocks
func TestAdminCommands(t *testing.T) {
	mock := &MockAPIClient{}

	// Test add item
	result, err := mock.AddItem("discord", "123", "sword", 5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Test remove item
	result, err = mock.RemoveItem("discord", "123", "sword", 2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

package progression

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequiredNodes_NoPrerequisites(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Money only requires root, which is auto-unlocked
	required, err := service.GetRequiredNodes(ctx, "item_money")
	assert.NoError(t, err)
	assert.Empty(t, required, "Money should have no locked prerequisites (root is unlocked)")
}

func TestGetRequiredNodes_DirectPrerequisite(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Economy requires money (which is NOT unlocked)
	required, err := service.GetRequiredNodes(ctx, "feature_economy")
	assert.NoError(t, err)
	assert.Len(t, required, 1)
	assert.Equal(t, "item_money", required[0].NodeKey)
}

func TestGetRequiredNodes_MultiplePrerequisites(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Upgrade requires lootbox0 (which is NOT unlocked)
	required, err := service.GetRequiredNodes(ctx, FeatureUpgrade)
	assert.NoError(t, err)
	assert.Len(t, required, 1)

	// Should include lootbox0
	keys := make(map[string]bool)
	for _, node := range required {
		keys[node.NodeKey] = true
	}
	assert.True(t, keys["item_lootbox0"])
}

func TestGetRequiredNodes_PartiallyUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock money
	repo.UnlockNode(ctx, 2, 1, "test", 0)

	// Now economy only requires money (which IS unlocked)
	required, err := service.GetRequiredNodes(ctx, "feature_economy")
	assert.NoError(t, err)
	assert.Empty(t, required, "Economy should have no locked prerequisites after unlocking money")
}

func TestGetRequiredNodes_AllUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock the lootbox0 chain
	repo.UnlockNode(ctx, 4, 1, "test", 0) // lootbox0

	// Upgrade should have no locked prerequisites
	required, err := service.GetRequiredNodes(ctx, FeatureUpgrade)
	assert.NoError(t, err)
	assert.Empty(t, required)
}

func TestGetRequiredNodes_NodeNotFound(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	_, err := service.GetRequiredNodes(ctx, "nonexistent_node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node not found")
}

package item

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestLoader_SyncToDatabase_Unchanged(t *testing.T) {
	content := `{
		"version": "1.0",
		"description": "Test items",
		"items": [
			{
				"internal_name": "test_item",
				"public_name": "testitem",
				"description": "A test item",
				"tier": 0,
				"max_stack": 10,
				"base_value": 100,
				"tags": ["consumable"],
				"handler": "lootbox",
				"default_display": "Test Box"
			}
		]
	}`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	loader := NewLoader()
	config, err := loader.Load(tmpFile)
	require.NoError(t, err)

	mockRepo := mocks.NewMockRepositoryItem(t)

	fileInfo, err := os.Stat(tmpFile)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	// Return matching metadata to signify file hasn't changed
	mockRepo.On("GetSyncMetadata", mock.Anything, ConfigFileName).Return(&domain.SyncMetadata{
		FileHash:    fileHash,
		FileModTime: fileInfo.ModTime(),
	}, nil)

	result, err := loader.SyncToDatabase(context.Background(), config, mockRepo, tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, &SyncResult{}, result)

	mockRepo.AssertExpectations(t)
}

func TestLoader_SyncToDatabase_InsertNewItem(t *testing.T) {
	content := `{
		"version": "1.0",
		"description": "Test items",
		"items": [
			{
				"internal_name": "new_item",
				"public_name": "newitem",
				"description": "A new item",
				"tier": 1,
				"max_stack": 5,
				"base_value": 50,
				"tags": ["material"],
				"default_display": "New Item"
			}
		]
	}`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	loader := NewLoader()
	config, err := loader.Load(tmpFile)
	require.NoError(t, err)

	mockRepo := mocks.NewMockRepositoryItem(t)

	// No previous sync metadata exists
	mockRepo.On("GetSyncMetadata", mock.Anything, ConfigFileName).Return(nil, errors.New("not found"))

	// DB state: no existing items
	mockRepo.On("GetAllItems", mock.Anything).Return([]domain.Item{}, nil)

	// DB state: one existing type
	mockRepo.On("GetAllItemTypes", mock.Anything).Return([]domain.ItemType{
		{ID: 1, Name: "material"},
	}, nil)

	// Expect insertion of new item
	mockRepo.On("InsertItem", mock.Anything, mock.MatchedBy(func(item *domain.Item) bool {
		return item.InternalName == "new_item" && item.PublicName == "newitem"
	})).Return(100, nil)

	// Expect tag assignments
	mockRepo.On("ClearItemTags", mock.Anything, 100).Return(nil)
	mockRepo.On("AssignItemTag", mock.Anything, 100, 1).Return(nil)

	// Expect updating metadata at the end
	mockRepo.On("UpsertSyncMetadata", mock.Anything, mock.AnythingOfType("*domain.SyncMetadata")).Return(nil)

	result, err := loader.SyncToDatabase(context.Background(), config, mockRepo, tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ItemsInserted)
	assert.Equal(t, 0, result.ItemsUpdated)
	assert.Equal(t, 0, result.ItemsSkipped)

	mockRepo.AssertExpectations(t)
}

func TestLoader_SyncToDatabase_UpdateExistingItem(t *testing.T) {
	content := `{
		"version": "1.0",
		"description": "Test items",
		"items": [
			{
				"internal_name": "existing_item",
				"public_name": "updated_name",
				"description": "Updated description",
				"tier": 1,
				"max_stack": 20,
				"base_value": 200,
				"tags": ["material", "compostable"],
				"default_display": "Updated Item"
			}
		]
	}`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	loader := NewLoader()
	config, err := loader.Load(tmpFile)
	require.NoError(t, err)

	mockRepo := mocks.NewMockRepositoryItem(t)

	// File changed (different hash)
	mockRepo.On("GetSyncMetadata", mock.Anything, ConfigFileName).Return(&domain.SyncMetadata{
		FileHash:    "oldhash",
		FileModTime: time.Now().Add(-1 * time.Hour),
	}, nil)

	// DB state: one existing item (with old properties)
	mockRepo.On("GetAllItems", mock.Anything).Return([]domain.Item{
		{
			ID:             50,
			InternalName:   "existing_item",
			PublicName:     "old_name",
			Description:    "Old description",
			BaseValue:      100,
			DefaultDisplay: "Old Item",
		},
	}, nil)

	// DB state: existing types
	mockRepo.On("GetAllItemTypes", mock.Anything).Return([]domain.ItemType{
		{ID: 1, Name: "material"},
	}, nil)

	// The new tag doesn't exist, expect insertion
	mockRepo.On("InsertItemType", mock.Anything, "compostable").Return(2, nil)

	// Expect update of existing item
	mockRepo.On("UpdateItem", mock.Anything, 50, mock.MatchedBy(func(item *domain.Item) bool {
		return item.InternalName == "existing_item" &&
		       item.PublicName == "updated_name" &&
		       item.BaseValue == 200
	})).Return(nil)

	// Expect tag assignments (clear old, assign both new)
	mockRepo.On("ClearItemTags", mock.Anything, 50).Return(nil)
	mockRepo.On("AssignItemTag", mock.Anything, 50, 1).Return(nil)
	mockRepo.On("AssignItemTag", mock.Anything, 50, 2).Return(nil)

	// Expect updating metadata at the end
	mockRepo.On("UpsertSyncMetadata", mock.Anything, mock.AnythingOfType("*domain.SyncMetadata")).Return(nil)

	result, err := loader.SyncToDatabase(context.Background(), config, mockRepo, tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ItemsInserted)
	assert.Equal(t, 1, result.ItemsUpdated)
	assert.Equal(t, 0, result.ItemsSkipped)

	mockRepo.AssertExpectations(t)
}

func TestLoader_SyncToDatabase_SkipUnchangedItems(t *testing.T) {
	content := `{
		"version": "1.0",
		"description": "Test items",
		"items": [
			{
				"internal_name": "unchanged_item",
				"public_name": "Same Name",
				"description": "Same description",
				"tier": 1,
				"max_stack": 10,
				"base_value": 100,
				"tags": ["material"],
				"default_display": "Same Display"
			}
		]
	}`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	loader := NewLoader()
	config, err := loader.Load(tmpFile)
	require.NoError(t, err)

	mockRepo := mocks.NewMockRepositoryItem(t)

	// Force file change trigger so we actually sync
	mockRepo.On("GetSyncMetadata", mock.Anything, ConfigFileName).Return(nil, errors.New("not found"))

	// DB state: identical existing item
	mockRepo.On("GetAllItems", mock.Anything).Return([]domain.Item{
		{
			ID:             75,
			InternalName:   "unchanged_item",
			PublicName:     "Same Name",
			Description:    "Same description",
			BaseValue:      100,
			DefaultDisplay: "Same Display",
		},
	}, nil)

	// DB state: existing type
	mockRepo.On("GetAllItemTypes", mock.Anything).Return([]domain.ItemType{
		{ID: 1, Name: "material"},
	}, nil)

	// We still sync tags even if the item properties were skipped
	mockRepo.On("ClearItemTags", mock.Anything, 75).Return(nil)
	mockRepo.On("AssignItemTag", mock.Anything, 75, 1).Return(nil)

	// Expect updating metadata at the end
	mockRepo.On("UpsertSyncMetadata", mock.Anything, mock.AnythingOfType("*domain.SyncMetadata")).Return(nil)

	result, err := loader.SyncToDatabase(context.Background(), config, mockRepo, tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ItemsInserted)
	assert.Equal(t, 0, result.ItemsUpdated)
	assert.Equal(t, 1, result.ItemsSkipped)

	mockRepo.AssertExpectations(t)
}

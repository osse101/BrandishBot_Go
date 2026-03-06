package syncutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// MetadataRepository defines the interface for repositories that store sync metadata
type MetadataRepository interface {
	GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error)
	UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error
}

// FileState represents the state of a file used for change detection
type FileState struct {
	Hash    string
	ModTime time.Time
}

// GetFileState reads the file and calculates its hash and mod time
func GetFileState(path string) (*FileState, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	return &FileState{
		Hash:    fileHash,
		ModTime: fileInfo.ModTime(),
	}, nil
}

// HasChanged checks if the provided file state differs from what's in the database
func HasChanged(ctx context.Context, repo MetadataRepository, configName string, state *FileState) (bool, error) {
	syncMeta, err := repo.GetSyncMetadata(ctx, configName)
	if err != nil {
		// First sync - no metadata exists
		return true, nil
	}

	if syncMeta.FileHash != state.Hash || !syncMeta.FileModTime.Equal(state.ModTime) {
		return true, nil
	}

	return false, nil
}

// UpdateMetadata updates the sync metadata in the database with the provided file state
func UpdateMetadata(ctx context.Context, repo MetadataRepository, configName string, state *FileState) error {
	return repo.UpsertSyncMetadata(ctx, &domain.SyncMetadata{
		ConfigName:   configName,
		LastSyncTime: time.Now(),
		FileHash:     state.Hash,
		FileModTime:  state.ModTime,
	})
}

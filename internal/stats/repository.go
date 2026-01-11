package stats

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for stats repository operations.
// It embeds repository.Stats to enable mock generation in this package.
// Generated mock will be in internal/stats/mocks/mock_repository.go
type Repository interface {
	repository.Stats
}

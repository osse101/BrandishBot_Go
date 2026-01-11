package progression

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for progression repository operations.
// It embeds repository.Progression to enable mock generation in this package.
// Generated mock will be in internal/progression/mocks/mock_repository.go
type Repository interface {
	repository.Progression
}

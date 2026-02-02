package linking

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for linking repository operations.
// It embeds repository.Linking to enable mock generation in this package.
// Generated mock will be in internal/linking/mocks/mock_repository.go
type Repository interface {
	repository.Linking
}

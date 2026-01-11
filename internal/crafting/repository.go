package crafting

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for crafting repository operations.
// It embeds repository.Crafting to enable mock generation in this package.
// Generated mock will be in internal/crafting/mocks/mock_repository.go
type Repository interface {
	repository.Crafting
}

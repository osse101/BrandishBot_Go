package gamble

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for gamble repository operations.
// It embeds repository.Gamble to enable mock generation in this package.
// Generated mock will be in internal/gamble/mocks/mock_repository.go
type Repository interface {
	repository.Gamble
}

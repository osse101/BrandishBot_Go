package user

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for user repository operations.
// It embeds repository.User to enable mock generation in this package.
// Generated mock will be in internal/user/mocks/mock_repository.go
type Repository interface {
	repository.User
}

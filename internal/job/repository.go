package job

import (
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository is a local interface for job repository operations.
// It embeds repository.Job to enable mock generation in this package.
// Generated mock will be in internal/job/mocks/mock_repository.go
type Repository interface {
	repository.Job
}

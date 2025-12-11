package progression

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// UnlockCheckerJob periodically checks if unlock criteria are met
type UnlockCheckerJob struct {
	service Service
}

// NewUnlockCheckerJob creates a new unlock checker job
func NewUnlockCheckerJob(service Service) *UnlockCheckerJob {
	return &UnlockCheckerJob{
		service: service,
	}
}

// Process runs the unlock check (implements worker.Job interface)
func (j *UnlockCheckerJob) Process(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Check and unlock if criteria met
	unlock, err := j.service.CheckAndUnlockCriteria(ctx)
	if err != nil {
		log.Error("Failed to check unlock criteria", "error", err)
		return err
	}

	if unlock != nil {
		log.Info("Node unlocked via scheduler",
			"nodeID", unlock.NodeID,
			"level", unlock.CurrentLevel,
			"contributionScore", unlock.EngagementScore)
	}

	return nil
}

package eventlog_test

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/eventlog/mocks"
)

func TestCleanupJob_Process(t *testing.T) {
	mockRepo := mocks.NewMockRepository(t)
	service := eventlog.NewService(mockRepo)
	job := eventlog.NewCleanupJob(service, 10)
	ctx := context.Background()

	// Expect CleanupOldEvents to be called
	mockRepo.On("CleanupOldEvents", ctx, 10).Return(int64(100), nil)

	err := job.Process(ctx)
	if err != nil {
		t.Fatalf("expected  no error, got %v", err)
	}

	mockRepo.AssertExpectations(t)
}

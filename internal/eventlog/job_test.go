package eventlog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCleanupJob_Process(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	job := NewCleanupJob(service, 10)
	ctx := context.Background()

	// Expect CleanupOldEvents to be called
	mockRepo.On("CleanupOldEvents", mock.Anything, 10).Return(int64(100), nil)

	err := job.Process(ctx)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

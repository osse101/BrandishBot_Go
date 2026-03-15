package eventlog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/eventlog/mocks"
)

func TestCleanupJob_Process(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		retentionDays int
		mockSetup     func(repo *mocks.MockRepository)
		expectedErr   error
	}{
		{
			name:          "Success - Events Cleaned Up",
			retentionDays: 10,
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("CleanupOldEvents", context.Background(), 10).Return(int64(100), nil)
			},
			expectedErr: nil,
		},
		{
			name:          "Failure - Service Returns Error",
			retentionDays: 5,
			mockSetup: func(repo *mocks.MockRepository) {
				repo.On("CleanupOldEvents", context.Background(), 5).Return(int64(0), errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepo := mocks.NewMockRepository(t)
			tt.mockSetup(mockRepo)

			service := eventlog.NewService(mockRepo)
			job := eventlog.NewCleanupJob(service, tt.retentionDays)

			err := job.Process(context.Background())

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBus for testing event publication
type MockBus struct {
	mock.Mock
}

func (m *MockBus) Publish(ctx context.Context, e event.Event) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *MockBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

func TestResilientEvents_Integration(t *testing.T) {
	// Setup mocks
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	statsSvc := new(MockStatsService)
	mockBus := new(MockBus)

	// Create ResilientPublisher
	deadLetterPath := "test_deadletter.jsonl"
	publisher, err := event.NewResilientPublisher(mockBus, 3, 10*time.Millisecond, deadLetterPath)
	if err != nil {
		t.Fatalf("Failed to create resilient publisher: %v", err)
	}
	defer publisher.Shutdown(context.Background())

	// Create Service
	svc := NewService(repo, prog, statsSvc, mockBus, publisher)
	ctx := context.Background()

	// Test case: XP award triggers level up, event publish initially fails but retries succeed
	t.Run("RetrySuccess", func(t *testing.T) {
		userID := "user1"
		jobKey := JobKeyBlacksmith
		jobID := 1
		baseXP := 150

		job := &domain.Job{ID: jobID, JobKey: jobKey}

		// Setup repo/progression expectations for successful XP award
		prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
		repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
		repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
			UserID: userID, JobID: jobID, CurrentXP: 0, CurrentLevel: 0,
		}, nil)
		repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
			return uj.CurrentLevel == 1 // Leveled up
		})).Return(nil)
		repo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)
		statsSvc.On("RecordUserEvent", ctx, userID, domain.EventJobLevelUp, mock.Anything).Return(nil)

		// Setup bus expectations: Fail once, then succeed
		// Need to match event type specifically
		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
			return e.Type == event.Type(domain.EventJobLevelUp)
		})).Return(errors.New("temporary failure")).Once()

		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
			return e.Type == event.Type(domain.EventJobLevelUp)
		})).Return(nil).Once()

		// Execute AwardXP
		result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

		// Verification
		assert.NoError(t, err)
		assert.True(t, result.LeveledUp)

		// Wait for retry to happen (async)
		time.Sleep(100 * time.Millisecond)

		mockBus.AssertExpectations(t)
	})

	t.Run("RetryExhaustion", func(t *testing.T) {
		userID := "user2"
		jobKey := JobKeyBlacksmith
		jobID := 1
		baseXP := 150

		job := &domain.Job{ID: jobID, JobKey: jobKey}

		// Setup repo/progression expectations
		prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
		repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
		repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
			UserID: userID, JobID: jobID, CurrentXP: 0, CurrentLevel: 0,
		}, nil)
		repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
			return uj.CurrentLevel == 1
		})).Return(nil)
		repo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)
		statsSvc.On("RecordUserEvent", ctx, userID, domain.EventJobLevelUp, mock.Anything).Return(nil)

		// Setup bus expectations: Fail always (initial + 3 retries = 4 calls)
		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
			return e.Type == event.Type(domain.EventJobLevelUp)
		})).Return(errors.New("permanent failure")).Times(4)

		// Execute AwardXP
		result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

		// Verification - should still succeed from user perspective
		assert.NoError(t, err)
		assert.True(t, result.LeveledUp)

		// Wait for retries
		time.Sleep(200 * time.Millisecond)

		mockBus.AssertExpectations(t)
	})
}

package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestSubscriptionWorker_StartAndShutdown(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	// Since we set checkInterval to 50ms, the ticker will tick during our sleep
	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{}, nil)

	worker := NewSubscriptionWorker(svc, repo, 50*time.Millisecond)
	worker.Start()

	// Allow enough time for ticker to fire
	time.Sleep(120 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSubscriptionWorker_Start_DefaultInterval(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	// Use interval 0 to trigger default 6*time.Hour
	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{}, nil).Once()

	worker := NewSubscriptionWorker(svc, repo, 0)
	assert.Equal(t, 6*time.Hour, worker.checkInterval)
	worker.Start()

	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSubscriptionWorker_CheckExpiringSubscriptions_NoExpired(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{}, nil)

	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)
	worker.checkExpiringSubscriptions()

	repo.AssertExpectations(t)
}

func TestSubscriptionWorker_CheckExpiringSubscriptions_RepoError(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil, assert.AnError)

	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)
	worker.checkExpiringSubscriptions()

	repo.AssertExpectations(t)
}

func TestSubscriptionWorker_CheckExpiringSubscriptions_Success(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	sub1 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID:   "user1",
			Platform: "discord",
		},
		TierName: "Tier 1",
	}
	sub2 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID:   "user2",
			Platform: "discord",
		},
		TierName: "Tier 2",
	}

	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{*sub1, *sub2}, nil)

	repo.On("MarkSubscriptionExpired", mock.Anything, sub1.UserID, sub1.Platform).Return(nil)
	svc.On("VerifyAndUpdateSubscription", mock.Anything, sub1.UserID, sub1.Platform).Return(nil)

	repo.On("MarkSubscriptionExpired", mock.Anything, sub2.UserID, sub2.Platform).Return(nil)
	svc.On("VerifyAndUpdateSubscription", mock.Anything, sub2.UserID, sub2.Platform).Return(nil)

	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)

	// Fast track processing
	worker.checkExpiringSubscriptions()

	repo.AssertExpectations(t)
	svc.AssertExpectations(t)
}

func TestSubscriptionWorker_CheckExpiringSubscriptions_MarkError(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	sub1 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID:   "user1",
			Platform: "discord",
		},
	}
	sub2 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID:   "user2",
			Platform: "discord",
		},
	}

	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{*sub1, *sub2}, nil)

	// Mark Error on first sub, skip verify
	repo.On("MarkSubscriptionExpired", mock.Anything, sub1.UserID, sub1.Platform).Return(assert.AnError)

	// Second sub works fine
	repo.On("MarkSubscriptionExpired", mock.Anything, sub2.UserID, sub2.Platform).Return(nil)
	svc.On("VerifyAndUpdateSubscription", mock.Anything, sub2.UserID, sub2.Platform).Return(nil)

	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)
	worker.checkExpiringSubscriptions()

	repo.AssertExpectations(t)
	svc.AssertExpectations(t)
}

func TestSubscriptionWorker_CheckExpiringSubscriptions_VerifyError(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	sub1 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID:   "user1",
			Platform: "discord",
		},
	}

	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{*sub1}, nil)
	repo.On("MarkSubscriptionExpired", mock.Anything, sub1.UserID, sub1.Platform).Return(nil)
	svc.On("VerifyAndUpdateSubscription", mock.Anything, sub1.UserID, sub1.Platform).Return(assert.AnError)

	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)
	worker.checkExpiringSubscriptions()

	repo.AssertExpectations(t)
	svc.AssertExpectations(t)
}

func TestSubscriptionWorker_CheckExpiringSubscriptions_ShutdownDuringProcessing(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	sub1 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID: "user1", Platform: "discord",
		},
	}
	sub2 := &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID: "user2", Platform: "discord",
		},
	}

	repo.On("GetExpiringSubscriptions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]domain.SubscriptionWithTier{*sub1, *sub2}, nil)

	// Configure worker
	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)

	// Since we don't mock the Mark or Verify methods, if the loop processes the subs, it will panic due to unexpected calls.
	// But we'll close shutdown before entering the loop to ensure it aborts early.
	// Actually checkExpiringSubscriptions is synchronous.
	// We'll simulate shutdown signal being ready
	close(worker.shutdown)

	worker.checkExpiringSubscriptions()

	repo.AssertExpectations(t)
	svc.AssertExpectations(t)
}

func TestSubscriptionWorker_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockSubscriptionService(t)
	repo := mocks.NewMockRepositorySubscription(t)

	worker := NewSubscriptionWorker(svc, repo, 1*time.Hour)

	// Add to wg to block shutdown
	worker.wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := worker.Shutdown(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Cleanup wg
	worker.wg.Done()
}

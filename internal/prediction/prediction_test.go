package prediction

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func setupTestService(t *testing.T) (*service, *mocks.MockProgressionService, *mocks.MockUserService, *mocks.MockEventBus, *event.ResilientPublisher) {
	mockProgSvc := mocks.NewMockProgressionService(t)
	mockUserSvc := mocks.NewMockUserService(t)
	mockEventBus := mocks.NewMockEventBus(t)

	tmpFile := t.TempDir() + "/dead_letters.jsonl"
	rp, err := event.NewResilientPublisher(mockEventBus, 3, 10*time.Millisecond, tmpFile)
	require.NoError(t, err)

	svc := NewService(mockProgSvc, mockUserSvc, mockEventBus, rp)
	return svc.(*service), mockProgSvc, mockUserSvc, mockEventBus, rp
}

func TestApplyContributionModifier(t *testing.T) {
	svc, mockProgSvc, _, _, _ := setupTestService(t)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockProgSvc.On("GetModifiedValue", ctx, "", "contribution", 10.0).Return(15.0, nil).Once()
		val, err := svc.applyContributionModifier(ctx, 10)
		assert.NoError(t, err)
		assert.Equal(t, 15, val)
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("modifier error")
		mockProgSvc.On("GetModifiedValue", ctx, "", "contribution", 10.0).Return(0.0, expectedErr).Once()
		val, err := svc.applyContributionModifier(ctx, 10)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 10, val) // returns base value on error
	})
}

func TestRecordTotalEngagement(t *testing.T) {
	svc, mockProgSvc, _, _, _ := setupTestService(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockProgSvc.On("RecordEngagement", ctx, "prediction_system", domain.MetricTypePredictionContribution, 100).Return(nil).Once()
		err := svc.recordTotalEngagement(ctx, 100)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("engagement error")
		mockProgSvc.On("RecordEngagement", ctx, "prediction_system", domain.MetricTypePredictionContribution, 100).Return(expectedErr).Once()
		err := svc.recordTotalEngagement(ctx, 100)
		assert.ErrorContains(t, err, "failed to record engagement")
	})
}

func TestPublishPredictionEvent(t *testing.T) {
	svc, _, _, mockBus, _ := setupTestService(t)
	ctx := context.Background()

	req := &domain.PredictionOutcomeRequest{
		Platform:         "twitch",
		Winner:           domain.PredictionWinner{Username: "winnerUser"},
		TotalPointsSpent: 5000,
		Participants:     []domain.PredictionParticipant{{Username: "p1"}},
	}

	mockBus.On("Publish", ctx, mock.MatchedBy(func(e event.Event) bool {
		return e.Type == event.Type(domain.EventTypePredictionProcessed) &&
			e.Payload.(map[string]interface{})["platform"] == "twitch" &&
			e.Payload.(map[string]interface{})["winner"] == "winnerUser" &&
			e.Payload.(map[string]interface{})["total_points"] == 5000 &&
			e.Payload.(map[string]interface{})["contribution"] == 10 &&
			e.Payload.(map[string]interface{})["participants_count"] == 1
	})).Return(nil).Once()

	svc.publishPredictionEvent(ctx, req, 10)

	// Wait for ResilientPublisher to process
	time.Sleep(50 * time.Millisecond)
	mockBus.AssertExpectations(t)
}

func TestEnsureUserRegistered(t *testing.T) {
	ctx := context.Background()

	t.Run("found by platform username", func(t *testing.T) {
		svc, _, mockUserSvc, _, _ := setupTestService(t)
		expectedUser := &domain.User{ID: "123", Username: "testuser"}
		mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "testuser").Return(expectedUser, nil).Once()

		user, err := svc.ensureUserRegistered(ctx, "testuser", "twitch", "pid1")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
	})

	t.Run("found by platform id", func(t *testing.T) {
		svc, _, mockUserSvc, _, _ := setupTestService(t)
		expectedUser := &domain.User{ID: "123", Username: "testuser"}
		mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "testuser").Return(nil, errors.New("not found")).Once()
		mockUserSvc.On("FindUserByPlatformID", ctx, "twitch", "pid1").Return(expectedUser, nil).Once()

		user, err := svc.ensureUserRegistered(ctx, "testuser", "twitch", "pid1")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
	})

	t.Run("auto register twitch", func(t *testing.T) {
		svc, _, mockUserSvc, _, _ := setupTestService(t)
		mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "testuser").Return(nil, errors.New("not found")).Once()
		mockUserSvc.On("FindUserByPlatformID", ctx, "twitch", "pid1").Return(nil, errors.New("not found")).Once()

		newUser := domain.User{Username: "testuser", TwitchID: "pid1"}
		registeredUser := domain.User{ID: "456", Username: "testuser", TwitchID: "pid1"}
		mockUserSvc.On("RegisterUser", ctx, newUser).Return(registeredUser, nil).Once()

		user, err := svc.ensureUserRegistered(ctx, "testuser", "twitch", "pid1")
		assert.NoError(t, err)
		assert.Equal(t, "456", user.ID)
	})

	t.Run("auto register youtube", func(t *testing.T) {
		svc, _, mockUserSvc, _, _ := setupTestService(t)
		mockUserSvc.On("GetUserByPlatformUsername", ctx, "youtube", "testuser").Return(nil, errors.New("not found")).Once()
		mockUserSvc.On("FindUserByPlatformID", ctx, "youtube", "pid1").Return(nil, errors.New("not found")).Once()

		newUser := domain.User{Username: "testuser", YoutubeID: "pid1"}
		registeredUser := domain.User{ID: "456", Username: "testuser", YoutubeID: "pid1"}
		mockUserSvc.On("RegisterUser", ctx, newUser).Return(registeredUser, nil).Once()

		user, err := svc.ensureUserRegistered(ctx, "testuser", "youtube", "pid1")
		assert.NoError(t, err)
		assert.Equal(t, "456", user.ID)
	})

	t.Run("auto register discord", func(t *testing.T) {
		svc, _, mockUserSvc, _, _ := setupTestService(t)
		mockUserSvc.On("GetUserByPlatformUsername", ctx, "discord", "testuser").Return(nil, errors.New("not found")).Once()
		mockUserSvc.On("FindUserByPlatformID", ctx, "discord", "pid1").Return(nil, errors.New("not found")).Once()

		newUser := domain.User{Username: "testuser", DiscordID: "pid1"}
		registeredUser := domain.User{ID: "456", Username: "testuser", DiscordID: "pid1"}
		mockUserSvc.On("RegisterUser", ctx, newUser).Return(registeredUser, nil).Once()

		user, err := svc.ensureUserRegistered(ctx, "testuser", "discord", "pid1")
		assert.NoError(t, err)
		assert.Equal(t, "456", user.ID)
	})

	t.Run("auto register fails", func(t *testing.T) {
		svc, _, mockUserSvc, _, _ := setupTestService(t)
		mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "testuser").Return(nil, errors.New("not found")).Once()
		mockUserSvc.On("FindUserByPlatformID", ctx, "twitch", "pid1").Return(nil, errors.New("not found")).Once()

		newUser := domain.User{Username: "testuser", TwitchID: "pid1"}
		mockUserSvc.On("RegisterUser", ctx, newUser).Return(domain.User{}, errors.New("db error")).Once()

		user, err := svc.ensureUserRegistered(ctx, "testuser", "twitch", "pid1")
		assert.ErrorContains(t, err, "failed to auto-register user")
		assert.Nil(t, user)
	})
}

func TestAwardParticipantsXP(t *testing.T) {
	svc, _, mockUserSvc, mockBus, _ := setupTestService(t)
	ctx := context.Background()

	participants := []domain.PredictionParticipant{
		{Username: "p1", PlatformID: "pid1"},
	}

	expectedUser := &domain.User{ID: "user1", Username: "p1"}
	mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "p1").Return(expectedUser, nil).Once()

	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
		payload := e.Payload.(domain.PredictionParticipantPayload)
		return e.Type == event.Type(domain.EventTypePredictionParticipated) &&
			payload.UserID == "user1" &&
			payload.Username == "p1" &&
			payload.Platform == "twitch" &&
			payload.PlatformID == "pid1" &&
			payload.XP == ParticipantXP &&
			!payload.IsWinner
	})).Return(nil).Once()

	svc.awardParticipantsXP(ctx, "twitch", participants)

	time.Sleep(50 * time.Millisecond) // Let publisher work
}

func TestAwardWinnerRewards(t *testing.T) {
	svc, mockProgSvc, mockUserSvc, mockBus, _ := setupTestService(t)
	ctx := context.Background()

	winner := domain.PredictionWinner{Username: "w1", PlatformID: "wid1"}

	// WaitGroup used in this function
	expectedUser := &domain.User{ID: "wuser", Username: "w1"}

	// Called in main thread
	mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "w1").Return(expectedUser, nil).Once()

	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
		payload := e.Payload.(domain.PredictionParticipantPayload)
		return e.Type == event.Type(domain.EventTypePredictionParticipated) &&
			payload.UserID == "wuser" &&
			payload.Username == "w1" &&
			payload.IsWinner
	})).Return(nil).Once()

	// Called in background goroutine
	mockUserSvc.On("GetUserByPlatformUsername", mock.Anything, "twitch", "w1").Return(expectedUser, nil).Once()
	mockProgSvc.On("IsItemUnlocked", mock.Anything, GrenadeItemName).Return(true, nil).Once()
	mockUserSvc.On("AddItemByUsername", mock.Anything, "twitch", "w1", GrenadeItemName, GrenadeQuantity).Return(nil).Once()

	xp := svc.awardWinnerRewards(ctx, "twitch", winner)
	assert.Equal(t, WinnerXP, xp)

	svc.wg.Wait()
	time.Sleep(50 * time.Millisecond) // Let publisher work
	mockBus.AssertExpectations(t)
}

func TestProcessOutcome(t *testing.T) {
	svc, mockProgSvc, mockUserSvc, mockBus, _ := setupTestService(t)
	ctx := context.Background()

	req := &domain.PredictionOutcomeRequest{
		Platform:         "twitch",
		Winner:           domain.PredictionWinner{Username: "w1", PlatformID: "wid1"},
		TotalPointsSpent: 1000000,
		Participants:     []domain.PredictionParticipant{{Username: "p1", PlatformID: "pid1"}},
	}

	// Mock ensureUserRegistered for winner
	wUser := &domain.User{ID: "u1", Username: "w1"}
	mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "w1").Return(wUser, nil).Once()

	// Mock publish for winner XP
	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
		return e.Type == event.Type(domain.EventTypePredictionParticipated) && e.Payload.(domain.PredictionParticipantPayload).IsWinner == true
	})).Return(nil).Once()

	// Background goroutine check
	mockUserSvc.On("GetUserByPlatformUsername", mock.Anything, "twitch", "w1").Return(wUser, nil).Once()
	mockProgSvc.On("IsItemUnlocked", mock.Anything, GrenadeItemName).Return(false, nil).Once() // skip grenade logic

	// Mock ensureUserRegistered for participants
	pUser := &domain.User{ID: "u2", Username: "p1"}
	mockUserSvc.On("GetUserByPlatformUsername", ctx, "twitch", "p1").Return(pUser, nil).Once()

	// Mock publish for participant XP
	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
		return e.Type == event.Type(domain.EventTypePredictionParticipated) && e.Payload.(domain.PredictionParticipantPayload).IsWinner == false
	})).Return(nil).Once()

	// Mock applyContributionModifier
	mockProgSvc.On("GetModifiedValue", ctx, "", "contribution", mock.AnythingOfType("float64")).Return(55.0, nil).Once()

	// Mock recordTotalEngagement
	mockProgSvc.On("RecordEngagement", ctx, "prediction_system", domain.MetricTypePredictionContribution, 55).Return(nil).Once()

	// Mock publishPredictionEvent
	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
		return e.Type == event.Type(domain.EventTypePredictionProcessed)
	})).Return(nil).Once()

	result, err := svc.ProcessOutcome(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1000000, result.TotalPoints)
	assert.Equal(t, 55, result.ContributionAwarded)
	assert.Equal(t, 1, result.ParticipantsProcessed)
	assert.Equal(t, WinnerXP, result.WinnerXPAwarded)

	svc.wg.Wait()
	time.Sleep(100 * time.Millisecond) // Let publisher finish
	mockProgSvc.AssertExpectations(t)
	mockUserSvc.AssertExpectations(t)
	mockBus.AssertExpectations(t)
}

func TestProcessOutcome_RecordEngagementError(t *testing.T) {
	svc, mockProgSvc, _, _, _ := setupTestService(t)
	ctx := context.Background()

	req := &domain.PredictionOutcomeRequest{
		Platform:         "twitch",
		TotalPointsSpent: 100000,
	}

	mockProgSvc.On("GetModifiedValue", ctx, "", "contribution", mock.AnythingOfType("float64")).Return(26.0, nil).Once()
	expectedErr := errors.New("db down")
	mockProgSvc.On("RecordEngagement", ctx, "prediction_system", domain.MetricTypePredictionContribution, 26).Return(expectedErr).Once()

	result, err := svc.ProcessOutcome(ctx, req)
	assert.ErrorContains(t, err, "failed to record engagement")
	assert.Nil(t, result)
}

func TestShutdown(t *testing.T) {
	svc, _, _, _, _ := setupTestService(t)

	// Simulate a running goroutine
	svc.wg.Add(1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		svc.wg.Done()
	}()

	ctx := context.Background()
	err := svc.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdown_Timeout(t *testing.T) {
	svc, _, _, _, _ := setupTestService(t)

	svc.wg.Add(1) // Will never finish

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := svc.Shutdown(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

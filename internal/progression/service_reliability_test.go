package progression

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// ReliabilityMockRepository is a minimal mock for testing reliability
type ReliabilityMockRepository struct {
	mock.Mock
}

// Implement only necessary methods for the test
func (m *ReliabilityMockRepository) GetActiveSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProgressionVotingSession), args.Error(1)
}

func (m *ReliabilityMockRepository) EndVotingSession(ctx context.Context, sessionID int, winningOptionID int) error {
	args := m.Called(ctx, sessionID, winningOptionID)
	return args.Error(0)
}

func (m *ReliabilityMockRepository) GetActiveUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UnlockProgress), args.Error(1)
}

func (m *ReliabilityMockRepository) SetUnlockTarget(ctx context.Context, progressID int, nodeID int, targetLevel int, sessionID int) error {
	args := m.Called(ctx, progressID, nodeID, targetLevel, sessionID)
	return args.Error(0)
}

func (m *ReliabilityMockRepository) GetEngagementScore(ctx context.Context, since *time.Time) (int, error) {
	args := m.Called(ctx, since)
	return args.Int(0), args.Error(1)
}

func (m *ReliabilityMockRepository) UnlockNode(ctx context.Context, nodeID int, level int, unlockedBy string, engagementScore int) error {
	args := m.Called(ctx, nodeID, level, unlockedBy, engagementScore)
	return args.Error(0)
}

func (m *ReliabilityMockRepository) CompleteUnlock(ctx context.Context, progressID int, rolloverPoints int) (int, error) {
	args := m.Called(ctx, progressID, rolloverPoints)
	return args.Int(0), args.Error(1)
}

func (m *ReliabilityMockRepository) GetAvailableUnlocks(ctx context.Context) ([]*domain.ProgressionNode, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProgressionNode), args.Error(1)
}

func (m *ReliabilityMockRepository) CreateVotingSession(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *ReliabilityMockRepository) GetUnlock(ctx context.Context, nodeID int, level int) (*domain.ProgressionUnlock, error) {
	args := m.Called(ctx, nodeID, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProgressionUnlock), args.Error(1)
}

// Add voting option implementation if needed
func (m *ReliabilityMockRepository) AddVotingOption(ctx context.Context, sessionID, nodeID, targetLevel int) error {
	args := m.Called(ctx, sessionID, nodeID, targetLevel)
	return args.Error(0)
}

// Other interface methods to satisfy interface... panic if called unexpectedly
func (m *ReliabilityMockRepository) GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ProgressionNode), args.Error(1)
}

func (m *ReliabilityMockRepository) GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetDependents(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetPrerequisites(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetAllUnlocks(ctx context.Context) ([]*domain.ProgressionUnlock, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) RelockNode(ctx context.Context, nodeID int, level int) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetSessionByID(ctx context.Context, sessionID int) (*domain.ProgressionVotingSession, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) IncrementOptionVote(ctx context.Context, optionID int) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetSessionVoters(ctx context.Context, sessionID int) ([]string, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) HasUserVotedInSession(ctx context.Context, userID string, sessionID int) (bool, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) RecordUserSessionVote(ctx context.Context, userID string, sessionID, optionID, nodeID int) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) CreateUnlockProgress(ctx context.Context) (int, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) AddContribution(ctx context.Context, progressID int, amount int) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) UnlockUserProgression(ctx context.Context, userID string, progressionType string, key string, metadata map[string]interface{}) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) IsUserProgressionUnlocked(ctx context.Context, userID string, progressionType string, key string) (bool, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) RecordEngagement(ctx context.Context, metric *domain.EngagementMetric) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetEngagementWeights(ctx context.Context) (map[string]float64, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) RecordReset(ctx context.Context, reset *domain.ProgressionReset) error {
	panic("not implemented")
}
func (m *ReliabilityMockRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	panic("not implemented")
}

func (m *ReliabilityMockRepository) GetDailyEngagementTotals(ctx context.Context, since time.Time) (map[time.Time]int, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[time.Time]int), args.Error(1)
}


// ReliabilityMockBus is a minimal mock for event bus
type ReliabilityMockBus struct {
	mock.Mock
}

func (m *ReliabilityMockBus) Publish(ctx context.Context, evt event.Event) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func (m *ReliabilityMockBus) Subscribe(topic event.Type, handler event.Handler) {
	m.Called(topic, handler)
}

func (m *ReliabilityMockBus) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestForceInstantUnlock_Reliability(t *testing.T) {
	mockRepo := new(ReliabilityMockRepository)
	mockBus := new(ReliabilityMockBus)

	// Allow service to subscribe to events
	mockBus.On("Subscribe", mock.Anything, mock.Anything).Return()

	service := NewService(mockRepo, mockBus)
	ctx := context.Background()

	// Setup active session
	session := &domain.ProgressionVotingSession{
		ID:     1,
		Status: "voting",
		Options: []domain.ProgressionVotingOption{
			{ID: 10, NodeID: 100, TargetLevel: 1, VoteCount: 5},
			{ID: 11, NodeID: 101, TargetLevel: 1, VoteCount: 2},
		},
	}
	mockRepo.On("GetActiveSession", mock.Anything).Return(session, nil)

	// End voting success
	mockRepo.On("EndVotingSession", mock.Anything, 1, 10).Return(nil)

	// Get unlock progress
	mockRepo.On("GetActiveUnlockProgress", mock.Anything).Return(&domain.UnlockProgress{ID: 5}, nil)
	mockRepo.On("SetUnlockTarget", mock.Anything, 5, 100, 1, 1).Return(nil)

	// Get Engagement Score (Simulate error but should proceed with 0)
	mockRepo.On("GetEngagementScore", mock.Anything, mock.Anything).Return(0, errors.New("db error"))

	// Unlock Node success
	mockRepo.On("UnlockNode", mock.Anything, 100, 1, "instant_override", 0).Return(nil)

	// Complete Unlock FAILURE - This is what we want to test being handled
	// Return int 0 and error
	mockRepo.On("CompleteUnlock", mock.Anything, 5, 0).Return(0, errors.New("critical db fail"))

	// Start Voting Session (Async)
	// Expect GetAllNodes call from StartVotingSession -> GetAvailableUnlocks
	// Return empty list so it stops there elegantly
	mockRepo.On("GetAllNodes", mock.Anything).Return([]*domain.ProgressionNode{}, nil)

	// Get Unlock (final return)
	mockRepo.On("GetUnlock", mock.Anything, 100, 1).Return(&domain.ProgressionUnlock{ID: 99}, nil)

	// Execute
	unlock, err := service.ForceInstantUnlock(ctx)

	// Verification
	// Wait a bit for async goroutine to execute
	time.Sleep(100 * time.Millisecond)

	assert.NoError(t, err)
	assert.NotNil(t, unlock)
	// We confirm that despite "critical db fail" in CompleteUnlock, execution continued.
}

func (m *ReliabilityMockRepository) GetNodeByFeatureKey(ctx context.Context, featureKey string) (*domain.ProgressionNode, int, error) {
	return nil, 0, nil
}

func (m *ReliabilityMockRepository) GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error) {
	args := m.Called(ctx, configName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SyncMetadata), args.Error(1)
}

func (m *ReliabilityMockRepository) UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error {
	args := m.Called(ctx, metadata)
	return args.Error(0)
}

package user

import (
	"context"
	"strings"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/stretchr/testify/mock"
)

type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	args := m.Called(ctx, userID, jobKey, baseAmount, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.XPAwardResult), args.Error(1)
}

func (m *MockJobService) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	return nil, nil
}
func (m *MockJobService) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	return nil, nil
}
func (m *MockJobService) GetPrimaryJob(ctx context.Context, userID string) (*domain.UserJobInfo, error) {
	return nil, nil
}
func (m *MockJobService) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	return 0, nil
}
func (m *MockJobService) GetJobBonus(ctx context.Context, userID, jobKey, bonusType string) (float64, error) {
	return 0, nil
}
func (m *MockJobService) CalculateLevel(totalXP int64) int {
	return 0
}
func (m *MockJobService) GetXPForLevel(level int) int64 {
	return 0
}
func (m *MockJobService) GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64) {
	return 0, 0
}

func TestHandleSearch_LevelUpFeedback(t *testing.T) {
	repo := newMockSearchRepo() // Defined in search_test.go
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:           1,
		InternalName: domain.ItemLootbox0,
		BaseValue:    10,
	}

	user := &domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}
	repo.users["alice"] = user

	// Mock Stats Service
	statsSvc := &mockStatsService{ // Defined in search_test.go
		mockCounts: map[domain.EventType]int{domain.EventSearch: 1},
	}

	// Mock Job Service
	jobSvc := new(MockJobService)
	jobSvc.On("AwardXP", mock.Anything, "user-alice", job.JobKeyExplorer, mock.Anything, "search", mock.Anything).Return(&domain.XPAwardResult{
		JobKey:    job.JobKeyExplorer,
		XPGained:  10,
		NewLevel:  5,
		LeveledUp: true,
	}, nil)

	// Mock Cooldown
	cooldownSvc := &mockCooldownService{repo: repo} // Defined in search_test.go

	svc := NewService(repo, statsSvc, jobSvc, nil, NewMockNamingResolver(), cooldownSvc, false) // NewMockNamingResolver in service_test.go
	ctx := context.Background()

	// Alice searches
	msg, err := svc.HandleSearch(ctx, domain.PlatformTwitch, "alice123", "alice")
	if err != nil {
		t.Fatalf("HandleSearch failed: %v", err)
	}

	// Check for Level Up message
	expected := "EXPLORER LEVEL UP!** You are now Level 5!"
	if !strings.Contains(msg, expected) {
		t.Errorf("Expected message to contain level up feedback '%s', got: %s", expected, msg)
	}
}

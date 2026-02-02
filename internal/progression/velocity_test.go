package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestGetEngagementVelocity(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	t.Run("Empty Data", func(t *testing.T) {
		repo.dailyTotals = make(map[time.Time]int)

		velocity, err := service.GetEngagementVelocity(ctx, 7)
		assert.NoError(t, err)
		assert.NotNil(t, velocity)
		assert.Equal(t, 0.0, velocity.PointsPerDay)
		assert.Equal(t, "stable", velocity.Trend)
		assert.Equal(t, 0, velocity.SampleSize)
	})

	t.Run("Stable Trend", func(t *testing.T) {
		repo.dailyTotals = make(map[time.Time]int)
		now := time.Now()
		// 7 days of constant activity
		for i := 0; i < 7; i++ {
			repo.dailyTotals[now.AddDate(0, 0, -i)] = 100
		}

		velocity, err := service.GetEngagementVelocity(ctx, 7)
		assert.NoError(t, err)
		assert.NotNil(t, velocity)
		assert.InDelta(t, 100.0, velocity.PointsPerDay, 0.1) // 700 / 7
		assert.Equal(t, "stable", velocity.Trend)
		assert.Equal(t, 7, velocity.SampleSize)
	})

	t.Run("Increasing Trend", func(t *testing.T) {
		repo.dailyTotals = make(map[time.Time]int)
		now := time.Now() // This is "today"

		// 6 days ago (oldest): 10
		// ...
		// 0 days ago (newest): 70
		// Chronologically: 10, 20, 30, 40, 50, 60, 70
		for i := 0; i < 7; i++ {
			// i=0 is today (70). i=6 is 6 days ago (10).
			val := (7 - i) * 10
			repo.dailyTotals[now.AddDate(0, 0, -i)] = val
		}

		velocity, err := service.GetEngagementVelocity(ctx, 7)
		assert.NoError(t, err)
		assert.Equal(t, "increasing", velocity.Trend)
		assert.Equal(t, 40.0, velocity.PointsPerDay) // Avg of 10..70 is 40
	})

	t.Run("Decreasing Trend", func(t *testing.T) {
		repo.dailyTotals = make(map[time.Time]int)
		now := time.Now()

		// Opposite of above
		for i := 0; i < 7; i++ {
			val := (i + 1) * 10 // Today (i=0) -> 10. 6 days ago (i=6) -> 70.
			repo.dailyTotals[now.AddDate(0, 0, -i)] = val
		}

		velocity, err := service.GetEngagementVelocity(ctx, 7)
		assert.NoError(t, err)
		assert.Equal(t, "decreasing", velocity.Trend)
	})
}

func TestEstimateUnlockTime(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Setup: Node costs 1000.
	node := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "target_node",
		UnlockCost:  1000,
		MaxLevel:    1,
		DisplayName: "Target Node",
	}
	repo.nodes = map[int]*domain.ProgressionNode{1: node}
	repo.nodesByKey = map[string]*domain.ProgressionNode{"target_node": node}

	// Setup: Current progress 500 (500 remaining)
	repo.UnlockNode(ctx, 0, 0, "", 0) // Just to enable things if needed
	progressID, _ := repo.CreateUnlockProgress(ctx)
	repo.SetUnlockTarget(ctx, progressID, 1, 1, 0)
	repo.AddContribution(ctx, progressID, 500)

	t.Run("High Confidence Estimate", func(t *testing.T) {
		// 7 days of 100 pts/day -> Velocity 100.
		repo.dailyTotals = make(map[time.Time]int)
		now := time.Now()
		for i := 0; i < 7; i++ {
			repo.dailyTotals[now.AddDate(0, 0, -i)] = 100
		}

		estimate, err := service.EstimateUnlockTime(ctx, "target_node")
		assert.NoError(t, err)
		assert.NotNil(t, estimate)
		assert.Equal(t, 500, estimate.RequiredPoints) // 1000 - 500
		assert.InDelta(t, 100.0, estimate.CurrentVelocity, 0.1)
		assert.InDelta(t, 5.0, estimate.EstimatedDays, 0.1) // 500 / 100 = 5 days
		assert.Equal(t, "high", estimate.Confidence)        // 7 days sample + stable
		assert.NotNil(t, estimate.EstimatedUnlockDate)
	})

	t.Run("Low Confidence (Not enough data)", func(t *testing.T) {
		repo.dailyTotals = make(map[time.Time]int)
		repo.dailyTotals[time.Now()] = 100 // Only 1 day

		estimate, err := service.EstimateUnlockTime(ctx, "target_node")
		assert.NoError(t, err)
		assert.Equal(t, "low", estimate.Confidence)
	})

	t.Run("Already Unlocked", func(t *testing.T) {
		repo.UnlockNode(ctx, 1, 1, "admin", 0)

		estimate, err := service.EstimateUnlockTime(ctx, "target_node")
		assert.NoError(t, err)
		assert.Equal(t, 0.0, estimate.EstimatedDays)
		assert.Equal(t, 0, estimate.RequiredPoints)
	})
}

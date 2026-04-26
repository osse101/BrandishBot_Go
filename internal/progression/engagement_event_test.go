package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

func TestService_HandleEngagement(t *testing.T) {
	repo := NewMockRepository()
	bus := event.NewMemoryBus()
	// No publisher needed for this test as we just want to verify recording
	_ = NewService(repo, nil, bus, nil, nil, false)

	ctx := context.Background()
	userID := "user-uuid"
	metricType := domain.MetricTypeMessage
	value := 1

	// Create engagement event
	metric := &domain.EngagementMetric{
		UserID:      userID,
		MetricType:  metricType,
		MetricValue: value,
		RecordedAt:  time.Now(),
	}

	evt := event.Event{
		Version: "1.0",
		Type:    domain.EventTypeEngagement,
		Payload: metric,
	}

	// Publish event - handleEngagement should be called automatically
	err := bus.Publish(ctx, evt)
	assert.NoError(t, err)

	// Verify repo has the metric
	breakdown, err := repo.GetUserEngagement(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 1, breakdown.MessagesSent)

	// Verify loop prevention works (manual publish with recorded: true)
	evtRecord := event.Event{
		Version: "1.0",
		Type:    domain.EventTypeEngagement,
		Payload: metric,
		Metadata: map[string]interface{}{
			domain.MetadataKeyRecorded: true,
		},
	}

	// Reset repo for clean check
	repo.engagementMetrics = make([]*domain.EngagementMetric, 0)

	err = bus.Publish(ctx, evtRecord)
	assert.NoError(t, err)

	breakdown, err = repo.GetUserEngagement(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 0, breakdown.MessagesSent, "Should be skipped due to recorded flag")
}

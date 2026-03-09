package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestExpeditionWorker_Start(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	exp := &domain.ExpeditionDetails{
		Expedition: domain.Expedition{
			ID:           uuid.New(),
			State:        domain.ExpeditionStateRecruiting,
			JoinDeadline: time.Now().Add(1 * time.Hour),
		},
	}

	svc.On("GetActiveExpedition", mock.Anything).Return(exp, nil)

	worker.Start()

	// Give a bit of time for internal operations if needed
	time.Sleep(10 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_Start_GetActiveExpeditionError(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	svc.On("GetActiveExpedition", mock.Anything).Return(nil, assert.AnError)

	worker.Start()

	// Give a bit of time for internal operations if needed
	time.Sleep(10 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_SubscribeAndHandle(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	mockBus := mocks.NewMockEventBus(t)
	mockBus.On("Subscribe", event.Type(domain.EventExpeditionStarted), mock.AnythingOfType("event.Handler")).Return()

	worker.Subscribe(mockBus)

	// Since we mocked bus, let's call the handler manually
	exp := &domain.Expedition{
		ID:           uuid.New(),
		State:        domain.ExpeditionStateRecruiting,
		JoinDeadline: time.Now().Add(1 * time.Hour),
	}

	e := event.Event{
		Type:    event.Type(domain.EventExpeditionStarted),
		Payload: exp,
	}

	err := worker.handleExpeditionStarted(context.Background(), e)
	assert.NoError(t, err)

	mockBus.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_HandleExpeditionStarted_InvalidPayload(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	e := event.Event{
		Type:    event.Type(domain.EventExpeditionStarted),
		Payload: "invalid-payload",
	}

	err := worker.handleExpeditionStarted(context.Background(), e)
	assert.NoError(t, err) // Should return nil, ignoring invalid payload

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_ExecuteExpedition(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	expID := uuid.New()
	svc.On("ExecuteExpedition", mock.Anything, expID).Return(nil)

	worker.executeExpedition(expID)

	// Wait for goroutine to finish
	time.Sleep(50 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_ExecuteExpedition_Error(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	expID := uuid.New()
	svc.On("ExecuteExpedition", mock.Anything, expID).Return(assert.AnError)

	worker.executeExpedition(expID)

	// Wait for goroutine to finish
	time.Sleep(50 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_ScheduleExecution_Immediate(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	expID := uuid.New()
	exp := &domain.Expedition{
		ID:           expID,
		JoinDeadline: time.Now().Add(-1 * time.Hour), // Past deadline
	}

	svc.On("ExecuteExpedition", mock.Anything, expID).Return(nil)

	worker.scheduleExecution(exp)

	// Wait for goroutine
	time.Sleep(50 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestExpeditionWorker_ScheduleExecution_Future(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockExpeditionService(t)
	worker := NewExpeditionWorker(svc)

	expID := uuid.New()
	exp := &domain.Expedition{
		ID:           expID,
		JoinDeadline: time.Now().Add(50 * time.Millisecond),
	}

	svc.On("ExecuteExpedition", mock.Anything, expID).Return(nil)

	worker.scheduleExecution(exp)

	// Give enough time for timer to fire and goroutine to execute
	time.Sleep(150 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

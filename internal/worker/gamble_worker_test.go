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

func TestGambleWorker_Start(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	g := &domain.Gamble{
		ID:           uuid.New(),
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(1 * time.Hour),
	}

	svc.On("GetActiveGamble", mock.Anything).Return(g, nil)

	worker.Start()

	// Give a bit of time for internal operations if needed
	time.Sleep(10 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_Start_GetActiveGambleError(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	svc.On("GetActiveGamble", mock.Anything).Return(nil, assert.AnError)

	worker.Start()

	// Give a bit of time for internal operations if needed
	time.Sleep(10 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_SubscribeAndHandle(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	mockBus := mocks.NewMockEventBus(t)
	mockBus.On("Subscribe", event.Type(domain.EventGambleStarted), mock.AnythingOfType("event.Handler")).Return()

	worker.Subscribe(mockBus)

	g := &domain.Gamble{
		ID:           uuid.New(),
		State:        domain.GambleStateJoining,
		JoinDeadline: time.Now().Add(1 * time.Hour),
	}

	e := event.Event{
		Type:    event.Type(domain.EventGambleStarted),
		Payload: g,
	}

	err := worker.handleGambleStarted(context.Background(), e)
	assert.NoError(t, err)

	mockBus.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_HandleGambleStarted_InvalidPayload(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	e := event.Event{
		Type:    event.Type(domain.EventGambleStarted),
		Payload: "invalid-payload",
	}

	err := worker.handleGambleStarted(context.Background(), e)
	assert.NoError(t, err) // Should return nil, ignoring invalid payload

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_ExecuteGamble(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	gambleID := uuid.New()
	svc.On("ExecuteGamble", mock.Anything, gambleID).Return(nil, nil)

	worker.executeGamble(gambleID)

	// Wait for goroutine to finish
	time.Sleep(50 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_ExecuteGamble_Error(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	gambleID := uuid.New()
	svc.On("ExecuteGamble", mock.Anything, gambleID).Return(nil, assert.AnError)

	worker.executeGamble(gambleID)

	// Wait for goroutine to finish
	time.Sleep(50 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_ScheduleExecution_Immediate(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	gambleID := uuid.New()
	g := &domain.Gamble{
		ID:           gambleID,
		JoinDeadline: time.Now().Add(-1 * time.Hour), // Past deadline
	}

	svc.On("ExecuteGamble", mock.Anything, gambleID).Return(nil, nil)

	worker.scheduleExecution(g)

	// Wait for goroutine
	time.Sleep(50 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestGambleWorker_ScheduleExecution_Future(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockGambleService(t)
	worker := NewGambleWorker(svc)

	gambleID := uuid.New()
	g := &domain.Gamble{
		ID:           gambleID,
		JoinDeadline: time.Now().Add(50 * time.Millisecond),
	}

	svc.On("ExecuteGamble", mock.Anything, gambleID).Return(nil, nil)

	worker.scheduleExecution(g)

	// Give enough time for timer to fire and goroutine to execute
	time.Sleep(150 * time.Millisecond)

	svc.AssertExpectations(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

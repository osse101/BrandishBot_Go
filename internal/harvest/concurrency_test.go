package harvest

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/mocks"
)

// mockEventBus is a minimal implementation of event.Bus for testing
type mockEventBus struct {
	mu          sync.Mutex
	lastContext context.Context
	lastEvent   event.Event
	callCount   int
	publishCh   chan struct{} // Used to signal when publish is called
	continueCh  chan struct{} // Used to block publish for graceful shutdown test
}

func (m *mockEventBus) Publish(ctx context.Context, e event.Event) error {
	m.mu.Lock()
	m.lastContext = ctx
	m.lastEvent = e
	m.callCount++
	m.mu.Unlock()

	if m.publishCh != nil {
		m.publishCh <- struct{}{}
	}
	if m.continueCh != nil {
		<-m.continueCh
	}
	return nil
}

func (m *mockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	// Not used
}

func (m *mockEventBus) GetLastContext() context.Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastContext
}

func (m *mockEventBus) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func setupHarvestService(t *testing.T, bus *mockEventBus) (Service, *mocks.MockRepositoryHarvestRepository, *mocks.MockRepositoryUser, *mocks.MockProgressionService) {
	mockHarvestRepo := mocks.NewMockRepositoryHarvestRepository(t)
	mockUserRepo := new(mocks.MockRepositoryUser)
	mockProgressionSvc := new(mocks.MockProgressionService)
	mockJobSvc := new(mocks.MockJobService)

	// Setup ResilientPublisher with our mock bus
	tmpFile := t.TempDir() + "/deadletter_test.jsonl"
	// Use minimal retry to speed up tests, but allow enough for reliable execution
	rp, err := event.NewResilientPublisher(bus, 1, time.Millisecond, tmpFile)
	require.NoError(t, err)

	// Start publisher worker (handled by NewResilientPublisher, but we should handle shutdown)
	t.Cleanup(func() {
		_ = rp.Shutdown(context.Background())
		_ = os.Remove(tmpFile)
	})

	svc := NewService(mockHarvestRepo, mockUserRepo, mockProgressionSvc, mockJobSvc, rp)
	return svc, mockHarvestRepo, mockUserRepo, mockProgressionSvc
}

func TestHarvest_GracefulShutdown(t *testing.T) {
	// Setup with a channel to wait for publish
	publishCh := make(chan struct{}, 1)
	continueCh := make(chan struct{})
	bus := &mockEventBus{
		publishCh:  publishCh,
		continueCh: continueCh,
	}
	svc, mockHarvestRepo, mockUserRepo, mockProgressionSvc := setupHarvestService(t, bus)

	// Setup User and Harvest State
	userID := "user-shutdown-test"
	mockUserRepo.On("GetUserByPlatformID", mock.Anything, domain.PlatformDiscord, "123").Return(&domain.User{ID: userID}, nil)
	mockProgressionSvc.On("IsFeatureUnlocked", mock.Anything, progression.FeatureFarming).Return(true, nil)

	// Setup initial harvest state check
	lastHarvested := time.Now().Add(-6 * time.Hour) // 6 hours -> XP award triggered
	mockHarvestRepo.On("GetHarvestState", mock.Anything, userID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	// Setup Transaction
	mockTx := mocks.NewMockRepositoryHarvestTx(t)
	mockHarvestRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	mockTx.On("Rollback", mock.Anything).Return(nil).Maybe()

	mockTx.On("GetHarvestStateWithLock", mock.Anything, userID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	// Job Bonus
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureHarvestYield, 1.0).Return(1.0, nil).Maybe()
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureGrowthSpeed, 1.0).Return(1.0, nil).Maybe()
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureSpoilExtension, 0.0).Return(0.0, nil).Maybe()
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureHarvestTier, 3.0).Return(9.0, nil).Maybe()
	// Progression/Items
	mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(true, nil).Maybe()

	// Inventory/Update
	mockTx.On("GetInventory", mock.Anything, userID).Return(&domain.Inventory{}, nil)
	mockUserRepo.On("GetItemsByNames", mock.Anything, mock.Anything).Return([]domain.Item{{InternalName: "money", ID: 1}}, nil)
	mockTx.On("UpdateInventory", mock.Anything, userID, mock.Anything).Return(nil)
	mockTx.On("UpdateHarvestState", mock.Anything, userID, mock.Anything).Return(nil)
	mockTx.On("Commit", mock.Anything).Return(nil)

	// Execute Harvest
	_, err := svc.Harvest(context.Background(), domain.PlatformDiscord, "123", "User")
	require.NoError(t, err)

	// Wait for async task to start publishing
	select {
	case <-publishCh:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for publish to start")
	}

	shutdownDone := make(chan struct{})
	go func() {
		err := svc.Shutdown(context.Background())
		assert.NoError(t, err)
		close(shutdownDone)
	}()

	// Ensure shutdown doesn't complete immediately
	select {
	case <-shutdownDone:
		t.Fatal("Shutdown completed before async task finished")
	case <-time.After(50 * time.Millisecond):
		// Expected, shutdown is blocked waiting for bus.Publish to finish
	}

	// Unblock publish
	close(continueCh)

	// Wait for shutdown to complete
	select {
	case <-shutdownDone:
		// success
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not complete after async task finished")
	}

	// Verify bus was called
	assert.Equal(t, 1, bus.CallCount(), "XP event should have been published")
}

func TestHarvest_ContextCancellation(t *testing.T) {
	// Setup with a channel to wait for publish
	publishCh := make(chan struct{}, 1)
	bus := &mockEventBus{
		publishCh: publishCh,
	}
	svc, mockHarvestRepo, mockUserRepo, mockProgressionSvc := setupHarvestService(t, bus)

	// Setup User and Harvest State
	userID := "user-ctx-test"
	mockUserRepo.On("GetUserByPlatformID", mock.Anything, domain.PlatformDiscord, "456").Return(&domain.User{ID: userID}, nil)
	mockProgressionSvc.On("IsFeatureUnlocked", mock.Anything, progression.FeatureFarming).Return(true, nil)

	// Setup initial harvest state check
	lastHarvested := time.Now().Add(-6 * time.Hour) // 6 hours -> XP award triggered
	mockHarvestRepo.On("GetHarvestState", mock.Anything, userID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	// Setup Transaction
	mockTx := mocks.NewMockRepositoryHarvestTx(t)
	mockHarvestRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	mockTx.On("Rollback", mock.Anything).Return(nil).Maybe()

	mockTx.On("GetHarvestStateWithLock", mock.Anything, userID).Return(&domain.HarvestState{LastHarvestedAt: lastHarvested}, nil)

	// Job Bonus
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureHarvestYield, 1.0).Return(1.0, nil).Maybe()
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureGrowthSpeed, 1.0).Return(1.0, nil).Maybe()
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureSpoilExtension, 0.0).Return(0.0, nil).Maybe()
	mockProgressionSvc.On("GetModifiedValue", mock.Anything, mock.Anything, featureHarvestTier, 3.0).Return(9.0, nil).Maybe()
	// Progression/Items
	mockProgressionSvc.On("IsItemUnlocked", mock.Anything, mock.Anything).Return(true, nil).Maybe()

	// Inventory/Update
	mockTx.On("GetInventory", mock.Anything, userID).Return(&domain.Inventory{}, nil)
	mockUserRepo.On("GetItemsByNames", mock.Anything, mock.Anything).Return([]domain.Item{{InternalName: "money", ID: 1}}, nil)
	mockTx.On("UpdateInventory", mock.Anything, userID, mock.Anything).Return(nil)
	mockTx.On("UpdateHarvestState", mock.Anything, userID, mock.Anything).Return(nil)
	mockTx.On("Commit", mock.Anything).Return(nil)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Execute Harvest
	_, err := svc.Harvest(ctx, domain.PlatformDiscord, "456", "User")
	require.NoError(t, err)

	// Cancel context immediately
	cancel()

	// Wait for async task to publish
	select {
	case <-publishCh:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for publish to complete")
	}

	// Wait for async task to complete
	_ = svc.Shutdown(context.Background())

	// Verify bus context was NOT cancelled
	busCtx := bus.GetLastContext()
	require.NotNil(t, busCtx, "Bus should have received a context")

	// If context.WithoutCancel worked, busCtx.Err() should be nil
	assert.NoError(t, busCtx.Err(), "Async task should receive uncancelled context even if parent is cancelled")

	// Verify original context is indeed cancelled
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}

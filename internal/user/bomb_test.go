package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestBombQueuingAndDetonation(t *testing.T) {
	ctx := context.Background()
	svc := setupTimeoutService().(*service)

	platform := domain.PlatformTwitch
	setter := "alice"
	timeout := 10 * time.Minute

	// 1. Set first bomb
	err := svc.SetPendingBomb(ctx, platform, setter, timeout)
	assert.NoError(t, err)

	// 2. Set second bomb (should queue)
	err = svc.SetPendingBomb(ctx, platform, "bob", timeout)
	assert.NoError(t, err)

	svc.recentChatterMu.Lock()
	require.Equal(t, 2, len(svc.bombQueues[platform]))
	svc.recentChatterMu.Unlock()

	// 3. Pulse with users (Peak)
	svc.recentChatterMu.Lock()
	svc.recentChatterWindow[platform] = map[string]bool{
		"user-alice": true, "user-bob": true, "user-charlie": true,
	}
	svc.recentChatterMu.Unlock()

	svc.processRecentChatters() // Pulse 1

	svc.recentChatterMu.Lock()
	assert.Equal(t, 3, len(svc.bombQueues[platform][0].AccumulatedUsers))
	assert.Equal(t, 0, len(svc.recentChatterWindow[platform]))
	svc.recentChatterMu.Unlock()

	// 4. Pulse with more users
	svc.recentChatterMu.Lock()
	svc.recentChatterWindow[platform] = map[string]bool{
		"user-dave": true, "user-eve": true,
	}
	svc.recentChatterMu.Unlock()

	svc.processRecentChatters() // Pulse 2

	svc.recentChatterMu.Lock()
	assert.Equal(t, 5, len(svc.bombQueues[platform][0].AccumulatedUsers))
	svc.recentChatterMu.Unlock()

	// 5. Pulse with empty window (Slowdown) -> Detonation
	repo := svc.repo.(*FakeRepository)
	repo.users["alice"] = &domain.User{ID: "user-alice", Username: "alice", TwitchID: "a1"}
	repo.users["bob"] = &domain.User{ID: "user-bob", Username: "bob", TwitchID: "b2"}
	repo.users["charlie"] = &domain.User{ID: "user-charlie", Username: "charlie", TwitchID: "c3"}
	repo.users["dave"] = &domain.User{ID: "user-dave", Username: "dave", TwitchID: "d4"}
	repo.users["eve"] = &domain.User{ID: "user-eve", Username: "eve", TwitchID: "e5"}

	svc.processRecentChatters() // Pulse 3 (Detonation)

	svc.recentChatterMu.Lock()
	assert.Equal(t, 1, len(svc.bombQueues[platform]), "First bomb should be shifted out of queue")
	assert.Equal(t, 0, len(svc.bombQueues[platform][0].AccumulatedUsers), "New active bomb should be clean")
	svc.recentChatterMu.Unlock()

	// Verify timeouts were applied to all 5 users
	victims := []string{"alice", "bob", "charlie", "dave", "eve"}
	for _, v := range victims {
		dur, err := svc.GetTimeoutPlatform(ctx, platform, v)
		assert.NoError(t, err)
		assert.True(t, dur > 0, "User %s should be timed out", v)
	}
}

package gamble

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

func setupBenchGambleService(numParticipants int) (*service, context.Context, uuid.UUID) {
	mockRng := func(n int) int { return 0 } // Deterministic RNG
	ctx := context.Background()
	gambleID := uuid.New()

	participants := make([]domain.Participant, 0, numParticipants)
	for i := 0; i < numParticipants; i++ {
		userID := fmt.Sprintf("user%d", i)
		participants = append(participants, domain.Participant{
			UserID: userID,
			LootboxBets: []domain.LootboxBet{
				{ItemName: domain.ItemLootbox1, Quantity: 1},
			},
		})
	}

	gamble := &domain.Gamble{
		ID:           gambleID,
		State:        domain.GambleStateJoining,
		Participants: participants,
	}

	lootboxItem := &domain.Item{ID: 1, InternalName: domain.ItemLootbox1}
	rewardItem := &domain.Item{ID: 10, PublicName: "Reward Item"}

	repo := &FakeRepository{
		Gamble: gamble,
		Item:   lootboxItem,
		Reward: rewardItem,
	}

	eventBus := new(MockEventBus)
	resilientPub := new(MockResilientPublisher)
	lootboxSvc := new(MockLootboxService)
	namingResolver := new(MockNamingResolver)

	namingResolver.On("ResolvePublicName", domain.ItemLootbox1).Return("", false)

	drops := []lootbox.DroppedItem{{ItemID: 10, ItemName: domain.ItemMoney, Quantity: 1, Value: 10}}
	lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox1, 1, mock.Anything).Return(drops, nil)

	resilientPub.On("PublishWithRetry", ctx, mock.Anything).Return()

	svc := NewService(repo, eventBus, resilientPub, lootboxSvc, time.Minute, nil, namingResolver, mockRng)

	return svc.(*service), ctx, gambleID
}

func BenchmarkService_ExecuteGamble(b *testing.B) {
	svc, ctx, gambleID := setupBenchGambleService(2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		repo := svc.repo.(*FakeRepository)
		repo.Gamble.State = domain.GambleStateJoining
		b.StartTimer()

		_, err := svc.ExecuteGamble(ctx, gambleID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_ExecuteGamble_10Participants(b *testing.B) {
	svc, ctx, gambleID := setupBenchGambleService(10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		repo := svc.repo.(*FakeRepository)
		repo.Gamble.State = domain.GambleStateJoining
		b.StartTimer()

		_, err := svc.ExecuteGamble(ctx, gambleID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_ExecuteGamble_100Participants(b *testing.B) {
	svc, ctx, gambleID := setupBenchGambleService(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		repo := svc.repo.(*FakeRepository)
		repo.Gamble.State = domain.GambleStateJoining
		b.StartTimer()

		_, err := svc.ExecuteGamble(ctx, gambleID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

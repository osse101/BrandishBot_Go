package gamble

import (
	"context"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// FakeRepository is a lightweight mock for benchmarking
type FakeRepository struct {
	Gamble *domain.Gamble
	Item   *domain.Item
	Reward *domain.Item
}

func (f *FakeRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}

func (f *FakeRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return nil, nil
}

func (f *FakeRepository) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	return f.Item, nil
}

func (f *FakeRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	if id >= 10 {
		return f.Reward, nil
	}
	return f.Item, nil
}

func (f *FakeRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	return nil
}

func (f *FakeRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	return f.Gamble, nil
}

func (f *FakeRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	return nil, nil
}

func (f *FakeRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	return nil
}

func (f *FakeRepository) GetParticipants(ctx context.Context, gambleID uuid.UUID) ([]domain.Participant, error) {
	return nil, nil
}

func (f *FakeRepository) BeginGambleTx(ctx context.Context) (repository.GambleTx, error) {
	return &FakeTx{}, nil
}

type FakeTx struct{}

func (f *FakeTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{}, nil
}

func (f *FakeTx) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error {
	return nil
}

func (f *FakeTx) UpdateGambleStateIfMatches(ctx context.Context, gambleID uuid.UUID, oldState, newState domain.GambleState) (int64, error) {
	return 1, nil
}

func (f *FakeTx) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	return nil
}

func (f *FakeTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	return nil
}

func (f *FakeRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	return nil
}

func (f *FakeRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return &domain.Inventory{}, nil
}

func (f *FakeRepository) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error {
	return nil
}

func (f *FakeRepository) UpdateGambleStateIfMatches(ctx context.Context, gambleID uuid.UUID, oldState, newState domain.GambleState) (int64, error) {
	return 1, nil
}

func (f *FakeRepository) UpdateGambleState(ctx context.Context, gambleID uuid.UUID, state domain.GambleState) error {
	return nil
}

func (f *FakeRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	return nil
}

func (f *FakeRepository) RefundGamble(ctx context.Context, gambleID uuid.UUID) error {
	return nil
}

func (f *FakeTx) RefundGamble(ctx context.Context, gambleID uuid.UUID) error {
	return nil
}

func (f *FakeTx) Commit(ctx context.Context) error {
	return nil
}

func (f *FakeTx) Rollback(ctx context.Context) error {
	return nil
}

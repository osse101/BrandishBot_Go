package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repositories holds all repository implementations used by the application.
// This provides a centralized location for repository initialization and
// makes dependency injection clearer.
type Repositories struct {
	User        repository.User
	Crafting    repository.Crafting
	Economy     repository.Economy
	Stats       repository.Stats
	Item        repository.Item
	Job         repository.Job
	EventLog    eventlog.Repository
	Gamble      repository.Gamble
	Linking     repository.Linking
	Progression repository.Progression
	Harvest     repository.HarvestRepository
}

// InitializeRepositories creates all repository implementations.
// Most repositories only need the database pool, but ProgressionRepository
// also requires the event bus for publishing progression events.
func InitializeRepositories(dbPool *pgxpool.Pool, eventBus event.Bus) *Repositories {
	return &Repositories{
		User:        postgres.NewUserRepository(dbPool),
		Crafting:    postgres.NewCraftingRepository(dbPool),
		Economy:     postgres.NewEconomyRepository(dbPool),
		Stats:       postgres.NewStatsRepository(dbPool),
		Item:        postgres.NewItemRepository(dbPool),
		Job:         postgres.NewJobRepository(dbPool),
		EventLog:    postgres.NewEventLogRepository(dbPool),
		Gamble:      postgres.NewGambleRepository(dbPool),
		Linking:     postgres.NewLinkingRepository(dbPool),
		Progression: postgres.NewProgressionRepository(dbPool, eventBus),
		Harvest:     postgres.NewHarvestRepository(dbPool),
	}
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// ExpeditionRepository implements the expedition repository for PostgreSQL
type ExpeditionRepository struct {
	*UserRepository
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewExpeditionRepository creates a new ExpeditionRepository
func NewExpeditionRepository(db *pgxpool.Pool) *ExpeditionRepository {
	return &ExpeditionRepository{
		UserRepository: NewUserRepository(db),
		db:             db,
		q:              generated.New(db),
	}
}

func (r *ExpeditionRepository) CreateExpedition(ctx context.Context, expedition *domain.Expedition) error {
	var metadataBytes []byte
	if expedition.Metadata != nil {
		var err error
		metadataBytes, err = json.Marshal(expedition.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	params := generated.CreateExpeditionParams{
		ID:                 expedition.ID,
		InitiatorID:        expedition.InitiatorID,
		ExpeditionType:     expedition.ExpeditionType,
		State:              string(expedition.State),
		CreatedAt:          pgtype.Timestamptz{Time: expedition.CreatedAt, Valid: true},
		JoinDeadline:       pgtype.Timestamptz{Time: expedition.JoinDeadline, Valid: true},
		CompletionDeadline: pgtype.Timestamptz{Time: expedition.CompletionDeadline, Valid: true},
		Metadata:           metadataBytes,
	}

	return r.q.CreateExpedition(ctx, params)
}

func (r *ExpeditionRepository) GetExpedition(ctx context.Context, id uuid.UUID) (*domain.ExpeditionDetails, error) {
	return getExpeditionHelper(ctx, r.q, r, id)
}

func getExpeditionHelper(ctx context.Context, q *generated.Queries, pProvider participantProvider, id uuid.UUID) (*domain.ExpeditionDetails, error) {
	exp, err := q.GetExpedition(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get expedition: %w", err)
	}

	expedition := mapExpedition(exp)
	participants, err := pProvider.GetParticipants(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.ExpeditionDetails{
		Expedition:   *expedition,
		Participants: participants,
	}, nil
}

type participantProvider interface {
	GetParticipants(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionParticipant, error)
}

func (r *ExpeditionRepository) AddParticipant(ctx context.Context, participant *domain.ExpeditionParticipant) error {
	params := generated.AddExpeditionParticipantParams{
		ExpeditionID: participant.ExpeditionID,
		UserID:       participant.UserID,
		JoinedAt:     pgtype.Timestamptz{Time: participant.JoinedAt, Valid: true},
		Username:     pgtype.Text{String: participant.Username, Valid: participant.Username != ""},
	}
	return r.q.AddExpeditionParticipant(ctx, params)
}

func (r *ExpeditionRepository) UpdateExpeditionState(ctx context.Context, id uuid.UUID, state domain.ExpeditionState) error {
	params := generated.UpdateExpeditionStateParams{
		State: string(state),
		ID:    id,
	}
	return r.q.UpdateExpeditionState(ctx, params)
}

func (r *ExpeditionRepository) UpdateExpeditionStateIfMatches(ctx context.Context, id uuid.UUID, expected, newState domain.ExpeditionState) (int64, error) {
	params := generated.UpdateExpeditionStateIfMatchesParams{
		State:   string(newState),
		ID:      id,
		State_2: string(expected),
	}
	tag, err := r.q.UpdateExpeditionStateIfMatches(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("failed to update expedition state: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *ExpeditionRepository) GetActiveExpedition(ctx context.Context) (*domain.ExpeditionDetails, error) {
	exp, err := r.q.GetActiveExpedition(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active expedition: %w", err)
	}

	expedition := mapExpedition(exp)
	participants, err := r.GetParticipants(ctx, expedition.ID)
	if err != nil {
		return nil, err
	}

	return &domain.ExpeditionDetails{
		Expedition:   *expedition,
		Participants: participants,
	}, nil
}

func (r *ExpeditionRepository) GetParticipants(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionParticipant, error) {
	rows, err := r.q.GetExpeditionParticipants(ctx, expeditionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	participants := make([]domain.ExpeditionParticipant, 0, len(rows))
	for _, row := range rows {
		p := domain.ExpeditionParticipant{
			ExpeditionID: row.ExpeditionID,
			UserID:       row.UserID,
			JoinedAt:     row.JoinedAt.Time,
			Username:     row.UUsername,
			IsLeader:     row.IsLeader.Bool,
		}

		if row.Rewards != nil {
			var rewards domain.ExpeditionRewards
			if err := json.Unmarshal(row.Rewards, &rewards); err == nil {
				p.Rewards = &rewards
			}
		}

		if row.JobLevels != nil {
			var jobLevels map[string]int
			if err := json.Unmarshal(row.JobLevels, &jobLevels); err == nil {
				p.JobLevels = jobLevels
			}
		}

		if row.FinalMoney.Valid {
			p.FinalMoney = int(row.FinalMoney.Int32)
		}
		if row.FinalXp.Valid {
			p.FinalXP = int(row.FinalXp.Int32)
		}
		if row.FinalItems != nil {
			var items []string
			if err := json.Unmarshal(row.FinalItems, &items); err == nil {
				p.FinalItems = items
			}
		}

		participants = append(participants, p)
	}

	return participants, nil
}

func (r *ExpeditionRepository) SaveParticipantRewards(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, rewards *domain.ExpeditionRewards) error {
	rewardsBytes, err := json.Marshal(rewards)
	if err != nil {
		return fmt.Errorf("failed to marshal rewards: %w", err)
	}

	params := generated.SaveExpeditionParticipantRewardsParams{
		ExpeditionID: expeditionID,
		UserID:       userID,
		Rewards:      rewardsBytes,
	}
	return r.q.SaveExpeditionParticipantRewards(ctx, params)
}

func (r *ExpeditionRepository) UpdateParticipantResults(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, isLeader bool, jobLevels map[string]int, money int, xp int, items []string) error {
	return updateParticipantResultsHelper(ctx, r.q, expeditionID, userID, isLeader, jobLevels, money, xp, items)
}

func updateParticipantResultsHelper(ctx context.Context, q *generated.Queries, expeditionID uuid.UUID, userID uuid.UUID, isLeader bool, jobLevels map[string]int, money int, xp int, items []string) error {
	jobLevelsBytes, err := json.Marshal(jobLevels)
	if err != nil {
		return fmt.Errorf("failed to marshal job levels: %w", err)
	}

	itemsBytes, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	params := generated.UpdateExpeditionParticipantResultsParams{
		ExpeditionID: expeditionID,
		UserID:       userID,
		IsLeader:     pgtype.Bool{Bool: isLeader, Valid: true},
		JobLevels:    jobLevelsBytes,
		FinalMoney:   pgtype.Int4{Int32: int32(money), Valid: true},
		FinalXp:      pgtype.Int4{Int32: int32(xp), Valid: true},
		FinalItems:   itemsBytes,
	}
	return q.UpdateExpeditionParticipantResults(ctx, params)
}

func (r *ExpeditionRepository) CompleteExpedition(ctx context.Context, expeditionID uuid.UUID) error {
	return r.q.CompleteExpedition(ctx, expeditionID)
}

func (r *ExpeditionRepository) GetLastCompletedExpedition(ctx context.Context) (*domain.Expedition, error) {
	exp, err := r.q.GetLastCompletedExpedition(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last completed expedition: %w", err)
	}
	return mapExpedition(exp), nil
}

func (r *ExpeditionRepository) SaveJournalEntry(ctx context.Context, entry *domain.ExpeditionJournalEntry) error {
	return saveJournalEntryHelper(ctx, r.q, entry)
}

func saveJournalEntryHelper(ctx context.Context, q *generated.Queries, entry *domain.ExpeditionJournalEntry) error {
	params := generated.SaveExpeditionJournalEntryParams{
		ExpeditionID:  entry.ExpeditionID,
		TurnNumber:    int32(entry.TurnNumber),
		EncounterType: entry.EncounterType,
		Outcome:       entry.Outcome,
		SkillChecked:  pgtype.Text{String: entry.SkillChecked, Valid: entry.SkillChecked != ""},
		SkillPassed:   pgtype.Bool{Bool: entry.SkillPassed, Valid: true},
		PrimaryMember: pgtype.Text{String: entry.PrimaryMember, Valid: entry.PrimaryMember != ""},
		Narrative:     entry.Narrative,
		Fatigue:       int32(entry.Fatigue),
		Purse:         int32(entry.Purse),
	}
	return q.SaveExpeditionJournalEntry(ctx, params)
}

func (r *ExpeditionRepository) GetJournalEntries(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionJournalEntry, error) {
	rows, err := r.q.GetExpeditionJournalEntries(ctx, expeditionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get journal entries: %w", err)
	}

	entries := make([]domain.ExpeditionJournalEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, domain.ExpeditionJournalEntry{
			ID:            int(row.ID),
			ExpeditionID:  row.ExpeditionID,
			TurnNumber:    int(row.TurnNumber),
			EncounterType: row.EncounterType,
			Outcome:       row.Outcome,
			SkillChecked:  row.SkillChecked.String,
			SkillPassed:   row.SkillPassed.Bool,
			PrimaryMember: row.PrimaryMember.String,
			Narrative:     row.Narrative,
			Fatigue:       int(row.Fatigue),
			Purse:         int(row.Purse),
			CreatedAt:     row.CreatedAt.Time,
		})
	}

	return entries, nil
}

func (r *ExpeditionRepository) BeginExpeditionTx(ctx context.Context) (repository.ExpeditionTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &expeditionTx{
		tx:       tx,
		q:        r.q.WithTx(tx),
		userRepo: r.UserRepository,
	}, nil
}

// expeditionTx implements the ExpeditionTx interface
type expeditionTx struct {
	tx       pgx.Tx
	q        *generated.Queries
	userRepo *UserRepository
}

func (t *expeditionTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *expeditionTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *expeditionTx) GetExpedition(ctx context.Context, id uuid.UUID) (*domain.ExpeditionDetails, error) {
	return getExpeditionHelper(ctx, t.q, t, id)
}

func (t *expeditionTx) UpdateExpeditionState(ctx context.Context, id uuid.UUID, state domain.ExpeditionState) error {
	params := generated.UpdateExpeditionStateParams{
		State: string(state),
		ID:    id,
	}
	return t.q.UpdateExpeditionState(ctx, params)
}

func (t *expeditionTx) UpdateExpeditionStateIfMatches(ctx context.Context, id uuid.UUID, expected, newState domain.ExpeditionState) (int64, error) {
	params := generated.UpdateExpeditionStateIfMatchesParams{
		State:   string(newState),
		ID:      id,
		State_2: string(expected),
	}
	tag, err := t.q.UpdateExpeditionStateIfMatches(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("failed to update expedition state in tx: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (t *expeditionTx) GetParticipants(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionParticipant, error) {
	rows, err := t.q.GetExpeditionParticipants(ctx, expeditionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants in tx: %w", err)
	}

	participants := make([]domain.ExpeditionParticipant, 0, len(rows))
	for _, row := range rows {
		p := domain.ExpeditionParticipant{
			ExpeditionID: row.ExpeditionID,
			UserID:       row.UserID,
			JoinedAt:     row.JoinedAt.Time,
			Username:     row.UUsername,
			IsLeader:     row.IsLeader.Bool,
		}

		if row.Rewards != nil {
			var rewards domain.ExpeditionRewards
			if err := json.Unmarshal(row.Rewards, &rewards); err == nil {
				p.Rewards = &rewards
			}
		}

		participants = append(participants, p)
	}

	return participants, nil
}

func (t *expeditionTx) SaveParticipantRewards(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, rewards *domain.ExpeditionRewards) error {
	rewardsBytes, err := json.Marshal(rewards)
	if err != nil {
		return fmt.Errorf("failed to marshal rewards: %w", err)
	}

	params := generated.SaveExpeditionParticipantRewardsParams{
		ExpeditionID: expeditionID,
		UserID:       userID,
		Rewards:      rewardsBytes,
	}
	return t.q.SaveExpeditionParticipantRewards(ctx, params)
}

func (t *expeditionTx) UpdateParticipantResults(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, isLeader bool, jobLevels map[string]int, money int, xp int, items []string) error {
	return updateParticipantResultsHelper(ctx, t.q, expeditionID, userID, isLeader, jobLevels, money, xp, items)
}

func (t *expeditionTx) SaveJournalEntry(ctx context.Context, entry *domain.ExpeditionJournalEntry) error {
	return saveJournalEntryHelper(ctx, t.q, entry)
}

func (t *expeditionTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	userTx := &userTx{tx: t.tx, q: t.q}
	return userTx.GetInventory(ctx, userID)
}

func (t *expeditionTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	userTx := &userTx{tx: t.tx, q: t.q}
	return userTx.UpdateInventory(ctx, userID, inventory)
}

// mapExpedition converts a generated Expedition to a domain Expedition
func mapExpedition(exp generated.Expedition) *domain.Expedition {
	expedition := &domain.Expedition{
		ID:                 exp.ID,
		InitiatorID:        exp.InitiatorID,
		ExpeditionType:     exp.ExpeditionType,
		State:              domain.ExpeditionState(exp.State),
		CreatedAt:          exp.CreatedAt.Time,
		JoinDeadline:       exp.JoinDeadline.Time,
		CompletionDeadline: exp.CompletionDeadline.Time,
	}

	if exp.CompletedAt.Valid {
		t := exp.CompletedAt.Time
		expedition.CompletedAt = &t
	}

	if exp.Metadata != nil {
		var metadata domain.ExpeditionMetadata
		if err := json.Unmarshal(exp.Metadata, &metadata); err == nil {
			expedition.Metadata = &metadata
		}
	}

	return expedition
}

// userTx is a helper to reuse the user repository transaction methods
type userTx struct {
	tx pgx.Tx
	q  *generated.Queries
}

func (t *userTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	row, err := t.q.GetInventoryForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	var inv domain.Inventory
	if row != nil {
		if err := json.Unmarshal(row, &inv); err != nil {
			return nil, fmt.Errorf("failed to unmarshal inventory: %w", err)
		}
	}
	return &inv, nil
}

func (t *userTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	invBytes, err := json.Marshal(inventory)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	return t.q.UpdateInventory(ctx, generated.UpdateInventoryParams{
		InventoryData: invBytes,
		UserID:        id,
	})
}

package expedition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ExecuteExpedition processes an expedition and generates rewards
func (s *service) ExecuteExpedition(ctx context.Context, expeditionID uuid.UUID) error {
	log := logger.FromContext(ctx)

	// 1. CAS: transition state from Recruiting to InProgress
	rowsAffected, err := s.repo.UpdateExpeditionStateIfMatches(ctx, expeditionID, domain.ExpeditionStateRecruiting, domain.ExpeditionStateInProgress)
	if err != nil {
		return fmt.Errorf("failed to update expedition state: %w", err)
	}
	if rowsAffected == 0 {
		log.Info("Expedition state already changed, skipping execution", "expeditionID", expeditionID)
		return nil
	}

	// 2. Load expedition and participants
	details, err := s.repo.GetExpedition(ctx, expeditionID)
	if err != nil {
		return fmt.Errorf("failed to get expedition: %w", err)
	}
	if details == nil {
		return fmt.Errorf("expedition not found: %s", expeditionID)
	}

	log.Info("Executing expedition", "expeditionID", expeditionID, "participants", len(details.Participants))

	// 3. Prepare party member state
	partyMembers := s.preparePartyMembers(ctx, details.Participants)

	// 4. Run the expedition engine
	seed := time.Now().UnixNano()
	engine := NewEngine(s.config, partyMembers, seed)
	result := engine.Run()

	log.Info("Expedition completed", "expeditionID", expeditionID, "turns", result.TotalTurns, "won", result.Won)

	// 5. Process results
	s.processJournalEntries(ctx, expeditionID, result.Journal)
	s.distributeExpeditionRewards(ctx, expeditionID, result.PartyRewards, partyMembers)

	// 6. Finalize execution
	return s.finalizeExpedition(ctx, expeditionID, details.Participants, result)
}

func (s *service) preparePartyMembers(ctx context.Context, participants []domain.ExpeditionParticipant) []*domain.PartyMemberState {
	log := logger.FromContext(ctx)
	partyMembers := make([]*domain.PartyMemberState, 0, len(participants))
	for _, p := range participants {
		jobLevels := make(map[string]int)
		if s.jobSvc != nil {
			jobs, err := s.jobSvc.GetUserJobs(ctx, p.UserID.String())
			if err != nil {
				log.Error("Failed to get user jobs", "userID", p.UserID, "error", err)
			} else {
				for _, j := range jobs {
					jobLevels[j.JobKey] = j.Level
				}
			}
		}

		partyMembers = append(partyMembers, &domain.PartyMemberState{
			UserID:      p.UserID,
			Username:    p.Username,
			JobLevels:   jobLevels,
			IsConscious: true,
			PrizeItems:  make([]string, 0),
		})
	}
	return partyMembers
}

func (s *service) processJournalEntries(ctx context.Context, expeditionID uuid.UUID, journal []domain.ExpeditionTurn) {
	log := logger.FromContext(ctx)
	for _, turn := range journal {
		entry := &domain.ExpeditionJournalEntry{
			ExpeditionID:  expeditionID,
			TurnNumber:    turn.TurnNumber,
			EncounterType: string(turn.EncounterType),
			Outcome:       string(turn.Outcome),
			SkillChecked:  string(turn.SkillChecked),
			SkillPassed:   turn.SkillPassed,
			PrimaryMember: turn.PrimaryMember,
			Narrative:     turn.Narrative,
			Fatigue:       turn.Fatigue,
			Purse:         turn.PurseAfter,
		}

		if err := s.repo.SaveJournalEntry(ctx, entry); err != nil {
			log.Error("Failed to save journal entry", "turn", turn.TurnNumber, "error", err)
		}

		_ = s.eventBus.Publish(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventExpeditionTurn),
			Payload: map[string]interface{}{
				"expedition_id": expeditionID.String(),
				"turn_number":   turn.TurnNumber,
				"narrative":     turn.Narrative,
				"fatigue":       turn.Fatigue,
				"purse":         turn.PurseAfter,
			},
		})
	}
}

func (s *service) distributeExpeditionRewards(ctx context.Context, expeditionID uuid.UUID, rewards []domain.PartyMemberReward, partyMembers []*domain.PartyMemberState) {
	log := logger.FromContext(ctx)
	for _, reward := range rewards {
		// Add items
		if s.userSvc != nil {
			for _, itemKey := range reward.Items {
				if err := s.userSvc.AddItemByUsername(ctx, "twitch", reward.Username, itemKey, 1); err != nil {
					log.Error("Failed to add item to inventory", "username", reward.Username, "item", itemKey, "error", err)
				}
			}
		}

		// Award XP via event (publisher handles job XP for all jobs)
		if s.publisher != nil && reward.XP > 0 {
			jobXP := make(map[string]int, len(SkillJobMap))
			for _, jobKey := range SkillJobMap {
				jobXP[jobKey] = reward.XP
			}
			s.publisher.PublishWithRetry(ctx, event.Event{
				Version: "1.0",
				Type:    event.Type(domain.EventTypeExpeditionRewarded),
				Payload: domain.ExpeditionRewardedPayload{
					ExpeditionID: expeditionID.String(),
					UserID:       reward.UserID.String(),
					JobXP:        jobXP,
					Timestamp:    time.Now().Unix(),
				},
			})
		}

		// Save rewards and results
		expeditionRewards := &domain.ExpeditionRewards{Items: reward.Items, XP: reward.XP, Money: reward.Money}
		if err := s.repo.SaveParticipantRewards(ctx, expeditionID, reward.UserID, expeditionRewards); err != nil {
			log.Error("Failed to save participant rewards", "userID", reward.UserID, "error", err)
		}

		var jobLevels map[string]int
		for _, m := range partyMembers {
			if m.UserID == reward.UserID {
				jobLevels = m.JobLevels
				break
			}
		}

		if err := s.repo.UpdateParticipantResults(ctx, expeditionID, reward.UserID, reward.IsLeader, jobLevels, reward.Money, reward.XP, reward.Items); err != nil {
			log.Error("Failed to update participant results", "userID", reward.UserID, "error", err)
		}
	}
}

func (s *service) finalizeExpedition(ctx context.Context, expeditionID uuid.UUID, participants []domain.ExpeditionParticipant, result *domain.ExpeditionResult) error {
	if err := s.repo.CompleteExpedition(ctx, expeditionID); err != nil {
		return fmt.Errorf("failed to complete expedition: %w", err)
	}

	if s.cooldownSvc != nil {
		_ = s.cooldownSvc.EnforceCooldown(ctx, "global", "expedition", func() error { return nil })
	}

	_ = s.eventBus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    event.Type(domain.EventExpeditionCompleted),
		Payload: map[string]interface{}{
			"expedition_id": expeditionID.String(),
			"total_turns":   result.TotalTurns,
			"won":           result.Won,
			"all_ko":        result.AllKnockedOut,
			"rewards":       result.PartyRewards,
		},
	})

	for _, p := range participants {
		if s.progressionSvc != nil {
			_ = s.progressionSvc.RecordEngagement(ctx, p.Username, "expedition_completed", 3)
		}
	}

	return nil
}

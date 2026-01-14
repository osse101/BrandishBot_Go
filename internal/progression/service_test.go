package progression

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository implements Repository for testing
//
// IMPORTANT: This mock is NOT thread-safe by design. Per docs/architecture/journal.md,
// application-level locks (mutexes) are an anti-pattern that don't work in multi-instance
// deployments. Real thread-safety is provided by database transactions with row-level locking.
//
// Use this mock for:
// - Unit tests that don't require concurrency
// - Single-goroutine test scenarios
//
// For concurrency testing:
// - Use integration tests with real PostgreSQL (see internal/database/postgres/*_integration_test.go)
// - Real database provides proper transaction isolation via SELECT ... FOR UPDATE
type MockRepository struct {
	mu                sync.RWMutex
	nodes             map[int]*domain.ProgressionNode
	nodesByKey        map[string]*domain.ProgressionNode
	unlocks           map[int]map[int]*domain.ProgressionUnlock // nodeID -> level -> unlock
	voting            *domain.ProgressionVoting
	userVotes         map[string]map[int]map[int]bool // userID -> nodeID -> level -> voted
	userProgressions  map[string]map[string]map[string]*domain.UserProgression
	engagementWeights map[string]float64
	engagementMetrics []*domain.EngagementMetric

	// Prerequisites junction table (v2.0)
	prerequisites map[int][]int // nodeID -> []prerequisiteNodeIDs

	// Voting session state
	sessions       map[int]*domain.ProgressionVotingSession
	sessionCounter int
	sessionOptions map[int][]domain.ProgressionVotingOption // sessionID -> options
	sessionVotes   map[int]map[string]bool                  // sessionID -> userID -> voted

	// Unlock progress state
	unlockProgress   map[int]*domain.UnlockProgress
	progressCounter  int
	activeProgressID int

	// Velocity testing
	dailyTotals map[time.Time]int

	// Sync metadata
	syncMetadata map[string]*domain.SyncMetadata
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		nodes:            make(map[int]*domain.ProgressionNode),
		nodesByKey:       make(map[string]*domain.ProgressionNode),
		unlocks:          make(map[int]map[int]*domain.ProgressionUnlock),
		userVotes:        make(map[string]map[int]map[int]bool),
		userProgressions: make(map[string]map[string]map[string]*domain.UserProgression),
		engagementWeights: map[string]float64{
			"message":      1.0,
			"command":      2.0,
			"item_crafted": 3.0,
			"item_used":    1.5,
			"vote_cast":    5.0,
		},
		engagementMetrics: make([]*domain.EngagementMetric, 0),
		prerequisites:     make(map[int][]int),
		sessions:          make(map[int]*domain.ProgressionVotingSession),
		sessionOptions:    make(map[int][]domain.ProgressionVotingOption),
		sessionVotes:      make(map[int]map[string]bool),
		unlockProgress:    make(map[int]*domain.UnlockProgress),
		dailyTotals:       make(map[time.Time]int),
		syncMetadata:      make(map[string]*domain.SyncMetadata),
	}
}

func (m *MockRepository) GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if node, ok := m.nodesByKey[nodeKey]; ok {
		return node, nil
	}
	return nil, nil
}
func (m *MockRepository) GetNodeByFeatureKey(ctx context.Context, featureKey string) (*domain.ProgressionNode, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find node with matching feature_key in ModifierConfig
	for _, node := range m.nodes {
		if node.ModifierConfig != nil && node.ModifierConfig.FeatureKey == featureKey {
			// lock level for this node
			levels, ok := m.unlocks[node.ID]
			if !ok {
				return nil, 0, nil
			}
			// highest level
			for level := range levels {
				return node, level, nil
			}
		}
	}
	return nil, 0, nil
}

func (m *MockRepository) GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if node, ok := m.nodes[id]; ok {
		return node, nil
	}
	return nil, nil
}

func (m *MockRepository) GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	nodes := make([]*domain.ProgressionNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (m *MockRepository) GetPrerequisites(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prereqIDs, ok := m.prerequisites[nodeID]
	if !ok {
		return []*domain.ProgressionNode{}, nil
	}

	prereqs := make([]*domain.ProgressionNode, 0, len(prereqIDs))
	for _, prereqID := range prereqIDs {
		if node, ok := m.nodes[prereqID]; ok {
			prereqs = append(prereqs, node)
		}
	}
	return prereqs, nil
}

func (m *MockRepository) GetDependents(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dependents := make([]*domain.ProgressionNode, 0)
	for depNodeID, prereqIDs := range m.prerequisites {
		for _, prereqID := range prereqIDs {
			if prereqID == nodeID {
				if node, ok := m.nodes[depNodeID]; ok {
					dependents = append(dependents, node)
				}
				break
			}
		}
	}
	return dependents, nil
}

func (m *MockRepository) GetUnlock(ctx context.Context, nodeID int, level int) (*domain.ProgressionUnlock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if levels, ok := m.unlocks[nodeID]; ok {
		if unlock, ok := levels[level]; ok {
			return unlock, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAllUnlocks(ctx context.Context) ([]*domain.ProgressionUnlock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	unlocks := make([]*domain.ProgressionUnlock, 0)
	for _, levels := range m.unlocks {
		for _, unlock := range levels {
			unlocks = append(unlocks, unlock)
		}
	}
	return unlocks, nil
}

func (m *MockRepository) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	node, err := m.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		return false, err
	}

	if levels, ok := m.unlocks[node.ID]; ok {
		if _, ok := levels[level]; ok {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockRepository) UnlockNode(ctx context.Context, nodeID int, level int, unlockedBy string, engagementScore int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unlocks[nodeID] == nil {
		m.unlocks[nodeID] = make(map[int]*domain.ProgressionUnlock)
	}

	m.unlocks[nodeID][level] = &domain.ProgressionUnlock{
		ID:              len(m.unlocks) + 1,
		NodeID:          nodeID,
		CurrentLevel:    level,
		UnlockedAt:      time.Now(),
		UnlockedBy:      unlockedBy,
		EngagementScore: engagementScore,
	}
	return nil
}

func (m *MockRepository) RelockNode(ctx context.Context, nodeID int, level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if levels, ok := m.unlocks[nodeID]; ok {
		delete(levels, level)
	}
	return nil
}

func (m *MockRepository) GetActiveVoting(ctx context.Context) (*domain.ProgressionVoting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.voting != nil && m.voting.IsActive {
		return m.voting, nil
	}
	return nil, nil
}

func (m *MockRepository) StartVoting(ctx context.Context, nodeID int, level int, endsAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.voting = &domain.ProgressionVoting{
		ID:              1,
		NodeID:          nodeID,
		TargetLevel:     level,
		VoteCount:       0,
		VotingStartedAt: time.Now(),
		VotingEndsAt:    endsAt,
		IsActive:        true,
	}
	return nil
}

func (m *MockRepository) GetVoting(ctx context.Context, nodeID int, level int) (*domain.ProgressionVoting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.voting != nil && m.voting.NodeID == nodeID && m.voting.TargetLevel == level {
		return m.voting, nil
	}
	return nil, nil
}

func (m *MockRepository) IncrementVote(ctx context.Context, nodeID int, level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.voting != nil && m.voting.NodeID == nodeID && m.voting.TargetLevel == level {
		m.voting.VoteCount++
	}
	return nil
}

func (m *MockRepository) EndVoting(ctx context.Context, nodeID int, level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.voting != nil && m.voting.NodeID == nodeID && m.voting.TargetLevel == level {
		m.voting.IsActive = false
	}
	return nil
}

func (m *MockRepository) HasUserVoted(ctx context.Context, userID string, nodeID int, level int) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if nodes, ok := m.userVotes[userID]; ok {
		if levels, ok := nodes[nodeID]; ok {
			return levels[level], nil
		}
	}
	return false, nil
}

func (m *MockRepository) RecordUserVote(ctx context.Context, userID string, nodeID int, level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.userVotes[userID] == nil {
		m.userVotes[userID] = make(map[int]map[int]bool)
	}
	if m.userVotes[userID][nodeID] == nil {
		m.userVotes[userID][nodeID] = make(map[int]bool)
	}
	m.userVotes[userID][nodeID][level] = true
	return nil
}

func (m *MockRepository) UnlockUserProgression(ctx context.Context, userID string, progressionType string, key string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.userProgressions[userID] == nil {
		m.userProgressions[userID] = make(map[string]map[string]*domain.UserProgression)
	}
	if m.userProgressions[userID][progressionType] == nil {
		m.userProgressions[userID][progressionType] = make(map[string]*domain.UserProgression)
	}

	m.userProgressions[userID][progressionType][key] = &domain.UserProgression{
		UserID:          userID,
		ProgressionType: progressionType,
		ProgressionKey:  key,
		UnlockedAt:      time.Now(),
		Metadata:        metadata,
	}
	return nil
}

func (m *MockRepository) IsUserProgressionUnlocked(ctx context.Context, userID string, progressionType string, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if types, ok := m.userProgressions[userID]; ok {
		if keys, ok := types[progressionType]; ok {
			_, exists := keys[key]
			return exists, nil
		}
	}
	return false, nil
}

func (m *MockRepository) GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	progressions := make([]*domain.UserProgression, 0)
	if types, ok := m.userProgressions[userID]; ok {
		if keys, ok := types[progressionType]; ok {
			for _, prog := range keys {
				progressions = append(progressions, prog)
			}
		}
	}
	return progressions, nil
}

func (m *MockRepository) RecordEngagement(ctx context.Context, metric *domain.EngagementMetric) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engagementMetrics = append(m.engagementMetrics, metric)
	return nil
}

func (m *MockRepository) GetEngagementScore(ctx context.Context, since *time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	totalScore := 0
	for _, metric := range m.engagementMetrics {
		if since != nil && metric.RecordedAt.Before(*since) {
			continue
		}
		weight := m.engagementWeights[metric.MetricType]
		totalScore += int(float64(metric.MetricValue) * weight)
	}
	return totalScore, nil
}

func (m *MockRepository) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	breakdown := &domain.ContributionBreakdown{}

	for _, metric := range m.engagementMetrics {
		if metric.UserID != userID {
			continue
		}

		weight := m.engagementWeights[metric.MetricType]
		breakdown.TotalScore += int(float64(metric.MetricValue) * weight)

		switch metric.MetricType {
		case "message":
			breakdown.MessagesSent += metric.MetricValue
		case "command":
			breakdown.CommandsUsed += metric.MetricValue
		case "item_crafted":
			breakdown.ItemsCrafted += metric.MetricValue
		case "item_used":
			breakdown.ItemsUsed += metric.MetricValue
		}
	}

	return breakdown, nil
}

func (m *MockRepository) GetEngagementWeights(ctx context.Context) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.engagementWeights, nil
}

func (m *MockRepository) ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Keep only root unlock
	newUnlocks := make(map[int]map[int]*domain.ProgressionUnlock)
	for nodeID, levels := range m.unlocks {
		if node, ok := m.nodes[nodeID]; ok && node.NodeKey == FeatureProgressionSystem {
			newUnlocks[nodeID] = levels
		}
	}
	m.unlocks = newUnlocks

	m.voting = nil
	m.userVotes = make(map[string]map[int]map[int]bool)

	if !preserveUserData {
		m.userProgressions = make(map[string]map[string]map[string]*domain.UserProgression)
	}

	return nil
}

func (m *MockRepository) RecordReset(ctx context.Context, reset *domain.ProgressionReset) error {
	return nil
}

// Sync metadata operations
func (m *MockRepository) GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if meta, ok := m.syncMetadata[configName]; ok {
		return meta, nil
	}
	return nil, fmt.Errorf("sync metadata not found")
}

func (m *MockRepository) UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncMetadata[metadata.ConfigName] = metadata
	return nil
}

// NodeInserter implementation
func (m *MockRepository) InsertNode(ctx context.Context, node *domain.ProgressionNode) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create ID if not present
	if node.ID == 0 {
		node.ID = len(m.nodes) + 1
	}

	m.nodes[node.ID] = node
	m.nodesByKey[node.NodeKey] = node
	return node.ID, nil
}

// NodeUpdater implementation
func (m *MockRepository) UpdateNode(ctx context.Context, nodeID int, node *domain.ProgressionNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[nodeID]; !exists {
		return fmt.Errorf("node not found")
	}

	node.ID = nodeID
	m.nodes[nodeID] = node
	m.nodesByKey[node.NodeKey] = node
	return nil
}

// Session-based voting mock methods
func (m *MockRepository) CreateVotingSession(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionCounter++
	sessionID := m.sessionCounter

	m.sessions[sessionID] = &domain.ProgressionVotingSession{
		ID:        sessionID,
		Status:    "voting",
		StartedAt: time.Now(),
		Options:   []domain.ProgressionVotingOption{},
	}
	m.sessionOptions[sessionID] = []domain.ProgressionVotingOption{}
	m.sessionVotes[sessionID] = make(map[string]bool)

	return sessionID, nil
}

func (m *MockRepository) AddVotingOption(ctx context.Context, sessionID, nodeID, targetLevel int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	node := m.nodes[nodeID]
	if node == nil {
		return fmt.Errorf("node not found")
	}

	optionID := len(m.sessionOptions[sessionID]) + 1
	option := domain.ProgressionVotingOption{
		ID:          optionID,
		SessionID:   sessionID,
		NodeID:      nodeID,
		TargetLevel: targetLevel,
		VoteCount:   0,
		NodeDetails: node,
	}

	m.sessionOptions[sessionID] = append(m.sessionOptions[sessionID], option)

	// Update session with options
	if session, ok := m.sessions[sessionID]; ok {
		session.Options = m.sessionOptions[sessionID]
	}

	return nil
}

func (m *MockRepository) GetActiveSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if session.Status == "voting" {
			return session, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetSessionByID(ctx context.Context, sessionID int) (*domain.ProgressionVotingSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if session, ok := m.sessions[sessionID]; ok {
		return session, nil
	}
	return nil, nil
}

func (m *MockRepository) IncrementOptionVote(ctx context.Context, optionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for sessionID, options := range m.sessionOptions {
		for i := range options {
			if options[i].ID == optionID {
				options[i].VoteCount++
				now := time.Now()
				options[i].LastHighestVoteAt = &now
				m.sessionOptions[sessionID] = options

				// Update session
				if session, ok := m.sessions[sessionID]; ok {
					session.Options = options
				}
				return nil
			}
		}
	}
	return fmt.Errorf("option not found")
}

func (m *MockRepository) EndVotingSession(ctx context.Context, sessionID int, winningOptionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session, ok := m.sessions[sessionID]; ok {
		session.Status = "ended"
		session.EndedAt = timePtr(time.Now())
		session.WinningOptionID = &winningOptionID
		return nil
	}
	return fmt.Errorf("session not found")
}

func (m *MockRepository) GetSessionVoters(ctx context.Context, sessionID int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	voters := make([]string, 0)
	if userMap, ok := m.sessionVotes[sessionID]; ok {
		for userID := range userMap {
			voters = append(voters, userID)
		}
	}
	return voters, nil
}

func (m *MockRepository) HasUserVotedInSession(ctx context.Context, userID string, sessionID int) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if userMap, ok := m.sessionVotes[sessionID]; ok {
		return userMap[userID], nil
	}
	return false, nil
}

func (m *MockRepository) RecordUserSessionVote(ctx context.Context, userID string, sessionID, optionID, nodeID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessionVotes[sessionID] == nil {
		m.sessionVotes[sessionID] = make(map[string]bool)
	}
	m.sessionVotes[sessionID][userID] = true
	return nil
}

// Unlock progress mock methods
func (m *MockRepository) CreateUnlockProgress(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progressCounter++
	progressID := m.progressCounter

	m.unlockProgress[progressID] = &domain.UnlockProgress{
		ID:                       progressID,
		ContributionsAccumulated: 0,
		StartedAt:                time.Now(),
	}
	m.activeProgressID = progressID
	return progressID, nil
}

func (m *MockRepository) GetActiveUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.activeProgressID == 0 {
		return nil, nil // No active progress found (not an error)
	}

	if progress, ok := m.unlockProgress[m.activeProgressID]; ok {
		if progress.UnlockedAt == nil {
			// Return a copy to avoid pointer reference issues in tests
			p := *progress
			return &p, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) AddContribution(ctx context.Context, progressID int, amount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if progress, ok := m.unlockProgress[progressID]; ok {
		progress.ContributionsAccumulated += amount
		return nil
	}
	return fmt.Errorf("unlock progress not found")
}

func (m *MockRepository) SetUnlockTarget(ctx context.Context, progressID int, nodeID int, targetLevel int, sessionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if progress, ok := m.unlockProgress[progressID]; ok {
		progress.NodeID = &nodeID
		progress.TargetLevel = &targetLevel
		progress.VotingSessionID = &sessionID
		return nil
	}
	return fmt.Errorf("unlock progress not found")
}

func (m *MockRepository) CompleteUnlock(ctx context.Context, progressID int, rolloverPoints int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if progress, ok := m.unlockProgress[progressID]; ok {
		progress.UnlockedAt = timePtr(time.Now())

		// Create new progress with rollover
		m.progressCounter++
		newProgressID := m.progressCounter

		m.unlockProgress[newProgressID] = &domain.UnlockProgress{
			ID:                       newProgressID,
			ContributionsAccumulated: rolloverPoints,
			StartedAt:                time.Now(),
		}
		m.activeProgressID = newProgressID

		return newProgressID, nil
	}
	return 0, fmt.Errorf("unlock progress not found")
}

func (m *MockRepository) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	return []domain.ContributionLeaderboardEntry{}, nil
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return nil, fmt.Errorf("transactions not supported in mock")
}

func (m *MockRepository) GetDailyEngagementTotals(ctx context.Context, since time.Time) (map[time.Time]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[time.Time]int)
	for t, v := range m.dailyTotals {
		if !t.Before(since) {
			result[t] = v
		}
	}
	return result, nil
}

// Test helper functions

func setupTestTree(repo *MockRepository) {
	// Root node
	rootID := 1
	root := &domain.ProgressionNode{
		ID:          rootID,
		NodeKey:     FeatureProgressionSystem,
		NodeType:    "feature",
		DisplayName: "Progression System",
		Description: "Root progression system",
		MaxLevel:    1,
		UnlockCost:  0,
		SortOrder:   0,
		CreatedAt:   time.Now(),
	}
	repo.nodes[rootID] = root
	repo.nodesByKey[FeatureProgressionSystem] = root

	// Unlock root
	repo.UnlockNode(context.Background(), rootID, 1, "auto", 0)

	// Money node (child of root)
	moneyID := 2
	money := &domain.ProgressionNode{
		ID:          moneyID,
		NodeKey:     "item_money",
		NodeType:    "item",
		DisplayName: "Money",
		Description: "Money item",
		MaxLevel:    1,
		UnlockCost:  500,
		SortOrder:   1,
		CreatedAt:   time.Now(),
	}
	repo.nodes[moneyID] = money
	repo.nodesByKey["item_money"] = money

	// Economy node (child of money)
	economyID := 3
	economy := &domain.ProgressionNode{
		ID:          economyID,
		NodeKey:     "feature_economy",
		NodeType:    "feature",
		DisplayName: "Economy System",
		Description: "Economy features",
		MaxLevel:    1,
		UnlockCost:  1500,
		SortOrder:   10,
		CreatedAt:   time.Now(),
	}
	repo.nodes[economyID] = economy
	repo.nodesByKey["feature_economy"] = economy

	// Buy node (child of economy)
	buyID := 6
	buy := &domain.ProgressionNode{
		ID:          buyID,
		NodeKey:     FeatureBuy,
		NodeType:    "feature",
		DisplayName: "Buy Items",
		Description: "Buy items feature",
		MaxLevel:    1,
		UnlockCost:  800,
		SortOrder:   11,
		CreatedAt:   time.Now(),
	}
	repo.nodes[buyID] = buy
	repo.nodesByKey[FeatureBuy] = buy

	// Sell node (child of economy)
	sellID := 7
	sell := &domain.ProgressionNode{
		ID:          sellID,
		NodeKey:     FeatureSell,
		NodeType:    "feature",
		DisplayName: "Sell Items",
		Description: "Sell items feature",
		MaxLevel:    1,
		UnlockCost:  800,
		SortOrder:   12,
		CreatedAt:   time.Now(),
	}
	repo.nodes[sellID] = sell
	repo.nodesByKey[FeatureSell] = sell

	// Lootbox0 node (child of root)
	lootbox0ID := 4
	lootbox0 := &domain.ProgressionNode{
		ID:          lootbox0ID,
		NodeKey:     "item_lootbox0",
		NodeType:    "item",
		DisplayName: "Basic Lootbox",
		Description: "Basic lootbox",
		MaxLevel:    1,
		UnlockCost:  500,
		SortOrder:   2,
		CreatedAt:   time.Now(),
	}
	repo.nodes[lootbox0ID] = lootbox0
	repo.nodesByKey["item_lootbox0"] = lootbox0

	// Upgrade node (child of lootbox0)
	upgradeID := 8
	upgrade := &domain.ProgressionNode{
		ID:          upgradeID,
		NodeKey:     FeatureUpgrade,
		NodeType:    "feature",
		DisplayName: "Upgrade Items",
		Description: "Upgrade system",
		MaxLevel:    1,
		UnlockCost:  1500,
		SortOrder:   20,
		CreatedAt:   time.Now(),
	}
	repo.nodes[upgradeID] = upgrade
	repo.nodesByKey[FeatureUpgrade] = upgrade

	// Disassemble node (child of lootbox0)
	disassembleID := 9
	disassemble := &domain.ProgressionNode{
		ID:          disassembleID,
		NodeKey:     FeatureDisassemble,
		NodeType:    "feature",
		DisplayName: "Disassemble Items",
		Description: "Disassemble system",
		MaxLevel:    1,
		UnlockCost:  1000,
		SortOrder:   21,
		CreatedAt:   time.Now(),
	}
	repo.nodes[disassembleID] = disassemble
	repo.nodesByKey[FeatureDisassemble] = disassemble

	// Search node (child of lootbox0)
	searchID := 10
	search := &domain.ProgressionNode{
		ID:          searchID,
		NodeKey:     FeatureSearch,
		NodeType:    "feature",
		DisplayName: "Search System",
		Description: "Search system",
		MaxLevel:    1,
		UnlockCost:  1000,
		SortOrder:   23,
		CreatedAt:   time.Now(),
	}
	repo.nodes[searchID] = search
	repo.nodesByKey[FeatureSearch] = search

	// Cooldown reduction (multi-level node, child of economy)
	cooldownID := 5
	cooldown := &domain.ProgressionNode{
		ID:          cooldownID,
		NodeKey:     "upgrade_cooldown_reduction",
		NodeType:    "upgrade",
		DisplayName: "Cooldown Reduction",
		Description: "Reduce cooldowns",
		MaxLevel:    5, // 5 levels
		UnlockCost:  1500,
		SortOrder:   40,
		CreatedAt:   time.Now(),
	}
	repo.nodes[cooldownID] = cooldown
	repo.nodesByKey["upgrade_cooldown_reduction"] = cooldown

	// Setup prerequisite relationships (v2.0 junction table simulation)
	// These mirror the old parent-child relationships
	repo.prerequisites[moneyID] = []int{rootID}           // money requires root
	repo.prerequisites[economyID] = []int{moneyID}        // economy requires money
	repo.prerequisites[buyID] = []int{economyID}          // buy requires economy
	repo.prerequisites[sellID] = []int{economyID}         // sell requires economy
	repo.prerequisites[lootbox0ID] = []int{rootID}        // lootbox0 requires root
	repo.prerequisites[upgradeID] = []int{lootbox0ID}     // upgrade requires lootbox0
	repo.prerequisites[disassembleID] = []int{lootbox0ID} // disassemble requires lootbox0
	repo.prerequisites[searchID] = []int{lootbox0ID}      // search requires lootbox0
	repo.prerequisites[cooldownID] = []int{economyID}     // cooldown requires economy
}

// Tests

func TestGetProgressionTree(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	tree, err := service.GetProgressionTree(ctx)
	if err != nil {
		t.Fatalf("GetProgressionTree failed: %v", err)
	}

	if len(tree) != 10 {
		t.Errorf("Expected 10 nodes, got %d", len(tree))
	}

	// Check root is unlocked
	var rootNode *domain.ProgressionTreeNode
	for _, node := range tree {
		if node.NodeKey == "progression_system" {
			rootNode = node
			break
		}
	}

	if rootNode == nil {
		t.Fatal("Root node not found")
	}
	if !rootNode.IsUnlocked {
		t.Error("Root node should be unlocked")
	}
	if rootNode.UnlockedLevel != 1 {
		t.Errorf("Root node should be at level 1, got %d", rootNode.UnlockedLevel)
	}
}

func TestGetAvailableUnlocks(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Initially, only money and lootbox0 should be available (root is unlocked)
	available, err := service.GetAvailableUnlocks(ctx)
	if err != nil {
		t.Fatalf("GetAvailableUnlocks failed: %v", err)
	}

	if len(available) != 2 {
		t.Errorf("Expected 2 available nodes, got %d", len(available))
	}

	// Check money is available
	moneyAvailable := false
	for _, node := range available {
		if node.NodeKey == "item_money" {
			moneyAvailable = true
		}
	}
	if !moneyAvailable {
		t.Error("Money should be available for unlock")
	}
}

// TestVoteForUnlock is obsolete - voting now uses session-based system
// See voting_sessions_test.go for new voting tests
func TestVoteForUnlock(t *testing.T) {
	t.Skip("Obsolete test - voting now uses session-based system")
}

func TestIsFeatureUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Progression system should be unlocked
	unlocked, err := service.IsFeatureUnlocked(ctx, "progression_system")
	if err != nil {
		t.Fatalf("IsFeatureUnlocked failed: %v", err)
	}
	if !unlocked {
		t.Error("Progression system should be unlocked")
	}

	// Economy should not be unlocked
	unlocked, err = service.IsFeatureUnlocked(ctx, "feature_economy")
	if err != nil {
		t.Fatalf("IsFeatureUnlocked failed: %v", err)
	}
	if unlocked {
		t.Error("Economy should not be unlocked yet")
	}
}

func TestIsItemUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Money should not be unlocked
	unlocked, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if unlocked {
		t.Error("Money should not be unlocked yet")
	}

	// Unlock money via repository (bypasses event system in test)
	repo.UnlockNode(ctx, 2, 1, "test", 0)

	// In production, the event bus would trigger cache invalidation
	// In tests without event bus, we manually clear the cache
	service.InvalidateUnlockCacheForTest()

	unlockedNow, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if !unlockedNow {
		t.Error("Money should be unlocked now")
	}
}

func TestEngagementTracking(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Record engagement metrics
	service.RecordEngagement(ctx, "test-user-1", "message", 10)
	service.RecordEngagement(ctx, "test-user-1", "command", 5)
	service.RecordEngagement(ctx, "user2", "item_crafted", 3)

	// Get user1 engagement
	breakdown, err := service.GetUserEngagement(ctx, "discord", "user1")
	if err != nil {
		t.Fatalf("GetUserEngagement failed: %v", err)
	}

	if breakdown.MessagesSent != 10 {
		t.Errorf("Expected 10 messages, got %d", breakdown.MessagesSent)
	}
	if breakdown.CommandsUsed != 5 {
		t.Errorf("Expected 5 commands, got %d", breakdown.CommandsUsed)
	}

	// Check weighted score: 10*1.0 + 5*2.0 = 20
	expectedScore := 20
	if breakdown.TotalScore != expectedScore {
		t.Errorf("Expected total score %d, got %d", expectedScore, breakdown.TotalScore)
	}

	// Get total engagement score
	totalScore, err := service.GetEngagementScore(ctx)
	if err != nil {
		t.Fatalf("GetEngagementScore failed: %v", err)
	}

	// 10*1.0 + 5*2.0 + 3*3.0 = 29
	expectedTotal := 29
	if totalScore != expectedTotal {
		t.Errorf("Expected total engagement %d, got %d", expectedTotal, totalScore)
	}
}

func TestAdminUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Admin unlock money
	err := service.AdminUnlock(ctx, "item_money", 1)
	if err != nil {
		t.Fatalf("AdminUnlock failed: %v", err)
	}

	// Verify it's unlocked
	unlocked, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if !unlocked {
		t.Error("Money should be unlocked after admin unlock")
	}
}

func TestAdminRelock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock then relock
	service.AdminUnlock(ctx, "item_money", 1)
	err := service.AdminRelock(ctx, "item_money", 1)
	if err != nil {
		t.Fatalf("AdminRelock failed: %v", err)
	}

	// Verify it's locked again
	unlocked, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if unlocked {
		t.Error("Money should be locked after admin relock")
	}
}

func TestResetProgressionTree(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock some nodes
	service.AdminUnlock(ctx, "item_money", 1)
	service.AdminUnlock(ctx, "feature_economy", 1)

	// Add user progression
	repo.UnlockUserProgression(ctx, "user1", "recipe", "recipe_lootbox1", nil)

	// Reset without preserving user data
	err := service.ResetProgressionTree(ctx, "admin", "test reset", false)
	if err != nil {
		t.Fatalf("ResetProgressionTree failed: %v", err)
	}

	// Check root is still unlocked
	rootUnlocked, _ := service.IsFeatureUnlocked(ctx, "progression_system")
	if !rootUnlocked {
		t.Error("Root should remain unlocked after reset")
	}

	// Check money is locked
	moneyUnlocked, _ := service.IsItemUnlocked(ctx, "money")
	if moneyUnlocked {
		t.Error("Money should be locked after reset")
	}

	// Check user progression was cleared
	progressions, _ := repo.GetUserProgressions(ctx, "user1", "recipe")
	if len(progressions) != 0 {
		t.Error("User progressions should be cleared")
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}

func TestMultiLevelUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock economy first (prerequisite for cooldown reduction)
	repo.UnlockNode(ctx, 2, 1, "test", 0) // money
	repo.UnlockNode(ctx, 3, 1, "test", 0) // economy

	// Unlock cooldown reduction level 1
	service.AdminUnlock(ctx, "upgrade_cooldown_reduction", 1)

	// Unlock level 2
	service.AdminUnlock(ctx, "upgrade_cooldown_reduction", 2)

	// Check both levels are unlocked
	level1, _ := repo.IsNodeUnlocked(ctx, "upgrade_cooldown_reduction", 1)
	level2, _ := repo.IsNodeUnlocked(ctx, "upgrade_cooldown_reduction", 2)

	if !level1 {
		t.Error("Cooldown reduction level 1 should be unlocked")
	}
	if !level2 {
		t.Error("Cooldown reduction level 2 should be unlocked")
	}
}

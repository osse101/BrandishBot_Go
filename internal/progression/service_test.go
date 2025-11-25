package progression

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	nodes            map[int]*domain.ProgressionNode
	nodesByKey       map[string]*domain.ProgressionNode
	unlocks          map[int]map[int]*domain.ProgressionUnlock // nodeID -> level -> unlock
	voting           *domain.ProgressionVoting
	userVotes        map[string]map[int]map[int]bool // userID -> nodeID -> level -> voted
	userProgressions map[string]map[string]map[string]*domain.UserProgression
	engagementWeights map[string]float64
	engagementMetrics []*domain.EngagementMetric
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
		},
		engagementMetrics: make([]*domain.EngagementMetric, 0),
	}
}

func (m *MockRepository) GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error) {
	if node, ok := m.nodesByKey[nodeKey]; ok {
		return node, nil
	}
	return nil, nil
}

func (m *MockRepository) GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	if node, ok := m.nodes[id]; ok {
		return node, nil
	}
	return nil, nil
}

func (m *MockRepository) GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error) {
	nodes := make([]*domain.ProgressionNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (m *MockRepository) GetChildNodes(ctx context.Context, parentID int) ([]*domain.ProgressionNode, error) {
	children := make([]*domain.ProgressionNode, 0)
	for _, node := range m.nodes {
		if node.ParentNodeID != nil && *node.ParentNodeID == parentID {
			children = append(children, node)
		}
	}
	return children, nil
}

func (m *MockRepository) GetUnlock(ctx context.Context, nodeID int, level int) (*domain.ProgressionUnlock, error) {
	if levels, ok := m.unlocks[nodeID]; ok {
		if unlock, ok := levels[level]; ok {
			return unlock, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAllUnlocks(ctx context.Context) ([]*domain.ProgressionUnlock, error) {
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
	if levels, ok := m.unlocks[nodeID]; ok {
		delete(levels, level)
	}
	return nil
}

func (m *MockRepository) GetActiveVoting(ctx context.Context) (*domain.ProgressionVoting, error) {
	if m.voting != nil && m.voting.IsActive {
		return m.voting, nil
	}
	return nil, nil
}

func (m *MockRepository) StartVoting(ctx context.Context, nodeID int, level int, endsAt *time.Time) error {
	m.voting = &domain.ProgressionVoting{
		ID:               1,
		NodeID:           nodeID,
		TargetLevel:      level,
		VoteCount:        0,
		VotingStartedAt:  time.Now(),
		VotingEndsAt:     endsAt,
		IsActive:         true,
	}
	return nil
}

func (m *MockRepository) GetVoting(ctx context.Context, nodeID int, level int) (*domain.ProgressionVoting, error) {
	if m.voting != nil && m.voting.NodeID == nodeID && m.voting.TargetLevel == level {
		return m.voting, nil
	}
	return nil, nil
}

func (m *MockRepository) IncrementVote(ctx context.Context, nodeID int, level int) error {
	if m.voting != nil && m.voting.NodeID == nodeID && m.voting.TargetLevel == level {
		m.voting.VoteCount++
	}
	return nil
}

func (m *MockRepository) EndVoting(ctx context.Context, nodeID int, level int) error {
	if m.voting != nil && m.voting.NodeID == nodeID && m.voting.TargetLevel == level {
		m.voting.IsActive = false
	}
	return nil
}

func (m *MockRepository) HasUserVoted(ctx context.Context, userID string, nodeID int, level int) (bool, error) {
	if nodes, ok := m.userVotes[userID]; ok {
		if levels, ok := nodes[nodeID]; ok {
			return levels[level], nil
		}
	}
	return false, nil
}

func (m *MockRepository) RecordUserVote(ctx context.Context, userID string, nodeID int, level int) error {
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
	if types, ok := m.userProgressions[userID]; ok {
		if keys, ok := types[progressionType]; ok {
			_, exists := keys[key]
			return exists, nil
		}
	}
	return false, nil
}

func (m *MockRepository) GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error) {
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
	m.engagementMetrics = append(m.engagementMetrics, metric)
	return nil
}

func (m *MockRepository) GetEngagementScore(ctx context.Context, since *time.Time) (int, error) {
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

func (m *MockRepository) GetUserEngagement(ctx context.Context, userID string) (*domain.EngagementBreakdown, error) {
	breakdown := &domain.EngagementBreakdown{}
	
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
	return m.engagementWeights, nil
}

func (m *MockRepository) ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	// Keep only root unlock
	newUnlocks := make(map[int]map[int]*domain.ProgressionUnlock)
	for nodeID, levels := range m.unlocks {
		if node, ok := m.nodes[nodeID]; ok && node.NodeKey == "progression_system" {
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

func (m *MockRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return nil, fmt.Errorf("transactions not supported in mock")
}

// Test helper functions

func setupTestTree(repo *MockRepository) {
	// Root node
	rootID := 1
	root := &domain.ProgressionNode{
		ID:           rootID,
		NodeKey:      "progression_system",
		NodeType:     "feature",
		DisplayName:  "Progression System",
		Description:  "Root progression system",
		ParentNodeID: nil,
		MaxLevel:     1,
		UnlockCost:   0,
		SortOrder:    0,
		CreatedAt:    time.Now(),
	}
	repo.nodes[rootID] = root
	repo.nodesByKey["progression_system"] = root
	
	// Unlock root
	repo.UnlockNode(context.Background(), rootID, 1, "auto", 0)
	
	// Money node (child of root)
	moneyID := 2
	parentID := rootID
	money := &domain.ProgressionNode{
		ID:           moneyID,
		NodeKey:      "item_money",
		NodeType:     "item",
		DisplayName:  "Money",
		Description:  "Money item",
		ParentNodeID: &parentID,
		MaxLevel:     1,
		UnlockCost:   500,
		SortOrder:    1,
		CreatedAt:    time.Now(),
	}
	repo.nodes[moneyID] = money
	repo.nodesByKey["item_money"] = money
	
	// Economy node (child of money)
	economyID := 3
	ecoParent := moneyID
	economy := &domain.ProgressionNode{
		ID:           economyID,
		NodeKey:      "feature_economy",
		NodeType:     "feature",
		DisplayName:  "Economy System",
		Description:  "Economy features",
		ParentNodeID: &ecoParent,
		MaxLevel:     1,
		UnlockCost:   1500,
		SortOrder:    10,
		CreatedAt:    time.Now(),
	}
	repo.nodes[economyID] = economy
	repo.nodesByKey["feature_economy"] = economy
	
	// Lootbox0 node (child of root)
	lootbox0ID := 4
	lb0Parent := rootID
	lootbox0 := &domain.ProgressionNode{
		ID:           lootbox0ID,
		NodeKey:      "item_lootbox0",
		NodeType:     "item",
		DisplayName:  "Basic Lootbox",
		Description:  "Basic lootbox",
		ParentNodeID: &lb0Parent,
		MaxLevel:     1,
		UnlockCost:   500,
		SortOrder:    2,
		CreatedAt:    time.Now(),
	}
	repo.nodes[lootbox0ID] = lootbox0
	repo.nodesByKey["item_lootbox0"] = lootbox0
	
	// Cooldown reduction (multi-level node)
	cooldownID := 5
	cdParent := economyID
	cooldown := &domain.ProgressionNode{
		ID:           cooldownID,
		NodeKey:      "upgrade_cooldown_reduction",
		NodeType:     "upgrade",
		DisplayName:  "Cooldown Reduction",
		Description:  "Reduce cooldowns",
		ParentNodeID: &cdParent,
		MaxLevel:     5, // 5 levels
		UnlockCost:   1500,
		SortOrder:    40,
		CreatedAt:    time.Now(),
	}
	repo.nodes[cooldownID] = cooldown
	repo.nodesByKey["upgrade_cooldown_reduction"] = cooldown
}

// Tests

func TestGetProgressionTree(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo)
	ctx := context.Background()
	
	tree, err := service.GetProgressionTree(ctx)
	if err != nil {
		t.Fatalf("GetProgressionTree failed: %v", err)
	}
	
	if len(tree) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(tree))
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
	service := NewService(repo)
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

func TestVoteForUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo)
	ctx := context.Background()
	
	// Vote for money
	err := service.VoteForUnlock(ctx, "user1", "item_money")
	if err != nil {
		t.Fatalf("VoteForUnlock failed: %v", err)
	}
	
	// Verify vote was recorded
	hasVoted, _ := repo.HasUserVoted(ctx, "user1", 2, 1) // money node ID is 2
	if !hasVoted {
		t.Error("User vote should be recorded")
	}
	
	// Try to vote again - should fail
	err = service.VoteForUnlock(ctx, "user1", "item_money")
	if err == nil {
		t.Error("Expected error for double voting")
	}
}

func TestIsFeatureUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo)
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
	service := NewService(repo)
	ctx := context.Background()
	
	// Money should not be unlocked
	unlocked, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if unlocked {
		t.Error("Money should not be unlocked yet")
	}
	
	// Unlock money
	repo.UnlockNode(ctx, 2, 1, "test", 0)
	
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
	service := NewService(repo)
	ctx := context.Background()
	
	// Record engagement metrics
	service.RecordEngagement(ctx, "user1", "message", 10)
	service.RecordEngagement(ctx, "user1", "command", 5)
	service.RecordEngagement(ctx, "user2", "item_crafted", 3)
	
	// Get user1 engagement
	breakdown, err := service.GetUserEngagement(ctx, "user1")
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
	service := NewService(repo)
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
	service := NewService(repo)
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
	service := NewService(repo)
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

func TestMultiLevelUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo)
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

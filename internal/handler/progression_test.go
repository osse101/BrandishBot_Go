package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestProgressionHandlers_HandleGetTree(t *testing.T) {
	mockSvc := mocks.NewMockProgressionService(t)
	handler := NewProgressionHandlers(mockSvc)

	// Mock response
	mockNodes := []*domain.ProgressionTreeNode{
		{
			ProgressionNode: domain.ProgressionNode{
				NodeKey:  "blacksmith_tier1",
				MaxLevel: 1,
			},
			UnlockedLevel: 0,
		},
	}
	mockSvc.On("GetProgressionTree", mock.Anything).Return(mockNodes, nil)

	req := httptest.NewRequest("GET", "/progression/tree", nil)
	rec := httptest.NewRecorder()

	handler.HandleGetTree()(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ProgressionTreeResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Nodes, 1)
	assert.Equal(t, "blacksmith_tier1", resp.Nodes[0].NodeKey)
}

func TestProgressionHandlers_HandleVote(t *testing.T) {
	InitValidator() // Ensure validator is initialized

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*mocks.MockProgressionService)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "Success",
			body: VoteRequest{Platform: "discord", PlatformID: "u1", Username: "user1", OptionIndex: 1},
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("VoteForUnlock", mock.Anything, "discord", "u1", "user1", 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Vote recorded successfully",
		},
		{
			name:           "Invalid Body (Validation)",
			body:           VoteRequest{Platform: "discord", PlatformID: "", Username: "user1", OptionIndex: 1},
			setupMock:      func(m *mocks.MockProgressionService) {},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid request",
		},
		{
			name: "Service Error",
			body: VoteRequest{Platform: "discord", PlatformID: "u1", Username: "user1", OptionIndex: 1},
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("VoteForUnlock", mock.Anything, "discord", "u1", "user1", 1).Return(errors.New("already voted"))
			},
			expectedStatus: http.StatusBadRequest, // Handler returns 400 on service error for vote
			expectedMsg:    "already voted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockProgressionService(t)
			tt.setupMock(mockSvc)

			handler := NewProgressionHandlers(mockSvc)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/progression/vote", bytes.NewReader(bodyBytes))
			rec := httptest.NewRecorder()

			handler.HandleVote()(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.expectedMsg)
		})
	}
}

func TestProgressionHandlers_HandleGetStatus(t *testing.T) {
	mockSvc := mocks.NewMockProgressionService(t)
	handler := NewProgressionHandlers(mockSvc)

	mockStatus := &domain.ProgressionStatus{
		TotalUnlocked: 5,
	}
	mockSvc.On("GetProgressionStatus", mock.Anything).Return(mockStatus, nil)

	req := httptest.NewRequest("GET", "/progression/status", nil)
	rec := httptest.NewRecorder()

	handler.HandleGetStatus()(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp domain.ProgressionStatus
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 5, resp.TotalUnlocked)
}

func TestProgressionHandlers_HandleAdminUnlock(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		body           map[string]interface{}
		setupMock      func(*mocks.MockProgressionService)
		expectedStatus int
	}{
		{
			name: "Success",
			body: map[string]interface{}{"node_key": "n1", "level": 1},
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("AdminUnlock", mock.Anything, "n1", 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Validation Error",
			body:           map[string]interface{}{"node_key": "", "level": 1},
			setupMock:      func(m *mocks.MockProgressionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service Error",
			body: map[string]interface{}{"node_key": "n1", "level": 1},
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("AdminUnlock", mock.Anything, "n1", 1).Return(errors.New("failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockProgressionService(t)
			tt.setupMock(mockSvc)

			handler := NewProgressionHandlers(mockSvc)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/progression/admin/unlock", bytes.NewReader(bodyBytes))
			rec := httptest.NewRecorder()

			handler.HandleAdminUnlock()(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestProgressionHandlers_HandleGetEstimate(t *testing.T) {
	tests := []struct {
		name           string
		nodeKey        string
		setupMock      func(*mocks.MockProgressionService)
		expectedStatus int
	}{
		{
			name:    "Success",
			nodeKey: "node_1",
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("EstimateUnlockTime", mock.Anything, "node_1").Return(&domain.UnlockEstimate{
					NodeKey: "node_1",
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Node Not Found",
			nodeKey: "unknown",
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("EstimateUnlockTime", mock.Anything, "unknown").Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:    "Service Error",
			nodeKey: "node_error",
			setupMock: func(m *mocks.MockProgressionService) {
				m.On("EstimateUnlockTime", mock.Anything, "node_error").Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Missing Node Key",
			nodeKey:        "",
			setupMock:      func(m *mocks.MockProgressionService) {},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockProgressionService(t)
			tt.setupMock(mockSvc)

			handler := NewProgressionHandlers(mockSvc)

			// Setup request using Chi context to simulate URL params
			req := httptest.NewRequest("GET", "/progression/estimate/"+tt.nodeKey, nil)

			r := chi.NewRouter()
			r.Get("/progression/estimate/{nodeKey}", handler.HandleGetEstimate())

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestProgressionHandlers_HandleGetUnlockProgress(t *testing.T) {
	mockSvc := mocks.NewMockProgressionService(t)
	handler := NewProgressionHandlers(mockSvc)

	// Mock node
	node := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "node_1",
		UnlockCost:  1000,
		DisplayName: "Node 1",
	}

	// Mock progress
	progress := &domain.UnlockProgress{
		ID:                       1,
		NodeID:                   &node.ID,
		ContributionsAccumulated: 100,
	}

	now := time.Now()
	// Mock estimate
	estimate := &domain.UnlockEstimate{
		NodeKey:             "node_1",
		EstimatedDays:       5.0,
		Confidence:          "high",
		RequiredPoints:      900,
		CurrentProgress:     100,
		CurrentVelocity:     180,
		EstimatedUnlockDate: &now,
	}

	mockSvc.On("GetUnlockProgress", mock.Anything).Return(progress, nil)
	mockSvc.On("GetNode", mock.Anything, 1).Return(node, nil)
	mockSvc.On("EstimateUnlockTime", mock.Anything, "node_1").Return(estimate, nil)

	req := httptest.NewRequest("GET", "/progression/unlock-progress", nil)
	rec := httptest.NewRecorder()

	handler.HandleGetUnlockProgress()(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify enriched fields
	assert.Equal(t, 100.0, resp["contributions_accumulated"])
	assert.Equal(t, "Node 1", resp["target_node_name"])
	assert.Equal(t, 10.0, resp["completion_percentage"])

	// Verify estimate is present
	_, hasEstimate := resp["estimated_unlock_date"]
	assert.True(t, hasEstimate, "response should contain estimated_unlock_date")
}

func TestProgressionHandlers_HandleGetVotingSession_WithEstimates(t *testing.T) {
	mockSvc := mocks.NewMockProgressionService(t)
	handler := NewProgressionHandlers(mockSvc)

	now := time.Now()
	sessionWithEst := &domain.ProgressionVotingSession{
		ID:     1,
		Status: "active",
		Options: []domain.ProgressionVotingOption{
			{NodeID: 1, VoteCount: 10, EstimatedUnlockDate: &now},
		},
	}

	mockSvc.On("GetActiveVotingSession", mock.Anything).Return(sessionWithEst, nil)

	req := httptest.NewRequest("GET", "/progression/session", nil)
	rec := httptest.NewRecorder()

	handler.HandleGetVotingSession()(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp VotingSessionResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotNil(t, resp.Session)
	assert.Equal(t, 1, resp.Session.ID)
	assert.Len(t, resp.Session.Options, 1)
	assert.NotNil(t, resp.Session.Options[0].EstimatedUnlockDate)
}

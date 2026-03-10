package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestQuestHandler_GetActiveQuests(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockQuestService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
				q.On("GetActiveQuests", mock.Anything).Return([]domain.Quest{
					{QuestID: 1, Description: "Test Quest"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"description":"Test Quest"`,
		},
		{
			name: "Feature Locked",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(false, nil)
				p.On("GetRequiredNodes", mock.Anything, "feature_weekly_quests").Return([]*domain.ProgressionNode{
					{DisplayName: "Weekly Quests"},
				}, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "LOCKED_NODES: Weekly Quests",
		},
		{
			name: "Service Error",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
				q.On("GetActiveQuests", mock.Anything).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to retrieve quests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuest := mocks.NewMockQuestService(t)
			mockProg := mocks.NewMockProgressionService(t)
			tt.setupMock(mockQuest, mockProg)

			handler := NewQuestHandler(mockQuest, mockProg)
			req := httptest.NewRequest("GET", "/quests", nil)
			w := httptest.NewRecorder()

			handler.GetActiveQuests(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockQuest.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestQuestHandler_GetUserQuestProgress(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(*mocks.MockQuestService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			userID: "user123",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
				q.On("GetUserQuestProgress", mock.Anything, "user123").Return([]domain.QuestProgress{
					{QuestID: 1, ProgressCurrent: 5},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"progress_current":5`,
		},
		{
			name:   "Missing UserID",
			userID: "",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "user_id is required",
		},
		{
			name:   "Service Error",
			userID: "user123",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
				q.On("GetUserQuestProgress", mock.Anything, "user123").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to retrieve quest progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuest := mocks.NewMockQuestService(t)
			mockProg := mocks.NewMockProgressionService(t)
			tt.setupMock(mockQuest, mockProg)

			handler := NewQuestHandler(mockQuest, mockProg)
			url := "/quests/progress"
			if tt.userID != "" {
				url += "?user_id=" + tt.userID
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handler.GetUserQuestProgress(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockQuest.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestQuestHandler_ClaimQuestReward(t *testing.T) {
	type claimRequest struct {
		UserID  string `json:"user_id"`
		QuestID int    `json:"quest_id"`
	}

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockQuestService, *mocks.MockProgressionService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			requestBody: claimRequest{
				UserID:  "user123",
				QuestID: 101,
			},
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
				q.On("ClaimQuestReward", mock.Anything, "user123", 101).Return(500, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"money_earned":500`,
		},
		{
			name:        "Invalid Request Body",
			requestBody: "invalid json",
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name: "Service Error - Not Found/Not Claimable",
			requestBody: claimRequest{
				UserID:  "user123",
				QuestID: 999,
			},
			setupMock: func(q *mocks.MockQuestService, p *mocks.MockProgressionService) {
				p.On("IsFeatureUnlocked", mock.Anything, "feature_weekly_quests").Return(true, nil)
				q.On("ClaimQuestReward", mock.Anything, "user123", 999).Return(0, errors.New("quest not found or not claimable"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to claim reward",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQuest := mocks.NewMockQuestService(t)
			mockProg := mocks.NewMockProgressionService(t)
			tt.setupMock(mockQuest, mockProg)

			handler := NewQuestHandler(mockQuest, mockProg)

			var body []byte
			if s, ok := tt.requestBody.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/quests/claim", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ClaimQuestReward(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockQuest.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

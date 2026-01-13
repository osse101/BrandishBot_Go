package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleGetUnlockProgress_WithPercentage(t *testing.T) {
	mockService := new(mocks.MockProgressionService)
	handler := &ProgressionHandlers{service: mockService}

	nodeID := 101
	targetLevel := 2
	unlockCost := 1000
	accumulated := 250

	mockProgress := &domain.UnlockProgress{
		ID:                       1,
		NodeID:                   &nodeID,
		TargetLevel:              &targetLevel,
		ContributionsAccumulated: accumulated,
		StartedAt:                time.Now(),
	}

	mockNode := &domain.ProgressionNode{
		ID:          nodeID,
		DisplayName: "Test Node",
		UnlockCost:  unlockCost,
	}

	mockService.On("GetUnlockProgress", mock.Anything).Return(mockProgress, nil)
	mockService.On("GetNode", mock.Anything, nodeID).Return(mockNode, nil)

	req, _ := http.NewRequest("GET", "/progression/unlock-progress", nil)
	rr := httptest.NewRecorder()

	handler.HandleGetUnlockProgress().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(25), response["completion_percentage"]) // 250 / 1000 * 100 = 25%
	assert.Equal(t, float64(unlockCost), response["target_unlock_cost"])
	assert.Equal(t, "Test Node", response["target_node_name"])
}

func TestHandleGetUnlockProgress_NoTarget(t *testing.T) {
	mockService := new(mocks.MockProgressionService)
	handler := &ProgressionHandlers{service: mockService}

	mockProgress := &domain.UnlockProgress{
		ID:                       1,
		NodeID:                   nil, // No target selected (voting phase)
		ContributionsAccumulated: 0,
		StartedAt:                time.Now(),
	}

	mockService.On("GetUnlockProgress", mock.Anything).Return(mockProgress, nil)

	req, _ := http.NewRequest("GET", "/progression/unlock-progress", nil)
	rr := httptest.NewRecorder()

	handler.HandleGetUnlockProgress().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(0), response["completion_percentage"])
	assert.Equal(t, float64(0), response["target_unlock_cost"])
}

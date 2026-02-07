package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestCheckFeatureLocked_Unlocked(t *testing.T) {
	svc := mocks.NewMockProgressionService(t)
	key := "test_feature"

	svc.On("IsFeatureUnlocked", mock.Anything, key).Return(true, nil)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	locked := CheckFeatureLocked(w, req, svc, key)

	assert.False(t, locked)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode) // No response written yet
}

func TestCheckFeatureLocked_Locked_NoRequiredNodes(t *testing.T) {
	svc := mocks.NewMockProgressionService(t)
	key := "test_feature"

	svc.On("IsFeatureUnlocked", mock.Anything, key).Return(false, nil)
	svc.On("GetRequiredNodes", mock.Anything, key).Return([]*domain.ProgressionNode{}, nil)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	locked := CheckFeatureLocked(w, req, svc, key)

	assert.True(t, locked)
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), ErrMsgFeatureLocked)
}

func TestCheckFeatureLocked_Locked_WithRequiredNodes(t *testing.T) {
	svc := mocks.NewMockProgressionService(t)
	key := "test_feature"

	requiredNodes := []*domain.ProgressionNode{
		{DisplayName: "Node A"},
		{DisplayName: "Node B"},
	}

	svc.On("IsFeatureUnlocked", mock.Anything, key).Return(false, nil)
	svc.On("GetRequiredNodes", mock.Anything, key).Return(requiredNodes, nil)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	locked := CheckFeatureLocked(w, req, svc, key)

	assert.True(t, locked)
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "LOCKED_NODES: Node A, Node B")
}

func TestCheckFeatureLocked_ServiceError_IsFeatureUnlocked(t *testing.T) {
	svc := mocks.NewMockProgressionService(t)
	key := "test_feature"

	svc.On("IsFeatureUnlocked", mock.Anything, key).Return(false, errors.New("database error"))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	locked := CheckFeatureLocked(w, req, svc, key)

	assert.True(t, locked)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestCheckFeatureLocked_ServiceError_GetRequiredNodes(t *testing.T) {
	svc := mocks.NewMockProgressionService(t)
	key := "test_feature"

	svc.On("IsFeatureUnlocked", mock.Anything, key).Return(false, nil)
	svc.On("GetRequiredNodes", mock.Anything, key).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	locked := CheckFeatureLocked(w, req, svc, key)

	assert.True(t, locked)
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode) // Still forbidden, fallback error
	assert.Contains(t, w.Body.String(), ErrMsgFeatureLocked)
}

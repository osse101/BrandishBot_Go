package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBPool mocks the database.Pool interface
type MockDBPool struct {
	mock.Mock
}

func (m *MockDBPool) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDBPool) Close() {
	m.Called()
}

// ... other methods if needed, but Ping is what we need for Readyz

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler := HandleHealthz()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"status":"ok"}`+"\n", w.Body.String())
}

func TestHandleReadyz(t *testing.T) {
	mockDB := &MockDBPool{}
	mockDB.On("Ping", mock.Anything).Return(nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()

	handler := HandleReadyz(mockDB)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"status":"ok"}`+"\n", w.Body.String())
	mockDB.AssertExpectations(t)
}

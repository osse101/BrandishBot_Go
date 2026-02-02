package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func init() {
	// Set log level to WARN for benchmarks (reduces noise)
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}

// BenchmarkHandler_HandleMessage benchmarks the full HTTP handler
func BenchmarkHandler_HandleMessage(b *testing.B) {
	mockUserService := mocks.NewMockUserService(b)
	mockProgressionService := mocks.NewMockProgressionService(b)
	mockEventBus := mocks.NewMockEventBus(b)

	// Set up expectations (minimal for benchmarking)
	mockUserService.On("HandleIncomingMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.MessageResult{
			User: domain.User{
				ID:       "bench-user-123",
				Username: "benchuser",
			},
			Matches: []domain.FoundString{},
		}, nil)

	handler := HandleMessageHandler(mockUserService, mockProgressionService, mockEventBus)

	reqBody := HandleMessageRequest{
		Platform:   "twitch",
		PlatformID: "12345",
		Username:   "benchuser",
		Message:    "hello world",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/message/handle", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkHandler_HandleMessage_ExistingUser benchmarks with cached user lookup
func BenchmarkHandler_HandleMessage_ExistingUser(b *testing.B) {
	mockUserService := mocks.NewMockUserService(b)
	mockProgressionService := mocks.NewMockProgressionService(b)
	mockEventBus := mocks.NewMockEventBus(b)

	mockUserService.On("HandleIncomingMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.MessageResult{
			User: domain.User{
				ID:       "bench-user-123",
				Username: "existinguser",
			},
			Matches: []domain.FoundString{},
		}, nil)

	handler := HandleMessageHandler(mockUserService, mockProgressionService, mockEventBus)

	reqBody := HandleMessageRequest{
		Platform:   "twitch",
		PlatformID: "existing-user-123",
		Username:   "existinguser",
		Message:    "test message",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/message/handle", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
	}
}

// BenchmarkHandler_HandleMessage_WithMatches benchmarks message with string matches
func BenchmarkHandler_HandleMessage_WithMatches(b *testing.B) {
	mockUserService := mocks.NewMockUserService(b)
	mockProgressionService := mocks.NewMockProgressionService(b)
	mockEventBus := mocks.NewMockEventBus(b)

	mockUserService.On("HandleIncomingMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.MessageResult{
			User: domain.User{
				ID:       "bench-user-123",
				Username: "matchuser",
			},
			Matches: []domain.FoundString{
				{Code: "test", Value: "test item"},
			},
		}, nil)

	handler := HandleMessageHandler(mockUserService, mockProgressionService, mockEventBus)

	reqBody := HandleMessageRequest{
		Platform:   "discord",
		PlatformID: "discord-789",
		Username:   "matchuser",
		Message:    "this message contains multiple words for matching",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/message/handle", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
	}
}

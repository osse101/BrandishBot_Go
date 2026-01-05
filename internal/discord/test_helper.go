package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// MockRoundTripper implements http.RoundTripper for intercepting requests
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// NewTestContext sets up the test environment:
// 1. Mock Backend API (httptest.Server)
// 2. Mock Discord Session (with intercepted HTTP client)
// 3. APIClient configured to talk to Mock Backend
func NewTestContext(t *testing.T) (*httptest.Server, *APIClient, *discordgo.Session, *MockRoundTripper) {
	// 1. Mock Backend API
	// Default handler returns 200 OK with empty JSON to prevent crashes if unlimited
	// Tests should override the mux handler or use a specific one
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// 2. APIClient pointing to Mock Backend
	client := NewAPIClient(server.URL, "test-api-key")

	// 3. Mock Discord Session
	session, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("Failed to create mock session: %v", err)
	}

	// Intercept Discord API calls
	mockTransport := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Default response for Discord interactions
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("{}")),
				Header:     make(http.Header),
			}, nil
		},
	}
	session.Client = &http.Client{Transport: mockTransport}

	t.Cleanup(func() {
		server.Close()
	})

	return server, client, session, mockTransport
}

// Helper to register a backend handler
func RegisterBackendHandler(server *httptest.Server, method, path string, handler http.HandlerFunc) {
	// Note: httptest.Server uses a mux that doesn't support method matching easily without wrapper
	// We'll rely on the caller to configure the mux passed to NewTestContext if they need complex routing
	// Or we can just modify the mux?
	// The problem is httptest.NewServer takes a handler.
	// Let's modify NewTestContext to return the mux.
}

// Simplified version for easier usage
type TestContext struct {
	Server        *httptest.Server
	Mux           *http.ServeMux
	APIClient     *APIClient
	Session       *discordgo.Session
	DiscordMocks  *MockRoundTripper
	CapturedEdits []*discordgo.WebhookEdit
}

func SetupTestContext(t *testing.T) *TestContext {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	client := NewAPIClient(server.URL, "test-api-key")

	session, _ := discordgo.New("Bot test-token")

	ctx := &TestContext{
		Server:    server,
		Mux:       mux,
		APIClient: client,
		Session:   session,
	}

	// Capture Discord calls
	ctx.DiscordMocks = &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Basic capture of interactions
			// Check if it's an interaction response edit/callback
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("{}")),
				Header:     make(http.Header),
			}, nil
		},
	}
	session.Client = &http.Client{Transport: ctx.DiscordMocks}

	t.Cleanup(func() {
		server.Close()
	})

	return ctx
}

// Helper to return JSON success
func WriteJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error if encoding fails in test helper
		_ = err
	}
}

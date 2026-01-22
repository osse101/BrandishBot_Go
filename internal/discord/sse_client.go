package discord

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEEvent represents a parsed SSE event
type SSEEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// SSEEventHandler handles a specific event type
type SSEEventHandler func(event SSEEvent) error

// SSEClient manages the connection to the API's SSE endpoint
type SSEClient struct {
	baseURL      string
	apiKey       string
	eventTypes   []string
	handlers     map[string][]SSEEventHandler
	httpClient   *http.Client
	mu           sync.RWMutex
	shutdown     chan struct{}
	wg           sync.WaitGroup
	connected    bool
	reconnecting bool
}

// NewSSEClient creates a new SSE client
func NewSSEClient(baseURL, apiKey string, eventTypes []string) *SSEClient {
	return &SSEClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		eventTypes: eventTypes,
		handlers:   make(map[string][]SSEEventHandler),
		httpClient: &http.Client{
			Timeout: 0, // No timeout for SSE connections
		},
		shutdown: make(chan struct{}),
	}
}

// OnEvent registers a handler for a specific event type
func (c *SSEClient) OnEvent(eventType string, handler SSEEventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = append(c.handlers[eventType], handler)
}

// Start begins the SSE connection with auto-reconnect
func (c *SSEClient) Start(ctx context.Context) {
	c.wg.Add(1)
	go c.connectLoop(ctx)
}

// Stop gracefully shuts down the SSE client
func (c *SSEClient) Stop() {
	close(c.shutdown)
	c.wg.Wait()
}

// IsConnected returns true if the client is connected
func (c *SSEClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *SSEClient) connectLoop(ctx context.Context) {
	defer c.wg.Done()

	backoff := sseInitialBackoff
	consecutiveFailures := 0

	for {
		select {
		case <-c.shutdown:
			slog.Info(sseLogMsgClientStopped)
			return
		case <-ctx.Done():
			slog.Info(sseLogMsgClientStopped)
			return
		default:
		}

		err := c.connect(ctx)
		if err != nil {
			consecutiveFailures++
			c.mu.Lock()
			c.connected = false
			c.reconnecting = true
			c.mu.Unlock()

			slog.Warn(sseLogMsgConnectionFailed,
				"error", err,
				"backoff", backoff,
				"consecutive_failures", consecutiveFailures)

			select {
			case <-time.After(backoff):
				// Exponential backoff with max
				backoff = time.Duration(float64(backoff) * sseBackoffMultiplier)
				if backoff > sseMaxBackoff {
					backoff = sseMaxBackoff
				}
			case <-c.shutdown:
				return
			case <-ctx.Done():
				return
			}
		} else {
			// Reset backoff on successful connection
			backoff = sseInitialBackoff
			consecutiveFailures = 0
		}
	}
}

func (c *SSEClient) connect(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/events", c.baseURL)
	if len(c.eventTypes) > 0 {
		url += "?types=" + strings.Join(c.eventTypes, ",")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	c.mu.Lock()
	c.connected = true
	c.reconnecting = false
	c.mu.Unlock()

	slog.Info(sseLogMsgClientConnected, "url", url)

	return c.readEvents(ctx, resp.Body)
}

func (c *SSEClient) readEvents(ctx context.Context, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, sseBufferSize), sseBufferSize)

	var eventID, eventType, data string

	for scanner.Scan() {
		select {
		case <-c.shutdown:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		if line == "" {
			// Empty line means end of event
			if data != "" {
				c.dispatchEvent(eventID, eventType, data)
			}
			eventID, eventType, data = "", "", ""
			continue
		}

		if strings.HasPrefix(line, "id: ") {
			eventID = strings.TrimPrefix(line, "id: ")
		} else if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return fmt.Errorf("stream closed unexpectedly")
}

func (c *SSEClient) dispatchEvent(id, eventType, data string) {
	if eventType == "" || eventType == "keepalive" || eventType == "connected" {
		return // Skip keepalive and connection events
	}

	var event SSEEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		slog.Warn(sseLogMsgParseError, "error", err, "data", data)
		return
	}

	// Override type from event line if present
	if eventType != "" {
		event.Type = eventType
	}
	if id != "" {
		event.ID = id
	}

	c.mu.RLock()
	handlers := c.handlers[event.Type]
	c.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			slog.Error(sseLogMsgHandlerError,
				"event_type", event.Type,
				"error", err)
		}
	}
}

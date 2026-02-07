package streamerbot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client manages the WebSocket connection to Streamer.bot
type Client struct {
	url      string
	password string
	conn     *websocket.Conn
	mu       sync.RWMutex
	shutdown chan struct{}
	wg       sync.WaitGroup

	// Connection state
	connected    bool
	reconnecting bool
	dormant      bool // Set to true after too many consecutive failures

	// Reconnection management
	wakeup chan struct{} // Used to trigger reconnection from dormant mode

	// Message handling
	responses map[string]chan *Response
	respMu    sync.RWMutex
}

// Request represents a Streamer.bot WebSocket request
type Request struct {
	Request string      `json:"request"`
	ID      string      `json:"id"`
	Action  *Action     `json:"action,omitempty"`
	Args    interface{} `json:"args,omitempty"`
}

// Action identifies a Streamer.bot action
type Action struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Response represents a Streamer.bot WebSocket response
type Response struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Error  string `json:"error,omitempty"`
}

// AuthChallenge represents the authentication challenge from Streamer.bot
type AuthChallenge struct {
	Info struct {
		Authentication struct {
			Challenge string `json:"challenge"`
			Salt      string `json:"salt"`
		} `json:"authentication"`
	} `json:"info"`
}

// NewClient creates a new Streamer.bot WebSocket client
func NewClient(url, password string) *Client {
	if url == "" {
		url = DefaultURL
	}
	return &Client{
		url:       url,
		password:  password,
		shutdown:  make(chan struct{}),
		wakeup:    make(chan struct{}, 1), // Buffered to avoid blocking
		responses: make(map[string]chan *Response),
	}
}

// Start begins the WebSocket connection with auto-reconnect
func (c *Client) Start(ctx context.Context) {
	c.wg.Add(1)
	go c.connectLoop(ctx)
}

// Stop gracefully shuts down the client
func (c *Client) Stop() {
	close(c.shutdown)
	c.wg.Wait()

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// DoAction sends a DoAction request to Streamer.bot
func (c *Client) DoAction(actionName string, args map[string]string) error {
	// If dormant, wake up and attempt reconnection
	c.mu.RLock()
	isDormant := c.dormant
	c.mu.RUnlock()

	if isDormant {
		slog.Debug(LogMsgDormantRetry)
		select {
		case c.wakeup <- struct{}{}:
		default:
			// Already waking up
		}
		return fmt.Errorf("Streamer.bot is dormant, reconnection triggered")
	}

	if !c.IsConnected() {
		return fmt.Errorf("not connected to Streamer.bot")
	}

	req := Request{
		Request: RequestDoAction,
		ID:      uuid.New().String(),
		Action: &Action{
			Name: actionName,
		},
		Args: args,
	}

	slog.Debug(LogMsgSendingAction, "action", actionName, "args", args)

	if err := c.sendRequest(req); err != nil {
		slog.Error(LogMsgActionFailed, "action", actionName, "error", err)
		return err
	}

	slog.Info(LogMsgActionSent, "action", actionName)
	return nil
}

func (c *Client) connectLoop(ctx context.Context) {
	defer c.wg.Done()

	backoff := DefaultReconnectDelay
	consecutiveFailures := 0

	for {
		select {
		case <-c.shutdown:
			slog.Info(LogMsgClientStopped)
			return
		case <-ctx.Done():
			slog.Info(LogMsgClientStopped)
			return
		default:
		}

		err := c.connect(ctx)
		if err != nil {
			consecutiveFailures++
			c.setConnected(false)

			// Check if we should give up and enter dormant mode
			if consecutiveFailures >= MaxConsecutiveFailures {
				if stop := c.handleDormantMode(ctx, &consecutiveFailures, &backoff); stop {
					return
				}
				continue
			}

			// Only log first few failures and then periodically to avoid log spam
			if consecutiveFailures <= 3 || consecutiveFailures%100 == 0 {
				slog.Warn(LogMsgReconnecting,
					"error", err,
					"backoff", backoff,
					"consecutive_failures", consecutiveFailures)
			}

			select {
			case <-time.After(backoff):
				backoff = time.Duration(float64(backoff) * ReconnectMultiplier)
				if backoff > MaxReconnectDelay {
					backoff = MaxReconnectDelay
				}
			case <-c.shutdown:
				return
			case <-ctx.Done():
				return
			}
		} else {
			// Log reconnection success if we had failures
			if consecutiveFailures > 0 {
				slog.Info("Streamer.bot connection restored", "after_failures", consecutiveFailures)
			}
			// Reset backoff and dormant state on successful connection
			backoff = DefaultReconnectDelay
			consecutiveFailures = 0
			c.mu.Lock()
			c.dormant = false
			c.mu.Unlock()
		}
	}
}

// handleDormantMode enters dormant mode after too many failures and waits for a wakeup signal
func (c *Client) handleDormantMode(ctx context.Context, consecutiveFailures *int, backoff *time.Duration) bool {
	c.mu.Lock()
	c.dormant = true
	c.mu.Unlock()

	slog.Warn(LogMsgGivingUp,
		"consecutive_failures", *consecutiveFailures,
		"max_allowed", MaxConsecutiveFailures)

	// Wait for wakeup signal or shutdown
	select {
	case <-c.wakeup:
		slog.Info("Streamer.bot waking from dormant mode")
		c.mu.Lock()
		c.dormant = false
		c.mu.Unlock()
		// Reset counters for fresh retry
		*backoff = DefaultReconnectDelay
		*consecutiveFailures = 0
		return false
	case <-c.shutdown:
		return true
	case <-ctx.Done():
		return true
	}
}

func (c *Client) connect(ctx context.Context) error {
	slog.Info(LogMsgConnecting, "url", c.url)

	dialer := websocket.Dialer{
		ReadBufferSize:  ReadBufferSize,
		WriteBufferSize: WriteBufferSize,
	}

	conn, resp, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("failed to connect: %w (status: %s, code: %d)", err, resp.Status, resp.StatusCode)
		}
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Read initial message (may contain auth challenge)
	// We use a short timeout because if auth is disabled, Streamer.bot might not send anything
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	_ = conn.SetReadDeadline(time.Time{}) // Reset deadline

	if err == nil {
		// Check if authentication is required
		var challenge AuthChallenge
		if err := json.Unmarshal(msg, &challenge); err == nil {
			if challenge.Info.Authentication.Challenge != "" {
				slog.Info(LogMsgAuthRequired)
				if err := c.authenticate(challenge); err != nil {
					conn.Close()
					return fmt.Errorf("authentication failed: %w", err)
				}
				slog.Info(LogMsgAuthSuccess)
			}
		}
	} else {
		slog.Debug("No initial message from Streamer.bot, proceeding assuming no auth required", "error", err)
	}

	c.setConnected(true)
	slog.Info(LogMsgConnected, "url", c.url)

	// Start read loop
	return c.readLoop(ctx)
}

func (c *Client) authenticate(challenge AuthChallenge) error {
	if c.password == "" {
		return fmt.Errorf("password required but not configured")
	}

	authHash := GenerateAuthHash(
		c.password,
		challenge.Info.Authentication.Salt,
		challenge.Info.Authentication.Challenge,
	)

	req := Request{
		Request: RequestAuthenticate,
		ID:      uuid.New().String(),
	}

	// Create auth request with authentication field
	authReq := struct {
		Request        string `json:"request"`
		ID             string `json:"id"`
		Authentication string `json:"authentication"`
	}{
		Request:        RequestAuthenticate,
		ID:             req.ID,
		Authentication: authHash,
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if err := conn.WriteJSON(authReq); err != nil {
		return fmt.Errorf("failed to send auth request: %w", err)
	}

	// Read auth response
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(msg, &resp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	if resp.Status != StatusOK {
		return fmt.Errorf("auth rejected: %s", resp.Error)
	}

	return nil
}

func (c *Client) readLoop(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	for {
		select {
		case <-c.shutdown:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			slog.Warn(LogMsgReadError, "error", err)
			return err
		}

		// Parse response and route to waiting callers if any
		var resp Response
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue // Ignore unparseable messages
		}

		if resp.ID != "" {
			c.respMu.RLock()
			ch, ok := c.responses[resp.ID]
			c.respMu.RUnlock()
			if ok {
				select {
				case ch <- &resp:
				default:
				}
			}
		}
	}
}

func (c *Client) sendRequest(req Request) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("no connection")
	}

	_ = conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
	return conn.WriteJSON(req)
}

func (c *Client) setConnected(connected bool) {
	c.mu.Lock()
	c.connected = connected
	c.reconnecting = !connected
	c.mu.Unlock()
}

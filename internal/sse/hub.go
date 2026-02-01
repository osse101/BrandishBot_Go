package sse

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event represents an event sent over SSE
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Client represents a connected SSE client
type Client struct {
	ID           string
	EventChannel chan Event
	EventFilter  map[string]bool // nil means all events, otherwise only specified types
	done         chan struct{}
}

// Hub manages SSE client connections and event broadcasting
type Hub struct {
	clients    map[string]*Client
	broadcast  chan Event
	register   chan *Client
	unregister chan string
	mu         sync.RWMutex
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

// NewHub creates a new SSE Hub
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan Event, BroadcastBufferSize),
		register:   make(chan *Client, ClientChannelBuffer),
		unregister: make(chan string, ClientChannelBuffer),
		shutdown:   make(chan struct{}),
	}
	return h
}

// Start starts the hub's broadcast loop
func (h *Hub) Start() {
	h.wg.Add(1)
	go h.run()
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	close(h.shutdown)
	h.wg.Wait()

	// Close all client channels
	h.mu.Lock()
	for _, client := range h.clients {
		close(client.EventChannel)
	}
	h.clients = make(map[string]*Client)
	h.mu.Unlock()
}

// run is the main broadcast loop
func (h *Hub) run() {
	defer h.wg.Done()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()

		case clientID := <-h.unregister:
			h.mu.Lock()
			if client, ok := h.clients[clientID]; ok {
				close(client.EventChannel)
				delete(h.clients, clientID)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				// Check if client wants this event type
				if client.EventFilter != nil && !client.EventFilter[event.Type] {
					continue
				}

				// Non-blocking send
				select {
				case client.EventChannel <- event:
					// Sent successfully
				default:
					// Client buffer full, skip this event
				}
			}
			h.mu.RUnlock()

		case <-h.shutdown:
			return
		}
	}
}

// Register adds a new client to the hub
func (h *Hub) Register(eventTypes []string) *Client {
	client := &Client{
		ID:           uuid.New().String(),
		EventChannel: make(chan Event, ClientEventBuffer),
		done:         make(chan struct{}),
	}

	// Set up event filter if specific types requested
	if len(eventTypes) > 0 {
		client.EventFilter = make(map[string]bool)
		for _, t := range eventTypes {
			client.EventFilter[t] = true
		}
	}

	h.register <- client
	return client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(clientID string) {
	select {
	case h.unregister <- clientID:
	case <-h.shutdown:
	}
}

// Broadcast sends an event to all interested clients
func (h *Hub) Broadcast(eventType string, payload interface{}) {
	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	select {
	case h.broadcast <- event:
		// Sent successfully
	default:
		// Buffer full, drop event (logged elsewhere)
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// FormatSSEMessage formats an SSE event for transmission
func FormatSSEMessage(event Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	// SSE format: "id: <id>\nevent: <type>\ndata: <json>\n\n"
	msg := "id: " + event.ID + "\n"
	msg += "event: " + event.Type + "\n"
	msg += "data: " + string(data) + "\n\n"

	return []byte(msg), nil
}

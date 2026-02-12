package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEBroker manages server-sent event subscriptions and broadcasting.
type SSEBroker struct {
	mu          sync.RWMutex
	clients     map[chan SSEEvent]struct{}
	register    chan chan SSEEvent
	unregister  chan chan SSEEvent
	broadcast   chan SSEEvent
	heartbeat   time.Duration
}

// NewSSEBroker creates a new SSE broker with default heartbeat interval.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan SSEEvent]struct{}),
		register:   make(chan chan SSEEvent),
		unregister: make(chan chan SSEEvent),
		broadcast:  make(chan SSEEvent, 256),
		heartbeat:  30 * time.Second,
	}
}

// Start runs the SSE broker event loop. It blocks until the context is cancelled.
func (b *SSEBroker) Start(ctx context.Context) {
	ticker := time.NewTicker(b.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.mu.Lock()
			for ch := range b.clients {
				close(ch)
				delete(b.clients, ch)
			}
			b.mu.Unlock()
			return

		case ch := <-b.register:
			b.mu.Lock()
			b.clients[ch] = struct{}{}
			b.mu.Unlock()

		case ch := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[ch]; ok {
				close(ch)
				delete(b.clients, ch)
			}
			b.mu.Unlock()

		case event := <-b.broadcast:
			b.mu.RLock()
			for ch := range b.clients {
				select {
				case ch <- event:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()

		case <-ticker.C:
			heartbeat := SSEEvent{
				Type: "heartbeat",
				Data: map[string]int64{"timestamp": time.Now().Unix()},
			}
			b.mu.RLock()
			for ch := range b.clients {
				select {
				case ch <- heartbeat:
				default:
				}
			}
			b.mu.RUnlock()
		}
	}
}

// Publish sends an event to all connected clients.
func (b *SSEBroker) Publish(event SSEEvent) {
	select {
	case b.broadcast <- event:
	default:
		// Broadcast buffer full, drop event
	}
}

// Subscribe registers a new client and returns a channel for receiving events.
func (b *SSEBroker) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	b.register <- ch
	return ch
}

// Unsubscribe removes a client.
func (b *SSEBroker) Unsubscribe(ch chan SSEEvent) {
	b.unregister <- ch
}

// ClientCount returns the number of connected SSE clients.
func (b *SSEBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// ServeHTTP handles SSE connections.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	ctx := r.Context()

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}

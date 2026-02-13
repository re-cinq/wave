//go:build webui

package webui

import (
	"sync"
)

// SSEBroker manages Server-Sent Events connections for real-time dashboard updates.
type SSEBroker struct {
	clients    map[chan SSEEvent]struct{}
	register   chan chan SSEEvent
	unregister chan chan SSEEvent
	broadcast  chan SSEEvent
	stop       chan struct{}
	mu         sync.RWMutex
}

// SSEEvent represents a server-sent event.
type SSEEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan SSEEvent]struct{}),
		register:   make(chan chan SSEEvent),
		unregister: make(chan chan SSEEvent),
		broadcast:  make(chan SSEEvent, 256),
		stop:       make(chan struct{}),
	}
}

// Start runs the broker event loop. Call in a goroutine.
func (b *SSEBroker) Start() {
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = struct{}{}
			b.mu.Unlock()

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
			}
			b.mu.Unlock()

		case event := <-b.broadcast:
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client <- event:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()

		case <-b.stop:
			b.mu.Lock()
			for client := range b.clients {
				close(client)
				delete(b.clients, client)
			}
			b.mu.Unlock()
			return
		}
	}
}

// Stop shuts down the broker.
func (b *SSEBroker) Stop() {
	select {
	case b.stop <- struct{}{}:
	default:
	}
}

// Subscribe registers a new SSE client and returns its event channel.
func (b *SSEBroker) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	b.register <- ch
	return ch
}

// Unsubscribe removes an SSE client.
func (b *SSEBroker) Unsubscribe(ch chan SSEEvent) {
	b.unregister <- ch
}

// Publish sends an event to all connected clients.
func (b *SSEBroker) Publish(event SSEEvent) {
	select {
	case b.broadcast <- event:
	default:
		// Broadcast buffer full, drop event
	}
}

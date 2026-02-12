package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSEBroker(t *testing.T) {
	broker := NewSSEBroker()
	assert.NotNil(t, broker)
	assert.Equal(t, 0, broker.ClientCount())
}

func TestSSEBrokerSubscribeUnsubscribe(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)

	// Allow broker to start
	time.Sleep(10 * time.Millisecond)

	// Subscribe
	ch := broker.Subscribe()
	require.NotNil(t, ch)

	// Wait for registration to be processed
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, broker.ClientCount())

	// Subscribe another
	ch2 := broker.Subscribe()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, broker.ClientCount())

	// Unsubscribe first
	broker.Unsubscribe(ch)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, broker.ClientCount())

	// Unsubscribe second
	broker.Unsubscribe(ch2)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, broker.ClientCount())
}

func TestSSEBrokerBroadcast(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	ch1 := broker.Subscribe()
	ch2 := broker.Subscribe()
	time.Sleep(10 * time.Millisecond)

	// Publish an event
	event := SSEEvent{
		Type: "run_update",
		Data: map[string]string{"run_id": "test-123"},
	}
	broker.Publish(event)

	// Both clients should receive it
	select {
	case received := <-ch1:
		assert.Equal(t, "run_update", received.Type)
	case <-time.After(time.Second):
		t.Fatal("client 1 did not receive event")
	}

	select {
	case received := <-ch2:
		assert.Equal(t, "run_update", received.Type)
	case <-time.After(time.Second):
		t.Fatal("client 2 did not receive event")
	}

	broker.Unsubscribe(ch1)
	broker.Unsubscribe(ch2)
}

func TestSSEBrokerHeartbeat(t *testing.T) {
	broker := NewSSEBroker()
	broker.heartbeat = 50 * time.Millisecond // Short interval for testing

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	ch := broker.Subscribe()
	time.Sleep(10 * time.Millisecond)

	// Wait for a heartbeat
	select {
	case received := <-ch:
		assert.Equal(t, "heartbeat", received.Type)
	case <-time.After(time.Second):
		t.Fatal("did not receive heartbeat")
	}

	broker.Unsubscribe(ch)
}

func TestSSEBrokerContextCancellation(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())

	go broker.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	ch := broker.Subscribe()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, broker.ClientCount())

	// Cancel context - should close all clients
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after context cancellation")
}

func TestSSEBrokerPublishNoClients(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go broker.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Should not panic when publishing with no clients
	broker.Publish(SSEEvent{Type: "test", Data: "data"})
}

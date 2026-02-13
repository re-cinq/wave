//go:build webui

package webui

import (
	"testing"
	"time"
)

func TestSSEBroker_PubSub(t *testing.T) {
	broker := NewSSEBroker()
	go broker.Start()
	defer broker.Stop()

	// Give broker time to start
	time.Sleep(10 * time.Millisecond)

	// Subscribe
	ch := broker.Subscribe()

	// Publish
	broker.Publish(SSEEvent{
		Event: "test",
		Data:  `{"msg":"hello"}`,
	})

	// Receive with timeout
	select {
	case event := <-ch:
		if event.Event != "test" {
			t.Errorf("expected event type 'test', got %q", event.Event)
		}
		if event.Data != `{"msg":"hello"}` {
			t.Errorf("expected data %q, got %q", `{"msg":"hello"}`, event.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Unsubscribe
	broker.Unsubscribe(ch)
}

func TestSSEBroker_MultipleClients(t *testing.T) {
	broker := NewSSEBroker()
	go broker.Start()
	defer broker.Stop()

	time.Sleep(10 * time.Millisecond)

	ch1 := broker.Subscribe()
	ch2 := broker.Subscribe()

	broker.Publish(SSEEvent{Event: "test", Data: "data"})

	// Both clients should receive the event
	for i, ch := range []chan SSEEvent{ch1, ch2} {
		select {
		case event := <-ch:
			if event.Event != "test" {
				t.Errorf("client %d: expected event 'test', got %q", i, event.Event)
			}
		case <-time.After(time.Second):
			t.Errorf("client %d: timeout waiting for event", i)
		}
	}

	broker.Unsubscribe(ch1)
	broker.Unsubscribe(ch2)
}

func TestSSEBroker_UnsubscribeStopsDelivery(t *testing.T) {
	broker := NewSSEBroker()
	go broker.Start()
	defer broker.Stop()

	time.Sleep(10 * time.Millisecond)

	ch := broker.Subscribe()
	broker.Unsubscribe(ch)

	// Give time for unsubscribe to process
	time.Sleep(10 * time.Millisecond)

	// Publish after unsubscribe - should not panic or block
	broker.Publish(SSEEvent{Event: "test", Data: "data"})

	// Channel should be closed after unsubscribe
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after unsubscribe")
		}
	case <-time.After(100 * time.Millisecond):
		// Channel is closed and drained, or was never written to
	}
}

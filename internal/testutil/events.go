package testutil

import (
	"sync"

	"github.com/recinq/wave/internal/event"
)

// EventCollector is a thread-safe event.EventEmitter that collects events for test assertions.
type EventCollector struct {
	mu     sync.Mutex
	events []event.Event
}

// NewEventCollector creates a new EventCollector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]event.Event, 0),
	}
}

// Emit records an event. Safe for concurrent use.
func (c *EventCollector) Emit(e event.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

// GetEvents returns a copy of all collected events.
func (c *EventCollector) GetEvents() []event.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]event.Event, len(c.events))
	copy(result, c.events)
	return result
}

// GetPipelineID returns the pipeline ID from the first event that has a non-empty PipelineID.
func (c *EventCollector) GetPipelineID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e.PipelineID != "" {
			return e.PipelineID
		}
	}
	return ""
}

// HasEventWithState returns true if any collected event has the given state.
func (c *EventCollector) HasEventWithState(state string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e.State == state {
			return true
		}
	}
	return false
}

// GetEventsByStep returns all events for a given step ID.
func (c *EventCollector) GetEventsByStep(stepID string) []event.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []event.Event
	for _, e := range c.events {
		if e.StepID == stepID {
			result = append(result, e)
		}
	}
	return result
}

// GetStepExecutionOrder returns step IDs in the order they entered "running" state.
func (c *EventCollector) GetStepExecutionOrder() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var order []string
	seen := make(map[string]bool)
	for _, e := range c.events {
		if e.StepID != "" && e.State == "running" && !seen[e.StepID] {
			order = append(order, e.StepID)
			seen[e.StepID] = true
		}
	}
	return order
}

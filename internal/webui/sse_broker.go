//go:build webui

package webui

import (
	"encoding/json"

	"github.com/recinq/wave/internal/event"
)

// EmitProgress implements the event.ProgressEmitter interface so the SSE broker
// can receive pipeline progress events and broadcast them to connected clients.
func (b *SSEBroker) EmitProgress(ev event.Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	b.Publish(SSEEvent{
		Event: ev.State,
		Data:  string(data),
	})

	return nil
}

// Emit implements event.EventEmitter so the broker can be used as the
// executor's event sink, bridging pipeline progress into SSE streams.
func (b *SSEBroker) Emit(ev event.Event) {
	_ = b.EmitProgress(ev)
}

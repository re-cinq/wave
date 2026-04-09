package pipeline

import "github.com/recinq/wave/internal/event"

// emitterMixin provides a reusable emit() method that wraps the nil-check
// pattern used by all executor types. Embed this struct in executor types
// that have a direct emitter field.
type emitterMixin struct {
	emitter event.EventEmitter
}

func (m *emitterMixin) emit(ev event.Event) {
	if m.emitter != nil {
		m.emitter.Emit(ev)
	}
}

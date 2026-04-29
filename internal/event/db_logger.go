package event

// EventLogger is the minimal persistence surface DBLoggingEmitter needs.
// Defined here (rather than depending on internal/state directly) to avoid
// an import cycle: state already imports event for the canonical event types.
//
// The signature matches state.EventStore.LogEvent so a *state.EventStore
// satisfies this interface implicitly.
type EventLogger interface {
	LogEvent(runID string, stepID string, state string, persona string, message string, tokens int, durationMs int64, model string, configuredModel string, adapter string) error
}

// LogErrorFunc is an optional callback for logging persistence failures.
// Pass nil to silently ignore LogEvent errors.
type LogErrorFunc func(runID string, err error)

// DBLoggingEmitter wraps an EventEmitter and also persists each event via an
// EventLogger so that "wave logs <run-id>" returns a complete history for
// CLI-launched runs and the webui dashboard timeline stays populated.
//
// Empty-payload heartbeat ticks are skipped (they carry no useful information).
// A stream_activity event with a non-empty ToolName is preserved; its message
// is composed from ToolName+ToolTarget when the message field is empty.
//
// The event's PipelineID is preferred over the constructor RunID so child
// sub-pipeline events are logged under the child's run ID; RunID is used as
// the fallback when PipelineID is empty.
type DBLoggingEmitter struct {
	Inner    EventEmitter
	Store    EventLogger
	RunID    string
	OnError  LogErrorFunc // optional; nil silently swallows LogEvent errors
}

// Emit forwards the event to the wrapped emitter and persists it (when
// non-heartbeat) via the configured EventLogger.
func (d *DBLoggingEmitter) Emit(ev Event) {
	if d.Inner != nil {
		d.Inner.Emit(ev)
	}
	if d.Store == nil {
		return
	}
	if isHeartbeatTick(ev) {
		return
	}
	msg := ev.Message
	if msg == "" && ev.ToolName != "" {
		msg = ev.ToolName + " " + ev.ToolTarget
	}
	runID := ev.PipelineID
	if runID == "" {
		runID = d.RunID
	}
	if err := d.Store.LogEvent(runID, ev.StepID, ev.State, ev.Persona, msg, ev.TokensUsed, ev.DurationMs, ev.Model, ev.ConfiguredModel, ev.Adapter); err != nil && d.OnError != nil {
		d.OnError(runID, err)
	}
}

// isHeartbeatTick returns true for progress/stream_activity ticker events
// that carry no useful information. A stream_activity event with a non-empty
// ToolName is NOT a heartbeat — it represents a real Claude Code tool call.
func isHeartbeatTick(ev Event) bool {
	if ev.Message != "" || ev.ToolName != "" {
		return false
	}
	if ev.TokensUsed != 0 || ev.DurationMs != 0 {
		return false
	}
	return ev.State == StateStepProgress || ev.State == StateStreamActivity
}

package tui

import (
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
)

// eventFromLogRecord adapts a persisted state.LogRecord into an event.Event so
// the canonical display.EventLine formatter can render it.
//
// LogRecord lacks several fields present on event.Event (TokensIn/Out,
// ToolName/Target, ValidationPhase, EstimatedTimeMs, etc.). Only fields the
// formatter consumes are mapped; absent fields stay zero, and the formatter's
// existing graceful-degradation branches handle them. The pre-refactor
// formatStoredEvent special-cased contract_validating and stream_activity by
// reading ev.Message; the formatter mirrors that behavior via its
// stored-fallback branches keyed off empty ToolName / non-empty Message.
//
// When LogRecord.StepID is empty, the bracket label falls back to "pipeline"
// to match the pre-refactor formatStoredEvent default.
func eventFromLogRecord(rec state.LogRecord) event.Event {
	stepID := rec.StepID
	if stepID == "" {
		stepID = "pipeline"
	}

	evt := event.Event{
		Timestamp:       rec.Timestamp,
		StepID:          stepID,
		State:           rec.State,
		DurationMs:      rec.DurationMs,
		Message:         rec.Message,
		Persona:         rec.Persona,
		TokensUsed:      rec.TokensUsed,
		Model:           rec.Model,
		ConfiguredModel: rec.ConfiguredModel,
		Adapter:         rec.Adapter,
	}

	// formatStoredEvent's contract_validating branch reads phase from
	// ev.Message. The canonical formatter reads it from ValidationPhase, so
	// promote Message -> ValidationPhase for that one state.
	if rec.State == "contract_validating" {
		evt.ValidationPhase = rec.Message
	}

	return evt
}

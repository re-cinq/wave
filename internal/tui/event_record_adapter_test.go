package tui

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

func TestEventFromLogRecord_FieldMapping(t *testing.T) {
	ts := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	rec := state.LogRecord{
		Timestamp:       ts,
		StepID:          "specify",
		State:           "completed",
		DurationMs:      5000,
		Message:         "ok",
		Persona:         "navigator",
		TokensUsed:      12345,
		Model:           "opus",
		ConfiguredModel: "cheapest",
		Adapter:         "claude",
	}
	evt := eventFromLogRecord(rec)

	if !evt.Timestamp.Equal(ts) {
		t.Errorf("Timestamp lost: got %v want %v", evt.Timestamp, ts)
	}
	if evt.StepID != "specify" {
		t.Errorf("StepID got %q", evt.StepID)
	}
	if evt.State != "completed" || evt.DurationMs != 5000 || evt.Message != "ok" || evt.Persona != "navigator" ||
		evt.TokensUsed != 12345 || evt.Model != "opus" || evt.ConfiguredModel != "cheapest" || evt.Adapter != "claude" {
		t.Errorf("field mapping mismatch: %+v", evt)
	}
}

func TestEventFromLogRecord_StepIDFallback(t *testing.T) {
	rec := state.LogRecord{State: "started"}
	evt := eventFromLogRecord(rec)
	if evt.StepID != "pipeline" {
		t.Errorf("expected fallback StepID 'pipeline', got %q", evt.StepID)
	}
}

func TestEventFromLogRecord_ContractValidatingPromotesPhase(t *testing.T) {
	rec := state.LogRecord{StepID: "plan", State: "contract_validating", Message: "json_schema"}
	evt := eventFromLogRecord(rec)
	if evt.ValidationPhase != "json_schema" {
		t.Errorf("expected ValidationPhase from Message; got %q", evt.ValidationPhase)
	}
}

func TestEventFromLogRecord_OtherStatesNoPhasePromotion(t *testing.T) {
	rec := state.LogRecord{StepID: "s", State: "running", Message: "doing"}
	evt := eventFromLogRecord(rec)
	if evt.ValidationPhase != "" {
		t.Errorf("ValidationPhase should be empty for non-contract states; got %q", evt.ValidationPhase)
	}
	if evt.Message != "doing" {
		t.Errorf("Message should still be set; got %q", evt.Message)
	}
}

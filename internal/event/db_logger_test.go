package event

import (
	"errors"
	"testing"
)

type fakeLogger struct {
	calls []logCall
	err   error
}

type logCall struct {
	runID, stepID, state, persona, message string
	tokens                                 int
	durationMs                             int64
}

func (f *fakeLogger) LogEvent(runID, stepID, state, persona, message string, tokens int, durationMs int64, _, _, _ string) error {
	f.calls = append(f.calls, logCall{runID, stepID, state, persona, message, tokens, durationMs})
	return f.err
}

type captureEmitter struct{ events []Event }

func (c *captureEmitter) Emit(ev Event) { c.events = append(c.events, ev) }

func TestDBLoggingEmitter_HeartbeatSuppression(t *testing.T) {
	tests := []struct {
		name      string
		ev        Event
		wantStore bool
	}{
		{
			name:      "empty step_progress is suppressed",
			ev:        Event{State: StateStepProgress},
			wantStore: false,
		},
		{
			name:      "empty stream_activity is suppressed",
			ev:        Event{State: StateStreamActivity},
			wantStore: false,
		},
		{
			name:      "stream_activity with ToolName is persisted",
			ev:        Event{State: StateStreamActivity, ToolName: "Read", ToolTarget: "/tmp/foo"},
			wantStore: true,
		},
		{
			name:      "step_progress with message is persisted",
			ev:        Event{State: StateStepProgress, Message: "halfway"},
			wantStore: true,
		},
		{
			name:      "stream_activity with tokens is persisted",
			ev:        Event{State: StateStreamActivity, TokensUsed: 100},
			wantStore: true,
		},
		{
			name:      "non-progress states are always persisted",
			ev:        Event{State: StateCompleted},
			wantStore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLogger{}
			cap := &captureEmitter{}
			d := &DBLoggingEmitter{Inner: cap, Store: fake, RunID: "run-1"}
			d.Emit(tt.ev)
			if len(cap.events) != 1 {
				t.Fatalf("inner emitter not invoked: got %d events", len(cap.events))
			}
			gotStore := len(fake.calls) == 1
			if gotStore != tt.wantStore {
				t.Errorf("store invoked=%v, want %v (calls=%d)", gotStore, tt.wantStore, len(fake.calls))
			}
		})
	}
}

func TestDBLoggingEmitter_ToolMessageComposition(t *testing.T) {
	fake := &fakeLogger{}
	d := &DBLoggingEmitter{Inner: &captureEmitter{}, Store: fake, RunID: "run-1"}
	d.Emit(Event{State: StateStreamActivity, ToolName: "Bash", ToolTarget: "ls -la"})
	if len(fake.calls) != 1 {
		t.Fatalf("expected 1 LogEvent call, got %d", len(fake.calls))
	}
	if fake.calls[0].message != "Bash ls -la" {
		t.Errorf("message=%q, want %q", fake.calls[0].message, "Bash ls -la")
	}
}

func TestDBLoggingEmitter_PipelineIDOverridesRunID(t *testing.T) {
	fake := &fakeLogger{}
	d := &DBLoggingEmitter{Inner: &captureEmitter{}, Store: fake, RunID: "parent"}

	d.Emit(Event{State: StateCompleted, PipelineID: "child-run"})
	d.Emit(Event{State: StateCompleted})

	if len(fake.calls) != 2 {
		t.Fatalf("expected 2 LogEvent calls, got %d", len(fake.calls))
	}
	if fake.calls[0].runID != "child-run" {
		t.Errorf("first runID=%q, want child-run", fake.calls[0].runID)
	}
	if fake.calls[1].runID != "parent" {
		t.Errorf("second runID=%q, want parent (fallback)", fake.calls[1].runID)
	}
}

func TestDBLoggingEmitter_OnError(t *testing.T) {
	fake := &fakeLogger{err: errors.New("boom")}
	var captured struct {
		runID string
		err   error
	}
	d := &DBLoggingEmitter{
		Inner: &captureEmitter{},
		Store: fake,
		RunID: "run-1",
		OnError: func(rid string, err error) {
			captured.runID = rid
			captured.err = err
		},
	}
	d.Emit(Event{State: StateCompleted})
	if captured.runID != "run-1" || captured.err == nil {
		t.Errorf("OnError not invoked correctly: runID=%q err=%v", captured.runID, captured.err)
	}
}

func TestDBLoggingEmitter_NilStoreSkipsPersist(t *testing.T) {
	cap := &captureEmitter{}
	d := &DBLoggingEmitter{Inner: cap, Store: nil, RunID: "run-1"}
	d.Emit(Event{State: StateCompleted, Message: "hello"})
	if len(cap.events) != 1 {
		t.Errorf("inner emitter should still receive event, got %d", len(cap.events))
	}
}

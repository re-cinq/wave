package commands

import (
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
)

type logEventCall struct {
	runID, stepID, state, persona, message string
	tokens                                 int
	durationMs                             int64
	model, configuredModel, adapter        string
}

// fakeLogEventStore satisfies state.StateStore by embedding the interface.
// Only LogEvent is implemented — any other method call panics, which is fine
// because dbLoggingEmitter.Emit only invokes LogEvent.
type fakeLogEventStore struct {
	state.StateStore
	calls []logEventCall
}

func (f *fakeLogEventStore) LogEvent(runID, stepID, st, persona, message string, tokens int, durationMs int64, model, configuredModel, adapter string) error {
	f.calls = append(f.calls, logEventCall{
		runID:           runID,
		stepID:          stepID,
		state:           st,
		persona:         persona,
		message:         message,
		tokens:          tokens,
		durationMs:      durationMs,
		model:           model,
		configuredModel: configuredModel,
		adapter:         adapter,
	})
	return nil
}

type fakeEventEmitter struct {
	events []event.Event
}

func (f *fakeEventEmitter) Emit(e event.Event) {
	f.events = append(f.events, e)
}

func TestDBLoggingEmitter_Emit(t *testing.T) {
	tests := []struct {
		name        string
		ev          event.Event
		wantPersist bool
		wantMessage string
		wantRunID   string
	}{
		{
			name:        "empty step_progress heartbeat is dropped",
			ev:          event.Event{State: "step_progress"},
			wantPersist: false,
		},
		{
			name:        "empty stream_activity heartbeat is dropped",
			ev:          event.Event{State: "stream_activity"},
			wantPersist: false,
		},
		{
			name: "stream_activity with ToolName composes message",
			ev: event.Event{
				State:      "stream_activity",
				ToolName:   "Read",
				ToolTarget: "cmd/wave/commands/run.go",
				StepID:     "step-1",
				Persona:    "navigator",
			},
			wantPersist: true,
			wantMessage: "Read cmd/wave/commands/run.go",
			wantRunID:   "default-run",
		},
		{
			name: "step_progress with tokens used is persisted",
			ev: event.Event{
				State:      "step_progress",
				TokensUsed: 42,
				StepID:     "step-1",
			},
			wantPersist: true,
			wantMessage: "",
			wantRunID:   "default-run",
		},
		{
			name: "step_progress with duration is persisted",
			ev: event.Event{
				State:      "step_progress",
				DurationMs: 100,
				StepID:     "step-1",
			},
			wantPersist: true,
			wantMessage: "",
			wantRunID:   "default-run",
		},
		{
			name: "running state with message is persisted",
			ev: event.Event{
				State:   "running",
				Message: "step started",
				StepID:  "step-1",
				Persona: "implementer",
			},
			wantPersist: true,
			wantMessage: "step started",
			wantRunID:   "default-run",
		},
		{
			name: "completed state with no message still persists (not heartbeat)",
			ev: event.Event{
				State:  "completed",
				StepID: "step-1",
			},
			wantPersist: true,
			wantMessage: "",
			wantRunID:   "default-run",
		},
		{
			name: "event with PipelineID overrides default runID",
			ev: event.Event{
				State:      "running",
				Message:    "child running",
				PipelineID: "child-run-id",
				StepID:     "step-1",
			},
			wantPersist: true,
			wantMessage: "child running",
			wantRunID:   "child-run-id",
		},
		{
			name: "stream_activity with ToolName but no target",
			ev: event.Event{
				State:    "stream_activity",
				ToolName: "Bash",
				StepID:   "step-1",
			},
			wantPersist: true,
			wantMessage: "Bash ",
			wantRunID:   "default-run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &fakeEventEmitter{}
			store := &fakeLogEventStore{}
			d := &dbLoggingEmitter{inner: inner, store: store, runID: "default-run"}

			d.Emit(tt.ev)

			if len(inner.events) != 1 {
				t.Fatalf("inner.Emit called %d times, want 1", len(inner.events))
			}
			if inner.events[0].State != tt.ev.State || inner.events[0].StepID != tt.ev.StepID || inner.events[0].Message != tt.ev.Message {
				t.Errorf("inner received unexpected event: got %+v want %+v", inner.events[0], tt.ev)
			}

			if !tt.wantPersist {
				if len(store.calls) != 0 {
					t.Errorf("LogEvent called %d times, want 0 (heartbeat should be dropped); calls=%+v", len(store.calls), store.calls)
				}
				return
			}

			if len(store.calls) != 1 {
				t.Fatalf("LogEvent called %d times, want 1", len(store.calls))
			}
			c := store.calls[0]
			if c.runID != tt.wantRunID {
				t.Errorf("LogEvent runID = %q, want %q", c.runID, tt.wantRunID)
			}
			if c.message != tt.wantMessage {
				t.Errorf("LogEvent message = %q, want %q", c.message, tt.wantMessage)
			}
			if c.state != tt.ev.State {
				t.Errorf("LogEvent state = %q, want %q", c.state, tt.ev.State)
			}
			if c.stepID != tt.ev.StepID {
				t.Errorf("LogEvent stepID = %q, want %q", c.stepID, tt.ev.StepID)
			}
			if c.persona != tt.ev.Persona {
				t.Errorf("LogEvent persona = %q, want %q", c.persona, tt.ev.Persona)
			}
			if c.tokens != tt.ev.TokensUsed {
				t.Errorf("LogEvent tokens = %d, want %d", c.tokens, tt.ev.TokensUsed)
			}
			if c.durationMs != tt.ev.DurationMs {
				t.Errorf("LogEvent durationMs = %d, want %d", c.durationMs, tt.ev.DurationMs)
			}
		})
	}
}

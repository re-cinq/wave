package display

import (
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func TestEventLine_LiveTUI_Started(t *testing.T) {
	evt := event.Event{StepID: "specify", State: event.StateStarted, Persona: "navigator", Model: "opus"}
	line, emit := EventLine(evt, LiveTUIProfile(true))
	if !emit {
		t.Fatal("emit should be true for live profile")
	}
	for _, want := range []string{"[specify]", "Starting...", "navigator", "opus"} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q missing %q", line, want)
		}
	}
}

func TestEventLine_LiveTUI_Completed_Color(t *testing.T) {
	evt := event.Event{StepID: "plan", State: event.StateCompleted, DurationMs: 42000}
	got, _ := EventLine(evt, LiveTUIProfile(true))
	want := "[plan] ✓ Completed (42s)"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestEventLine_LiveTUI_Completed_NoColor(t *testing.T) {
	evt := event.Event{StepID: "plan", State: event.StateCompleted, DurationMs: 42000}
	got, _ := EventLine(evt, LiveTUIProfile(false))
	want := "[plan] Completed (42s)"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestEventLine_LiveTUI_FallbackPrefix(t *testing.T) {
	evt := event.Event{PipelineID: "pipe-1", State: event.StateStarted}
	got, _ := EventLine(evt, LiveTUIProfile(true))
	if !strings.HasPrefix(got, "[pipe-1]") {
		t.Errorf("expected pipeline-id fallback prefix; got %q", got)
	}
}

func TestEventLine_LiveTUI_Failed_EmptyMessage(t *testing.T) {
	// LogRecord-style empty Failed message should fall back to "unknown error"
	evt := event.Event{StepID: "implement", State: event.StateFailed}
	got, _ := EventLine(evt, LiveTUIProfile(true))
	if !strings.Contains(got, "unknown error") {
		t.Errorf("expected 'unknown error' fallback; got %q", got)
	}
}

func TestEventLine_LiveTUI_StreamActivity_Truncation(t *testing.T) {
	long := strings.Repeat("a", 100)
	evt := event.Event{StepID: "s", State: event.StateStreamActivity, ToolName: "Bash", ToolTarget: long}
	got, _ := EventLine(evt, LiveTUIProfile(true))
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected truncation suffix; got %q", got)
	}
}

func TestEventLine_LiveTUI_StreamActivity_StoredFallback(t *testing.T) {
	// LogRecord adapter sets Message and leaves ToolName/Target empty.
	evt := event.Event{StepID: "s", State: event.StateStreamActivity, Message: "Read /a/b"}
	got, _ := EventLine(evt, LiveTUIProfile(true))
	want := "[s] Read /a/b"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestEventLine_LiveTUI_AllStates(t *testing.T) {
	cases := map[string]event.Event{
		"running":               {StepID: "s", State: event.StateRunning, Message: "doing work"},
		"warning":               {StepID: "s", State: "warning", Message: "be careful"},
		"contract_passed":       {StepID: "s", State: "contract_passed"},
		"contract_failed":       {StepID: "s", State: "contract_failed"},
		"contract_soft_failure": {StepID: "s", State: "contract_soft_failure"},
		"step_progress_action":  {StepID: "s", State: event.StateStepProgress, CurrentAction: "thinking"},
		"step_progress_tokens":  {StepID: "s", State: event.StateStepProgress, TokensIn: 100, TokensOut: 50},
		"step_progress_pct":     {StepID: "s", State: event.StateStepProgress, Progress: 50},
		"step_progress_beat":   {StepID: "s", State: event.StateStepProgress},
		"eta_with":              {StepID: "s", State: event.StateETAUpdated, EstimatedTimeMs: 30000},
		"eta_calc":              {StepID: "s", State: event.StateETAUpdated},
		"compaction":            {StepID: "s", State: event.StateCompactionProgress},
		"default_msg":           {StepID: "s", State: "weird", Message: "msg"},
		"default_no_msg":        {StepID: "s", State: "weird"},
	}
	for name, evt := range cases {
		t.Run(name+"_color", func(t *testing.T) {
			got, emit := EventLine(evt, LiveTUIProfile(true))
			if !emit {
				t.Fatal("emit should always be true for live profile")
			}
			if !strings.HasPrefix(got, "[s]") {
				t.Errorf("missing prefix: %q", got)
			}
		})
		t.Run(name+"_nocolor", func(t *testing.T) {
			got, _ := EventLine(evt, LiveTUIProfile(false))
			if !strings.HasPrefix(got, "[s]") {
				t.Errorf("missing prefix: %q", got)
			}
		})
	}
}

func TestEventLine_BasicCLI_PipelineWarning(t *testing.T) {
	ts := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	evt := event.Event{Timestamp: ts, State: "warning", Message: "global"}
	line, emit := EventLine(evt, BasicCLIProfile(ts.Format("15:04:05"), nil, false))
	if !emit {
		t.Fatal("expected emit=true for pipeline warning")
	}
	want := "[12:00:00] ⚠ global"
	if line != want {
		t.Errorf("got %q want %q", line, want)
	}
}

func TestEventLine_BasicCLI_PipelineLevelOtherSuppressed(t *testing.T) {
	evt := event.Event{State: "completed"} // no StepID, no warning
	_, emit := EventLine(evt, BasicCLIProfile("00:00:00", nil, false))
	if emit {
		t.Error("expected emit=false for pipeline-level non-warning")
	}
}

func TestEventLine_BasicCLI_StartedNoPersona_Suppressed(t *testing.T) {
	evt := event.Event{StepID: "s", State: "started"}
	_, emit := EventLine(evt, BasicCLIProfile("00:00:00", nil, false))
	if emit {
		t.Error("expected emit=false when persona empty")
	}
}

func TestEventLine_BasicCLI_StartedWithModel(t *testing.T) {
	evt := event.Event{StepID: "s", State: "started", Persona: "nav", Model: "opus", Adapter: "claude"}
	got, _ := EventLine(evt, BasicCLIProfile("12:00:00", nil, false))
	want := "[12:00:00] → s (nav) [opus via claude]"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestEventLine_BasicCLI_Completed(t *testing.T) {
	evt := event.Event{StepID: "s", State: "completed", DurationMs: 1500, TokensUsed: 12345}
	got, _ := EventLine(evt, BasicCLIProfile("12:00:00", nil, false))
	want := "[12:00:00] ✓ s completed (1.5s, 12.3k tokens)"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestEventLine_BasicCLI_StreamActivity_VerboseFalse_Suppressed(t *testing.T) {
	evt := event.Event{StepID: "s", State: "stream_activity", ToolName: "Read", ToolTarget: "/a/b"}
	_, emit := EventLine(evt, BasicCLIProfile("12:00:00", nil, false))
	if emit {
		t.Error("expected emit=false when verbose=false")
	}
}

func TestEventLine_BasicCLI_StreamActivity_Verbose(t *testing.T) {
	evt := event.Event{StepID: "s", State: "stream_activity", ToolName: "Read", ToolTarget: "/a/b"}
	ti := &TerminalInfo{capabilities: &TerminalCapabilities{Width: 120}}
	got, emit := EventLine(evt, BasicCLIProfile("12:00:00", ti, true))
	if !emit {
		t.Fatal("expected emit=true")
	}
	if !strings.Contains(got, "Read → /a/b") {
		t.Errorf("missing tool segment: %q", got)
	}
}

func TestEventLine_BasicCLI_DefaultSuppressed(t *testing.T) {
	evt := event.Event{StepID: "s", State: "eta_updated", EstimatedTimeMs: 1000}
	_, emit := EventLine(evt, BasicCLIProfile("00:00:00", nil, false))
	if emit {
		t.Error("expected emit=false for unhandled state")
	}
}

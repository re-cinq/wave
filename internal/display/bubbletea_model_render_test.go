package display

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/pathfmt"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// renderSteps renders the step list for ctx with ANSI styling stripped so
// assertions can compare plain text.
func renderSteps(ctx *PipelineContext) []string {
	out := ansiEscape.ReplaceAllString((&ProgressModel{ctx: ctx}).renderCurrentStep(), "")
	lines := strings.Split(out, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return lines
}

func TestRenderCurrentStep_Empty(t *testing.T) {
	got := (&ProgressModel{ctx: &PipelineContext{}}).renderCurrentStep()
	if got != "Waiting for pipeline to start..." {
		t.Errorf("empty step order: got %q", got)
	}
}

func TestRenderCurrentStep_TerminalAndPendingStates(t *testing.T) {
	ctx := &PipelineContext{
		StepOrder: []string{"done", "broke", "skip", "stop", "todo", "unknown"},
		StepStatuses: map[string]ProgressState{
			"done":  StateCompleted,
			"broke": StateFailed,
			"skip":  StateSkipped,
			"stop":  StateCancelled,
			"todo":  StateNotStarted,
		},
		StepPersonas: map[string]string{
			"done": "craftsman", "broke": "reviewer", "skip": "navigator",
			"stop": "auditor", "todo": "planner",
		},
		StepDurations: map[string]int64{"done": 1500, "broke": 250},
	}
	want := []string{
		"\u2713 done (craftsman) (1.5s)",
		"\u2717 broke (reviewer) (0.2s)",
		"\u2014 skip (navigator)",
		"\u229b stop (auditor)",
		"\u25cb todo (planner)",
		"\u25cb unknown",
	}
	assertLines(t, renderSteps(ctx), want)
}

func TestRenderCurrentStep_FailedWithoutDuration(t *testing.T) {
	ctx := &PipelineContext{
		StepOrder:    []string{"broke"},
		StepStatuses: map[string]ProgressState{"broke": StateFailed},
	}
	assertLines(t, renderSteps(ctx), []string{"\u2717 broke"})
}

func TestRenderCurrentStep_CompletedTokenVariants(t *testing.T) {
	tests := []struct {
		name string
		ctx  PipelineContext
		want string
	}{
		{
			name: "no duration map defaults to 0.0s",
			ctx:  PipelineContext{},
			want: "\u2713 s (0.0s)",
		},
		{
			name: "model shown",
			ctx: PipelineContext{
				StepModels:    map[string]string{"s": "sonnet"},
				StepDurations: map[string]int64{"s": 2000},
			},
			want: "\u2713 s [sonnet] (2.0s)",
		},
		{
			name: "split tokens preferred",
			ctx: PipelineContext{
				StepDurations: map[string]int64{"s": 1000},
				StepTokensIn:  map[string]int{"s": 1200},
				StepTokensOut: map[string]int{"s": 34},
				StepTokens:    map[string]int{"s": 9999},
			},
			want: "\u2713 s (1.0s, 1.2k in / 34 out)",
		},
		{
			name: "split map present but empty falls back to total",
			ctx: PipelineContext{
				StepDurations: map[string]int64{"s": 1000},
				StepTokensIn:  map[string]int{},
				StepTokensOut: map[string]int{},
				StepTokens:    map[string]int{"s": 5000},
			},
			want: "\u2713 s (1.0s, 5.0k tokens)",
		},
		{
			name: "split map present, total zero",
			ctx: PipelineContext{
				StepDurations: map[string]int64{"s": 1000},
				StepTokensIn:  map[string]int{},
				StepTokens:    map[string]int{"s": 0},
			},
			want: "\u2713 s (1.0s)",
		},
		{
			name: "split map present, no total map",
			ctx: PipelineContext{
				StepDurations: map[string]int64{"s": 1000},
				StepTokensIn:  map[string]int{},
			},
			want: "\u2713 s (1.0s)",
		},
		{
			name: "total tokens only",
			ctx: PipelineContext{
				StepDurations: map[string]int64{"s": 1000},
				StepTokens:    map[string]int{"s": 42},
			},
			want: "\u2713 s (1.0s, 42 tokens)",
		},
		{
			name: "total tokens map without entry",
			ctx: PipelineContext{
				StepDurations: map[string]int64{"s": 1000},
				StepTokens:    map[string]int{},
			},
			want: "\u2713 s (1.0s)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			ctx.StepOrder = []string{"s"}
			ctx.StepStatuses = map[string]ProgressState{"s": StateCompleted}
			assertLines(t, renderSteps(&ctx), []string{tt.want})
		})
	}
}

func TestRenderCurrentStep_CompletedMetadataTree(t *testing.T) {
	base := func(verbose bool) *PipelineContext {
		return &PipelineContext{
			Verbose:      verbose,
			StepOrder:    []string{"s"},
			StepStatuses: map[string]ProgressState{"s": StateCompleted},
			DeliverablesByStep: map[string][]string{
				"s": {"pr: https://example.com/pr/1"},
			},
			HandoversByStep: map[string]*HandoverInfo{
				"s": {
					ArtifactPaths:  []string{"/tmp/out.json"},
					ContractStatus: "soft_failure",
					TargetStep:     "next",
				},
			},
		}
	}

	assertLines(t, renderSteps(base(false)), []string{
		"\u2713 s (0.0s)",
		"   \u2514\u2500 pr: https://example.com/pr/1",
	})

	assertLines(t, renderSteps(base(true)), []string{
		"\u2713 s (0.0s)",
		"   \u251c\u2500 pr: https://example.com/pr/1",
		"   \u251c\u2500 artifact: " + pathfmt.FileURI("/tmp/out.json") + " (written)",
		"   \u251c\u2500 contract: contract \u26a0 soft failure",
		"   \u2514\u2500 handover \u2192 next",
	})

	for status, want := range map[string]string{
		"passed": "contract: json_schema \u2713 valid",
		"failed": "contract: json_schema \u2717 failed",
	} {
		ctx := base(true)
		ctx.DeliverablesByStep = nil
		ctx.HandoversByStep["s"] = &HandoverInfo{ContractStatus: status, ContractSchema: "json_schema"}
		assertLines(t, renderSteps(ctx), []string{"\u2713 s (0.0s)", "   \u2514\u2500 " + want})
	}
}

func TestRenderCurrentStep_Running(t *testing.T) {
	now := time.Now()
	ctx := &PipelineContext{
		StepOrder:        []string{"a", "b", "c"},
		StepStatuses:     map[string]ProgressState{"a": StateRunning, "b": StateRunning, "c": StateRunning},
		StepPersonas:     map[string]string{"a": "craftsman"},
		StepModels:       map[string]string{"a": "opus", "b": "haiku"},
		StepAdapters:     map[string]string{"a": "claude"},
		StepStartTimes:   map[string]int64{"a": now.Add(-90 * time.Second).UnixNano()},
		CurrentStepStart: now.Add(-5 * time.Second).UnixNano(),
		CurrentStepID:    "b",
		CurrentAction:    "thinking",
		StepToolActivity: map[string][2]string{"a": {"Read", "main.go"}},
		LastToolName:     "Bash",
		LastToolTarget:   strings.Repeat("x", 500),
	}
	lines := renderSteps(ctx)
	if len(lines) != 5 {
		t.Fatalf("want 5 lines (3 steps, 2 tool lines), got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}

	assertRunningLine(t, lines[0], "a (craftsman) [opus via claude] (1m 30s)")
	if lines[1] != "   Read \u2192 main.go" {
		t.Errorf("per-step tool line: got %q", lines[1])
	}

	// b is the current step: falls back to CurrentStepStart, shows the action
	// and the global tool activity, truncated to the terminal width.
	assertRunningLine(t, lines[2], "b [haiku] (5s) \u2022 thinking")
	if !strings.HasPrefix(lines[3], "   Bash \u2192 xxx") || !strings.HasSuffix(lines[3], "...") {
		t.Errorf("global tool line should be truncated: got %q", lines[3])
	}
	if len(lines[3]) >= 500 {
		t.Errorf("global tool line not truncated (len %d)", len(lines[3]))
	}

	// c is neither current nor has tool activity: no action, no tool line.
	if !strings.HasSuffix(lines[4], " c (5s)") {
		t.Errorf("running step without start time: got %q", lines[4])
	}
}

func assertRunningLine(t *testing.T, got, wantSuffix string) {
	t.Helper()
	icon, rest, ok := strings.Cut(got, " ")
	spinnerFrames := "\u280b\u2819\u2839\u2838\u283c\u2834\u2826\u2827\u2807\u280f"
	if !ok || len([]rune(icon)) != 1 || !strings.ContainsRune(spinnerFrames, []rune(icon)[0]) {
		t.Errorf("running line should start with a spinner frame: %q", got)
	}
	if rest != wantSuffix {
		t.Errorf("running line: got %q, want %q", rest, wantSuffix)
	}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rendered lines mismatch\n got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

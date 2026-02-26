package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		current  int
		width    int
		wantLen  int // Approximate length (may vary with unicode)
	}{
		{
			name:    "empty progress",
			total:   100,
			current: 0,
			width:   20,
			wantLen: 20, // Just the bar itself
		},
		{
			name:    "half progress",
			total:   100,
			current: 50,
			width:   20,
			wantLen: 20,
		},
		{
			name:    "full progress",
			total:   100,
			current: 100,
			width:   20,
			wantLen: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(tt.total, tt.width)
			pb.SetProgress(tt.current)
			result := pb.Render()

			// Verify we got some output
			if len(result) == 0 {
				t.Error("Expected non-empty progress bar")
			}

			// Verify it contains brackets
			if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
				t.Errorf("Expected progress bar to contain brackets, got: %s", result)
			}

			// Verify percentage appears
			if !strings.Contains(result, "%") {
				t.Errorf("Expected progress bar to contain percentage, got: %s", result)
			}
		})
	}
}

func TestStepStatus(t *testing.T) {
	tests := []struct {
		name       string
		state      ProgressState
		wantIcon   bool
		wantStatus bool
	}{
		{
			name:       "not started",
			state:      StateNotStarted,
			wantIcon:   true,
			wantStatus: true,
		},
		{
			name:       "running",
			state:      StateRunning,
			wantIcon:   true,
			wantStatus: true,
		},
		{
			name:       "completed",
			state:      StateCompleted,
			wantIcon:   true,
			wantStatus: true,
		},
		{
			name:       "failed",
			state:      StateFailed,
			wantIcon:   true,
			wantStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := NewStepStatus("test-step", "Test Step", "test-persona")
			ss.UpdateState(tt.state)

			result := ss.Render()

			// Verify we got some output
			if len(result) == 0 {
				t.Error("Expected non-empty step status")
			}

			// Verify step name appears
			if !strings.Contains(result, "Test Step") {
				t.Errorf("Expected step status to contain step name, got: %s", result)
			}
		})
	}
}

func TestStepStatusStateTransitions(t *testing.T) {
	ss := NewStepStatus("test-step", "Test Step", "test-persona")

	// Initial state
	if ss.State != StateNotStarted {
		t.Errorf("Expected initial state to be NotStarted, got %v", ss.State)
	}

	// Transition to running
	ss.UpdateState(StateRunning)
	if ss.State != StateRunning {
		t.Errorf("Expected state to be Running, got %v", ss.State)
	}

	// Check that StartTime is set
	if ss.StartTime.IsZero() {
		t.Error("Expected StartTime to be set when transitioning to Running")
	}

	// Transition to completed
	time.Sleep(10 * time.Millisecond) // Small delay to ensure elapsed time
	ss.UpdateState(StateCompleted)
	if ss.State != StateCompleted {
		t.Errorf("Expected state to be Completed, got %v", ss.State)
	}

	// Check that EndTime is set
	if ss.EndTime == nil {
		t.Error("Expected EndTime to be set when transitioning to Completed")
	}

	// Check that ElapsedMs is calculated
	if ss.ElapsedMs <= 0 {
		t.Errorf("Expected ElapsedMs to be positive, got %d", ss.ElapsedMs)
	}
}

func TestProgressDisplay(t *testing.T) {
	pd := NewProgressDisplay("test-pipeline", "Test Pipeline", 3)

	// Add steps
	pd.AddStep("step1", "Step 1", "persona1")
	pd.AddStep("step2", "Step 2", "persona2")
	pd.AddStep("step3", "Step 3", "persona3")

	// Verify steps are registered
	if len(pd.steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(pd.steps))
	}

	// Update step state
	pd.UpdateStep("step1", StateRunning, "Executing", 50)
	if pd.steps["step1"].State != StateRunning {
		t.Errorf("Expected step1 to be Running, got %v", pd.steps["step1"].State)
	}

	// Complete step
	pd.UpdateStep("step1", StateCompleted, "Done", 100)
	if pd.steps["step1"].State != StateCompleted {
		t.Errorf("Expected step1 to be Completed, got %v", pd.steps["step1"].State)
	}

	// Verify overall progress bar is updated
	if pd.overallBar.current != 1 {
		t.Errorf("Expected overall progress to be 1, got %d", pd.overallBar.current)
	}
}

func TestBasicProgressDisplay(t *testing.T) {
	bpd := NewBasicProgressDisplay()

	// Create test events
	events := []event.Event{
		{
			Timestamp: time.Now(),
			PipelineID: "test-pipeline",
			StepID:     "step1",
			State:      "started",
			Persona:    "test-persona",
		},
		{
			Timestamp:  time.Now(),
			PipelineID: "test-pipeline",
			StepID:     "step1",
			State:      "completed",
			DurationMs: 1500,
			TokensUsed: 5000,
		},
	}

	// Should not panic
	for _, ev := range events {
		if err := bpd.EmitProgress(ev); err != nil {
			t.Errorf("EmitProgress failed: %v", err)
		}
	}
}

func TestSpinner(t *testing.T) {
	spinner := NewSpinner(AnimationSpinner)

	// Initial state
	if spinner.IsRunning() {
		t.Error("Expected spinner to not be running initially")
	}

	// Start spinner
	spinner.Start()
	if !spinner.IsRunning() {
		t.Error("Expected spinner to be running after Start()")
	}

	// Get current frame
	frame := spinner.Current()
	if frame == "" {
		t.Error("Expected non-empty spinner frame")
	}

	// Stop spinner
	spinner.Stop()
	if spinner.IsRunning() {
		t.Error("Expected spinner to not be running after Stop()")
	}
}

func TestProgressAnimation(t *testing.T) {
	pa := NewProgressAnimation("Loading", 100, 20)

	// Start animation
	pa.Start()
	if !pa.spinner.IsRunning() {
		t.Error("Expected spinner to be running")
	}

	// Update progress
	pa.SetProgress(50)
	if pa.progressBar.current != 50 {
		t.Errorf("Expected progress to be 50, got %d", pa.progressBar.current)
	}

	// Render
	result := pa.Render()
	if !strings.Contains(result, "Loading") {
		t.Errorf("Expected animation to contain label, got: %s", result)
	}

	// Stop animation
	pa.Stop()
	if pa.spinner.IsRunning() {
		t.Error("Expected spinner to not be running after Stop()")
	}
}

func TestFormatStepDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			want:     "500ms",
		},
		{
			name:     "seconds",
			duration: 5 * time.Second,
			want:     "5s",
		},
		{
			name:     "minutes and seconds",
			duration: 90 * time.Second,
			want:     "1m 30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStepDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatStepDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestBubbleTeaToPipelineContext_StepPersonas(t *testing.T) {
	// We can't easily test BubbleTeaProgressDisplay.toPipelineContext() directly
	// because it requires a TTY. Instead, test that ProgressDisplay.toPipelineContext()
	// populates StepPersonas correctly.
	pd := NewProgressDisplay("test-pipeline", "Test Pipeline", 2)
	pd.AddStep("s1", "step-1", "navigator")
	pd.AddStep("s2", "step-2", "implementer")

	pd.mu.Lock()
	ctx := pd.toPipelineContext()
	pd.mu.Unlock()

	if ctx.StepPersonas == nil {
		t.Fatal("StepPersonas should not be nil")
	}
	if ctx.StepPersonas["s1"] != "navigator" {
		t.Errorf("StepPersonas[s1] = %q, want %q", ctx.StepPersonas["s1"], "navigator")
	}
	if ctx.StepPersonas["s2"] != "implementer" {
		t.Errorf("StepPersonas[s2] = %q, want %q", ctx.StepPersonas["s2"], "implementer")
	}
}

func TestProgressDisplay_ToPipelineContext_StepOrder(t *testing.T) {
	pd := NewProgressDisplay("test-pipeline", "Test Pipeline", 3)
	pd.AddStep("alpha", "Alpha Step", "p1")
	pd.AddStep("beta", "Beta Step", "p2")
	pd.AddStep("gamma", "Gamma Step", "p3")

	pd.mu.Lock()
	ctx := pd.toPipelineContext()
	pd.mu.Unlock()

	if len(ctx.StepOrder) != 3 {
		t.Fatalf("StepOrder length = %d, want 3", len(ctx.StepOrder))
	}
	if ctx.StepOrder[0] != "alpha" || ctx.StepOrder[1] != "beta" || ctx.StepOrder[2] != "gamma" {
		t.Errorf("StepOrder = %v, want [alpha, beta, gamma]", ctx.StepOrder)
	}
}

func TestCreatePipelineContext_WithPersonas(t *testing.T) {
	personas := map[string]string{
		"step-1": "navigator",
		"step-2": "implementer",
	}
	ctx := CreatePipelineContext("wave.yaml", "test", "/tmp", 2, []string{"step-1", "step-2"}, personas)

	if ctx.StepPersonas == nil {
		t.Fatal("StepPersonas should not be nil")
	}
	if ctx.StepPersonas["step-1"] != "navigator" {
		t.Errorf("StepPersonas[step-1] = %q, want %q", ctx.StepPersonas["step-1"], "navigator")
	}
	if ctx.StepPersonas["step-2"] != "implementer" {
		t.Errorf("StepPersonas[step-2] = %q, want %q", ctx.StepPersonas["step-2"], "implementer")
	}

	// Also verify StepOrder is populated
	if len(ctx.StepOrder) != 2 {
		t.Fatalf("StepOrder length = %d, want 2", len(ctx.StepOrder))
	}
	if ctx.StepOrder[0] != "step-1" || ctx.StepOrder[1] != "step-2" {
		t.Errorf("StepOrder = %v, want [step-1, step-2]", ctx.StepOrder)
	}
}

func TestBasicProgressDisplay_HandoverMetadata_VerboseMode(t *testing.T) {
	var buf bytes.Buffer
	bpd := NewBasicProgressDisplayWithVerbose(true)
	bpd.writer = &buf

	now := time.Now()

	// Emit a started event (to track step order)
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "started",
		Persona:    "analyst",
	})

	// Emit a second step started (to track as next step)
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "implementer",
		State:      "started",
		Persona:    "implementer",
	})

	// Emit validating event (to capture contract schema)
	bpd.EmitProgress(event.Event{
		Timestamp:       now,
		PipelineID:      "test-pipeline",
		StepID:          "analyst",
		State:           "validating",
		ValidationPhase: "json_schema",
	})

	// Emit contract_passed event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "contract_passed",
	})

	// Clear buffer before the completed event (so we only capture the completed output)
	buf.Reset()

	// Emit completed event with artifacts
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "completed",
		DurationMs: 45200,
		TokensUsed: 5000,
		Artifacts:  []string{".wave/artifacts/analysis"},
	})

	output := buf.String()

	// Should contain the completed line
	if !strings.Contains(output, "analyst completed") {
		t.Errorf("Output should contain completed line, got:\n%s", output)
	}

	// Should contain artifact line
	if !strings.Contains(output, "artifact: .wave/artifacts/analysis (written)") {
		t.Errorf("Output should contain artifact path in verbose mode, got:\n%s", output)
	}

	// Should contain contract line
	if !strings.Contains(output, "contract: json_schema") {
		t.Errorf("Output should contain contract schema in verbose mode, got:\n%s", output)
	}

	// Should contain handover target
	if !strings.Contains(output, "handover") || !strings.Contains(output, "implementer") {
		t.Errorf("Output should contain handover target in verbose mode, got:\n%s", output)
	}

	// Should contain tree connectors
	if !strings.Contains(output, "├─") {
		t.Errorf("Output should contain ├─ connector, got:\n%s", output)
	}
	if !strings.Contains(output, "└─") {
		t.Errorf("Output should contain └─ connector, got:\n%s", output)
	}
}

func TestBasicProgressDisplay_HandoverMetadata_NonVerboseMode(t *testing.T) {
	var buf bytes.Buffer
	bpd := NewBasicProgressDisplay() // non-verbose
	bpd.writer = &buf

	now := time.Now()

	// Emit started event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "started",
		Persona:    "analyst",
	})

	// Emit validating event
	bpd.EmitProgress(event.Event{
		Timestamp:       now,
		PipelineID:      "test-pipeline",
		StepID:          "analyst",
		State:           "validating",
		ValidationPhase: "json_schema",
	})

	// Emit contract_passed event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "contract_passed",
	})

	buf.Reset()

	// Emit completed event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "completed",
		DurationMs: 30000,
		TokensUsed: 3000,
		Artifacts:  []string{".wave/artifacts/analysis"},
	})

	output := buf.String()

	// Should contain the completed line
	if !strings.Contains(output, "analyst completed") {
		t.Errorf("Output should contain completed line, got:\n%s", output)
	}

	// Should NOT contain handover metadata
	if strings.Contains(output, "artifact:") {
		t.Errorf("Non-verbose output should NOT contain artifact metadata, got:\n%s", output)
	}
	if strings.Contains(output, "contract:") {
		t.Errorf("Non-verbose output should NOT contain contract metadata, got:\n%s", output)
	}
	if strings.Contains(output, "handover") {
		t.Errorf("Non-verbose output should NOT contain handover metadata, got:\n%s", output)
	}
}

func TestBasicProgressDisplay_HandoverMetadata_FailedContract(t *testing.T) {
	var buf bytes.Buffer
	bpd := NewBasicProgressDisplayWithVerbose(true)
	bpd.writer = &buf

	now := time.Now()

	// Emit started event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "started",
		Persona:    "analyst",
	})

	// Emit validating event
	bpd.EmitProgress(event.Event{
		Timestamp:       now,
		PipelineID:      "test-pipeline",
		StepID:          "analyst",
		State:           "validating",
		ValidationPhase: "json_schema",
	})

	// Emit contract_failed event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "contract_failed",
		Message:    "schema validation error",
	})

	buf.Reset()

	// Emit completed event
	bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "completed",
		DurationMs: 30000,
		TokensUsed: 3000,
	})

	output := buf.String()

	// Should contain failed contract status
	if !strings.Contains(output, "failed") {
		t.Errorf("Output should contain 'failed' for failed contract, got:\n%s", output)
	}
}

func TestBasicProgressDisplay_BuildHandoverLines(t *testing.T) {
	bpd := NewBasicProgressDisplayWithVerbose(true)
	bpd.stepOrder = []string{"step1", "step2", "step3"}

	tests := []struct {
		name      string
		stepID    string
		info      *HandoverInfo
		wantLines int
		wantLast  string // substring expected in last line
	}{
		{
			name:   "all metadata present",
			stepID: "step1",
			info: &HandoverInfo{
				ArtifactPaths:  []string{".wave/artifacts/analysis"},
				ContractStatus: "passed",
				ContractSchema: "json_schema",
				TargetStep:     "",
			},
			wantLines: 3, // artifact + contract + handover (target derived from stepOrder)
			wantLast:  "└─",
		},
		{
			name:   "only artifacts",
			stepID: "step3", // last step, no handover target
			info: &HandoverInfo{
				ArtifactPaths: []string{".wave/artifacts/review"},
			},
			wantLines: 1,
			wantLast:  "└─",
		},
		{
			name:   "multiple artifacts",
			stepID: "step1",
			info: &HandoverInfo{
				ArtifactPaths:  []string{".wave/artifacts/a", ".wave/artifacts/b"},
				ContractStatus: "passed",
				ContractSchema: "json_schema",
			},
			wantLines: 4, // 2 artifacts + contract + handover target
			wantLast:  "└─",
		},
		{
			name:   "empty info",
			stepID: "step1",
			info:   &HandoverInfo{},
			wantLines: 1, // only handover target (derived from step order)
			wantLast:  "└─",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := bpd.buildHandoverLines(tt.stepID, tt.info)
			if len(lines) != tt.wantLines {
				t.Errorf("buildHandoverLines() returned %d lines, want %d; lines: %v", len(lines), tt.wantLines, lines)
			}
			if len(lines) > 0 {
				lastLine := lines[len(lines)-1]
				if !strings.Contains(lastLine, tt.wantLast) {
					t.Errorf("last line should contain %q, got: %s", tt.wantLast, lastLine)
				}
			}
		})
	}
}

func TestBasicProgressDisplay_HandoverLineFormat(t *testing.T) {
	bpd := NewBasicProgressDisplayWithVerbose(true)
	bpd.stepOrder = []string{"analyst", "implementer", "reviewer"}

	tests := []struct {
		name           string
		stepID         string
		targetStep     string
		wantHandover   string
	}{
		{
			name:         "explicit target step",
			stepID:       "analyst",
			targetStep:   "implementer",
			wantHandover: "handover → step 2: implementer",
		},
		{
			name:         "derived target from step order",
			stepID:       "implementer",
			targetStep:   "", // should derive "reviewer" as step 3
			wantHandover: "handover → step 3: reviewer",
		},
		{
			name:         "first to second step",
			stepID:       "analyst",
			targetStep:   "", // should derive "implementer" as step 2
			wantHandover: "handover → step 2: implementer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &HandoverInfo{
				TargetStep: tt.targetStep,
			}
			lines := bpd.buildHandoverLines(tt.stepID, info)
			
			// Should have exactly one line (the handover line)
			if len(lines) != 1 {
				t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
			}
			
			// Check if the handover line contains the expected format
			handoverLine := lines[0]
			if !strings.Contains(handoverLine, tt.wantHandover) {
				t.Errorf("handover line should contain %q, got: %s", tt.wantHandover, handoverLine)
			}
			
			// Verify it has the tree connector
			if !strings.HasPrefix(handoverLine, "└─") {
				t.Errorf("handover line should start with tree connector, got: %s", handoverLine)
			}
		})
	}
}

func TestBasicProgressDisplay_StreamActivityGuard(t *testing.T) {
	tests := []struct {
		name       string
		stepState  string // pre-set state in stepStates map ("running", "completed", or "" for not-started)
		wantOutput bool
	}{
		{
			name:       "running step produces output",
			stepState:  "running",
			wantOutput: true,
		},
		{
			name:       "completed step produces no output",
			stepState:  "completed",
			wantOutput: false,
		},
		{
			name:       "not-started step produces no output",
			stepState:  "",
			wantOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			bpd := NewBasicProgressDisplayWithVerbose(true)
			bpd.writer = &buf

			// Pre-set step state
			if tt.stepState != "" {
				bpd.stepStates["test-step"] = tt.stepState
			}

			// Emit a stream_activity event
			err := bpd.EmitProgress(event.Event{
				Timestamp:  time.Now(),
				PipelineID: "test-pipeline",
				StepID:     "test-step",
				State:      "stream_activity",
				ToolName:   "Read",
				ToolTarget: "main.go",
			})
			if err != nil {
				t.Fatalf("EmitProgress failed: %v", err)
			}

			output := buf.String()
			if tt.wantOutput && output == "" {
				t.Error("expected output for stream_activity on running step, got none")
			}
			if !tt.wantOutput && output != "" {
				t.Errorf("expected no output for stream_activity on %s step, got: %s",
					tt.stepState, output)
			}
		})
	}
}

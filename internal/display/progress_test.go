package display

import (
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
			want:     "5.0s",
		},
		{
			name:     "minutes and seconds",
			duration: 90 * time.Second,
			want:     "1m30s",
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

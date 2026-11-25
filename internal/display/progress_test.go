package display

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func TestBasicProgressDisplay(t *testing.T) {
	bpd := NewBasicProgressDisplayWithVerbose(false)

	// Create test events
	events := []event.Event{
		{
			Timestamp:  time.Now(),
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

func TestBasicProgressDisplay_HandoverMetadata_VerboseMode(t *testing.T) {
	var buf bytes.Buffer
	bpd := NewBasicProgressDisplayWithVerbose(true)
	bpd.writer = &buf

	now := time.Now()

	// Emit a started event (to track step order)
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "started",
		Persona:    "analyst",
	})

	// Emit a second step started (to track as next step)
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "implementer",
		State:      "started",
		Persona:    "implementer",
	})

	// Emit validating event (to capture contract schema)
	_ = bpd.EmitProgress(event.Event{
		Timestamp:       now,
		PipelineID:      "test-pipeline",
		StepID:          "analyst",
		State:           "validating",
		ValidationPhase: "json_schema",
	})

	// Emit contract_passed event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "contract_passed",
	})

	// Clear buffer before the completed event (so we only capture the completed output)
	buf.Reset()

	// Emit completed event with artifacts
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "completed",
		DurationMs: 45200,
		TokensUsed: 5000,
		Artifacts:  []string{".agents/artifacts/analysis"},
	})

	output := buf.String()

	// Should contain the completed line
	if !strings.Contains(output, "analyst completed") {
		t.Errorf("Output should contain completed line, got:\n%s", output)
	}

	// Should contain artifact line
	if !strings.Contains(output, "artifact: .agents/artifacts/analysis (written)") {
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
	bpd := NewBasicProgressDisplayWithVerbose(false) // non-verbose
	bpd.writer = &buf

	now := time.Now()

	// Emit started event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "started",
		Persona:    "analyst",
	})

	// Emit validating event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:       now,
		PipelineID:      "test-pipeline",
		StepID:          "analyst",
		State:           "validating",
		ValidationPhase: "json_schema",
	})

	// Emit contract_passed event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "contract_passed",
	})

	buf.Reset()

	// Emit completed event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "completed",
		DurationMs: 30000,
		TokensUsed: 3000,
		Artifacts:  []string{".agents/artifacts/analysis"},
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
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "started",
		Persona:    "analyst",
	})

	// Emit validating event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:       now,
		PipelineID:      "test-pipeline",
		StepID:          "analyst",
		State:           "validating",
		ValidationPhase: "json_schema",
	})

	// Emit contract_failed event
	_ = bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "analyst",
		State:      "contract_failed",
		Message:    "schema validation error",
	})

	buf.Reset()

	// Emit completed event
	_ = bpd.EmitProgress(event.Event{
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
				ArtifactPaths:  []string{".agents/artifacts/analysis"},
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
				ArtifactPaths: []string{".agents/artifacts/review"},
			},
			wantLines: 1,
			wantLast:  "└─",
		},
		{
			name:   "multiple artifacts",
			stepID: "step1",
			info: &HandoverInfo{
				ArtifactPaths:  []string{".agents/artifacts/a", ".agents/artifacts/b"},
				ContractStatus: "passed",
				ContractSchema: "json_schema",
			},
			wantLines: 4, // 2 artifacts + contract + handover target
			wantLast:  "└─",
		},
		{
			name:      "empty info",
			stepID:    "step1",
			info:      &HandoverInfo{},
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
		name         string
		stepID       string
		targetStep   string
		wantHandover string
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

func TestBasicProgressDisplay_RetryingEvent(t *testing.T) {
	var buf bytes.Buffer
	bpd := NewBasicProgressDisplayWithVerbose(false)
	bpd.writer = &buf

	now := time.Now()

	// Emit a retrying event
	err := bpd.EmitProgress(event.Event{
		Timestamp:  now,
		PipelineID: "test-pipeline",
		StepID:     "implement",
		State:      "retrying",
		Message:    "attempt 2/3",
	})
	if err != nil {
		t.Fatalf("EmitProgress failed: %v", err)
	}

	output := buf.String()

	// Should contain retry indicator and message
	if !strings.Contains(output, "implement") {
		t.Errorf("Output should contain step ID, got:\n%s", output)
	}
	if !strings.Contains(output, "retrying") {
		t.Errorf("Output should contain 'retrying', got:\n%s", output)
	}
	if !strings.Contains(output, "attempt 2/3") {
		t.Errorf("Output should contain attempt message, got:\n%s", output)
	}

	// Should update step state to running
	if bpd.stepStates["implement"] != "running" {
		t.Errorf("Step state should be 'running' after retrying, got %q", bpd.stepStates["implement"])
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

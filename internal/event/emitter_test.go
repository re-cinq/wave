package event

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestEmitter(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		validate func(*testing.T, string, Event)
	}{
		{
			name: "basic event",
			event: Event{
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID: "pipeline-001",
				StepID:     "step-001",
				State:      "started",
				DurationMs: 0,
				Message:    "Pipeline started",
			},
			validate: func(t *testing.T, output string, event Event) {
				if !strings.Contains(output, `"pipeline_id":"pipeline-001"`) {
					t.Errorf("output missing pipeline_id")
				}
				if !strings.Contains(output, `"step_id":"step-001"`) {
					t.Errorf("output missing step_id")
				}
				if !strings.Contains(output, `"state":"started"`) {
					t.Errorf("output missing state")
				}
			},
		},
		{
			name: "event with duration",
			event: Event{
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID: "pipeline-002",
				StepID:     "step-002",
				State:      "completed",
				DurationMs: 1234,
				Message:    "Step completed",
			},
			validate: func(t *testing.T, output string, event Event) {
				if !strings.Contains(output, `"duration_ms":1234`) {
					t.Errorf("output missing duration_ms")
				}
				if !strings.Contains(output, `"state":"completed"`) {
					t.Errorf("output missing state")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				emitter := NewNDJSONEmitter()
				emitter.Emit(tt.event)
			})

			var decoded Event
			lines := strings.Split(strings.TrimSpace(output), "\n")
			if len(lines) == 0 {
				t.Fatal("no output")
			}

			if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
				t.Fatalf("failed to decode NDJSON: %v", err)
			}

			if decoded.PipelineID != tt.event.PipelineID {
				t.Errorf("PipelineID = %v, want %v", decoded.PipelineID, tt.event.PipelineID)
			}
			if decoded.StepID != tt.event.StepID {
				t.Errorf("StepID = %v, want %v", decoded.StepID, tt.event.StepID)
			}
			if decoded.State != tt.event.State {
				t.Errorf("State = %v, want %v", decoded.State, tt.event.State)
			}
			if decoded.DurationMs != tt.event.DurationMs {
				t.Errorf("DurationMs = %v, want %v", decoded.DurationMs, tt.event.DurationMs)
			}
			if decoded.Message != tt.event.Message {
				t.Errorf("Message = %v, want %v", decoded.Message, tt.event.Message)
			}

			if tt.validate != nil {
				tt.validate(t, output, tt.event)
			}
		})
	}
}

func TestNDJSONFormat(t *testing.T) {
	output := captureOutput(func() {
		emitter := NewNDJSONEmitter()
		emitter.Emit(Event{
			Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			PipelineID: "test-pipeline",
			StepID:     "test-step",
			State:      "running",
			DurationMs: 100,
			Message:    "Test message",
		})
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, ok := parsed["timestamp"]; !ok {
		t.Error("missing timestamp field")
	}
	if _, ok := parsed["pipeline_id"]; !ok {
		t.Error("missing pipeline_id field")
	}
	if _, ok := parsed["step_id"]; !ok {
		t.Error("missing step_id field")
	}
	if _, ok := parsed["state"]; !ok {
		t.Error("missing state field")
	}
	if _, ok := parsed["duration_ms"]; !ok {
		t.Error("missing duration_ms field")
	}
	if _, ok := parsed["message"]; !ok {
		t.Error("missing message field")
	}
}

func TestHumanReadableFormat(t *testing.T) {
	output := captureOutput(func() {
		emitter := NewNDJSONEmitterWithHumanReadable()
		emitter.Emit(Event{
			Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			PipelineID: "test-pipeline",
			StepID:     "test-step",
			State:      "running",
			DurationMs: 100,
			Message:    "Test message",
		})
	})

	if !strings.Contains(output, "[2024-01-01 12:00:00]") {
		t.Error("missing formatted timestamp")
	}
	if !strings.Contains(output, "Pipeline:test-pipeline") {
		t.Error("missing pipeline ID")
	}
	if !strings.Contains(output, "Step:test-step") {
		t.Error("missing step ID")
	}
	if !strings.Contains(output, "State:running") {
		t.Error("missing state")
	}
	if !strings.Contains(output, "Duration:100ms") {
		t.Error("missing duration")
	}
	if !strings.Contains(output, "Test message") {
		t.Error("missing message")
	}
}

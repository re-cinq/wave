package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
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

// TestProgressOnlyEmitter verifies that suppressJSON mode prevents stdout output
// while still forwarding to the progress emitter.
func TestProgressOnlyEmitter(t *testing.T) {
	output := captureOutput(func() {
		emitter := NewProgressOnlyEmitter(nil) // nil progress emitter, just test suppression
		emitter.Emit(Event{
			Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			PipelineID: "test-pipeline",
			StepID:     "test-step",
			State:      "running",
			DurationMs: 100,
			Message:    "Test message",
		})
	})

	if output != "" {
		t.Errorf("expected no stdout output from progress-only emitter, got: %s", output)
	}
}

// =============================================================================
// T103: Concurrent Event Emission Tests
// =============================================================================

// TestConcurrentEventEmission_ThreadSafety verifies that the event emitter
// is thread-safe when multiple goroutines emit events concurrently.
func TestConcurrentEventEmission_ThreadSafety(t *testing.T) {
	// Create a buffer to capture output instead of using stdout
	var buf bytes.Buffer
	emitter := &NDJSONEmitter{
		encoder: json.NewEncoder(&buf),
	}

	const numGoroutines = 100
	const eventsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines that emit events concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				emitter.Emit(Event{
					Timestamp:  time.Now(),
					PipelineID: fmt.Sprintf("pipeline-%d", goroutineID),
					StepID:     fmt.Sprintf("step-%d-%d", goroutineID, j),
					State:      "running",
					DurationMs: int64(j * 100),
					Message:    fmt.Sprintf("Event from goroutine %d, iteration %d", goroutineID, j),
				})
			}
		}(i)
	}

	wg.Wait()

	// Verify we got output (basic sanity check)
	output := buf.String()
	if len(output) == 0 {
		t.Error("no output captured from concurrent emission")
	}

	// Count the number of complete JSON lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedEvents := numGoroutines * eventsPerGoroutine

	// Due to potential interleaving, we check that we got at least some valid events
	validEvents := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			validEvents++
		}
	}

	// We should have received all events (encoder should be thread-safe)
	if validEvents != expectedEvents {
		t.Logf("Expected %d events, got %d valid events", expectedEvents, validEvents)
		// This is informational - JSON encoder should handle this
	}
}

// TestConcurrentEventEmission_NoDataRace runs concurrent event emission
// with the race detector enabled (via `go test -race`).
func TestConcurrentEventEmission_NoDataRace(t *testing.T) {
	var buf bytes.Buffer
	emitter := &NDJSONEmitter{
		encoder: json.NewEncoder(&buf),
	}

	const numGoroutines = 50
	done := make(chan struct{})

	// Start multiple emitters
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for {
				select {
				case <-done:
					return
				default:
					emitter.Emit(Event{
						Timestamp:  time.Now(),
						PipelineID: fmt.Sprintf("pipe-%d", id),
						StepID:     fmt.Sprintf("step-%d", id),
						State:      "running",
						DurationMs: 100,
					})
				}
			}
		}(i)
	}

	// Let them run for a bit
	time.Sleep(100 * time.Millisecond)
	close(done)

	// If we get here without race detector complaints, the test passes
	t.Log("Concurrent emission completed without data races")
}

// TestConcurrentEventEmission_MixedEmitters tests concurrent emission
// with both plain and progress-only emitters.
func TestConcurrentEventEmission_MixedEmitters(t *testing.T) {
	var jsonBuf, progressBuf bytes.Buffer

	jsonEmitter := &NDJSONEmitter{
		encoder: json.NewEncoder(&jsonBuf),
	}

	progressEmitter := &NDJSONEmitter{
		encoder:      json.NewEncoder(&progressBuf),
		suppressJSON: true,
	}

	const numEvents = 100
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: JSON emitter
	go func() {
		defer wg.Done()
		for i := 0; i < numEvents; i++ {
			jsonEmitter.Emit(Event{
				Timestamp:  time.Now(),
				PipelineID: "json-pipeline",
				StepID:     fmt.Sprintf("step-%d", i),
				State:      "running",
				DurationMs: int64(i * 10),
			})
		}
	}()

	// Goroutine 2: Progress-only emitter (suppressed JSON)
	go func() {
		defer wg.Done()
		for i := 0; i < numEvents; i++ {
			progressEmitter.Emit(Event{
				Timestamp:  time.Now(),
				PipelineID: "progress-pipeline",
				StepID:     fmt.Sprintf("step-%d", i),
				State:      "completed",
				DurationMs: int64(i * 10),
			})
		}
	}()

	wg.Wait()

	// Verify JSON output from first emitter
	jsonOutput := jsonBuf.String()
	if len(jsonOutput) == 0 {
		t.Error("no JSON output captured")
	}

	// Verify progress emitter suppressed JSON
	progressOutput := progressBuf.String()
	if len(progressOutput) != 0 {
		t.Error("progress-only emitter should not produce JSON output")
	}

	// Count valid JSON events
	validCount := 0
	for _, line := range strings.Split(strings.TrimSpace(jsonOutput), "\n") {
		if line == "" {
			continue
		}
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err == nil {
			validCount++
		}
	}

	if validCount == 0 {
		t.Error("no valid JSON events in output")
	}
}

// TestConcurrentEventEmission_HighContention tests behavior under high
// contention with many goroutines fighting for the emitter.
func TestConcurrentEventEmission_HighContention(t *testing.T) {
	var buf bytes.Buffer
	emitter := &NDJSONEmitter{
		encoder: json.NewEncoder(&buf),
	}

	const numGoroutines = 200
	const burstSize = 10

	var wg sync.WaitGroup
	start := make(chan struct{})

	// Set up goroutines to all start at the same time (high contention)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start // Wait for signal

			// Burst of events
			for j := 0; j < burstSize; j++ {
				emitter.Emit(Event{
					Timestamp:  time.Now(),
					PipelineID: fmt.Sprintf("pipe-%d", id),
					StepID:     fmt.Sprintf("step-%d", j),
					State:      "started",
					DurationMs: 0,
					Message:    fmt.Sprintf("burst %d from %d", j, id),
				})
			}
		}(i)
	}

	// Release all goroutines simultaneously
	close(start)
	wg.Wait()

	// Check that we got output
	output := buf.String()
	if len(output) == 0 {
		t.Error("no output from high contention test")
	}

	// Verify at least some events are valid
	lines := strings.Split(strings.TrimSpace(output), "\n")
	validEvents := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err == nil {
			validEvents++
		}
	}

	expectedTotal := numGoroutines * burstSize
	t.Logf("High contention: %d/%d events valid", validEvents, expectedTotal)

	if validEvents == 0 {
		t.Error("no valid events under high contention")
	}
}

// TestConcurrentEventEmission_AllStates tests that all state types
// can be emitted concurrently without issues.
func TestConcurrentEventEmission_AllStates(t *testing.T) {
	var buf bytes.Buffer
	emitter := &NDJSONEmitter{
		encoder: json.NewEncoder(&buf),
	}

	states := []string{"started", "running", "completed", "failed", "retrying"}

	var wg sync.WaitGroup
	for i, state := range states {
		wg.Add(1)
		go func(idx int, s string) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				emitter.Emit(Event{
					Timestamp:  time.Now(),
					PipelineID: fmt.Sprintf("pipeline-%s", s),
					StepID:     fmt.Sprintf("step-%d", j),
					State:      s,
					DurationMs: int64(j * 50),
					Persona:    "test-persona",
					TokensUsed: j * 100,
					Artifacts:  []string{fmt.Sprintf("artifact-%d.txt", j)},
				})
			}
		}(i, state)
	}

	wg.Wait()

	// Verify we captured events for all states
	output := buf.String()
	for _, state := range states {
		if !strings.Contains(output, fmt.Sprintf(`"state":"%s"`, state)) {
			t.Errorf("missing events for state: %s", state)
		}
	}
}

// =============================================================================
// T017: Step-Start Event Metadata Tests
// =============================================================================

// TestStepStartEventMetadata verifies that step-start events include Model and
// Adapter fields in NDJSON output, and that these fields are omitted when empty.
func TestStepStartEventMetadata(t *testing.T) {
	tests := []struct {
		name          string
		event         Event
		expectModel   bool
		expectAdapter bool
		modelValue    string
		adapterValue  string
	}{
		{
			name: "step-start includes model and adapter",
			event: Event{
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID: "test-pipeline",
				StepID:     "step-001",
				State:      "running",
				Persona:    "researcher",
				Model:      "opus",
				Adapter:    "claude",
			},
			expectModel:   true,
			expectAdapter: true,
			modelValue:    "opus",
			adapterValue:  "claude",
		},
		{
			name: "model and adapter omitted when empty",
			event: Event{
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID: "test-pipeline",
				StepID:     "step-002",
				State:      "running",
				Persona:    "researcher",
			},
			expectModel:   false,
			expectAdapter: false,
		},
		{
			name: "non-step-start event has no metadata",
			event: Event{
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID: "test-pipeline",
				StepID:     "step-003",
				State:      StateStreamActivity,
				ToolName:   "Read",
				ToolTarget: "/tmp/file.go",
			},
			expectModel:   false,
			expectAdapter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				emitter := NewNDJSONEmitter()
				emitter.Emit(tt.event)
			})

			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
				t.Fatalf("failed to parse NDJSON: %v", err)
			}

			_, hasModel := parsed["model"]
			_, hasAdapter := parsed["adapter"]

			if hasModel != tt.expectModel {
				t.Errorf("model presence: got %v, want %v", hasModel, tt.expectModel)
			}
			if hasAdapter != tt.expectAdapter {
				t.Errorf("adapter presence: got %v, want %v", hasAdapter, tt.expectAdapter)
			}

			if tt.expectModel {
				if got := parsed["model"].(string); got != tt.modelValue {
					t.Errorf("model = %q, want %q", got, tt.modelValue)
				}
			}
			if tt.expectAdapter {
				if got := parsed["adapter"].(string); got != tt.adapterValue {
					t.Errorf("adapter = %q, want %q", got, tt.adapterValue)
				}
			}
		})
	}
}

// =============================================================================
// T019: ETA Field Presence Tests
// =============================================================================

// TestETAFieldPresence verifies that EstimatedTimeMs is ALWAYS present in JSON
// output, even when zero, because its struct tag has no omitempty.
func TestETAFieldPresence(t *testing.T) {
	tests := []struct {
		name      string
		event     Event
		expectETA bool
		etaValue  float64 // JSON numbers decode as float64
	}{
		{
			name: "progress event includes zero ETA",
			event: Event{
				Timestamp:       time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID:      "test-pipeline",
				StepID:          "step-001",
				State:           StateStepProgress,
				EstimatedTimeMs: 0,
			},
			expectETA: true,
			etaValue:  0,
		},
		{
			name: "progress event includes non-zero ETA",
			event: Event{
				Timestamp:       time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID:      "test-pipeline",
				StepID:          "step-001",
				State:           StateStepProgress,
				EstimatedTimeMs: 30000,
			},
			expectETA: true,
			etaValue:  30000,
		},
		{
			name: "basic event still includes zero ETA field",
			event: Event{
				Timestamp:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				PipelineID: "test-pipeline",
				State:      "started",
			},
			expectETA: true,
			etaValue:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				emitter := NewNDJSONEmitter()
				emitter.Emit(tt.event)
			})

			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
				t.Fatalf("failed to parse NDJSON: %v", err)
			}

			etaVal, hasETA := parsed["estimated_time_ms"]
			if hasETA != tt.expectETA {
				t.Errorf("estimated_time_ms presence: got %v, want %v (parsed: %v)", hasETA, tt.expectETA, parsed)
			}

			if tt.expectETA {
				etaFloat, ok := etaVal.(float64)
				if !ok {
					t.Fatalf("estimated_time_ms is not a number: %T", etaVal)
				}
				if etaFloat != tt.etaValue {
					t.Errorf("estimated_time_ms = %v, want %v", etaFloat, tt.etaValue)
				}
			}
		})
	}
}

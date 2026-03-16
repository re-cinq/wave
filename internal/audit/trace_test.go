package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewDebugTracer_CreatesFile(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "test-run-001")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}
	defer tracer.Close()

	// Verify trace dir was created.
	if _, err := os.Stat(traceDir); os.IsNotExist(err) {
		t.Error("trace directory not created")
	}

	// Verify trace file was created.
	tracePath := tracer.TracePath()
	if _, err := os.Stat(tracePath); os.IsNotExist(err) {
		t.Error("trace file not created")
	}

	// Verify filename format.
	if !strings.HasSuffix(tracePath, "test-run-001.json") {
		t.Errorf("unexpected trace file name: %s", tracePath)
	}
}

func TestDebugTracer_EmitWritesNDJSON(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "test-run-002")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}

	// Emit several events.
	events := []TraceEvent{
		{EventType: "adapter_start", StepID: "investigate", Metadata: map[string]string{"persona": "navigator"}},
		{EventType: "adapter_end", StepID: "investigate", DurationMs: 5000, Metadata: map[string]string{"status": "success"}},
		{EventType: "contract_validation_start", StepID: "investigate", Metadata: map[string]string{"type": "json_schema"}},
		{EventType: "contract_validation_end", StepID: "investigate", DurationMs: 150, Metadata: map[string]string{"result": "pass"}},
	}

	for _, ev := range events {
		if err := tracer.Emit(ev); err != nil {
			t.Fatalf("Emit failed: %v", err)
		}
	}

	tracer.Close()

	// Read and parse the trace file.
	data, err := os.ReadFile(tracer.TracePath())
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	// Parse each line as JSON.
	for i, line := range lines {
		var ev TraceEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("line %d: failed to parse JSON: %v", i, err)
			continue
		}
		if ev.Timestamp == "" {
			t.Errorf("line %d: missing timestamp", i)
		}
		if ev.EventType != events[i].EventType {
			t.Errorf("line %d: expected event_type %q, got %q", i, events[i].EventType, ev.EventType)
		}
		if ev.StepID != events[i].StepID {
			t.Errorf("line %d: expected step_id %q, got %q", i, events[i].StepID, ev.StepID)
		}
	}
}

func TestDebugTracer_CredentialScrubbing(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "test-scrub")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}

	err = tracer.Emit(TraceEvent{
		EventType: "adapter_start",
		StepID:    "step1",
		Metadata:  map[string]string{"args": "API_KEY=sk-secret123abc"},
	})
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	tracer.Close()

	data, err := os.ReadFile(tracer.TracePath())
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "sk-secret123abc") {
		t.Errorf("trace file contains unredacted secret: %s", content)
	}
	if !strings.Contains(content, "[REDACTED]") {
		t.Errorf("trace file missing [REDACTED] marker: %s", content)
	}
}

func TestDebugTracer_ConcurrentEmit(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "test-concurrent")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			tracer.Emit(TraceEvent{
				EventType: "adapter_start",
				StepID:    "step1",
			})
		}(i)
	}
	wg.Wait()

	tracer.Close()

	events, err := ReadTraceFile(tracer.TracePath())
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}
	if len(events) != 50 {
		t.Errorf("expected 50 events, got %d", len(events))
	}
}

func TestDebugTracer_EmitAfterClose(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "test-closed")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}

	tracer.Close()

	err = tracer.Emit(TraceEvent{EventType: "test"})
	if err == nil {
		t.Error("expected error when emitting after close")
	}
}

func TestReadTraceFile(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "test-read")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}

	tracer.Emit(TraceEvent{EventType: "adapter_start", StepID: "s1", DurationMs: 0})
	tracer.Emit(TraceEvent{EventType: "adapter_end", StepID: "s1", DurationMs: 3000})
	tracer.Emit(TraceEvent{EventType: "artifact_injection", StepID: "s2", DurationMs: 50})
	tracer.Close()

	events, err := ReadTraceFile(tracer.TracePath())
	if err != nil {
		t.Fatalf("ReadTraceFile failed: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].EventType != "adapter_start" {
		t.Errorf("event 0: expected adapter_start, got %s", events[0].EventType)
	}
	if events[1].DurationMs != 3000 {
		t.Errorf("event 1: expected duration 3000, got %d", events[1].DurationMs)
	}
	if events[2].StepID != "s2" {
		t.Errorf("event 2: expected step_id s2, got %s", events[2].StepID)
	}
}

func TestReadTraceFile_NotFound(t *testing.T) {
	_, err := ReadTraceFile("/nonexistent/path/trace.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFindTraceFile(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	tracer, err := NewDebugTracer(traceDir, "findme-run")
	if err != nil {
		t.Fatalf("failed to create debug tracer: %v", err)
	}
	tracer.Close()

	path, err := FindTraceFile(traceDir, "findme-run")
	if err != nil {
		t.Fatalf("FindTraceFile failed: %v", err)
	}
	if !strings.HasSuffix(path, "findme-run.json") {
		t.Errorf("unexpected path: %s", path)
	}

	_, err = FindTraceFile(traceDir, "nonexistent-run")
	if err == nil {
		t.Error("expected error for nonexistent run")
	}
}

func TestTraceEvent_OmitsEmptyFields(t *testing.T) {
	ev := TraceEvent{
		Timestamp: "2026-01-01T00:00:00Z",
		EventType: "adapter_start",
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "step_id") {
		t.Error("empty step_id should be omitted")
	}
	if strings.Contains(content, "duration_ms") {
		t.Error("zero duration_ms should be omitted")
	}
	if strings.Contains(content, "metadata") {
		t.Error("nil metadata should be omitted")
	}
}

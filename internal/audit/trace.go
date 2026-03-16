package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TraceEvent represents a single structured trace event written as NDJSON.
type TraceEvent struct {
	Timestamp  string            `json:"timestamp"`
	EventType  string            `json:"event_type"`
	StepID     string            `json:"step_id,omitempty"`
	DurationMs int64             `json:"duration_ms,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DebugTracer writes structured NDJSON trace events to .wave/traces/<run-id>.json.
// It is enabled by the --debug flag and provides fine-grained timing for
// adapter execution, contract validation, and artifact injection.
type DebugTracer struct {
	mu        sync.Mutex
	file      *os.File
	runID     string
	traceDir  string
	tracePath string
	scrubber  *TraceLogger // reuse credential scrubbing from existing logger
}

// NewDebugTracer creates a new debug tracer that writes NDJSON events to
// .wave/traces/<runID>.json. The caller must call Close() when done.
func NewDebugTracer(traceDir, runID string) (*DebugTracer, error) {
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create trace dir: %w", err)
	}

	tracePath := filepath.Join(traceDir, runID+".json")
	file, err := os.OpenFile(tracePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}

	// Build a minimal scrubber for credential redaction.
	// We only need the regex, not the file handle, so we create a TraceLogger
	// in a temp dir and immediately close it. This reuses the existing pattern.
	scrubber, err := NewTraceLoggerWithDir(os.TempDir())
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create scrubber: %w", err)
	}
	// Close the scrubber's file — we only use its scrub() method.
	scrubber.file.Close()
	scrubber.file = nil

	return &DebugTracer{
		file:      file,
		runID:     runID,
		traceDir:  traceDir,
		tracePath: tracePath,
		scrubber:  scrubber,
	}, nil
}

// Emit writes a single trace event as an NDJSON line. Thread-safe.
func (t *DebugTracer) Emit(ev TraceEvent) error {
	if ev.Timestamp == "" {
		ev.Timestamp = time.Now().Format(time.RFC3339Nano)
	}

	// Scrub credential patterns from metadata values.
	if t.scrubber != nil && len(ev.Metadata) > 0 {
		scrubbed := make(map[string]string, len(ev.Metadata))
		for k, v := range ev.Metadata {
			scrubbed[k] = t.scrubber.scrub(v)
		}
		ev.Metadata = scrubbed
	}

	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal trace event: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.file == nil {
		return fmt.Errorf("tracer is closed")
	}

	_, err = t.file.Write(append(data, '\n'))
	return err
}

// TracePath returns the absolute path to the trace file.
func (t *DebugTracer) TracePath() string {
	return t.tracePath
}

// Close flushes and closes the trace file.
func (t *DebugTracer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.file != nil {
		err := t.file.Close()
		t.file = nil
		return err
	}
	return nil
}

// ReadTraceFile reads and parses an NDJSON trace file, returning all events.
func ReadTraceFile(tracePath string) ([]TraceEvent, error) {
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace file: %w", err)
	}

	var events []TraceEvent
	// Split by newlines, parse each line as JSON.
	for _, line := range splitNDJSON(data) {
		if len(line) == 0 {
			continue
		}
		var ev TraceEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // skip malformed lines
		}
		events = append(events, ev)
	}

	return events, nil
}

// FindTraceFile locates a trace file for a given run ID in the traces directory.
func FindTraceFile(traceDir, runID string) (string, error) {
	candidate := filepath.Join(traceDir, runID+".json")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("trace file not found for run %q in %s", runID, traceDir)
}

// splitNDJSON splits NDJSON data into individual JSON lines.
func splitNDJSON(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	// Handle last line without trailing newline.
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

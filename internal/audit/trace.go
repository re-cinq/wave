package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Trace event type constants for structured debug output.
const (
	TracePromptLoad        = "prompt_load"
	TracePromptLoadError   = "prompt_load_error"
	TraceArtifactWrite     = "artifact_write"
	TraceArtifactSkipEmpty = "artifact_skip_empty"
	TraceArtifactPreserved = "artifact_preserved"
	TraceThreadInject      = "thread_inject" // Thread transcript prepended to step prompt
	TraceThreadAppend      = "thread_append" // Step output appended to thread transcript
)

// TraceEvent represents a single structured trace event written as NDJSON.
type TraceEvent struct {
	Timestamp  string            `json:"timestamp"`
	EventType  string            `json:"event_type"`
	StepID     string            `json:"step_id,omitempty"`
	DurationMs int64             `json:"duration_ms,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DebugTracerOption configures a DebugTracer via functional options.
type DebugTracerOption func(*DebugTracer)

// WithStderrMirror enables human-readable debug output on stderr alongside NDJSON file output.
func WithStderrMirror(enabled bool) DebugTracerOption {
	return func(t *DebugTracer) {
		if enabled {
			t.stderrMirror = os.Stderr
		} else {
			t.stderrMirror = nil
		}
	}
}

// withStderrWriter sets a custom writer for stderr mirror output (for testing).
func withStderrWriter(w io.Writer) DebugTracerOption {
	return func(t *DebugTracer) {
		t.stderrMirror = w
	}
}

// DebugTracer writes structured NDJSON trace events to .wave/traces/<run-id>.json.
// It is enabled by the --debug flag and provides fine-grained timing for
// adapter execution, contract validation, and artifact injection.
type DebugTracer struct {
	mu           sync.Mutex
	file         *os.File
	runID        string
	traceDir     string
	tracePath    string
	scrubber     *TraceLogger // reuse credential scrubbing from existing logger
	stderrMirror io.Writer    // optional: mirror trace events to stderr in human-readable format
}

// NewDebugTracer creates a new debug tracer that writes NDJSON events to
// .wave/traces/<runID>.json. The caller must call Close() when done.
func NewDebugTracer(traceDir, runID string, opts ...DebugTracerOption) (*DebugTracer, error) {
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

	t := &DebugTracer{
		file:      file,
		runID:     runID,
		traceDir:  traceDir,
		tracePath: tracePath,
		scrubber:  scrubber,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t, nil
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

	// Write NDJSON line to trace file.
	if _, err = t.file.Write(append(data, '\n')); err != nil {
		return err
	}

	// Mirror to stderr in human-readable format if enabled.
	if t.stderrMirror != nil {
		line := formatTraceForStderr(ev)
		fmt.Fprintln(t.stderrMirror, line)
	}

	return nil
}

// formatTraceForStderr formats a TraceEvent as a human-readable debug line.
func formatTraceForStderr(ev TraceEvent) string {
	msg := fmt.Sprintf("[TRACE] %s", ev.EventType)
	if ev.StepID != "" {
		msg += fmt.Sprintf(" step=%s", ev.StepID)
	}
	if ev.DurationMs > 0 {
		msg += fmt.Sprintf(" duration=%dms", ev.DurationMs)
	}
	for k, v := range ev.Metadata {
		msg += fmt.Sprintf(" %s=%s", k, v)
	}
	return msg
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

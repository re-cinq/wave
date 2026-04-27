package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// logsTestHelper provides common utilities for logs command tests.
type logsTestHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
	store   state.StateStore
}

// newLogsTestHelper creates a new test helper with a temporary directory and database.
func newLogsTestHelper(t *testing.T) *logsTestHelper {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	waveDir := filepath.Join(tmpDir, ".agents")
	err = os.MkdirAll(waveDir, 0755)
	require.NoError(t, err, "failed to create .wave directory")

	dbPath := filepath.Join(waveDir, "state.db")
	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err, "failed to create state store")

	return &logsTestHelper{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
		store:   store,
	}
}

// chdir changes to the temporary directory.
func (h *logsTestHelper) chdir() {
	h.t.Helper()
	err := os.Chdir(h.tmpDir)
	require.NoError(h.t, err, "failed to change to temp directory")
}

// restore returns to the original directory and closes the store.
func (h *logsTestHelper) restore() {
	h.t.Helper()
	_ = os.Chdir(h.origDir)
	if h.store != nil {
		h.store.Close()
	}
}

// createRun creates a run in the database.
func (h *logsTestHelper) createRun(runID, pipelineName, status string, startedAt time.Time) {
	h.t.Helper()
	require.NoError(h.t, state.SeedRun(h.store, state.SeedRunOptions{
		RunID:        runID,
		PipelineName: pipelineName,
		Status:       status,
		StartedAt:    startedAt,
	}), "failed to create run")
}

// createLogEntry creates a log entry in the database.
func (h *logsTestHelper) createLogEntry(runID string, timestamp time.Time, stepID, evState, persona, message string, tokens int, durationMs int64) {
	h.t.Helper()
	require.NoError(h.t, state.SeedEvent(h.store, state.SeedEventOptions{
		RunID:      runID,
		Timestamp:  timestamp,
		StepID:     stepID,
		State:      evState,
		Persona:    persona,
		Message:    message,
		TokensUsed: tokens,
		DurationMs: durationMs,
	}), "failed to create log entry")
}

// executeLogsCmd runs the logs command with given arguments and returns output/error.
func executeLogsCmd(args ...string) (stdout, stderr string, err error) { //nolint:unparam // kept for interface/future use
	cmd := NewLogsCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	// Capture stdout since logs command uses fmt.Printf directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String(), errBuf.String(), err
}

// TestLogsCmd_NoDatabase tests when no state database exists.
func TestLogsCmd_NoDatabase(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	// Remove the database
	_ = os.RemoveAll(filepath.Join(h.tmpDir, ".agents"))

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No logs found")
}

// TestLogsCmd_NoRuns tests when no pipeline runs exist.
func TestLogsCmd_NoRuns(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No pipeline runs found")
}

// TestLogsCmd_BasicLogRetrieval tests basic log retrieval.
func TestLogsCmd_BasicLogRetrieval(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run-001", "test-pipeline", "completed", now.Add(-5*time.Minute))

	h.createLogEntry("test-run-001", now.Add(-5*time.Minute), "investigate", "started", "investigator", "Starting investigation", 0, 0)
	h.createLogEntry("test-run-001", now.Add(-3*time.Minute), "investigate", "running", "investigator", "", 0, 0)
	h.createLogEntry("test-run-001", now.Add(-1*time.Minute), "investigate", "completed", "investigator", "", 45000, 120000)

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "started")
	assert.Contains(t, stdout, "investigate")
	assert.Contains(t, stdout, "investigator")
	assert.Contains(t, stdout, "completed")
}

// TestLogsCmd_SpecificRunID tests logs for specific run ID.
func TestLogsCmd_SpecificRunID(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("run-001", "pipeline-a", "completed", now.Add(-10*time.Minute))
	h.createRun("run-002", "pipeline-b", "completed", now.Add(-5*time.Minute))

	h.createLogEntry("run-001", now.Add(-10*time.Minute), "step1", "started", "persona1", "Run 001 log", 0, 0)
	h.createLogEntry("run-002", now.Add(-5*time.Minute), "step1", "started", "persona1", "Run 002 log", 0, 0)

	stdout, _, err := executeLogsCmd("run-001")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Run 001 log")
	assert.NotContains(t, stdout, "Run 002 log")
}

// TestLogsCmd_RunNotFound tests when run ID is not found.
func TestLogsCmd_RunNotFound(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("existing-run", "test-pipeline", "completed", time.Now())

	stdout, _, err := executeLogsCmd("nonexistent-run")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Run not found")
}

// TestLogsCmd_StepFilter tests --step filtering.
func TestLogsCmd_StepFilter(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createLogEntry("test-run", now.Add(-5*time.Minute), "investigate", "started", "investigator", "Investigating", 0, 0)
	h.createLogEntry("test-run", now.Add(-3*time.Minute), "plan", "started", "planner", "Planning", 0, 0)
	h.createLogEntry("test-run", now.Add(-1*time.Minute), "execute", "started", "executor", "Executing", 0, 0)

	stdout, _, err := executeLogsCmd("--step", "plan")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Planning")
	assert.NotContains(t, stdout, "Investigating")
	assert.NotContains(t, stdout, "Executing")
}

// TestLogsCmd_ErrorsFilter tests --errors filtering.
func TestLogsCmd_ErrorsFilter(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "failed", now)

	h.createLogEntry("test-run", now.Add(-5*time.Minute), "step1", "started", "persona1", "Starting", 0, 0)
	h.createLogEntry("test-run", now.Add(-3*time.Minute), "step1", "running", "persona1", "Running", 0, 0)
	h.createLogEntry("test-run", now.Add(-1*time.Minute), "step1", "failed", "persona1", "Error occurred", 0, 0)

	stdout, _, err := executeLogsCmd("--errors")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Error occurred")
	assert.NotContains(t, stdout, "Starting")
	assert.NotContains(t, stdout, "Running")
}

// TestLogsCmd_TailFilter tests --tail filtering.
func TestLogsCmd_TailFilter(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	// Create 10 log entries
	for i := 0; i < 10; i++ {
		h.createLogEntry("test-run", now.Add(time.Duration(i)*time.Minute), "step1", "running", "persona1", "Log entry "+string(rune('0'+i)), 0, 0)
	}

	stdout, _, err := executeLogsCmd("--tail", "3")
	require.NoError(t, err)
	// Should only have last 3 entries
	assert.Contains(t, stdout, "Log entry 7")
	assert.Contains(t, stdout, "Log entry 8")
	assert.Contains(t, stdout, "Log entry 9")
	assert.NotContains(t, stdout, "Log entry 0")
	assert.NotContains(t, stdout, "Log entry 5")
}

// TestLogsCmd_SinceFilter tests --since filtering.
func TestLogsCmd_SinceFilter(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createLogEntry("test-run", now.Add(-2*time.Hour), "step1", "started", "persona1", "Old log", 0, 0)
	h.createLogEntry("test-run", now.Add(-5*time.Minute), "step1", "running", "persona1", "Recent log", 0, 0)

	stdout, _, err := executeLogsCmd("--since", "10m")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Recent log")
	assert.NotContains(t, stdout, "Old log")
}

// TestLogsCmd_EmptyLogs tests when run has no logs.
func TestLogsCmd_EmptyLogs(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("test-run", "test-pipeline", "running", time.Now())

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No logs found")
}

// TestLogsCmd_JSONFormat tests JSON output format.
func TestLogsCmd_JSONFormat(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createLogEntry("test-run", now.Add(-5*time.Minute), "investigate", "started", "investigator", "Starting", 0, 0)
	h.createLogEntry("test-run", now.Add(-3*time.Minute), "investigate", "completed", "investigator", "Done", 45000, 120000)

	stdout, _, err := executeLogsCmd("--format", "json")
	require.NoError(t, err)

	var output LogsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, "test-run", output.RunID)
	require.Len(t, output.Logs, 2)
	assert.Equal(t, "started", output.Logs[0].State)
	assert.Equal(t, "investigate", output.Logs[0].StepID)
	assert.Equal(t, "completed", output.Logs[1].State)
	assert.Equal(t, 45000, output.Logs[1].TokensUsed)
}

// TestLogsCmd_JSONFormatEmpty tests JSON output when no logs exist.
func TestLogsCmd_JSONFormatEmpty(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("test-run", "test-pipeline", "running", time.Now())

	stdout, _, err := executeLogsCmd("--format", "json")
	require.NoError(t, err)

	var output LogsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, "test-run", output.RunID)
	assert.Empty(t, output.Logs)
}

// TestLogsCmd_MultiStepPipeline tests logs from multi-step pipeline.
func TestLogsCmd_MultiStepPipeline(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	// Create logs for multiple steps
	h.createLogEntry("test-run", now.Add(-10*time.Minute), "navigate", "started", "navigator", "", 0, 0)
	h.createLogEntry("test-run", now.Add(-8*time.Minute), "navigate", "completed", "navigator", "", 10000, 120000)
	h.createLogEntry("test-run", now.Add(-7*time.Minute), "plan", "started", "planner", "", 0, 0)
	h.createLogEntry("test-run", now.Add(-5*time.Minute), "plan", "completed", "planner", "", 15000, 120000)
	h.createLogEntry("test-run", now.Add(-4*time.Minute), "execute", "started", "executor", "", 0, 0)
	h.createLogEntry("test-run", now.Add(-1*time.Minute), "execute", "completed", "executor", "", 20000, 180000)

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "navigate")
	assert.Contains(t, stdout, "plan")
	assert.Contains(t, stdout, "execute")
	assert.Contains(t, stdout, "navigator")
	assert.Contains(t, stdout, "planner")
	assert.Contains(t, stdout, "executor")
}

// TestLogsCmd_LevelError tests --level error filtering.
func TestLogsCmd_LevelError(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "failed", now)

	h.createLogEntry("test-run", now.Add(-5*time.Minute), "step1", "started", "persona1", "Starting", 0, 0)
	h.createLogEntry("test-run", now.Add(-1*time.Minute), "step1", "failed", "persona1", "Error", 0, 0)

	stdout, _, err := executeLogsCmd("--level", "error")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Error")
	assert.NotContains(t, stdout, "Starting")
}

// TestLogsCmd_TimestampFormat tests log timestamp formatting.
func TestLogsCmd_TimestampFormat(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	specificTime := time.Date(2026, 2, 2, 14, 30, 22, 0, time.Local)
	h.createRun("test-run", "test-pipeline", "completed", specificTime)
	h.createLogEntry("test-run", specificTime, "step1", "started", "persona1", "Test", 0, 0)

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "14:30:22")
}

// TestLogsCmd_TokensAndDuration tests that tokens and duration are displayed.
func TestLogsCmd_TokensAndDuration(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)
	h.createLogEntry("test-run", now, "step1", "completed", "persona1", "", 45000, 143200)

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "45k tokens")
	assert.Contains(t, stdout, "143.2s")
}

// TestLogsCmd_MostRecentRun tests default to most recent run.
func TestLogsCmd_MostRecentRun(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	// Create older run
	h.createRun("old-run", "pipeline-a", "completed", now.Add(-1*time.Hour))
	h.createLogEntry("old-run", now.Add(-1*time.Hour), "step1", "started", "persona1", "Old run log", 0, 0)

	// Create newer run
	h.createRun("new-run", "pipeline-b", "completed", now)
	h.createLogEntry("new-run", now, "step1", "started", "persona1", "New run log", 0, 0)

	stdout, _, err := executeLogsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "New run log")
	assert.NotContains(t, stdout, "Old run log")
}

// TestLogsCmd_SinceDaysParsing tests parsing of day duration in --since.
func TestLogsCmd_SinceDaysParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"10m", 10 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1d12h", 36 * time.Hour, false},
		{"invalid", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseDuration(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestLogsCmd_CombinedFilters tests multiple filters combined.
func TestLogsCmd_CombinedFilters(t *testing.T) {
	h := newLogsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "failed", now)

	// Mix of logs from different steps and states
	h.createLogEntry("test-run", now.Add(-10*time.Minute), "step1", "started", "persona1", "Step1 start", 0, 0)
	h.createLogEntry("test-run", now.Add(-8*time.Minute), "step1", "failed", "persona1", "Step1 error", 0, 0)
	h.createLogEntry("test-run", now.Add(-5*time.Minute), "step2", "started", "persona2", "Step2 start", 0, 0)
	h.createLogEntry("test-run", now.Add(-3*time.Minute), "step2", "failed", "persona2", "Step2 error", 0, 0)

	// Filter by step and errors
	stdout, _, err := executeLogsCmd("--step", "step2", "--errors")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Step2 error")
	assert.NotContains(t, stdout, "Step1")
	assert.NotContains(t, stdout, "start")
}

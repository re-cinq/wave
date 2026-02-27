package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// statusTestHelper provides common utilities for status command tests.
type statusTestHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
	db      *sql.DB
}

// newStatusTestHelper creates a new test helper with a temporary directory and database.
func newStatusTestHelper(t *testing.T) *statusTestHelper {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	// Create .wave directory
	waveDir := filepath.Join(tmpDir, ".wave")
	err = os.MkdirAll(waveDir, 0755)
	require.NoError(t, err, "failed to create .wave directory")

	// Create and initialize database
	dbPath := filepath.Join(waveDir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err, "failed to open database")

	// Initialize schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pipeline_run (
			run_id TEXT PRIMARY KEY,
			pipeline_name TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
			input TEXT,
			current_step TEXT,
			total_tokens INTEGER DEFAULT 0,
			started_at INTEGER NOT NULL,
			completed_at INTEGER,
			cancelled_at INTEGER,
			error_message TEXT
		);
		CREATE TABLE IF NOT EXISTS event_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			step_id TEXT,
			state TEXT NOT NULL,
			persona TEXT,
			message TEXT,
			tokens_used INTEGER,
			duration_ms INTEGER,
			FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
		);
	`)
	require.NoError(t, err, "failed to initialize schema")

	return &statusTestHelper{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
		db:      db,
	}
}

// chdir changes to the temporary directory.
func (h *statusTestHelper) chdir() {
	h.t.Helper()
	err := os.Chdir(h.tmpDir)
	require.NoError(h.t, err, "failed to change to temp directory")
}

// restore returns to the original directory and closes the database.
func (h *statusTestHelper) restore() {
	h.t.Helper()
	_ = os.Chdir(h.origDir)
	if h.db != nil {
		h.db.Close()
	}
}

// createRun creates a run in the database.
func (h *statusTestHelper) createRun(runID, pipelineName, status, currentStep string, tokens int, startedAt time.Time, completedAt *time.Time) {
	h.t.Helper()

	var completedAtUnix *int64
	if completedAt != nil {
		unix := completedAt.Unix()
		completedAtUnix = &unix
	}

	_, err := h.db.Exec(`
		INSERT INTO pipeline_run (run_id, pipeline_name, status, current_step, total_tokens, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, runID, pipelineName, status, currentStep, tokens, startedAt.Unix(), completedAtUnix)
	require.NoError(h.t, err, "failed to create run")
}

// createRunWithInput creates a run with input and optional error message.
func (h *statusTestHelper) createRunWithInput(runID, pipelineName, status, input string, startedAt time.Time, errorMsg string) {
	h.t.Helper()

	_, err := h.db.Exec(`
		INSERT INTO pipeline_run (run_id, pipeline_name, status, input, total_tokens, started_at, error_message)
		VALUES (?, ?, ?, ?, 0, ?, ?)
	`, runID, pipelineName, status, input, startedAt.Unix(), errorMsg)
	require.NoError(h.t, err, "failed to create run")
}

// executeStatusCmd runs the status command with given arguments and returns output/error.
func executeStatusCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewStatusCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	// Capture stdout since status command uses fmt.Printf directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	return buf.String(), errBuf.String(), err
}

// TestStatusCmd_NoDatabase tests when no state database exists.
func TestStatusCmd_NoDatabase(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	// Remove the database
	os.RemoveAll(filepath.Join(h.tmpDir, ".wave"))

	stdout, _, err := executeStatusCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No pipelines found")
}

// TestStatusCmd_NoRunningPipelines tests when no pipelines are running.
func TestStatusCmd_NoRunningPipelines(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create a completed run
	completed := time.Now().Add(-1 * time.Minute)
	h.createRun("test-run-001", "test-pipeline", "completed", "", 1000, completed.Add(-2*time.Minute), &completed)

	stdout, _, err := executeStatusCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No running pipelines")
}

// TestStatusCmd_RunningPipeline tests showing a running pipeline.
func TestStatusCmd_RunningPipeline(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	// Set terminal width for non-TTY test environment
	t.Setenv("COLUMNS", "120")

	h.createRun("debug-20260202-143022", "debug", "running", "investigate", 45000, time.Now().Add(-2*time.Minute), nil)

	stdout, _, err := executeStatusCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "debug-20260202-143022")
	assert.Contains(t, stdout, "debug")
	assert.Contains(t, stdout, "running")
	assert.Contains(t, stdout, "investigate")
	assert.Contains(t, stdout, "45k")
}

// TestStatusCmd_MultipleRunningPipelines tests showing multiple running pipelines.
func TestStatusCmd_MultipleRunningPipelines(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("run-001", "pipeline-a", "running", "step1", 10000, time.Now().Add(-1*time.Minute), nil)
	h.createRun("run-002", "pipeline-b", "running", "step2", 20000, time.Now().Add(-2*time.Minute), nil)

	stdout, _, err := executeStatusCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "run-001")
	assert.Contains(t, stdout, "run-002")
	assert.Contains(t, stdout, "pipeline-a")
	assert.Contains(t, stdout, "pipeline-b")
}

// TestStatusCmd_AllFlag tests the --all flag showing recent pipelines.
func TestStatusCmd_AllFlag(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	// Create various runs
	now := time.Now()
	completed := now.Add(-1 * time.Minute)
	h.createRun("run-001", "pipeline-a", "completed", "", 5000, now.Add(-5*time.Minute), &completed)
	h.createRun("run-002", "pipeline-b", "failed", "", 3000, now.Add(-3*time.Minute), &completed)
	h.createRun("run-003", "pipeline-c", "running", "step1", 1000, now.Add(-1*time.Minute), nil)

	stdout, _, err := executeStatusCmd("--all")
	require.NoError(t, err)
	assert.Contains(t, stdout, "run-001")
	assert.Contains(t, stdout, "run-002")
	assert.Contains(t, stdout, "run-003")
	assert.Contains(t, stdout, "completed")
	assert.Contains(t, stdout, "failed")
	assert.Contains(t, stdout, "running")
}

// TestStatusCmd_SpecificRunID tests showing details for a specific run.
func TestStatusCmd_SpecificRunID(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRunWithInput("test-run-123", "my-pipeline", "running", "fix bug in auth", time.Now().Add(-5*time.Minute), "")

	stdout, _, err := executeStatusCmd("test-run-123")
	require.NoError(t, err)
	assert.Contains(t, stdout, "test-run-123")
	assert.Contains(t, stdout, "my-pipeline")
	assert.Contains(t, stdout, "running")
	assert.Contains(t, stdout, "fix bug in auth")
}

// TestStatusCmd_SpecificRunIDNotFound tests when specific run ID is not found.
func TestStatusCmd_SpecificRunIDNotFound(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	stdout, _, err := executeStatusCmd("nonexistent-run")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Run not found")
}

// TestStatusCmd_JSONFormat tests JSON output format.
func TestStatusCmd_JSONFormat(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("test-run-001", "test-pipeline", "running", "step1", 5000, time.Now().Add(-1*time.Minute), nil)

	stdout, _, err := executeStatusCmd("--format", "json")
	require.NoError(t, err)

	var output StatusOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	require.Len(t, output.Runs, 1)
	assert.Equal(t, "test-run-001", output.Runs[0].RunID)
	assert.Equal(t, "test-pipeline", output.Runs[0].Pipeline)
	assert.Equal(t, "running", output.Runs[0].Status)
}

// TestStatusCmd_JSONFormatAll tests JSON output with --all flag.
func TestStatusCmd_JSONFormatAll(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	completed := now.Add(-1 * time.Minute)
	h.createRun("run-001", "pipeline-a", "completed", "", 5000, now.Add(-5*time.Minute), &completed)
	h.createRun("run-002", "pipeline-b", "running", "step1", 3000, now.Add(-1*time.Minute), nil)

	stdout, _, err := executeStatusCmd("--all", "--format", "json")
	require.NoError(t, err)

	var output StatusOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	assert.Len(t, output.Runs, 2)
}

// TestStatusCmd_JSONFormatSpecificRun tests JSON output for specific run.
func TestStatusCmd_JSONFormatSpecificRun(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRunWithInput("test-run-123", "my-pipeline", "completed", "test input", time.Now().Add(-5*time.Minute), "")

	stdout, _, err := executeStatusCmd("test-run-123", "--format", "json")
	require.NoError(t, err)

	var output StatusOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	require.Len(t, output.Runs, 1)
	assert.Equal(t, "test-run-123", output.Runs[0].RunID)
	assert.Equal(t, "test input", output.Runs[0].Input)
}

// TestStatusCmd_JSONFormatEmpty tests JSON output when no runs exist.
func TestStatusCmd_JSONFormatEmpty(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	stdout, _, err := executeStatusCmd("--format", "json")
	require.NoError(t, err)

	var output StatusOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	assert.Empty(t, output.Runs)
}

// TestStatusCmd_CompletedPipeline tests showing completed pipeline status.
func TestStatusCmd_CompletedPipeline(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	completed := now.Add(-1 * time.Minute)
	h.createRun("completed-run", "test-pipeline", "completed", "", 12000, now.Add(-2*time.Minute), &completed)

	stdout, _, err := executeStatusCmd("--all")
	require.NoError(t, err)
	assert.Contains(t, stdout, "completed-run")
	assert.Contains(t, stdout, "completed")
	assert.Contains(t, stdout, "12k")
}

// TestStatusCmd_FailedPipeline tests showing failed pipeline status.
func TestStatusCmd_FailedPipeline(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRunWithInput("failed-run", "test-pipeline", "failed", "test input", time.Now().Add(-5*time.Minute), "contract validation failed")

	stdout, _, err := executeStatusCmd("failed-run")
	require.NoError(t, err)
	assert.Contains(t, stdout, "failed-run")
	assert.Contains(t, stdout, "failed")
	assert.Contains(t, stdout, "contract validation failed")
}

// TestStatusCmd_CancelledPipeline tests showing cancelled pipeline status.
func TestStatusCmd_CancelledPipeline(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	cancelled := now.Add(-1 * time.Minute)
	h.createRun("cancelled-run", "test-pipeline", "cancelled", "", 500, now.Add(-3*time.Minute), &cancelled)

	stdout, _, err := executeStatusCmd("--all")
	require.NoError(t, err)
	assert.Contains(t, stdout, "cancelled-run")
	assert.Contains(t, stdout, "cancelled")
}

// TestStatusCmd_ElapsedTimeFormat tests elapsed time formatting.
func TestStatusCmd_ElapsedTimeFormat(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds only", 45 * time.Second, "0m45s"},
		{"minutes and seconds", 2*time.Minute + 34*time.Second, "2m34s"},
		{"hours and minutes", 1*time.Hour + 23*time.Minute, "1h23m"},
		{"just over an hour", 1*time.Hour + 5*time.Minute, "1h5m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatElapsed(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestStatusCmd_TokensFormat tests token count formatting.
func TestStatusCmd_TokensFormat(t *testing.T) {
	tests := []struct {
		tokens   int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1k"},
		{1500, "1k"},
		{45000, "45k"},
		{100000, "100k"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatTokens(tc.tokens)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestStatusCmd_TableHeader tests that table output includes header.
func TestStatusCmd_TableHeader(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("test-run", "test-pipeline", "running", "step1", 1000, time.Now(), nil)

	stdout, _, err := executeStatusCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "RUN_ID")
	assert.Contains(t, stdout, "PIPELINE")
	assert.Contains(t, stdout, "STATUS")
	assert.Contains(t, stdout, "STEP")
	assert.Contains(t, stdout, "ELAPSED")
	assert.Contains(t, stdout, "TOKENS")
}

// TestStatusCmd_NoStepShowsDash tests that missing step shows dash.
func TestStatusCmd_NoStepShowsDash(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	completed := now.Add(-1 * time.Minute)
	h.createRun("test-run", "test-pipeline", "completed", "", 1000, now.Add(-2*time.Minute), &completed)

	stdout, _, err := executeStatusCmd("--all")
	require.NoError(t, err)
	// Output should have dash for empty step
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "test-run") {
			assert.Contains(t, line, "-")
			break
		}
	}
}

// TestStatusCmd_TruncatesLongRunID tests that long run IDs are truncated.
func TestStatusCmd_TruncatesLongRunID(t *testing.T) {
	h := newStatusTestHelper(t)
	h.chdir()
	defer h.restore()

	longRunID := "very-long-run-id-that-exceeds-the-column-width"
	h.createRun(longRunID, "test-pipeline", "running", "step1", 1000, time.Now(), nil)

	stdout, _, err := executeStatusCmd()
	require.NoError(t, err)
	// Should contain truncated version with ...
	assert.Contains(t, stdout, "...")
}

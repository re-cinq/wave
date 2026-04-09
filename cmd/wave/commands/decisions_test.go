package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// decisionsTestHelper provides common utilities for decisions command tests.
type decisionsTestHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
	db      *sql.DB
}

// newDecisionsTestHelper creates a new test helper with a temporary directory and database.
func newDecisionsTestHelper(t *testing.T) *decisionsTestHelper {
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

	// Initialize schema (pipeline_run + decision_log)
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
		CREATE TABLE IF NOT EXISTS decision_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			step_id TEXT NOT NULL DEFAULT '',
			timestamp INTEGER NOT NULL,
			category TEXT NOT NULL,
			decision TEXT NOT NULL,
			rationale TEXT NOT NULL DEFAULT '',
			context_json TEXT NOT NULL DEFAULT '{}',
			FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
		);
	`)
	require.NoError(t, err, "failed to initialize schema")

	return &decisionsTestHelper{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
		db:      db,
	}
}

// chdir changes to the temporary directory.
func (h *decisionsTestHelper) chdir() {
	h.t.Helper()
	err := os.Chdir(h.tmpDir)
	require.NoError(h.t, err, "failed to change to temp directory")
}

// restore returns to the original directory and closes the database.
func (h *decisionsTestHelper) restore() {
	h.t.Helper()
	_ = os.Chdir(h.origDir)
	if h.db != nil {
		h.db.Close()
	}
}

// createRun creates a run in the database.
func (h *decisionsTestHelper) createRun(runID, pipelineName, status string, startedAt time.Time) {
	h.t.Helper()
	_, err := h.db.Exec(`
		INSERT INTO pipeline_run (run_id, pipeline_name, status, total_tokens, started_at)
		VALUES (?, ?, ?, 0, ?)
	`, runID, pipelineName, status, startedAt.Unix())
	require.NoError(h.t, err, "failed to create run")
}

// createDecision creates a decision entry in the database.
func (h *decisionsTestHelper) createDecision(runID string, timestamp time.Time, stepID, category, decision, rationale, contextJSON string) {
	h.t.Helper()
	if contextJSON == "" {
		contextJSON = "{}"
	}
	_, err := h.db.Exec(`
		INSERT INTO decision_log (run_id, step_id, timestamp, category, decision, rationale, context_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, runID, stepID, timestamp.Unix(), category, decision, rationale, contextJSON)
	require.NoError(h.t, err, "failed to create decision entry")
}

// executeDecisionsCmd runs the decisions command with given arguments and returns output/error.
func executeDecisionsCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewDecisionsCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	// Capture stdout since decisions command uses fmt.Printf directly
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

// TestDecisionsCmd_NoDatabase tests when no state database exists.
func TestDecisionsCmd_NoDatabase(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	// Remove the database
	_ = os.RemoveAll(filepath.Join(h.tmpDir, ".wave"))

	stdout, _, err := executeDecisionsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No decisions found")
}

// TestDecisionsCmd_NoRuns tests when no pipeline runs exist.
func TestDecisionsCmd_NoRuns(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	stdout, _, err := executeDecisionsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No pipeline runs found")
}

// TestDecisionsCmd_BasicRetrieval tests basic decision retrieval.
func TestDecisionsCmd_BasicRetrieval(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run-001", "test-pipeline", "completed", now.Add(-5*time.Minute))

	h.createDecision("test-run-001", now.Add(-5*time.Minute), "investigate", "model_routing",
		"selected model claude-opus", "per-persona config", `{"model":"claude-opus"}`)
	h.createDecision("test-run-001", now.Add(-3*time.Minute), "plan", "model_routing",
		"selected model claude-sonnet", "per-step pinning", `{"model":"claude-sonnet"}`)
	h.createDecision("test-run-001", now.Add(-1*time.Minute), "implement", "contract",
		"contract passed", "test_suite OK", `{"contract_type":"test_suite"}`)

	stdout, _, err := executeDecisionsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "model_routing")
	assert.Contains(t, stdout, "selected model claude-opus")
	assert.Contains(t, stdout, "contract")
	assert.Contains(t, stdout, "contract passed")
}

// TestDecisionsCmd_SpecificRunID tests decisions for specific run ID.
func TestDecisionsCmd_SpecificRunID(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("run-001", "pipeline-a", "completed", now.Add(-10*time.Minute))
	h.createRun("run-002", "pipeline-b", "completed", now.Add(-5*time.Minute))

	h.createDecision("run-001", now.Add(-10*time.Minute), "step1", "model_routing",
		"Run 001 model decision", "reason", "")
	h.createDecision("run-002", now.Add(-5*time.Minute), "step1", "model_routing",
		"Run 002 model decision", "reason", "")

	stdout, _, err := executeDecisionsCmd("run-001")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Run 001 model decision")
	assert.NotContains(t, stdout, "Run 002 model decision")
}

// TestDecisionsCmd_RunNotFound tests when run ID is not found.
func TestDecisionsCmd_RunNotFound(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("existing-run", "test-pipeline", "completed", time.Now())

	stdout, _, err := executeDecisionsCmd("nonexistent-run")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Run not found")
}

// TestDecisionsCmd_StepFilter tests --step filtering.
func TestDecisionsCmd_StepFilter(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createDecision("test-run", now.Add(-5*time.Minute), "investigate", "model_routing",
		"investigate decision", "reason", "")
	h.createDecision("test-run", now.Add(-3*time.Minute), "plan", "model_routing",
		"plan decision", "reason", "")
	h.createDecision("test-run", now.Add(-1*time.Minute), "implement", "contract",
		"implement decision", "reason", "")

	stdout, _, err := executeDecisionsCmd("--step", "plan")
	require.NoError(t, err)
	assert.Contains(t, stdout, "plan decision")
	assert.NotContains(t, stdout, "investigate decision")
	assert.NotContains(t, stdout, "implement decision")
}

// TestDecisionsCmd_CategoryFilter tests --category filtering.
func TestDecisionsCmd_CategoryFilter(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createDecision("test-run", now.Add(-5*time.Minute), "step1", "model_routing",
		"model decision", "reason", "")
	h.createDecision("test-run", now.Add(-3*time.Minute), "step1", "retry",
		"retry decision", "reason", "")
	h.createDecision("test-run", now.Add(-1*time.Minute), "step1", "contract",
		"contract decision", "reason", "")

	stdout, _, err := executeDecisionsCmd("--category", "retry")
	require.NoError(t, err)
	assert.Contains(t, stdout, "retry decision")
	assert.NotContains(t, stdout, "model decision")
	assert.NotContains(t, stdout, "contract decision")
}

// TestDecisionsCmd_EmptyDecisions tests when run has no decisions.
func TestDecisionsCmd_EmptyDecisions(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("test-run", "test-pipeline", "running", time.Now())

	stdout, _, err := executeDecisionsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "No decisions found")
}

// TestDecisionsCmd_JSONFormat tests JSON output format.
func TestDecisionsCmd_JSONFormat(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createDecision("test-run", now.Add(-5*time.Minute), "investigate", "model_routing",
		"selected model opus", "CLI override", `{"model":"opus"}`)
	h.createDecision("test-run", now.Add(-3*time.Minute), "investigate", "contract",
		"contract passed", "test_suite validated", `{"contract_type":"test_suite"}`)

	stdout, _, err := executeDecisionsCmd("--format", "json")
	require.NoError(t, err)

	var output DecisionsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, "test-run", output.RunID)
	require.Len(t, output.Decisions, 2)
	assert.Equal(t, "model_routing", output.Decisions[0].Category)
	assert.Equal(t, "investigate", output.Decisions[0].StepID)
	assert.Equal(t, "selected model opus", output.Decisions[0].Decision)
	assert.Equal(t, "CLI override", output.Decisions[0].Rationale)
	assert.Equal(t, "contract", output.Decisions[1].Category)
}

// TestDecisionsCmd_JSONFormatEmpty tests JSON output when no decisions exist.
func TestDecisionsCmd_JSONFormatEmpty(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	h.createRun("test-run", "test-pipeline", "running", time.Now())

	stdout, _, err := executeDecisionsCmd("--format", "json")
	require.NoError(t, err)

	var output DecisionsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err, "output should be valid JSON")
	assert.Equal(t, "test-run", output.RunID)
	assert.Empty(t, output.Decisions)
}

// TestDecisionsCmd_CombinedFilters tests multiple filters combined.
func TestDecisionsCmd_CombinedFilters(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createDecision("test-run", now.Add(-5*time.Minute), "step1", "model_routing",
		"step1 model", "reason", "")
	h.createDecision("test-run", now.Add(-4*time.Minute), "step1", "contract",
		"step1 contract", "reason", "")
	h.createDecision("test-run", now.Add(-3*time.Minute), "step2", "model_routing",
		"step2 model", "reason", "")
	h.createDecision("test-run", now.Add(-2*time.Minute), "step2", "contract",
		"step2 contract", "reason", "")

	// Filter by step AND category
	stdout, _, err := executeDecisionsCmd("--step", "step2", "--category", "contract")
	require.NoError(t, err)
	assert.Contains(t, stdout, "step2 contract")
	assert.NotContains(t, stdout, "step1")
	assert.NotContains(t, stdout, "model")
}

// TestDecisionsCmd_MostRecentRun tests default to most recent run.
func TestDecisionsCmd_MostRecentRun(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("old-run", "pipeline-a", "completed", now.Add(-1*time.Hour))
	h.createDecision("old-run", now.Add(-1*time.Hour), "step1", "model_routing",
		"Old run decision", "reason", "")

	h.createRun("new-run", "pipeline-b", "completed", now)
	h.createDecision("new-run", now, "step1", "model_routing",
		"New run decision", "reason", "")

	stdout, _, err := executeDecisionsCmd()
	require.NoError(t, err)
	assert.Contains(t, stdout, "New run decision")
	assert.NotContains(t, stdout, "Old run decision")
}

// TestDecisionsCmd_JSONContextField tests that context JSON is properly included.
func TestDecisionsCmd_JSONContextField(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	now := time.Now()
	h.createRun("test-run", "test-pipeline", "completed", now)

	h.createDecision("test-run", now, "step1", "model_routing",
		"model chosen", "auto-route", `{"model":"opus","complexity":"high"}`)

	stdout, _, err := executeDecisionsCmd("--format", "json")
	require.NoError(t, err)

	var output DecisionsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err)
	require.Len(t, output.Decisions, 1)

	// Verify context is valid JSON
	var ctx map[string]interface{}
	err = json.Unmarshal(output.Decisions[0].Context, &ctx)
	require.NoError(t, err)
	assert.Equal(t, "opus", ctx["model"])
	assert.Equal(t, "high", ctx["complexity"])
}

// TestDecisionsCmd_NoDatabaseJSON tests JSON output when no state database exists.
func TestDecisionsCmd_NoDatabaseJSON(t *testing.T) {
	h := newDecisionsTestHelper(t)
	h.chdir()
	defer h.restore()

	_ = os.RemoveAll(filepath.Join(h.tmpDir, ".wave"))

	stdout, _, err := executeDecisionsCmd("--format", "json")
	require.NoError(t, err)

	var output DecisionsOutput
	err = json.Unmarshal([]byte(stdout), &output)
	require.NoError(t, err)
	assert.Empty(t, output.Decisions)
}

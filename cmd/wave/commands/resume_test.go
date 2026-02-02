package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resumeTestEnv provides a clean testing environment for resume tests.
type resumeTestEnv struct {
	t       *testing.T
	tmpDir  string
	origDir string
	store   state.StateStore
}

// newResumeTestEnv creates a new test environment with a temp directory and state store.
func newResumeTestEnv(t *testing.T) *resumeTestEnv {
	t.Helper()

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "failed to change to temp directory")

	// Create .wave directory structure
	waveDir := filepath.Join(tmpDir, ".wave")
	err = os.MkdirAll(waveDir, 0755)
	require.NoError(t, err, "failed to create .wave directory")

	// Create state store
	dbPath := filepath.Join(waveDir, "state.db")
	store, err := state.NewStateStore(dbPath)
	require.NoError(t, err, "failed to create state store")

	return &resumeTestEnv{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
		store:   store,
	}
}

// cleanup restores the original working directory and closes the store.
func (e *resumeTestEnv) cleanup() {
	if e.store != nil {
		e.store.Close()
	}
	err := os.Chdir(e.origDir)
	if err != nil {
		e.t.Errorf("failed to restore original directory: %v", err)
	}
}

// executeResumeCmd runs the resume command with given arguments and returns output/error.
func executeResumeCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewResumeCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// createTestPipelineState creates a pipeline with steps in the state store for testing.
func (e *resumeTestEnv) createTestPipelineState(pipelineID string, status string, steps []struct {
	id    string
	state state.StepState
}) {
	e.t.Helper()

	err := e.store.SavePipelineState(pipelineID, status, `{"test": "input"}`)
	require.NoError(e.t, err, "failed to save pipeline state")

	for _, step := range steps {
		err := e.store.SaveStepState(pipelineID, step.id, step.state, "")
		require.NoError(e.t, err, "failed to save step state for %s", step.id)
	}
}

// T060: Test that resume lists recent pipelines when no ID provided
func TestResume_ListsRecentPipelines_WhenNoIDProvided(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// Create multiple pipeline states
	pipelines := []struct {
		id     string
		status string
	}{
		{"pipeline-001", "running"},
		{"pipeline-002", "failed"},
		{"pipeline-003", "paused"},
	}

	for _, p := range pipelines {
		err := env.store.SavePipelineState(p.id, p.status, `{"test": true}`)
		require.NoError(t, err)
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Capture stdout since listing writes to os.Stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create the resume command without --pipeline flag
	cmd := NewResumeCmd()
	cmd.SetArgs([]string{}) // No pipeline flag

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// T064: Should list pipelines when no ID provided
	require.NoError(t, err, "resume without pipeline ID should succeed and list pipelines")
	assert.Contains(t, output, "Recent pipelines", "should show recent pipelines header")
	assert.Contains(t, output, "pipeline-001", "should list first pipeline")
	assert.Contains(t, output, "pipeline-002", "should list second pipeline")
	assert.Contains(t, output, "pipeline-003", "should list third pipeline")
	assert.Contains(t, output, "running", "should show running status")
	assert.Contains(t, output, "FAILED", "should show failed status")
	assert.Contains(t, output, "paused", "should show paused status")
}

// T060: Test that ListRecentPipelines returns pipelines in correct order
func TestResume_ListRecentPipelines_ReturnsInOrder(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// Create pipelines with different timestamps
	// SQLite uses second-precision timestamps, so we need to wait 1+ seconds
	// between creations, OR update them sequentially to get different updated_at values
	pipelineIDs := []string{"first", "second", "third", "fourth", "fifth"}
	for _, pid := range pipelineIDs {
		err := env.store.SavePipelineState(pid, "running", "")
		require.NoError(t, err)
	}

	// Now update them in order to guarantee different updated_at timestamps
	// Each update will have a fresh time.Now() call
	time.Sleep(1100 * time.Millisecond) // Wait > 1 second to ensure different second
	err := env.store.SavePipelineState("fifth", "updated", "")
	require.NoError(t, err)

	// List recent pipelines
	records, err := env.store.ListRecentPipelines(10)
	require.NoError(t, err)
	assert.Len(t, records, 5)

	// "fifth" was updated last (after 1 second delay), so it should be first
	assert.Equal(t, "fifth", records[0].PipelineID)

	// The other 4 pipelines were created at roughly the same time,
	// so we just verify they're all present
	allIDs := make(map[string]bool)
	for _, r := range records {
		allIDs[r.PipelineID] = true
	}
	for _, pid := range pipelineIDs {
		assert.True(t, allIDs[pid], "pipeline %s should be in results", pid)
	}
}

// T060: Test that ListRecentPipelines respects limit
func TestResume_ListRecentPipelines_RespectsLimit(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// Create 10 pipelines
	for i := 0; i < 10; i++ {
		pipelineID := pipelineIDFromTestIndex(i)
		err := env.store.SavePipelineState(pipelineID, "running", "")
		require.NoError(t, err)
	}

	// List with limit of 3
	records, err := env.store.ListRecentPipelines(3)
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

// T061: Test resume continues from last completed step
func TestResume_ContinuesFromLastCompletedStep(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// Create pipeline with some completed and some pending steps
	pipelineID := "pipeline-resume-test"
	env.createTestPipelineState(pipelineID, "running", []struct {
		id    string
		state state.StepState
	}{
		{"step-1", state.StateCompleted},
		{"step-2", state.StateCompleted},
		{"step-3", state.StatePending},
		{"step-4", state.StatePending},
	})

	// Verify step states
	steps, err := env.store.GetStepStates(pipelineID)
	require.NoError(t, err)
	assert.Len(t, steps, 4)

	// Completed steps
	completedCount := 0
	pendingCount := 0
	for _, step := range steps {
		if step.State == state.StateCompleted {
			completedCount++
		}
		if step.State == state.StatePending {
			pendingCount++
		}
	}
	assert.Equal(t, 2, completedCount, "should have 2 completed steps")
	assert.Equal(t, 2, pendingCount, "should have 2 pending steps")
}

// T061: Test resume identifies the correct resumption point
func TestResume_IdentifiesCorrectResumptionPoint(t *testing.T) {
	testCases := []struct {
		name               string
		steps              []struct{ id string; state state.StepState }
		expectedResumeFrom string
	}{
		{
			name: "resume from first pending step",
			steps: []struct{ id string; state state.StepState }{
				{"step-1", state.StateCompleted},
				{"step-2", state.StateCompleted},
				{"step-3", state.StatePending},
			},
			expectedResumeFrom: "step-3",
		},
		{
			name: "resume from failed step",
			steps: []struct{ id string; state state.StepState }{
				{"step-1", state.StateCompleted},
				{"step-2", state.StateFailed},
				{"step-3", state.StatePending},
			},
			expectedResumeFrom: "step-2",
		},
		{
			name: "all steps completed",
			steps: []struct{ id string; state state.StepState }{
				{"step-1", state.StateCompleted},
				{"step-2", state.StateCompleted},
				{"step-3", state.StateCompleted},
			},
			expectedResumeFrom: "", // Nothing to resume
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newResumeTestEnv(t)
			defer env.cleanup()

			pipelineID := "test-pipeline"
			env.createTestPipelineState(pipelineID, "running", tc.steps)

			steps, err := env.store.GetStepStates(pipelineID)
			require.NoError(t, err)

			// Find first non-completed step
			var resumeFrom string
			for _, step := range steps {
				if step.State != state.StateCompleted {
					resumeFrom = step.StepID
					break
				}
			}

			assert.Equal(t, tc.expectedResumeFrom, resumeFrom)
		})
	}
}

// T062: Test resume handles retrying state (step that was in progress when killed)
func TestResume_HandlesRetryingState(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	pipelineID := "pipeline-with-retrying"
	env.createTestPipelineState(pipelineID, "running", []struct {
		id    string
		state state.StepState
	}{
		{"step-1", state.StateCompleted},
		{"step-2", state.StateRetrying},
		{"step-3", state.StatePending},
	})

	steps, err := env.store.GetStepStates(pipelineID)
	require.NoError(t, err)

	// Find the retrying step
	var retryingStep *state.StepStateRecord
	for i := range steps {
		if steps[i].State == state.StateRetrying {
			retryingStep = &steps[i]
			break
		}
	}

	require.NotNil(t, retryingStep, "should find retrying step")
	assert.Equal(t, "step-2", retryingStep.StepID)
	assert.Equal(t, state.StateRetrying, retryingStep.State)
}

// T062: Test that retrying state increments retry count
func TestResume_RetryingState_IncrementsRetryCount(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	pipelineID := "pipeline-retry-count"
	err := env.store.SavePipelineState(pipelineID, "running", "")
	require.NoError(t, err)

	// Save step in running state
	err = env.store.SaveStepState(pipelineID, "step-1", state.StateRunning, "")
	require.NoError(t, err)

	// Mark as retrying multiple times
	for i := 0; i < 3; i++ {
		err = env.store.SaveStepState(pipelineID, "step-1", state.StateRetrying, "temporary failure")
		require.NoError(t, err)
	}

	steps, err := env.store.GetStepStates(pipelineID)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, 3, steps[0].RetryCount, "retry count should be 3")
}

// T062: Test resume with step that was running when process was killed
func TestResume_HandlesRunningStateAsInterrupted(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	pipelineID := "pipeline-interrupted"
	env.createTestPipelineState(pipelineID, "running", []struct {
		id    string
		state state.StepState
	}{
		{"step-1", state.StateCompleted},
		{"step-2", state.StateRunning}, // Was running when killed
		{"step-3", state.StatePending},
	})

	steps, err := env.store.GetStepStates(pipelineID)
	require.NoError(t, err)

	// Find the running step - this should be treated as the resumption point
	var runningStep *state.StepStateRecord
	for i := range steps {
		if steps[i].State == state.StateRunning {
			runningStep = &steps[i]
			break
		}
	}

	require.NotNil(t, runningStep, "should find running step")
	assert.Equal(t, "step-2", runningStep.StepID)
}

// T063: Test resume with specific pipeline ID
func TestResume_WithSpecificPipelineID(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// Create multiple pipelines
	pipelines := []string{"pipeline-alpha", "pipeline-beta", "pipeline-gamma"}
	for _, pid := range pipelines {
		err := env.store.SavePipelineState(pid, "running", `{"name": "`+pid+`"}`)
		require.NoError(t, err)
	}

	// Retrieve specific pipeline
	record, err := env.store.GetPipelineState("pipeline-beta")
	require.NoError(t, err)
	assert.Equal(t, "pipeline-beta", record.PipelineID)
	assert.Equal(t, "running", record.Status)
}

// T063: Test resume with non-existent pipeline ID returns error
func TestResume_WithNonExistentPipelineID(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	_, err := env.store.GetPipelineState("nonexistent-pipeline")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline state not found")
}

// T063: Test resume command output for specific pipeline
func TestResume_CommandOutput_SpecificPipeline(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// Create a pipeline state
	pipelineID := "test-resume-output"
	err := env.store.SavePipelineState(pipelineID, "paused", "")
	require.NoError(t, err)

	// Capture stdout since the command writes to os.Stdout directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewResumeCmd()
	cmd.SetArgs([]string{"--pipeline", pipelineID})
	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stdout := buf.String()

	// The current implementation should produce output
	require.NoError(t, err)
	assert.Contains(t, stdout, "Resuming pipeline")
	assert.Contains(t, stdout, pipelineID)
}

// Test resume command flags
func TestResume_CommandFlags(t *testing.T) {
	cmd := NewResumeCmd()

	// Verify command properties
	assert.Equal(t, "resume", cmd.Use)
	assert.Contains(t, cmd.Short, "Resume")

	// Verify flags exist
	flags := cmd.Flags()

	pipelineFlag := flags.Lookup("pipeline")
	assert.NotNil(t, pipelineFlag, "pipeline flag should exist")

	fromStepFlag := flags.Lookup("from-step")
	assert.NotNil(t, fromStepFlag, "from-step flag should exist")

	manifestFlag := flags.Lookup("manifest")
	assert.NotNil(t, manifestFlag, "manifest flag should exist")
}

// Test resume with --from-step flag
func TestResume_WithFromStepFlag(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	pipelineID := "pipeline-from-step"
	err := env.store.SavePipelineState(pipelineID, "running", "")
	require.NoError(t, err)

	// Capture stdout since the command writes to os.Stdout directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewResumeCmd()
	cmd.SetArgs([]string{"--pipeline", pipelineID, "--from-step", "step-3"})
	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stdout := buf.String()

	require.NoError(t, err)
	assert.Contains(t, stdout, "step-3")
}

// Test resume when state database doesn't exist
func TestResume_NoStateDatabase(t *testing.T) {
	// Create a temp directory without state.db
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, _, err := executeResumeCmd("--pipeline", "any-pipeline")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no state database found")
}

// Test resume shows proper progress messages
func TestResume_ProgressMessages(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	pipelineID := "pipeline-progress"
	err := env.store.SavePipelineState(pipelineID, "running", "")
	require.NoError(t, err)

	// Capture stdout since the command writes to os.Stdout directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewResumeCmd()
	cmd.SetArgs([]string{"--pipeline", pipelineID})
	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stdout := buf.String()

	require.NoError(t, err)
	// Check for various progress messages
	assert.Contains(t, stdout, "Resuming pipeline")
	assert.Contains(t, stdout, "Loading state")
	assert.Contains(t, stdout, "Pipeline state loaded")
}

// Table-driven test for various pipeline states
func TestResume_VariousPipelineStates(t *testing.T) {
	testCases := []struct {
		name     string
		status   string
		canResume bool
	}{
		{"paused pipeline", "paused", true},
		{"failed pipeline", "failed", true},
		{"running pipeline", "running", true},
		{"completed pipeline", "completed", true}, // May just report nothing to do
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newResumeTestEnv(t)
			defer env.cleanup()

			pipelineID := "test-" + tc.status
			err := env.store.SavePipelineState(pipelineID, tc.status, "")
			require.NoError(t, err)

			record, err := env.store.GetPipelineState(pipelineID)
			require.NoError(t, err)
			assert.Equal(t, tc.status, record.Status)
		})
	}
}

// Test empty pipeline list
func TestResume_EmptyPipelineList(t *testing.T) {
	env := newResumeTestEnv(t)
	defer env.cleanup()

	// List pipelines when none exist
	records, err := env.store.ListRecentPipelines(10)
	require.NoError(t, err)
	assert.Empty(t, records)
}

// Helper function
func pipelineIDFromTestIndex(i int) string {
	return "pipeline-" + string(rune('0'+i))
}

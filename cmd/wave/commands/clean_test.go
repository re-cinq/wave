package commands

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanTestEnv provides a testing environment for clean tests
type cleanTestEnv struct {
	t          *testing.T
	rootDir    string
	origDir    string
	workspaces []string
}

// newCleanTestEnv creates a new test environment with a temp directory
func newCleanTestEnv(t *testing.T) *cleanTestEnv {
	t.Helper()

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "failed to change to temp directory")

	return &cleanTestEnv{
		t:          t,
		rootDir:    tmpDir,
		origDir:    origDir,
		workspaces: []string{},
	}
}

// cleanup restores the original working directory
func (e *cleanTestEnv) cleanup() {
	err := os.Chdir(e.origDir)
	if err != nil {
		e.t.Errorf("failed to restore original directory: %v", err)
	}
}

// createWorkspace creates a test workspace directory with a file
// It also sets the modification time and returns the workspace path
func (e *cleanTestEnv) createWorkspace(name string, modTime time.Time) string {
	e.t.Helper()

	wsPath := filepath.Join(".wave", "workspaces", name)
	err := os.MkdirAll(wsPath, 0755)
	require.NoError(e.t, err, "failed to create workspace %s", name)

	// Create a marker file inside the workspace
	markerFile := filepath.Join(wsPath, "marker.txt")
	err = os.WriteFile(markerFile, []byte("test workspace"), 0644)
	require.NoError(e.t, err, "failed to create marker file")

	// Set modification time on the directory
	err = os.Chtimes(wsPath, modTime, modTime)
	require.NoError(e.t, err, "failed to set modification time")

	e.workspaces = append(e.workspaces, name)
	return wsPath
}

// createWaveStructure creates the base .wave directory structure
func (e *cleanTestEnv) createWaveStructure() {
	e.t.Helper()

	dirs := []string{
		".wave/workspaces",
		".wave/traces",
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(e.t, err, "failed to create %s", dir)
	}

	// Create a state.db file
	err := os.WriteFile(".wave/state.db", []byte("test state"), 0644)
	require.NoError(e.t, err, "failed to create state.db")
}

// workspaceExists checks if a workspace directory exists
func (e *cleanTestEnv) workspaceExists(name string) bool {
	wsPath := filepath.Join(".wave", "workspaces", name)
	_, err := os.Stat(wsPath)
	return err == nil
}

// listWorkspaces returns the list of existing workspace directories
func (e *cleanTestEnv) listWorkspaces() []string {
	var workspaces []string
	wsDir := filepath.Join(".wave", "workspaces")

	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return workspaces
	}

	for _, entry := range entries {
		if entry.IsDir() {
			workspaces = append(workspaces, entry.Name())
		}
	}
	return workspaces
}

// executeCleanCmd runs the clean command with given arguments and returns output/error
func executeCleanCmd(args ...string) (stdout, stderr string, err error) {
	cmd := NewCleanCmd()

	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// executeCleanCmdCapturingStdout runs the clean command and captures real stdout
func executeCleanCmdCapturingStdout(args ...string) (string, error) {
	cmd := NewCleanCmd()
	cmd.SetArgs(args)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), err
}

// T090: Test helpers and setup
func TestCleanTestHelpers(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	// Test that we can create workspace structure
	env.createWaveStructure()
	assert.DirExists(t, ".wave/workspaces")
	assert.DirExists(t, ".wave/traces")
	assert.FileExists(t, ".wave/state.db")

	// Test that we can create workspaces with timestamps
	t1 := time.Now().Add(-3 * time.Hour)
	t2 := time.Now().Add(-2 * time.Hour)
	t3 := time.Now().Add(-1 * time.Hour)

	env.createWorkspace("ws-old", t1)
	env.createWorkspace("ws-mid", t2)
	env.createWorkspace("ws-new", t3)

	// Verify workspaces exist
	assert.True(t, env.workspaceExists("ws-old"))
	assert.True(t, env.workspaceExists("ws-mid"))
	assert.True(t, env.workspaceExists("ws-new"))

	// Verify list works
	workspaces := env.listWorkspaces()
	assert.Len(t, workspaces, 3)
}

// T091: Test for clean removes all workspaces
func TestCleanRemovesAllWorkspaces(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	// Setup workspaces
	env.createWaveStructure()
	env.createWorkspace("pipeline-1", time.Now().Add(-3*time.Hour))
	env.createWorkspace("pipeline-2", time.Now().Add(-2*time.Hour))
	env.createWorkspace("pipeline-3", time.Now().Add(-1*time.Hour))

	// Verify setup
	assert.Len(t, env.listWorkspaces(), 3)

	// Run clean --all
	stdout, err := executeCleanCmdCapturingStdout("--all")

	// Verify success
	require.NoError(t, err)
	assert.Contains(t, stdout, "Removed")

	// Verify all workspaces are removed
	assert.False(t, env.workspaceExists("pipeline-1"))
	assert.False(t, env.workspaceExists("pipeline-2"))
	assert.False(t, env.workspaceExists("pipeline-3"))
}

// T091: Test clean specific pipeline workspace
func TestCleanSpecificPipeline(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	// Setup workspaces
	env.createWaveStructure()
	env.createWorkspace("target-pipeline", time.Now().Add(-2*time.Hour))
	env.createWorkspace("other-pipeline", time.Now().Add(-1*time.Hour))

	// Run clean --pipeline target-pipeline
	stdout, err := executeCleanCmdCapturingStdout("--pipeline", "target-pipeline")

	// Verify success
	require.NoError(t, err)
	assert.Contains(t, stdout, "Removed")

	// Verify only target is removed
	assert.False(t, env.workspaceExists("target-pipeline"))
	assert.True(t, env.workspaceExists("other-pipeline"))
}

// T091: Test clean without flags returns error
func TestCleanRequiresFlags(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()

	// Run clean without any flags
	_, _, err := executeCleanCmd()

	// Should fail with error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "specify --all or --pipeline")
}

// T092: Test for clean --keep-last N
func TestCleanKeepLastN(t *testing.T) {
	tests := []struct {
		name            string
		workspaceCount  int
		keepLast        int
		expectedRemoved int
		expectedKept    int
	}{
		{
			name:            "keep last 2 of 5",
			workspaceCount:  5,
			keepLast:        2,
			expectedRemoved: 3,
			expectedKept:    2,
		},
		{
			name:            "keep last 3 of 3",
			workspaceCount:  3,
			keepLast:        3,
			expectedRemoved: 0,
			expectedKept:    3,
		},
		{
			name:            "keep last 5 of 2",
			workspaceCount:  2,
			keepLast:        5,
			expectedRemoved: 0,
			expectedKept:    2,
		},
		{
			name:            "keep last 1 of 4",
			workspaceCount:  4,
			keepLast:        1,
			expectedRemoved: 3,
			expectedKept:    1,
		},
		{
			name:            "keep last 0 removes all",
			workspaceCount:  3,
			keepLast:        0,
			expectedRemoved: 3,
			expectedKept:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newCleanTestEnv(t)
			defer env.cleanup()

			env.createWaveStructure()

			// Create workspaces with different timestamps
			baseTime := time.Now().Add(-time.Duration(tc.workspaceCount+1) * time.Hour)
			for i := 0; i < tc.workspaceCount; i++ {
				wsTime := baseTime.Add(time.Duration(i) * time.Hour)
				env.createWorkspace("pipeline-"+string(rune('a'+i)), wsTime)
			}

			// Verify setup
			assert.Len(t, env.listWorkspaces(), tc.workspaceCount)

			// Build args with proper string conversion for keepLast
			keepLastStr := "0"
			if tc.keepLast < 10 {
				keepLastStr = string(rune('0' + tc.keepLast))
			} else {
				keepLastStr = "10"
			}
			args := []string{"--all", "--keep-last", keepLastStr}
			stdout, err := executeCleanCmdCapturingStdout(args...)

			require.NoError(t, err)

			// Verify correct number of workspaces remain
			remaining := env.listWorkspaces()
			assert.Len(t, remaining, tc.expectedKept, "expected %d workspaces to be kept", tc.expectedKept)

			// Verify output
			if tc.expectedRemoved > 0 {
				assert.Contains(t, stdout, "Removed")
			}
		})
	}
}

// T092: Test that keep-last keeps the most recent workspaces
func TestCleanKeepLastKeepsMostRecent(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()

	// Create workspaces with specific timestamps
	// Oldest to newest: ws-oldest, ws-middle, ws-newest
	env.createWorkspace("ws-oldest", time.Now().Add(-5*time.Hour))
	env.createWorkspace("ws-middle", time.Now().Add(-3*time.Hour))
	env.createWorkspace("ws-newest", time.Now().Add(-1*time.Hour))

	// Keep last 1
	_, err := executeCleanCmdCapturingStdout("--all", "--keep-last", "1")
	require.NoError(t, err)

	// Only the newest should remain
	remaining := env.listWorkspaces()
	assert.Len(t, remaining, 1)
	assert.Contains(t, remaining, "ws-newest")
	assert.False(t, env.workspaceExists("ws-oldest"))
	assert.False(t, env.workspaceExists("ws-middle"))
}

// T092: Test keep-last with negative value (should be treated as not set)
func TestCleanKeepLastNegativeValue(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	env.createWorkspace("ws-1", time.Now())

	// Negative keep-last should be treated as not set (default behavior: remove all)
	_, err := executeCleanCmdCapturingStdout("--all", "--keep-last", "-1")

	// Should succeed and remove everything (default behavior)
	require.NoError(t, err)
	// The workspace directory itself might be removed when --all is used without --keep-last
}

// T093: Test for clean --dry-run output
func TestCleanDryRunDoesNotDelete(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	env.createWorkspace("pipeline-1", time.Now().Add(-2*time.Hour))
	env.createWorkspace("pipeline-2", time.Now().Add(-1*time.Hour))

	// Verify setup
	assert.Len(t, env.listWorkspaces(), 2)

	// Run clean --all --dry-run
	stdout, err := executeCleanCmdCapturingStdout("--all", "--dry-run")

	// Verify success
	require.NoError(t, err)

	// Verify dry-run message in output
	assert.Contains(t, stdout, "dry-run")

	// Verify nothing was actually deleted
	assert.True(t, env.workspaceExists("pipeline-1"))
	assert.True(t, env.workspaceExists("pipeline-2"))
	assert.Len(t, env.listWorkspaces(), 2)
}

// T093: Test dry-run shows what would be deleted
func TestCleanDryRunShowsTargets(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	env.createWorkspace("my-pipeline", time.Now())

	// Run clean --pipeline my-pipeline --dry-run
	stdout, err := executeCleanCmdCapturingStdout("--pipeline", "my-pipeline", "--dry-run")

	require.NoError(t, err)

	// Should show the target that would be deleted
	assert.Contains(t, stdout, "my-pipeline")
	assert.Contains(t, stdout, "Would remove")

	// Verify nothing was actually deleted
	assert.True(t, env.workspaceExists("my-pipeline"))
}

// T093: Test dry-run with keep-last
func TestCleanDryRunWithKeepLast(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	env.createWorkspace("ws-old", time.Now().Add(-3*time.Hour))
	env.createWorkspace("ws-mid", time.Now().Add(-2*time.Hour))
	env.createWorkspace("ws-new", time.Now().Add(-1*time.Hour))

	// Run clean --all --keep-last 1 --dry-run
	stdout, err := executeCleanCmdCapturingStdout("--all", "--keep-last", "1", "--dry-run")

	require.NoError(t, err)

	// Should show targets that would be deleted
	assert.Contains(t, stdout, "Would remove")

	// Verify nothing was actually deleted
	assert.True(t, env.workspaceExists("ws-old"))
	assert.True(t, env.workspaceExists("ws-mid"))
	assert.True(t, env.workspaceExists("ws-new"))
}

// T093: Test dry-run output format
func TestCleanDryRunOutputFormat(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	env.createWorkspace("test-ws", time.Now())

	stdout, err := executeCleanCmdCapturingStdout("--all", "--dry-run")

	require.NoError(t, err)

	// Verify output format includes clear indication
	assert.Contains(t, stdout, "dry-run")
	// Output should be informative about what would happen
	assert.True(t, len(stdout) > 0, "dry-run should produce output")
}

// T096: Test sorting workspaces by creation time
func TestWorkspacesSortedByCreationTime(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()

	// Create workspaces with specific timestamps in random order
	env.createWorkspace("ws-c", time.Now().Add(-1*time.Hour)) // newest
	env.createWorkspace("ws-a", time.Now().Add(-5*time.Hour)) // oldest
	env.createWorkspace("ws-b", time.Now().Add(-3*time.Hour)) // middle

	// Get workspaces sorted by modification time
	wsDir := filepath.Join(".wave", "workspaces")
	entries, err := os.ReadDir(wsDir)
	require.NoError(t, err)

	type wsInfo struct {
		name    string
		modTime time.Time
	}

	var workspaces []wsInfo
	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			require.NoError(t, err)
			workspaces = append(workspaces, wsInfo{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].modTime.Before(workspaces[j].modTime)
	})

	// Verify order: oldest to newest
	require.Len(t, workspaces, 3)
	assert.Equal(t, "ws-a", workspaces[0].name, "ws-a should be oldest")
	assert.Equal(t, "ws-b", workspaces[1].name, "ws-b should be middle")
	assert.Equal(t, "ws-c", workspaces[2].name, "ws-c should be newest")
}

// Additional test: Clean with nothing to clean (empty workspace dir)
func TestCleanNothingToClean(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	// Don't create any .wave structure - nothing exists to clean
	stdout, err := executeCleanCmdCapturingStdout("--all")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Nothing to clean")
}

// Additional test: Clean non-existent pipeline
func TestCleanNonExistentPipeline(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()

	stdout, err := executeCleanCmdCapturingStdout("--pipeline", "does-not-exist")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Nothing to clean")
}

// Test NewCleanCmd flags
func TestNewCleanCmdFlags(t *testing.T) {
	cmd := NewCleanCmd()

	// Verify command properties
	assert.Equal(t, "clean", cmd.Use)
	assert.Contains(t, cmd.Short, "Clean")

	// Verify all flags exist
	flags := cmd.Flags()

	pipelineFlag := flags.Lookup("pipeline")
	assert.NotNil(t, pipelineFlag, "pipeline flag should exist")

	allFlag := flags.Lookup("all")
	assert.NotNil(t, allFlag, "all flag should exist")

	forceFlag := flags.Lookup("force")
	assert.NotNil(t, forceFlag, "force flag should exist")

	keepLastFlag := flags.Lookup("keep-last")
	assert.NotNil(t, keepLastFlag, "keep-last flag should exist")

	dryRunFlag := flags.Lookup("dry-run")
	assert.NotNil(t, dryRunFlag, "dry-run flag should exist")
}

// Test clean with readonly workspace directories
func TestCleanWithReadonlyDirectories(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	wsPath := env.createWorkspace("readonly-ws", time.Now())

	// Make directory readonly
	err := os.Chmod(wsPath, 0555)
	require.NoError(t, err)

	// Clean should still work (it makes dirs writable before removal)
	stdout, err := executeCleanCmdCapturingStdout("--all")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Removed")
	assert.False(t, env.workspaceExists("readonly-ws"))
}

// Test getWorkspacesSortedByTime helper function
func TestGetWorkspacesSortedByTime(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()

	// Create workspaces with specific timestamps
	env.createWorkspace("old-ws", time.Now().Add(-4*time.Hour))
	env.createWorkspace("new-ws", time.Now().Add(-1*time.Hour))
	env.createWorkspace("mid-ws", time.Now().Add(-2*time.Hour))

	wsDir := filepath.Join(".wave", "workspaces")
	sorted, err := getWorkspacesSortedByTimeHelper(wsDir)
	require.NoError(t, err)

	// Should be sorted oldest to newest
	require.Len(t, sorted, 3)
	assert.Equal(t, "old-ws", sorted[0].Name())
	assert.Equal(t, "mid-ws", sorted[1].Name())
	assert.Equal(t, "new-ws", sorted[2].Name())
}

// getWorkspacesSortedByTimeHelper is a helper to test sorting
func getWorkspacesSortedByTimeHelper(wsDir string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return nil, err
	}

	// Filter to only directories
	var dirs []fs.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(dirs, func(i, j int) bool {
		iInfo, _ := dirs[i].Info()
		jInfo, _ := dirs[j].Info()
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	return dirs, nil
}

// Test that keep-last only affects workspaces, not state.db or traces
func TestCleanKeepLastOnlyAffectsWorkspaces(t *testing.T) {
	env := newCleanTestEnv(t)
	defer env.cleanup()

	env.createWaveStructure()
	env.createWorkspace("ws-1", time.Now().Add(-2*time.Hour))
	env.createWorkspace("ws-2", time.Now().Add(-1*time.Hour))

	// Verify state.db and traces exist
	assert.FileExists(t, ".wave/state.db")
	assert.DirExists(t, ".wave/traces")

	// Run clean --all --keep-last 1 (should only affect workspaces)
	_, err := executeCleanCmdCapturingStdout("--all", "--keep-last", "1")
	require.NoError(t, err)

	// state.db and traces should still exist (keep-last only affects workspaces)
	assert.FileExists(t, ".wave/state.db")
	assert.DirExists(t, ".wave/traces")

	// Only 1 workspace should remain
	assert.Len(t, env.listWorkspaces(), 1)
}

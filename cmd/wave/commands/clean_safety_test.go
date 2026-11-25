package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests reproduce review findings on the workspace move: with a
// misconfigured runtime.workspace_root, clean/cleanup must never delete the
// project, its siblings or other data Wave did not create.

// safetyProject creates <parent>/proj with project files, a sibling
// <parent>/sibling, writes wave.yaml with the given workspace_root and
// chdirs into the project. It returns parent and project.
func safetyProject(t *testing.T, workspaceRoot string) (string, string) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	parent := t.TempDir()
	proj := filepath.Join(parent, "proj")
	for _, d := range []string{"docs", "src/pkg", ".agents/traces", filepath.Join("..", "sibling")} {
		require.NoError(t, os.MkdirAll(filepath.Join(proj, d), 0755))
	}
	for _, f := range []string{"docs/readme.md", "src/main.go", "../sibling/data.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(proj, f), []byte("keep"), 0644))
	}
	manifest := "runtime:\n  workspace_root: " + workspaceRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(proj, "wave.yaml"), []byte(manifest), 0644))
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(proj))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return parent, proj
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t"}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

func assertProjectIntact(t *testing.T, parent, proj string) {
	t.Helper()
	for _, p := range []string{"docs/readme.md", "src/main.go", "wave.yaml"} {
		assert.FileExists(t, filepath.Join(proj, p))
	}
	assert.FileExists(t, filepath.Join(parent, "sibling", "data.txt"))
}

// Repro 1: `wave cleanup --force` with workspace_root ".." matched the main
// worktree by prefix and deleted the whole project via the RemoveAll
// fallback.
func TestCleanupRefusesRootContainingProject(t *testing.T) {
	parent, proj := safetyProject(t, "..")
	gitIn(t, proj, "init", "-q", "-b", "main")
	gitIn(t, proj, "add", "-A")
	gitIn(t, proj, "commit", "-q", "-m", "init")

	err := runCleanup(CleanupOptions{Force: true, Manifest: "wave.yaml"})
	assert.Error(t, err, "a root containing the project must be refused")
	assertProjectIntact(t, parent, proj)
	assert.DirExists(t, filepath.Join(proj, ".git"))
}

// cleanup with a safe root only removes worktrees Wave created there, not a
// user's own worktree that happens to live under the root.
func TestCleanupKeepsForeignWorktreesUnderRoot(t *testing.T) {
	root := t.TempDir()
	parent, proj := safetyProject(t, root)
	gitIn(t, proj, "init", "-q", "-b", "main")
	gitIn(t, proj, "add", "-A")
	gitIn(t, proj, "commit", "-q", "-m", "init")

	mine := filepath.Join(root, "mine")
	gitIn(t, proj, "worktree", "add", "-q", "-b", "mine", mine)
	run := filepath.Join(root, "run-1")
	require.NoError(t, os.MkdirAll(run, 0755))
	require.NoError(t, workspace.MarkRunDir(run))
	wave := filepath.Join(run, "__wt_run-1")
	gitIn(t, proj, "worktree", "add", "-q", "-b", "wave/run-1", wave)

	require.NoError(t, runCleanup(CleanupOptions{Force: true, Manifest: "wave.yaml"}))
	assert.NoDirExists(t, wave, "Wave's own worktree is removed")
	assert.DirExists(t, mine, "a worktree Wave did not create is kept")
	assertProjectIntact(t, parent, proj)
}

// Repro 2: filtered clean skipped the guard; with "." it removed project
// files and .git, with ".." it reached siblings.
func TestCleanFilteredRefusesRootContainingProject(t *testing.T) {
	for _, root := range []string{".", ".."} {
		for _, args := range [][]string{
			{"--all", "--keep-last", "0", "--force"},
			{"--older-than", "1s", "--force"},
			{"--status", "completed", "--force"},
		} {
			t.Run(root+" "+args[0], func(t *testing.T) {
				parent, proj := safetyProject(t, root)
				require.NoError(t, os.MkdirAll(filepath.Join(proj, ".git"), 0755))
				old := time.Now().Add(-48 * time.Hour)
				for _, p := range []string{filepath.Join(proj, "docs"), filepath.Join(proj, "src"), filepath.Join(parent, "sibling")} {
					require.NoError(t, os.Chtimes(p, old, old))
				}
				_, err := executeCleanCmdCapturingStdout(args...)
				assert.Error(t, err)
				assertProjectIntact(t, parent, proj)
				assert.DirExists(t, filepath.Join(proj, ".git"))
				assert.DirExists(t, filepath.Join(proj, ".agents"))
			})
		}
	}
}

// Repro 3: --pipeline joined the name onto the root without containment.
func TestCleanPipelineStaysInsideRoot(t *testing.T) {
	t.Run("dotdot root, sibling", func(t *testing.T) {
		parent, proj := safetyProject(t, "..")
		_, err := executeCleanCmdCapturingStdout("--pipeline", "sibling", "--force")
		assert.Error(t, err)
		assertProjectIntact(t, parent, proj)
	})
	t.Run("dot root, .git", func(t *testing.T) {
		parent, proj := safetyProject(t, ".")
		require.NoError(t, os.MkdirAll(filepath.Join(proj, ".git"), 0755))
		_, err := executeCleanCmdCapturingStdout("--pipeline", ".git", "--force")
		assert.Error(t, err)
		assert.DirExists(t, filepath.Join(proj, ".git"))
		assertProjectIntact(t, parent, proj)
	})
	t.Run("safe root, unmarked dir", func(t *testing.T) {
		root := t.TempDir()
		parent, proj := safetyProject(t, root)
		require.NoError(t, os.MkdirAll(filepath.Join(root, "foreign"), 0755))
		_, err := executeCleanCmdCapturingStdout("--pipeline", "foreign", "--force")
		require.NoError(t, err)
		assert.DirExists(t, filepath.Join(root, "foreign"))
		assertProjectIntact(t, parent, proj)
	})
}

// Repro 4: bare --all removed any root that was not the project or an
// ancestor: workspace_root "src" deleted src, an outside dir was removed
// wholesale.
func TestCleanAllOnlyRemovesWaveRunDirs(t *testing.T) {
	t.Run("src root", func(t *testing.T) {
		parent, proj := safetyProject(t, "src")
		_, err := executeCleanCmdCapturingStdout("--all", "--force")
		require.NoError(t, err)
		assertProjectIntact(t, parent, proj)
		assert.DirExists(t, filepath.Join(proj, "src", "pkg"))
	})
	t.Run("absolute outside root", func(t *testing.T) {
		root := t.TempDir()
		parent, proj := safetyProject(t, root)
		require.NoError(t, os.MkdirAll(filepath.Join(root, "keep"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "keep", "f.txt"), []byte("x"), 0644))
		run := filepath.Join(root, "run-1")
		require.NoError(t, os.MkdirAll(run, 0755))
		require.NoError(t, workspace.MarkRunDir(run))

		_, err := executeCleanCmdCapturingStdout("--all", "--force")
		require.NoError(t, err)
		assert.NoDirExists(t, run, "Wave's run dir is removed")
		assert.FileExists(t, filepath.Join(root, "keep", "f.txt"))
		assert.DirExists(t, root, "the root itself is never removed")
		assertProjectIntact(t, parent, proj)
	})
}

// wave validate rejects a workspace_root that equals or contains the
// project.
func TestValidateRejectsRootContainingProject(t *testing.T) {
	for _, root := range []string{".", ".."} {
		t.Run(root, func(t *testing.T) {
			_, proj := safetyProject(t, root)
			m := "apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: p\nruntime:\n  workspace_root: \"" + root + "\"\n"
			require.NoError(t, os.WriteFile(filepath.Join(proj, "wave.yaml"), []byte(m), 0644))
			cmd := NewValidateCmd()
			cmd.SetArgs([]string{})
			assert.Error(t, cmd.Execute())
		})
	}
}

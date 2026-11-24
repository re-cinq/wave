package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/testutil"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestWarnOnUnexpectedArtifacts_CleanWorktree guards against a worktree
// step that changed nothing being reported as having written every tracked
// file: an empty git status must not fall back to walking the checkout.
func TestWarnOnUnexpectedArtifacts_CleanWorktree(t *testing.T) {
	ws := t.TempDir()
	gitRun(t, ws, "init", "-q")
	for _, f := range []string{"README.md", "src/main.go"} {
		p := filepath.Join(ws, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, ws, "add", ".")
	gitRun(t, ws, "commit", "-q", "-m", "init")

	if got := changedFilesViaGit(ws); got == nil || len(got) != 0 {
		t.Fatalf("changedFilesViaGit on a clean tree = %#v, want an empty non-nil slice", got)
	}

	collector := testutil.NewEventCollector()
	e := NewDefaultPipelineExecutor(nil, WithEmitter(collector))
	execution := &PipelineExecution{Status: &PipelineStatus{ID: "run-1"}}
	e.warnOnUnexpectedArtifacts(execution, &Step{ID: "solve"}, ws)
	for _, ev := range collector.GetEvents() {
		if strings.Contains(ev.Message, "outside declared output_artifacts") {
			t.Errorf("clean worktree produced warning: %s", ev.Message)
		}
	}

	// A real change is still reported.
	if err := os.WriteFile(filepath.Join(ws, "src", "main.go"), []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.warnOnUnexpectedArtifacts(execution, &Step{ID: "solve"}, ws)
	var warned bool
	for _, ev := range collector.GetEvents() {
		if strings.Contains(ev.Message, "1 file(s) outside declared output_artifacts paths: src/main.go") {
			warned = true
		}
	}
	if !warned {
		t.Error("a modified tracked file should still be reported")
	}
}

package bench

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeWave writes a shell script standing in for the wave binary. It
// records its arguments, and on `run` edits hello.txt and emits the NDJSON
// completion event with a patch deliverable produced by `git diff`.
func writeFakeWave(t *testing.T, dir string, runBody string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-wave")
	script := `#!/bin/sh
echo "$@" >> "` + filepath.Join(dir, "calls.log") + `"
case "$1" in
  init) echo "kind: WaveManifest" > wave.yaml; exit 0 ;;
  run) ` + runBody + ` ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

const fakeSolve = `echo "hello fixed" > hello.txt
mkdir -p .agents/output
git diff > .agents/output/model.patch
echo '{"pipeline_id":"bench-solve-1","state":"running"}'
echo '{"pipeline_id":"bench-solve-1","state":"completed","outcomes":{"deliverables":[{"type":"file","name":"patch","path":"'$PWD'/.agents/output/model.patch"}]}}'
exit 0`

func TestSubprocessRunner_WaveModeCapturesPatch(t *testing.T) {
	dir := t.TempDir()
	rc, sha := seedCache(t, dir)
	r := &subprocessRunner{cache: rc}
	cfg := RunConfig{
		Pipeline: "bench-solve",
		Adapter:  "codex",
		Model:    "gpt-test",
		WorkDir:  filepath.Join(dir, "work"),
		WaveBin:  writeFakeWave(t, dir, fakeSolve),
	}
	task := BenchTask{ID: "test__repo-1", Repo: "test/repo", BaseCommit: sha, Problem: "fix hello", TestPatch: "SECRET"}

	res, err := r.RunTask(context.Background(), task, cfg)
	if err != nil {
		t.Fatalf("RunTask() error = %v", err)
	}
	if res.Status != StatusPatched {
		t.Fatalf("Status = %q (%s), want %q: a patch without verification is not a pass", res.Status, res.Error, StatusPatched)
	}
	if res.RunID != "bench-solve-1" {
		t.Errorf("RunID = %q, want bench-solve-1", res.RunID)
	}
	if !strings.Contains(res.PatchDiff, "+hello fixed") {
		t.Errorf("PatchDiff = %q, want the solver's change", res.PatchDiff)
	}
	calls, _ := os.ReadFile(filepath.Join(dir, "calls.log"))
	for _, want := range []string{"init --yes --force --all --adapter codex", "run bench-solve --json --no-retro --adapter codex --model gpt-test --force-model -- fix hello"} {
		if !strings.Contains(string(calls), want) {
			t.Errorf("wave calls missing %q:\n%s", want, calls)
		}
	}
	if strings.Contains(string(calls), "SECRET") {
		t.Errorf("the hidden test patch leaked to the solver:\n%s", calls)
	}
	if _, err := os.Stat(filepath.Join(cfg.WorkDir, task.ID, "src")); !os.IsNotExist(err) {
		t.Errorf("worktree should be removed without KeepWorkspaces")
	}
}

func TestSubprocessRunner_WaveModeErrors(t *testing.T) {
	tests := []struct {
		name    string
		runBody string
		wantErr string
	}{
		{"pipeline fails", `echo boom >&2; exit 1`, "wave run bench-solve"},
		{"no patch artifact", `echo '{"pipeline_id":"p","state":"completed","outcomes":{"deliverables":[]}}'; exit 0`, `declared no "patch" output artifact`},
		{"empty patch", `: > empty.patch; echo '{"pipeline_id":"p","outcomes":{"deliverables":[{"name":"patch","path":"'$PWD'/empty.patch"}]}}'; exit 0`, "solver produced no patch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			rc, sha := seedCache(t, dir)
			r := &subprocessRunner{cache: rc}
			cfg := RunConfig{Pipeline: "bench-solve", WorkDir: filepath.Join(dir, "work"), WaveBin: writeFakeWave(t, dir, tt.runBody)}
			res, err := r.RunTask(context.Background(), BenchTask{ID: "t-1", Repo: "test/repo", BaseCommit: sha, Problem: "p"}, cfg)
			if err != nil {
				t.Fatalf("RunTask() error = %v", err)
			}
			if res.Status != StatusError || !strings.Contains(res.Error, tt.wantErr) {
				t.Errorf("got %q %q, want error containing %q", res.Status, res.Error, tt.wantErr)
			}
		})
	}
}

func TestSubprocessRunner_TestCommandAppliesTestPatchAfterSolve(t *testing.T) {
	dir := t.TempDir()
	rc, sha := seedCache(t, dir)
	r := &subprocessRunner{cache: rc}
	cfg := RunConfig{Pipeline: "bench-solve", WorkDir: filepath.Join(dir, "work"), WaveBin: writeFakeWave(t, dir, fakeSolve)}
	testPatch := "--- /dev/null\n+++ b/check.sh\n@@ -0,0 +1 @@\n+grep -q fixed hello.txt\n"

	res, err := r.RunTask(context.Background(), BenchTask{ID: "t-2", Repo: "test/repo", BaseCommit: sha, Problem: "p", TestPatch: testPatch, TestCommand: "sh check.sh"}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusPass {
		t.Fatalf("Status = %q (%s), want pass", res.Status, res.Error)
	}
	if strings.Contains(res.PatchDiff, "check.sh") {
		t.Errorf("the test patch must not be part of the model patch:\n%s", res.PatchDiff)
	}

	res, err = r.RunTask(context.Background(), BenchTask{ID: "t-3", Repo: "test/repo", BaseCommit: sha, Problem: "p", TestCommand: "false"}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusFail {
		t.Errorf("Status = %q, want fail", res.Status)
	}
}

func TestSubprocessRunner_RejectsUnsafeTaskID(t *testing.T) {
	r := NewSubprocessRunner(t.TempDir())
	if _, err := r.RunTask(context.Background(), BenchTask{ID: "../escape"}, RunConfig{WorkDir: t.TempDir()}); err == nil {
		t.Fatal("expected an error for a path-traversing task id")
	}
}

func TestParseWaveEvents(t *testing.T) {
	out := []byte("not json\n" +
		`{"pipeline_id":"run-1","state":"running"}` + "\n" +
		`{"pipeline_id":"run-1","state":"completed","outcomes":{"deliverables":[{"name":"other","path":"/x"},{"name":"patch","path":"/p/model.patch"}]}}` + "\n")
	runID, patch := parseWaveEvents(out)
	if runID != "run-1" || patch != "/p/model.patch" {
		t.Errorf("parseWaveEvents = %q, %q", runID, patch)
	}
}

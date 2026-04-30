package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runGit runs git in dir and returns nothing — fatal-fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// initRepoWithTest seeds a git repo containing one *_test.go file with two
// test functions, then commits. Returns the dir path.
func initRepoWithTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	writeFile(t, dir, "x_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
func TestBeta(t *testing.T) { _ = t }
`)
	runGit(t, dir, "add", "x_test.go")
	runGit(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func TestTestDiff_NoChanges_Passes(t *testing.T) {
	dir := initRepoWithTest(t)
	v := &testDiffValidator{}
	if err := v.Validate(ContractConfig{Type: "test_diff"}, dir); err != nil {
		t.Fatalf("expected no error on clean diff, got: %v", err)
	}
}

func TestTestDiff_Addition_Passes(t *testing.T) {
	dir := initRepoWithTest(t)
	writeFile(t, dir, "x_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
func TestBeta(t *testing.T) { _ = t }
func TestGamma(t *testing.T) { _ = t }
`)
	v := &testDiffValidator{}
	if err := v.Validate(ContractConfig{Type: "test_diff"}, dir); err != nil {
		t.Fatalf("expected no error when test added, got: %v", err)
	}
}

func TestTestDiff_NetDeletion_Fails(t *testing.T) {
	dir := initRepoWithTest(t)
	writeFile(t, dir, "x_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
`)
	v := &testDiffValidator{}
	err := v.Validate(ContractConfig{Type: "test_diff"}, dir)
	if err == nil {
		t.Fatal("expected error for net deletion, got nil")
	}
}

func TestTestDiff_RenameNets_Passes(t *testing.T) {
	dir := initRepoWithTest(t)
	writeFile(t, dir, "x_test.go", `package x

import "testing"

func TestAlphaRenamed(t *testing.T) { _ = t }
func TestBeta(t *testing.T) { _ = t }
`)
	v := &testDiffValidator{}
	if err := v.Validate(ContractConfig{Type: "test_diff"}, dir); err != nil {
		t.Fatalf("expected rename to net to zero, got: %v", err)
	}
}

func TestTestDiff_HigherToleranceConfig(t *testing.T) {
	dir := initRepoWithTest(t)
	writeFile(t, dir, "x_test.go", `package x
`)
	v := &testDiffValidator{}
	cfg := ContractConfig{Type: "test_diff", MaxTestDeletions: 2}
	if err := v.Validate(cfg, dir); err != nil {
		t.Fatalf("expected pass with MaxTestDeletions=2, got: %v", err)
	}
}

func TestTestDiff_PythonConfig_DetectsDeletion(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	writeFile(t, dir, "test_things.py", `def test_alpha():
    assert True

def test_beta():
    assert True
`)
	runGit(t, dir, "add", "test_things.py")
	runGit(t, dir, "commit", "-q", "-m", "init")
	writeFile(t, dir, "test_things.py", `def test_alpha():
    assert True
`)
	v := &testDiffValidator{}
	cfg := ContractConfig{
		Type:            "test_diff",
		TestFilePattern: []string{"test_*.py", "*_test.py"},
		TestFuncPattern: `^[ \t]*def[ \t]+test_\w+`,
	}
	if err := v.Validate(cfg, dir); err == nil {
		t.Fatal("expected error for python net deletion, got nil")
	}
}

func TestTestDiff_JavaScriptConfig_DetectsDeletion(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	writeFile(t, dir, "x.test.ts", `it("alpha", () => {});
it("beta", () => {});
`)
	runGit(t, dir, "add", "x.test.ts")
	runGit(t, dir, "commit", "-q", "-m", "init")
	writeFile(t, dir, "x.test.ts", `it("alpha", () => {});
`)
	v := &testDiffValidator{}
	cfg := ContractConfig{
		Type:            "test_diff",
		TestFilePattern: []string{"*.test.ts", "*.test.js", "*.spec.ts", "*.spec.js"},
		TestFuncPattern: `^[ \t]*(it|test)\(`,
	}
	if err := v.Validate(cfg, dir); err == nil {
		t.Fatal("expected error for js net deletion, got nil")
	}
}

func TestTestDiff_NoGitRepo_PassesSilently(t *testing.T) {
	dir := t.TempDir()
	v := &testDiffValidator{}
	if err := v.Validate(ContractConfig{Type: "test_diff"}, dir); err != nil {
		t.Fatalf("expected silent pass without git, got: %v", err)
	}
}

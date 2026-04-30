package contract

import (
	"testing"
)

// initRepoTwoCommits seeds a repo with one *_test.go (2 tests), commits,
// then optionally mutates and recommits. Returns the dir.
func initRepoTwoCommits(t *testing.T, mutate func(dir string)) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	writeFile(t, dir, "x_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
func TestBeta(t *testing.T) { _ = t }
`)
	runGit(t, dir, "add", "x_test.go")
	runGit(t, dir, "commit", "-q", "-m", "base")
	if mutate != nil {
		mutate(dir)
		runGit(t, dir, "add", "-A")
		runGit(t, dir, "commit", "-q", "-m", "head")
	}
	return dir
}

func TestTestCountBaseline_NoChange_Passes(t *testing.T) {
	dir := initRepoTwoCommits(t, func(d string) {
		writeFile(t, d, "y.go", `package x
`)
	})
	v := &testCountBaselineValidator{}
	if err := v.Validate(ContractConfig{Type: "test_count_baseline"}, dir); err != nil {
		t.Fatalf("expected pass, got: %v", err)
	}
}

func TestTestCountBaseline_Addition_Passes(t *testing.T) {
	dir := initRepoTwoCommits(t, func(d string) {
		writeFile(t, d, "x_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
func TestBeta(t *testing.T) { _ = t }
func TestGamma(t *testing.T) { _ = t }
`)
	})
	v := &testCountBaselineValidator{}
	if err := v.Validate(ContractConfig{Type: "test_count_baseline"}, dir); err != nil {
		t.Fatalf("expected pass on addition, got: %v", err)
	}
}

func TestTestCountBaseline_Deletion_Fails(t *testing.T) {
	dir := initRepoTwoCommits(t, func(d string) {
		writeFile(t, d, "x_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
`)
	})
	v := &testCountBaselineValidator{}
	if err := v.Validate(ContractConfig{Type: "test_count_baseline"}, dir); err == nil {
		t.Fatal("expected error on deletion, got nil")
	}
}

func TestTestCountBaseline_FileMove_NetsZero(t *testing.T) {
	dir := initRepoTwoCommits(t, func(d string) {
		// Delete x_test.go, recreate same tests under different filename.
		runGit(t, d, "rm", "-q", "x_test.go")
		writeFile(t, d, "renamed_test.go", `package x

import "testing"

func TestAlpha(t *testing.T) { _ = t }
func TestBeta(t *testing.T) { _ = t }
`)
	})
	v := &testCountBaselineValidator{}
	if err := v.Validate(ContractConfig{Type: "test_count_baseline"}, dir); err != nil {
		t.Fatalf("expected pass on file move, got: %v", err)
	}
}

func TestTestCountBaseline_HigherTolerance_Passes(t *testing.T) {
	dir := initRepoTwoCommits(t, func(d string) {
		writeFile(t, d, "x_test.go", `package x
`)
	})
	v := &testCountBaselineValidator{}
	cfg := ContractConfig{Type: "test_count_baseline", MaxTestDeletions: 2}
	if err := v.Validate(cfg, dir); err != nil {
		t.Fatalf("expected pass with tolerance=2, got: %v", err)
	}
}

func TestTestCountBaseline_NoBaseRef_PassesSilently(t *testing.T) {
	// Single-commit repo — HEAD~1 doesn't resolve.
	dir := initRepoTwoCommits(t, nil)
	v := &testCountBaselineValidator{}
	if err := v.Validate(ContractConfig{Type: "test_count_baseline"}, dir); err != nil {
		t.Fatalf("expected silent pass without base ref, got: %v", err)
	}
}

func TestTestCountBaseline_PythonConfig_DetectsDeletion(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	writeFile(t, dir, "test_things.py", `def test_alpha():
    pass

def test_beta():
    pass
`)
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "base")
	writeFile(t, dir, "test_things.py", `def test_alpha():
    pass
`)
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "head")
	v := &testCountBaselineValidator{}
	cfg := ContractConfig{
		Type:            "test_count_baseline",
		TestFilePattern: []string{"test_*.py", "*_test.py"},
		TestFuncPattern: `(?m)^[ \t]*def[ \t]+test_\w+`,
	}
	if err := v.Validate(cfg, dir); err == nil {
		t.Fatal("expected error for python deletion, got nil")
	}
}

func TestTestCountBaseline_NoGit_PassesSilently(t *testing.T) {
	dir := t.TempDir()
	v := &testCountBaselineValidator{}
	if err := v.Validate(ContractConfig{Type: "test_count_baseline"}, dir); err != nil {
		t.Fatalf("expected silent pass without git, got: %v", err)
	}
}

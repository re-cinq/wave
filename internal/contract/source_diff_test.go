package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a temporary git repository for testing.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	// Create initial commit so HEAD exists
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("init\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "init")

	return dir
}

// addAndCommit creates a file with content and commits it.
func addAndCommit(t *testing.T, dir, relPath, content string) {
	t.Helper()
	path := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", relPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "add "+relPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// stageFile writes a file and stages it with git add, so it appears in `git diff HEAD`.
func stageFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	path := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", relPath)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add %s: %v\n%s", relPath, err, out)
	}
}

func TestSourceDiffValidator(t *testing.T) {
	v := &sourceDiffValidator{}

	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		cfg     ContractConfig
		wantErr bool
	}{
		{
			name:  "no diff fails with min_files=1",
			setup: func(t *testing.T, dir string) {}, // nothing changed
			cfg: ContractConfig{
				Type:     "source_diff",
				MinFiles: 1,
			},
			wantErr: true,
		},
		{
			name: "diff with matching file passes",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "main.go", "package main\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				Glob:     "*.go",
				MinFiles: 1,
			},
			wantErr: false,
		},
		{
			name: "diff with only excluded files fails",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "specs/foo.md", "# spec\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				MinFiles: 1,
				Exclude:  []string{"specs/**"},
			},
			wantErr: true,
		},
		{
			name: "empty glob matches all files",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "internal/foo.go", "package internal\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				MinFiles: 1,
				// Glob is empty — matches all
			},
			wantErr: false,
		},
		{
			name: "glob non-match fails",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "README.md", "updated\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				Glob:     "*.go",
				MinFiles: 1,
			},
			wantErr: true,
		},
		{
			name: "min_files=0 treated as 1, fails on no diff",
			setup: func(t *testing.T, dir string) {}, // nothing changed
			cfg: ContractConfig{
				Type:     "source_diff",
				MinFiles: 0,
			},
			wantErr: true,
		},
		{
			name: "min_files=2 requires two files",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "a.go", "package a\n")
				stageFile(t, dir, "b.go", "package b\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				Glob:     "*.go",
				MinFiles: 2,
			},
			wantErr: false,
		},
		{
			name: "min_files=2 fails with only one file",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "a.go", "package a\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				Glob:     "*.go",
				MinFiles: 2,
			},
			wantErr: true,
		},
		{
			name: "exclude .wave files",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, ".wave/output/foo.json", "{}")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				MinFiles: 1,
				Exclude:  []string{".wave/**"},
			},
			wantErr: true,
		},
		{
			name: "mix of excluded and qualifying files — qualifying wins",
			setup: func(t *testing.T, dir string) {
				stageFile(t, dir, "specs/foo.md", "# spec\n")
				stageFile(t, dir, "internal/bar.go", "package internal\n")
			},
			cfg: ContractConfig{
				Type:     "source_diff",
				MinFiles: 1,
				Exclude:  []string{"specs/**"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initGitRepo(t)
			tt.setup(t, dir)

			err := v.Validate(tt.cfg, dir)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

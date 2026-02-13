package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// initTestRepo creates a temporary git repository for testing.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run %v: %v\n%s", args, err, out)
		}
	}

	// Create an initial commit (worktree requires at least one commit)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", dir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "initial")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}

	return dir
}

func TestNewManager(t *testing.T) {
	dir := initTestRepo(t)

	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	// RepoRoot returns the canonical path, which should resolve to the same directory
	if mgr.RepoRoot() == "" {
		t.Error("expected non-empty repo root")
	}
}

func TestNewManager_NotARepo(t *testing.T) {
	dir := t.TempDir()

	_, err := NewManager(dir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestCreateAndRemove(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	worktreePath := filepath.Join(t.TempDir(), "my-worktree")

	// Create worktree with new branch
	if err := mgr.Create(ctx, worktreePath, "test-branch"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify worktree exists and has files
	readmePath := filepath.Join(worktreePath, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("expected README.md in worktree, got error: %v", err)
	}

	// Verify it's on the right branch
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get branch: %v", err)
	}
	branch := string(out[:len(out)-1]) // trim newline
	if branch != "test-branch" {
		t.Errorf("expected branch 'test-branch', got %q", branch)
	}

	// Remove worktree
	if err := mgr.Remove(ctx, worktreePath); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}
}

func TestCreateExistingBranch(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a branch first
	cmd := exec.Command("git", "-C", dir, "branch", "existing-branch")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create branch: %v\n%s", err, out)
	}

	ctx := context.Background()
	worktreePath := filepath.Join(t.TempDir(), "existing-wt")

	// Create worktree using existing branch
	if err := mgr.Create(ctx, worktreePath, "existing-branch"); err != nil {
		t.Fatalf("Create failed for existing branch: %v", err)
	}

	// Cleanup
	mgr.Remove(ctx, worktreePath)
}

func TestCreate_EmptyPath(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := mgr.Create(ctx, "", "test-branch"); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestCreate_EmptyBranch(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := mgr.Create(ctx, "/tmp/test-wt", ""); err == nil {
		t.Fatal("expected error for empty branch")
	}
}

func TestRemove_EmptyPath(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := mgr.Remove(ctx, ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestRemoveDirtyWorktree(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	worktreePath := filepath.Join(t.TempDir(), "dirty-wt")
	if err := mgr.Create(ctx, worktreePath, "dirty-branch"); err != nil {
		t.Fatal(err)
	}

	// Make the worktree dirty
	if err := os.WriteFile(filepath.Join(worktreePath, "dirty-file.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}

	// Force removal should still work
	if err := mgr.Remove(ctx, worktreePath); err != nil {
		t.Fatalf("Remove failed for dirty worktree: %v", err)
	}
}

func TestConcurrentWorktreeCreation(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	errCh := make(chan error, 3)
	paths := make([]string, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		i := i
		paths[i] = filepath.Join(t.TempDir(), "concurrent-wt")
		go func() {
			defer wg.Done()
			branchName := filepath.Base(paths[i]) + "-" + string(rune('a'+i))
			if err := mgr.Create(ctx, paths[i], branchName); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent create failed: %v", err)
	}

	// Cleanup
	for _, p := range paths {
		mgr.Remove(ctx, p)
	}
}

// TestConcurrentCrossManagerWorktree verifies that 10 Manager instances on the same repo
// doing create/remove concurrently produce zero git errors (T010/SC-001).
func TestConcurrentCrossManagerWorktree(t *testing.T) {
	dir := initTestRepo(t)

	const managers = 10
	var wg sync.WaitGroup
	errCh := make(chan error, managers*2)

	for i := 0; i < managers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()

			mgr, err := NewManager(dir)
			if err != nil {
				errCh <- err
				return
			}

			ctx := context.Background()
			wtPath := filepath.Join(t.TempDir(), "cross-mgr-wt")
			branchName := "cross-mgr-" + string(rune('a'+i))

			if err := mgr.Create(ctx, wtPath, branchName); err != nil {
				errCh <- err
				return
			}

			if err := mgr.Remove(ctx, wtPath); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("cross-manager concurrent operation failed: %v", err)
	}
}

func TestNewManager_WithLockTimeout(t *testing.T) {
	dir := initTestRepo(t)

	mgr, err := NewManager(dir, WithLockTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewManager with option failed: %v", err)
	}

	if mgr.lockTimeout != 5*time.Second {
		t.Errorf("expected lockTimeout 5s, got %v", mgr.lockTimeout)
	}
}

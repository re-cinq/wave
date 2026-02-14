package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
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
	if mgr.RepoRoot() != dir {
		t.Errorf("expected repo root %q, got %q", dir, mgr.RepoRoot())
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

	worktreePath := filepath.Join(t.TempDir(), "my-worktree")

	// Create worktree with new branch
	if err := mgr.Create(worktreePath, "test-branch", ""); err != nil {
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
	if err := mgr.Remove(worktreePath); err != nil {
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

	worktreePath := filepath.Join(t.TempDir(), "existing-wt")

	// Create worktree using existing branch
	if err := mgr.Create(worktreePath, "existing-branch", ""); err != nil {
		t.Fatalf("Create failed for existing branch: %v", err)
	}

	// Cleanup
	mgr.Remove(worktreePath)
}

func TestCreateWithBase(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	worktreePath := filepath.Join(t.TempDir(), "base-wt")

	// Create worktree with new branch from a specific base
	// Use HEAD as the base since we only have one commit
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	headRef := string(out[:len(out)-1])

	if err := mgr.Create(worktreePath, "base-branch", headRef); err != nil {
		t.Fatalf("Create with base failed: %v", err)
	}

	// Verify worktree is on the right branch
	branchCmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		t.Fatalf("failed to get branch: %v", err)
	}
	branch := string(branchOut[:len(branchOut)-1])
	if branch != "base-branch" {
		t.Errorf("expected branch 'base-branch', got %q", branch)
	}

	// Cleanup
	mgr.Remove(worktreePath)
}

func TestCreateDetachedWithBase(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	worktreePath := filepath.Join(t.TempDir(), "detached-wt")

	// Create a second commit so we can verify the base ref
	testFile := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(testFile, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	addCmd := exec.Command("git", "-C", dir, "add", ".")
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	commitCmd := exec.Command("git", "-C", dir, "commit", "-m", "second commit")
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}

	// Get the first commit hash to use as base
	logCmd := exec.Command("git", "-C", dir, "rev-list", "--max-parents=0", "HEAD")
	logOut, err := logCmd.Output()
	if err != nil {
		t.Fatalf("failed to get first commit: %v", err)
	}
	firstCommit := string(logOut[:len(logOut)-1])

	// Create detached HEAD worktree (empty branch, base set)
	if err := mgr.Create(worktreePath, "", firstCommit); err != nil {
		t.Fatalf("Create detached failed: %v", err)
	}

	// Verify worktree is in detached HEAD state
	branchCmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		t.Fatalf("failed to get branch: %v", err)
	}
	branch := string(branchOut[:len(branchOut)-1])
	if branch != "HEAD" {
		t.Errorf("expected detached HEAD, got %q", branch)
	}

	// Verify it's at the right commit
	revCmd := exec.Command("git", "-C", worktreePath, "rev-parse", "HEAD")
	revOut, err := revCmd.Output()
	if err != nil {
		t.Fatalf("failed to get HEAD rev: %v", err)
	}
	headRev := string(revOut[:len(revOut)-1])
	if headRev != firstCommit {
		t.Errorf("expected HEAD at %s, got %s", firstCommit, headRev)
	}

	// Verify the worktree only has files from the first commit (README.md, no second.txt)
	if _, err := os.Stat(filepath.Join(worktreePath, "second.txt")); !os.IsNotExist(err) {
		t.Error("detached HEAD worktree should not have second.txt (it was committed after the base)")
	}

	// Cleanup
	mgr.Remove(worktreePath)
}

func TestCreate_EmptyPath(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Create("", "test-branch", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestCreate_EmptyBranch(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.Create("/tmp/test-wt", "", "")
	if err == nil {
		t.Fatal("expected error for empty branch and empty base")
	}
	if err.Error() != "branch name or base ref is required" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRemove_EmptyPath(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Remove(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestRemoveDirtyWorktree(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	worktreePath := filepath.Join(t.TempDir(), "dirty-wt")
	if err := mgr.Create(worktreePath, "dirty-branch", ""); err != nil {
		t.Fatal(err)
	}

	// Make the worktree dirty
	if err := os.WriteFile(filepath.Join(worktreePath, "dirty-file.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}

	// Force removal should still work
	if err := mgr.Remove(worktreePath); err != nil {
		t.Fatalf("Remove failed for dirty worktree: %v", err)
	}
}

func TestConcurrentWorktreeCreation(t *testing.T) {
	dir := initTestRepo(t)
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

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
			if err := mgr.Create(paths[i], branchName, ""); err != nil {
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
		mgr.Remove(p)
	}
}

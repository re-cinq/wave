package bench

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initBareRepo creates a local bare repo with one commit for testing.
// Returns the path to the bare repo and the commit SHA.
func initBareRepo(t *testing.T, dir string) (barePath string, commitSHA string) {
	t.Helper()

	// Create a normal repo with a commit.
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"git", "init", "-q", srcDir},
		{"git", "-C", srcDir, "config", "user.email", "test@test.com"},
		{"git", "-C", srcDir, "config", "user.name", "Test"},
		{"git", "-C", srcDir, "config", "commit.gpgsign", "false"},
		{"git", "-C", srcDir, "config", "tag.gpgsign", "false"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

	// Add a file and commit.
	if err := os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "-C", srcDir, "add", "."},
		{"git", "-C", srcDir, "commit", "-q", "-m", "initial"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

	// Get commit SHA.
	out, err := exec.Command("git", "-C", srcDir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := string(out[:len(out)-1]) // trim newline

	// Clone as bare.
	barePath = filepath.Join(dir, "bare.git")
	if out, err := exec.Command("git", "clone", "--bare", "-q", srcDir, barePath).CombinedOutput(); err != nil {
		t.Fatalf("bare clone: %v\n%s", err, out)
	}

	return barePath, sha
}

func TestRepoCache_ClonePath(t *testing.T) {
	rc := &RepoCache{CacheDir: "/tmp/cache"}
	got := rc.clonePath("django/django")
	want := "/tmp/cache/django__django"
	if got != want {
		t.Errorf("clonePath = %q, want %q", got, want)
	}
}

func TestRepoCache_EnsureCloned_LocalFetch(t *testing.T) {
	dir := t.TempDir()
	barePath, _ := initBareRepo(t, dir)

	// Pre-populate the cache with our local bare repo.
	rc := &RepoCache{CacheDir: dir}
	cachePath := rc.clonePath("test/repo")

	// Copy bare repo to cache location.
	if out, err := exec.Command("cp", "-r", barePath, cachePath).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v\n%s", err, out)
	}

	ctx := context.Background()
	got, err := rc.EnsureCloned(ctx, "test/repo")
	if err != nil {
		t.Fatalf("EnsureCloned() error = %v", err)
	}
	if got != cachePath {
		t.Errorf("EnsureCloned() = %q, want %q", got, cachePath)
	}
}

func TestRepoCache_PrepareWorktree(t *testing.T) {
	dir := t.TempDir()
	barePath, sha := initBareRepo(t, dir)

	// Set up cache with the bare repo.
	rc := &RepoCache{CacheDir: dir}
	cachePath := rc.clonePath("test/repo")
	if out, err := exec.Command("cp", "-r", barePath, cachePath).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v\n%s", err, out)
	}

	ctx := context.Background()
	wtPath := filepath.Join(dir, "worktree")

	err := rc.PrepareWorktree(ctx, "test/repo", sha, wtPath, "")
	if err != nil {
		t.Fatalf("PrepareWorktree() error = %v", err)
	}

	// Verify worktree has the file.
	content, err := os.ReadFile(filepath.Join(wtPath, "hello.txt"))
	if err != nil {
		t.Fatalf("read hello.txt: %v", err)
	}
	if string(content) != "hello\n" {
		t.Errorf("hello.txt = %q, want %q", content, "hello\n")
	}

	// Clean up worktree.
	err = rc.RemoveWorktree(ctx, "test/repo", wtPath)
	if err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree dir should be removed")
	}
}

func TestRepoCache_PrepareWorktree_WithPatch(t *testing.T) {
	dir := t.TempDir()
	barePath, sha := initBareRepo(t, dir)

	rc := &RepoCache{CacheDir: dir}
	cachePath := rc.clonePath("test/repo")
	if out, err := exec.Command("cp", "-r", barePath, cachePath).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v\n%s", err, out)
	}

	ctx := context.Background()
	wtPath := filepath.Join(dir, "worktree-patch")

	patch := `--- a/hello.txt
+++ b/hello.txt
@@ -1 +1 @@
-hello
+hello world
`

	err := rc.PrepareWorktree(ctx, "test/repo", sha, wtPath, patch)
	if err != nil {
		t.Fatalf("PrepareWorktree() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(wtPath, "hello.txt"))
	if err != nil {
		t.Fatalf("read hello.txt: %v", err)
	}
	if string(content) != "hello world\n" {
		t.Errorf("hello.txt = %q, want %q", content, "hello world\n")
	}

	// Clean up.
	_ = rc.RemoveWorktree(ctx, "test/repo", wtPath)
}

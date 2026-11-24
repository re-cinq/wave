package bench

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// seedCache copies a bare repo into the cache slot for "test/repo".
func seedCache(t *testing.T, dir string) (*repoCache, string) {
	t.Helper()
	barePath, sha := initBareRepo(t, dir)
	rc := &repoCache{CacheDir: filepath.Join(dir, "cache")}
	if err := os.MkdirAll(rc.CacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cp", "-r", barePath, rc.clonePath("test/repo")).CombinedOutput(); err != nil {
		t.Fatalf("cp: %v\n%s", err, out)
	}
	return rc, sha
}

func Test_repoCache_ClonePath(t *testing.T) {
	rc := &repoCache{CacheDir: "/tmp/cache"}
	if got, want := rc.clonePath("django/django"), "/tmp/cache/django__django"; got != want {
		t.Errorf("clonePath = %q, want %q", got, want)
	}
}

func Test_repoCache_EnsureCloned_KnownCommitSkipsFetch(t *testing.T) {
	dir := t.TempDir()
	rc, sha := seedCache(t, dir)
	// The seeded clone's origin points at a path that no longer exists, so a
	// fetch would fail: success proves the present commit short-circuits it.
	if err := os.RemoveAll(filepath.Join(dir, "src")); err != nil {
		t.Fatal(err)
	}
	got, err := rc.EnsureCloned(context.Background(), "test/repo", sha)
	if err != nil {
		t.Fatalf("EnsureCloned() error = %v", err)
	}
	if got != rc.clonePath("test/repo") {
		t.Errorf("EnsureCloned() = %q, want %q", got, rc.clonePath("test/repo"))
	}
}

func Test_repoCache_EnsureCloned_MissingCommitFetches(t *testing.T) {
	dir := t.TempDir()
	rc, _ := seedCache(t, dir)
	_, err := rc.EnsureCloned(context.Background(), "test/repo", "0123456789abcdef0123456789abcdef01234567")
	if err != nil {
		t.Fatalf("EnsureCloned() should fetch from origin, got %v", err)
	}
}

func Test_repoCache_PrepareAndRemoveWorktree(t *testing.T) {
	dir := t.TempDir()
	rc, sha := seedCache(t, dir)
	ctx := context.Background()
	wtPath := filepath.Join(dir, "worktree")

	if err := rc.PrepareWorktree(ctx, "test/repo", sha, wtPath); err != nil {
		t.Fatalf("PrepareWorktree() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(wtPath, "hello.txt"))
	if err != nil || string(content) != "hello\n" {
		t.Fatalf("hello.txt = %q, %v", content, err)
	}
	// Re-preparing the same path is idempotent.
	if err := rc.PrepareWorktree(ctx, "test/repo", sha, wtPath); err != nil {
		t.Fatalf("second PrepareWorktree() error = %v", err)
	}
	if err := rc.RemoveWorktree(ctx, "test/repo", wtPath); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree dir should be removed")
	}
}

func Test_capturePatch_IncludesNewFilesExcludesWaveFiles(t *testing.T) {
	dir := t.TempDir()
	rc, sha := seedCache(t, dir)
	ctx := context.Background()
	wtPath := filepath.Join(dir, "worktree")
	if err := rc.PrepareWorktree(ctx, "test/repo", sha, wtPath); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"hello.txt":                 "hello world\n",
		"pkg/new.py":                "x = 1\n",
		"wave.yaml":                 "kind: x\n",
		"AGENTS.md":                 "rules\n",
		".agents/output/x.json":     "{}\n",
		".claude/settings.json":     "{}\n",
		"pkg/__pycache__/new.pyc":   "bin",
		"pkg.egg-info/PKG-INFO":     "info\n",
		".pytest_cache/v/cache/abc": "c\n",
	}
	for rel, body := range files {
		p := filepath.Join(wtPath, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	patch, err := capturePatch(ctx, wtPath, sha)
	if err != nil {
		t.Fatalf("capturePatch() error = %v", err)
	}
	for _, want := range []string{"+++ b/hello.txt", "+++ b/pkg/new.py"} {
		if !strings.Contains(patch, want) {
			t.Errorf("patch missing %q:\n%s", want, patch)
		}
	}
	for _, unwanted := range []string{"wave.yaml", "AGENTS.md", ".agents/", ".claude/", "__pycache__", "egg-info", ".pytest_cache"} {
		if strings.Contains(patch, unwanted) {
			t.Errorf("patch should not contain %q:\n%s", unwanted, patch)
		}
	}
}

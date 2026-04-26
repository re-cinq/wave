package bench

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// repoCache manages bare-clone caching and worktree creation for benchmark
// repositories. Each repo is cloned once (bare) and subsequent tasks create
// lightweight worktrees from the cached bare clone.
type repoCache struct {
	// CacheDir is the root directory for cached bare clones.
	CacheDir string

	// mu protects repoLocks.
	mu sync.Mutex
	// repoLocks holds a per-repo mutex to serialize clone/fetch operations.
	repoLocks map[string]*sync.Mutex
}

// repoMu returns a per-repo mutex, creating one if needed.
func (rc *repoCache) repoMu(repo string) *sync.Mutex {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.repoLocks == nil {
		rc.repoLocks = make(map[string]*sync.Mutex)
	}
	m, ok := rc.repoLocks[repo]
	if !ok {
		m = &sync.Mutex{}
		rc.repoLocks[repo] = m
	}
	return m
}

// EnsureCloned fetches a bare clone of the given repo into the cache directory.
// If the clone already exists it runs git fetch instead.
// repo is a GitHub slug like "django/django".
func (rc *repoCache) EnsureCloned(ctx context.Context, repo string) (string, error) {
	// Serialize clone/fetch per repo to avoid concurrent clone races.
	mu := rc.repoMu(repo)
	mu.Lock()
	defer mu.Unlock()

	cloneDir := rc.clonePath(repo)

	if _, err := os.Stat(filepath.Join(cloneDir, "HEAD")); err == nil {
		// Already cloned — fetch updates.
		cmd := exec.CommandContext(ctx, "git", "fetch", "--all", "--quiet")
		cmd.Dir = cloneDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git fetch %s: %w\n%s", repo, err, out)
		}
		return cloneDir, nil
	}

	url := "https://github.com/" + repo + ".git"
	if err := os.MkdirAll(filepath.Dir(cloneDir), 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--bare", "--quiet", url, cloneDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone --bare %s: %w\n%s", repo, err, out)
	}
	return cloneDir, nil
}

// PrepareWorktree creates a detached worktree at worktreePath checked out to
// baseCommit. If testPatch is non-empty it is applied after checkout.
func (rc *repoCache) PrepareWorktree(ctx context.Context, repo, baseCommit, worktreePath, testPatch string) error {
	cloneDir := rc.clonePath(repo)

	// Resolve to absolute path so git worktree commands work from the bare clone dir.
	absWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("resolve worktree path: %w", err)
	}

	// Remove existing worktree entry if present (idempotent).
	removeCmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", absWorktree)
	removeCmd.Dir = cloneDir
	_ = removeCmd.Run()
	// Also remove leftover directory if worktree entry was already pruned.
	_ = os.RemoveAll(absWorktree)

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", absWorktree, baseCommit)
	cmd.Dir = cloneDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add at %s: %w\n%s", baseCommit, err, out)
	}

	// Apply test patch if provided.
	if testPatch != "" {
		applyCmd := exec.CommandContext(ctx, "git", "apply", "--allow-empty", "-")
		applyCmd.Dir = worktreePath
		applyCmd.Stdin = strings.NewReader(testPatch)
		if out, err := applyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("apply test patch: %w\n%s", err, out)
		}
	}

	return nil
}

// RemoveWorktree removes a worktree created by PrepareWorktree.
func (rc *repoCache) RemoveWorktree(ctx context.Context, repo, worktreePath string) error {
	cloneDir := rc.clonePath(repo)
	absWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("resolve worktree path: %w", err)
	}
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", absWorktree)
	cmd.Dir = cloneDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove %s: %w\n%s", worktreePath, err, out)
	}
	return nil
}

// clonePath returns the filesystem path for a cached bare clone.
// "django/django" → "<CacheDir>/django__django".
func (rc *repoCache) clonePath(repo string) string {
	safe := strings.ReplaceAll(repo, "/", "__")
	return filepath.Join(rc.CacheDir, safe)
}

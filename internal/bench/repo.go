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
	// repoLocks holds a per-repo mutex to serialize git operations that
	// write the bare clone's metadata (clone, fetch, worktree add/remove).
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

// EnsureCloned makes sure a bare clone of repo containing commit exists in the
// cache directory. An existing clone is only fetched when commit is missing.
// repo is a GitHub slug like "django/django".
func (rc *repoCache) EnsureCloned(ctx context.Context, repo, commit string) (string, error) {
	mu := rc.repoMu(repo)
	mu.Lock()
	defer mu.Unlock()

	cloneDir := rc.clonePath(repo)

	if _, err := os.Stat(filepath.Join(cloneDir, "HEAD")); err == nil {
		if commit != "" && hasCommit(ctx, cloneDir, commit) {
			return cloneDir, nil
		}
		cmd := exec.CommandContext(ctx, "git", "fetch", "--quiet", "origin", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*")
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

func hasCommit(ctx context.Context, gitDir, commit string) bool {
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-e", commit+"^{commit}")
	cmd.Dir = gitDir
	return cmd.Run() == nil
}

// PrepareWorktree creates a detached worktree at worktreePath checked out to
// baseCommit.
func (rc *repoCache) PrepareWorktree(ctx context.Context, repo, baseCommit, worktreePath string) error {
	mu := rc.repoMu(repo)
	mu.Lock()
	defer mu.Unlock()

	cloneDir := rc.clonePath(repo)

	// Resolve to absolute path so git worktree commands work from the bare clone dir.
	absWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("resolve worktree path: %w", err)
	}

	// Remove an existing worktree entry if present (idempotent re-runs).
	removeCmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", absWorktree)
	removeCmd.Dir = cloneDir
	_ = removeCmd.Run()
	_ = os.RemoveAll(absWorktree)
	pruneWorktrees(ctx, cloneDir)

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", absWorktree, baseCommit)
	cmd.Dir = cloneDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add at %s: %w\n%s", baseCommit, err, out)
	}
	return nil
}

// RemoveWorktree removes a worktree created by PrepareWorktree. Nested
// worktrees a pipeline created inside it are pruned as well.
func (rc *repoCache) RemoveWorktree(ctx context.Context, repo, worktreePath string) error {
	mu := rc.repoMu(repo)
	mu.Lock()
	defer mu.Unlock()

	cloneDir := rc.clonePath(repo)
	absWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("resolve worktree path: %w", err)
	}
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", absWorktree)
	cmd.Dir = cloneDir
	out, rmErr := cmd.CombinedOutput()
	// `worktree remove` refuses when the tree contains nested worktrees;
	// delete the directory and prune the stale entries instead.
	if rmErr != nil {
		if err := os.RemoveAll(absWorktree); err != nil {
			return fmt.Errorf("git worktree remove %s: %w\n%s", worktreePath, rmErr, out)
		}
	}
	pruneWorktrees(ctx, cloneDir)
	return nil
}

func pruneWorktrees(ctx context.Context, cloneDir string) {
	cmd := exec.CommandContext(ctx, "git", "worktree", "prune")
	cmd.Dir = cloneDir
	_ = cmd.Run()
}

// clonePath returns the filesystem path for a cached bare clone.
// "django/django" -> "<CacheDir>/django__django".
func (rc *repoCache) clonePath(repo string) string {
	safe := strings.ReplaceAll(repo, "/", "__")
	return filepath.Join(rc.CacheDir, safe)
}

// applyPatch applies a unified diff in dir.
func applyPatch(ctx context.Context, dir, patch string) error {
	cmd := exec.CommandContext(ctx, "git", "apply", "--allow-empty", "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(patch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git apply: %w\n%s", err, out)
	}
	return nil
}

// patchExcludes are pathspecs left out of a captured patch: files Wave and
// the adapters write into a workspace, and Python build/test litter.
var patchExcludes = []string{
	":(exclude).agents",
	":(exclude).claude",
	":(exclude).codex",
	":(exclude).gemini",
	":(exclude).opencode",
	":(exclude)wave.yaml",
	":(exclude)AGENTS.md",
	":(exclude)CLAUDE.md",
	":(exclude)CODEX.md",
	":(exclude)GEMINI.md",
	":(exclude,glob)**/__pycache__/**",
	":(exclude,glob)**/*.pyc",
	":(exclude,glob)**/.pytest_cache/**",
	":(exclude,glob)**/*.egg-info/**",
}

// capturePatch stages every change in dir (tracked and untracked, minus
// patchExcludes) and returns the diff against baseCommit.
func capturePatch(ctx context.Context, dir, baseCommit string) (string, error) {
	add := exec.CommandContext(ctx, "git", append([]string{"add", "-A", "--", "."}, patchExcludes...)...)
	add.Dir = dir
	if out, err := add.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add: %w\n%s", err, out)
	}
	diff := exec.CommandContext(ctx, "git", "diff", "--cached", baseCommit)
	diff.Dir = dir
	out, err := diff.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}

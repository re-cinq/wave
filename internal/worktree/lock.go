package worktree

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

// repoLock provides a context-aware mutex for serializing git worktree
// operations on a specific repository. Uses a buffered channel of capacity 1
// as a semaphore, enabling context-based timeout on acquisition.
type repoLock struct {
	sem chan struct{}
}

func newRepoLock() *repoLock {
	return &repoLock{sem: make(chan struct{}, 1)}
}

// LockWithContext acquires the lock or returns an error if the context
// is cancelled or times out before the lock can be acquired.
func (rl *repoLock) LockWithContext(ctx context.Context) error {
	select {
	case rl.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("lock acquisition timed out: %w", ctx.Err())
	}
}

// Unlock releases the lock. Must be called exactly once per successful
// LockWithContext call. Calling Unlock on an already-unlocked repoLock
// will block indefinitely.
func (rl *repoLock) Unlock() {
	<-rl.sem
}

// repoLocks is a package-level registry of per-repository locks.
// Keys are canonical (absolute, symlink-resolved) repository root paths.
var repoLocks sync.Map // map[string]*repoLock

// getRepoLock returns the lock for the given canonical repository path,
// creating one atomically if it doesn't exist.
func getRepoLock(canonicalRepoRoot string) *repoLock {
	val, _ := repoLocks.LoadOrStore(canonicalRepoRoot, newRepoLock())
	return val.(*repoLock)
}

// canonicalPath resolves symlinks and returns the absolute path.
// This ensures that different path representations of the same repository
// (e.g., relative paths, symlinks) resolve to the same canonical key.
func canonicalPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	return resolved, nil
}

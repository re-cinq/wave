package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Manager handles git worktree lifecycle for isolated workspace execution.
type Manager struct {
	repoRoot string
	mu       sync.Mutex
}

// NewManager creates a worktree manager for the given git repository root.
func NewManager(repoRoot string) (*Manager, error) {
	if repoRoot == "" {
		// Auto-detect repo root
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to detect git repository root: %w", err)
		}
		repoRoot = strings.TrimSpace(string(out))
	}

	// Verify it's a git repo
	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", repoRoot)
	}

	return &Manager{repoRoot: repoRoot}, nil
}

// Create creates a new git worktree at the given path on the specified branch.
// If the branch doesn't exist, it creates a new branch from HEAD.
func (m *Manager) Create(worktreePath, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if worktreePath == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clean up stale worktree if path already exists from a previous failed run
	if _, err := os.Stat(worktreePath); err == nil {
		// Prune stale worktree references first
		pruneCmd := exec.Command("git", "-C", m.repoRoot, "worktree", "prune")
		_ = pruneCmd.Run()

		// Try to remove the existing worktree
		removeCmd := exec.Command("git", "-C", m.repoRoot, "worktree", "remove", "--force", worktreePath)
		_ = removeCmd.Run()

		// If git couldn't remove it, clean up the directory manually
		if _, err := os.Stat(worktreePath); err == nil {
			if err := os.RemoveAll(worktreePath); err != nil {
				return fmt.Errorf("failed to clean up stale worktree at %s: %w", worktreePath, err)
			}
		}
	}

	// Check if branch exists
	branchExists := m.branchExists(branch)

	var cmd *exec.Cmd
	if branchExists {
		cmd = exec.Command("git", "-C", m.repoRoot, "worktree", "add", worktreePath, branch)
	} else {
		// Create new branch from HEAD
		cmd = exec.Command("git", "-C", m.repoRoot, "worktree", "add", "-b", branch, worktreePath)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %w\noutput: %s", err, string(out))
	}

	return nil
}

// Remove removes a git worktree at the given path.
func (m *Manager) Remove(worktreePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if worktreePath == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	// Try normal removal first
	cmd := exec.Command("git", "-C", m.repoRoot, "worktree", "remove", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Try force removal if normal fails (dirty worktree)
		forceCmd := exec.Command("git", "-C", m.repoRoot, "worktree", "remove", "--force", worktreePath)
		if forceOut, forceErr := forceCmd.CombinedOutput(); forceErr != nil {
			return fmt.Errorf("git worktree remove failed: %w\nnormal output: %s\nforce output: %s", forceErr, string(out), string(forceOut))
		}
	}

	return nil
}

// RepoRoot returns the repository root path.
func (m *Manager) RepoRoot() string {
	return m.repoRoot
}

// branchExists checks if a git branch exists locally.
func (m *Manager) branchExists(branch string) bool {
	cmd := exec.Command("git", "-C", m.repoRoot, "rev-parse", "--verify", "refs/heads/"+branch)
	return cmd.Run() == nil
}

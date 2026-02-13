package worktree

import "sync"

// WorktreeEntry represents a single worktree created during pipeline execution.
type WorktreeEntry struct {
	StepID       string // Pipeline step that created this worktree
	WorktreePath string // Absolute filesystem path to the worktree
	RepoRoot     string // Canonical repository root path
}

// WorktreeRegistry tracks worktrees created during a single pipeline execution
// for targeted cleanup on completion or failure.
type WorktreeRegistry struct {
	mu      sync.Mutex
	entries []WorktreeEntry
}

// NewWorktreeRegistry creates an empty worktree registry.
func NewWorktreeRegistry() *WorktreeRegistry {
	return &WorktreeRegistry{}
}

// Register adds a worktree entry to the registry.
// Safe for concurrent use.
func (r *WorktreeRegistry) Register(entry WorktreeEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
}

// Entries returns a copy of all registered worktree entries.
// Safe for concurrent use.
func (r *WorktreeRegistry) Entries() []WorktreeEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]WorktreeEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

// Count returns the number of registered worktrees.
// Safe for concurrent use.
func (r *WorktreeRegistry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

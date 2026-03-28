package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCleanupCmd(t *testing.T) {
	cmd := NewCleanupCmd()

	assert.Equal(t, "cleanup", cmd.Use)
	assert.Contains(t, cmd.Short, "orphaned worktrees")

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("dry-run"), "dry-run flag should exist")
	assert.NotNil(t, flags.Lookup("force"), "force flag should exist")
}

func TestListGitWorktrees(t *testing.T) {
	// This test calls actual git, so it requires a git repo.
	// The test repo (wave itself) should have at least one worktree (the main one).
	entries, err := listGitWorktrees()
	if err != nil {
		t.Skipf("git worktree list not available: %v", err)
	}

	// At minimum, the main worktree should be present.
	assert.NotEmpty(t, entries, "should find at least one worktree")

	// First entry should have a non-empty path.
	assert.NotEmpty(t, entries[0].Path, "first worktree should have a path")
}

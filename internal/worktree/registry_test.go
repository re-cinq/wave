package worktree

import (
	"sync"
	"testing"
)

func TestWorktreeRegistry_Empty(t *testing.T) {
	reg := NewWorktreeRegistry()

	if reg.Count() != 0 {
		t.Errorf("expected count 0, got %d", reg.Count())
	}

	entries := reg.Entries()
	if len(entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(entries))
	}
}

func TestWorktreeRegistry_RegisterAndEntries(t *testing.T) {
	reg := NewWorktreeRegistry()

	entry1 := WorktreeEntry{
		StepID:       "step-1",
		WorktreePath: "/tmp/wt-1",
		RepoRoot:     "/repo",
	}
	entry2 := WorktreeEntry{
		StepID:       "step-2",
		WorktreePath: "/tmp/wt-2",
		RepoRoot:     "/repo",
	}

	reg.Register(entry1)
	reg.Register(entry2)

	if reg.Count() != 2 {
		t.Errorf("expected count 2, got %d", reg.Count())
	}

	entries := reg.Entries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].StepID != "step-1" {
		t.Errorf("expected first entry step-1, got %s", entries[0].StepID)
	}
	if entries[1].StepID != "step-2" {
		t.Errorf("expected second entry step-2, got %s", entries[1].StepID)
	}
}

func TestWorktreeRegistry_EntryIsolation(t *testing.T) {
	reg := NewWorktreeRegistry()

	reg.Register(WorktreeEntry{
		StepID:       "step-1",
		WorktreePath: "/tmp/wt-1",
		RepoRoot:     "/repo",
	})

	// Get entries and modify the copy
	entries := reg.Entries()
	entries[0].StepID = "modified"

	// Original should be unaffected
	original := reg.Entries()
	if original[0].StepID != "step-1" {
		t.Errorf("expected original entry unmodified, got %s", original[0].StepID)
	}
}

func TestWorktreeRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewWorktreeRegistry()

	const goroutines = 50
	var wg sync.WaitGroup

	// Half goroutines register, half read
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				reg.Register(WorktreeEntry{
					StepID:       "step",
					WorktreePath: "/tmp/wt",
					RepoRoot:     "/repo",
				})
			} else {
				_ = reg.Entries()
				_ = reg.Count()
			}
		}()
	}

	wg.Wait()

	// Should have exactly goroutines/2 entries (the even-numbered goroutines)
	if reg.Count() != goroutines/2 {
		t.Errorf("expected count %d, got %d", goroutines/2, reg.Count())
	}
}

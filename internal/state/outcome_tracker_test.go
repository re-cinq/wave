package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutcomeTracker(t *testing.T) {
	tracker := NewOutcomeTracker("pipeline-1", nil)

	assert.Equal(t, 0, tracker.Count())
	assert.Empty(t, tracker.GetAll())
	assert.Empty(t, tracker.OutcomeWarnings())
}

func TestOutcomeTrackerSetPipelineID(t *testing.T) {
	tracker := NewOutcomeTracker("old", nil)
	tracker.SetPipelineID("new")

	tracker.mu.RLock()
	got := tracker.pipelineID
	tracker.mu.RUnlock()
	assert.Equal(t, "new", got)
}

func TestOutcomeTrackerAdd(t *testing.T) {
	t.Run("adds outcome", func(t *testing.T) {
		tracker := NewOutcomeTracker("p1", nil)
		tracker.AddFile("step-1", "out", "/tmp/out.json", "output")

		assert.Equal(t, 1, tracker.Count())
		all := tracker.GetAll()
		require.Len(t, all, 1)
		assert.Equal(t, "out", all[0].Label)
		assert.Equal(t, OutcomeTypeFile, all[0].Type)
	})

	t.Run("deduplicates by value and stepID", func(t *testing.T) {
		tracker := NewOutcomeTracker("p1", nil)
		tracker.AddFile("step-1", "out", "/tmp/out.json", "first")
		tracker.AddFile("step-1", "out-dup", "/tmp/out.json", "duplicate")

		assert.Equal(t, 1, tracker.Count())
	})

	t.Run("allows same value different stepID", func(t *testing.T) {
		tracker := NewOutcomeTracker("p1", nil)
		tracker.AddFile("step-1", "out", "/tmp/out.json", "from step 1")
		tracker.AddFile("step-2", "out", "/tmp/out.json", "from step 2")

		assert.Equal(t, 2, tracker.Count())
	})

	t.Run("nil record is no-op", func(t *testing.T) {
		tracker := NewOutcomeTracker("p1", nil)
		tracker.Add(nil)
		assert.Equal(t, 0, tracker.Count())
	})
}

func TestOutcomeTrackerConvenienceAdders(t *testing.T) {
	tests := []struct {
		name     string
		addFunc  func(*OutcomeTracker)
		wantType OutcomeType
	}{
		{"AddFile", func(tr *OutcomeTracker) { tr.AddFile("s", "n", "/p", "d") }, OutcomeTypeFile},
		{"AddURL", func(tr *OutcomeTracker) { tr.AddURL("s", "n", "https://example.com", "d") }, OutcomeTypeURL},
		{"AddPR", func(tr *OutcomeTracker) { tr.AddPR("s", "n", "https://github.com/pr/1", "d") }, OutcomeTypePR},
		{"AddDeployment", func(tr *OutcomeTracker) { tr.AddDeployment("s", "n", "https://deploy.example.com", "d") }, OutcomeTypeDeployment},
		{"AddLog", func(tr *OutcomeTracker) { tr.AddLog("s", "n", "/var/log/step.log", "d") }, OutcomeTypeLog},
		{"AddContract", func(tr *OutcomeTracker) { tr.AddContract("s", "n", "/contracts/out.json", "d") }, OutcomeTypeContract},
		{"AddArtifact", func(tr *OutcomeTracker) { tr.AddArtifact("s", "n", "/art/out.bin", "d") }, OutcomeTypeArtifact},
		{"AddBranch", func(tr *OutcomeTracker) { tr.AddBranch("s", "feat/x", "/ws/x", "d") }, OutcomeTypeBranch},
		{"AddIssue", func(tr *OutcomeTracker) { tr.AddIssue("s", "Issue #1", "https://github.com/issues/1", "d") }, OutcomeTypeIssue},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := NewOutcomeTracker("p1", nil)
			tc.addFunc(tracker)
			require.Equal(t, 1, tracker.Count())
			assert.Equal(t, tc.wantType, tracker.GetAll()[0].Type)
		})
	}
}

func TestOutcomeTrackerAddBranchInitialMetadata(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	tracker.AddBranch("s1", "feat/x", "/ws", "d")

	branches := tracker.GetByType(OutcomeTypeBranch)
	require.Len(t, branches, 1)
	pushed, ok := branches[0].Metadata["pushed"].(bool)
	require.True(t, ok)
	assert.False(t, pushed)
}

func TestOutcomeTrackerGetAll(t *testing.T) {
	t.Run("returns sorted copy", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		now := time.Now()
		tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "first", Value: "/a", StepID: "s1", CreatedAt: now.Add(-2 * time.Second)})
		tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "third", Value: "/c", StepID: "s1", CreatedAt: now})
		tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "second", Value: "/b", StepID: "s1", CreatedAt: now.Add(-1 * time.Second)})

		all := tracker.GetAll()
		require.Len(t, all, 3)
		assert.Equal(t, "first", all[0].Label)
		assert.Equal(t, "second", all[1].Label)
		assert.Equal(t, "third", all[2].Label)
	})

	t.Run("returns independent copy", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		tracker.AddFile("s1", "f", "/f", "d")

		c1 := tracker.GetAll()
		c2 := tracker.GetAll()
		c1[0] = nil
		require.NotNil(t, c2[0])
	})
}

func TestOutcomeTrackerGetByStep(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	now := time.Now()
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "a", Value: "/a", StepID: "step-1", CreatedAt: now.Add(-1 * time.Second)})
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "b", Value: "/b", StepID: "step-2", CreatedAt: now})
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "c", Value: "/c", StepID: "step-1", CreatedAt: now.Add(1 * time.Second)})

	results := tracker.GetByStep("step-1")
	require.Len(t, results, 2)
	assert.Equal(t, "a", results[0].Label)
	assert.Equal(t, "c", results[1].Label)

	assert.Nil(t, tracker.GetByStep("nonexistent"))
}

func TestOutcomeTrackerGetByType(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	tracker.AddFile("s1", "f", "/f", "d")
	tracker.AddURL("s1", "u", "https://example.com", "d")
	tracker.AddFile("s2", "f2", "/f2", "d")

	assert.Len(t, tracker.GetByType(OutcomeTypeFile), 2)
	assert.Len(t, tracker.GetByType(OutcomeTypeURL), 1)
	assert.Len(t, tracker.GetByType(OutcomeTypePR), 0)
}

func TestOutcomeTrackerCount(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	assert.Equal(t, 0, tracker.Count())

	tracker.AddFile("s1", "a", "/a", "d")
	assert.Equal(t, 1, tracker.Count())

	tracker.AddURL("s1", "b", "https://b.com", "d")
	assert.Equal(t, 2, tracker.Count())

	tracker.AddFile("s1", "a-dup", "/a", "d")
	assert.Equal(t, 2, tracker.Count())
}

func TestOutcomeTrackerFormatSummary(t *testing.T) {
	t.Setenv("NERD_FONT", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("TERMINAL_EMULATOR", "")

	t.Run("empty", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		assert.Equal(t, "", tracker.FormatSummary())
	})

	t.Run("populated", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		tracker.AddFile("s1", "output", "/tmp/output.json", "the output")
		tracker.AddURL("s1", "link", "https://example.com", "a link")

		summary := tracker.FormatSummary()
		assert.Contains(t, summary, "Artifacts (2):")
		assert.Contains(t, summary, "output.json")
		assert.Contains(t, summary, "https://example.com")
	})

	t.Run("nerd font prefix", func(t *testing.T) {
		t.Setenv("NERD_FONT", "1")
		tracker := NewOutcomeTracker("p", nil)
		tracker.AddFile("s1", "f", "/tmp/f.txt", "d")
		assert.Contains(t, tracker.FormatSummary(), "\U0001f4e6 Artifacts")
	})
}

func TestOutcomeTrackerFormatByStep(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	now := time.Now()
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "a", Value: "/a", StepID: "step-1", CreatedAt: now})
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeURL, Label: "b", Value: "https://b.com", StepID: "step-2", CreatedAt: now})
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "c", Value: "/c", StepID: "step-1", CreatedAt: now})

	result := tracker.FormatByStep()
	require.Contains(t, result, "step-1")
	require.Contains(t, result, "step-2")
	assert.Len(t, result["step-1"], 2)
	assert.Len(t, result["step-2"], 1)

	assert.Empty(t, NewOutcomeTracker("p", nil).FormatByStep())
}

func TestOutcomeTrackerGetLatestForStep(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	now := time.Now()
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "old", Value: "/old", StepID: "s1", CreatedAt: now.Add(-10 * time.Second)})
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "newest", Value: "/new", StepID: "s1", CreatedAt: now.Add(10 * time.Second)})
	tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "mid", Value: "/mid", StepID: "s1", CreatedAt: now})

	latest := tracker.GetLatestForStep("s1")
	require.NotNil(t, latest)
	assert.Equal(t, "newest", latest.Label)

	assert.Nil(t, tracker.GetLatestForStep("nope"))
}

func TestOutcomeTrackerAddWorkspaceFiles(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"output.json": `{"status":"ok"}`,
		"result.yaml": "key: value",
		"step.log":    "log",
		"data.json":   `{"data":true}`,
		"config.yml":  "setting: true",
		"README.md":   "# Hello",
		"notes.txt":   "notes",
		"binary.bin":  "no",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644))
	}

	tracker := NewOutcomeTracker("p", nil)
	tracker.AddWorkspaceFiles("step-1", tmp)

	all := tracker.GetAll()
	bases := map[string]bool{}
	for _, r := range all {
		bases[filepath.Base(r.Value)] = true
		assert.Equal(t, OutcomeTypeFile, r.Type)
		assert.Equal(t, "step-1", r.StepID)
	}
	for _, want := range []string{"output.json", "result.yaml", "step.log", "data.json", "config.yml", "README.md", "notes.txt"} {
		assert.True(t, bases[want], "expected %s in workspace files", want)
	}
	assert.False(t, bases["binary.bin"])
}

func TestOutcomeTrackerAddWorkspaceFilesDeduplicatesAcrossPatterns(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "output.json"), []byte("{}"), 0o644))

	tracker := NewOutcomeTracker("p", nil)
	tracker.AddWorkspaceFiles("step-1", tmp)

	count := 0
	for _, r := range tracker.GetAll() {
		if filepath.Base(r.Value) == "output.json" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestOutcomeTrackerAddWorkspaceFilesNonexistent(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	tracker.AddWorkspaceFiles("step-1", "/no/such/path/xyz")
	assert.Equal(t, 0, tracker.Count())
}

func TestOutcomeTrackerOutcomeWarnings(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	tracker.AddOutcomeWarning("warning 1")
	tracker.AddOutcomeWarning("warning 2")

	w := tracker.OutcomeWarnings()
	require.Len(t, w, 2)
	assert.Equal(t, "warning 1", w[0])

	w[0] = "mutated"
	assert.Equal(t, "warning 1", tracker.OutcomeWarnings()[0])

	assert.NotNil(t, NewOutcomeTracker("p", nil).OutcomeWarnings())
}

func TestOutcomeTrackerUpdateMetadata(t *testing.T) {
	t.Run("updates existing", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		tracker.AddBranch("s1", "feat/x", "/ws", "d")
		tracker.UpdateMetadata(OutcomeTypeBranch, "feat/x", "pushed", true)
		tracker.UpdateMetadata(OutcomeTypeBranch, "feat/x", "remote_ref", "origin/feat/x")

		branches := tracker.GetByType(OutcomeTypeBranch)
		require.Len(t, branches, 1)
		assert.Equal(t, true, branches[0].Metadata["pushed"])
		assert.Equal(t, "origin/feat/x", branches[0].Metadata["remote_ref"])
	})

	t.Run("no-op on miss", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		tracker.AddFile("s1", "f", "/f", "d")
		tracker.UpdateMetadata(OutcomeTypeFile, "nope", "k", "v")
		assert.Equal(t, 1, tracker.Count())
	})

	t.Run("initialises nil metadata", func(t *testing.T) {
		tracker := NewOutcomeTracker("p", nil)
		tracker.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: "bare", Value: "/bare", StepID: "s1"})
		tracker.UpdateMetadata(OutcomeTypeFile, "bare", "k", 42)
		assert.Equal(t, 42, tracker.GetAll()[0].Metadata["k"])
	})
}

// TestOutcomeTrackerPersistsToStore verifies that Add writes through to a real
// state store and survives a fresh GetOutcomes read.
func TestOutcomeTrackerPersistsToStore(t *testing.T) {
	store, err := NewStateStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	runID, err := store.CreateRun("test-pipeline", "input")
	require.NoError(t, err)

	tracker := NewOutcomeTracker(runID, store)
	tracker.AddPR("step-1", "PR #42", "https://github.com/org/repo/pull/42", "Implements feature X")
	tracker.AddBranch("step-2", "feat/persist", "/ws/feat", "Feature branch")

	persisted, err := store.GetOutcomes(runID)
	require.NoError(t, err)
	require.Len(t, persisted, 2)

	byType := map[OutcomeType]OutcomeRecord{}
	for _, r := range persisted {
		byType[r.Type] = r
	}

	pr := byType[OutcomeTypePR]
	assert.Equal(t, "PR #42", pr.Label)
	assert.Equal(t, "https://github.com/org/repo/pull/42", pr.Value)
	assert.Equal(t, "Implements feature X", pr.Description)

	branch := byType[OutcomeTypeBranch]
	assert.Equal(t, "feat/persist", branch.Label)
	require.NotNil(t, branch.Metadata)
	pushed, ok := branch.Metadata["pushed"].(bool)
	require.True(t, ok)
	assert.False(t, pushed)
}

// TestOutcomeTrackerPersistsToStoreSkipsWhenNoPipelineID guards against double-writes
// from a tracker constructed before the run ID is known.
func TestOutcomeTrackerPersistsToStoreSkipsWhenNoPipelineID(t *testing.T) {
	store, err := NewStateStore(":memory:")
	require.NoError(t, err)
	defer store.Close()

	tracker := NewOutcomeTracker("", store)
	tracker.AddPR("step-1", "PR", "https://example/pr/1", "")
	assert.Empty(t, tracker.OutcomeWarnings())
}

func TestOutcomeTrackerConcurrentAccess(t *testing.T) {
	tracker := NewOutcomeTracker("p", nil)
	const goroutines = 20
	const ops = 50

	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				tracker.Add(&OutcomeRecord{
					Type:      OutcomeTypeFile,
					Label:     fmt.Sprintf("file-%d-%d", id, j),
					Value:     fmt.Sprintf("/path/%d/%d", id, j),
					StepID:    fmt.Sprintf("step-%d", id%3),
					CreatedAt: time.Now(),
				})
			}
		}(i)
	}

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				_ = tracker.GetAll()
				_ = tracker.Count()
				_ = tracker.GetByType(OutcomeTypeFile)
			}
		}()
	}

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				tracker.SetPipelineID(fmt.Sprintf("p-%d-%d", id, j))
				tracker.AddOutcomeWarning(fmt.Sprintf("w-%d-%d", id, j))
				_ = tracker.OutcomeWarnings()
				tracker.UpdateMetadata(OutcomeTypeFile, fmt.Sprintf("file-%d-%d", id, j), "k", true)
				_ = tracker.GetByStep(fmt.Sprintf("step-%d", id%3))
				_ = tracker.GetLatestForStep(fmt.Sprintf("step-%d", id%3))
			}
		}(i)
	}

	wg.Wait()

	assert.Greater(t, tracker.Count(), 0)
	assert.Greater(t, len(tracker.OutcomeWarnings()), 0)
}

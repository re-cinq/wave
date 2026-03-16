package deliverable

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

func TestNewTracker(t *testing.T) {
	tracker := NewTracker("pipeline-1")

	assert.Equal(t, 0, tracker.Count(), "new tracker should have zero deliverables")
	assert.Empty(t, tracker.GetAll(), "new tracker GetAll should return empty slice")
	assert.Empty(t, tracker.OutcomeWarnings(), "new tracker should have no warnings")
}

func TestSetPipelineID(t *testing.T) {
	tracker := NewTracker("old-id")

	tracker.SetPipelineID("new-id")

	// Verify by reading the field under lock (same package access)
	tracker.mu.RLock()
	got := tracker.pipelineID
	tracker.mu.RUnlock()
	assert.Equal(t, "new-id", got)
}

func TestAdd(t *testing.T) {
	t.Run("adds deliverable", func(t *testing.T) {
		tracker := NewTracker("p1")
		d := NewFileDeliverable("step-1", "out", "/tmp/out.json", "output")

		tracker.Add(d)

		assert.Equal(t, 1, tracker.Count())
		all := tracker.GetAll()
		require.Len(t, all, 1)
		assert.Equal(t, "out", all[0].Name)
	})

	t.Run("deduplicates by path and stepID", func(t *testing.T) {
		tracker := NewTracker("p1")
		d1 := NewFileDeliverable("step-1", "out", "/tmp/out.json", "first")
		d2 := NewFileDeliverable("step-1", "out-dup", "/tmp/out.json", "duplicate")

		tracker.Add(d1)
		tracker.Add(d2)

		assert.Equal(t, 1, tracker.Count(), "duplicate (same path+stepID) should be skipped")
	})

	t.Run("allows same path different stepID", func(t *testing.T) {
		tracker := NewTracker("p1")
		d1 := NewFileDeliverable("step-1", "out", "/tmp/out.json", "from step 1")
		d2 := NewFileDeliverable("step-2", "out", "/tmp/out.json", "from step 2")

		tracker.Add(d1)
		tracker.Add(d2)

		assert.Equal(t, 2, tracker.Count(), "same path with different stepID should both be added")
	})

	t.Run("allows same stepID different path", func(t *testing.T) {
		tracker := NewTracker("p1")
		d1 := NewFileDeliverable("step-1", "a", "/tmp/a.json", "file a")
		d2 := NewFileDeliverable("step-1", "b", "/tmp/b.json", "file b")

		tracker.Add(d1)
		tracker.Add(d2)

		assert.Equal(t, 2, tracker.Count(), "same stepID with different paths should both be added")
	})
}

func TestConvenienceAdders(t *testing.T) {
	tests := []struct {
		name     string
		addFunc  func(tracker *Tracker)
		wantType DeliverableType
	}{
		{"AddFile", func(tr *Tracker) { tr.AddFile("s", "n", "/p", "d") }, TypeFile},
		{"AddURL", func(tr *Tracker) { tr.AddURL("s", "n", "https://example.com", "d") }, TypeURL},
		{"AddPR", func(tr *Tracker) { tr.AddPR("s", "n", "https://github.com/pr/1", "d") }, TypePR},
		{"AddDeployment", func(tr *Tracker) { tr.AddDeployment("s", "n", "https://deploy.example.com", "d") }, TypeDeployment},
		{"AddLog", func(tr *Tracker) { tr.AddLog("s", "n", "/var/log/step.log", "d") }, TypeLog},
		{"AddContract", func(tr *Tracker) { tr.AddContract("s", "n", "/contracts/out.json", "d") }, TypeContract},
		{"AddBranch", func(tr *Tracker) { tr.AddBranch("s", "feat/x", "/ws/x", "d") }, TypeBranch},
		{"AddIssue", func(tr *Tracker) { tr.AddIssue("s", "Issue #1", "https://github.com/issues/1", "d") }, TypeIssue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewTracker("p1")
			tt.addFunc(tracker)

			require.Equal(t, 1, tracker.Count())
			all := tracker.GetAll()
			assert.Equal(t, tt.wantType, all[0].Type)
		})
	}
}

func TestGetAll(t *testing.T) {
	t.Run("returns sorted copy", func(t *testing.T) {
		tracker := NewTracker("p1")

		// Add with controlled timestamps so we can verify sort order
		d1 := &Deliverable{Type: TypeFile, Name: "first", Path: "/a", StepID: "s1", CreatedAt: time.Now().Add(-2 * time.Second)}
		d2 := &Deliverable{Type: TypeFile, Name: "third", Path: "/c", StepID: "s1", CreatedAt: time.Now()}
		d3 := &Deliverable{Type: TypeFile, Name: "second", Path: "/b", StepID: "s1", CreatedAt: time.Now().Add(-1 * time.Second)}

		tracker.Add(d1)
		tracker.Add(d2)
		tracker.Add(d3)

		all := tracker.GetAll()
		require.Len(t, all, 3)
		assert.Equal(t, "first", all[0].Name)
		assert.Equal(t, "second", all[1].Name)
		assert.Equal(t, "third", all[2].Name)
	})

	t.Run("returns independent copy", func(t *testing.T) {
		tracker := NewTracker("p1")
		tracker.AddFile("s1", "f", "/f", "d")

		copy1 := tracker.GetAll()
		copy2 := tracker.GetAll()

		// Mutating one copy should not affect the other
		copy1[0] = nil
		require.NotNil(t, copy2[0], "GetAll should return independent copies")
	})

	t.Run("empty tracker returns empty slice", func(t *testing.T) {
		tracker := NewTracker("p1")
		all := tracker.GetAll()
		assert.NotNil(t, all)
		assert.Len(t, all, 0)
	})
}

func TestGetByStep(t *testing.T) {
	tracker := NewTracker("p1")
	tracker.Add(&Deliverable{Type: TypeFile, Name: "a", Path: "/a", StepID: "step-1", CreatedAt: time.Now().Add(-1 * time.Second)})
	tracker.Add(&Deliverable{Type: TypeFile, Name: "b", Path: "/b", StepID: "step-2", CreatedAt: time.Now()})
	tracker.Add(&Deliverable{Type: TypeFile, Name: "c", Path: "/c", StepID: "step-1", CreatedAt: time.Now().Add(1 * time.Second)})

	t.Run("filters by stepID", func(t *testing.T) {
		results := tracker.GetByStep("step-1")
		require.Len(t, results, 2)
		assert.Equal(t, "a", results[0].Name)
		assert.Equal(t, "c", results[1].Name)
	})

	t.Run("returns sorted by creation time", func(t *testing.T) {
		results := tracker.GetByStep("step-1")
		assert.True(t, results[0].CreatedAt.Before(results[1].CreatedAt))
	})

	t.Run("returns nil for unknown step", func(t *testing.T) {
		results := tracker.GetByStep("nonexistent")
		assert.Nil(t, results)
	})

	t.Run("single match", func(t *testing.T) {
		results := tracker.GetByStep("step-2")
		require.Len(t, results, 1)
		assert.Equal(t, "b", results[0].Name)
	})
}

func TestGetByType(t *testing.T) {
	tracker := NewTracker("p1")
	tracker.AddFile("s1", "f", "/f", "d")
	tracker.AddURL("s1", "u", "https://example.com", "d")
	tracker.AddFile("s2", "f2", "/f2", "d")

	files := tracker.GetByType(TypeFile)
	assert.Len(t, files, 2)

	urls := tracker.GetByType(TypeURL)
	assert.Len(t, urls, 1)

	prs := tracker.GetByType(TypePR)
	assert.Len(t, prs, 0)
}

func TestCount(t *testing.T) {
	tracker := NewTracker("p1")
	assert.Equal(t, 0, tracker.Count())

	tracker.AddFile("s1", "a", "/a", "d")
	assert.Equal(t, 1, tracker.Count())

	tracker.AddURL("s1", "b", "https://b.com", "d")
	assert.Equal(t, 2, tracker.Count())

	// Duplicate should not increase count
	tracker.AddFile("s1", "a-dup", "/a", "d")
	assert.Equal(t, 2, tracker.Count())
}

func TestFormatSummary(t *testing.T) {
	// Ensure consistent output by disabling nerd font detection
	t.Setenv("NERD_FONT", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("TERMINAL_EMULATOR", "")

	t.Run("empty tracker returns empty string", func(t *testing.T) {
		tracker := NewTracker("p1")
		assert.Equal(t, "", tracker.FormatSummary())
	})

	t.Run("populated tracker returns formatted summary", func(t *testing.T) {
		tracker := NewTracker("p1")
		tracker.AddFile("s1", "output", "/tmp/output.json", "the output")
		tracker.AddURL("s1", "link", "https://example.com", "a link")

		summary := tracker.FormatSummary()
		assert.NotEmpty(t, summary)
		assert.Contains(t, summary, "Artifacts (2):")
		// Should contain paths/URLs for each deliverable
		assert.Contains(t, summary, "output.json")
		assert.Contains(t, summary, "https://example.com")
	})

	t.Run("nerd font mode includes emoji prefix", func(t *testing.T) {
		t.Setenv("NERD_FONT", "1")
		tracker := NewTracker("p1")
		tracker.AddFile("s1", "f", "/tmp/f.txt", "d")

		summary := tracker.FormatSummary()
		assert.Contains(t, summary, "\U0001f4e6 Artifacts") // 📦
	})
}

func TestFormatByStep(t *testing.T) {
	tracker := NewTracker("p1")
	tracker.Add(&Deliverable{Type: TypeFile, Name: "a", Path: "/a", StepID: "step-1", CreatedAt: time.Now()})
	tracker.Add(&Deliverable{Type: TypeURL, Name: "b", Path: "https://b.com", StepID: "step-2", CreatedAt: time.Now()})
	tracker.Add(&Deliverable{Type: TypeFile, Name: "c", Path: "/c", StepID: "step-1", CreatedAt: time.Now()})

	result := tracker.FormatByStep()

	require.Contains(t, result, "step-1")
	require.Contains(t, result, "step-2")
	assert.Len(t, result["step-1"], 2)
	assert.Len(t, result["step-2"], 1)
}

func TestFormatByStepEmpty(t *testing.T) {
	tracker := NewTracker("p1")
	result := tracker.FormatByStep()
	assert.Empty(t, result)
}

func TestGetLatestForStep(t *testing.T) {
	t.Run("returns most recent deliverable", func(t *testing.T) {
		tracker := NewTracker("p1")
		now := time.Now()

		tracker.Add(&Deliverable{Type: TypeFile, Name: "old", Path: "/old", StepID: "s1", CreatedAt: now.Add(-10 * time.Second)})
		tracker.Add(&Deliverable{Type: TypeFile, Name: "newest", Path: "/new", StepID: "s1", CreatedAt: now.Add(10 * time.Second)})
		tracker.Add(&Deliverable{Type: TypeFile, Name: "mid", Path: "/mid", StepID: "s1", CreatedAt: now})

		latest := tracker.GetLatestForStep("s1")
		require.NotNil(t, latest)
		assert.Equal(t, "newest", latest.Name)
	})

	t.Run("returns nil for empty step", func(t *testing.T) {
		tracker := NewTracker("p1")
		assert.Nil(t, tracker.GetLatestForStep("nonexistent"))
	})

	t.Run("returns single deliverable when only one exists", func(t *testing.T) {
		tracker := NewTracker("p1")
		tracker.AddFile("s1", "only", "/only", "d")

		latest := tracker.GetLatestForStep("s1")
		require.NotNil(t, latest)
		assert.Equal(t, "only", latest.Name)
	})

	t.Run("ignores other steps", func(t *testing.T) {
		tracker := NewTracker("p1")
		now := time.Now()
		tracker.Add(&Deliverable{Type: TypeFile, Name: "s1-item", Path: "/s1", StepID: "s1", CreatedAt: now})
		tracker.Add(&Deliverable{Type: TypeFile, Name: "s2-item", Path: "/s2", StepID: "s2", CreatedAt: now.Add(1 * time.Hour)})

		latest := tracker.GetLatestForStep("s1")
		require.NotNil(t, latest)
		assert.Equal(t, "s1-item", latest.Name)
	})
}

func TestAddWorkspaceFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files matching various patterns
	filesToCreate := []struct {
		name    string
		content string
	}{
		{"output.json", `{"status":"ok"}`},
		{"result.yaml", "key: value"},
		{"step.log", "log line 1"},
		{"data.json", `{"data":true}`},
		{"config.yml", "setting: true"},
		{"README.md", "# Hello"},
		{"notes.txt", "some notes"},
		{"binary.bin", "not matched"},
	}

	for _, f := range filesToCreate {
		err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0o644)
		require.NoError(t, err)
	}

	tracker := NewTracker("p1")
	tracker.AddWorkspaceFiles("step-1", tmpDir)

	// Should discover files matching the glob patterns
	all := tracker.GetAll()
	assert.Greater(t, len(all), 0, "should discover workspace files")

	// Verify specific patterns matched
	paths := make(map[string]bool)
	for _, d := range all {
		paths[filepath.Base(d.Path)] = true
	}

	assert.True(t, paths["output.json"], "should find output.json")
	assert.True(t, paths["result.yaml"], "should find result.yaml")
	assert.True(t, paths["step.log"], "should find step.log")
	assert.True(t, paths["data.json"], "should find data.json")
	assert.True(t, paths["config.yml"], "should find config.yml")
	assert.True(t, paths["README.md"], "should find README.md")
	assert.True(t, paths["notes.txt"], "should find notes.txt")
	assert.False(t, paths["binary.bin"], "should NOT find binary.bin (no matching pattern)")

	// All discovered deliverables should be file type from the given step
	for _, d := range all {
		assert.Equal(t, TypeFile, d.Type)
		assert.Equal(t, "step-1", d.StepID)
	}
}

func TestAddWorkspaceFilesDeduplication(t *testing.T) {
	tmpDir := t.TempDir()

	// output.json matches both "output.*" and "*.json" patterns
	err := os.WriteFile(filepath.Join(tmpDir, "output.json"), []byte("{}"), 0o644)
	require.NoError(t, err)

	tracker := NewTracker("p1")
	tracker.AddWorkspaceFiles("step-1", tmpDir)

	// output.json should appear only once despite matching multiple patterns
	all := tracker.GetAll()
	count := 0
	for _, d := range all {
		if filepath.Base(d.Path) == "output.json" {
			count++
		}
	}
	assert.Equal(t, 1, count, "output.json should only be added once (internal dedup)")
}

func TestAddWorkspaceFilesEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	tracker := NewTracker("p1")
	tracker.AddWorkspaceFiles("step-1", tmpDir)

	assert.Equal(t, 0, tracker.Count(), "empty directory should produce no deliverables")
}

func TestAddWorkspaceFilesNonexistentDir(t *testing.T) {
	tracker := NewTracker("p1")
	// Should not panic on nonexistent directory
	tracker.AddWorkspaceFiles("step-1", "/nonexistent/path/xyz")
	assert.Equal(t, 0, tracker.Count())
}

func TestAddOutcomeWarning(t *testing.T) {
	tracker := NewTracker("p1")

	tracker.AddOutcomeWarning("warning 1")
	tracker.AddOutcomeWarning("warning 2")

	warnings := tracker.OutcomeWarnings()
	require.Len(t, warnings, 2)
	assert.Equal(t, "warning 1", warnings[0])
	assert.Equal(t, "warning 2", warnings[1])
}

func TestOutcomeWarningsReturnsCopy(t *testing.T) {
	tracker := NewTracker("p1")
	tracker.AddOutcomeWarning("original")

	warnings1 := tracker.OutcomeWarnings()
	warnings1[0] = "mutated"

	warnings2 := tracker.OutcomeWarnings()
	assert.Equal(t, "original", warnings2[0], "OutcomeWarnings should return independent copy")
}

func TestOutcomeWarningsEmpty(t *testing.T) {
	tracker := NewTracker("p1")
	warnings := tracker.OutcomeWarnings()
	assert.NotNil(t, warnings)
	assert.Len(t, warnings, 0)
}

func TestUpdateMetadata(t *testing.T) {
	t.Run("updates existing deliverable", func(t *testing.T) {
		tracker := NewTracker("p1")
		tracker.AddBranch("s1", "feat/x", "/ws", "d")

		tracker.UpdateMetadata(TypeBranch, "feat/x", "pushed", true)

		branches := tracker.GetByType(TypeBranch)
		require.Len(t, branches, 1)
		assert.Equal(t, true, branches[0].Metadata["pushed"])
	})

	t.Run("no-op for nonexistent deliverable", func(t *testing.T) {
		tracker := NewTracker("p1")
		tracker.AddFile("s1", "f", "/f", "d")

		// Should not panic
		tracker.UpdateMetadata(TypeFile, "nonexistent", "key", "value")
		assert.Equal(t, 1, tracker.Count())
	})

	t.Run("initializes nil metadata map", func(t *testing.T) {
		tracker := NewTracker("p1")
		tracker.Add(&Deliverable{
			Type:   TypeFile,
			Name:   "bare",
			Path:   "/bare",
			StepID: "s1",
		})

		tracker.UpdateMetadata(TypeFile, "bare", "newkey", 42)

		all := tracker.GetAll()
		require.Len(t, all, 1)
		assert.Equal(t, 42, all[0].Metadata["newkey"])
	})
}

// TestConcurrentAccess verifies that all tracker methods are safe for
// concurrent use. This test is designed to trigger the race detector.
func TestConcurrentAccess(t *testing.T) {
	tracker := NewTracker("p1")
	const goroutines = 20
	const opsPerGoroutine = 50

	var wg sync.WaitGroup

	// Writers: Add deliverables
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				tracker.Add(&Deliverable{
					Type:      TypeFile,
					Name:      fmt.Sprintf("file-%d-%d", id, j),
					Path:      fmt.Sprintf("/path/%d/%d", id, j),
					StepID:    fmt.Sprintf("step-%d", id%3),
					CreatedAt: time.Now(),
				})
			}
		}(i)
	}

	// Readers: GetAll
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = tracker.GetAll()
			}
		}()
	}

	// Readers: GetByStep
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = tracker.GetByStep(fmt.Sprintf("step-%d", id%3))
			}
		}(i)
	}

	// SetPipelineID
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				tracker.SetPipelineID(fmt.Sprintf("pipeline-%d-%d", id, j))
			}
		}(i)
	}

	// AddOutcomeWarning + OutcomeWarnings
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				tracker.AddOutcomeWarning(fmt.Sprintf("warn-%d-%d", id, j))
				_ = tracker.OutcomeWarnings()
			}
		}(i)
	}

	// Count
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = tracker.Count()
			}
		}()
	}

	// GetByType
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = tracker.GetByType(TypeFile)
			}
		}()
	}

	// GetLatestForStep
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				_ = tracker.GetLatestForStep(fmt.Sprintf("step-%d", id%3))
			}
		}(i)
	}

	// UpdateMetadata
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				tracker.UpdateMetadata(TypeFile, fmt.Sprintf("file-%d-%d", id, j), "concurrent", true)
			}
		}(i)
	}

	wg.Wait()

	// Basic sanity: we should have added some deliverables
	assert.Greater(t, tracker.Count(), 0, "concurrent adds should have produced deliverables")
	assert.Greater(t, len(tracker.OutcomeWarnings()), 0, "concurrent warnings should have been recorded")
}

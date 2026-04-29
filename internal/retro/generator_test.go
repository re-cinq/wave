package retro

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/metrics"
	"github.com/recinq/wave/internal/state"
)

// fullMockStore combines StateQuerier and RetroIndexer for Generator tests.
type fullMockStore struct {
	mockStateQuerier
	mockRetroIndexer
}

func TestGenerator_Generate(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)
	tmpDir := t.TempDir()
	retrosDir := filepath.Join(tmpDir, "retros")

	mock := &fullMockStore{
		mockStateQuerier: mockStateQuerier{
			run: &state.RunRecord{
				RunID:        "gen-test-1",
				PipelineName: "impl-issue",
				Status:       "completed",
				StartedAt:    now,
				CompletedAt:  &completed,
			},
			perfMetrics: []metrics.PerformanceMetricRecord{
				{StepID: "plan", DurationMs: 30000, TokensUsed: 1000, Success: true},
			},
			attempts: map[string][]state.StepAttemptRecord{},
		},
		mockRetroIndexer: mockRetroIndexer{
			records: make(map[string]*metrics.RetrospectiveRecord),
		},
	}

	enabled := true
	narrate := false
	cfg := &manifest.RetrosConfig{
		Enabled: &enabled,
		Narrate: &narrate,
	}

	gen := &Generator{
		collector: NewCollector(&mock.mockStateQuerier),
		storage:   NewStorage(retrosDir, &mock.mockRetroIndexer),
		config:    cfg,
	}

	gen.Generate("gen-test-1", "impl-issue")

	// Verify retro file exists
	filePath := filepath.Join(retrosDir, "gen-test-1.json")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("retro file not created: %v", err)
	}

	// Verify index entry
	if _, ok := mock.records["gen-test-1"]; !ok {
		t.Error("index entry not created")
	}
}

func TestGenerator_Generate_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	retrosDir := filepath.Join(tmpDir, "retros")

	mock := &fullMockStore{
		mockStateQuerier: mockStateQuerier{
			run: &state.RunRecord{RunID: "disabled-test"},
		},
		mockRetroIndexer: mockRetroIndexer{
			records: make(map[string]*metrics.RetrospectiveRecord),
		},
	}

	enabled := false
	cfg := &manifest.RetrosConfig{Enabled: &enabled}

	gen := &Generator{
		collector: NewCollector(&mock.mockStateQuerier),
		storage:   NewStorage(retrosDir, &mock.mockRetroIndexer),
		config:    cfg,
	}

	gen.Generate("disabled-test", "test")

	// Verify no file created
	if _, err := os.Stat(filepath.Join(retrosDir, "disabled-test.json")); err == nil {
		t.Error("retro file should not be created when disabled")
	}
}

func TestGenerator_GenerateNarrativeSync(t *testing.T) {
	tmpDir := t.TempDir()
	retrosDir := filepath.Join(tmpDir, "retros")

	now := time.Now()
	completedAt := now.Add(time.Minute)

	mock := &fullMockStore{
		mockStateQuerier: mockStateQuerier{
			run: &state.RunRecord{
				RunID:        "narrate-test",
				PipelineName: "impl-issue",
				Status:       "completed",
				StartedAt:    now,
				CompletedAt:  &completedAt,
			},
			perfMetrics:  []metrics.PerformanceMetricRecord{},
			attempts: map[string][]state.StepAttemptRecord{},
		},
		mockRetroIndexer: mockRetroIndexer{
			records: make(map[string]*metrics.RetrospectiveRecord),
		},
	}

	enabled := true
	narrate := true
	cfg := &manifest.RetrosConfig{
		Enabled: &enabled,
		Narrate: &narrate,
	}

	gen := &Generator{
		collector: NewCollector(&mock.mockStateQuerier),
		storage:   NewStorage(retrosDir, &mock.mockRetroIndexer),
		config:    cfg,
	}

	// First generate the quantitative retro (no narrator yet to avoid async goroutine)
	gen.Generate("narrate-test", "impl-issue")

	// Now attach the narrator and generate narrative synchronously
	mockRunner := &mockAdapterRunner{
		resultContent: `{"smoothness": "smooth", "intent": "test", "outcome": "done"}`,
	}
	gen.narrator = NewNarrator(mockRunner, "test-model")
	ctx := context.Background()
	if err := gen.GenerateNarrativeSync(ctx, "narrate-test"); err != nil {
		t.Fatalf("GenerateNarrativeSync failed: %v", err)
	}

	// Verify narrative was added
	loaded, err := gen.GetStorage().Load("narrate-test")
	if err != nil {
		t.Fatalf("failed to load retro: %v", err)
	}
	if loaded.Narrative == nil {
		t.Fatal("narrative not attached")
	}
	if loaded.Narrative.Smoothness != SmoothnessSmooth {
		t.Errorf("smoothness: got %s, want smooth", loaded.Narrative.Smoothness)
	}
}

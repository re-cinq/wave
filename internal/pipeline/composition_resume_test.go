package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
)

// artifactCapturingStore wraps MockStateStore and captures RegisterArtifact
// calls plus serves them back from GetArtifacts. Used by composition resume
// tests to verify the executor registers aggregate/iterate outputs and that
// loadResumeState reads them back.
type artifactCapturingStore struct {
	*testutil.MockStateStore
	mu      sync.Mutex
	records []state.ArtifactRecord
}

func newArtifactCapturingStore() *artifactCapturingStore {
	return &artifactCapturingStore{MockStateStore: testutil.NewMockStateStore()}
}

func (s *artifactCapturingStore) RegisterArtifact(runID, stepID, name, path, artifactType string, sizeBytes int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, state.ArtifactRecord{
		RunID:     runID,
		StepID:    stepID,
		Name:      name,
		Path:      path,
		Type:      artifactType,
		SizeBytes: sizeBytes,
		CreatedAt: time.Now(),
	})
	return nil
}

func (s *artifactCapturingStore) GetArtifacts(runID, stepID string) ([]state.ArtifactRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]state.ArtifactRecord, 0, len(s.records))
	for _, rec := range s.records {
		if rec.RunID != runID {
			continue
		}
		if stepID != "" && rec.StepID != stepID {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

func (s *artifactCapturingStore) snapshot() []state.ArtifactRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]state.ArtifactRecord, len(s.records))
	copy(cp, s.records)
	return cp
}

// TestExecuteAggregateInDAG_RegistersArtifact verifies that aggregate steps
// register their output in the DB so resume + WebUI OUT pills can find it.
// Regression: Bug 1 from issue #1434.
func TestExecuteAggregateInDAG_RegistersArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	store := newArtifactCapturingStore()
	exec := NewDefaultPipelineExecutor(adapter.NewMockAdapter(), WithStateStore(store))

	outputPath := filepath.Join(".agents", "artifacts", "merge-findings", "merged-findings.json")
	step := &Step{
		ID: "merge-findings",
		Aggregate: &AggregateConfig{
			Strategy: "concat",
			From:     `[{"id":1}]`,
			Into:     outputPath,
		},
	}

	execution := &PipelineExecution{
		Pipeline:      &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
		States:        map[string]string{},
		ArtifactPaths: map[string]string{},
		Status:        &PipelineStatus{ID: "run-abc"},
		Context:       NewPipelineContext("run-abc", "test", "merge-findings"),
	}

	if err := exec.executeAggregateInDAG(context.Background(), execution, step); err != nil {
		t.Fatalf("executeAggregateInDAG failed: %v", err)
	}

	recs := store.snapshot()
	if len(recs) != 1 {
		t.Fatalf("expected 1 registered artifact, got %d", len(recs))
	}
	got := recs[0]
	if got.RunID != "run-abc" {
		t.Errorf("runID: got %q, want run-abc", got.RunID)
	}
	if got.StepID != "merge-findings" {
		t.Errorf("stepID: got %q, want merge-findings", got.StepID)
	}
	if got.Name != "merged-findings" {
		t.Errorf("name: got %q, want merged-findings", got.Name)
	}
	if got.Type != "json" {
		t.Errorf("type: got %q, want json", got.Type)
	}
	if got.Path != outputPath {
		t.Errorf("path: got %q, want %q", got.Path, outputPath)
	}
}

// TestCollectIterateOutputs_RegistersCollectedOutput verifies iterate steps
// register their collected-output artifact for resume recovery. Bug 1.
func TestCollectIterateOutputs_RegistersCollectedOutput(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	store := newArtifactCapturingStore()
	exec := NewDefaultPipelineExecutor(adapter.NewMockAdapter(), WithStateStore(store))

	pctx := NewPipelineContext("run-iter", "test", "parallel-review")
	pctx.ArtifactPaths = map[string]string{}

	// Stage two child outputs so collectIterateOutputs has data to merge.
	staged := filepath.Join(tmpDir, "child-output.json")
	if err := os.WriteFile(staged, []byte(`{"finding":"x"}`), 0644); err != nil {
		t.Fatalf("stage child output: %v", err)
	}
	pctx.ArtifactPaths["audit-alpha.scan:output"] = staged
	pctx.ArtifactPaths["audit-beta.scan:output"] = staged

	step := &Step{ID: "parallel-review"}

	execution := &PipelineExecution{
		Pipeline:      &Pipeline{Metadata: PipelineMetadata{Name: "test"}},
		States:        map[string]string{},
		ArtifactPaths: map[string]string{},
		Status:        &PipelineStatus{ID: "run-iter"},
		Context:       pctx,
	}

	if err := exec.collectIterateOutputs(execution, step, []string{"audit-alpha", "audit-beta"}); err != nil {
		t.Fatalf("collectIterateOutputs failed: %v", err)
	}

	recs := store.snapshot()
	if len(recs) != 1 {
		t.Fatalf("expected 1 registered artifact, got %d", len(recs))
	}
	got := recs[0]
	if got.Name != "collected-output" {
		t.Errorf("name: got %q, want collected-output", got.Name)
	}
	if got.StepID != "parallel-review" {
		t.Errorf("stepID: got %q, want parallel-review", got.StepID)
	}
	if got.RunID != "run-iter" {
		t.Errorf("runID: got %q, want run-iter", got.RunID)
	}
}

// TestWithWorkspaceOverride_PreservesPath verifies that the executor honors
// the override path and does NOT wipe its contents on Execute. Bug 2.
func TestWithWorkspaceOverride_PreservesPath(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Pre-create override dir with a marker file that must survive.
	override := filepath.Join(tmpDir, "preserved-ws")
	if err := os.MkdirAll(override, 0755); err != nil {
		t.Fatalf("mkdir override: %v", err)
	}
	marker := filepath.Join(override, "prior-output.txt")
	if err := os.WriteFile(marker, []byte("prior data"), 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status":"success"}`),
	)

	exec := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(testutil.NewEventCollector()),
		WithWorkspaceOverride(override),
	)

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {Adapter: "claude", Temperature: 0.1},
		},
		Runtime: manifest.Runtime{WorkspaceRoot: tmpDir, DefaultTimeoutMin: 5},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "override-test"},
		Steps: []Step{
			{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "noop"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = exec.Execute(ctx, p, m, "test")

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("marker file should still exist after Execute with workspace override: %v", err)
	}
}

package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

// mockChatStore implements the subset of state.StateStore needed by chatcontext.go.
// It embeds a base that panics on unimplemented methods so we only need to define
// the methods we actually test.
type mockChatStore struct {
	state.StateStore // embed interface; unimplemented methods will panic
	run              *state.RunRecord
	events           []state.LogRecord
	artifacts        []state.ArtifactRecord
	runs             []state.RunRecord
	runErr           error
}

func (m *mockChatStore) GetRun(runID string) (*state.RunRecord, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	if m.run != nil && m.run.RunID == runID {
		return m.run, nil
	}
	return nil, fmt.Errorf("run not found: %s", runID)
}

func (m *mockChatStore) GetEvents(runID string, opts state.EventQueryOptions) ([]state.LogRecord, error) {
	return m.events, nil
}

func (m *mockChatStore) GetArtifacts(runID string, stepID string) ([]state.ArtifactRecord, error) {
	if stepID == "" {
		return m.artifacts, nil
	}
	var filtered []state.ArtifactRecord
	for _, a := range m.artifacts {
		if a.StepID == stepID {
			filtered = append(filtered, a)
		}
	}
	return filtered, nil
}

func (m *mockChatStore) ListRuns(opts state.ListRunsOptions) ([]state.RunRecord, error) {
	if m.runs == nil {
		return nil, nil
	}
	result := m.runs
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}
	return result, nil
}

func (m *mockChatStore) Close() error { return nil }

func TestBuildChatContext(t *testing.T) {
	now := time.Now()
	completed := now.Add(5 * time.Minute)

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "test-run-001",
			PipelineName: "test-pipeline",
			Status:       "completed",
			Input:        "test input",
			TotalTokens:  5000,
			StartedAt:    now,
			CompletedAt:  &completed,
		},
		events: []state.LogRecord{
			{StepID: "step-1", State: "completed", Persona: "navigator", TokensUsed: 2000, DurationMs: 30000},
			{StepID: "step-2", State: "completed", Persona: "craftsman", TokensUsed: 3000, DurationMs: 60000},
		},
		artifacts: []state.ArtifactRecord{
			{StepID: "step-1", Name: "plan.json", Path: ".wave/output/plan.json", Type: "json", SizeBytes: 1024},
			{StepID: "step-2", Name: "result.md", Path: ".wave/output/result.md", Type: "markdown", SizeBytes: 2048},
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "step-1", Persona: "navigator"},
			{ID: "step-2", Persona: "craftsman"},
		},
	}

	tmpDir := t.TempDir()

	ctx, err := BuildChatContext(store, "test-run-001", p, tmpDir)
	if err != nil {
		t.Fatalf("BuildChatContext failed: %v", err)
	}

	if ctx.Run.RunID != "test-run-001" {
		t.Errorf("expected run ID test-run-001, got %s", ctx.Run.RunID)
	}

	if len(ctx.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(ctx.Steps))
	}

	// Step 1 checks
	s1 := ctx.Steps[0]
	if s1.StepID != "step-1" {
		t.Errorf("step 1: expected ID step-1, got %s", s1.StepID)
	}
	if s1.State != "completed" {
		t.Errorf("step 1: expected state completed, got %s", s1.State)
	}
	if s1.TokensUsed != 2000 {
		t.Errorf("step 1: expected 2000 tokens, got %d", s1.TokensUsed)
	}
	if s1.Duration != 30*time.Second {
		t.Errorf("step 1: expected 30s duration, got %s", s1.Duration)
	}
	if len(s1.Artifacts) != 1 {
		t.Errorf("step 1: expected 1 artifact, got %d", len(s1.Artifacts))
	}

	// Step 2 checks
	s2 := ctx.Steps[1]
	if s2.StepID != "step-2" {
		t.Errorf("step 2: expected ID step-2, got %s", s2.StepID)
	}
	if s2.Persona != "craftsman" {
		t.Errorf("step 2: expected persona craftsman, got %s", s2.Persona)
	}

	if len(ctx.Artifacts) != 2 {
		t.Errorf("expected 2 total artifacts, got %d", len(ctx.Artifacts))
	}
}

func TestBuildChatContext_WithWorkspace(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)

	tmpDir := t.TempDir()

	// Create a fake workspace directory
	wsDir := filepath.Join(tmpDir, ".wave", "workspaces", "my-pipeline", "analyze")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "ws-run-001",
			PipelineName: "my-pipeline",
			Status:       "completed",
			StartedAt:    now,
			CompletedAt:  &completed,
		},
		events: []state.LogRecord{
			{StepID: "analyze", State: "completed", Persona: "navigator", TokensUsed: 1000, DurationMs: 10000},
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "my-pipeline"},
		Steps: []Step{
			{ID: "analyze", Persona: "navigator"},
		},
	}

	ctx, err := BuildChatContext(store, "ws-run-001", p, tmpDir)
	if err != nil {
		t.Fatalf("BuildChatContext failed: %v", err)
	}

	if ctx.Steps[0].WorkspacePath != wsDir {
		t.Errorf("expected workspace path %s, got %s", wsDir, ctx.Steps[0].WorkspacePath)
	}
}

func TestBuildChatContext_FailedStep(t *testing.T) {
	now := time.Now()

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "fail-run",
			PipelineName: "test",
			Status:       "failed",
			StartedAt:    now,
			ErrorMessage: "step failed",
		},
		events: []state.LogRecord{
			{StepID: "build", State: "failed", Persona: "craftsman", Message: "compilation error", DurationMs: 5000},
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{ID: "build", Persona: "craftsman"},
		},
	}

	ctx, err := BuildChatContext(store, "fail-run", p, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Steps[0].ErrorMessage != "compilation error" {
		t.Errorf("expected error message 'compilation error', got %q", ctx.Steps[0].ErrorMessage)
	}
	if ctx.Steps[0].State != "failed" {
		t.Errorf("expected state 'failed', got %q", ctx.Steps[0].State)
	}
}

func TestBuildChatContext_RunNotFound(t *testing.T) {
	store := &mockChatStore{
		runErr: fmt.Errorf("run not found: missing-id"),
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test"},
		Steps:    []Step{},
	}

	_, err := BuildChatContext(store, "missing-id", p, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing run")
	}
}

func TestMostRecentCompletedRunID(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)

	store := &mockChatStore{
		runs: []state.RunRecord{
			{RunID: "running-1", Status: "running", StartedAt: now},
			{RunID: "completed-1", Status: "completed", StartedAt: now, CompletedAt: &completed},
			{RunID: "completed-2", Status: "completed", StartedAt: now.Add(-time.Hour), CompletedAt: &completed},
		},
	}

	runID, err := MostRecentCompletedRunID(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runID != "completed-1" {
		t.Errorf("expected completed-1, got %s", runID)
	}
}

func TestMostRecentCompletedRunID_NoRuns(t *testing.T) {
	store := &mockChatStore{
		runs: nil,
	}

	_, err := MostRecentCompletedRunID(store)
	if err == nil {
		t.Fatal("expected error when no runs found")
	}
}

func TestMostRecentCompletedRunID_OnlyRunning(t *testing.T) {
	now := time.Now()

	store := &mockChatStore{
		runs: []state.RunRecord{
			{RunID: "running-1", Status: "running", StartedAt: now},
		},
	}

	runID, err := MostRecentCompletedRunID(store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to most recent run
	if runID != "running-1" {
		t.Errorf("expected running-1, got %s", runID)
	}
}

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
			{StepID: "step-1", Name: "plan.json", Path: ".agents/output/plan.json", Type: "json", SizeBytes: 1024},
			{StepID: "step-2", Name: "result.md", Path: ".agents/output/result.md", Type: "markdown", SizeBytes: 2048},
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
	wsDir := filepath.Join(tmpDir, ".agents", "workspaces", "my-pipeline", "analyze")
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

func TestBuildChatContext_WithChatContextConfig(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)
	tmpDir := t.TempDir()

	// Create artifact files on disk
	artDir := filepath.Join(tmpDir, ".agents", "output")
	if err := os.MkdirAll(artDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "result.json"), []byte(`{"score":95,"verdict":"pass"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "notes.md"), []byte("# Summary\n\nAll good.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "ctx-run",
			PipelineName: "test-pipeline",
			Status:       "completed",
			StartedAt:    now,
			CompletedAt:  &completed,
		},
		events: []state.LogRecord{
			{StepID: "step-1", State: "completed", Persona: "navigator"},
		},
		artifacts: []state.ArtifactRecord{
			{StepID: "step-1", Name: "result.json", Path: ".agents/output/result.json", Type: "json"},
			{StepID: "step-1", Name: "notes.md", Path: ".agents/output/notes.md", Type: "markdown"},
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps:    []Step{{ID: "step-1", Persona: "navigator"}},
		ChatContext: &ChatContextConfig{
			ArtifactSummaries:  []string{"result.json", "notes.md"},
			SuggestedQuestions: []string{"How did the analysis go?"},
			FocusAreas:         []string{"score", "verdict"},
		},
	}

	ctx, err := BuildChatContext(store, "ctx-run", p, tmpDir)
	if err != nil {
		t.Fatalf("BuildChatContext failed: %v", err)
	}

	// ChatConfig should be propagated
	if ctx.ChatConfig == nil {
		t.Fatal("expected ChatConfig to be set")
	}
	if len(ctx.ChatConfig.SuggestedQuestions) != 1 {
		t.Errorf("expected 1 suggested question, got %d", len(ctx.ChatConfig.SuggestedQuestions))
	}

	// ArtifactContents should be populated
	if len(ctx.ArtifactContents) != 2 {
		t.Fatalf("expected 2 artifact contents, got %d", len(ctx.ArtifactContents))
	}

	if _, ok := ctx.ArtifactContents["result.json"]; !ok {
		t.Error("missing artifact content for result.json")
	}
	if _, ok := ctx.ArtifactContents["notes.md"]; !ok {
		t.Error("missing artifact content for notes.md")
	}
}

func TestBuildChatContext_NoChatContextConfig(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "no-ctx-run",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
			CompletedAt:  &completed,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test"},
		Steps:    []Step{{ID: "step-1", Persona: "navigator"}},
		// No ChatContext configured
	}

	ctx, err := BuildChatContext(store, "no-ctx-run", p, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.ChatConfig != nil {
		t.Error("expected ChatConfig to be nil when pipeline has no chat_context")
	}
	if ctx.ArtifactContents != nil {
		t.Error("expected ArtifactContents to be nil when no chat_context")
	}
}

func TestBuildChatContext_TokenBudget(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)
	tmpDir := t.TempDir()

	// Create a large artifact
	artDir := filepath.Join(tmpDir, ".agents", "output")
	if err := os.MkdirAll(artDir, 0755); err != nil {
		t.Fatal(err)
	}
	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(artDir, "large.txt"), largeContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "small.txt"), []byte("small content"), 0644); err != nil {
		t.Fatal(err)
	}

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "budget-run",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
			CompletedAt:  &completed,
		},
		artifacts: []state.ArtifactRecord{
			{StepID: "s1", Name: "large.txt", Path: ".agents/output/large.txt", Type: "text"},
			{StepID: "s1", Name: "small.txt", Path: ".agents/output/small.txt", Type: "text"},
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test"},
		Steps:    []Step{{ID: "s1", Persona: "nav"}},
		ChatContext: &ChatContextConfig{
			ArtifactSummaries: []string{"large.txt", "small.txt"},
			MaxContextTokens:  500, // Very small budget: 500 tokens ~2000 bytes
		},
	}

	ctx, err := BuildChatContext(store, "budget-run", p, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Large artifact should be truncated or budget message for small
	totalLen := 0
	for _, content := range ctx.ArtifactContents {
		totalLen += len(content)
	}
	// Total should be within budget (500 tokens * 4 bytes = 2000 bytes + some overhead)
	if totalLen > 3000 {
		t.Errorf("artifact contents exceed expected budget: %d bytes", totalLen)
	}
}

func TestBuildChatContext_MissingArtifactFile(t *testing.T) {
	now := time.Now()
	completed := now.Add(time.Minute)

	store := &mockChatStore{
		run: &state.RunRecord{
			RunID:        "missing-art-run",
			PipelineName: "test",
			Status:       "completed",
			StartedAt:    now,
			CompletedAt:  &completed,
		},
		artifacts: []state.ArtifactRecord{
			{StepID: "s1", Name: "exists.txt", Path: ".agents/output/exists.txt"},
			{StepID: "s1", Name: "missing.txt", Path: ".agents/output/missing.txt"},
		},
	}

	tmpDir := t.TempDir()
	artDir := filepath.Join(tmpDir, ".agents", "output")
	if err := os.MkdirAll(artDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "exists.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test"},
		Steps:    []Step{{ID: "s1", Persona: "nav"}},
		ChatContext: &ChatContextConfig{
			ArtifactSummaries: []string{"exists.txt", "missing.txt"},
		},
	}

	ctx, err := BuildChatContext(store, "missing-art-run", p, tmpDir)
	if err != nil {
		t.Fatalf("missing artifact should not cause error: %v", err)
	}

	// Should have content for the existing artifact
	if _, ok := ctx.ArtifactContents["exists.txt"]; !ok {
		t.Error("expected content for exists.txt")
	}
	// Missing artifact should be silently skipped
	if _, ok := ctx.ArtifactContents["missing.txt"]; ok {
		t.Error("expected missing.txt to be skipped")
	}
}

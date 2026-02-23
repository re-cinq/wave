package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/state"
)

// mockStepStore implements the subset of state.StateStore needed by stepcontroller.go.
type mockStepStore struct {
	state.StateStore // embed interface; unimplemented methods panic
	loggedEvents     []loggedEvent
}

type loggedEvent struct {
	runID, stepID, state, persona, message string
}

func (m *mockStepStore) LogEvent(runID string, stepID string, st string, persona string, message string, tokens int, durationMs int64) error {
	m.loggedEvents = append(m.loggedEvents, loggedEvent{runID, stepID, st, persona, message})
	return nil
}

func (m *mockStepStore) Close() error { return nil }

// ---------------------------------------------------------------------------
// Helper to build a minimal ChatContext for tests.
// ---------------------------------------------------------------------------

func newTestChatContext(projectRoot string, steps []ChatStepContext, pipeline *Pipeline) *ChatContext {
	return &ChatContext{
		Run: &state.RunRecord{
			RunID:        "test-run-001",
			PipelineName: "test-pipeline",
			Status:       "completed",
			StartedAt:    time.Now(),
		},
		Steps:       steps,
		Pipeline:    pipeline,
		ProjectRoot: projectRoot,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewStepController(t *testing.T) {
	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	if ctrl == nil {
		t.Fatal("expected non-nil controller")
	}
	if ctrl.store != store {
		t.Error("expected controller store to match provided store")
	}
	if ctrl.model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", ctrl.model)
	}
}

func TestContinueStep_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace directory that BuildChatContext would normally discover
	wsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "analyze")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "analyze",
			Persona:       "navigator",
			State:         "completed",
			WorkspacePath: wsDir,
		},
	}, &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps:    []Step{{ID: "analyze", Persona: "navigator"}},
	})

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.ContinueStep(context.Background(), chatCtx, "analyze")

	// LaunchInteractive will fail because claude binary is not available in test env.
	// The important thing is that CLAUDE.md was written before the launch attempt.
	claudeMdPath := filepath.Join(wsDir, "CLAUDE.md")
	if _, statErr := os.Stat(claudeMdPath); statErr != nil {
		t.Fatalf("expected CLAUDE.md to be written at %s, got stat error: %v", claudeMdPath, statErr)
	}

	// Read the written CLAUDE.md and verify it contains expected content
	data, readErr := os.ReadFile(claudeMdPath)
	if readErr != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", readErr)
	}
	content := string(data)
	if !strings.Contains(content, "Continue") {
		t.Errorf("CLAUDE.md should contain 'Continue', got:\n%s", content)
	}
	if !strings.Contains(content, "analyze") {
		t.Errorf("CLAUDE.md should contain step ID 'analyze', got:\n%s", content)
	}

	// err should be non-nil (claude CLI not found), but that's expected
	if err == nil {
		t.Fatal("expected error from LaunchInteractive (claude CLI not found in test env)")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected error to mention 'claude', got: %v", err)
	}
}

func TestContinueStep_NoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "analyze",
			Persona:       "navigator",
			State:         "completed",
			WorkspacePath: "", // no workspace
		},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.ContinueStep(context.Background(), chatCtx, "analyze")
	if err == nil {
		t.Fatal("expected error when step has no workspace")
	}
	if !strings.Contains(err.Error(), "no preserved workspace") {
		t.Errorf("expected error about 'no preserved workspace', got: %v", err)
	}
}

func TestContinueStep_StepNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{StepID: "analyze", Persona: "navigator"},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.ContinueStep(context.Background(), chatCtx, "nonexistent-step")
	if err == nil {
		t.Fatal("expected error for non-existent step ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain 'not found', got: %v", err)
	}
}

func TestExtendStep_RequiresInstructions(t *testing.T) {
	tmpDir := t.TempDir()

	wsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "build")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "build",
			Persona:       "craftsman",
			State:         "completed",
			WorkspacePath: wsDir,
		},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.ExtendStep(context.Background(), chatCtx, "build", "")
	if err == nil {
		t.Fatal("expected error when instructions are empty")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error about empty instructions, got: %v", err)
	}
}

func TestRevertStep_Preview(t *testing.T) {
	tmpDir := t.TempDir()

	wsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "build")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some files in the workspace
	for _, name := range []string{"main.go", "util.go", "README.md"} {
		fpath := filepath.Join(wsDir, name)
		if err := os.WriteFile(fpath, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "build",
			Persona:       "craftsman",
			State:         "completed",
			WorkspacePath: wsDir,
			Artifacts: []state.ArtifactRecord{
				{Name: "output.json", Path: ".wave/output/output.json", Type: "json"},
			},
		},
	}, &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps:    []Step{{ID: "build", Persona: "craftsman"}},
	})

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	preview, err := ctrl.RevertStep(context.Background(), chatCtx, "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if preview.StepID != "build" {
		t.Errorf("expected StepID 'build', got %q", preview.StepID)
	}
	if preview.WorkspacePath != wsDir {
		t.Errorf("expected WorkspacePath %q, got %q", wsDir, preview.WorkspacePath)
	}
	if preview.FilesAffected < 3 {
		t.Errorf("expected at least 3 files affected, got %d", preview.FilesAffected)
	}
	if preview.WorkspaceType != "directory" {
		t.Errorf("expected workspace type 'directory', got %q", preview.WorkspaceType)
	}
	if len(preview.Artifacts) != 1 || preview.Artifacts[0] != "output.json" {
		t.Errorf("expected artifacts [output.json], got %v", preview.Artifacts)
	}
}

func TestRevertStep_NoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "build",
			Persona:       "craftsman",
			State:         "completed",
			WorkspacePath: "", // no workspace
		},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	_, err := ctrl.RevertStep(context.Background(), chatCtx, "build")
	if err == nil {
		t.Fatal("expected error when step has no workspace")
	}
	if !strings.Contains(err.Error(), "no preserved workspace") {
		t.Errorf("expected error about 'no preserved workspace', got: %v", err)
	}
}

func TestConfirmRevert_DirectoryWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	wsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "build")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files to be removed
	for _, name := range []string{"main.go", "util.go"} {
		fpath := filepath.Join(wsDir, name)
		if err := os.WriteFile(fpath, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "build",
			Persona:       "craftsman",
			State:         "completed",
			WorkspacePath: wsDir,
		},
	}, &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps:    []Step{{ID: "build", Persona: "craftsman"}},
	})

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.ConfirmRevert(context.Background(), chatCtx, "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify workspace directory was removed
	if _, statErr := os.Stat(wsDir); !os.IsNotExist(statErr) {
		t.Errorf("expected workspace directory to be removed, but it still exists at %s", wsDir)
	}

	// Verify LogEvent was called with "reverted" state
	if len(store.loggedEvents) == 0 {
		t.Fatal("expected LogEvent to be called at least once")
	}
	found := false
	for _, evt := range store.loggedEvents {
		if evt.stepID == "build" && evt.state == "reverted" {
			found = true
			if evt.runID != "test-run-001" {
				t.Errorf("expected runID 'test-run-001', got %q", evt.runID)
			}
			if !strings.Contains(evt.message, "reverted") {
				t.Errorf("expected message to contain 'reverted', got %q", evt.message)
			}
			break
		}
	}
	if !found {
		t.Error("expected a logged event with state 'reverted' for step 'build'")
	}
}

func TestRewriteStep_CreatesWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	// Create upstream step workspace with an artifact
	upstreamWsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "analyze")
	if err := os.MkdirAll(upstreamWsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create the artifact file at the project root (absolute path resolution)
	artifactDir := filepath.Join(tmpDir, ".wave", "output")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(artifactDir, "plan.json")
	if err := os.WriteFile(artifactPath, []byte(`{"plan": "test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// The step to rewrite
	rewriteWsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "implement")

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "analyze",
			Persona:       "navigator",
			State:         "completed",
			WorkspacePath: upstreamWsDir,
			Artifacts: []state.ArtifactRecord{
				{
					StepID: "analyze",
					Name:   "plan.json",
					Path:   ".wave/output/plan.json",
					Type:   "json",
				},
			},
		},
		{
			StepID:        "implement",
			Persona:       "craftsman",
			State:         "failed",
			WorkspacePath: rewriteWsDir,
		},
	}, &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "analyze", Persona: "navigator"},
			{ID: "implement", Persona: "craftsman", Dependencies: []string{"analyze"}},
		},
	})

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.RewriteStep(context.Background(), chatCtx, "implement", "Rewrite the implementation with better error handling")

	// The RewriteStep call will fail on LaunchInteractive, but we can verify
	// everything up to that point.

	// Verify workspace was created
	if _, statErr := os.Stat(rewriteWsDir); statErr != nil {
		t.Fatalf("expected workspace to be created at %s, got error: %v", rewriteWsDir, statErr)
	}

	// Verify CLAUDE.md was written
	claudeMdPath := filepath.Join(rewriteWsDir, "CLAUDE.md")
	data, readErr := os.ReadFile(claudeMdPath)
	if readErr != nil {
		t.Fatalf("expected CLAUDE.md at %s, got read error: %v", claudeMdPath, readErr)
	}
	content := string(data)
	if !strings.Contains(content, "Rewrite") {
		t.Errorf("CLAUDE.md should contain 'Rewrite', got:\n%s", content)
	}
	if !strings.Contains(content, "better error handling") {
		t.Errorf("CLAUDE.md should contain the new prompt text, got:\n%s", content)
	}

	// Verify upstream artifacts were copied into the workspace
	copiedArtifact := filepath.Join(rewriteWsDir, ".wave", "artifacts", "analyze", "plan.json")
	if _, statErr := os.Stat(copiedArtifact); statErr != nil {
		t.Errorf("expected upstream artifact at %s, got error: %v", copiedArtifact, statErr)
	}

	// Verify the rewrite event was logged
	foundRewrite := false
	for _, evt := range store.loggedEvents {
		if evt.stepID == "implement" && evt.state == "rewriting" {
			foundRewrite = true
			break
		}
	}
	if !foundRewrite {
		t.Error("expected a logged event with state 'rewriting' for step 'implement'")
	}

	// err should be non-nil (claude CLI not found in test env)
	if err == nil {
		t.Fatal("expected error from LaunchInteractive (claude CLI not found in test env)")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected error to mention 'claude', got: %v", err)
	}
}

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a known number of files (including nested)
	files := []string{
		"a.txt",
		"b.txt",
		filepath.Join("subdir", "c.txt"),
		filepath.Join("subdir", "nested", "d.txt"),
	}
	for _, f := range files {
		fpath := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fpath, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	count, err := countFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != len(files) {
		t.Errorf("expected %d files, got %d", len(files), count)
	}
}

func TestCountFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := countFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", count)
	}
}

func TestFindStep_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{StepID: "analyze", Persona: "navigator"},
		{StepID: "build", Persona: "craftsman"},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	_, err := ctrl.findStep(chatCtx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent step")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain 'not found', got: %v", err)
	}
}

func TestFindStep_NilContext(t *testing.T) {
	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	_, err := ctrl.findStep(nil, "any-step")
	if err == nil {
		t.Fatal("expected error for nil chat context")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error to mention 'nil', got: %v", err)
	}
}

func TestFindStep_Found(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{StepID: "analyze", Persona: "navigator"},
		{StepID: "build", Persona: "craftsman", State: "failed", ErrorMessage: "compile error"},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	step, err := ctrl.findStep(chatCtx, "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step.StepID != "build" {
		t.Errorf("expected step ID 'build', got %q", step.StepID)
	}
	if step.Persona != "craftsman" {
		t.Errorf("expected persona 'craftsman', got %q", step.Persona)
	}
	if step.ErrorMessage != "compile error" {
		t.Errorf("expected error message 'compile error', got %q", step.ErrorMessage)
	}
}

func TestRevertStep_WorktreeType(t *testing.T) {
	tmpDir := t.TempDir()

	wsDir := filepath.Join(tmpDir, ".wave", "workspaces", "test-pipeline", "implement")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file so FilesAffected > 0
	if err := os.WriteFile(filepath.Join(wsDir, "code.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "implement",
			Persona:       "craftsman",
			State:         "completed",
			WorkspacePath: wsDir,
		},
	}, &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{{
			ID:      "implement",
			Persona: "craftsman",
			Workspace: WorkspaceConfig{
				Type: "worktree",
			},
		}},
	})

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	preview, err := ctrl.RevertStep(context.Background(), chatCtx, "implement")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preview.WorkspaceType != "worktree" {
		t.Errorf("expected workspace type 'worktree', got %q", preview.WorkspaceType)
	}
}

func TestRewriteStep_EmptyPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "build",
			Persona:       "craftsman",
			State:         "failed",
			WorkspacePath: filepath.Join(tmpDir, "ws"),
		},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.RewriteStep(context.Background(), chatCtx, "build", "")
	if err == nil {
		t.Fatal("expected error when prompt is empty")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error about empty prompt, got: %v", err)
	}
}

func TestConfirmRevert_NoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	chatCtx := newTestChatContext(tmpDir, []ChatStepContext{
		{
			StepID:        "build",
			Persona:       "craftsman",
			State:         "completed",
			WorkspacePath: "",
		},
	}, nil)

	store := &mockStepStore{}
	ctrl := NewStepController(store, "sonnet")

	err := ctrl.ConfirmRevert(context.Background(), chatCtx, "build")
	if err == nil {
		t.Fatal("expected error when step has no workspace")
	}
	if !strings.Contains(err.Error(), "no preserved workspace") {
		t.Errorf("expected error about 'no preserved workspace', got: %v", err)
	}
}

func TestFindPipelineStep_NotFound(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-1"},
			{ID: "step-2"},
		},
	}
	result := findPipelineStep(p, "nonexistent")
	if result != nil {
		t.Errorf("expected nil for non-existent pipeline step, got %+v", result)
	}
}

func TestFindPipelineStep_NilPipeline(t *testing.T) {
	result := findPipelineStep(nil, "any")
	if result != nil {
		t.Errorf("expected nil for nil pipeline, got %+v", result)
	}
}

func TestFindPipelineStep_Found(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step-1", Persona: "navigator"},
			{ID: "step-2", Persona: "craftsman"},
		},
	}
	result := findPipelineStep(p, "step-2")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "step-2" {
		t.Errorf("expected ID 'step-2', got %q", result.ID)
	}
	if result.Persona != "craftsman" {
		t.Errorf("expected persona 'craftsman', got %q", result.Persona)
	}
}

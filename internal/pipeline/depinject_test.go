package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// fixtureExecution constructs a PipelineExecution with a two-step pipeline:
// the upstream "fetch" step declares a single output artifact "pr-context".
// The caller decides how to register the artifact (in-memory, context, fs).
func fixtureExecution(t *testing.T, fetchOutputs []ArtifactDef) *PipelineExecution {
	t.Helper()
	pipe := &Pipeline{
		Steps: []Step{
			{ID: "fetch", OutputArtifacts: fetchOutputs},
			{ID: "consume", Dependencies: []string{"fetch"}},
		},
	}
	exec := &PipelineExecution{
		Pipeline:       pipe,
		Manifest:       &manifest.Manifest{},
		ArtifactPaths:  map[string]string{},
		WorkspacePaths: map[string]string{},
		WorktreePaths:  map[string]*WorktreeInfo{},
		Status:         &PipelineStatus{ID: "test-run-1"},
		Context:        NewPipelineContext("test-run-1", "test", "consume"),
	}
	return exec
}

// writeArtifactFile is a tiny helper that writes content under tmp and returns the path.
func writeArtifactFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

// TestResolveDependencyArtifacts_InMemoryArtifactPaths covers tier 1: the
// primary lookup path used by persona / command steps that successfully
// registered their outputs in execution.ArtifactPaths.
func TestResolveDependencyArtifacts_InMemoryArtifactPaths(t *testing.T) {
	tmp := t.TempDir()
	src := writeArtifactFile(t, tmp, "pr-context.json", `{"pr":1441}`)

	exec := fixtureExecution(t, []ArtifactDef{
		{Name: "pr-context", Type: "json", Required: true},
	})
	exec.ArtifactPaths["fetch:pr-context"] = src

	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	resolved, err := ex.ResolveDependencyArtifacts(exec, &exec.Pipeline.Steps[1])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got, ok := resolved["fetch:pr-context"]
	if !ok {
		t.Fatalf("missing fetch:pr-context entry; got %v", resolved)
	}
	if got.Path != src {
		t.Errorf("path = %q, want %q", got.Path, src)
	}
	if got.Optional {
		t.Error("expected required (Optional=false)")
	}
}

// TestResolveDependencyArtifacts_ContextNamespacedFallback covers tier 3:
// composition writers register under "<dep>.<name>" in the context.
func TestResolveDependencyArtifacts_ContextNamespacedFallback(t *testing.T) {
	tmp := t.TempDir()
	src := writeArtifactFile(t, tmp, "merged-findings.json", `[]`)

	exec := fixtureExecution(t, []ArtifactDef{
		{Name: "merged-findings", Type: "json", Required: true},
	})
	exec.Context.SetArtifactPath("fetch.merged-findings", src)

	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	resolved, err := ex.ResolveDependencyArtifacts(exec, &exec.Pipeline.Steps[1])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got, ok := resolved["fetch:merged-findings"]
	if !ok {
		t.Fatalf("missing entry; got %v", resolved)
	}
	if got.Path != src {
		t.Errorf("path = %q, want %q", got.Path, src)
	}
}

// TestResolveDependencyArtifacts_FilesystemFallback covers tier 4: the
// dep registered nothing, but the file is present in its workspace at the
// canonical path.
func TestResolveDependencyArtifacts_FilesystemFallback(t *testing.T) {
	tmp := t.TempDir()
	depWorkspace := filepath.Join(tmp, "ws-fetch")
	canonicalDir := filepath.Join(depWorkspace, ".agents", "artifacts", "fetch")
	src := writeArtifactFile(t, canonicalDir, "pr-context", `{"pr":1452}`)

	exec := fixtureExecution(t, []ArtifactDef{
		{Name: "pr-context", Type: "json", Required: true},
	})
	exec.WorkspacePaths["fetch"] = depWorkspace

	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	resolved, err := ex.ResolveDependencyArtifacts(exec, &exec.Pipeline.Steps[1])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got, ok := resolved["fetch:pr-context"]
	if !ok {
		t.Fatalf("missing entry; got %v", resolved)
	}
	if got.Path != src {
		t.Errorf("path = %q, want %q", got.Path, src)
	}
}

// TestResolveDependencyArtifacts_RequiredMissingErrors verifies a clear
// error is returned naming both dep and artifact when nothing matches.
func TestResolveDependencyArtifacts_RequiredMissingErrors(t *testing.T) {
	exec := fixtureExecution(t, []ArtifactDef{
		{Name: "pr-context", Type: "json", Required: true},
	})

	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	_, err := ex.ResolveDependencyArtifacts(exec, &exec.Pipeline.Steps[1])
	if err == nil {
		t.Fatal("expected error for missing required artifact")
	}
	msg := err.Error()
	if !strings.Contains(msg, "fetch") || !strings.Contains(msg, "pr-context") {
		t.Errorf("error should name dep and artifact; got: %v", err)
	}
}

// TestResolveDependencyArtifacts_OptionalMissingNoError verifies optional
// artifacts that cannot be located are silently skipped.
func TestResolveDependencyArtifacts_OptionalMissingNoError(t *testing.T) {
	exec := fixtureExecution(t, []ArtifactDef{
		{Name: "pr-context", Type: "json", Required: false},
	})

	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	resolved, err := ex.ResolveDependencyArtifacts(exec, &exec.Pipeline.Steps[1])
	if err != nil {
		t.Fatalf("optional missing should not error; got: %v", err)
	}
	if _, ok := resolved["fetch:pr-context"]; ok {
		t.Errorf("optional missing should be omitted; got %v", resolved)
	}
}

// TestInjectDependencyArtifacts_CanonicalAndAlias verifies both the
// canonical and back-compat alias paths get populated, and that
// {{ artifacts.<dep>.<name> }} resolves through the context.
func TestInjectDependencyArtifacts_CanonicalAndAlias(t *testing.T) {
	tmp := t.TempDir()
	src := writeArtifactFile(t, filepath.Join(tmp, "src"), "pr-context.json", `{"pr":1441}`)
	workspace := filepath.Join(tmp, "consume-ws")

	exec := fixtureExecution(t, []ArtifactDef{
		{Name: "pr-context", Type: "json", Required: true},
	})
	exec.ArtifactPaths["fetch:pr-context"] = src

	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	if err := ex.injectDependencyArtifacts(exec, &exec.Pipeline.Steps[1], workspace); err != nil {
		t.Fatalf("inject: %v", err)
	}

	canonical := filepath.Join(workspace, ".agents", "artifacts", "fetch", "pr-context")
	if _, err := os.Stat(canonical); err != nil {
		t.Errorf("canonical missing: %v", err)
	}
	alias := filepath.Join(workspace, ".agents", "output", "pr-context")
	if _, err := os.Stat(alias); err != nil {
		t.Errorf("alias missing: %v", err)
	}

	// Template resolution.
	got := exec.Context.ResolvePlaceholders("{{ artifacts.fetch.pr-context }}")
	if got != canonical {
		t.Errorf("template resolved to %q, want %q", got, canonical)
	}
}

// TestInjectDependencyArtifacts_NoDepsNoOp verifies a step with no
// dependencies makes no filesystem changes.
func TestInjectDependencyArtifacts_NoDepsNoOp(t *testing.T) {
	tmp := t.TempDir()
	exec := &PipelineExecution{
		Pipeline:       &Pipeline{Steps: []Step{{ID: "solo"}}},
		ArtifactPaths:  map[string]string{},
		WorkspacePaths: map[string]string{},
		Status:         &PipelineStatus{ID: "test"},
		Context:        NewPipelineContext("test", "t", "solo"),
	}
	ex := NewDefaultPipelineExecutor(adapter.NewMockAdapter())
	if err := ex.injectDependencyArtifacts(exec, &exec.Pipeline.Steps[0], tmp); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".agents")); err == nil {
		t.Errorf(".agents created for step with no deps")
	}
}

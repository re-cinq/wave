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
	canonicalMap, err := ex.injectDependencyArtifacts(exec, &exec.Pipeline.Steps[1], workspace)
	if err != nil {
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

	// Template resolution under the existing artifacts namespace.
	got := exec.Context.ResolvePlaceholders("{{ artifacts.fetch.pr-context }}")
	if got != canonical {
		t.Errorf("artifacts template resolved to %q, want %q", got, canonical)
	}
	// Template resolution under the new dep-scoped namespace (issue #1452 phase 3).
	gotDeps := exec.Context.ResolvePlaceholders("{{ deps.fetch.pr-context }}")
	if gotDeps != canonical {
		t.Errorf("deps template resolved to %q, want %q", gotDeps, canonical)
	}

	// Returned canonical map should reflect post-injection paths.
	got2, ok := canonicalMap["fetch:pr-context"]
	if !ok {
		t.Fatalf("canonical map missing fetch:pr-context entry; got %v", canonicalMap)
	}
	if got2.Path != canonical {
		t.Errorf("canonical map path = %q, want %q", got2.Path, canonical)
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
	if _, err := ex.injectDependencyArtifacts(exec, &exec.Pipeline.Steps[0], tmp); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".agents")); err == nil {
		t.Errorf(".agents created for step with no deps")
	}
}

// TestBuildDepEnvVars verifies the env-name slug rule and that
// WAVE_DEPS_DIR is always emitted alongside per-artifact entries.
func TestBuildDepEnvVars(t *testing.T) {
	resolved := map[string]ResolvedArtifact{
		"fetch-pr:pr-context":     {DepStep: "fetch-pr", Name: "pr-context", Path: "/ws/.agents/artifacts/fetch-pr/pr-context"},
		"merge-findings:findings": {DepStep: "merge-findings", Name: "findings", Path: "/ws/.agents/artifacts/merge-findings/findings"},
	}
	got := BuildDepEnvVars(resolved, "/ws")

	want := map[string]string{
		"WAVE_DEPS_DIR":                "/ws/.agents/artifacts",
		"WAVE_DEP_FETCH_PR_PR_CONTEXT": "/ws/.agents/artifacts/fetch-pr/pr-context",
		"WAVE_DEP_MERGE_FINDINGS_FINDINGS": "/ws/.agents/artifacts/merge-findings/findings",
	}
	gotMap := make(map[string]string, len(got))
	for _, kv := range got {
		idx := -1
		for i, c := range kv {
			if c == '=' {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatalf("malformed env entry %q", kv)
		}
		gotMap[kv[:idx]] = kv[idx+1:]
	}
	for k, v := range want {
		if gotMap[k] != v {
			t.Errorf("%s = %q, want %q (full env: %v)", k, gotMap[k], v, gotMap)
		}
	}
}

// TestBuildDepEnvVars_EmptyWorkspace returns nil when no workspace path.
func TestBuildDepEnvVars_EmptyWorkspace(t *testing.T) {
	if got := BuildDepEnvVars(map[string]ResolvedArtifact{"x:y": {}}, ""); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

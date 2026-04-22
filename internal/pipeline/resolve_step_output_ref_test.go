package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// newTestExecution produces a bare-bones PipelineExecution suitable for
// resolveStepOutputRef: it only populates ArtifactPaths + Results.
func newTestExecution() *PipelineExecution {
	return &PipelineExecution{
		ArtifactPaths: make(map[string]string),
		Results:       make(map[string]map[string]interface{}),
		States:        make(map[string]string),
	}
}

// writeTempJSON writes a JSON blob to a fresh tempfile and returns the path.
func writeTempJSON(t *testing.T, body string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(f, []byte(body), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return f
}

// TestResolveStepOutputRef_TypedNamedOutput exercises ADR-011 rule 4:
// {{ stepID.out.<name> }} must resolve via exact "<stepID>:<name>" lookup,
// not a prefix scan. This is the deterministic path.
func TestResolveStepOutputRef_TypedNamedOutput(t *testing.T) {
	e := &DefaultPipelineExecutor{}
	exec := newTestExecution()
	path := writeTempJSON(t, `{"url":"https://example.com/pr/1","number":1}`)
	exec.ArtifactPaths["scope:pr"] = path

	// Whole-file
	got := e.resolveStepOutputRef("{{ scope.out.pr }}", exec)
	if got != `{"url":"https://example.com/pr/1","number":1}` {
		t.Errorf("whole-file typed resolve = %q", got)
	}

	// Field extraction
	got = e.resolveStepOutputRef("{{ scope.out.pr.url }}", exec)
	if got != "https://example.com/pr/1" {
		t.Errorf("field typed resolve = %q, want https://example.com/pr/1", got)
	}
}

// TestResolveStepOutputRef_TypedNamed_Miss confirms that a reference to an
// unknown named output leaves the template unchanged (does NOT fall through
// to the legacy prefix scan).
func TestResolveStepOutputRef_TypedNamed_Miss(t *testing.T) {
	e := &DefaultPipelineExecutor{}
	exec := newTestExecution()
	exec.ArtifactPaths["scope:pr"] = writeTempJSON(t, `{"url":"x"}`)

	got := e.resolveStepOutputRef("{{ scope.out.nonexistent }}", exec)
	if got != "{{ scope.out.nonexistent }}" {
		t.Errorf("missing named-output should leave template literal, got %q", got)
	}
}

// TestResolveStepOutputRef_LegacyStillWorks confirms backward-compat: the
// legacy {{ stepID.output }} / {{ stepID.output.field }} form still resolves.
func TestResolveStepOutputRef_LegacyStillWorks(t *testing.T) {
	e := &DefaultPipelineExecutor{}
	exec := newTestExecution()
	path := writeTempJSON(t, `{"url":"https://example.com/pr/2"}`)
	exec.ArtifactPaths["scope:pr"] = path

	got := e.resolveStepOutputRef("{{ scope.output.url }}", exec)
	if got != "https://example.com/pr/2" {
		t.Errorf("legacy field resolve = %q", got)
	}

	got = e.resolveStepOutputRef("{{ scope.output }}", exec)
	if got != `{"url":"https://example.com/pr/2"}` {
		t.Errorf("legacy whole-file resolve = %q", got)
	}
}

// TestResolveStepOutputRef_TypedAndLegacyCoexist verifies both addressing
// modes coexist inside a single template expression.
func TestResolveStepOutputRef_TypedAndLegacyCoexist(t *testing.T) {
	e := &DefaultPipelineExecutor{}
	exec := newTestExecution()
	exec.ArtifactPaths["a:main"] = writeTempJSON(t, `{"x":"A"}`)
	exec.ArtifactPaths["b:main"] = writeTempJSON(t, `{"x":"B"}`)

	got := e.resolveStepOutputRef("[{{ a.out.main.x }}|{{ b.output.x }}]", exec)
	if got != "[A|B]" {
		t.Errorf("mixed typed+legacy = %q, want [A|B]", got)
	}
}

// TestResolveStepOutputRef_MalformedOut makes sure "{{ stepID.out }}" alone
// (no name) returns the literal unchanged rather than panicking or resolving.
func TestResolveStepOutputRef_MalformedOut(t *testing.T) {
	e := &DefaultPipelineExecutor{}
	exec := newTestExecution()
	exec.ArtifactPaths["scope:pr"] = writeTempJSON(t, `{}`)

	got := e.resolveStepOutputRef("{{ scope.out }}", exec)
	if got != "{{ scope.out }}" {
		t.Errorf("bare stepID.out should be a no-op, got %q", got)
	}
}

package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/ontology"
)

// TestResolveWorkspaceStepRefs tests the template resolution for step artifact/output references.
func TestResolveWorkspaceStepRefs(t *testing.T) {
	newExecutor := func() *DefaultPipelineExecutor {
		return NewDefaultPipelineExecutor(adaptertest.NewMockAdapter(), WithOntologyService(ontology.NoOp{}))
	}

	newExecution := func(artifacts map[string]string) *PipelineExecution {
		return &PipelineExecution{
			ArtifactPaths: artifacts,
		}
	}

	t.Run("no template delimiters returns string unchanged", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(nil)
		result, err := ex.resolveWorkspaceStepRefs("plain-string-no-templates", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "plain-string-no-templates" {
			t.Errorf("expected %q, got %q", "plain-string-no-templates", result)
		}
	})

	t.Run("non-steps template left unchanged", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(nil)
		result, err := ex.resolveWorkspaceStepRefs("{{ pipeline_id }}", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "{{ pipeline_id }}" {
			t.Errorf("expected non-steps template to be left unchanged, got %q", result)
		}
	})

	t.Run("artifacts segment with no JSON path returns raw content", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "art.json")
		if err := os.WriteFile(artFile, []byte("  raw content here  \n"), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		result, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart }}", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "raw content here" {
			t.Errorf("expected trimmed raw content, got %q", result)
		}
	})

	t.Run("artifacts segment with JSON path extracts field", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "art.json")
		if err := os.WriteFile(artFile, []byte(`{"branch": "feat/cool"}`), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		result, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart.branch }}", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "feat/cool" {
			t.Errorf("expected %q, got %q", "feat/cool", result)
		}
	})

	t.Run("output segment with no JSON path returns raw content", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "out.json")
		if err := os.WriteFile(artFile, []byte("  output data  \n"), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:default": artFile})

		result, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.output }}", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "output data" {
			t.Errorf("expected trimmed raw content, got %q", result)
		}
	})

	t.Run("output segment with JSON path extracts field", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "out.json")
		if err := os.WriteFile(artFile, []byte(`{"status": "done"}`), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:result": artFile})

		result, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.output.status }}", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "done" {
			t.Errorf("expected %q, got %q", "done", result)
		}
	})

	t.Run("unknown segment returns error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(nil)

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.unknown.field }}", exec)
		if err == nil {
			t.Fatal("expected error for unknown segment")
		}
		if !strings.Contains(err.Error(), "unknown segment") {
			t.Errorf("error should mention 'unknown segment', got: %v", err)
		}
	})

	t.Run("malformed template with only step ID returns error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(nil)

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.myid }}", exec)
		if err == nil {
			t.Fatal("expected error for malformed template")
		}
		if !strings.Contains(err.Error(), "expected steps.<step-id>.artifacts") {
			t.Errorf("error should describe expected format, got: %v", err)
		}
	})

	t.Run("artifacts segment missing artifact name returns error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(nil)

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.myid.artifacts }}", exec)
		if err == nil {
			t.Fatal("expected error for missing artifact name")
		}
		if !strings.Contains(err.Error(), "missing artifact name") {
			t.Errorf("error should mention 'missing artifact name', got: %v", err)
		}
	})

	t.Run("artifact not found returns descriptive error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(map[string]string{})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.missing }}", exec)
		if err == nil {
			t.Fatal("expected error for missing artifact")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should mention 'not found', got: %v", err)
		}
	})

	t.Run("artifact file read error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": "/nonexistent/path/art.json"})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart }}", exec)
		if err == nil {
			t.Fatal("expected error for unreadable artifact file")
		}
		if !strings.Contains(err.Error(), "failed to read artifact") {
			t.Errorf("error should mention 'failed to read artifact', got: %v", err)
		}
	})

	t.Run("artifact JSON path extraction error", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "art.json")
		if err := os.WriteFile(artFile, []byte(`{"name": "test"}`), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart.nonexistent }}", exec)
		if err == nil {
			t.Fatal("expected error for invalid JSON path")
		}
		if !strings.Contains(err.Error(), "JSON path") {
			t.Errorf("error should mention 'JSON path', got: %v", err)
		}
	})

	t.Run("output no artifact found returns error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(map[string]string{})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.output.field }}", exec)
		if err == nil {
			t.Fatal("expected error for missing output")
		}
		if !strings.Contains(err.Error(), "no output found") {
			t.Errorf("error should mention 'no output found', got: %v", err)
		}
	})

	t.Run("output file read error", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:out": "/nonexistent/path/out.json"})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.output.field }}", exec)
		if err == nil {
			t.Fatal("expected error for unreadable output file")
		}
		if !strings.Contains(err.Error(), "failed to read output") {
			t.Errorf("error should mention 'failed to read output', got: %v", err)
		}
	})

	t.Run("output JSON path extraction error", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "out.json")
		if err := os.WriteFile(artFile, []byte(`{"status": "ok"}`), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:result": artFile})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.output.missing }}", exec)
		if err == nil {
			t.Fatal("expected error for invalid output JSON path")
		}
		if !strings.Contains(err.Error(), "JSON path") {
			t.Errorf("error should mention 'JSON path', got: %v", err)
		}
	})

	t.Run("multiple templates in single string", func(t *testing.T) {
		tmpDir := t.TempDir()
		artA := filepath.Join(tmpDir, "a.json")
		artB := filepath.Join(tmpDir, "b.json")
		if err := os.WriteFile(artA, []byte(`{"branch": "feat-a"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(artB, []byte(`{"branch": "feat-b"}`), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{
			"stepa:art": artA,
			"stepb:art": artB,
		})

		result, err := ex.resolveWorkspaceStepRefs(
			"prefix-{{ steps.stepa.artifacts.art.branch }}-{{ steps.stepb.artifacts.art.branch }}-suffix",
			exec,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "prefix-feat-a-feat-b-suffix"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("mixed steps and non-steps templates", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "art.json")
		if err := os.WriteFile(artFile, []byte(`{"branch": "feat-x"}`), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		result, err := ex.resolveWorkspaceStepRefs(
			"{{ steps.mystep.artifacts.myart.branch }}-{{ pipeline_id }}",
			exec,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// steps ref resolved, pipeline_id left unchanged
		if !strings.HasPrefix(result, "feat-x-") {
			t.Errorf("expected steps ref to be resolved, got %q", result)
		}
		if !strings.Contains(result, "{{ pipeline_id }}") {
			t.Errorf("expected non-steps ref to be left unchanged, got %q", result)
		}
	})

	t.Run("early return when first error already set", func(t *testing.T) {
		ex := newExecutor()
		exec := newExecution(map[string]string{})

		// Both templates reference missing artifacts — only the first error matters
		_, err := ex.resolveWorkspaceStepRefs(
			"{{ steps.a.artifacts.missing }}-{{ steps.b.artifacts.missing }}",
			exec,
		)
		if err == nil {
			t.Fatal("expected error")
		}
		// Error should reference step "a" (the first failing ref)
		if !strings.Contains(err.Error(), `"missing"`) && !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected first-match error, got: %v", err)
		}
	})

	t.Run("empty artifact file returns empty string for raw content", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "empty.json")
		if err := os.WriteFile(artFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		result, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart }}", exec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("non-JSON artifact with JSON path returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "notjson.txt")
		if err := os.WriteFile(artFile, []byte("this is not json"), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		_, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart.field }}", exec)
		if err == nil {
			t.Fatal("expected error for non-JSON content with JSON path")
		}
		if !strings.Contains(err.Error(), "JSON path") {
			t.Errorf("error should mention 'JSON path', got: %v", err)
		}
	})

	t.Run("concurrent access to ArtifactPaths is safe", func(t *testing.T) {
		tmpDir := t.TempDir()
		artFile := filepath.Join(tmpDir, "art.json")
		if err := os.WriteFile(artFile, []byte("concurrent-data"), 0644); err != nil {
			t.Fatal(err)
		}

		ex := newExecutor()
		exec := newExecution(map[string]string{"mystep:myart": artFile})

		var wg sync.WaitGroup
		errs := make(chan error, 10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, err := ex.resolveWorkspaceStepRefs("{{ steps.mystep.artifacts.myart }}", exec)
				if err != nil {
					errs <- err
					return
				}
				if result != "concurrent-data" {
					errs <- fmt.Errorf("expected 'concurrent-data', got %q", result)
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Errorf("concurrent access error: %v", err)
		}
	})
}

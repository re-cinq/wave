package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempPipeline writes a pipeline YAML into .agents/pipelines/ relative
// to a fresh tempDir, then chdirs into tempDir so LoadPipelineByName
// (which resolves paths relative to the working directory) can find it.
func writeTempPipeline(t *testing.T, name, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".agents", "pipelines"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, ".agents", "pipelines", name+".yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestLoadPipelineByName_Roundtrip(t *testing.T) {
	body := `metadata:
  name: smoke
  description: smoke test
steps:
  - id: a
    persona: navigator
  - id: b
    persona: navigator
    dependencies: [a]
`
	writeTempPipeline(t, "smoke", body)

	p, err := LoadPipelineByName("smoke")
	if err != nil {
		t.Fatalf("LoadPipelineByName: %v", err)
	}
	if p.Metadata.Name != "smoke" || len(p.Steps) != 2 {
		t.Fatalf("loaded pipeline missing fields: %+v", p)
	}
}

func TestLoadPipelineByName_RejectsTraversal(t *testing.T) {
	if _, err := LoadPipelineByName("../etc/passwd"); err == nil {
		t.Error("expected error for traversal-shaped name")
	}
	if _, err := LoadPipelineByName(""); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestPipelineHasStep(t *testing.T) {
	body := `metadata:
  name: hasstep
steps:
  - id: alpha
    persona: navigator
  - id: beta
    persona: navigator
`
	writeTempPipeline(t, "hasstep", body)

	got, err := PipelineHasStep("hasstep", "alpha")
	if err != nil || !got {
		t.Errorf("PipelineHasStep(alpha) = %v,%v want true,nil", got, err)
	}
	got, err = PipelineHasStep("hasstep", "missing")
	if err != nil || got {
		t.Errorf("PipelineHasStep(missing) = %v,%v want false,nil", got, err)
	}
	if _, err := PipelineHasStep("nonexistent", "x"); err == nil {
		t.Error("expected error for missing pipeline")
	}
}

func TestForkController_ResumeStepAfter(t *testing.T) {
	body := `metadata:
  name: order
steps:
  - id: one
    persona: navigator
  - id: two
    persona: navigator
  - id: three
    persona: navigator
`
	writeTempPipeline(t, "order", body)

	fc := NewForkController(nil)

	got, err := fc.ResumeStepAfter("order", "one")
	if err != nil || got != "two" {
		t.Errorf("ResumeStepAfter(one) = %q,%v want two,nil", got, err)
	}
	got, err = fc.ResumeStepAfter("order", "three")
	if err != nil || got != "" {
		t.Errorf("ResumeStepAfter(three) = %q,%v want empty,nil", got, err)
	}
	got, err = fc.ResumeStepAfter("order", "ghost")
	if err != nil || got != "" {
		t.Errorf("ResumeStepAfter(ghost) = %q,%v want empty,nil", got, err)
	}
}

func TestForkController_PlanRewind(t *testing.T) {
	body := `metadata:
  name: rewind
steps:
  - id: a
    persona: navigator
  - id: b
    persona: navigator
  - id: c
    persona: navigator
  - id: d
    persona: navigator
`
	writeTempPipeline(t, "rewind", body)

	fc := NewForkController(nil)

	plan, err := fc.PlanRewind("rewind", "b")
	if err != nil {
		t.Fatalf("PlanRewind: %v", err)
	}
	if plan.StepIndex != 1 {
		t.Errorf("StepIndex = %d, want 1", plan.StepIndex)
	}
	wantDeleted := []string{"c", "d"}
	if len(plan.StepsDeleted) != len(wantDeleted) {
		t.Fatalf("StepsDeleted = %v, want %v", plan.StepsDeleted, wantDeleted)
	}
	for i, s := range wantDeleted {
		if plan.StepsDeleted[i] != s {
			t.Errorf("StepsDeleted[%d] = %q, want %q", i, plan.StepsDeleted[i], s)
		}
	}

	plan, err = fc.PlanRewind("rewind", "ghost")
	if err != nil {
		t.Fatalf("PlanRewind ghost: %v", err)
	}
	if plan.StepIndex != -1 || len(plan.StepsDeleted) != 0 {
		t.Errorf("missing-step plan = %+v, want StepIndex=-1 empty deleted", plan)
	}
}

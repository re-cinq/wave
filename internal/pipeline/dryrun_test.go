package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

// buildManifestWithPersonas creates a minimal manifest with navigator and craftsman.
func buildManifestWithPersonas() *manifest.Manifest {
	m := &manifest.Manifest{}
	m.Adapters = map[string]manifest.Adapter{
		"claude": {Binary: "claude"},
	}
	m.Personas = map[string]manifest.Persona{
		"navigator": {
			Adapter:          "claude",
			SystemPromptFile: "navigator.md",
		},
		"craftsman": {
			Adapter:          "claude",
			SystemPromptFile: "craftsman.md",
		},
	}
	return m
}

// buildSimplePipeline creates a two-step pipeline for baseline tests.
func buildSimplePipeline() *Pipeline {
	return &Pipeline{
		Kind: "WavePipeline",
		Metadata: PipelineMetadata{
			Name: "test-pipeline",
		},
		Steps: []Step{
			{
				ID:      "navigate",
				Persona: "navigator",
				Memory:  MemoryConfig{Strategy: "fresh"},
				Exec:    ExecConfig{Type: "prompt", Source: "Analyze {{ input }}"},
				OutputArtifacts: []ArtifactDef{
					{Name: "analysis", Path: ".agents/artifacts/analysis.json", Type: "json"},
				},
			},
			{
				ID:           "implement",
				Persona:      "craftsman",
				Dependencies: []string{"navigate"},
				Memory: MemoryConfig{
					Strategy: "fresh",
					InjectArtifacts: []ArtifactRef{
						{Step: "navigate", Artifact: "analysis", As: "analysis"},
					},
				},
				Exec: ExecConfig{Type: "prompt", Source: "Implement {{ input }}"},
			},
		},
	}
}

func TestDryRunValidator_ValidPipeline(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()

	report := v.Validate(p, m)
	if report.HasErrors() {
		t.Fatalf("expected no errors for valid pipeline, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_MissingPersona(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Persona = "unknown-persona"

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for unknown persona")
	}
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "navigate" && f.Field == "persona" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected persona-not-found error on step navigate, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_MissingPersonaWhenNoManifest(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := buildSimplePipeline()

	// No manifest: persona check is skipped, no error expected for unknown persona.
	report := v.Validate(p, nil)
	for _, f := range report.Findings {
		if f.Field == "persona" && f.Severity == SeverityError {
			t.Fatalf("unexpected persona error when manifest is nil: %s", f)
		}
	}
}

func TestDryRunValidator_UnknownExecType(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Exec.Type = "magic"

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for unknown exec.type")
	}
}

func TestDryRunValidator_MissingExecSource(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Exec.Source = ""
	p.Steps[0].Exec.SourcePath = ""

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error when prompt has no source or source_path")
	}
}

func TestDryRunValidator_InjectArtifactFromUnknownStep(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[1].Memory.InjectArtifacts = []ArtifactRef{
		{Step: "nonexistent", Artifact: "foo", As: "foo"},
	}

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for inject_artifact referencing unknown step")
	}
}

func TestDryRunValidator_InjectArtifactMissingDependency(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	// Remove the dependency listing.
	p.Steps[1].Dependencies = nil

	report := v.Validate(p, m)
	// Should produce a warning, not an error.
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityWarning && f.StepID == "implement" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning about missing dependency, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_UnknownArtifactName_Warning(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[1].Memory.InjectArtifacts = []ArtifactRef{
		{Step: "navigate", Artifact: "unknown-art", As: "x"},
	}

	report := v.Validate(p, m)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityWarning && f.StepID == "implement" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected warning about unknown artifact name, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_ContractSchemaPathMissing(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Handover = HandoverConfig{
		Contract: ContractConfig{
			Type:       "json_schema",
			SchemaPath: "/nonexistent/schema.json",
		},
	}

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for missing schema file")
	}
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "navigate" && f.Field == "handover.contract.schema_path" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected schema_path error, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_ContractSchemaPathExists(t *testing.T) {
	dir := t.TempDir()
	schemaFile := filepath.Join(dir, "schema.json")
	schema := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
	}
	data, _ := json.Marshal(schema)
	if err := os.WriteFile(schemaFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Handover = HandoverConfig{
		Contract: ContractConfig{
			Type:       "json_schema",
			SchemaPath: schemaFile,
		},
	}

	report := v.Validate(p, m)
	for _, f := range report.Findings {
		if f.StepID == "navigate" && f.Field == "handover.contract.schema_path" {
			t.Fatalf("unexpected schema_path error: %s", f)
		}
	}
}

func TestDryRunValidator_UnknownContractType(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Handover = HandoverConfig{
		Contract: ContractConfig{Type: "magic_validator"},
	}

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for unknown contract type")
	}
}

func TestDryRunValidator_JSONSchemaContractMissingSchema(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Handover = HandoverConfig{
		Contract: ContractConfig{Type: "json_schema"},
	}

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for json_schema without schema or schema_path")
	}
}

func TestDryRunValidator_InvalidContractOnFailure(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Handover = HandoverConfig{
		Contract: ContractConfig{Type: "non_empty_file", OnFailure: "explode"},
	}

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for unknown on_failure value")
	}
}

func TestDryRunValidator_TestSuiteMissingCommand(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	m := buildManifestWithPersonas()
	p := buildSimplePipeline()
	p.Steps[0].Handover = HandoverConfig{
		Contract: ContractConfig{Type: "test_suite"},
	}

	report := v.Validate(p, m)
	if !report.HasErrors() {
		t.Fatal("expected error for test_suite without command")
	}
}

// --- Gate validation ---

func TestDryRunValidator_GateMissingType(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{ID: "wait", Gate: &GateConfig{}},
		},
	}

	report := v.Validate(p, nil)
	if !report.HasErrors() {
		t.Fatal("expected error for gate without type")
	}
}

func TestDryRunValidator_GateUnknownType(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{ID: "wait", Gate: &GateConfig{Type: "magic_gate"}},
		},
	}

	report := v.Validate(p, nil)
	if !report.HasErrors() {
		t.Fatal("expected error for unknown gate type")
	}
}

func TestDryRunValidator_GateTimerMissingTimeout(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{ID: "wait", Gate: &GateConfig{Type: "timer"}},
		},
	}

	report := v.Validate(p, nil)
	if !report.HasErrors() {
		t.Fatal("expected error for timer gate without timeout")
	}
}

func TestDryRunValidator_GateApprovalValid(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{ID: "wait", Gate: &GateConfig{Type: "approval", Auto: true}},
		},
	}

	report := v.Validate(p, nil)
	if report.HasErrors() {
		t.Fatalf("expected no errors for valid approval gate, got:\n%s", report.Format())
	}
}

// --- Iterate validation ---

func TestDryRunValidator_IterateMissingOver(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID:          "fan-out",
				SubPipeline: "some-pipeline",
				Iterate:     &IterateConfig{Mode: "sequential"},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "fan-out" && f.Field == "iterate.over" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected iterate.over required error, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_IterateMissingPipeline(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID:      "fan-out",
				Iterate: &IterateConfig{Over: "{{ input }}", Mode: "sequential"},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "fan-out" && f.Field == "pipeline" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected pipeline required error for iterate, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_IterateUnknownMode(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID:          "fan-out",
				SubPipeline: "some-pipeline",
				Iterate:     &IterateConfig{Over: "{{ input }}", Mode: "explode"},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "fan-out" && f.Field == "iterate.mode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected iterate.mode error, got:\n%s", report.Format())
	}
}

// --- Branch validation ---

func TestDryRunValidator_BranchMissingOn(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID:     "choose",
				Branch: &BranchConfig{Cases: map[string]string{"a": "skip"}},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "choose" && f.Field == "branch.on" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected branch.on required error, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_BranchNoCases(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID:     "choose",
				Branch: &BranchConfig{On: "{{ input }}"},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "choose" && f.Field == "branch.cases" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected branch.cases required error, got:\n%s", report.Format())
	}
}

// --- Loop validation ---

func TestDryRunValidator_LoopMaxIterationsZero(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{ID: "loop-step", Loop: &LoopConfig{MaxIterations: 0}},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "loop-step" && f.Field == "loop.max_iterations" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected loop.max_iterations error, got:\n%s", report.Format())
	}
}

// --- Aggregate validation ---

func TestDryRunValidator_AggregateUnknownStrategy(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID: "collect",
				Aggregate: &AggregateConfig{
					From:     "{{ input }}",
					Into:     "/tmp/out.json",
					Strategy: "magic",
				},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "collect" && f.Field == "aggregate.strategy" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected aggregate.strategy error, got:\n%s", report.Format())
	}
}

func TestDryRunValidator_AggregateMissingFrom(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID: "collect",
				Aggregate: &AggregateConfig{
					Into:     "/tmp/out.json",
					Strategy: "concat",
				},
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "collect" && f.Field == "aggregate.from" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected aggregate.from required error, got:\n%s", report.Format())
	}
}

// --- DAG error ---

func TestDryRunValidator_DAGCycle(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID:           "step-a",
				Persona:      "navigator",
				Dependencies: []string{"step-b"},
				Memory:       MemoryConfig{Strategy: "fresh"},
				Exec:         ExecConfig{Type: "prompt", Source: "x"},
			},
			{
				ID:           "step-b",
				Persona:      "craftsman",
				Dependencies: []string{"step-a"},
				Memory:       MemoryConfig{Strategy: "fresh"},
				Exec:         ExecConfig{Type: "prompt", Source: "y"},
			},
		},
	}

	report := v.Validate(p, nil)
	if !report.HasErrors() {
		t.Fatal("expected DAG cycle error")
	}
	// DAG error stops further checks.
	if len(report.Findings) != 1 {
		t.Fatalf("expected exactly 1 finding (the DAG error), got %d", len(report.Findings))
	}
}

// --- DryRunReport helpers ---

func TestDryRunReport_Format_NoFindings(t *testing.T) {
	r := &DryRunReport{PipelineName: "my-pipeline"}
	out := r.Format()
	if out == "" {
		t.Fatal("Format() should return non-empty string")
	}
	if r.HasErrors() {
		t.Fatal("empty report should not have errors")
	}
	if r.ErrorCount() != 0 || r.WarningCount() != 0 {
		t.Fatal("counts should be zero")
	}
}

func TestDryRunReport_Counts(t *testing.T) {
	r := &DryRunReport{
		PipelineName: "p",
		Findings: []ValidationFinding{
			{Severity: SeverityError, StepID: "a", Message: "err1"},
			{Severity: SeverityError, StepID: "b", Message: "err2"},
			{Severity: SeverityWarning, StepID: "c", Message: "warn1"},
		},
	}
	if r.ErrorCount() != 2 {
		t.Fatalf("expected 2 errors, got %d", r.ErrorCount())
	}
	if r.WarningCount() != 1 {
		t.Fatalf("expected 1 warning, got %d", r.WarningCount())
	}
	if !r.HasErrors() {
		t.Fatal("should have errors")
	}
}

func TestValidationFinding_String(t *testing.T) {
	f := ValidationFinding{Severity: SeverityError, StepID: "foo", Field: "persona", Message: "not found"}
	s := f.String()
	if s == "" {
		t.Fatal("String() should not be empty")
	}

	fNoField := ValidationFinding{Severity: SeverityWarning, StepID: "bar", Message: "check this"}
	s2 := fNoField.String()
	if s2 == "" {
		t.Fatal("String() should not be empty when Field is empty")
	}

	fNoStep := ValidationFinding{Severity: SeverityError, Message: "global issue"}
	s3 := fNoStep.String()
	if s3 == "" {
		t.Fatal("String() should not be empty when StepID is empty")
	}
}

func TestDryRunValidator_UnbalancedTemplate(t *testing.T) {
	v := NewDryRunValidator(".agents/pipelines")
	p := &Pipeline{
		Kind:     "WavePipeline",
		Metadata: PipelineMetadata{Name: "test"},
		Steps: []Step{
			{
				ID: "bad",
				Iterate: &IterateConfig{
					Over: "{{ input",
					Mode: "sequential",
				},
				SubPipeline: "some-pipeline",
			},
		},
	}

	report := v.Validate(p, nil)
	found := false
	for _, f := range report.Findings {
		if f.Severity == SeverityError && f.StepID == "bad" && f.Field == "iterate.over" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unbalanced template error on iterate.over, got:\n%s", report.Format())
	}
}

func TestDryRun_SubPipelineConfig(t *testing.T) {
	tmpDir := t.TempDir()
	v := NewDryRunValidator(tmpDir)
	m := buildManifestWithPersonas()

	t.Run("valid config", func(t *testing.T) {
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test"},
			Steps: []Step{
				{
					ID:          "sub",
					SubPipeline: "child",
					Config: &SubPipelineConfig{
						Inject:  []string{"plan"},
						Extract: []string{"impl"},
						Timeout: "30m",
					},
				},
			},
		}
		report := v.Validate(p, m)
		// Should have no errors related to config (may have sub-pipeline not found)
		for _, f := range report.Findings {
			if f.Field == "config" && f.Severity == SeverityError {
				t.Errorf("unexpected config error: %s", f.Message)
			}
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test"},
			Steps: []Step{
				{
					ID:          "sub",
					SubPipeline: "child",
					Config: &SubPipelineConfig{
						Timeout: "invalid",
					},
				},
			},
		}
		report := v.Validate(p, m)
		if !report.HasErrors() {
			t.Error("expected error for invalid timeout")
		}
	})

	t.Run("config without pipeline", func(t *testing.T) {
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test"},
			Steps: []Step{
				{
					ID:      "regular",
					Persona: "navigator",
					Exec:    ExecConfig{Type: "prompt", Source: "test"},
					Config: &SubPipelineConfig{
						Inject: []string{"plan"},
					},
				},
			},
		}
		report := v.Validate(p, m)
		found := false
		for _, f := range report.Findings {
			if f.Field == "config" && f.Severity == SeverityWarning {
				found = true
			}
		}
		if !found {
			t.Error("expected warning for config without pipeline")
		}
	})

	t.Run("unbalanced stop_condition template", func(t *testing.T) {
		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "test"},
			Steps: []Step{
				{
					ID:          "sub",
					SubPipeline: "child",
					Config: &SubPipelineConfig{
						StopCondition: "{{context.done",
					},
				},
			},
		}
		report := v.Validate(p, m)
		found := false
		for _, f := range report.Findings {
			if f.Field == "config.stop_condition" && f.Severity == SeverityError {
				found = true
			}
		}
		if !found {
			t.Error("expected error for unbalanced stop_condition template")
		}
	})
}

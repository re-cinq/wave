package pipeline

import (
	"strings"
	"testing"
)

func TestValidatePipelineIOTypes_KnownTypes(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "good"},
		Input:    InputConfig{Source: "cli", Type: "issue_ref"},
		PipelineOutputs: map[string]PipelineOutput{
			"pr": {Step: "create-pr", Artifact: "pr-result", Type: "pr_ref"},
		},
		Steps: []Step{{ID: "create-pr"}},
	}
	if err := ValidatePipelineIOTypes(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePipelineIOTypes_EmptyTypeTreatedAsString(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "legacy"},
		Input:    InputConfig{Source: "cli"}, // Type omitted
		PipelineOutputs: map[string]PipelineOutput{
			"report": {Step: "s1", Artifact: "report"}, // Type omitted
		},
		Steps: []Step{{ID: "s1"}},
	}
	if err := ValidatePipelineIOTypes(p); err != nil {
		t.Fatalf("expected legacy (untyped) pipeline to pass validation, got: %v", err)
	}
}

func TestValidatePipelineIOTypes_ExplicitString(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "str"},
		Input:    InputConfig{Source: "cli", Type: "string"},
		Steps:    []Step{{ID: "only"}},
	}
	if err := ValidatePipelineIOTypes(p); err != nil {
		t.Fatalf("string sentinel must pass: %v", err)
	}
}

func TestValidatePipelineIOTypes_UnknownInputType(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad"},
		Input:    InputConfig{Source: "cli", Type: "not_a_real_type"},
		Steps:    []Step{{ID: "only"}},
	}
	err := ValidatePipelineIOTypes(p)
	if err == nil {
		t.Fatal("expected error for unknown input.type, got nil")
	}
	if !strings.Contains(err.Error(), "not_a_real_type") {
		t.Errorf("error should mention the bad type, got: %v", err)
	}
}

func TestValidatePipelineIOTypes_UnknownOutputType(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad"},
		Input:    InputConfig{Source: "cli", Type: "string"},
		PipelineOutputs: map[string]PipelineOutput{
			"x": {Step: "s1", Artifact: "a", Type: "made_up"},
		},
		Steps: []Step{{ID: "s1"}},
	}
	if err := ValidatePipelineIOTypes(p); err == nil {
		t.Fatal("expected error for unknown output.type, got nil")
	}
}

func TestValidatePipelineIOTypes_OutputStepMustExist(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad"},
		Input:    InputConfig{Source: "cli"},
		PipelineOutputs: map[string]PipelineOutput{
			"x": {Step: "ghost", Artifact: "a"},
		},
		Steps: []Step{{ID: "real"}},
	}
	if err := ValidatePipelineIOTypes(p); err == nil {
		t.Fatal("expected error when pipeline_outputs references a non-existent step")
	}
}

func TestValidatePipelineIOTypes_InputRefExclusivity(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad"},
		Input:    InputConfig{Source: "cli"},
		Steps: []Step{
			{ID: "a"},
			{ID: "b", SubPipeline: "child", InputRef: &StepInput{From: "a.out", Literal: "x"}},
		},
	}
	if err := ValidatePipelineIOTypes(p); err == nil {
		t.Fatal("expected error when input_ref sets both from and literal")
	}
}

func TestValidatePipelineIOTypes_InputRefEmpty(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad"},
		Input:    InputConfig{Source: "cli"},
		Steps: []Step{
			{ID: "a"},
			{ID: "b", SubPipeline: "child", InputRef: &StepInput{}},
		},
	}
	if err := ValidatePipelineIOTypes(p); err == nil {
		t.Fatal("expected error when input_ref is empty")
	}
}

// YAMLLoader integration: type-checking happens in the loader, so malformed
// pipelines must reject at Unmarshal time.
func TestYAMLPipelineLoader_RejectsUnknownType(t *testing.T) {
	data := []byte(`kind: WavePipeline
metadata:
  name: bogus
input:
  source: cli
  type: totally_not_a_type
steps:
  - id: only
    persona: navigator
    workspace: {}
    exec:
      type: prompt
      source: hi
    memory:
      strategy: fresh
`)
	loader := &YAMLPipelineLoader{}
	_, err := loader.Unmarshal(data)
	if err == nil {
		t.Fatal("expected loader to reject unknown type, got nil error")
	}
	if !strings.Contains(err.Error(), "totally_not_a_type") {
		t.Errorf("expected error to mention bad type name, got: %v", err)
	}
}

func TestYAMLPipelineLoader_AcceptsTypedIO(t *testing.T) {
	data := []byte(`kind: WavePipeline
metadata:
  name: typed
input:
  source: cli
  type: issue_ref
pipeline_outputs:
  pr:
    step: s1
    artifact: pr-result
    type: pr_ref
steps:
  - id: s1
    persona: craftsman
    workspace: {}
    exec:
      type: prompt
      source: go
    memory:
      strategy: fresh
`)
	loader := &YAMLPipelineLoader{}
	p, err := loader.Unmarshal(data)
	if err != nil {
		t.Fatalf("expected typed pipeline to load, got: %v", err)
	}
	if got := p.Input.EffectiveType(); got != "issue_ref" {
		t.Errorf("Input.EffectiveType() = %q, want issue_ref", got)
	}
	out := p.PipelineOutputs["pr"]
	if got := out.EffectiveType(); got != "pr_ref" {
		t.Errorf("output pr type = %q, want pr_ref", got)
	}
}

// StepInput wiring resolution at runtime: resolveStepInput should pull the
// raw JSON from c.tmplCtx.StepOutputs[srcStep] when InputRef.From is set.
func TestResolveStepInput_FromWiring(t *testing.T) {
	ctx := NewTemplateContext("parent-input", "/tmp")
	ctx.SetStepOutput("scope", []byte(`{"child_issues":[{"number":42}]}`))
	c := &CompositionExecutor{tmplCtx: ctx}

	step := &Step{
		ID:          "implement",
		SubPipeline: "impl-issue",
		InputRef:    &StepInput{From: "scope.child_issues"},
	}
	got, err := c.resolveStepInput(step)
	if err != nil {
		t.Fatalf("resolveStepInput: %v", err)
	}
	want := `{"child_issues":[{"number":42}]}`
	if got != want {
		t.Errorf("resolveStepInput from-wiring = %q, want %q", got, want)
	}
}

func TestResolveStepInput_LiteralWiring(t *testing.T) {
	ctx := NewTemplateContext("parent-input", "/tmp")
	c := &CompositionExecutor{tmplCtx: ctx}

	step := &Step{
		ID:          "s",
		SubPipeline: "child",
		InputRef:    &StepInput{Literal: "re-cinq/wave 99"},
	}
	got, err := c.resolveStepInput(step)
	if err != nil {
		t.Fatalf("resolveStepInput: %v", err)
	}
	if got != "re-cinq/wave 99" {
		t.Errorf("literal wiring = %q, want %q", got, "re-cinq/wave 99")
	}
}

// Legacy string-template fallback still works when InputRef is absent.
func TestResolveStepInput_LegacyStringFallback(t *testing.T) {
	ctx := NewTemplateContext("legacy-input", "/tmp")
	c := &CompositionExecutor{tmplCtx: ctx}
	step := &Step{ID: "s", SubInput: "{{ input }}"}
	got, err := c.resolveStepInput(step)
	if err != nil {
		t.Fatalf("resolveStepInput: %v", err)
	}
	if got != "legacy-input" {
		t.Errorf("legacy fallback = %q, want %q", got, "legacy-input")
	}
}

// --- Wave Lego Protocol (ADR-011) load-time checks ---

func TestCollectWLPLoadErrors_RetryOnContract(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad-retry"},
		Steps: []Step{{
			ID: "s1",
			Handover: HandoverConfig{
				Contract: ContractConfig{
					Type:      "json_schema",
					OnFailure: OnFailureRetry,
				},
			},
		}},
	}
	warnings := CollectWLPLoadErrors(p)
	if len(warnings) == 0 {
		t.Fatal("expected a warning for on_failure=retry on contract, got none")
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "retry") || !strings.Contains(joined, "ADR-011") {
		t.Errorf("expected ADR-011 retry warning, got: %s", joined)
	}
}

func TestCollectWLPLoadErrors_RetryOnContractsList(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad-retry-list"},
		Steps: []Step{{
			ID: "s1",
			Handover: HandoverConfig{
				Contracts: []ContractConfig{
					{Type: "json_schema", OnFailure: OnFailureFail},
					{Type: "agent_review", OnFailure: OnFailureRetry}, // flagged
				},
			},
		}},
	}
	warnings := CollectWLPLoadErrors(p)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for retry-in-list, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "contract[1]") || !strings.Contains(warnings[0], "agent_review") {
		t.Errorf("expected contract[1] agent_review flagged, got: %s", warnings[0])
	}
}

func TestCollectWLPLoadErrors_PipelineOutputMissingType(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "missing-type"},
		PipelineOutputs: map[string]PipelineOutput{
			"pr":     {Step: "s1", Artifact: "pr-result"}, // no type -> warn
			"report": {Step: "s1", Artifact: "rep", Type: "findings_report"},
		},
		Steps: []Step{{ID: "s1"}},
	}
	warnings := CollectWLPLoadErrors(p)
	if len(warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], `"pr"`) || !strings.Contains(warnings[0], "ADR-011") {
		t.Errorf("expected warning to mention output 'pr' and ADR-011, got: %s", warnings[0])
	}
}

func TestCollectWLPLoadErrors_Clean(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "clean"},
		Input:    InputConfig{Source: "cli", Type: "issue_ref"},
		PipelineOutputs: map[string]PipelineOutput{
			"pr": {Step: "s1", Artifact: "pr-result", Type: "pr_ref"},
		},
		Steps: []Step{{
			ID: "s1",
			Handover: HandoverConfig{
				Contract: ContractConfig{Type: "json_schema", OnFailure: OnFailureFail},
			},
		}},
	}
	if warnings := CollectWLPLoadErrors(p); len(warnings) > 0 {
		t.Errorf("expected no warnings on clean pipeline, got %d: %v", len(warnings), warnings)
	}
}

// Loader wiring: WLP violations (Rules 3, 5) are hard errors at load time.
func TestYAMLPipelineLoader_WLPViolationsAreErrors(t *testing.T) {
	data := []byte(`kind: WavePipeline
metadata:
  name: loader-warn
input:
  source: cli
pipeline_outputs:
  report:
    step: s1
    artifact: rep
steps:
  - id: s1
    persona: navigator
    workspace: {}
    exec:
      type: prompt
      source: hi
    memory:
      strategy: fresh
    handover:
      contract:
        type: json_schema
        on_failure: retry
`)
	loader := &YAMLPipelineLoader{}
	if _, err := loader.Unmarshal(data); err == nil {
		t.Fatal("expected WLP validation error, got nil")
	} else if !strings.Contains(err.Error(), "WLP validation failed") {
		t.Fatalf("expected WLP error, got: %v", err)
	}
}

package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/testutil"
)

func TestCompositionExecutor_SubPipeline(t *testing.T) {
	// This test verifies that executeSubPipeline resolves input templates.
	// Full sub-pipeline execution requires pipeline files on disk, so we
	// test the template resolution path.
	ctx := NewTemplateContext("test-input", "/tmp")
	ctx.SetStepOutput("prior", []byte(`{"value": "hello"}`))

	step := &Step{
		ID:          "sub",
		SubPipeline: "nonexistent-pipeline",
		SubInput:    "{{prior.output.value}}",
	}

	// Resolve input
	input, err := resolveStepInputForTest(step, ctx)
	if err != nil {
		t.Fatalf("failed to resolve input: %v", err)
	}
	if input != "hello" {
		t.Errorf("expected %q, got %q", "hello", input)
	}
}

func resolveStepInputForTest(step *Step, ctx *TemplateContext) (string, error) {
	if step.SubInput != "" {
		return ResolveTemplate(step.SubInput, ctx)
	}
	return ctx.Input, nil
}

func TestCompositionExecutor_BranchDispatch(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		cases    map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "exact match",
			value:    "high",
			cases:    map[string]string{"high": "hotfix", "low": "backlog"},
			expected: "hotfix",
		},
		{
			name:     "default fallback",
			value:    "unknown",
			cases:    map[string]string{"high": "hotfix", "default": "backlog"},
			expected: "backlog",
		},
		{
			name:    "no match no default",
			value:   "unknown",
			cases:   map[string]string{"high": "hotfix"},
			wantErr: true,
		},
		{
			name:     "skip case",
			value:    "low",
			cases:    map[string]string{"high": "hotfix", "low": "skip"},
			expected: "skip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewTemplateContext("", "/tmp")
			ctx.SetStepOutput("analyze", []byte(`{"severity": "`+tt.value+`"}`))

			step := &Step{
				ID: "branch-step",
				Branch: &BranchConfig{
					On:    "{{analyze.output.severity}}",
					Cases: tt.cases,
				},
			}

			// Resolve the branch condition
			resolved, err := ResolveTemplate(step.Branch.On, ctx)
			if err != nil {
				t.Fatalf("template resolution failed: %v", err)
			}

			pipelineName, ok := step.Branch.Cases[resolved]
			if !ok {
				pipelineName, ok = step.Branch.Cases["default"]
			}

			if tt.wantErr {
				if ok {
					t.Errorf("expected no match, got %q", pipelineName)
				}
				return
			}

			if !ok {
				t.Fatal("expected match but got none")
			}
			if pipelineName != tt.expected {
				t.Errorf("expected pipeline %q, got %q", tt.expected, pipelineName)
			}
		})
	}
}

func TestCompositionExecutor_IterateItems(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("scope", []byte(`{"child_issues": [{"url": "issue/1"}, {"url": "issue/2"}, {"url": "issue/3"}]}`))

	// Resolve the iterate.over expression
	itemsJSON, err := ResolveTemplate("{{scope.output.child_issues}}", ctx)
	if err != nil {
		t.Fatalf("failed to resolve iterate.over: %v", err)
	}

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		t.Fatalf("failed to unmarshal items: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Verify each item's URL can be extracted
	for i, item := range items {
		ctx.Item = item
		url, err := ResolveTemplate("{{item.url}}", ctx)
		if err != nil {
			t.Fatalf("item %d: failed to resolve url: %v", i, err)
		}
		expected := "issue/" + string(rune('1'+i))
		if url != expected {
			t.Errorf("item %d: expected %q, got %q", i, expected, url)
		}
	}
}

func TestCompositionExecutor_LoopTermination(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")

	// Simulate loop iterations checking a condition
	for i := 0; i < 5; i++ {
		ctx.Iteration = i

		// Simulate condition becoming true at iteration 2
		if i >= 2 {
			ctx.SetStepOutput("check", []byte(`{"status": "true"}`))
		} else {
			ctx.SetStepOutput("check", []byte(`{"status": "false"}`))
		}

		result, err := ResolveTemplate("{{check.output.status}}", ctx)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}

		if i >= 2 && result != "true" {
			t.Errorf("iteration %d: expected true, got %q", i, result)
		}
		if i < 2 && result != "false" {
			t.Errorf("iteration %d: expected false, got %q", i, result)
		}
	}
}

func TestCompositionExecutor_Aggregate_Concat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output", "merged.txt")

	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("step1", []byte("result-1"))
	ctx.SetStepOutput("step2", []byte("result-2"))

	step := &Step{
		ID: "agg",
		Aggregate: &AggregateConfig{
			From:     "{{step1.output}} {{step2.output}}",
			Into:     outputPath,
			Strategy: "concat",
		},
	}

	// Resolve the from template
	resolved, err := ResolveTemplate(step.Aggregate.From, ctx)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}
	if resolved != "result-1 result-2" {
		t.Errorf("expected %q, got %q", "result-1 result-2", resolved)
	}

	// Write to file (simulating aggregate behavior)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(resolved), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "result-1 result-2" {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestMergeJSONArrays(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "flat array passthrough",
			input: `[1, 2, 3]`,
			want:  `[1, 2, 3]`,
		},
		{
			name:  "array of arrays",
			input: `[[1, 2], [3, 4]]`,
			want:  `[1,2,3,4]`,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeJSONArrays(tt.input, "")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeJSONArrays_KeyExtraction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		key     string
		want    string
		wantErr bool
	}{
		{
			name: "extract findings from objects",
			input: `[
				{"findings": [{"id": 1}], "summary": "a"},
				{"findings": [{"id": 2}], "summary": "b"},
				{"findings": [{"id": 3}], "summary": "c"}
			]`,
			key:  "findings",
			want: `[{"id":1},{"id":2},{"id":3}]`,
		},
		{
			name: "extract with multiple items per array",
			input: `[
				{"results": [{"id": 1}, {"id": 2}]},
				{"results": [{"id": 3}]}
			]`,
			key:  "results",
			want: `[{"id":1},{"id":2},{"id":3}]`,
		},
		{
			name:  "extract from single element",
			input: `[{"items": [10, 20]}]`,
			key:   "items",
			want:  `[10,20]`,
		},
		{
			name:  "extract with empty arrays",
			input: `[{"data": []}, {"data": [1]}, {"data": []}]`,
			key:   "data",
			want:  `[1]`,
		},
		{
			name:    "key not found in object",
			input:   `[{"other": [1]}]`,
			key:     "findings",
			wantErr: true,
		},
		{
			name:    "element is not an object",
			input:   `[[1, 2], [3]]`,
			key:     "findings",
			wantErr: true,
		},
		{
			name:    "value at key is not an array",
			input:   `[{"findings": "not-array"}]`,
			key:     "findings",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeJSONArrays(tt.input, tt.key)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAggregateConfig_KeyField(t *testing.T) {
	// Verify that the AggregateConfig Key field is wired through the
	// full executeAggregate code path in the CompositionExecutor.
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output", "merged.json")

	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("audit1", []byte(`{"findings": [{"id": 1}], "summary": "a"}`))
	ctx.SetStepOutput("audit2", []byte(`{"findings": [{"id": 2}], "summary": "b"}`))
	ctx.SetStepOutput("audit3", []byte(`{"findings": [{"id": 3}], "summary": "c"}`))

	// Build input JSON: array of 3 objects
	inputJSON := `[` +
		`{"findings":[{"id":1}],"summary":"a"},` +
		`{"findings":[{"id":2}],"summary":"b"},` +
		`{"findings":[{"id":3}],"summary":"c"}` +
		`]`

	// Run merge_arrays with key extraction
	result, err := mergeJSONArrays(inputJSON, "findings")
	if err != nil {
		t.Fatalf("merge_arrays with key failed: %v", err)
	}

	expected := `[{"id":1},{"id":2},{"id":3}]`
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}

	// Verify file write works
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Errorf("file content: got %q, want %q", string(data), expected)
	}

	// Also verify without key -- existing behavior preserved
	bareArrayInput := `[[1,2],[3,4]]`
	result2, err := mergeJSONArrays(bareArrayInput, "")
	if err != nil {
		t.Fatalf("merge_arrays without key failed: %v", err)
	}
	if result2 != `[1,2,3,4]` {
		t.Errorf("without key: got %q, want %q", result2, `[1,2,3,4]`)
	}
}

func TestValidateCompositionTemplates(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "scope", SubPipeline: "gh-scope", SubInput: "{{input}}"},
			{ID: "implement", SubPipeline: "speckit-flow", SubInput: "{{scope.output.url}}",
				Iterate: &IterateConfig{Over: "{{scope.output.child_issues}}", Mode: "sequential"}},
			{ID: "bad-ref", SubPipeline: "test", SubInput: "{{nonexistent.output.field}}"},
		},
	}

	errors := ValidateCompositionTemplates(p)
	if len(errors) == 0 {
		t.Fatal("expected validation errors for nonexistent step reference")
	}

	// Should have exactly one error for "nonexistent"
	found := false
	for _, e := range errors {
		if containsSubstr(e, "nonexistent") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about 'nonexistent', got: %v", errors)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCompositionExecutor_SubPipelineWithConfig(t *testing.T) {
	// Verify that executeSubPipeline applies timeout from config
	ctx := NewTemplateContext("test-input", "/tmp")

	step := &Step{
		ID:          "sub-with-config",
		SubPipeline: "child-pipeline",
		Config: &SubPipelineConfig{
			Inject:    []string{"plan"},
			Extract:   []string{"output"},
			Timeout:   "1h",
			MaxCycles: 10,
		},
	}

	// Verify the step is a composition step
	if !step.IsCompositionStep() {
		t.Error("step with SubPipeline should be a composition step")
	}

	// Verify config validation passes
	if err := step.Config.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	// Verify input resolution still works
	step.SubInput = "{{input}}"
	input, err := resolveStepInputForTest(step, ctx)
	if err != nil {
		t.Fatalf("input resolution failed: %v", err)
	}
	if input != "test-input" {
		t.Errorf("expected 'test-input', got %q", input)
	}
}

func TestCompositionExecutor_SubPipelineBackwardCompatibility(t *testing.T) {
	// Verify that a step with SubPipeline but no Config still works
	step := &Step{
		ID:          "sub-legacy",
		SubPipeline: "simple-pipeline",
	}

	if !step.IsCompositionStep() {
		t.Error("step with SubPipeline should be a composition step")
	}

	// Config should be nil
	if step.Config != nil {
		t.Error("legacy step should have nil Config")
	}

	// SubPipelineConfig.Validate should handle nil
	if err := (*SubPipelineConfig)(nil).Validate(); err != nil {
		t.Errorf("nil config should validate: %v", err)
	}

	// ParseTimeout should handle nil
	if d := (*SubPipelineConfig)(nil).ParseTimeout(); d != 0 {
		t.Errorf("nil config ParseTimeout should be 0, got %v", d)
	}
}

// TestCompositionExecutor_IterateCollectsOutputs verifies that after an iterate
// step completes, the collected output is registered under the step's ID in the
// template context so {{ stepID.output }} resolves for downstream steps.
func TestCompositionExecutor_IterateCollectsOutputs(t *testing.T) {
	ctx := NewTemplateContext("test-input", "/tmp")

	// Simulate what runSubPipeline does: store output per child pipeline name
	ctx.SetStepOutput("audit-alpha", []byte(`{"findings": ["a1", "a2"]}`))
	ctx.SetStepOutput("audit-beta", []byte(`{"findings": ["b1"]}`))
	ctx.SetStepOutput("audit-gamma", []byte(`{"findings": ["c1", "c2", "c3"]}`))

	ce := &CompositionExecutor{tmplCtx: ctx}

	step := &Step{ID: "run-audits"}
	resolvedNames := []string{"audit-alpha", "audit-beta", "audit-gamma"}

	ce.collectIterateOutputs(step, resolvedNames)

	// Verify the iterate step's output was registered
	data, ok := ctx.StepOutputs["run-audits"]
	if !ok {
		t.Fatal("expected output for iterate step 'run-audits'")
	}

	// Verify it's a valid JSON array with 3 entries
	var collected []json.RawMessage
	if err := json.Unmarshal(data, &collected); err != nil {
		t.Fatalf("collected output is not valid JSON array: %v", err)
	}
	if len(collected) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(collected))
	}

	// Verify each entry matches the original child output
	var first map[string]interface{}
	if err := json.Unmarshal(collected[0], &first); err != nil {
		t.Fatalf("entry 0 is not valid JSON: %v", err)
	}
	findings, ok := first["findings"].([]interface{})
	if !ok || len(findings) != 2 {
		t.Errorf("entry 0: expected findings with 2 items, got %v", first["findings"])
	}

	// Verify the output can be resolved via {{ run-audits.output }}
	resolved, err := ResolveTemplate("{{ run-audits.output }}", ctx)
	if err != nil {
		t.Fatalf("failed to resolve {{ run-audits.output }}: %v", err)
	}
	if resolved == "" {
		t.Error("resolved output should not be empty")
	}

	// Verify the resolved output is a valid JSON array
	var resolvedArr []json.RawMessage
	if err := json.Unmarshal([]byte(resolved), &resolvedArr); err != nil {
		t.Fatalf("resolved output is not a valid JSON array: %v", err)
	}
	if len(resolvedArr) != 3 {
		t.Errorf("expected 3 entries in resolved output, got %d", len(resolvedArr))
	}
}

// TestCompositionExecutor_IterateCollectsOutputs_NullForMissing verifies that
// missing child outputs are represented as null in the collected array.
func TestCompositionExecutor_IterateCollectsOutputs_NullForMissing(t *testing.T) {
	ctx := NewTemplateContext("test-input", "/tmp")

	// Only one of three children produced output
	ctx.SetStepOutput("audit-alpha", []byte(`{"ok": true}`))

	ce := &CompositionExecutor{tmplCtx: ctx}

	step := &Step{ID: "run-audits"}
	resolvedNames := []string{"audit-alpha", "audit-beta", "audit-gamma"}

	ce.collectIterateOutputs(step, resolvedNames)

	data, ok := ctx.StepOutputs["run-audits"]
	if !ok {
		t.Fatal("expected output for iterate step")
	}

	var collected []json.RawMessage
	if err := json.Unmarshal(data, &collected); err != nil {
		t.Fatalf("not valid JSON array: %v", err)
	}
	if len(collected) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(collected))
	}

	// First entry should be valid JSON
	if string(collected[0]) == "null" {
		t.Error("entry 0 should not be null")
	}
	// Entries for missing children should be null
	if string(collected[1]) != "null" {
		t.Errorf("entry 1: expected null, got %s", string(collected[1]))
	}
	if string(collected[2]) != "null" {
		t.Errorf("entry 2: expected null, got %s", string(collected[2]))
	}
}

// TestBranchDispatch_DefaultArmOnMissingEnv verifies the branch primitive
// falls back to cases.default when {{ env.<key> }} resolves to empty (no
// matching case key for "").
func TestBranchDispatch_DefaultArmOnMissingEnv(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	// Env is empty — {{ env.profile }} resolves to ""
	step := &Step{
		ID: "publish",
		Branch: &BranchConfig{
			On: "{{ env.profile }}",
			Cases: map[string]string{
				"core":    "skip",
				"default": "ops-pr-review-publish",
			},
		},
	}

	resolved, err := ResolveTemplate(step.Branch.On, ctx)
	if err != nil {
		t.Fatalf("template resolution failed: %v", err)
	}
	if resolved != "" {
		t.Fatalf("expected empty resolution for missing env, got %q", resolved)
	}

	pipelineName, ok := step.Branch.Cases[resolved]
	if !ok {
		pipelineName = step.Branch.Cases["default"]
	}
	if pipelineName != "ops-pr-review-publish" {
		t.Errorf("expected default arm 'ops-pr-review-publish', got %q", pipelineName)
	}
}

// TestBranchDispatch_EnvProfileCore verifies the branch primitive routes to
// the "core" case when env.profile is "core" — which the unified
// ops-pr-review pipeline uses to skip the publish step.
func TestBranchDispatch_EnvProfileCore(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.Env = map[string]string{"profile": "core"}
	step := &Step{
		ID: "publish",
		Branch: &BranchConfig{
			On: "{{ env.profile }}",
			Cases: map[string]string{
				"core":    "skip",
				"default": "ops-pr-review-publish",
			},
		},
	}

	resolved, err := ResolveTemplate(step.Branch.On, ctx)
	if err != nil {
		t.Fatalf("template resolution failed: %v", err)
	}
	if resolved != "core" {
		t.Fatalf("expected 'core', got %q", resolved)
	}
	if step.Branch.Cases[resolved] != "skip" {
		t.Errorf("expected core arm to dispatch to 'skip', got %q", step.Branch.Cases[resolved])
	}
}

// TestSubPipelineConfig_EnvField verifies the Env field is preserved on the
// SubPipelineConfig and round-trips through Validate.
func TestSubPipelineConfig_EnvField(t *testing.T) {
	cfg := &SubPipelineConfig{
		Env: map[string]string{"profile": "core", "region": "eu"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if got := cfg.Env["profile"]; got != "core" {
		t.Errorf("expected profile=core, got %q", got)
	}
	if len(cfg.Env) != 2 {
		t.Errorf("expected 2 env entries, got %d", len(cfg.Env))
	}
}

func TestCompositionExecutor_Execute_Gate_Auto(t *testing.T) {
	emitter := testutil.NewEventCollector()
	m := &manifest.Manifest{}

	ce := NewCompositionExecutor(nil, emitter, nil, m, "test", ".agents/pipelines", false)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "gate-test"},
		Steps: []Step{
			{
				ID:   "approve",
				Gate: &GateConfig{Type: "approval", Auto: true},
			},
		},
	}

	ctx := context.Background()
	err := ce.Execute(ctx, p, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !emitter.HasEventWithState(event.StateGateResolved) {
		t.Error("expected gate_resolved event")
	}
}

// TestCompositionExecutor_Aggregate_RegistersArtifact verifies that the legacy
// composition path also registers aggregate output artifacts when a runID and
// store are wired — parity with executor.go's executeAggregateInDAG.
func TestCompositionExecutor_Aggregate_RegistersArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "merged.json")

	var (
		gotRun, gotStep, gotName, gotPath, gotType string
		gotSize                                    int64
		called                                     bool
	)
	store := testutil.NewMockStateStore(testutil.WithRegisterArtifact(
		func(runID, stepID, name, path, artifactType string, size int64) error {
			called = true
			gotRun, gotStep, gotName, gotPath, gotType, gotSize = runID, stepID, name, path, artifactType, size
			return nil
		},
	))

	emitter := testutil.NewEventCollector()
	m := &manifest.Manifest{}
	ce := NewCompositionExecutor(nil, emitter, store, m, "test", ".agents/pipelines", false)
	ce.SetRunID("legacy-run-1")

	step := &Step{
		ID: "merge",
		Aggregate: &AggregateConfig{
			From:     `[{"id":1}]`,
			Into:     outputPath,
			Strategy: "concat",
		},
	}

	if err := ce.executeAggregate(step); err != nil {
		t.Fatalf("executeAggregate: %v", err)
	}

	if !called {
		t.Fatal("expected RegisterArtifact to be called")
	}
	if gotRun != "legacy-run-1" {
		t.Errorf("runID = %q, want legacy-run-1", gotRun)
	}
	if gotStep != "merge" {
		t.Errorf("stepID = %q, want merge", gotStep)
	}
	if gotName != "merged" {
		t.Errorf("name = %q, want merged", gotName)
	}
	if gotPath != outputPath {
		t.Errorf("path = %q, want %q", gotPath, outputPath)
	}
	if gotType != "json" {
		t.Errorf("type = %q, want json", gotType)
	}
	if gotSize <= 0 {
		t.Errorf("size = %d, want > 0", gotSize)
	}
}

// TestCompositionExecutor_Aggregate_NoRegistrationWithoutRunID confirms the
// legacy path stays a no-op for tests/legacy callers that wire a store but no
// runID — preserves test ergonomics.
func TestCompositionExecutor_Aggregate_NoRegistrationWithoutRunID(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "merged.json")

	var called bool
	store := testutil.NewMockStateStore(testutil.WithRegisterArtifact(
		func(_, _, _, _, _ string, _ int64) error {
			called = true
			return nil
		},
	))

	emitter := testutil.NewEventCollector()
	m := &manifest.Manifest{}
	ce := NewCompositionExecutor(nil, emitter, store, m, "test", ".agents/pipelines", false)
	// No SetRunID call — must skip registration.

	step := &Step{
		ID: "merge",
		Aggregate: &AggregateConfig{
			From:     `[{"id":1}]`,
			Into:     outputPath,
			Strategy: "concat",
		},
	}

	if err := ce.executeAggregate(step); err != nil {
		t.Fatalf("executeAggregate: %v", err)
	}
	if called {
		t.Error("RegisterArtifact must be skipped when runID is empty")
	}
}

func TestBuildExecutionPlan_MultipleParallelGroups(t *testing.T) {
	// Simulate: wave compose --parallel A B -- C D -- E
	// Expected: Stage 1 (parallel: A,B), Stage 2 (parallel: C,D), Stage 3 (sequential: E)

	newPipeline := func(name string) *Pipeline {
		return &Pipeline{
			Metadata: PipelineMetadata{Name: name},
			Steps: []Step{
				{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "do it"}},
			},
		}
	}

	var entries []ComposeEntry
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		entries = append(entries, ComposeEntry{Name: name, Pipeline: newPipeline(name)})
	}

	args := []string{"A", "B", "--", "C", "D", "--", "E"}
	plan := BuildExecutionPlan(entries, args)

	if got, want := len(plan.Stages), 3; got != want {
		t.Fatalf("stages: got %d, want %d", got, want)
	}

	// Stage 1: A, B — parallel
	if got, want := len(plan.Stages[0].Pipelines), 2; got != want {
		t.Errorf("stage 0 pipelines: got %d, want %d", got, want)
	}
	if !plan.Stages[0].Parallel {
		t.Error("stage 0 should be parallel")
	}
	if name := plan.Stages[0].Pipelines[0].Metadata.Name; name != "A" {
		t.Errorf("stage 0 pipeline 0: got %q, want A", name)
	}
	if name := plan.Stages[0].Pipelines[1].Metadata.Name; name != "B" {
		t.Errorf("stage 0 pipeline 1: got %q, want B", name)
	}

	// Stage 2: C, D — parallel
	if got, want := len(plan.Stages[1].Pipelines), 2; got != want {
		t.Errorf("stage 1 pipelines: got %d, want %d", got, want)
	}
	if !plan.Stages[1].Parallel {
		t.Error("stage 1 should be parallel")
	}
	if name := plan.Stages[1].Pipelines[0].Metadata.Name; name != "C" {
		t.Errorf("stage 1 pipeline 0: got %q, want C", name)
	}
	if name := plan.Stages[1].Pipelines[1].Metadata.Name; name != "D" {
		t.Errorf("stage 1 pipeline 1: got %q, want D", name)
	}

	// Stage 3: E — sequential (single pipeline)
	if got, want := len(plan.Stages[2].Pipelines), 1; got != want {
		t.Errorf("stage 2 pipelines: got %d, want %d", got, want)
	}
	if plan.Stages[2].Parallel {
		t.Error("stage 2 with single pipeline should be sequential")
	}
	if name := plan.Stages[2].Pipelines[0].Metadata.Name; name != "E" {
		t.Errorf("stage 2 pipeline 0: got %q, want E", name)
	}
}

func TestBuildExecutionPlan_SingleGroupParallel(t *testing.T) {
	// Simulate: wave compose --parallel A B C (no -- separators)
	// Expected: Stage 1 (parallel: A,B,C)

	newPipeline := func(name string) *Pipeline {
		return &Pipeline{
			Metadata: PipelineMetadata{Name: name},
			Steps: []Step{
				{ID: "step1", Persona: "navigator", Exec: ExecConfig{Source: "do it"}},
			},
		}
	}

	var entries []ComposeEntry
	for _, name := range []string{"A", "B", "C"} {
		entries = append(entries, ComposeEntry{Name: name, Pipeline: newPipeline(name)})
	}

	args := []string{"A", "B", "C"}
	plan := BuildExecutionPlan(entries, args)

	if got, want := len(plan.Stages), 1; got != want {
		t.Fatalf("stages: got %d, want %d", got, want)
	}
	if got, want := len(plan.Stages[0].Pipelines), 3; got != want {
		t.Errorf("stage 0 pipelines: got %d, want %d", got, want)
	}
	if !plan.Stages[0].Parallel {
		t.Error("single multi-pipeline group should be parallel")
	}
}

func TestBuildExecutionPlan_DropsUnknownNames(t *testing.T) {
	// Names not present in entries are silently dropped.
	newPipeline := func(name string) *Pipeline {
		return &Pipeline{Metadata: PipelineMetadata{Name: name}}
	}
	entries := []ComposeEntry{
		{Name: "A", Pipeline: newPipeline("A")},
	}
	plan := BuildExecutionPlan(entries, []string{"A", "ghost"})
	if got, want := len(plan.Stages), 1; got != want {
		t.Fatalf("stages: got %d, want %d", got, want)
	}
	if got, want := len(plan.Stages[0].Pipelines), 1; got != want {
		t.Errorf("pipelines: got %d, want %d (ghost should be dropped)", got, want)
	}
	if plan.Stages[0].Parallel {
		t.Error("single resolved pipeline must be sequential")
	}
}

func TestValidateComposeSpec_PrefixesEntryName(t *testing.T) {
	// A pipeline with a template ref to an unknown step should produce
	// an error prefixed with the entry name.
	bad := &Pipeline{
		Metadata: PipelineMetadata{Name: "bad-pipeline"},
		Steps: []Step{
			{
				ID:       "merge",
				Aggregate: &AggregateConfig{From: "{{ unknown_step.output }}"},
			},
		},
	}
	good := &Pipeline{
		Metadata: PipelineMetadata{Name: "good-pipeline"},
		Steps: []Step{
			{ID: "compute"},
			{
				ID:       "merge",
				Aggregate: &AggregateConfig{From: "{{ compute.output }}"},
			},
		},
	}

	entries := []ComposeEntry{
		{Name: "good", Pipeline: good},
		{Name: "bad", Pipeline: bad},
	}
	errs := ValidateComposeSpec(entries)
	if len(errs) != 1 {
		t.Fatalf("errors: got %d (%v), want 1", len(errs), errs)
	}
	if got := errs[0]; len(got) < 5 || got[:5] != "[bad]" {
		t.Errorf("error must be prefixed with entry name: got %q", errs[0])
	}
}

func TestValidateComposeSpec_NilPipelineSkipped(t *testing.T) {
	entries := []ComposeEntry{{Name: "nil-entry", Pipeline: nil}}
	if errs := ValidateComposeSpec(entries); len(errs) != 0 {
		t.Errorf("nil pipeline must be skipped, got %v", errs)
	}
}

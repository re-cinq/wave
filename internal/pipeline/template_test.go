package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTemplate_Input(t *testing.T) {
	ctx := NewTemplateContext("hello world", "/tmp")
	result, err := ResolveTemplate("{{input}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", result)
	}
}

func TestResolveTemplate_InputEmbedded(t *testing.T) {
	ctx := NewTemplateContext("issue-42", "/tmp")
	result, err := ResolveTemplate("Process: {{input}} with care", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Process: issue-42 with care" {
		t.Errorf("expected %q, got %q", "Process: issue-42 with care", result)
	}
}

func TestResolveTemplate_StepOutput(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("scope", []byte(`{"child_issues": [{"url": "https://github.com/org/repo/issues/1"}]}`))

	result, err := ResolveTemplate("{{scope.output}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"child_issues": [{"url": "https://github.com/org/repo/issues/1"}]}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolveTemplate_StepOutputField(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("analyze", []byte(`{"severity": "high", "score": 85}`))

	result, err := ResolveTemplate("{{analyze.output.severity}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "high" {
		t.Errorf("expected %q, got %q", "high", result)
	}
}

func TestResolveTemplate_StepOutputNestedField(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.SetStepOutput("research", []byte(`{"meta": {"status": "complete"}}`))

	result, err := ResolveTemplate("{{research.output.meta.status}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "complete" {
		t.Errorf("expected %q, got %q", "complete", result)
	}
}

func TestResolveTemplate_Item(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.Item = json.RawMessage(`{"url": "https://github.com/org/repo/issues/1", "title": "Fix bug"}`)

	result, err := ResolveTemplate("{{item.url}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "https://github.com/org/repo/issues/1" {
		t.Errorf("expected URL, got %q", result)
	}
}

func TestResolveTemplate_ItemFull(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	itemJSON := `{"url":"https://example.com","id":42}`
	ctx.Item = json.RawMessage(itemJSON)

	result, err := ResolveTemplate("{{item}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != itemJSON {
		t.Errorf("expected %q, got %q", itemJSON, result)
	}
}

func TestResolveTemplate_ItemStringUnquoted(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	// JSON string items should be unquoted: "audit-security" → audit-security
	ctx.Item = json.RawMessage(`"audit-security"`)

	result, err := ResolveTemplate("{{item}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "audit-security" {
		t.Errorf("expected %q, got %q", "audit-security", result)
	}
}

func TestResolveTemplate_ItemStringInPipelineName(t *testing.T) {
	// Simulates iterate resolving "{{ item }}" to a pipeline name
	ctx := NewTemplateContext("test-input", "/tmp")
	ctx.Item = json.RawMessage(`"audit-dead-code"`)

	result, err := ResolveTemplate("{{ item }}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "audit-dead-code" {
		t.Errorf("expected %q, got %q", "audit-dead-code", result)
	}
}

func TestResolveTemplate_Iteration(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	ctx.Iteration = 3

	result, err := ResolveTemplate("iteration-{{iteration}}", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "iteration-3" {
		t.Errorf("expected %q, got %q", "iteration-3", result)
	}
}

func TestResolveTemplate_MultipleExpressions(t *testing.T) {
	ctx := NewTemplateContext("feature-x", "/tmp")
	ctx.SetStepOutput("scope", []byte(`{"count": 5}`))

	result, err := ResolveTemplate("{{input}} has {{scope.output.count}} items", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "feature-x has 5 items" {
		t.Errorf("expected %q, got %q", "feature-x has 5 items", result)
	}
}

func TestResolveTemplate_NoExpressions(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	result, err := ResolveTemplate("plain text", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "plain text" {
		t.Errorf("expected %q, got %q", "plain text", result)
	}
}

func TestResolveTemplate_MissingStep(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	_, err := ResolveTemplate("{{nonexistent.output}}", ctx)
	if err == nil {
		t.Fatal("expected error for missing step output")
	}
}

func TestResolveTemplate_MissingItem(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	_, err := ResolveTemplate("{{item.field}}", ctx)
	if err == nil {
		t.Fatal("expected error when no item in context")
	}
}

func TestResolveTemplate_InvalidExpression(t *testing.T) {
	ctx := NewTemplateContext("", "/tmp")
	_, err := ResolveTemplate("{{invalid}}", ctx)
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
}

func TestIsCompositionStep(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		expected bool
	}{
		{"regular step", Step{ID: "test"}, false},
		{"sub-pipeline", Step{ID: "test", SubPipeline: "child"}, true},
		{"iterate", Step{ID: "test", Iterate: &IterateConfig{Over: "items"}}, true},
		{"branch", Step{ID: "test", Branch: &BranchConfig{On: "val"}}, true},
		{"gate", Step{ID: "test", Gate: &GateConfig{Type: "approval"}}, true},
		{"loop", Step{ID: "test", Loop: &LoopConfig{MaxIterations: 3}}, true},
		{"aggregate", Step{ID: "test", Aggregate: &AggregateConfig{Strategy: "concat"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.IsCompositionStep(); got != tt.expected {
				t.Errorf("IsCompositionStep() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadStepArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineID := "test-pipeline"
	stepID := "scope"
	artifactName := "result.json"

	// Create artifact in expected location
	artifactDir := filepath.Join(tmpDir, pipelineID, stepID, ".wave", "output")
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"status": "ok"}`)
	if err := os.WriteFile(filepath.Join(artifactDir, artifactName), content, 0644); err != nil {
		t.Fatal(err)
	}

	data, err := LoadStepArtifact(tmpDir, pipelineID, stepID, artifactName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(data))
	}
}

func TestLoadStepArtifact_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadStepArtifact(tmpDir, "pipeline", "step", "missing.json")
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}

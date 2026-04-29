package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/fileutil"
)

func TestSubPipelineConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *SubPipelineConfig
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid config with all fields set",
			cfg: &SubPipelineConfig{
				Inject:        []string{"plan"},
				Extract:       []string{"output"},
				Timeout:       "30s",
				MaxCycles:     5,
				StopCondition: "context.tests_pass=true",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: false,
		},
		{
			name: "invalid timeout format",
			cfg: &SubPipelineConfig{
				Timeout: "bad",
			},
			wantErr:   true,
			errSubstr: "invalid timeout",
		},
		{
			name: "negative max_cycles",
			cfg: &SubPipelineConfig{
				MaxCycles: -1,
			},
			wantErr:   true,
			errSubstr: "max_cycles must be >= 0",
		},
		{
			name: "config with only inject and extract",
			cfg: &SubPipelineConfig{
				Inject:  []string{"plan"},
				Extract: []string{"output"},
			},
			wantErr: false,
		},
		{
			name: "config with valid timeout 30s",
			cfg: &SubPipelineConfig{
				Timeout: "30s",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestSubPipelineConfig_ParseTimeout(t *testing.T) {
	tests := []struct {
		name string
		cfg  *SubPipelineConfig
		want time.Duration
	}{
		{
			name: "timeout 30s",
			cfg:  &SubPipelineConfig{Timeout: "30s"},
			want: 30 * time.Second,
		},
		{
			name: "empty timeout",
			cfg:  &SubPipelineConfig{Timeout: ""},
			want: 0,
		},
		{
			name: "nil config",
			cfg:  nil,
			want: 0,
		},
		{
			name: "invalid timeout returns zero",
			cfg:  &SubPipelineConfig{Timeout: "notaduration"},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ParseTimeout()
			if got != tt.want {
				t.Errorf("ParseTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectSubPipelineArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake parent artifact file
	parentArtifactsDir := filepath.Join(tmpDir, "parent", ".agents", "artifacts")
	if err := os.MkdirAll(parentArtifactsDir, 0755); err != nil {
		t.Fatalf("failed to create parent artifacts dir: %v", err)
	}
	planContent := []byte("this is the plan artifact")
	planPath := filepath.Join(parentArtifactsDir, "plan")
	if err := os.WriteFile(planPath, planContent, 0644); err != nil {
		t.Fatalf("failed to write plan artifact: %v", err)
	}

	// Create parent pipeline context and register the artifact
	parentCtx := &PipelineContext{
		ArtifactPaths:   map[string]string{"plan": planPath},
		CustomVariables: make(map[string]string),
	}

	// Create child workspace dir
	childWorkspaceDir := filepath.Join(tmpDir, "child")
	if err := os.MkdirAll(childWorkspaceDir, 0755); err != nil {
		t.Fatalf("failed to create child workspace dir: %v", err)
	}

	cfg := &SubPipelineConfig{
		Inject: []string{"plan"},
	}

	err := injectSubPipelineArtifacts(cfg, parentCtx, childWorkspaceDir)
	if err != nil {
		t.Fatalf("injectSubPipelineArtifacts() error: %v", err)
	}

	// Verify the file was copied to child workspace
	destPath := filepath.Join(childWorkspaceDir, ".agents", "artifacts", "plan")
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read injected artifact: %v", err)
	}
	if string(got) != string(planContent) {
		t.Errorf("injected artifact content = %q, want %q", string(got), string(planContent))
	}
}

func TestInjectSubPipelineArtifacts_MissingArtifact(t *testing.T) {
	parentCtx := &PipelineContext{
		ArtifactPaths:   make(map[string]string),
		CustomVariables: make(map[string]string),
	}

	cfg := &SubPipelineConfig{
		Inject: []string{"nonexistent"},
	}

	childWorkspaceDir := t.TempDir()
	err := injectSubPipelineArtifacts(cfg, parentCtx, childWorkspaceDir)
	if err == nil {
		t.Fatal("expected error for missing artifact, got nil")
	}
	if !strings.Contains(err.Error(), "not found in parent context") {
		t.Errorf("error %q should contain %q", err.Error(), "not found in parent context")
	}
}

func TestExtractSubPipelineArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create child artifact
	childArtifactsDir := filepath.Join(tmpDir, "child", ".agents", "artifacts")
	if err := os.MkdirAll(childArtifactsDir, 0755); err != nil {
		t.Fatalf("failed to create child artifacts dir: %v", err)
	}
	outputContent := []byte("child output data")
	outputPath := filepath.Join(childArtifactsDir, "output")
	if err := os.WriteFile(outputPath, outputContent, 0644); err != nil {
		t.Fatalf("failed to write output artifact: %v", err)
	}

	// Create child pipeline context
	childCtx := &PipelineContext{
		ArtifactPaths:   map[string]string{"output": outputPath},
		CustomVariables: make(map[string]string),
	}

	// Create parent workspace dir and context
	parentWorkspaceDir := filepath.Join(tmpDir, "parent")
	if err := os.MkdirAll(parentWorkspaceDir, 0755); err != nil {
		t.Fatalf("failed to create parent workspace dir: %v", err)
	}
	parentCtx := &PipelineContext{
		ArtifactPaths:   make(map[string]string),
		CustomVariables: make(map[string]string),
	}

	cfg := &SubPipelineConfig{
		Extract: []string{"output"},
	}

	err := extractSubPipelineArtifacts(cfg, childCtx, "child-pipeline", parentCtx, parentWorkspaceDir)
	if err != nil {
		t.Fatalf("extractSubPipelineArtifacts() error: %v", err)
	}

	// Verify file copied with namespaced name
	namespacedName := "child-pipeline.output"
	destPath := filepath.Join(parentWorkspaceDir, ".agents", "artifacts", namespacedName)
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted artifact: %v", err)
	}
	if string(got) != string(outputContent) {
		t.Errorf("extracted artifact content = %q, want %q", string(got), string(outputContent))
	}

	// Verify parent context has the artifact registered
	registeredPath := parentCtx.GetArtifactPath(namespacedName)
	if registeredPath != destPath {
		t.Errorf("parent context artifact path = %q, want %q", registeredPath, destPath)
	}
}

func TestEvaluateStopCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		ctx       *PipelineContext
		want      bool
	}{
		{
			name:      "empty condition",
			condition: "",
			ctx: &PipelineContext{
				CustomVariables: map[string]string{"tests_pass": "true"},
			},
			want: false,
		},
		{
			name:      "context variable matches true",
			condition: "context.tests_pass=true",
			ctx: &PipelineContext{
				CustomVariables: map[string]string{"tests_pass": "true"},
			},
			want: true,
		},
		{
			name:      "context variable does not match",
			condition: "context.tests_pass=true",
			ctx: &PipelineContext{
				CustomVariables: map[string]string{"tests_pass": "false"},
			},
			want: false,
		},
		{
			name:      "simple true after resolution",
			condition: "true",
			ctx: &PipelineContext{
				CustomVariables: make(map[string]string),
			},
			want: true,
		},
		{
			name:      "nil context",
			condition: "context.tests_pass=true",
			ctx:       nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateStopCondition(tt.condition, tt.ctx)
			if got != tt.want {
				t.Errorf("evaluateStopCondition(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}

func TestSubPipelineTimeout(t *testing.T) {
	t.Run("nil config does not cancel context", func(t *testing.T) {
		ctx := context.Background()
		wrappedCtx, cancel := subPipelineTimeout(ctx, nil)
		defer cancel()

		select {
		case <-wrappedCtx.Done():
			t.Error("context should not be cancelled with nil config")
		default:
			// expected: context is still active
		}
	})

	t.Run("empty timeout does not cancel context", func(t *testing.T) {
		ctx := context.Background()
		cfg := &SubPipelineConfig{Timeout: ""}
		wrappedCtx, cancel := subPipelineTimeout(ctx, cfg)
		defer cancel()

		select {
		case <-wrappedCtx.Done():
			t.Error("context should not be cancelled with empty timeout")
		default:
			// expected: context is still active
		}
	})

	t.Run("100ms timeout cancels context", func(t *testing.T) {
		ctx := context.Background()
		cfg := &SubPipelineConfig{Timeout: "100ms"}
		wrappedCtx, cancel := subPipelineTimeout(ctx, cfg)
		defer cancel()

		select {
		case <-wrappedCtx.Done():
			// expected: context was cancelled by timeout
		case <-time.After(500 * time.Millisecond):
			t.Error("context should have been cancelled after 100ms timeout")
		}
	})
}

func Test_detectSubPipelineCycles(t *testing.T) {
	tmpDir := t.TempDir()

	// pipeline-a references pipeline-b
	pipelineA := `kind: WavePipeline
metadata:
  name: pipeline-a
steps:
  - id: run-b
    pipeline: pipeline-b
`
	// pipeline-b references pipeline-a (cycle!)
	pipelineB := `kind: WavePipeline
metadata:
  name: pipeline-b
steps:
  - id: run-a
    pipeline: pipeline-a
`

	if err := os.WriteFile(filepath.Join(tmpDir, "pipeline-a.yaml"), []byte(pipelineA), 0644); err != nil {
		t.Fatalf("failed to write pipeline-a.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "pipeline-b.yaml"), []byte(pipelineB), 0644); err != nil {
		t.Fatalf("failed to write pipeline-b.yaml: %v", err)
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "pipeline-a"},
		Steps: []Step{
			{ID: "run-b", SubPipeline: "pipeline-b"},
		},
	}

	err := detectSubPipelineCycles(p, tmpDir)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error %q should mention circular reference", err.Error())
	}
}

func Test_detectSubPipelineCycles_Transitive(t *testing.T) {
	tmpDir := t.TempDir()

	// A -> B -> C -> A (transitive cycle)
	pipelineA := `kind: WavePipeline
metadata:
  name: pipeline-a
steps:
  - id: run-b
    pipeline: pipeline-b
`
	pipelineB := `kind: WavePipeline
metadata:
  name: pipeline-b
steps:
  - id: run-c
    pipeline: pipeline-c
`
	pipelineC := `kind: WavePipeline
metadata:
  name: pipeline-c
steps:
  - id: run-a
    pipeline: pipeline-a
`

	for name, content := range map[string]string{
		"pipeline-a.yaml": pipelineA,
		"pipeline-b.yaml": pipelineB,
		"pipeline-c.yaml": pipelineC,
	} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "pipeline-a"},
		Steps: []Step{
			{ID: "run-b", SubPipeline: "pipeline-b"},
		},
	}

	err := detectSubPipelineCycles(p, tmpDir)
	if err == nil {
		t.Fatal("expected cycle detection error for transitive cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error %q should mention circular reference", err.Error())
	}
}

func TestInjectSubPipelineArtifacts_NilConfig(t *testing.T) {
	err := injectSubPipelineArtifacts(nil, nil, t.TempDir())
	if err != nil {
		t.Errorf("expected nil error for nil config, got: %v", err)
	}
}

func TestInjectSubPipelineArtifacts_EmptyInject(t *testing.T) {
	cfg := &SubPipelineConfig{Inject: []string{}}
	err := injectSubPipelineArtifacts(cfg, nil, t.TempDir())
	if err != nil {
		t.Errorf("expected nil error for empty inject list, got: %v", err)
	}
}

func TestExtractSubPipelineArtifacts_NilConfig(t *testing.T) {
	err := extractSubPipelineArtifacts(nil, nil, "child", nil, t.TempDir())
	if err != nil {
		t.Errorf("expected nil error for nil config, got: %v", err)
	}
}

func TestExtractSubPipelineArtifacts_MissingArtifact(t *testing.T) {
	childCtx := &PipelineContext{
		ArtifactPaths:   make(map[string]string),
		CustomVariables: make(map[string]string),
	}
	parentCtx := &PipelineContext{
		ArtifactPaths:   make(map[string]string),
		CustomVariables: make(map[string]string),
	}

	cfg := &SubPipelineConfig{Extract: []string{"nonexistent"}}
	err := extractSubPipelineArtifacts(cfg, childCtx, "child", parentCtx, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing extract artifact, got nil")
	}
	if !strings.Contains(err.Error(), "not found in child context") {
		t.Errorf("error %q should contain 'not found in child context'", err.Error())
	}
}

func TestInjectSubPipelineArtifacts_DirectoryCopy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory artifact (not a file)
	parentDir := filepath.Join(tmpDir, "parent-artifacts")
	specDir := filepath.Join(parentDir, "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# Spec"), 0644); err != nil {
		t.Fatal(err)
	}

	parentCtx := &PipelineContext{
		ArtifactPaths:   map[string]string{"specs": parentDir},
		CustomVariables: make(map[string]string),
	}

	childDir := filepath.Join(tmpDir, "child")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &SubPipelineConfig{Inject: []string{"specs"}}
	err := injectSubPipelineArtifacts(cfg, parentCtx, childDir)
	if err != nil {
		t.Fatalf("injectSubPipelineArtifacts() error: %v", err)
	}

	// Verify directory was copied recursively
	copiedSpec := filepath.Join(childDir, ".agents", "artifacts", "specs", "specs", "spec.md")
	if _, err := os.Stat(copiedSpec); os.IsNotExist(err) {
		t.Errorf("expected directory artifact to be copied recursively, but %s does not exist", copiedSpec)
	}
}

func TestAdapterOverride_PropagatedToChildExecutor(t *testing.T) {
	// Regression test for #768: runNamedSubPipeline must propagate adapterOverride
	// to child executors so that --adapter CLI flag applies to all sub-pipelines.
	parent := NewDefaultPipelineExecutor(nil, WithAdapterOverride("opencode"))
	if parent.adapterOverride != "opencode" {
		t.Fatalf("parent adapterOverride = %q, want %q", parent.adapterOverride, "opencode")
	}

	// Mirror the childOpts construction in runNamedSubPipeline.
	var childOpts []ExecutorOption
	if parent.adapterOverride != "" {
		childOpts = append(childOpts, WithAdapterOverride(parent.adapterOverride))
	}

	allOpts_child := append([]ExecutorOption{}, childOpts...)
	child := NewDefaultPipelineExecutor(nil, allOpts_child...)
	if child.adapterOverride != "opencode" {
		t.Errorf("child adapterOverride = %q, want %q (should be inherited from parent)", child.adapterOverride, "opencode")
	}
}

func TestParentEnv_PropagatedToChildExecutor(t *testing.T) {
	// Mirrors runNamedSubPipeline's env propagation: child inherits parent
	// env first, then overlays step.Config.Env.
	parent := NewDefaultPipelineExecutor(nil, WithParentEnv(map[string]string{
		"profile": "core",
		"region":  "eu",
	}),
	)
	step := &Step{
		ID:          "review",
		SubPipeline: "ops-pr-review",
		Config: &SubPipelineConfig{
			Env: map[string]string{"profile": "full"}, // overrides parent
		},
	}

	stepEnv := map[string]string{}
	if step.Config != nil {
		stepEnv = step.Config.Env
	}
	merged := make(map[string]string, len(parent.parentEnv)+len(stepEnv))
	for k, v := range parent.parentEnv {
		merged[k] = v
	}
	for k, v := range stepEnv {
		merged[k] = v
	}

	child := NewDefaultPipelineExecutor(nil, WithParentEnv(merged))
	if got := child.parentEnv["profile"]; got != "full" {
		t.Errorf("child env profile = %q, want %q (step overrides parent)", got, "full")
	}
	if got := child.parentEnv["region"]; got != "eu" {
		t.Errorf("child env region = %q, want %q (inherited from parent)", got, "eu")
	}
}

func TestParentEnv_SeededAsCustomVariables(t *testing.T) {
	// Verifies parent env vars seeded into PipelineContext.CustomVariables
	// resolve via {{ env.<key> }} through ResolvePlaceholders.
	ctx := &PipelineContext{
		CustomVariables: make(map[string]string),
	}
	parentEnv := map[string]string{"profile": "core", "region": "us-west"}
	for k, v := range parentEnv {
		ctx.SetCustomVariable("env."+k, v)
	}

	got := ctx.ResolvePlaceholders("{{ env.profile }}-{{ env.region }}")
	want := "core-us-west"
	if got != want {
		t.Errorf("ResolvePlaceholders() = %q, want %q", got, want)
	}
}

func TestAdapterOverride_NotPropagatedWhenEmpty(t *testing.T) {
	// When the parent has no adapterOverride, the child should not receive one.
	parent := NewDefaultPipelineExecutor(nil)

	var childOpts []ExecutorOption
	if parent.adapterOverride != "" {
		childOpts = append(childOpts, WithAdapterOverride(parent.adapterOverride))
	}

	allOpts_child := append([]ExecutorOption{}, childOpts...)
	child := NewDefaultPipelineExecutor(nil, allOpts_child...)
	if child.adapterOverride != "" {
		t.Errorf("child adapterOverride = %q, want empty (no override on parent)", child.adapterOverride)
	}
}

func TestCopyPath(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := fileutil.CopyPath(src, dest); err != nil {
		t.Fatalf("CopyPath() error: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read dest: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("CopyPath content = %q, want %q", string(got), "hello")
	}
}

func TestEvaluateStopCondition_DoneValue(t *testing.T) {
	ctx := &PipelineContext{
		CustomVariables: make(map[string]string),
	}
	if !evaluateStopCondition("done", ctx) {
		t.Error("expected 'done' to evaluate as true")
	}
	if !evaluateStopCondition("yes", ctx) {
		t.Error("expected 'yes' to evaluate as true")
	}
	if evaluateStopCondition("no", ctx) {
		t.Error("expected 'no' to evaluate as false")
	}
}

func Test_detectSubPipelineCycles_NoCycle(t *testing.T) {
	tmpDir := t.TempDir()

	// pipeline-a references pipeline-b
	pipelineA := `kind: WavePipeline
metadata:
  name: pipeline-a
steps:
  - id: run-b
    pipeline: pipeline-b
`
	// pipeline-b has no sub-pipeline references
	pipelineB := `kind: WavePipeline
metadata:
  name: pipeline-b
steps:
  - id: do-work
    persona: navigator
    exec:
      type: prompt
      source: "do something"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "pipeline-a.yaml"), []byte(pipelineA), 0644); err != nil {
		t.Fatalf("failed to write pipeline-a.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "pipeline-b.yaml"), []byte(pipelineB), 0644); err != nil {
		t.Fatalf("failed to write pipeline-b.yaml: %v", err)
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "pipeline-a"},
		Steps: []Step{
			{ID: "run-b", SubPipeline: "pipeline-b"},
		},
	}

	err := detectSubPipelineCycles(p, tmpDir)
	if err != nil {
		t.Errorf("expected no error for acyclic pipelines, got: %v", err)
	}
}

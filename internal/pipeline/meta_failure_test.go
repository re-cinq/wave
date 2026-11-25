package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// promptCapturingRunner records the prompt it is invoked with and then
// fails, so the philosopher call stops right after the prompt is built.
type promptCapturingRunner struct {
	prompt string
}

func (r *promptCapturingRunner) Run(_ context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	r.prompt = cfg.Prompt
	return nil, errors.New("stop after prompt")
}

func TestGenerateOnly_FailureContextReachesPhilosopherPrompt(t *testing.T) {
	t.Chdir(t.TempDir())

	failure := "Pipeline: fail-demo\n- step build [contract_failed]: boom"
	tests := []struct {
		name        string
		opts        []MetaExecutorOption
		wantSection bool
	}{
		{"with failure context", []MetaExecutorOption{WithFailureContext(failure)}, true},
		{"without failure context", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &promptCapturingRunner{}
			e := NewMetaPipelineExecutor(runner, tt.opts...)
			if _, err := e.GenerateOnly(context.Background(), "build a thing", createTestMetaManifest()); err == nil {
				t.Fatal("expected the runner error to surface")
			}
			for _, want := range []string{"## Previous Attempt Failed", "avoids these failures", "Pipeline: fail-demo", "boom"} {
				if got := strings.Contains(runner.prompt, want); got != tt.wantSection {
					t.Errorf("prompt contains %q = %v, want %v", want, got, tt.wantSection)
				}
			}
			if !strings.Contains(runner.prompt, "build a thing") {
				t.Error("prompt lost the task")
			}
		})
	}
}

func TestExecute_DepthLimitStopsBeforePhilosopher(t *testing.T) {
	runner := &promptCapturingRunner{}
	e := NewMetaPipelineExecutor(runner)
	e.currentDepth = 2
	e.parentPipelineID = "root-pipeline"

	m := &manifest.Manifest{Runtime: manifest.Runtime{MetaPipeline: manifest.MetaConfig{MaxDepth: 2}}}
	_, err := e.Execute(context.Background(), "task", m)
	if err == nil {
		t.Fatal("expected depth limit error")
	}
	for _, want := range []string{"depth limit reached: current=2 max=2", "call stack: root-pipeline -> (current)", "increase runtime.meta_pipeline.max_depth"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
	if runner.prompt != "" {
		t.Error("philosopher must not be invoked past the depth limit")
	}
}

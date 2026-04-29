package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/adapter/adaptertest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCollectParentCrossPipelineArtifacts_PopulatesFromOutputArtifacts is a
// pure-unit check of the helper used at sub-pipeline launch time. The map
// should be keyed by the parent's metadata name and contain bytes for every
// output_artifact whose path is registered in execution.ArtifactPaths.
func TestCollectParentCrossPipelineArtifacts_PopulatesFromOutputArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a parent artifact file on disk
	artifactPath := filepath.Join(tmpDir, "pr-context.json")
	contextBytes := []byte(`{"branch":"feat/foo","sha":"abc123"}`)
	require.NoError(t, os.WriteFile(artifactPath, contextBytes, 0644))

	// Build a parent execution with one declared output artifact and a
	// matching ArtifactPaths entry.
	exec := &PipelineExecution{
		Pipeline: &Pipeline{
			Metadata: PipelineMetadata{Name: "ops-pr-respond"},
			Steps: []Step{
				{
					ID: "fetch-pr",
					OutputArtifacts: []ArtifactDef{
						{Name: "pr-context", Path: ".agents/output/pr-context.json", Type: "json"},
					},
				},
			},
		},
		ArtifactPaths: map[string]string{
			"fetch-pr:pr-context": artifactPath,
		},
	}

	e := NewDefaultPipelineExecutor(nil, WithOntologyService(ontology.NoOp{}))
	got := e.collectParentCrossPipelineArtifacts(exec)

	require.NotNil(t, got)
	require.Contains(t, got, "ops-pr-respond")
	parentArts := got["ops-pr-respond"]
	require.Contains(t, parentArts, "pr-context")
	assert.Equal(t, contextBytes, parentArts["pr-context"])
}

// TestCollectParentCrossPipelineArtifacts_SkipsMissing — when an output
// artifact is declared but the file isn't on disk yet (parent step still
// running, or path was pruned), the helper silently omits the entry. The
// child's optional flag governs the missing-ref path downstream.
func TestCollectParentCrossPipelineArtifacts_SkipsMissing(t *testing.T) {
	exec := &PipelineExecution{
		Pipeline: &Pipeline{
			Metadata: PipelineMetadata{Name: "parent"},
			Steps: []Step{
				{
					ID: "produce",
					OutputArtifacts: []ArtifactDef{
						{Name: "report", Path: "report.json"},
					},
				},
			},
		},
		// No ArtifactPaths registration — step hasn't completed.
		ArtifactPaths: map[string]string{},
	}

	e := NewDefaultPipelineExecutor(nil, WithOntologyService(ontology.NoOp{}))
	got := e.collectParentCrossPipelineArtifacts(exec)
	assert.Nil(t, got, "no produced artifacts → nil map (not empty parent slot)")
}

// TestCollectParentCrossPipelineArtifacts_HonorsPipelineOutputAlias — the
// pipeline_outputs alias map is the public surface for cross-pipeline
// references in some pipelines. The helper should expose those names so
// authors can ref them directly.
func TestCollectParentCrossPipelineArtifacts_HonorsPipelineOutputAlias(t *testing.T) {
	tmpDir := t.TempDir()

	// Source artifact file
	artifactPath := filepath.Join(tmpDir, "verdict.json")
	verdictBytes := []byte(`{"pass":true}`)
	require.NoError(t, os.WriteFile(artifactPath, verdictBytes, 0644))

	exec := &PipelineExecution{
		Pipeline: &Pipeline{
			Metadata: PipelineMetadata{Name: "ops-pr-review"},
			Steps: []Step{
				{
					ID: "score",
					OutputArtifacts: []ArtifactDef{
						{Name: "review-verdict", Path: "verdict.json"},
					},
				},
			},
			PipelineOutputs: map[string]PipelineOutput{
				"verdict": {Step: "score", Artifact: "review-verdict"},
			},
		},
		ArtifactPaths: map[string]string{
			"score:review-verdict": artifactPath,
		},
	}

	e := NewDefaultPipelineExecutor(nil, WithOntologyService(ontology.NoOp{}))
	got := e.collectParentCrossPipelineArtifacts(exec)

	require.NotNil(t, got)
	require.Contains(t, got, "ops-pr-review")
	parentArts := got["ops-pr-review"]
	// Output artifact name (used by ref `artifact: review-verdict`) AND the
	// alias name (used by ref `artifact: verdict`) should both resolve.
	assert.Equal(t, verdictBytes, parentArts["review-verdict"])
	assert.Equal(t, verdictBytes, parentArts["verdict"])
}

// crossArtifactCapturingAdapter captures the AdapterRunConfig of every Run
// invocation so the test can inspect what got injected into the child
// pipeline's prompt / workspace.
type crossArtifactCapturingAdapter struct {
	*adaptertest.MockAdapter
	mu      sync.Mutex
	configs []adapter.AdapterRunConfig
}

func (a *crossArtifactCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.configs = append(a.configs, cfg)
	a.mu.Unlock()
	return a.MockAdapter.Run(ctx, cfg)
}

func (a *crossArtifactCapturingAdapter) snapshot() []adapter.AdapterRunConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]adapter.AdapterRunConfig, len(a.configs))
	copy(out, a.configs)
	return out
}

// TestSubPipelineCrossPipelineInject_Resolves verifies the end-to-end fix
// for #1551: a child sub-pipeline's `inject_artifacts` ref of the form
// `pipeline: <parent>` now resolves to the bytes the parent's matching
// output_artifact produced — without needing `optional: true` to mask a
// missing-ref failure.
func TestSubPipelineCrossPipelineInject_Resolves(t *testing.T) {
	capAdapter := &crossArtifactCapturingAdapter{
		MockAdapter: adaptertest.NewMockAdapter(
			adaptertest.WithStdoutJSON(`{"status":"ok"}`),
			adaptertest.WithTokensUsed(10),
		),
	}

	executor := NewDefaultPipelineExecutor(capAdapter, WithOntologyService(ontology.NoOp{}))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	// Child pipeline declares a cross-pipeline inject_artifacts ref. The
	// `pipeline:` form should resolve to the parent's output bytes, and
	// the workspace path should have the file present for the persona to
	// read. No `optional: true` — the test asserts the strict path works.
	childYAML := `kind: WavePipeline
metadata:
  name: child
input:
  source: cli
  type: string
steps:
  - id: consume
    persona: navigator
    memory:
      inject_artifacts:
        - pipeline: parent
          artifact: report
          as: report
    exec:
      type: prompt
      source: "consume report"
`
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "child.yaml"), []byte(childYAML), 0644))

	// Override CWD so the executor finds .agents/pipelines/ relative to tmpDir
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Parent pipeline: produce step writes report.json, then call sub-pipeline.
	reportContent := `{"finding":"missing-import","file":"foo.go"}`
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "parent"},
		Steps: []Step{
			{
				ID:     "produce",
				Type:   StepTypeCommand,
				Script: fmt.Sprintf("mkdir -p .agents/output && printf '%%s' '%s' > .agents/output/report.json", reportContent),
				OutputArtifacts: []ArtifactDef{
					{Name: "report", Path: ".agents/output/report.json", Type: "json"},
				},
			},
			{
				ID:           "call-child",
				Dependencies: []string{"produce"},
				SubPipeline:  "child",
				SubInput:     "{{ input }}",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"))

	// Inspect the child's adapter call. The injected file must exist at
	// the workspace path the framework chose for the child's `consume`
	// step, and contain the parent's report bytes.
	configs := capAdapter.snapshot()
	require.NotEmpty(t, configs)

	// Find the child's consume step run by its workspace path
	// (.agents/artifacts/report inside the child workspace).
	var injected []byte
	walkErr := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "report" && strings.Contains(path, filepath.Join("consume", ".agents", "artifacts")) {
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				injected = data
			}
		}
		return nil
	})
	require.NoError(t, walkErr)
	require.NotNil(t, injected, "cross-pipeline injected artifact must exist on disk for the child step")
	assert.Equal(t, reportContent, string(injected),
		"injected report content must match parent's output bytes")
}

// TestSubPipelineCrossPipelineInject_MissingNonOptionalErrors verifies the
// negative path: when the parent never produced the referenced artifact and
// the ref is NOT marked optional, the child fails loudly. Confirms the
// fix didn't accidentally weaken the strict semantics.
func TestSubPipelineCrossPipelineInject_MissingNonOptionalErrors(t *testing.T) {
	capAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
	)

	executor := NewDefaultPipelineExecutor(capAdapter, WithOntologyService(ontology.NoOp{}))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	// Child wants a 'missing' artifact from parent that parent never produces.
	childYAML := `kind: WavePipeline
metadata:
  name: child2
input:
  source: cli
  type: string
steps:
  - id: consume
    persona: navigator
    memory:
      inject_artifacts:
        - pipeline: parent2
          artifact: missing
          as: missing
    exec:
      type: prompt
      source: "consume"
`
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "child2.yaml"), []byte(childYAML), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "parent2"},
		Steps: []Step{
			{
				ID:          "call-child",
				SubPipeline: "child2",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "child must fail when non-optional cross-pipeline ref is unresolved")
	// Surface the leaf inject error somewhere in the chain.
	assert.Contains(t, err.Error(), "missing",
		"error message should reference the missing artifact name")
}

// TestSubPipelineCrossPipelineInject_OptionalSkips verifies the optional
// semantics survive: when the parent never produced the referenced
// artifact and the ref IS marked optional, the child completes
// successfully (skipping the inject).
func TestSubPipelineCrossPipelineInject_OptionalSkips(t *testing.T) {
	capAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
	)

	executor := NewDefaultPipelineExecutor(capAdapter, WithOntologyService(ontology.NoOp{}))

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	pipelinesDir := filepath.Join(tmpDir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0755))

	childYAML := `kind: WavePipeline
metadata:
  name: child3
input:
  source: cli
  type: string
steps:
  - id: consume
    persona: navigator
    memory:
      inject_artifacts:
        - pipeline: parent3
          artifact: maybe
          as: maybe
          optional: true
    exec:
      type: prompt
      source: "consume"
`
	require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, "child3.yaml"), []byte(childYAML), 0644))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "parent3"},
		Steps: []Step{
			{
				ID:          "call-child",
				SubPipeline: "child3",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(t, executor.Execute(ctx, p, m, "test"),
		"optional cross-pipeline ref should not fail the child when parent never produced it")
}

// TestSubPipelineCrossPipelineArtifacts_ScopeIsParentOnly — the constraint
// in #1551 is that the child sees ONLY the direct parent's artifacts, not
// transitively the grandparent's (e.g. a sequence sibling above the
// parent). This guards against unintentional widening of cross-pipeline
// visibility.
func TestSubPipelineCrossPipelineArtifacts_ScopeIsParentOnly(t *testing.T) {
	// Synthesize a parent execution that itself was launched with
	// cross-pipeline artifacts from a grandparent. The helper must ignore
	// those — only the parent's own produced artifacts may flow.
	tmpDir := t.TempDir()
	parentArtPath := filepath.Join(tmpDir, "parent-out.json")
	require.NoError(t, os.WriteFile(parentArtPath, []byte(`"parent-bytes"`), 0644))

	parent := NewDefaultPipelineExecutor(nil,
		WithOntologyService(ontology.NoOp{}),
		// Simulate that this parent itself was launched via a sequence —
		// it carries grandparent's artifacts.
		WithCrossPipelineArtifacts(map[string]map[string][]byte{
			"grandparent": {"deep-secret": []byte(`"do-not-leak"`)},
		}),
	)

	exec := &PipelineExecution{
		Pipeline: &Pipeline{
			Metadata: PipelineMetadata{Name: "the-parent"},
			Steps: []Step{
				{
					ID: "produce",
					OutputArtifacts: []ArtifactDef{
						{Name: "parent-out", Path: "parent-out.json"},
					},
				},
			},
		},
		ArtifactPaths: map[string]string{
			"produce:parent-out": parentArtPath,
		},
	}

	got := parent.collectParentCrossPipelineArtifacts(exec)

	require.NotNil(t, got)
	// Parent's slot should be present
	require.Contains(t, got, "the-parent")
	assert.Equal(t, []byte(`"parent-bytes"`), got["the-parent"]["parent-out"])
	// Grandparent's slot must NOT have been forwarded
	assert.NotContains(t, got, "grandparent",
		"grandparent's artifacts must not transitively flow to the child")
}

// TestSequencePipelineCrossArtifactsStillWork — regression guard that the
// existing sequence-pipeline cross-pipeline flow keeps producing the same
// captured map shape after the sub-pipeline propagation lands. Mirrors
// the assertion in TestSequenceExecutor_CrossPipelineArtifacts.
func TestSequencePipelineCrossArtifactsStillWork(t *testing.T) {
	mockAdapter := adaptertest.NewMockAdapter(
		adaptertest.WithStdoutJSON(`{"status":"ok"}`),
	)

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)

	// Capture per-executor crossPipelineArtifacts map values.
	var captured []map[string]map[string][]byte
	var mu sync.Mutex

	seq := NewSequenceExecutor(
		func(opts ...ExecutorOption) *DefaultPipelineExecutor {
			ex := NewDefaultPipelineExecutor(mockAdapter,
				append([]ExecutorOption{WithOntologyService(ontology.NoOp{})}, opts...)...)
			mu.Lock()
			if ex.crossPipelineArtifacts != nil {
				captured = append(captured, ex.crossPipelineArtifacts)
			}
			mu.Unlock()
			return ex
		},
		nil, nil, nil,
	)

	seq.pipelineOutputs["upstream"] = map[string][]byte{
		"report.json": []byte(`{"r":"ok"}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := seq.Execute(ctx, []*Pipeline{
		{
			Metadata: PipelineMetadata{Name: "downstream"},
			Steps: []Step{
				{ID: "only-step", Persona: "navigator", Exec: ExecConfig{Source: "do thing"}},
			},
		},
	}, m, "input")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, captured, 1)
	assert.Contains(t, captured[0], "upstream")
	assert.Equal(t, []byte(`{"r":"ok"}`), captured[0]["upstream"]["report.json"])
}


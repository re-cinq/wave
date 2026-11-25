package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingRunStore records the run-linkage calls runNamedSubPipeline makes.
type recordingRunStore struct {
	*testutil.MockStateStore
	mu          sync.Mutex
	parents     []string // "child<-parent/step"
	composition []string // "kind:ref:mode"
	statuses    []string // "status"
	stepStates  []string // "step=state"
	branch      string
}

func newRecordingRunStore() *recordingRunStore {
	return &recordingRunStore{MockStateStore: testutil.NewMockStateStore()}
}

func (s *recordingRunStore) CreateRun(pipelineName, _ string) (string, error) {
	return pipelineName + "-child-run", nil
}

func (s *recordingRunStore) SetParentRun(child, parent, step string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parents = append(s.parents, child+"<-"+parent+"/"+step)
	return nil
}

func (s *recordingRunStore) SetRunComposition(_, kind, ref, mode string, _, _ *int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.composition = append(s.composition, kind+":"+ref+":"+mode)
	return nil
}

func (s *recordingRunStore) UpdateRunStatus(runID, status, _ string, _ int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if runID != "parent-run" {
		s.statuses = append(s.statuses, status)
	}
	return nil
}

func (s *recordingRunStore) SaveStepState(pipelineID, stepID string, st state.StepState, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pipelineID == "parent-run" {
		s.stepStates = append(s.stepStates, stepID+"="+string(st))
	}
	return nil
}

func (s *recordingRunStore) GetRun(runID string) (*state.RunRecord, error) {
	return &state.RunRecord{RunID: runID, BranchName: "wave/child-branch"}, nil
}

func (s *recordingRunStore) UpdateRunBranch(runID, branch string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if runID == "parent-run" {
		s.branch = branch
	}
	return nil
}

// subPipelineFixture chdirs into a fresh project with a manifest and writes
// the given child pipelines under .agents/pipelines.
func subPipelineFixture(t *testing.T, children map[string]string) *manifest.Manifest {
	t.Helper()
	dir := t.TempDir()
	m := testutil.CreateTestManifest(dir)
	pipelinesDir := filepath.Join(dir, ".agents", "pipelines")
	require.NoError(t, os.MkdirAll(pipelinesDir, 0o755))
	for name, body := range children {
		require.NoError(t, os.WriteFile(filepath.Join(pipelinesDir, name+".yaml"), []byte(body), 0o644))
	}
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return m
}

func newParentExecution(m *manifest.Manifest, steps ...Step) *PipelineExecution {
	p := &Pipeline{Metadata: PipelineMetadata{Name: "parent"}, Steps: steps}
	return &PipelineExecution{
		Pipeline:       p,
		Manifest:       m,
		States:         map[string]string{},
		ArtifactPaths:  map[string]string{},
		WorkspacePaths: map[string]string{},
		Status:         &PipelineStatus{ID: "parent-run"},
		Context:        NewPipelineContext("parent-run", "parent", ""),
	}
}

// childProducing is a one-step child pipeline whose command step writes
// content to an output artifact named art.
const childProducing = `kind: WavePipeline
metadata:
  name: child
steps:
  - id: make
    type: command
    script: "mkdir -p .agents/output && printf done > .agents/output/out.txt"
    output_artifacts:
      - name: art
        path: .agents/output/out.txt
        type: text
`

func TestRunNamedSubPipeline_LoadFailureMarksStepFailed(t *testing.T) {
	m := subPipelineFixture(t, nil)
	store := newRecordingRunStore()
	e := NewDefaultPipelineExecutor(nil, WithStateStore(store))
	step := &Step{ID: "call", SubPipeline: "missing"}
	execution := newParentExecution(m, *step)

	err := e.runNamedSubPipeline(context.Background(), execution, step, "missing", "in", compositionLaunchInfo{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `failed to load sub-pipeline "missing"`)
	assert.Equal(t, stateFailed, execution.States["call"])
	assert.Equal(t, []string{"call=failed"}, store.stepStates)
	assert.Empty(t, store.parents, "no child run is created when loading fails")
}

func TestRunNamedSubPipeline_MissingInjectedArtifact(t *testing.T) {
	m := subPipelineFixture(t, map[string]string{"child": childProducing})
	e := NewDefaultPipelineExecutor(nil)
	step := &Step{ID: "call", SubPipeline: "child", Config: &SubPipelineConfig{Inject: []string{"report"}}}
	execution := newParentExecution(m, *step)

	err := e.runNamedSubPipeline(context.Background(), execution, step, "child", "in", compositionLaunchInfo{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `artifact "report" not found in parent context for sub-pipeline injection`)
}

func TestRunNamedSubPipeline_InjectFallsBackToDependencyArtifacts(t *testing.T) {
	m := subPipelineFixture(t, map[string]string{"child": childProducing})
	e := NewDefaultPipelineExecutor(nil)
	step := &Step{
		ID:           "call",
		SubPipeline:  "child",
		Dependencies: []string{"fetch"},
		Config:       &SubPipelineConfig{Inject: []string{"report"}},
	}
	execution := newParentExecution(m, Step{ID: "fetch"}, *step)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, os.WriteFile(reportPath, []byte(`{}`), 0o644))
	execution.ArtifactPaths["fetch:report"] = reportPath

	require.NoError(t, e.runNamedSubPipeline(context.Background(), execution, step, "child", "in", compositionLaunchInfo{}))
	assert.Equal(t, stateCompleted, execution.States["call"])
}

func TestRunNamedSubPipeline_SuccessLinksRunsAndPropagatesArtifacts(t *testing.T) {
	m := subPipelineFixture(t, map[string]string{"child": childProducing})
	store := newRecordingRunStore()
	e := NewDefaultPipelineExecutor(nil, WithStateStore(store))
	step := &Step{ID: "call", SubPipeline: "child"}
	execution := newParentExecution(m, *step)

	idx, total := 1, 3
	info := compositionLaunchInfo{kind: "iterate_child", iterateIndex: &idx, iterateTotal: &total, iterateMode: "serial"}
	require.NoError(t, e.runNamedSubPipeline(context.Background(), execution, step, "child", "in", info))

	assert.Equal(t, stateCompleted, execution.States["call"])
	assert.Equal(t, []string{"call=completed"}, store.stepStates)
	assert.Equal(t, []string{"iterate_child:child:serial"}, store.composition)
	assert.Equal(t, "running", store.statuses[0])
	assert.Equal(t, "completed", store.statuses[len(store.statuses)-1])
	require.Len(t, store.parents, 2, "child is linked at creation and again on completion")
	assert.Equal(t, store.parents[0], store.parents[1])
	assert.Equal(t, "child-child-run<-parent-run/call", store.parents[0])
	assert.Equal(t, "wave/child-branch", store.branch, "child branch is propagated to the parent run")

	nsPath := execution.Context.GetArtifactPath("child.make:art")
	require.NotEmpty(t, nsPath, "child artifact is registered under <pipeline>.<step>:<name>")
	data, err := os.ReadFile(nsPath)
	require.NoError(t, err)
	assert.Equal(t, "done", string(data))
	assert.Equal(t, nsPath, execution.ArtifactPaths["call:art"],
		"bare sub-pipeline steps re-register child outputs under <step>:<name>")
}

func TestRunNamedSubPipeline_IterateStepDoesNotReRegisterUnderStepID(t *testing.T) {
	m := subPipelineFixture(t, map[string]string{"child": childProducing})
	e := NewDefaultPipelineExecutor(nil)
	step := &Step{ID: "fan", Iterate: &IterateConfig{Over: "[]"}}
	execution := newParentExecution(m, *step)

	require.NoError(t, e.runNamedSubPipeline(context.Background(), execution, step, "child", "in", compositionLaunchInfo{}))

	assert.NotEmpty(t, execution.Context.GetArtifactPath("child.make:art"))
	_, registered := execution.ArtifactPaths["fan:art"]
	assert.False(t, registered)
}

func TestRunNamedSubPipeline_ChildFailure(t *testing.T) {
	m := subPipelineFixture(t, map[string]string{"bad": `kind: WavePipeline
metadata:
  name: bad
steps:
  - id: boom
    type: command
    script: "exit 3"
`})
	store := newRecordingRunStore()
	e := NewDefaultPipelineExecutor(nil, WithStateStore(store))
	step := &Step{ID: "call", SubPipeline: "bad"}
	execution := newParentExecution(m, *step)

	err := e.runNamedSubPipeline(context.Background(), execution, step, "bad", "in", compositionLaunchInfo{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `sub-pipeline "bad" failed`)
	assert.Equal(t, stateFailed, execution.States["call"])
	assert.Equal(t, []string{"call=failed"}, store.stepStates)
	assert.Equal(t, "failed", store.statuses[len(store.statuses)-1])
	assert.Equal(t, []string{"sub_pipeline_child:bad:"}, store.composition, "zero launch info defaults to sub_pipeline_child")
}

func TestRunNamedSubPipeline_EnvExtractAndStopCondition(t *testing.T) {
	// The child writes the merged {{ env.* }} values into its artifact.
	m := subPipelineFixture(t, map[string]string{"child": `kind: WavePipeline
metadata:
  name: child
steps:
  - id: make
    type: command
    script: "mkdir -p .agents/output && printf '{{ env.greeting }}-{{ env.who }}' > .agents/output/out.txt"
    output_artifacts:
      - name: art
        path: .agents/output/out.txt
        type: text
`})
	e := NewDefaultPipelineExecutor(nil, WithParentEnv(map[string]string{"greeting": "hello", "who": "parent"}))
	parentWS := t.TempDir()
	step := &Step{ID: "call", SubPipeline: "child", Config: &SubPipelineConfig{
		Env:           map[string]string{"who": "step"},
		Extract:       []string{"art"},
		StopCondition: "true",
	}}
	execution := newParentExecution(m, *step)
	execution.WorkspacePaths["call"] = parentWS
	// The child context has no "art" entry, so extraction falls back to the
	// parent workspace's artifacts directory.
	require.NoError(t, os.MkdirAll(filepath.Join(parentWS, ".agents", "artifacts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parentWS, ".agents", "artifacts", "art"), []byte("from-workspace"), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, e.runNamedSubPipeline(ctx, execution, step, "child", "in", compositionLaunchInfo{}))

	produced, err := os.ReadFile(execution.Context.GetArtifactPath("child.make:art"))
	require.NoError(t, err)
	assert.Equal(t, "hello-step", string(produced), "step env overrides inherited parent env")

	extracted, err := os.ReadFile(filepath.Join(parentWS, ".agents", "artifacts", "child.art"))
	require.NoError(t, err, "extract copies the artifact into the parent workspace under <pipeline>.<name>")
	assert.Equal(t, "from-workspace", string(extracted))
	assert.Equal(t, "true", execution.Context.CustomVariables["sub_pipeline_stop"])
}

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/tui"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// composeTestHelper provides common utilities for compose command tests.
type composeTestHelper struct {
	t       *testing.T
	tmpDir  string
	origDir string
}

// newComposeTestHelper creates a test helper with a temporary directory
// pre-populated with a wave.yaml so checkOnboarding() grandfathers in.
func newComposeTestHelper(t *testing.T) *composeTestHelper {
	t.Helper()
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	h := &composeTestHelper{
		t:       t,
		tmpDir:  tmpDir,
		origDir: origDir,
	}

	// Create wave.yaml so checkOnboarding() grandfathers the project in.
	h.writeFile("wave.yaml", "apiVersion: v1\nkind: WaveManifest\nmetadata:\n  name: test\nruntime:\n  workspace_root: .agents/workspaces\n")

	// Create the pipelines directory.
	err = os.MkdirAll(filepath.Join(tmpDir, ".agents", "pipelines"), 0755)
	require.NoError(t, err, "failed to create pipelines directory")

	return h
}

// chdir changes the working directory to the temp directory.
func (h *composeTestHelper) chdir() {
	h.t.Helper()
	err := os.Chdir(h.tmpDir)
	require.NoError(h.t, err, "failed to chdir to temp directory")
}

// restore returns to the original working directory.
func (h *composeTestHelper) restore() {
	h.t.Helper()
	_ = os.Chdir(h.origDir)
}

// writeFile writes content to a file relative to the temp directory.
func (h *composeTestHelper) writeFile(relPath, content string) {
	h.t.Helper()
	fullPath := filepath.Join(h.tmpDir, relPath)
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(h.t, err, "failed to create directory: %s", dir)
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(h.t, err, "failed to write file: %s", relPath)
}

// newComposeCmdWithRoot creates a compose command under a root that has the
// persistent flags that the real CLI provides (output, verbose, debug).
func newComposeCmdWithRoot() *cobra.Command {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().StringP("output", "o", "auto", "Output format")
	root.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	root.PersistentFlags().Bool("debug", false, "Debug mode")
	composeCmd := NewComposeCmd()
	root.AddCommand(composeCmd)
	return root
}

// Pipeline YAML fixtures -----------------------------------------------

// pipelineA has output artifacts in its last step.
const pipelineAYAML = `kind: WavePipeline
metadata:
  name: pipeline-a
  description: Produces analysis output
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze the codebase"
    output_artifacts:
      - name: analysis
        path: .agents/output/analysis.json
        type: json
      - name: summary
        path: .agents/output/summary.md
        type: markdown
`

// pipelineB has inject_artifacts in its first step that match pipeline-a's
// outputs (artifact name "analysis").
const pipelineBYAML = `kind: WavePipeline
metadata:
  name: pipeline-b
  description: Consumes analysis from pipeline-a
steps:
  - id: implement
    persona: craftsman
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: analysis.json
    exec:
      type: prompt
      source: "Implement the feature"
`

// pipelineC has inject_artifacts in its first step that do NOT match
// pipeline-a's outputs — the required artifact name "design_doc" does not
// exist in pipeline-a.
const pipelineCYAML = `kind: WavePipeline
metadata:
  name: pipeline-c
  description: Expects a design doc that pipeline-a does not produce
steps:
  - id: review
    persona: navigator
    memory:
      strategy: fresh
      inject_artifacts:
        - step: plan
          artifact: design_doc
          as: design_doc.md
    exec:
      type: prompt
      source: "Review the design"
`

// T023-1: compose with valid compatible pipelines exits 0
func TestComposeCmd_ValidPipelines(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)
	h.writeFile(".agents/pipelines/pipeline-b.yaml", pipelineBYAML)

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a", "pipeline-b", "--validate-only"})

	err := root.Execute()
	assert.NoError(t, err, "compose with compatible pipelines should succeed")
}

// T023-2: compose with incompatible artifacts exits with error
func TestComposeCmd_IncompatibleArtifacts(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)
	h.writeFile(".agents/pipelines/pipeline-c.yaml", pipelineCYAML)

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a", "pipeline-c", "--validate-only"})

	err := root.Execute()
	assert.Error(t, err, "compose with incompatible artifacts should fail")
	assert.Contains(t, err.Error(), "incompatible artifact flows",
		"error should mention incompatible artifact flows")
}

// T023-3: compose --validate-only prints compatibility report with boundary info
func TestComposeCmd_ValidateOnlyPrintsReport(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)
	h.writeFile(".agents/pipelines/pipeline-b.yaml", pipelineBYAML)

	// Capture stdout — renderValidationReport writes to os.Stdout.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to create pipe")
	os.Stdout = w

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a", "pipeline-b", "--validate-only"})

	cmdErr := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, cmdErr, "validate-only with compatible pipelines should not error")
	assert.Contains(t, output, "Boundary 1", "report should contain boundary information")
	assert.Contains(t, output, "pipeline-a", "report should mention source pipeline")
	assert.Contains(t, output, "pipeline-b", "report should mention target pipeline")
	assert.Contains(t, output, "analysis", "report should mention the matched artifact")
	assert.Contains(t, output, "compatible", "report should indicate compatible artifact")
	assert.Contains(t, output, "0 error(s)", "report should show zero errors")
}

// T023-4: compose with only one argument errors
func TestComposeCmd_TooFewArgs(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a"})

	err := root.Execute()
	assert.Error(t, err, "compose with a single pipeline should fail")
	assert.Contains(t, err.Error(), "requires at least 2 arg(s)",
		"error should mention minimum argument count")
}

// T023-5: compose with nonexistent pipeline errors
func TestComposeCmd_NonexistentPipeline(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a", "does-not-exist"})

	err := root.Execute()
	assert.Error(t, err, "compose with nonexistent pipeline should fail")
	assert.Contains(t, err.Error(), "pipeline not found",
		"error should contain 'pipeline not found'")
}

// T023-extra: compose without --validate-only with incompatible pipelines errors
func TestComposeCmd_ExecutionModeIncompatible(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)
	h.writeFile(".agents/pipelines/pipeline-c.yaml", pipelineCYAML)

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a", "pipeline-c"})

	err := root.Execute()
	assert.Error(t, err, "execution mode with incompatible pipelines should fail")
	assert.Contains(t, err.Error(), "incompatible artifact flows",
		"error should mention incompatible artifact flows")
}

// T023-extra: compose without --validate-only with compatible pipelines
// attempts sequential execution. The test uses --mock to avoid real adapter
// calls; execution may still fail in the minimal test environment, but the
// key assertion is that the "Executing sequence:" banner is emitted —
// proving the blocking #249 message has been replaced with actual execution.
func TestComposeCmd_ExecutionModeCompatible(t *testing.T) {
	h := newComposeTestHelper(t)
	h.chdir()
	defer h.restore()

	h.writeFile(".agents/pipelines/pipeline-a.yaml", pipelineAYAML)
	h.writeFile(".agents/pipelines/pipeline-b.yaml", pipelineBYAML)

	// Capture stderr since the command writes informational messages there.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to create pipe")
	os.Stderr = w

	root := newComposeCmdWithRoot()
	root.SetArgs([]string{"compose", "pipeline-a", "pipeline-b", "--mock"})

	_ = root.Execute()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "Executing sequence:",
		"output should show the sequence execution banner")
	assert.Contains(t, output, "pipeline-a",
		"output should mention the first pipeline")
}

func TestBuildExecutionPlan_MultipleParallelGroups(t *testing.T) {
	// Simulate: wave compose --parallel A B -- C D -- E
	// Expected: Stage 1 (parallel: A,B), Stage 2 (parallel: C,D), Stage 3 (sequential: E)

	newPipeline := func(name string) *pipeline.Pipeline {
		return &pipeline.Pipeline{
			Metadata: pipeline.PipelineMetadata{Name: name},
			Steps: []pipeline.Step{
				{ID: "step1", Persona: "navigator", Exec: pipeline.ExecConfig{Source: "do it"}},
			},
		}
	}

	var seq tui.Sequence
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		seq.Add(name, newPipeline(name))
	}

	args := []string{"A", "B", "--", "C", "D", "--", "E"}
	plan := buildExecutionPlan(seq, args)

	require.Len(t, plan.Stages, 3, "should have 3 stages")

	// Stage 1: A, B — parallel
	assert.Len(t, plan.Stages[0].Pipelines, 2)
	assert.True(t, plan.Stages[0].Parallel, "stage 1 should be parallel")
	assert.Equal(t, "A", plan.Stages[0].Pipelines[0].Metadata.Name)
	assert.Equal(t, "B", plan.Stages[0].Pipelines[1].Metadata.Name)

	// Stage 2: C, D — parallel
	assert.Len(t, plan.Stages[1].Pipelines, 2)
	assert.True(t, plan.Stages[1].Parallel, "stage 2 should be parallel")
	assert.Equal(t, "C", plan.Stages[1].Pipelines[0].Metadata.Name)
	assert.Equal(t, "D", plan.Stages[1].Pipelines[1].Metadata.Name)

	// Stage 3: E — sequential (single pipeline)
	assert.Len(t, plan.Stages[2].Pipelines, 1)
	assert.False(t, plan.Stages[2].Parallel, "stage 3 with single pipeline should be sequential")
	assert.Equal(t, "E", plan.Stages[2].Pipelines[0].Metadata.Name)
}

func TestBuildExecutionPlan_SingleGroupParallel(t *testing.T) {
	// Simulate: wave compose --parallel A B C (no -- separators)
	// Expected: Stage 1 (parallel: A,B,C)

	newPipeline := func(name string) *pipeline.Pipeline {
		return &pipeline.Pipeline{
			Metadata: pipeline.PipelineMetadata{Name: name},
			Steps: []pipeline.Step{
				{ID: "step1", Persona: "navigator", Exec: pipeline.ExecConfig{Source: "do it"}},
			},
		}
	}

	var seq tui.Sequence
	for _, name := range []string{"A", "B", "C"} {
		seq.Add(name, newPipeline(name))
	}

	args := []string{"A", "B", "C"}
	plan := buildExecutionPlan(seq, args)

	require.Len(t, plan.Stages, 1)
	assert.Len(t, plan.Stages[0].Pipelines, 3)
	assert.True(t, plan.Stages[0].Parallel, "single multi-pipeline group should be parallel")
}

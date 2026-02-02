package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/recinq/wave/internal/manifest"
)

// createAdhocTestManifest creates a minimal manifest with the specified personas
func createAdhocTestManifest(personas []string) *manifest.Manifest {
	m := &manifest.Manifest{
		APIVersion: "v1",
		Kind:       "WaveManifest",
		Metadata: manifest.Metadata{
			Name:        "test-manifest",
			Description: "Test manifest for adhoc pipeline tests",
		},
		Adapters: map[string]manifest.Adapter{
			"claude": {
				Binary:       "claude",
				Mode:         "headless",
				OutputFormat: "json",
			},
		},
		Personas: make(map[string]manifest.Persona),
		Runtime: manifest.Runtime{
			WorkspaceRoot:        ".wave/workspaces",
			MaxConcurrentWorkers: 5,
			DefaultTimeoutMin:    30,
		},
	}

	for _, p := range personas {
		m.Personas[p] = manifest.Persona{
			Adapter:          "claude",
			SystemPromptFile: "personas/" + p + ".md",
			Temperature:      0.7,
			Permissions: manifest.Permissions{
				AllowedTools: []string{"Read", "Write"},
				Deny:         []string{},
			},
		}
	}

	return m
}

// TestGenerateAdHocPipeline_BasicGeneration tests that a basic adhoc pipeline
// is generated correctly with navigate and execute steps
func TestGenerateAdHocPipeline_BasicGeneration(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature X",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Verify pipeline structure
	assert.Equal(t, "WavePipeline", p.Kind)
	assert.Equal(t, "adhoc", p.Metadata.Name)
	assert.Equal(t, "Ad-hoc generated pipeline", p.Metadata.Description)
	assert.Equal(t, "cli", p.Input.Source)

	// Verify we have exactly 2 steps
	assert.Len(t, p.Steps, 2)
}

// TestGenerateAdHocPipeline_NavigateStep tests that the navigate step is
// correctly configured
func TestGenerateAdHocPipeline_NavigateStep(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find navigate step
	var navigateStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "navigate" {
			navigateStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, navigateStep, "navigate step should exist")

	// Verify navigate step configuration
	assert.Equal(t, "navigate", navigateStep.ID)
	assert.Equal(t, "navigator", navigateStep.Persona)
	assert.Equal(t, "fresh", navigateStep.Memory.Strategy)
	assert.Empty(t, navigateStep.Dependencies)

	// Verify workspace config
	assert.Equal(t, "./", navigateStep.Workspace.Root)
	require.NotEmpty(t, navigateStep.Workspace.Mount)
	assert.Equal(t, "readonly", navigateStep.Workspace.Mount[0].Mode)

	// Verify exec config
	assert.Equal(t, "prompt", navigateStep.Exec.Type)
	assert.Contains(t, navigateStep.Exec.Source, "implement feature")

	// Verify output artifacts
	require.NotEmpty(t, navigateStep.OutputArtifacts)
	assert.Equal(t, "analysis", navigateStep.OutputArtifacts[0].Name)
	assert.Equal(t, "json", navigateStep.OutputArtifacts[0].Type)
}

// TestGenerateAdHocPipeline_ExecuteStep tests that the execute step is
// correctly configured
func TestGenerateAdHocPipeline_ExecuteStep(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find execute step
	var executeStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep, "execute step should exist")

	// Verify execute step configuration
	assert.Equal(t, "execute", executeStep.ID)
	assert.Equal(t, "craftsman", executeStep.Persona)
	assert.Equal(t, "fresh", executeStep.Memory.Strategy)

	// Verify dependencies on navigate
	assert.Contains(t, executeStep.Dependencies, "navigate")

	// Verify workspace config
	assert.Equal(t, "./", executeStep.Workspace.Root)
	require.NotEmpty(t, executeStep.Workspace.Mount)
	assert.Equal(t, "readwrite", executeStep.Workspace.Mount[0].Mode)

	// Verify exec config
	assert.Equal(t, "prompt", executeStep.Exec.Type)
	assert.Contains(t, executeStep.Exec.Source, "implement feature")

	// Verify handover config
	assert.Equal(t, "test_suite", executeStep.Handover.Contract.Type)
	assert.Equal(t, "go test ./...", executeStep.Handover.Contract.Command)
	assert.True(t, executeStep.Handover.Contract.MustPass)
}

// TestGenerateAdHocPipeline_ArtifactInjection tests that artifact injection
// is correctly configured from navigate to execute
func TestGenerateAdHocPipeline_ArtifactInjection(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find execute step
	var executeStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)

	// Verify artifact injection
	require.NotEmpty(t, executeStep.Memory.InjectArtifacts)

	artifactRef := executeStep.Memory.InjectArtifacts[0]
	assert.Equal(t, "navigate", artifactRef.Step)
	assert.Equal(t, "analysis", artifactRef.Artifact)
	assert.Equal(t, "navigation_report", artifactRef.As)
}

// TestGenerateAdHocPipeline_CustomNavigatorPersona tests that a custom
// navigator persona can be specified
func TestGenerateAdHocPipeline_CustomNavigatorPersona(t *testing.T) {
	m := createAdhocTestManifest([]string{"custom-navigator", "craftsman"})

	opts := AdHocOptions{
		Input:            "implement feature",
		NavigatorPersona: "custom-navigator",
		ExecutePersona:   "craftsman",
		Manifest:         m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find navigate step
	var navigateStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "navigate" {
			navigateStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, navigateStep)
	assert.Equal(t, "custom-navigator", navigateStep.Persona)
}

// TestGenerateAdHocPipeline_DefaultNavigatorPersona tests that the default
// navigator persona is used when not specified
func TestGenerateAdHocPipeline_DefaultNavigatorPersona(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find navigate step
	var navigateStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "navigate" {
			navigateStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, navigateStep)
	assert.Equal(t, DefaultNavigatorPersona, navigateStep.Persona)
}

// TestGenerateAdHocPipeline_CustomExecutePersona tests that a custom execute
// persona is used correctly
func TestGenerateAdHocPipeline_CustomExecutePersona(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "architect"})

	opts := AdHocOptions{
		Input:          "design system",
		ExecutePersona: "architect",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find execute step
	var executeStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)
	assert.Equal(t, "architect", executeStep.Persona)
}

// TestGenerateAdHocPipeline_NilManifest tests that nil manifest returns error
func TestGenerateAdHocPipeline_NilManifest(t *testing.T) {
	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       nil,
	}

	p, err := GenerateAdHocPipeline(opts)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "manifest is required")
}

// TestGenerateAdHocPipeline_MissingExecutePersona tests that missing execute
// persona returns error
func TestGenerateAdHocPipeline_MissingExecutePersona(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "", // Empty
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "executePersona is required")
}

// TestGenerateAdHocPipeline_MissingNavigatorInManifest tests that referencing
// a missing navigator persona returns error
func TestGenerateAdHocPipeline_MissingNavigatorInManifest(t *testing.T) {
	m := createAdhocTestManifest([]string{"craftsman"}) // No navigator

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "navigator")
	assert.Contains(t, err.Error(), "not found")
}

// TestGenerateAdHocPipeline_MissingExecutePersonaInManifest tests that
// referencing a missing execute persona returns error
func TestGenerateAdHocPipeline_MissingExecutePersonaInManifest(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator"}) // No craftsman

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "craftsman")
	assert.Contains(t, err.Error(), "not found")
}

// TestGenerateAdHocPipeline_InputInPrompt tests that the input is embedded
// in both step prompts
func TestGenerateAdHocPipeline_InputInPrompt(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	testInput := "implement the frobnicator widget"
	opts := AdHocOptions{
		Input:          testInput,
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Verify input appears in both prompts
	for _, step := range p.Steps {
		assert.Contains(t, step.Exec.Source, testInput,
			"step %s should contain input in prompt", step.ID)
	}
}

// TestGenerateAdHocPipeline_NavigateHandoverConfig tests that navigate step
// has proper handover/contract configuration
func TestGenerateAdHocPipeline_NavigateHandoverConfig(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find navigate step
	var navigateStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "navigate" {
			navigateStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, navigateStep)

	// Verify handover config
	assert.Equal(t, "json_schema", navigateStep.Handover.Contract.Type)
	assert.Equal(t, "retry", navigateStep.Handover.Contract.OnFailure)
	assert.Equal(t, 2, navigateStep.Handover.Contract.MaxRetries)
}

// TestGenerateAdHocPipeline_ExecuteCompactionConfig tests that execute step
// has proper compaction configuration
func TestGenerateAdHocPipeline_ExecuteCompactionConfig(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Find execute step
	var executeStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)

	// Verify compaction config
	assert.Equal(t, "token_limit_80%", executeStep.Handover.Compaction.Trigger)
	assert.Equal(t, "summarizer", executeStep.Handover.Compaction.Persona)
}

// TestGenerateNavigateStep tests the generateNavigateStep helper function
func TestGenerateNavigateStep(t *testing.T) {
	step := generateNavigateStep("custom-nav", "test input")

	assert.Equal(t, "navigate", step.ID)
	assert.Equal(t, "custom-nav", step.Persona)
	assert.Equal(t, "fresh", step.Memory.Strategy)
	assert.Empty(t, step.Dependencies)
	assert.Equal(t, "prompt", step.Exec.Type)
	assert.Contains(t, step.Exec.Source, "test input")
	assert.Contains(t, step.Exec.Source, "{{ input }}")
}

// TestGenerateExecuteStep tests the generateExecuteStep helper function
func TestGenerateExecuteStep(t *testing.T) {
	step := generateExecuteStep("custom-exec", "test input")

	assert.Equal(t, "execute", step.ID)
	assert.Equal(t, "custom-exec", step.Persona)
	assert.Equal(t, "fresh", step.Memory.Strategy)
	assert.Contains(t, step.Dependencies, "navigate")
	assert.Equal(t, "prompt", step.Exec.Type)
	assert.Contains(t, step.Exec.Source, "test input")
}

// TestInjectArtifacts tests the injectArtifacts helper function
func TestInjectArtifacts(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "navigate", Persona: "navigator"},
			{ID: "execute", Persona: "craftsman"},
		},
	}

	injectArtifacts(p)

	// Verify execute step has artifact injection
	var executeStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == "execute" {
			executeStep = &p.Steps[i]
			break
		}
	}
	require.NotNil(t, executeStep)
	require.NotEmpty(t, executeStep.Memory.InjectArtifacts)

	ref := executeStep.Memory.InjectArtifacts[0]
	assert.Equal(t, "navigate", ref.Step)
	assert.Equal(t, "analysis", ref.Artifact)
	assert.Equal(t, "navigation_report", ref.As)
}

// TestInjectArtifacts_SingleStep tests that injectArtifacts handles
// pipelines with less than 2 steps
func TestInjectArtifacts_SingleStep(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "navigate", Persona: "navigator"},
		},
	}

	// Should not panic with single step
	injectArtifacts(p)

	// Navigate step should not have injection (no execute step)
	assert.Empty(t, p.Steps[0].Memory.InjectArtifacts)
}

// TestInjectArtifacts_NoExecuteStep tests that injectArtifacts handles
// pipelines without an execute step
func TestInjectArtifacts_NoExecuteStep(t *testing.T) {
	p := &Pipeline{
		Steps: []Step{
			{ID: "step1", Persona: "persona1"},
			{ID: "step2", Persona: "persona2"},
		},
	}

	// Should not panic without execute step
	injectArtifacts(p)

	// Neither step should have injection
	assert.Empty(t, p.Steps[0].Memory.InjectArtifacts)
	assert.Empty(t, p.Steps[1].Memory.InjectArtifacts)
}

// TestDefaultNavigatorPersona tests the DefaultNavigatorPersona constant
func TestDefaultNavigatorPersona(t *testing.T) {
	assert.Equal(t, "navigator", DefaultNavigatorPersona)
}

// TestGenerateAdHocPipeline_ValidDAG tests that the generated pipeline
// has a valid DAG structure
func TestGenerateAdHocPipeline_ValidDAG(t *testing.T) {
	m := createAdhocTestManifest([]string{"navigator", "craftsman"})

	opts := AdHocOptions{
		Input:          "implement feature",
		ExecutePersona: "craftsman",
		Manifest:       m,
	}

	p, err := GenerateAdHocPipeline(opts)
	require.NoError(t, err)

	// Validate DAG using existing validator
	validator := &DAGValidator{}
	err = validator.ValidateDAG(p)
	assert.NoError(t, err)

	// Verify topological sort works
	sorted, err := validator.TopologicalSort(p)
	assert.NoError(t, err)
	assert.Len(t, sorted, 2)

	// Navigate should come before execute
	var navIndex, execIndex int
	for i, step := range sorted {
		if step.ID == "navigate" {
			navIndex = i
		}
		if step.ID == "execute" {
			execIndex = i
		}
	}
	assert.Less(t, navIndex, execIndex)
}

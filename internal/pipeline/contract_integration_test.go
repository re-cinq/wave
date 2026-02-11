package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Integration Tests: Pipeline Execution with Contract Validation
// ============================================================================
// These tests verify the full pipeline execution flow with contract validation,
// including:
// - JSON schema contract produces valid JSON
// - Schema injection into prompts
// - Contract validation of step outputs
// - Artifact handover between steps via inject_artifacts
// ============================================================================

// contractTestPromptCapturingAdapter captures prompts sent to the adapter for inspection
type contractTestPromptCapturingAdapter struct {
	*adapter.MockAdapter
	mu              sync.Mutex
	capturedPrompts []string
	capturedConfigs []adapter.AdapterRunConfig
}

func newContractTestPromptCapturingAdapter(opts ...adapter.MockOption) *contractTestPromptCapturingAdapter {
	return &contractTestPromptCapturingAdapter{
		MockAdapter:     adapter.NewMockAdapter(opts...),
		capturedPrompts: make([]string, 0),
		capturedConfigs: make([]adapter.AdapterRunConfig, 0),
	}
}

func (a *contractTestPromptCapturingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	a.capturedPrompts = append(a.capturedPrompts, cfg.Prompt)
	a.capturedConfigs = append(a.capturedConfigs, cfg)
	a.mu.Unlock()
	return a.MockAdapter.Run(ctx, cfg)
}

func (a *contractTestPromptCapturingAdapter) GetCapturedPrompts() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]string, len(a.capturedPrompts))
	copy(result, a.capturedPrompts)
	return result
}

func (a *contractTestPromptCapturingAdapter) GetCapturedConfigs() []adapter.AdapterRunConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]adapter.AdapterRunConfig, len(a.capturedConfigs))
	copy(result, a.capturedConfigs)
	return result
}

// contractTestArtifactWritingAdapter writes artifacts to workspace during execution
type contractTestArtifactWritingAdapter struct {
	artifacts map[string]string // stepID -> JSON content to write
	mu        sync.Mutex
}

func newContractTestArtifactWritingAdapter(artifacts map[string]string) *contractTestArtifactWritingAdapter {
	return &contractTestArtifactWritingAdapter{
		artifacts: artifacts,
	}
}

func (a *contractTestArtifactWritingAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Determine step ID from workspace path
	stepID := filepath.Base(cfg.WorkspacePath)

	// Write artifact if configured for this step
	if content, ok := a.artifacts[stepID]; ok {
		artifactPath := filepath.Join(cfg.WorkspacePath, "artifact.json")
		if err := os.WriteFile(artifactPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}

	// Return mock result with the content as ResultContent
	return &adapter.AdapterResult{
		ExitCode:      0,
		Stdout:        strings.NewReader(`{"status": "success"}`),
		TokensUsed:    1000,
		Artifacts:     []string{"artifact.json"},
		ResultContent: a.artifacts[stepID],
	}, nil
}

// createContractTestManifest creates a manifest with configurable workspace root
func createContractTestManifest(workspaceRoot string) *manifest.Manifest {
	return &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "test-project"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:     "claude",
				Temperature: 0.1,
			},
			"craftsman": {
				Adapter:     "claude",
				Temperature: 0.7,
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     workspaceRoot,
			DefaultTimeoutMin: 5,
		},
	}
}

// ============================================================================
// Test 1: JSON Schema Contract Produces Valid JSON
// ============================================================================

func TestContractIntegration_JSONSchemaProducesValidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a schema file
	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"version": {"type": "string"},
			"features": {
				"type": "array",
				"items": {"type": "string"}
			}
		},
		"required": ["name", "version"]
	}`
	schemaPath := filepath.Join(schemaDir, "test-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create valid artifact content that matches the schema
	validArtifact := `{
		"name": "test-project",
		"version": "1.0.0",
		"features": ["feature1", "feature2"]
	}`

	// Create adapter that writes valid artifact
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"step1": validArtifact,
	})

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "json-schema-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Generate project metadata"},
				OutputArtifacts: []ArtifactDef{
					{Name: "metadata", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline should complete successfully with valid JSON")

	// Verify contract_passed event was emitted
	events := collector.GetEvents()
	hasContractPassed := false
	for _, e := range events {
		if e.State == "contract_passed" {
			hasContractPassed = true
			break
		}
	}
	assert.True(t, hasContractPassed, "Should emit contract_passed event for valid JSON")

	// Verify the artifact file was created
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	artifactPath := filepath.Join(tmpDir, runtimeID, "step1", "artifact.json")
	_, err = os.Stat(artifactPath)
	assert.NoError(t, err, "Artifact file should exist")
}

func TestContractIntegration_JSONSchemaValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a schema file
	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"version": {"type": "string"}
		},
		"required": ["name", "version"]
	}`
	schemaPath := filepath.Join(schemaDir, "test-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create invalid artifact content (missing required field)
	invalidArtifact := `{
		"name": "test-project"
	}`

	// Create adapter that writes invalid artifact
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"step1": invalidArtifact,
	})

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "json-schema-fail-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Generate metadata"},
				OutputArtifacts: []ArtifactDef{
					{Name: "metadata", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   true, // This should cause failure
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.Error(t, err, "Pipeline should fail with invalid JSON")
	assert.Contains(t, err.Error(), "contract validation failed", "Error should mention contract validation")

	// Verify contract_failed event was emitted
	events := collector.GetEvents()
	hasContractFailed := false
	for _, e := range events {
		if e.State == "contract_failed" {
			hasContractFailed = true
			break
		}
	}
	assert.True(t, hasContractFailed, "Should emit contract_failed event for invalid JSON")
}

// ============================================================================
// Test 2: Schema Injection into Prompt
// ============================================================================

func TestContractIntegration_SchemaInjectedIntoPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a schema file
	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"result": {"type": "string"},
			"confidence": {"type": "number"}
		},
		"required": ["result"]
	}`
	schemaPath := filepath.Join(schemaDir, "analysis-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create capturing adapter to verify prompt contents
	capturingAdapter := newContractTestPromptCapturingAdapter(
		adapter.WithStdoutJSON(`{"result": "success", "confidence": 0.95}`),
		adapter.WithTokensUsed(1000),
	)

	// Also write the valid artifact
	validArtifact := `{"result": "success", "confidence": 0.95}`

	// Create executor with security configuration that allows temp directory
	securityConfig := security.DefaultSecurityConfig()
	securityConfig.PathValidation.ApprovedDirectories = []string{tmpDir}
	securityLogger := security.NewSecurityLogger(false)

	executor := NewDefaultPipelineExecutor(capturingAdapter)
	// Override security components to allow temp directory paths
	executor.securityConfig = securityConfig
	executor.pathValidator = security.NewPathValidator(*securityConfig, securityLogger)
	executor.inputSanitizer = security.NewInputSanitizer(*securityConfig, securityLogger)
	executor.securityLogger = securityLogger

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "schema-injection-test"},
		Steps: []Step{
			{
				ID:      "analyze",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Analyze the codebase"},
				OutputArtifacts: []ArtifactDef{
					{Name: "analysis", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   false, // Don't fail on validation for this test
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Write artifact manually since we're using capturing adapter
	stepWorkspace := filepath.Join(tmpDir, "schema-injection-test", "analyze")
	os.MkdirAll(stepWorkspace, 0755)
	os.WriteFile(filepath.Join(stepWorkspace, "artifact.json"), []byte(validArtifact), 0644)

	_ = executor.Execute(ctx, p, m, "test")

	// Verify schema was injected into prompt
	prompts := capturingAdapter.GetCapturedPrompts()
	require.Len(t, prompts, 1, "Should have captured one prompt")

	prompt := prompts[0]
	assert.Contains(t, prompt, "OUTPUT REQUIREMENTS", "Prompt should contain OUTPUT REQUIREMENTS section")
	assert.Contains(t, prompt, "artifact.json", "Prompt should mention artifact.json")
	assert.Contains(t, prompt, "result", "Prompt should contain schema field 'result'")
	assert.Contains(t, prompt, "confidence", "Prompt should contain schema field 'confidence'")
	assert.Contains(t, prompt, "JSON", "Prompt should mention JSON format")
}

func TestContractIntegration_InlineSchemaInjectedIntoPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	// Use inline schema instead of file
	inlineSchema := `{"type": "object", "properties": {"status": {"type": "string"}}, "required": ["status"]}`

	capturingAdapter := newContractTestPromptCapturingAdapter(
		adapter.WithStdoutJSON(`{"status": "complete"}`),
		adapter.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(capturingAdapter)
	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "inline-schema-test"},
		Steps: []Step{
			{
				ID:      "check",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Check status"},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:     "json_schema",
						Schema:   inlineSchema,
						MustPass: false,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Write artifact
	stepWorkspace := filepath.Join(tmpDir, "inline-schema-test", "check")
	os.MkdirAll(stepWorkspace, 0755)
	os.WriteFile(filepath.Join(stepWorkspace, "artifact.json"), []byte(`{"status": "complete"}`), 0644)

	_ = executor.Execute(ctx, p, m, "test")

	prompts := capturingAdapter.GetCapturedPrompts()
	require.Len(t, prompts, 1)

	prompt := prompts[0]
	assert.Contains(t, prompt, "status", "Inline schema should be injected into prompt")
	assert.Contains(t, prompt, "OUTPUT REQUIREMENTS", "Should have output requirements section")
}

// ============================================================================
// Test 3: Contract Validator Checks Output
// ============================================================================

func TestContractIntegration_ValidatorChecksOutput(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid object with all fields",
			schema: `{
				"type": "object",
				"properties": {
					"id": {"type": "integer"},
					"name": {"type": "string"},
					"active": {"type": "boolean"}
				},
				"required": ["id", "name"]
			}`,
			artifact:    `{"id": 1, "name": "test", "active": true}`,
			expectError: false,
		},
		{
			name: "valid object with optional field missing",
			schema: `{
				"type": "object",
				"properties": {
					"id": {"type": "integer"},
					"name": {"type": "string"},
					"active": {"type": "boolean"}
				},
				"required": ["id", "name"]
			}`,
			artifact:    `{"id": 2, "name": "test2"}`,
			expectError: false,
		},
		{
			name: "missing required field",
			schema: `{
				"type": "object",
				"properties": {
					"id": {"type": "integer"},
					"name": {"type": "string"}
				},
				"required": ["id", "name"]
			}`,
			artifact:      `{"id": 1}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name: "wrong type for field",
			schema: `{
				"type": "object",
				"properties": {
					"count": {"type": "integer"}
				},
				"required": ["count"]
			}`,
			artifact:      `{"count": "not a number"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name: "nested object validation",
			schema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"email": {"type": "string", "format": "email"}
						},
						"required": ["email"]
					}
				},
				"required": ["user"]
			}`,
			artifact:    `{"user": {"email": "test@example.com"}}`,
			expectError: false,
		},
		{
			name: "array validation",
			schema: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"items": {"type": "string"},
						"minItems": 1
					}
				},
				"required": ["items"]
			}`,
			artifact:    `{"items": ["a", "b", "c"]}`,
			expectError: false,
		},
		{
			name: "empty array when minItems required",
			schema: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"items": {"type": "string"},
						"minItems": 1
					}
				},
				"required": ["items"]
			}`,
			artifact:      `{"items": []}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create schema file
			schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
			require.NoError(t, os.MkdirAll(schemaDir, 0755))
			schemaPath := filepath.Join(schemaDir, "test-schema.json")
			require.NoError(t, os.WriteFile(schemaPath, []byte(tt.schema), 0644))

			// Create adapter that writes the artifact
			mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
				"validate-step": tt.artifact,
			})

			collector := newTestEventCollector()
			executor := NewDefaultPipelineExecutor(mockAdapter,
				WithEmitter(collector),
			)

			m := createContractTestManifest(tmpDir)

			p := &Pipeline{
				Metadata: PipelineMetadata{Name: "validation-test"},
				Steps: []Step{
					{
						ID:      "validate-step",
						Persona: "navigator",
						Exec:    ExecConfig{Source: "Generate output"},
						OutputArtifacts: []ArtifactDef{
							{Name: "output", Path: "artifact.json"},
						},
						Handover: HandoverConfig{
							Contract: ContractConfig{
								Type:       "json_schema",
								SchemaPath: schemaPath,
								MustPass:   true,
							},
						},
					},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := executor.Execute(ctx, p, m, "test")

			if tt.expectError {
				require.Error(t, err, "Expected validation to fail")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err, "Expected validation to pass")
			}
		})
	}
}

// ============================================================================
// Test 4: Artifact Handover Between Steps with inject_artifacts
// ============================================================================

func TestContractIntegration_ArtifactHandoverBetweenSteps(t *testing.T) {
	tmpDir := t.TempDir()

	// Create schema files for both steps
	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	step1Schema := `{
		"type": "object",
		"properties": {
			"analysis": {"type": "string"},
			"files": {"type": "array", "items": {"type": "string"}}
		},
		"required": ["analysis", "files"]
	}`
	step1SchemaPath := filepath.Join(schemaDir, "step1-schema.json")
	require.NoError(t, os.WriteFile(step1SchemaPath, []byte(step1Schema), 0644))

	step2Schema := `{
		"type": "object",
		"properties": {
			"implementation": {"type": "string"},
			"source_analysis": {"type": "string"}
		},
		"required": ["implementation"]
	}`
	step2SchemaPath := filepath.Join(schemaDir, "step2-schema.json")
	require.NoError(t, os.WriteFile(step2SchemaPath, []byte(step2Schema), 0644))

	// Artifacts for each step
	step1Artifact := `{
		"analysis": "Found 3 key patterns",
		"files": ["main.go", "handler.go", "types.go"]
	}`
	step2Artifact := `{
		"implementation": "Added new feature",
		"source_analysis": "From step1"
	}`

	// Create adapter that writes appropriate artifacts per step
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"analyze":   step1Artifact,
		"implement": step2Artifact,
	})

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "handover-test"},
		Steps: []Step{
			{
				ID:      "analyze",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Analyze the codebase"},
				OutputArtifacts: []ArtifactDef{
					{Name: "analysis-result", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: step1SchemaPath,
						MustPass:   true,
					},
				},
			},
			{
				ID:           "implement",
				Persona:      "craftsman",
				Dependencies: []string{"analyze"},
				Memory: MemoryConfig{
					Strategy: "fresh",
					InjectArtifacts: []ArtifactRef{
						{
							Step:     "analyze",
							Artifact: "analysis-result",
							As:       "analysis.json",
						},
					},
				},
				Exec: ExecConfig{Source: "Implement based on analysis"},
				OutputArtifacts: []ArtifactDef{
					{Name: "impl-result", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: step2SchemaPath,
						MustPass:   true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline should complete with artifact handover")

	// Verify both steps completed
	events := collector.GetEvents()
	completedSteps := make(map[string]bool)
	for _, e := range events {
		if e.State == "completed" && e.StepID != "" {
			completedSteps[e.StepID] = true
		}
	}
	assert.True(t, completedSteps["analyze"], "Step 'analyze' should complete")
	assert.True(t, completedSteps["implement"], "Step 'implement' should complete")

	// Verify artifacts directory was created in step2's workspace
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	step2ArtifactsDir := filepath.Join(tmpDir, runtimeID, "implement", "artifacts")
	_, err = os.Stat(step2ArtifactsDir)
	assert.NoError(t, err, "Artifacts directory should exist in step2's workspace")

	// Verify the injected artifact exists
	injectedArtifactPath := filepath.Join(step2ArtifactsDir, "analysis.json")
	_, err = os.Stat(injectedArtifactPath)
	assert.NoError(t, err, "Injected artifact should exist")

	// Verify the injected artifact content
	content, err := os.ReadFile(injectedArtifactPath)
	require.NoError(t, err)

	var injectedData map[string]interface{}
	require.NoError(t, json.Unmarshal(content, &injectedData))
	assert.Equal(t, "Found 3 key patterns", injectedData["analysis"])
}

func TestContractIntegration_MultiStepArtifactChain(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple schema for all steps
	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{"type": "object", "properties": {"data": {"type": "string"}}, "required": ["data"]}`
	schemaPath := filepath.Join(schemaDir, "simple-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create adapter with artifacts for each step
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"step-a": `{"data": "from-step-a"}`,
		"step-b": `{"data": "from-step-b"}`,
		"step-c": `{"data": "from-step-c"}`,
	})

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	// Create a chain: A -> B -> C, each passing artifacts
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "chain-test"},
		Steps: []Step{
			{
				ID:      "step-a",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Step A"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-a", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   true,
					},
				},
			},
			{
				ID:           "step-b",
				Persona:      "navigator",
				Dependencies: []string{"step-a"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-a", Artifact: "output-a", As: "input-from-a.json"},
					},
				},
				Exec: ExecConfig{Source: "Step B"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-b", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   true,
					},
				},
			},
			{
				ID:           "step-c",
				Persona:      "navigator",
				Dependencies: []string{"step-b"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-a", Artifact: "output-a", As: "from-a.json"},
						{Step: "step-b", Artifact: "output-b", As: "from-b.json"},
					},
				},
				Exec: ExecConfig{Source: "Step C"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-c", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline chain should complete successfully")

	// Verify all steps completed
	order := collector.GetStepExecutionOrder()
	require.Len(t, order, 3, "All 3 steps should have executed")

	// Verify step C has artifacts from both A and B
	runtimeID := collector.GetPipelineID()
	require.NotEmpty(t, runtimeID, "should have a pipeline ID from events")
	stepCArtifactsDir := filepath.Join(tmpDir, runtimeID, "step-c", "artifacts")
	_, err = os.Stat(filepath.Join(stepCArtifactsDir, "from-a.json"))
	assert.NoError(t, err, "Step C should have artifact from step A")
	_, err = os.Stat(filepath.Join(stepCArtifactsDir, "from-b.json"))
	assert.NoError(t, err, "Step C should have artifact from step B")
}

// ============================================================================
// Test 5: Contract Soft Failure (must_pass: false)
// ============================================================================

func TestContractIntegration_SoftFailureContinues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create schema
	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{"type": "object", "properties": {"required_field": {"type": "string"}}, "required": ["required_field"]}`
	schemaPath := filepath.Join(schemaDir, "soft-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create invalid artifact (missing required field)
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"soft-step": `{"other_field": "value"}`,
	})

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "soft-fail-test"},
		Steps: []Step{
			{
				ID:      "soft-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Generate output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   false, // Soft failure - should continue
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline should continue despite soft contract failure")

	// Verify soft failure event was emitted
	events := collector.GetEvents()
	hasSoftFailure := false
	for _, e := range events {
		if e.State == "contract_soft_failure" {
			hasSoftFailure = true
			break
		}
	}
	assert.True(t, hasSoftFailure, "Should emit contract_soft_failure event")
}

// ============================================================================
// Test 6: Input Template Replacement
// ============================================================================

func TestContractIntegration_InputTemplateReplacement(t *testing.T) {
	tmpDir := t.TempDir()

	capturingAdapter := newContractTestPromptCapturingAdapter(
		adapter.WithStdoutJSON(`{"result": "done"}`),
		adapter.WithTokensUsed(500),
	)

	executor := NewDefaultPipelineExecutor(capturingAdapter)
	m := createContractTestManifest(tmpDir)

	testInput := "build feature XYZ"

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "input-template-test"},
		Steps: []Step{
			{
				ID:      "process",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Process the following request: {{ input }}"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = executor.Execute(ctx, p, m, testInput)

	prompts := capturingAdapter.GetCapturedPrompts()
	require.Len(t, prompts, 1)

	prompt := prompts[0]
	assert.Contains(t, prompt, testInput, "Input should be replaced in prompt")
	assert.NotContains(t, prompt, "{{ input }}", "Template variable should be replaced")
}

// ============================================================================
// Test 7: Contract Retry Mechanism
// ============================================================================

func TestContractIntegration_RetryOnContractFailure(t *testing.T) {
	tmpDir := t.TempDir()

	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{"type": "object", "properties": {"status": {"type": "string"}}, "required": ["status"]}`
	schemaPath := filepath.Join(schemaDir, "retry-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create an adapter that fails first 2 times, succeeds on 3rd
	var attemptCount int32
	retryAdapter := &contractTestRetryingArtifactAdapter{
		attempts:     &attemptCount,
		failUntil:    2,
		invalidJSON:  `{"wrong": 123}`,
		validJSON:    `{"status": "success"}`,
		workspaceDir: tmpDir,
	}

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(retryAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "retry-contract-test"},
		Steps: []Step{
			{
				ID:      "retry-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Generate status"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					MaxRetries: 3,
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						MustPass:   true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline should succeed after retries")

	// Verify retry events were emitted
	events := collector.GetEvents()
	retryCount := 0
	for _, e := range events {
		if e.State == "retrying" {
			retryCount++
		}
	}
	assert.Equal(t, 2, retryCount, "Should have 2 retry events")
}

// contractTestRetryingArtifactAdapter writes invalid artifacts first, then valid ones
type contractTestRetryingArtifactAdapter struct {
	attempts     *int32
	failUntil    int32
	invalidJSON  string
	validJSON    string
	workspaceDir string
	mu           sync.Mutex
}

func (a *contractTestRetryingArtifactAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	*a.attempts++
	attempt := *a.attempts

	var content string
	if attempt <= a.failUntil {
		content = a.invalidJSON
	} else {
		content = a.validJSON
	}

	// Write artifact
	artifactPath := filepath.Join(cfg.WorkspacePath, "artifact.json")
	if err := os.WriteFile(artifactPath, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &adapter.AdapterResult{
		ExitCode:      0,
		Stdout:        strings.NewReader(`{"status": "executed"}`),
		TokensUsed:    1000,
		Artifacts:     []string{"artifact.json"},
		ResultContent: content,
	}, nil
}

// ============================================================================
// Test 8: Persona-Contract Separation
// ============================================================================

func TestContractIntegration_PersonaContractSeparation(t *testing.T) {
	// This test verifies that personas and contracts are properly separated:
	// - Persona defines WHO does the work (permissions, adapter, temperature)
	// - Contract defines WHAT the output should be (schema validation)

	tmpDir := t.TempDir()

	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	// Different schemas for different contracts
	navigatorSchema := `{"type": "object", "properties": {"analysis": {"type": "string"}}, "required": ["analysis"]}`
	craftsmanSchema := `{"type": "object", "properties": {"code": {"type": "string"}}, "required": ["code"]}`

	navigatorSchemaPath := filepath.Join(schemaDir, "navigator-contract.json")
	craftsmanSchemaPath := filepath.Join(schemaDir, "craftsman-contract.json")
	require.NoError(t, os.WriteFile(navigatorSchemaPath, []byte(navigatorSchema), 0644))
	require.NoError(t, os.WriteFile(craftsmanSchemaPath, []byte(craftsmanSchema), 0644))

	// Create adapter that writes persona-specific artifacts
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"analyze":   `{"analysis": "Found patterns X and Y"}`,
		"implement": `{"code": "func main() {}"}`,
	})

	capturingCollector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(capturingCollector),
	)

	m := &manifest.Manifest{
		Metadata: manifest.Metadata{Name: "persona-contract-test"},
		Adapters: map[string]manifest.Adapter{
			"claude": {Binary: "claude", Mode: "headless"},
		},
		Personas: map[string]manifest.Persona{
			"navigator": {
				Adapter:     "claude",
				Temperature: 0.1, // Low temperature for analysis
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Grep", "Glob"},
					Deny:         []string{"Write", "Edit"},
				},
			},
			"craftsman": {
				Adapter:     "claude",
				Temperature: 0.7, // Higher temperature for creative coding
				Permissions: manifest.Permissions{
					AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
				},
			},
		},
		Runtime: manifest.Runtime{
			WorkspaceRoot:     tmpDir,
			DefaultTimeoutMin: 5,
		},
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "persona-test"},
		Steps: []Step{
			{
				ID:      "analyze",
				Persona: "navigator", // Navigator persona
				Exec:    ExecConfig{Source: "Analyze codebase"},
				OutputArtifacts: []ArtifactDef{
					{Name: "analysis", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: navigatorSchemaPath, // Navigator contract
						MustPass:   true,
					},
				},
			},
			{
				ID:           "implement",
				Persona:      "craftsman", // Craftsman persona
				Dependencies: []string{"analyze"},
				Exec:         ExecConfig{Source: "Implement feature"},
				OutputArtifacts: []ArtifactDef{
					{Name: "implementation", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: craftsmanSchemaPath, // Craftsman contract
						MustPass:   true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline with different personas and contracts should succeed")

	// Verify both contracts passed
	events := capturingCollector.GetEvents()
	contractPassedCount := 0
	for _, e := range events {
		if e.State == "contract_passed" {
			contractPassedCount++
		}
	}
	assert.Equal(t, 2, contractPassedCount, "Both contracts should pass")

	// Verify persona was recorded in events
	personas := make(map[string]bool)
	for _, e := range events {
		if e.Persona != "" {
			personas[e.Persona] = true
		}
	}
	assert.True(t, personas["navigator"], "Navigator persona should be recorded")
	assert.True(t, personas["craftsman"], "Craftsman persona should be recorded")
}

// ============================================================================
// Test 9: Custom Source Path for Contract Validation
// ============================================================================

func TestContractIntegration_CustomSourcePath(t *testing.T) {
	tmpDir := t.TempDir()

	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{"type": "object", "properties": {"result": {"type": "string"}}, "required": ["result"]}`
	schemaPath := filepath.Join(schemaDir, "custom-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create adapter that writes to custom path
	customAdapter := &contractTestCustomPathArtifactAdapter{
		content:  `{"result": "custom-output"}`,
		filename: "custom-output.json",
	}

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(customAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "custom-source-test"},
		Steps: []Step{
			{
				ID:      "custom-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Generate to custom path"},
				OutputArtifacts: []ArtifactDef{
					{Name: "custom-output", Path: "custom-output.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:       "json_schema",
						SchemaPath: schemaPath,
						Source:     "custom-output.json", // Custom source path
						MustPass:   true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Pipeline should validate custom source path")
}

type contractTestCustomPathArtifactAdapter struct {
	content  string
	filename string
}

func (a *contractTestCustomPathArtifactAdapter) Run(ctx context.Context, cfg adapter.AdapterRunConfig) (*adapter.AdapterResult, error) {
	// Write to custom filename
	artifactPath := filepath.Join(cfg.WorkspacePath, a.filename)
	if err := os.WriteFile(artifactPath, []byte(a.content), 0644); err != nil {
		return nil, err
	}

	return &adapter.AdapterResult{
		ExitCode:      0,
		Stdout:        strings.NewReader(`{"status": "success"}`),
		TokensUsed:    500,
		Artifacts:     []string{a.filename},
		ResultContent: a.content,
	}, nil
}

// ============================================================================
// Test 10: Complex Diamond Dependency with Contracts
// ============================================================================

func TestContractIntegration_DiamondDependencyWithContracts(t *testing.T) {
	// Test diamond dependency pattern:
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	// All steps have contracts that must pass

	tmpDir := t.TempDir()

	schemaDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schema := `{"type": "object", "properties": {"step": {"type": "string"}}, "required": ["step"]}`
	schemaPath := filepath.Join(schemaDir, "diamond-schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schema), 0644))

	// Create adapter with artifacts for each step
	mockAdapter := newContractTestArtifactWritingAdapter(map[string]string{
		"step-a": `{"step": "A"}`,
		"step-b": `{"step": "B"}`,
		"step-c": `{"step": "C"}`,
		"step-d": `{"step": "D"}`,
	})

	collector := newTestEventCollector()
	executor := NewDefaultPipelineExecutor(mockAdapter,
		WithEmitter(collector),
	)

	m := createContractTestManifest(tmpDir)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "diamond-test"},
		Steps: []Step{
			{
				ID:      "step-a",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "Step A"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-a", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{Type: "json_schema", SchemaPath: schemaPath, MustPass: true},
				},
			},
			{
				ID:           "step-b",
				Persona:      "navigator",
				Dependencies: []string{"step-a"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-a", Artifact: "output-a", As: "from-a.json"},
					},
				},
				Exec: ExecConfig{Source: "Step B"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-b", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{Type: "json_schema", SchemaPath: schemaPath, MustPass: true},
				},
			},
			{
				ID:           "step-c",
				Persona:      "navigator",
				Dependencies: []string{"step-a"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-a", Artifact: "output-a", As: "from-a.json"},
					},
				},
				Exec: ExecConfig{Source: "Step C"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-c", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{Type: "json_schema", SchemaPath: schemaPath, MustPass: true},
				},
			},
			{
				ID:           "step-d",
				Persona:      "navigator",
				Dependencies: []string{"step-b", "step-c"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-b", Artifact: "output-b", As: "from-b.json"},
						{Step: "step-c", Artifact: "output-c", As: "from-c.json"},
					},
				},
				Exec: ExecConfig{Source: "Step D"},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-d", Path: "artifact.json"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{Type: "json_schema", SchemaPath: schemaPath, MustPass: true},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Execute(ctx, p, m, "test")
	require.NoError(t, err, "Diamond dependency pipeline should complete")

	// Verify execution order respects dependencies
	order := collector.GetStepExecutionOrder()
	require.Len(t, order, 4, "All 4 steps should have executed")

	posA := contractTestIndexOf(order, "step-a")
	posB := contractTestIndexOf(order, "step-b")
	posC := contractTestIndexOf(order, "step-c")
	posD := contractTestIndexOf(order, "step-d")

	assert.True(t, posA < posB, "A must execute before B")
	assert.True(t, posA < posC, "A must execute before C")
	assert.True(t, posB < posD, "B must execute before D")
	assert.True(t, posC < posD, "C must execute before D")

	// Verify all 4 contracts passed
	events := collector.GetEvents()
	contractPassedCount := 0
	for _, e := range events {
		if e.State == "contract_passed" {
			contractPassedCount++
		}
	}
	assert.Equal(t, 4, contractPassedCount, "All 4 contracts should pass")
}

// contractTestIndexOf is a helper function to find index in slice (named to avoid conflicts)
func contractTestIndexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/testutil"
	"github.com/recinq/wave/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Pipeline Failure Modes Test Suite
// ============================================================================
// These tests verify that the pipeline executor correctly handles and reports
// failures across 7 distinct failure modes:
//   1. Contract schema mismatch (output doesn't match JSON schema)
//   2. Step timeout (context deadline exceeded)
//   3. Missing artifact (required inject_artifact not produced)
//   4. Malformed artifact (invalid JSON against strict schema)
//   5. Workspace corruption (workspace manager returns error)
//   6. Non-zero exit code (adapter exits non-zero without error)
//   7. Adapter error (adapter returns error directly)
// ============================================================================

// failureModeTestContext bundles the common objects for failure mode tests.
type failureModeTestContext struct {
	executor  *DefaultPipelineExecutor
	manifest  *manifest.Manifest
	collector *testutil.EventCollector
	tmpDir    string
}

// setupFailureModeTest creates a test executor with security configured for tmpDir.
func setupFailureModeTest(t *testing.T, runner adapter.AdapterRunner, opts ...ExecutorOption) *failureModeTestContext {
	t.Helper()

	tmpDir := t.TempDir()
	m := testutil.CreateTestManifest(tmpDir)
	collector := testutil.NewEventCollector()

	allOpts := append([]ExecutorOption{WithEmitter(collector)}, opts...)
	executor := NewDefaultPipelineExecutor(runner, allOpts...)

	// Configure security to allow the temp directory
	securityConfig := security.DefaultSecurityConfig()
	securityConfig.PathValidation.ApprovedDirectories = []string{tmpDir}
	securityLogger := security.NewSecurityLogger(false)
	executor.securityConfig = securityConfig
	executor.pathValidator = security.NewPathValidator(*securityConfig, securityLogger)
	executor.inputSanitizer = security.NewInputSanitizer(*securityConfig, securityLogger)
	executor.securityLogger = securityLogger

	return &failureModeTestContext{
		executor:  executor,
		manifest:  m,
		collector: collector,
		tmpDir:    tmpDir,
	}
}

// hasStepEventWithState checks whether a specific step has an event with the given state.
func hasStepEventWithState(collector *testutil.EventCollector, stepID, state string) bool {
	events := collector.GetEventsByStep(stepID)
	for _, e := range events {
		if e.State == state {
			return true
		}
	}
	return false
}

// ============================================================================
// Test 1: Contract Schema Mismatch
// ============================================================================

func TestFailureMode_ContractSchemaMismatch(t *testing.T) {
	// Adapter returns JSON that doesn't match the required schema.
	// Schema requires "name" (string) and "version" (string).
	// Adapter returns {"bad": true} which lacks required fields.
	badOutput := `{"bad": true}`
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(badOutput),
		adapter.WithTokensUsed(500),
	)

	tc := setupFailureModeTest(t, mockAdapter)

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"version": {"type": "string"}
		},
		"required": ["name", "version"]
	}`

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "contract-mismatch-test"},
		Steps: []Step{
			{
				ID:      "validate-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "result", Path: ".agents/artifact.json", Type: "json", Source: "stdout"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:     "json_schema",
						Schema:   schema,
						MustPass: true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := tc.executor.Execute(ctx, p, tc.manifest, "test")
	require.Error(t, err, "pipeline should fail when output doesn't match schema")
	assert.Contains(t, err.Error(), "contract validation failed",
		"error should indicate contract validation failure")

	// Verify contract_failed event was emitted for this step
	assert.True(t, hasStepEventWithState(tc.collector, "validate-step", "contract_failed"),
		"should emit contract_failed event for the step")

	// Verify no completed event for this step
	assert.False(t, hasStepEventWithState(tc.collector, "validate-step", "completed"),
		"should not emit completed event for a step that failed contract validation")
}

// ============================================================================
// Test 2: Step Timeout
// ============================================================================

func TestFailureMode_StepTimeout(t *testing.T) {
	// Adapter simulates a long-running step (5 seconds).
	// Context has a short timeout (200ms) so the step should be cancelled.
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithSimulatedDelay(5*time.Second),
		adapter.WithTokensUsed(500),
	)

	tc := setupFailureModeTest(t, mockAdapter)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "timeout-test"},
		Steps: []Step{
			{
				ID:      "slow-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something slow"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := tc.executor.Execute(ctx, p, tc.manifest, "test")
	require.Error(t, err, "pipeline should fail on timeout")

	// The error chain should contain context.DeadlineExceeded
	assert.True(t, errors.Is(err, context.DeadlineExceeded),
		"error should wrap context.DeadlineExceeded, got: %v", err)
}

// ============================================================================
// Test 3: Missing Artifact
// ============================================================================

func TestFailureMode_MissingArtifact(t *testing.T) {
	// Two-step pipeline: step2 injects an artifact from a step that never
	// executed ("nonexistent-step"). The DAG validator only checks
	// Dependencies, not inject_artifacts step references, so the pipeline
	// starts but fails at artifact injection time.
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)

	tc := setupFailureModeTest(t, mockAdapter)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "missing-artifact-test"},
		Steps: []Step{
			{
				ID:      "step1",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce output"},
			},
			{
				ID:           "step2",
				Persona:      "navigator",
				Dependencies: []string{"step1"},
				Exec:         ExecConfig{Source: "consume artifact"},
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{
							Step:     "nonexistent-step",
							Artifact: "analysis",
							As:       "analysis",
							// Optional defaults to false, so this is a required artifact
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := tc.executor.Execute(ctx, p, tc.manifest, "test")
	require.Error(t, err, "pipeline should fail when a required artifact is missing")
	assert.Contains(t, err.Error(), "required artifact",
		"error should mention 'required artifact'")
	assert.Contains(t, err.Error(), "not found",
		"error should mention 'not found'")
}

// ============================================================================
// Test 4: Malformed Artifact
// ============================================================================

func TestFailureMode_MalformedArtifact(t *testing.T) {
	// Step produces malformed JSON (truncated) as stdout.
	// Contract validation with json_schema should reject it.
	malformedJSON := `{"name": "test", "version": `
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(malformedJSON),
		adapter.WithTokensUsed(500),
	)

	tc := setupFailureModeTest(t, mockAdapter)

	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"version": {"type": "string"}
		},
		"required": ["name", "version"]
	}`

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "malformed-artifact-test"},
		Steps: []Step{
			{
				ID:      "malformed-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "produce malformed output"},
				OutputArtifacts: []ArtifactDef{
					{Name: "result", Path: ".agents/artifact.json", Type: "json", Source: "stdout"},
				},
				Handover: HandoverConfig{
					Contract: ContractConfig{
						Type:     "json_schema",
						Schema:   schema,
						MustPass: true,
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := tc.executor.Execute(ctx, p, tc.manifest, "test")
	require.Error(t, err, "pipeline should fail when artifact JSON is malformed")
	assert.Contains(t, err.Error(), "contract validation failed",
		"error should indicate contract validation failure")

	// Should have contract_failed event
	assert.True(t, hasStepEventWithState(tc.collector, "malformed-step", "contract_failed"),
		"should emit contract_failed event for malformed artifact")
}

// ============================================================================
// Test 5: Workspace Corruption
// ============================================================================

// failingWorkspaceManager always returns an error on Create.
type failingWorkspaceManager struct{}

func (f *failingWorkspaceManager) Create(cfg workspace.WorkspaceConfig, templateVars map[string]string) (string, error) {
	return "", fmt.Errorf("workspace creation failed: disk full")
}

func (f *failingWorkspaceManager) InjectArtifacts(workspacePath string, refs []workspace.ArtifactRef, resolvedPaths map[string]string) error {
	return nil
}

func (f *failingWorkspaceManager) CleanAll(root string) error {
	return nil
}

func TestFailureMode_WorkspaceCorruption(t *testing.T) {
	// Workspace manager returns an error on Create().
	// The step uses mount-based workspace to trigger wsManager.Create().
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithStdoutJSON(`{"status": "success"}`),
		adapter.WithTokensUsed(500),
	)

	tc := setupFailureModeTest(t, mockAdapter,
		WithWorkspaceManager(&failingWorkspaceManager{}),
	)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "workspace-corruption-test"},
		Steps: []Step{
			{
				ID:      "ws-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do work"},
				Workspace: WorkspaceConfig{
					Mount: []Mount{
						{Source: ".", Target: "project", Mode: "readonly"},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := tc.executor.Execute(ctx, p, tc.manifest, "test")
	require.Error(t, err, "pipeline should fail when workspace creation fails")
	assert.Contains(t, err.Error(), "workspace",
		"error should mention workspace")
}

// ============================================================================
// Test 6: Non-Zero Exit Code
// ============================================================================

func TestFailureMode_NonZeroExitCode(t *testing.T) {
	t.Run("with_strict_contract_fails_on_schema_mismatch", func(t *testing.T) {
		// Adapter returns exit code 1 but no error.
		// Output doesn't match schema, and contract has MustPass: true.
		// Pipeline should fail because contract validation fails (not because of exit code).
		mockAdapter := adapter.NewMockAdapter(
			adapter.WithExitCode(1),
			adapter.WithStdoutJSON(`{"bad": true}`),
			adapter.WithTokensUsed(500),
		)

		tc := setupFailureModeTest(t, mockAdapter)

		schema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"required": ["name"]
		}`

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "exit-code-contract-test"},
			Steps: []Step{
				{
					ID:      "exit-step",
					Persona: "navigator",
					Exec:    ExecConfig{Source: "produce output"},
					OutputArtifacts: []ArtifactDef{
						{Name: "result", Path: ".agents/artifact.json", Type: "json", Source: "stdout"},
					},
					Handover: HandoverConfig{
						Contract: ContractConfig{
							Type:     "json_schema",
							Schema:   schema,
							MustPass: true,
						},
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := tc.executor.Execute(ctx, p, tc.manifest, "test")
		require.Error(t, err, "pipeline should fail when contract validation fails")
		assert.Contains(t, err.Error(), "contract validation failed",
			"error should be about contract validation, not exit code")
	})

	t.Run("without_contract_completes_successfully", func(t *testing.T) {
		// Adapter returns exit code 1 but no error, and no contract is configured.
		// Pipeline should still complete since non-zero exit code alone is not fatal.
		mockAdapter := adapter.NewMockAdapter(
			adapter.WithExitCode(1),
			adapter.WithStdoutJSON(`{"status": "done"}`),
			adapter.WithTokensUsed(500),
		)

		tc := setupFailureModeTest(t, mockAdapter)

		p := &Pipeline{
			Metadata: PipelineMetadata{Name: "exit-code-no-contract-test"},
			Steps: []Step{
				{
					ID:      "exit-step",
					Persona: "navigator",
					Exec:    ExecConfig{Source: "produce output"},
					// No contract — pipeline should not fail on exit code alone
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := tc.executor.Execute(ctx, p, tc.manifest, "test")
		require.NoError(t, err, "pipeline should complete when exit code is non-zero but no contract is configured")

		// Verify warning event was emitted for the non-zero exit code
		hasWarning := false
		for _, evt := range tc.collector.GetEvents() {
			if evt.State == "warning" && strings.Contains(evt.Message, "exited with code") {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning,
			"should emit warning event for non-zero exit code")

		// Verify the step completed
		assert.True(t, hasStepEventWithState(tc.collector, "exit-step", "completed"),
			"step should have completed event despite non-zero exit code")
	})
}

// ============================================================================
// Test 7: Adapter Error
// ============================================================================

func TestFailureMode_AdapterError(t *testing.T) {
	// Adapter returns a direct error via WithFailure().
	// Pipeline should propagate the error and emit a failed event.
	adapterErr := fmt.Errorf("adapter crashed: out of memory")
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithFailure(adapterErr),
	)

	tc := setupFailureModeTest(t, mockAdapter)

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "adapter-error-test"},
		Steps: []Step{
			{
				ID:      "crash-step",
				Persona: "navigator",
				Exec:    ExecConfig{Source: "do something"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := tc.executor.Execute(ctx, p, tc.manifest, "test")
	require.Error(t, err, "pipeline should fail when adapter returns error")

	// The error should wrap the original adapter error
	assert.True(t, errors.Is(err, adapterErr),
		"error should wrap the original adapter error, got: %v", err)

	// Verify failed event was emitted
	assert.True(t, tc.collector.HasEventWithState("failed"),
		"should emit 'failed' event when adapter returns error")

	// Verify no completed event for the step
	assert.False(t, hasStepEventWithState(tc.collector, "crash-step", "completed"),
		"should not have completed event for a step where adapter crashed")
}

package tests_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/recovery"
)

// TestPreflightRecovery_MissingSkill verifies that when a pipeline fails due to a missing skill,
// the error message includes a "wave skill install <skill>" recovery hint.
func TestPreflightRecovery_MissingSkill(t *testing.T) {
	// Create a test pipeline that requires a skill that doesn't exist
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{
			Name:        "test-skill-pipeline",
			Description: "Test pipeline for skill preflight",
		},
		Requires: &pipeline.Requires{
			Skills: []string{"nonexistent-skill-xyz"},
		},
		Steps: []pipeline.Step{
			{
				ID:      "step1",
				Persona: "navigator",
			},
		},
	}

	// Create manifest without the required skill configured
	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			WorkspaceRoot:        ".wave/workspaces",
			PipelineIDHashLength: 8,
		},
		Skills: map[string]manifest.SkillConfig{},
		Adapters: map[string]manifest.Adapter{
			"mock": {},
		},
	}

	// Create executor with mock adapter
	mockAdapter := adapter.NewMockAdapter()
	executor := pipeline.NewDefaultPipelineExecutor(
		mockAdapter,
		pipeline.WithDebug(false),
	)

	// Execute the pipeline - should fail at preflight
	ctx := context.Background()
	err := executor.Execute(ctx, p, m, "test input")

	// Verify error occurred
	if err == nil {
		t.Fatal("expected preflight check to fail for missing skill")
	}

	// Verify error is a SkillError
	var skillErr *preflight.SkillError
	if !errors.As(err, &skillErr) {
		t.Fatalf("expected SkillError, got %T: %v", err, err)
	}

	// Verify the missing skill is correctly identified
	if len(skillErr.MissingSkills) != 1 {
		t.Errorf("expected 1 missing skill, got %d", len(skillErr.MissingSkills))
	}
	if skillErr.MissingSkills[0] != "nonexistent-skill-xyz" {
		t.Errorf("expected missing skill 'nonexistent-skill-xyz', got %v", skillErr.MissingSkills)
	}

	// Verify error classification
	errClass := recovery.ClassifyError(err)
	if errClass != recovery.ClassPreflight {
		t.Errorf("expected error class ClassPreflight, got %v", errClass)
	}

	// Build recovery block as the run command would
	meta := &recovery.PreflightMetadata{
		MissingSkills: skillErr.MissingSkills,
	}
	block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
		PipelineName:  p.Metadata.Name,
		Input:         "test input",
		RunID:         "test-run-12345678",
		WorkspaceRoot: ".wave/workspaces",
		ErrClass:      errClass,
		PreflightMeta: meta,
	})

	// Verify recovery hints include skill install command
	foundSkillHint := false
	for _, hint := range block.Hints {
		if strings.Contains(hint.Command, "wave.yaml skills.") &&
			strings.Contains(hint.Command, "nonexistent-skill-xyz") {
			foundSkillHint = true
			if hint.Label != "Install missing skill" {
				t.Errorf("expected hint label 'Install missing skill', got %q", hint.Label)
			}
			break
		}
	}
	if !foundSkillHint {
		t.Errorf("expected recovery hints to include 'wave skill install nonexistent-skill-xyz', got: %+v", block.Hints)
	}

	// Verify workspace path has no double slashes
	if strings.Contains(block.WorkspacePath, "//") {
		t.Errorf("workspace path contains double slashes: %s", block.WorkspacePath)
	}

	// Verify no resume hint for preflight failures (step hasn't started)
	for _, hint := range block.Hints {
		if hint.Type == recovery.HintResume {
			t.Errorf("preflight failures should not include resume hints, found: %+v", hint)
		}
	}

	// Verify error message doesn't have redundant "preflight check failed"
	errMsg := err.Error()
	count := strings.Count(errMsg, "preflight check failed")
	if count > 1 {
		t.Errorf("error message contains redundant 'preflight check failed' (%d times): %s", count, errMsg)
	}
}

// TestPreflightRecovery_MissingTool verifies that when a pipeline fails due to a missing tool,
// the error message includes helpful tool installation guidance.
func TestPreflightRecovery_MissingTool(t *testing.T) {
	// Create a test pipeline that requires a tool that doesn't exist
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{
			Name:        "test-tool-pipeline",
			Description: "Test pipeline for tool preflight",
		},
		Requires: &pipeline.Requires{
			Tools: []string{"nonexistent-cli-tool-xyz-999"},
		},
		Steps: []pipeline.Step{
			{
				ID:      "step1",
				Persona: "navigator",
			},
		},
	}

	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			WorkspaceRoot:         ".wave/workspaces",
			PipelineIDHashLength:  8,
		},
		Skills: map[string]manifest.SkillConfig{},
		Adapters: map[string]manifest.Adapter{
			"mock": {},
		},
	}

	mockAdapter := adapter.NewMockAdapter()
	executor := pipeline.NewDefaultPipelineExecutor(
		mockAdapter,
		pipeline.WithDebug(false),
	)

	ctx := context.Background()
	err := executor.Execute(ctx, p, m, "test input")

	// Verify error occurred
	if err == nil {
		t.Fatal("expected preflight check to fail for missing tool")
	}

	// Verify error is a ToolError
	var toolErr *preflight.ToolError
	if !errors.As(err, &toolErr) {
		t.Fatalf("expected ToolError, got %T: %v", err, err)
	}

	// Verify the missing tool is correctly identified
	if len(toolErr.MissingTools) != 1 {
		t.Errorf("expected 1 missing tool, got %d", len(toolErr.MissingTools))
	}
	if toolErr.MissingTools[0] != "nonexistent-cli-tool-xyz-999" {
		t.Errorf("expected missing tool 'nonexistent-cli-tool-xyz-999', got %v", toolErr.MissingTools)
	}

	// Verify error classification
	errClass := recovery.ClassifyError(err)
	if errClass != recovery.ClassPreflight {
		t.Errorf("expected error class ClassPreflight, got %v", errClass)
	}

	// Build recovery block
	meta := &recovery.PreflightMetadata{
		MissingTools: toolErr.MissingTools,
	}
	block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
		PipelineName:  p.Metadata.Name,
		Input:         "test input",
		RunID:         "test-run-87654321",
		WorkspaceRoot: ".wave/workspaces",
		ErrClass:      errClass,
		PreflightMeta: meta,
	})

	// Verify recovery hints include tool guidance
	foundToolHint := false
	for _, hint := range block.Hints {
		if strings.Contains(hint.Command, "nonexistent-cli-tool-xyz-999") &&
			(strings.Contains(hint.Command, "PATH") || strings.Contains(hint.Command, "package manager")) {
			foundToolHint = true
			if hint.Label != "Install missing tool" {
				t.Errorf("expected hint label 'Install missing tool', got %q", hint.Label)
			}
			break
		}
	}
	if !foundToolHint {
		t.Errorf("expected recovery hints to include tool installation guidance for 'nonexistent-cli-tool-xyz-999', got: %+v", block.Hints)
	}

	// Verify workspace path has no double slashes
	if strings.Contains(block.WorkspacePath, "//") {
		t.Errorf("workspace path contains double slashes: %s", block.WorkspacePath)
	}

	// Verify no resume hint for preflight failures
	for _, hint := range block.Hints {
		if hint.Type == recovery.HintResume {
			t.Errorf("preflight failures should not include resume hints, found: %+v", hint)
		}
	}

	// Verify error message doesn't have redundant "preflight check failed"
	errMsg := err.Error()
	count := strings.Count(errMsg, "preflight check failed")
	if count > 1 {
		t.Errorf("error message contains redundant 'preflight check failed' (%d times): %s", count, errMsg)
	}
}

// TestPreflightRecovery_MixedFailures verifies that when a pipeline fails due to both
// missing skills and missing tools, recovery hints for both are provided correctly.
func TestPreflightRecovery_MixedFailures(t *testing.T) {
	// Create a test pipeline requiring both a missing skill and a missing tool
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{
			Name:        "test-mixed-pipeline",
			Description: "Test pipeline for mixed preflight failures",
		},
		Requires: &pipeline.Requires{
			Skills: []string{"missing-skill-alpha", "missing-skill-beta"},
			Tools:  []string{"missing-tool-alpha", "missing-tool-beta"},
		},
		Steps: []pipeline.Step{
			{
				ID:      "step1",
				Persona: "navigator",
			},
		},
	}

	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			WorkspaceRoot:         ".wave/workspaces",
			PipelineIDHashLength:  8,
		},
		Skills: map[string]manifest.SkillConfig{},
		Adapters: map[string]manifest.Adapter{
			"mock": {},
		},
	}

	mockAdapter := adapter.NewMockAdapter()
	executor := pipeline.NewDefaultPipelineExecutor(
		mockAdapter,
		pipeline.WithDebug(false),
	)

	ctx := context.Background()
	err := executor.Execute(ctx, p, m, "test input")

	// Verify error occurred
	if err == nil {
		t.Fatal("expected preflight check to fail for missing dependencies")
	}

	// The Run() method prioritizes SkillError over ToolError, so we should get a SkillError
	var skillErr *preflight.SkillError
	if !errors.As(err, &skillErr) {
		t.Fatalf("expected SkillError (prioritized), got %T: %v", err, err)
	}

	// Verify error classification
	errClass := recovery.ClassifyError(err)
	if errClass != recovery.ClassPreflight {
		t.Errorf("expected error class ClassPreflight, got %v", errClass)
	}

	// For mixed failures, we need to check both skill and tool results
	// In reality, the preflight checker should report both in results,
	// but the error returned is only for skills (prioritized)
	// Let's verify we can extract both for recovery hints
	checker := preflight.NewChecker(m.Skills)
	results, _ := checker.Run(p.Requires.Tools, p.Requires.Skills)

	// Extract missing skills and tools from results
	var missingSkills, missingTools []string
	for _, r := range results {
		if !r.OK {
			if r.Kind == "skill" {
				missingSkills = append(missingSkills, r.Name)
			} else if r.Kind == "tool" {
				missingTools = append(missingTools, r.Name)
			}
		}
	}

	// Build recovery block with both missing skills and tools
	meta := &recovery.PreflightMetadata{
		MissingSkills: missingSkills,
		MissingTools:  missingTools,
	}
	block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
		PipelineName:  p.Metadata.Name,
		Input:         "test input",
		RunID:         "test-run-mixed123",
		WorkspaceRoot: ".wave/workspaces",
		ErrClass:      errClass,
		PreflightMeta: meta,
	})

	// Verify hints include both skill install commands
	skillHintCount := 0
	for _, hint := range block.Hints {
		if strings.Contains(hint.Command, "wave.yaml skills.") {
			skillHintCount++
			// Verify it mentions one of our missing skills
			if !strings.Contains(hint.Command, "missing-skill-alpha") &&
				!strings.Contains(hint.Command, "missing-skill-beta") {
				t.Errorf("skill hint doesn't mention expected skills: %s", hint.Command)
			}
		}
	}
	if skillHintCount != 2 {
		t.Errorf("expected 2 skill install hints, got %d", skillHintCount)
	}

	// Verify hints include both tool guidance
	toolHintCount := 0
	for _, hint := range block.Hints {
		if hint.Label == "Install missing tool" {
			toolHintCount++
			// Verify it mentions one of our missing tools
			if !strings.Contains(hint.Command, "missing-tool-alpha") &&
				!strings.Contains(hint.Command, "missing-tool-beta") {
				t.Errorf("tool hint doesn't mention expected tools: %s", hint.Command)
			}
		}
	}
	if toolHintCount != 2 {
		t.Errorf("expected 2 tool install hints, got %d", toolHintCount)
	}

	// Verify workspace path has no double slashes
	if strings.Contains(block.WorkspacePath, "//") {
		t.Errorf("workspace path contains double slashes: %s", block.WorkspacePath)
	}

	// Verify no resume hint for preflight failures
	for _, hint := range block.Hints {
		if hint.Type == recovery.HintResume {
			t.Errorf("preflight failures should not include resume hints, found: %+v", hint)
		}
	}

	// Verify error message doesn't have redundant "preflight check failed"
	errMsg := err.Error()
	count := strings.Count(errMsg, "preflight check failed")
	if count > 1 {
		t.Errorf("error message contains redundant 'preflight check failed' (%d times): %s", count, errMsg)
	}
}

// TestPreflightRecovery_NoRedundantErrorMessage verifies that the error message chain
// doesn't contain redundant "preflight check failed" text in all scenarios.
func TestPreflightRecovery_NoRedundantErrorMessage(t *testing.T) {
	tests := []struct {
		name         string
		requirements *pipeline.Requires
		skills       map[string]manifest.SkillConfig
		wantErrType  string // "skill" or "tool"
	}{
		{
			name: "skill failure only",
			requirements: &pipeline.Requires{
				Skills: []string{"nonexistent-skill"},
			},
			skills:      map[string]manifest.SkillConfig{},
			wantErrType: "skill",
		},
		{
			name: "tool failure only",
			requirements: &pipeline.Requires{
				Tools: []string{"nonexistent-tool-xyz"},
			},
			skills:      map[string]manifest.SkillConfig{},
			wantErrType: "tool",
		},
		{
			name: "both skill and tool failures",
			requirements: &pipeline.Requires{
				Skills: []string{"nonexistent-skill"},
				Tools:  []string{"nonexistent-tool-xyz"},
			},
			skills:      map[string]manifest.SkillConfig{},
			wantErrType: "skill", // skill errors are prioritized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &pipeline.Pipeline{
				Metadata: pipeline.PipelineMetadata{
					Name:        "test-pipeline",
					Description: "Test pipeline",
				},
				Requires: tt.requirements,
				Steps: []pipeline.Step{
					{
						ID:      "step1",
						Persona: "navigator",
					},
				},
			}

			m := &manifest.Manifest{
				Runtime: manifest.Runtime{
					WorkspaceRoot:        ".wave/workspaces",
					PipelineIDHashLength: 8,
				},
				Skills: tt.skills,
				Adapters: map[string]manifest.Adapter{
					"mock": {},
				},
			}

			mockAdapter := adapter.NewMockAdapter()
			executor := pipeline.NewDefaultPipelineExecutor(mockAdapter)

			ctx := context.Background()
			err := executor.Execute(ctx, p, m, "test input")

			if err == nil {
				t.Fatal("expected preflight check to fail")
			}

			// Verify "preflight check failed" appears at most once
			errMsg := err.Error()
			count := strings.Count(errMsg, "preflight check failed")
			if count > 1 {
				t.Errorf("error message contains redundant 'preflight check failed' (%d times): %s", count, errMsg)
			}

			// Also check for other potential redundancy patterns
			// The error should not say "preflight" multiple times in nested wrapping
			lowerMsg := strings.ToLower(errMsg)
			preflightCount := strings.Count(lowerMsg, "preflight")
			if preflightCount > 2 { // Allow some repetition but not excessive
				t.Logf("warning: 'preflight' appears %d times in error: %s", preflightCount, errMsg)
			}
		})
	}
}

// TestPreflightRecovery_CleanWorkspacePaths verifies that workspace paths in all
// scenarios do not contain double slashes.
func TestPreflightRecovery_CleanWorkspacePaths(t *testing.T) {
	tests := []struct {
		name          string
		workspaceRoot string
		runID         string
		stepID        string
		wantPath      string
	}{
		{
			name:          "preflight failure with empty stepID",
			workspaceRoot: ".wave/workspaces",
			runID:         "test-run-12345678",
			stepID:        "", // preflight failures have empty stepID
			wantPath:      ".wave/workspaces/test-run-12345678/",
		},
		{
			name:          "custom workspace root with empty stepID",
			workspaceRoot: "/tmp/wave-workspaces",
			runID:         "pipeline-abc123",
			stepID:        "",
			wantPath:      "/tmp/wave-workspaces/pipeline-abc123/",
		},
		{
			name:          "normal step failure with stepID",
			workspaceRoot: ".wave/workspaces",
			runID:         "test-run-xyz",
			stepID:        "step-1",
			wantPath:      ".wave/workspaces/test-run-xyz/step-1/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
				PipelineName:  "test-pipeline",
				Input:         "test input",
				StepID:        tt.stepID,
				RunID:         tt.runID,
				WorkspaceRoot: tt.workspaceRoot,
				ErrClass:      recovery.ClassPreflight,
			})

			// Verify path matches expected
			if block.WorkspacePath != tt.wantPath {
				t.Errorf("workspace path = %q, want %q", block.WorkspacePath, tt.wantPath)
			}

			// Verify no double slashes anywhere in path
			if strings.Contains(block.WorkspacePath, "//") {
				t.Errorf("workspace path contains double slashes: %s", block.WorkspacePath)
			}

			// Verify in recovery hints as well
			for _, hint := range block.Hints {
				if strings.Contains(hint.Command, "//") {
					t.Errorf("hint command contains double slashes: %s", hint.Command)
				}
			}
		})
	}
}

// TestPreflightRecovery_EndToEndFlow simulates a complete end-to-end flow including
// pipeline execution, error detection, classification, and recovery hint generation.
func TestPreflightRecovery_EndToEndFlow(t *testing.T) {
	// This test simulates what happens in cmd/wave/commands/run.go
	// when a pipeline fails preflight checks

	// Step 1: Create pipeline with missing dependencies
	p := &pipeline.Pipeline{
		Metadata: pipeline.PipelineMetadata{
			Name:        "e2e-test-pipeline",
			Description: "End-to-end test pipeline",
		},
		Requires: &pipeline.Requires{
			Skills: []string{"speckit"},
			Tools:  []string{"nonexistent-cli-xyz-e2e"},
		},
		Steps: []pipeline.Step{
			{
				ID:      "navigate",
				Persona: "navigator",
			},
		},
	}

	// Step 2: Create manifest without the required skill
	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			WorkspaceRoot:        ".wave/workspaces",
			PipelineIDHashLength: 8,
		},
		Skills: map[string]manifest.SkillConfig{
			// speckit is not configured
		},
		Adapters: map[string]manifest.Adapter{
			"mock": {},
		},
	}

	// Step 3: Execute pipeline
	mockAdapter := adapter.NewMockAdapter(
		adapter.WithSimulatedDelay(10 * time.Millisecond),
	)
	executor := pipeline.NewDefaultPipelineExecutor(
		mockAdapter,
		pipeline.WithDebug(false),
	)

	ctx := context.Background()
	execErr := executor.Execute(ctx, p, m, "test feature request")

	// Step 4: Verify execution failed
	if execErr == nil {
		t.Fatal("expected pipeline execution to fail at preflight")
	}

	// Step 5: Extract error details (as run.go does)
	var stepErr *pipeline.StepError
	var stepID string
	cause := execErr
	if errors.As(execErr, &stepErr) {
		stepID = stepErr.StepID
		cause = stepErr.Err
	}

	// Step 6: Classify error
	errClass := recovery.ClassifyError(cause)
	if errClass != recovery.ClassPreflight {
		t.Errorf("expected ClassPreflight, got %v", errClass)
	}

	// Step 7: Extract preflight metadata
	// Note: The error returned by Run() may only contain one type (skill or tool error),
	// but we need both. We run the checker again to get all results.
	var preflightMeta *recovery.PreflightMetadata
	if errClass == recovery.ClassPreflight {
		// Re-run the checker to get detailed results
		checker := preflight.NewChecker(m.Skills)
		results, _ := checker.Run(p.Requires.Tools, p.Requires.Skills)

		// Extract missing skills and tools from results
		var missingSkills, missingTools []string
		for _, r := range results {
			if !r.OK {
				if r.Kind == "skill" {
					missingSkills = append(missingSkills, r.Name)
				} else if r.Kind == "tool" {
					missingTools = append(missingTools, r.Name)
				}
			}
		}

		if len(missingSkills) > 0 || len(missingTools) > 0 {
			preflightMeta = &recovery.PreflightMetadata{
				MissingSkills: missingSkills,
				MissingTools:  missingTools,
			}
		}
	}

	// Step 8: Build recovery block
	runID := "e2e-test-run-12345678"
	block := recovery.BuildRecoveryBlock(recovery.RecoveryBlockOpts{
		PipelineName:  p.Metadata.Name,
		Input:         "test feature request",
		StepID:        stepID,
		RunID:         runID,
		WorkspaceRoot: m.Runtime.WorkspaceRoot,
		ErrClass:      errClass,
		PreflightMeta: preflightMeta,
	})

	// Step 9: Verify recovery block structure
	if block.PipelineName != p.Metadata.Name {
		t.Errorf("block.PipelineName = %q, want %q", block.PipelineName, p.Metadata.Name)
	}

	if block.Input != "test feature request" {
		t.Errorf("block.Input = %q, want %q", block.Input, "test feature request")
	}

	if block.ErrorClass != recovery.ClassPreflight {
		t.Errorf("block.ErrorClass = %v, want %v", block.ErrorClass, recovery.ClassPreflight)
	}

	// Step 10: Verify recovery hints
	var hasSkillHint, hasToolHint, hasWorkspaceHint bool
	var hasResumeHint bool

	for _, hint := range block.Hints {
		if strings.Contains(hint.Command, "wave.yaml skills.speckit.install") {
			hasSkillHint = true
		}
		if strings.Contains(hint.Command, "nonexistent-cli-xyz-e2e") && strings.Contains(hint.Command, "package manager") {
			hasToolHint = true
		}
		if hint.Type == recovery.HintWorkspace {
			hasWorkspaceHint = true
		}
		if hint.Type == recovery.HintResume {
			hasResumeHint = true
		}
	}

	if !hasSkillHint {
		t.Error("expected recovery hints to reference wave.yaml skills config")
	}

	if !hasToolHint {
		t.Error("expected recovery hints to include tool installation guidance for 'nonexistent-cli-xyz-e2e'")
	}

	if !hasWorkspaceHint {
		t.Error("expected recovery hints to include workspace inspection hint")
	}

	if hasResumeHint {
		t.Error("preflight failures should not include resume hints")
	}

	// Step 11: Verify workspace path is clean
	if strings.Contains(block.WorkspacePath, "//") {
		t.Errorf("workspace path contains double slashes: %s", block.WorkspacePath)
	}

	// Step 12: Verify error message quality
	errMsg := execErr.Error()
	preflightCount := strings.Count(errMsg, "preflight check failed")
	if preflightCount > 1 {
		t.Errorf("error message contains redundant 'preflight check failed' (%d times): %s", preflightCount, errMsg)
	}

	// Step 13: Verify formatted recovery block
	formattedBlock := recovery.FormatRecoveryBlock(block)
	if formattedBlock == "" {
		t.Error("expected non-empty formatted recovery block")
	}

	if !strings.Contains(formattedBlock, "Recovery options:") {
		t.Error("formatted block should contain 'Recovery options:' header")
	}

	// Step 14: Verify all acceptance criteria
	t.Run("acceptance_criteria", func(t *testing.T) {
		// AC1: Recovery options suggest installing the missing skill
		if !strings.Contains(formattedBlock, "wave.yaml skills.") {
			t.Error("AC1 failed: recovery options should suggest installing the missing skill")
		}

		// AC2: Workspace paths do not contain double slashes
		if strings.Contains(block.WorkspacePath, "//") {
			t.Error("AC2 failed: workspace paths should not contain double slashes")
		}

		// AC3: Error message chain does not redundantly repeat "preflight check failed"
		if strings.Count(errMsg, "preflight check failed") > 1 {
			t.Error("AC3 failed: error message should not redundantly repeat 'preflight check failed'")
		}

		// AC4: Recovery options are tailored to the specific failure type
		skillHintCount := 0
		toolHintCount := 0
		for _, hint := range block.Hints {
			if strings.Contains(hint.Command, "wave.yaml skills.") {
				skillHintCount++
			}
			if hint.Label == "Install missing tool" {
				toolHintCount++
			}
		}
		if skillHintCount == 0 {
			t.Error("AC4 failed: expected skill-specific recovery hints")
		}
		if toolHintCount == 0 {
			t.Error("AC4 failed: expected tool-specific recovery hints")
		}
	})
}

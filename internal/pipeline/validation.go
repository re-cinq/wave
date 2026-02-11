package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PhaseSkipValidator validates that pipeline phases are not skipped
type PhaseSkipValidator struct {
	prototypePhasesOrder []string
}

// NewPhaseSkipValidator creates a new phase skip validator
func NewPhaseSkipValidator() *PhaseSkipValidator {
	return &PhaseSkipValidator{
		prototypePhasesOrder: []string{"spec", "docs", "dummy", "implement"},
	}
}

// ValidatePhaseSequence validates that phases are not skipped in the prototype pipeline
func (v *PhaseSkipValidator) ValidatePhaseSequence(p *Pipeline, fromStep string) error {
	// Only apply phase skip validation to prototype pipeline
	if p.Metadata.Name != "prototype" {
		return nil
	}

	if fromStep == "" {
		return nil // Starting from beginning, no validation needed
	}

	// Find the index of the fromStep in the expected order
	fromIndex := -1
	for i, phase := range v.prototypePhasesOrder {
		if phase == fromStep {
			fromIndex = i
			break
		}
	}

	if fromIndex == -1 {
		return nil // Not a prototype phase, skip validation
	}

	// Verify that all prerequisite phases have been completed
	workspaceRoot := fmt.Sprintf(".wave/workspaces/%s", p.Metadata.Name)

	for i := 0; i < fromIndex; i++ {
		prerequisitePhase := v.prototypePhasesOrder[i]

		// Check if prerequisite phase workspace exists and has valid outputs
		phaseWorkspace := filepath.Join(workspaceRoot, prerequisitePhase)
		if err := v.validatePhaseCompletion(prerequisitePhase, phaseWorkspace); err != nil {
			return fmt.Errorf("cannot skip to phase '%s': prerequisite phase '%s' not completed: %w",
				fromStep, prerequisitePhase, err)
		}
	}

	return nil
}

// validatePhaseCompletion checks if a phase has been properly completed
func (v *PhaseSkipValidator) validatePhaseCompletion(phase, workspacePath string) error {
	// Check if workspace directory exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return fmt.Errorf("workspace not found")
	}

	// Check if artifact.json exists (required for contract validation)
	artifactPath := filepath.Join(workspacePath, "artifact.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return fmt.Errorf("contract validation artifact missing")
	}

	// Check phase-specific artifacts
	switch phase {
	case "spec":
		specPath := filepath.Join(workspacePath, "spec.md")
		if _, err := os.Stat(specPath); os.IsNotExist(err) {
			return fmt.Errorf("spec.md artifact missing")
		}
	case "docs":
		docsPath := filepath.Join(workspacePath, "feature-docs.md")
		if _, err := os.Stat(docsPath); os.IsNotExist(err) {
			return fmt.Errorf("feature-docs.md artifact missing")
		}
	case "dummy":
		prototypePath := filepath.Join(workspacePath, "prototype")
		if _, err := os.Stat(prototypePath); os.IsNotExist(err) {
			return fmt.Errorf("prototype/ directory missing")
		}
		interfacesPath := filepath.Join(workspacePath, "interfaces.md")
		if _, err := os.Stat(interfacesPath); os.IsNotExist(err) {
			return fmt.Errorf("interfaces.md artifact missing")
		}
	case "implement":
		planPath := filepath.Join(workspacePath, "implementation-plan.md")
		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			return fmt.Errorf("implementation-plan.md artifact missing")
		}
	}

	return nil
}

// StaleArtifactDetector detects when upstream artifacts have been modified
type StaleArtifactDetector struct {
	artifactTimestamps map[string]time.Time
}

// NewStaleArtifactDetector creates a new stale artifact detector
func NewStaleArtifactDetector() *StaleArtifactDetector {
	return &StaleArtifactDetector{
		artifactTimestamps: make(map[string]time.Time),
	}
}

// DetectStaleArtifacts checks if any upstream artifacts are newer than dependent phase artifacts
func (d *StaleArtifactDetector) DetectStaleArtifacts(p *Pipeline, currentStep string) ([]string, error) {
	if p.Metadata.Name != "prototype" {
		return nil, nil
	}

	staleReasons := []string{}
	workspaceRoot := fmt.Sprintf(".wave/workspaces/%s", p.Metadata.Name)

	// Get current step dependencies
	var currentStepObj *Step
	for i := range p.Steps {
		if p.Steps[i].ID == currentStep {
			currentStepObj = &p.Steps[i]
			break
		}
	}

	if currentStepObj == nil {
		return nil, fmt.Errorf("step %s not found", currentStep)
	}

	currentStepWorkspace := filepath.Join(workspaceRoot, currentStep)
	currentStepTime, err := d.getWorkspaceModTime(currentStepWorkspace)
	if err != nil {
		// If current step workspace doesn't exist, no staleness to check
		return nil, nil
	}

	// Check each dependency for staleness
	for _, depStep := range currentStepObj.Dependencies {
		depWorkspace := filepath.Join(workspaceRoot, depStep)
		depModTime, err := d.getWorkspaceModTime(depWorkspace)
		if err != nil {
			continue // Skip if dependency workspace doesn't exist
		}

		// If dependency was modified after current step, it's stale
		if depModTime.After(currentStepTime) {
			staleReasons = append(staleReasons, fmt.Sprintf(
				"upstream phase '%s' was re-run at %s, after '%s' completed at %s",
				depStep, depModTime.Format(time.RFC3339), currentStep, currentStepTime.Format(time.RFC3339)))
		}
	}

	return staleReasons, nil
}

// getWorkspaceModTime gets the modification time of the most recently modified file in a workspace
func (d *StaleArtifactDetector) getWorkspaceModTime(workspacePath string) (time.Time, error) {
	var latestTime time.Time

	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}
		return nil
	})

	if err != nil {
		return time.Time{}, err
	}

	return latestTime, nil
}

// ErrorMessageProvider provides clear error messages with retry guidance
type ErrorMessageProvider struct{}

// NewErrorMessageProvider creates a new error message provider
func NewErrorMessageProvider() *ErrorMessageProvider {
	return &ErrorMessageProvider{}
}

// FormatPhaseFailureError formats a clear error message for phase failures
func (e *ErrorMessageProvider) FormatPhaseFailureError(phase string, originalError error, pipelineName ...string) error {
	pName := "prototype"
	if len(pipelineName) > 0 && pipelineName[0] != "" {
		pName = pipelineName[0]
	}
	var guidance strings.Builder

	guidance.WriteString(fmt.Sprintf("❌ Phase '%s' failed: %v\n\n", phase, originalError))
	guidance.WriteString("🔧 Troubleshooting Guide:\n")

	switch phase {
	case "spec":
		guidance.WriteString("  • Verify project description is clear and complete\n")
		guidance.WriteString("  • Check that craftsman persona has write permissions\n")
		guidance.WriteString("  • Ensure spec.md and requirements.md are created\n")
		guidance.WriteString("  • Validate that artifact.json follows the contract schema\n")
	case "docs":
		guidance.WriteString("  • Verify spec phase completed successfully\n")
		guidance.WriteString("  • Check that artifacts/input-spec.md is accessible\n")
		guidance.WriteString("  • Ensure feature-docs.md and stakeholder-summary.md are created\n")
		guidance.WriteString("  • Validate documentation quality meets contract requirements\n")
	case "dummy":
		guidance.WriteString("  • Verify docs phase completed successfully\n")
		guidance.WriteString("  • Check that prototype/ directory is created with working code\n")
		guidance.WriteString("  • Ensure interfaces.md documents all interfaces\n")
		guidance.WriteString("  • Validate prototype is marked as runnable if applicable\n")
	case "implement":
		guidance.WriteString("  • Verify all previous phases (spec, docs, dummy) completed\n")
		guidance.WriteString("  • Check that implementation-plan.md provides clear guidance\n")
		guidance.WriteString("  • Ensure implementation-checklist.md tracks progress\n")
		guidance.WriteString("  • Validate test execution and coverage requirements\n")
	default:
		guidance.WriteString("  • Check pipeline configuration and dependencies\n")
		guidance.WriteString("  • Verify workspace permissions and file access\n")
		guidance.WriteString("  • Review contract validation requirements\n")
	}

	guidance.WriteString("\n🔄 Retry Options:\n")
	guidance.WriteString(fmt.Sprintf("  • Re-run the same phase: wave run %s --from-step %s\n", pName, phase))
	guidance.WriteString(fmt.Sprintf("  • Start from the beginning: wave run %s\n", pName))

	guidance.WriteString("\n📋 Debug Information:\n")
	guidance.WriteString(fmt.Sprintf("  • Phase: %s\n", phase))
	guidance.WriteString(fmt.Sprintf("  • Pipeline: %s\n", pName))
	guidance.WriteString(fmt.Sprintf("  • Workspace: .wave/workspaces/%s/%s/\n", pName, phase))
	guidance.WriteString("  • Logs: .wave/traces/\n")

	return fmt.Errorf("%s", guidance.String())
}

// FormatContractValidationError formats contract validation errors with specific guidance
func (e *ErrorMessageProvider) FormatContractValidationError(phase string, contractError error) error {
	var guidance strings.Builder

	guidance.WriteString(fmt.Sprintf("📋 Contract validation failed for phase '%s'\n\n", phase))
	guidance.WriteString(fmt.Sprintf("Error: %v\n\n", contractError))

	guidance.WriteString("🔍 Contract Requirements:\n")
	switch phase {
	case "spec":
		guidance.WriteString("  • artifact.json with phase: 'spec'\n")
		guidance.WriteString("  • spec.md file must exist and be non-empty\n")
		guidance.WriteString("  • validation.specification_quality: ['poor'|'fair'|'good'|'excellent']\n")
		guidance.WriteString("  • metadata.timestamp and metadata.input_description required\n")
	case "docs":
		guidance.WriteString("  • artifact.json with phase: 'docs'\n")
		guidance.WriteString("  • feature-docs.md and stakeholder-summary.md must exist\n")
		guidance.WriteString("  • validation.documentation_quality: ['poor'|'fair'|'good'|'excellent']\n")
		guidance.WriteString("  • validation.coverage_percentage: 0-100\n")
	case "dummy":
		guidance.WriteString("  • artifact.json with phase: 'dummy'\n")
		guidance.WriteString("  • prototype/ directory must exist\n")
		guidance.WriteString("  • interfaces.md must exist and document interfaces\n")
		guidance.WriteString("  • validation.runnable: true|false\n")
		guidance.WriteString("  • validation.prototype_quality: ['poor'|'fair'|'good'|'excellent']\n")
	case "implement":
		guidance.WriteString("  • artifact.json with phase: 'implement'\n")
		guidance.WriteString("  • implementation-plan.md must exist\n")
		guidance.WriteString("  • implementation-checklist.md must exist\n")
		guidance.WriteString("  • validation.tests_executed: true|false\n")
		guidance.WriteString("  • validation.implementation_readiness: ['ready'|'partial'|'needs_work']\n")
	}

	guidance.WriteString("\n📖 Schema Location: .wave/contracts/")
	guidance.WriteString(phase)
	guidance.WriteString("-phase.schema.json\n")

	return fmt.Errorf("%s", guidance.String())
}

// ConcurrencyValidator validates and prevents concurrent pipeline executions
type ConcurrencyValidator struct {
	runningPipelines map[string]string // pipelineID -> workspaceID
	workspaceLocks   map[string]bool   // workspaceID -> isLocked
}

// NewConcurrencyValidator creates a new concurrency validator
func NewConcurrencyValidator() *ConcurrencyValidator {
	return &ConcurrencyValidator{
		runningPipelines: make(map[string]string),
		workspaceLocks:   make(map[string]bool),
	}
}

// AcquireWorkspaceLock attempts to acquire a lock on the workspace for a pipeline
func (c *ConcurrencyValidator) AcquireWorkspaceLock(pipelineID, workspaceID string) error {
	if c.workspaceLocks[workspaceID] {
		return fmt.Errorf("workspace '%s' is already in use by another pipeline execution. "+
			"Wait for the current execution to complete or use a different workspace", workspaceID)
	}

	if existingWorkspace, exists := c.runningPipelines[pipelineID]; exists {
		return fmt.Errorf("pipeline '%s' is already running with workspace '%s'. "+
			"Use 'wave status' to check execution status or 'wave cancel %s' to stop",
			pipelineID, existingWorkspace, pipelineID)
	}

	c.workspaceLocks[workspaceID] = true
	c.runningPipelines[pipelineID] = workspaceID
	return nil
}

// ReleaseWorkspaceLock releases the workspace lock for a pipeline
func (c *ConcurrencyValidator) ReleaseWorkspaceLock(pipelineID string) {
	if workspaceID, exists := c.runningPipelines[pipelineID]; exists {
		delete(c.runningPipelines, pipelineID)
		delete(c.workspaceLocks, workspaceID)
	}
}

// IsWorkspaceInUse checks if a workspace is currently in use
func (c *ConcurrencyValidator) IsWorkspaceInUse(workspaceID string) bool {
	return c.workspaceLocks[workspaceID]
}

// GetRunningPipelines returns a list of currently running pipelines
func (c *ConcurrencyValidator) GetRunningPipelines() map[string]string {
	result := make(map[string]string)
	for pipelineID, workspaceID := range c.runningPipelines {
		result[pipelineID] = workspaceID
	}
	return result
}
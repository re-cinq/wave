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

// ValidatePhaseSequence validates that phases are not skipped when resuming.
// For prototype pipelines, it checks prototype-specific phase artifacts.
// For other pipelines, it verifies that prior steps have workspace directories
// from at least one prior run.
func (v *PhaseSkipValidator) ValidatePhaseSequence(p *Pipeline, fromStep string) error {
	if fromStep == "" {
		return nil // Starting from beginning, no validation needed
	}

	// Prototype-specific validation
	if p.Metadata.Name == "impl-prototype" || p.Metadata.Name == "prototype" {
		return v.validatePrototypePhaseSequence(p, fromStep)
	}

	// Generic validation for non-prototype pipelines
	return v.validateGenericStepSequence(p, fromStep)
}

// validatePrototypePhaseSequence validates prototype pipeline phase prerequisites.
func (v *PhaseSkipValidator) validatePrototypePhaseSequence(p *Pipeline, fromStep string) error {
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

// validateGenericStepSequence validates that prior steps have workspace artifacts
// from at least one prior run. This catches the common case where a user tries
// to resume from a step that has no prior state at all.
func (v *PhaseSkipValidator) validateGenericStepSequence(p *Pipeline, fromStep string) error {
	// If fromStep is the first step, no prior work is needed
	if len(p.Steps) > 0 && p.Steps[0].ID == fromStep {
		return nil
	}

	wsRoot := ".wave/workspaces"

	// Collect run directories for this pipeline
	runDirs, _ := filepath.Glob(filepath.Join(wsRoot, p.Metadata.Name+"-*"))
	if info, err := os.Stat(filepath.Join(wsRoot, p.Metadata.Name)); err == nil && info.IsDir() {
		runDirs = append(runDirs, filepath.Join(wsRoot, p.Metadata.Name))
	}

	// If no run directories exist at all, prior steps can't have completed
	if len(runDirs) == 0 {
		return fmt.Errorf("cannot resume from '%s': no prior run state found for pipeline '%s'",
			fromStep, p.Metadata.Name)
	}

	// Check that each step before fromStep has a workspace in at least one run dir
	for _, step := range p.Steps {
		if step.ID == fromStep {
			break
		}

		if v.hasWorkspaceInAnyRun(step, runDirs) {
			continue
		}

		return fmt.Errorf("cannot resume from '%s': prior step '%s' has no workspace artifacts",
			fromStep, step.ID)
	}

	return nil
}

// hasWorkspaceInAnyRun checks if a step has a workspace directory in any run dir.
func (v *PhaseSkipValidator) hasWorkspaceInAnyRun(step Step, runDirs []string) bool {
	for _, runDir := range runDirs {
		// Check step-named directory
		if _, err := os.Stat(filepath.Join(runDir, step.ID)); err == nil {
			return true
		}
		// Check __wt_ directories (worktree steps)
		entries, _ := filepath.Glob(filepath.Join(runDir, "__wt_*"))
		if len(entries) > 0 {
			return true
		}
	}
	return false
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
	if p.Metadata.Name != "impl-prototype" && p.Metadata.Name != "prototype" {
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
		guidance.WriteString("  • Check that .wave/artifacts/input-spec.md is accessible\n")
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

// ResolvePipelineRetryPolicies calls ResolvePolicy() on each step's retry config,
// resolving named policies into concrete values before the executor runs.
func ResolvePipelineRetryPolicies(p *Pipeline) error {
	for i := range p.Steps {
		if err := p.Steps[i].Retry.ResolvePolicy(); err != nil {
			return fmt.Errorf("step %q: %w", p.Steps[i].ID, err)
		}
	}
	return nil
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

// ValidateThreadFields validates thread and fidelity fields across all pipeline steps.
// Returns a list of validation errors (empty if valid).
func ValidateThreadFields(p *Pipeline) []error {
	var errs []error
	for _, step := range p.Steps {
		if step.Fidelity != "" && !validFidelityValues[step.Fidelity] {
			errs = append(errs, fmt.Errorf("step %q: unknown fidelity value %q (valid: full, compact, summary, fresh)", step.ID, step.Fidelity))
		}
		if step.Fidelity != "" && step.Thread == "" {
			errs = append(errs, fmt.Errorf("step %q: fidelity %q set without thread — fidelity has no effect without a thread group", step.ID, step.Fidelity))
		}
	}
	return errs
}

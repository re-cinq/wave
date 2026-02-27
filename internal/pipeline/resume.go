package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

// ResumeManager handles pipeline resumption from specific steps
type ResumeManager struct {
	executor    *DefaultPipelineExecutor
	validator   *PhaseSkipValidator
	detector    *StaleArtifactDetector
	concurrency *ConcurrencyValidator
	errors      *ErrorMessageProvider
}

// NewResumeManager creates a new resume manager with all validation components
func NewResumeManager(executor *DefaultPipelineExecutor) *ResumeManager {
	return &ResumeManager{
		executor:    executor,
		validator:   NewPhaseSkipValidator(),
		detector:    NewStaleArtifactDetector(),
		concurrency: NewConcurrencyValidator(),
		errors:      NewErrorMessageProvider(),
	}
}

// ResumeFromStep resumes pipeline execution from a specific step with enhanced validation.
// When force is true, phase validation and stale artifact checks are skipped.
func (r *ResumeManager) ResumeFromStep(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string, fromStep string, force bool) error {
	if fromStep == "" {
		return fmt.Errorf("fromStep cannot be empty for resume operation")
	}

	// Validate that the step exists in the pipeline
	var targetStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == fromStep {
			targetStep = &p.Steps[i]
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("step '%s' not found in pipeline '%s'. Available steps: %s",
			fromStep, p.Metadata.Name, r.getAvailableSteps(p))
	}

	if !force {
		// Phase skip validation - ensure prerequisites are completed
		if err := r.validator.ValidatePhaseSequence(p, fromStep); err != nil {
			return r.errors.FormatPhaseFailureError(fromStep, err, p.Metadata.Name)
		}

		// Stale artifact detection - warn about outdated artifacts
		staleReasons, err := r.detector.DetectStaleArtifacts(p, fromStep)
		if err != nil {
			return fmt.Errorf("failed to detect stale artifacts: %w", err)
		}

		if len(staleReasons) > 0 {
			fmt.Printf("Warning: Stale artifacts detected for step '%s':\n", fromStep)
			for _, reason := range staleReasons {
				fmt.Printf("   - %s\n", reason)
			}
			fmt.Printf("\nRecommendation: Consider re-running upstream phases to ensure consistency.\n\n")
		}
	}

	// Concurrency protection - ensure workspace is not in use
	workspaceID := fmt.Sprintf("%s/%s", p.Metadata.Name, fromStep)
	if err := r.concurrency.AcquireWorkspaceLock(p.Metadata.Name, workspaceID); err != nil {
		return fmt.Errorf("cannot resume pipeline: %w", err)
	}

	// Ensure workspace lock is released when done
	defer r.concurrency.ReleaseWorkspaceLock(p.Metadata.Name)

	// Prepare partial pipeline starting from target step
	resumePipeline := r.createResumeSubpipeline(p, fromStep)

	// Initialize resume context with preserved state
	resumeState, err := r.loadResumeState(p, fromStep)
	if err != nil {
		return fmt.Errorf("failed to load resume state: %w", err)
	}

	// Emit resume state summary so the user knows what was recovered
	pipelineName := p.Metadata.Name
	if r.executor.emitter != nil {
		if len(resumeState.CompletedSteps) > 0 {
			r.executor.emitter.Emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineName,
				StepID:     fromStep,
				State:      "resuming",
				Message:    fmt.Sprintf("Resume: found %d completed step(s): %s", len(resumeState.CompletedSteps), strings.Join(resumeState.CompletedSteps, ", ")),
			})
		}
		for key, path := range resumeState.ArtifactPaths {
			r.executor.emitter.Emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineName,
				StepID:     fromStep,
				State:      "resuming",
				Message:    fmt.Sprintf("Resume: recovered artifact %s → %s", key, path),
			})
		}
		if len(resumeState.CompletedSteps) == 0 {
			r.executor.emitter.Emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineName,
				StepID:     fromStep,
				State:      "resuming",
				Message:    "Resume: no prior state found — starting fresh",
			})
		}

		// Emit synthetic completion events for prior steps so the display
		// marks them as completed (✓) instead of pending (○).
		for _, stepID := range resumeState.CompletedSteps {
			persona := r.lookupStepPersona(p, stepID)
			r.executor.emitter.Emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineName,
				StepID:     stepID,
				State:      "completed",
				Persona:    persona,
				Message:    "Completed in prior run",
			})
		}
	}

	// Generate a new runtime ID for this resumed execution.
	// Prefer CreateRun() so resumed runs appear in the dashboard.
	hashLength := m.Runtime.PipelineIDHashLength
	pipelineID := r.executor.createRunID(pipelineName, hashLength, input)

	// Create new execution with preserved artifacts and state
	execution := &PipelineExecution{
		Pipeline:       resumePipeline,
		Manifest:       m,
		States:         resumeState.States,
		Results:        resumeState.Results,
		ArtifactPaths:  resumeState.ArtifactPaths,
		WorkspacePaths: resumeState.WorkspacePaths,
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          input,
		Context:        newContextWithProject(pipelineID, pipelineName, fromStep, m),
		Status: &PipelineStatus{
			ID:             pipelineID,
			PipelineName:   pipelineName,
			State:          StateRunning,
			CurrentStep:    fromStep,
			CompletedSteps: resumeState.CompletedSteps,
			StartedAt:      time.Now(),
		},
	}

	// Store execution state
	r.executor.mu.Lock()
	r.executor.pipelines[pipelineID] = execution
	r.executor.mu.Unlock()

	// Execute starting from the target step
	return r.executeResumedPipeline(ctx, execution, fromStep)
}

// ResumeState holds state information needed for resumption
type ResumeState struct {
	States         map[string]string
	Results        map[string]map[string]interface{}
	ArtifactPaths  map[string]string
	WorkspacePaths map[string]string
	CompletedSteps []string
}

// lookupStepPersona finds the persona for a step by ID in the full pipeline.
func (r *ResumeManager) lookupStepPersona(p *Pipeline, stepID string) string {
	for _, step := range p.Steps {
		if step.ID == stepID {
			return step.Persona
		}
	}
	return ""
}

// loadResumeState loads state from previous execution for resumption
func (r *ResumeManager) loadResumeState(p *Pipeline, fromStep string) (*ResumeState, error) {
	state := &ResumeState{
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		CompletedSteps: []string{},
	}

	wsRoot := ".wave/workspaces"

	// Find the most recent run directory for this pipeline.
	// Run dirs are named <pipelineName>-<timestamp>-<hash> and sorted
	// lexicographically so the last match is the most recent.
	runDirs, _ := filepath.Glob(filepath.Join(wsRoot, p.Metadata.Name+"-*"))
	// Load completed steps state from workspace
	for _, step := range p.Steps {
		if step.ID == fromStep {
			break // Don't include the target step in completed steps
		}

		// Resolve workspace path for this step
		stepWorkspace := ""
		if step.Workspace.Ref != "" {
			// Ref steps share the referenced step's workspace
			if refPath, ok := state.WorkspacePaths[step.Workspace.Ref]; ok {
				stepWorkspace = refPath
			}
		}

		if stepWorkspace == "" {
			// Search across all run dirs (most recent first) for this step's workspace
			for i := len(runDirs) - 1; i >= 0; i-- {
				runDir := runDirs[i]

				// For worktree steps, look for __wt_* dirs
				if step.Workspace.Type == "worktree" {
					entries, _ := filepath.Glob(filepath.Join(runDir, "__wt_*"))
					for _, entry := range entries {
						// Check if this worktree has the step's output artifacts
						if hasStepArtifacts(entry, step) {
							stepWorkspace = entry
							break
						}
					}
					if stepWorkspace != "" {
						break
					}
					// Also check old-style step-named dirs
					candidate := filepath.Join(runDir, step.ID)
					if _, err := os.Stat(candidate); err == nil {
						stepWorkspace = candidate
						break
					}
				} else {
					candidate := filepath.Join(runDir, step.ID)
					if _, err := os.Stat(candidate); err == nil {
						stepWorkspace = candidate
						break
					}
				}
			}
		}

		if stepWorkspace != "" {
			if _, err := os.Stat(stepWorkspace); err == nil {
				// Step workspace exists, mark as completed
				state.States[step.ID] = StateCompleted
				state.CompletedSteps = append(state.CompletedSteps, step.ID)
				state.WorkspacePaths[step.ID] = stepWorkspace

				// Load artifact paths for this step
				for _, artifact := range step.OutputArtifacts {
					artifactKey := fmt.Sprintf("%s:%s", step.ID, artifact.Name)
					artifactPath := filepath.Join(stepWorkspace, artifact.Path)
					state.ArtifactPaths[artifactKey] = artifactPath
				}
			}
		}
	}

	return state, nil
}

// hasStepArtifacts checks if a workspace directory contains the output artifacts for a step.
func hasStepArtifacts(wsPath string, step Step) bool {
	if len(step.OutputArtifacts) == 0 {
		// No artifacts to check — the dir existing is enough
		return true
	}
	for _, art := range step.OutputArtifacts {
		if _, err := os.Stat(filepath.Join(wsPath, art.Path)); err == nil {
			return true
		}
	}
	return false
}

// createResumeSubpipeline creates a new pipeline starting from the specified step.
// Dependencies on prior (completed) steps are stripped since they're not in the subpipeline.
func (r *ResumeManager) createResumeSubpipeline(p *Pipeline, fromStep string) *Pipeline {
	// Find the starting step index
	startIndex := -1
	for i, step := range p.Steps {
		if step.ID == fromStep {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		return p // Fallback to full pipeline
	}

	// Collect IDs of steps included in the subpipeline
	includedSteps := make(map[string]bool)
	for _, step := range p.Steps[startIndex:] {
		includedSteps[step.ID] = true
	}

	// Copy steps and strip dependencies on excluded (prior) steps
	subSteps := make([]Step, len(p.Steps[startIndex:]))
	copy(subSteps, p.Steps[startIndex:])
	for i := range subSteps {
		var kept []string
		for _, dep := range subSteps[i].Dependencies {
			if includedSteps[dep] {
				kept = append(kept, dep)
			}
		}
		subSteps[i].Dependencies = kept
	}

	resumePipeline := &Pipeline{
		Kind:     p.Kind,
		Metadata: p.Metadata,
		Input:    p.Input,
		Steps:    subSteps,
	}

	return resumePipeline
}

// executeResumedPipeline executes the resumed pipeline starting from the target step
func (r *ResumeManager) executeResumedPipeline(ctx context.Context, execution *PipelineExecution, fromStep string) error {
	validator := &DAGValidator{}
	pipelineID := execution.Status.ID
	pipelineName := execution.Pipeline.Metadata.Name

	// Validate the subpipeline DAG
	if err := validator.ValidateDAG(execution.Pipeline); err != nil {
		return r.errors.FormatPhaseFailureError(fromStep, fmt.Errorf("invalid resume pipeline DAG: %w", err), pipelineName)
	}

	// Get topologically sorted steps starting from target
	sortedSteps, err := validator.TopologicalSort(execution.Pipeline)
	if err != nil {
		return r.errors.FormatPhaseFailureError(fromStep, fmt.Errorf("failed to sort resume pipeline: %w", err), pipelineName)
	}

	// Execute each step in order
	for _, step := range sortedSteps {
		select {
		case <-ctx.Done():
			return fmt.Errorf("pipeline execution cancelled: %w", ctx.Err())

		default:
			execution.Status.CurrentStep = step.ID

			if r.executor.emitter != nil {
				r.executor.emitter.Emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "started",
					Message:    fmt.Sprintf("Starting step %s (resumed from %s)", step.ID, fromStep),
				})
			}

			// Execute the step (reuse existing step execution logic)
			if err := r.executeStep(ctx, execution, step); err != nil {
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
				return r.errors.FormatPhaseFailureError(step.ID, err, pipelineName)
			}

			execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
		}
	}

	// Mark pipeline as completed
	execution.Status.State = StateCompleted
	now := time.Now()
	execution.Status.CompletedAt = &now

	if r.executor.emitter != nil {
		r.executor.emitter.Emit(event.Event{
			Timestamp:      now,
			PipelineID:     pipelineID,
			State:          "completed",
			Message:        fmt.Sprintf("Pipeline completed successfully (resumed from %s)", fromStep),
			CompletedSteps: len(execution.Status.CompletedSteps),
			TotalSteps:     len(sortedSteps),
			DurationMs:     now.Sub(execution.Status.StartedAt).Milliseconds(),
		})
	}

	return nil
}

// executeStep executes a single pipeline step by delegating to the underlying executor.
func (r *ResumeManager) executeStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	if r.executor == nil {
		return fmt.Errorf("pipeline executor is not configured")
	}

	return r.executor.executeStep(ctx, execution, step)
}

// getAvailableSteps returns a formatted string of available steps in the pipeline
func (r *ResumeManager) getAvailableSteps(p *Pipeline) string {
	steps := make([]string, len(p.Steps))
	for i, step := range p.Steps {
		steps[i] = step.ID
	}

	if len(steps) == 0 {
		return "none"
	}

	result := ""
	for i, step := range steps {
		if i > 0 {
			result += ", "
		}
		result += "'" + step + "'"
	}
	return result
}

// ValidateResumePoint validates that resuming from a specific step is safe and valid
func (r *ResumeManager) ValidateResumePoint(p *Pipeline, fromStep string) error {
	// Check step exists
	var targetStep *Step
	for i := range p.Steps {
		if p.Steps[i].ID == fromStep {
			targetStep = &p.Steps[i]
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("step '%s' not found in pipeline", fromStep)
	}

	// Validate phase sequence for prototype pipeline
	if err := r.validator.ValidatePhaseSequence(p, fromStep); err != nil {
		return fmt.Errorf("invalid resume point: %w", err)
	}

	// Check for workspace conflicts
	workspaceID := fmt.Sprintf("%s/%s", p.Metadata.Name, fromStep)
	if r.concurrency.IsWorkspaceInUse(workspaceID) {
		return fmt.Errorf("workspace for step '%s' is currently in use", fromStep)
	}

	return nil
}

// GetRecommendedResumePoint suggests the best step to resume from based on current state
func (r *ResumeManager) GetRecommendedResumePoint(p *Pipeline) (string, error) {
	if p.Metadata.Name != "prototype" {
		return "", fmt.Errorf("resume point recommendation only available for prototype pipeline")
	}

	workspaceRoot := fmt.Sprintf(".wave/workspaces/%s", p.Metadata.Name)

	// Check phases in forward order to find the first incomplete phase
	prototypePhasesOrder := []string{"spec", "docs", "dummy", "implement"}

	for _, phase := range prototypePhasesOrder {
		phaseWorkspace := filepath.Join(workspaceRoot, phase)
		if err := r.validator.validatePhaseCompletion(phase, phaseWorkspace); err != nil {
			return phase, nil
		}
	}

	// All phases complete, suggest implement phase for any additional work
	return "implement", nil
}
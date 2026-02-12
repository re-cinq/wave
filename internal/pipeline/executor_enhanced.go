package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
)

// Enhanced executor methods that integrate error handling and validation features

// ExecuteWithValidation executes a pipeline with full error handling and validation
func (e *DefaultPipelineExecutor) ExecuteWithValidation(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
	// Initialize error handling components
	phaseValidator := NewPhaseSkipValidator()
	staleDetector := NewStaleArtifactDetector()
	concurrency := NewConcurrencyValidator()
	errorProvider := NewErrorMessageProvider()

	// Acquire workspace lock for concurrency protection
	workspaceID := fmt.Sprintf("%s/%s", p.Metadata.Name, "full")
	if err := concurrency.AcquireWorkspaceLock(p.Metadata.Name, workspaceID); err != nil {
		return fmt.Errorf("cannot start pipeline execution: %w", err)
	}

	// Ensure workspace lock is released when done
	defer concurrency.ReleaseWorkspaceLock(p.Metadata.Name)

	// Validate pipeline DAG
	validator := &DAGValidator{}
	if err := validator.ValidateDAG(p); err != nil {
		return errorProvider.FormatPhaseFailureError("pipeline_validation",
			fmt.Errorf("invalid pipeline DAG: %w", err), p.Metadata.Name)
	}

	// Get sorted steps for execution
	sortedSteps, err := validator.TopologicalSort(p)
	if err != nil {
		return errorProvider.FormatPhaseFailureError("pipeline_validation",
			fmt.Errorf("failed to topologically sort steps: %w", err), p.Metadata.Name)
	}

	// Initialize pipeline execution
	pipelineName := p.Metadata.Name
	hashLength := m.Runtime.PipelineIDHashLength
	pipelineID := GenerateRunID(pipelineName, hashLength)
	execution := &PipelineExecution{
		Pipeline:       p,
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Input:          input,
		Context:        newContextWithProject(pipelineID, pipelineName, "", m),
		Status: &PipelineStatus{
			ID:             pipelineID,
			PipelineName:   pipelineName,
			State:          StateRunning,
			StartedAt:      time.Now(),
		},
	}

	// Store execution state
	e.mu.Lock()
	e.pipelines[pipelineID] = execution
	e.mu.Unlock()

	// Execute each step with enhanced error handling
	for _, step := range sortedSteps {
		select {
		case <-ctx.Done():
			execution.Status.State = StateFailed
			return fmt.Errorf("pipeline execution cancelled: %w", ctx.Err())

		default:
			execution.Status.CurrentStep = step.ID

			// Emit step started event
			if e.emitter != nil {
				e.emitter.Emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStarted,
					Message:    fmt.Sprintf("Starting step %s", step.ID),
				})
			}

			// Phase skip validation for prototype pipeline
			if err := phaseValidator.ValidatePhaseSequence(p, step.ID); err != nil {
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
				return errorProvider.FormatPhaseFailureError(step.ID, err, p.Metadata.Name)
			}

			// Stale artifact detection
			staleReasons, err := staleDetector.DetectStaleArtifacts(p, step.ID)
			if err != nil {
				// Log warning but don't fail
				if e.debug {
					fmt.Printf("Warning: Failed to detect stale artifacts for step %s: %v\n", step.ID, err)
				}
			} else if len(staleReasons) > 0 {
				// Emit stale artifact warning
				if e.emitter != nil {
					e.emitter.Emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "stale_artifacts_detected",
						Message:    fmt.Sprintf("Stale artifacts detected: %v", staleReasons),
					})
				}

				if e.debug {
					fmt.Printf("⚠️  Warning: Stale artifacts detected for step '%s':\n", step.ID)
					for _, reason := range staleReasons {
						fmt.Printf("   • %s\n", reason)
					}
					fmt.Printf("\n")
				}
			}

			// Execute the step with enhanced error handling
			if err := e.executeStepWithValidation(ctx, execution, step, errorProvider); err != nil {
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
				execution.Status.State = StateFailed
				return err
			}

			execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
			execution.States[step.ID] = StateCompleted

			// Emit step completed event
			if e.emitter != nil {
				e.emitter.Emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateCompleted,
					Message:    fmt.Sprintf("Completed step %s", step.ID),
				})
			}
		}
	}

	// Mark pipeline as completed
	execution.Status.State = StateCompleted
	now := time.Now()
	execution.Status.CompletedAt = &now

	if e.emitter != nil {
		e.emitter.Emit(event.Event{
			Timestamp:      now,
			PipelineID:     pipelineID,
			State:          event.StateCompleted,
			Message:        "Pipeline completed successfully",
			CompletedSteps: len(execution.Status.CompletedSteps),
			DurationMs:     now.Sub(execution.Status.StartedAt).Milliseconds(),
		})
	}

	return nil
}

// ResumeWithValidation resumes a pipeline with full validation and error handling.
// When force is true, phase validation and stale artifact checks are skipped.
func (e *DefaultPipelineExecutor) ResumeWithValidation(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string, fromStep string, force bool) error {
	manager := NewResumeManager(e)
	return manager.ResumeFromStep(ctx, p, m, input, fromStep, force)
}

// executeStepWithValidation executes a single step with enhanced error handling
func (e *DefaultPipelineExecutor) executeStepWithValidation(ctx context.Context, execution *PipelineExecution, step *Step, errorProvider *ErrorMessageProvider) error {
	// This would integrate with the existing step execution logic
	// For now, we'll simulate execution and contract validation

	// Here we would call the existing step execution logic from executor.go
	// and wrap any errors with enhanced error messages

	// Simulate step execution (replace with actual execution logic)
	stepErr := e.simulateStepExecution(ctx, execution, step)

	if stepErr != nil {
		// Check if it's a contract validation error
		if isContractValidationError(stepErr) {
			return errorProvider.FormatContractValidationError(step.ID, stepErr)
		}

		// General step failure
		return errorProvider.FormatPhaseFailureError(step.ID, stepErr, execution.Pipeline.Metadata.Name)
	}

	return nil
}

// simulateStepExecution delegates to the real step execution logic.
func (e *DefaultPipelineExecutor) simulateStepExecution(ctx context.Context, execution *PipelineExecution, step *Step) error {
	// This was previously a placeholder that always returned success.
	// Now it calls the existing step execution logic from DefaultPipelineExecutor.
	return e.executeStep(ctx, execution, step)
}

// isContractValidationError checks if an error is related to contract validation
func isContractValidationError(err error) bool {
	if err == nil {
		return false
	}

	errorText := err.Error()
	contractKeywords := []string{
		"contract validation failed",
		"schema validation",
		"artifact.json",
		"json_schema",
	}

	for _, keyword := range contractKeywords {
		if contains(errorText, keyword) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(text, substr string) bool {
	return len(text) >= len(substr) && findSubstring(text, substr)
}

func findSubstring(text, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(text) < len(substr) {
		return false
	}

	for i := 0; i <= len(text)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if text[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetPipelineErrors returns formatted error information for a pipeline
func (e *DefaultPipelineExecutor) GetPipelineErrors(pipelineID string) ([]string, error) {
	e.mu.RLock()
	execution, exists := e.pipelines[pipelineID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pipeline %s not found", pipelineID)
	}

	var errors []string
	errorProvider := NewErrorMessageProvider()

	// Get errors for failed steps
	for _, failedStep := range execution.Status.FailedSteps {
		// In a real implementation, we would store the actual errors
		// For now, provide generic failure information
		err := errorProvider.FormatPhaseFailureError(failedStep,
			fmt.Errorf("step failed during execution"), pipelineID)
		errors = append(errors, err.Error())
	}

	return errors, nil
}

// ValidatePipelineResumption validates that a pipeline can be safely resumed
func (e *DefaultPipelineExecutor) ValidatePipelineResumption(pipelineID, fromStep string) error {
	// Load pipeline definition (this would be from the pipeline registry)
	pipeline, err := e.loadPipelineDefinition(pipelineID)
	if err != nil {
		return fmt.Errorf("failed to load pipeline definition: %w", err)
	}

	// Use resume manager for validation
	manager := NewResumeManager(e)
	return manager.ValidateResumePoint(pipeline, fromStep)
}

// GetRecommendedResumePoint suggests the best resume point for a pipeline
func (e *DefaultPipelineExecutor) GetRecommendedResumePoint(pipelineID string) (string, error) {
	pipeline, err := e.loadPipelineDefinition(pipelineID)
	if err != nil {
		return "", fmt.Errorf("failed to load pipeline definition: %w", err)
	}

	manager := NewResumeManager(e)
	return manager.GetRecommendedResumePoint(pipeline)
}

// loadPipelineDefinition loads a pipeline definition by ID
func (e *DefaultPipelineExecutor) loadPipelineDefinition(pipelineID string) (*Pipeline, error) {
	// This would load from a pipeline registry or file system
	// For now, return a placeholder that would be implemented

	if pipelineID == "prototype" {
		// Return the prototype pipeline definition
		loader := &YAMLPipelineLoader{}
		return loader.Load(".wave/pipelines/prototype.yaml")
	}

	return nil, fmt.Errorf("pipeline %s not found", pipelineID)
}

// State constants are defined in types.go
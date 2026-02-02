package pipeline

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

type PipelineExecutor interface {
	Execute(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error
	Resume(ctx context.Context, pipelineID string, fromStep string) error
	GetStatus(pipelineID string) (*PipelineStatus, error)
}

type PipelineStatus struct {
	ID             string
	State          string
	CurrentStep    string
	CompletedSteps []string
	FailedSteps    []string
	StartedAt      time.Time
	CompletedAt    *time.Time
}

type DefaultPipelineExecutor struct {
	runner    adapter.AdapterRunner
	pipelines map[string]*PipelineExecution
	mu        sync.RWMutex
}

type PipelineExecution struct {
	Pipeline *Pipeline
	Manifest *manifest.Manifest
	States   map[string]string
	Results  map[string]map[string]interface{}
	Input    string
	Status   *PipelineStatus
}

func NewDefaultPipelineExecutor(runner adapter.AdapterRunner) *DefaultPipelineExecutor {
	return &DefaultPipelineExecutor{
		runner:    runner,
		pipelines: make(map[string]*PipelineExecution),
	}
}

func (e *DefaultPipelineExecutor) Execute(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
	validator := &DAGValidator{}
	if err := validator.ValidateDAG(p); err != nil {
		return fmt.Errorf("invalid pipeline DAG: %w", err)
	}

	sortedSteps, err := validator.TopologicalSort(p)
	if err != nil {
		return fmt.Errorf("failed to topologically sort steps: %w", err)
	}

	pipelineID := p.Metadata.Name
	execution := &PipelineExecution{
		Pipeline: p,
		Manifest: m,
		States:   make(map[string]string),
		Results:  make(map[string]map[string]interface{}),
		Input:    input,
		Status: &PipelineStatus{
			ID:             pipelineID,
			State:          StatePending,
			CompletedSteps: []string{},
			FailedSteps:    []string{},
			StartedAt:      time.Now(),
		},
	}

	for _, step := range p.Steps {
		execution.States[step.ID] = StatePending
	}

	e.mu.Lock()
	e.pipelines[pipelineID] = execution
	e.mu.Unlock()

	execution.Status.State = StateRunning

	for _, step := range sortedSteps {
		if err := e.executeStep(ctx, execution, step); err != nil {
			execution.Status.State = StateFailed
			execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
			return fmt.Errorf("step %q failed: %w", step.ID, err)
		}
		execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
	}

	now := time.Now()
	execution.Status.CompletedAt = &now
	execution.Status.State = StateCompleted

	return nil
}

func (e *DefaultPipelineExecutor) executeStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	execution.States[step.ID] = StateRunning
	execution.Status.CurrentStep = step.ID

	maxRetries := step.Handover.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			execution.States[step.ID] = StateRetrying
			time.Sleep(time.Second * time.Duration(attempt))
		}

		if err := e.runStepExecution(ctx, execution, step); err != nil {
			lastErr = err
			if attempt < maxRetries {
				continue
			}
			return lastErr
		}

		if step.Handover.Contract.Validate {
			if err := e.validateStepOutput(ctx, execution, step); err != nil {
				lastErr = err
				if attempt < maxRetries {
					continue
				}
				return lastErr
			}
		}

		execution.States[step.ID] = StateCompleted
		return nil
	}

	return lastErr
}

func (e *DefaultPipelineExecutor) runStepExecution(ctx context.Context, execution *PipelineExecution, step *Step) error {
	persona := execution.Manifest.GetPersona(step.Persona)
	if persona == nil {
		return fmt.Errorf("persona %q not found in manifest", step.Persona)
	}

	adapterDef := execution.Manifest.GetAdapter(persona.Adapter)
	if adapterDef == nil {
		return fmt.Errorf("adapter %q not found in manifest", persona.Adapter)
	}

	if err := e.injectArtifacts(ctx, execution, step); err != nil {
		return fmt.Errorf("failed to inject artifacts: %w", err)
	}

	timeout := execution.Manifest.Runtime.GetDefaultTimeout()

	cfg := adapter.AdapterRunConfig{
		Adapter:       adapterDef.Binary,
		Persona:       step.Persona,
		WorkspacePath: step.Workspace.Root,
		Prompt:        step.Exec.Source,
		Timeout:       timeout,
	}

	result, err := e.runner.Run(ctx, cfg)
	if err != nil {
		return fmt.Errorf("adapter execution failed: %w", err)
	}

	output := make(map[string]interface{})
	stdoutData, err := io.ReadAll(result.Stdout)
	if err == nil {
		output["stdout"] = string(stdoutData)
	}
	output["exit_code"] = result.ExitCode
	output["tokens_used"] = result.TokensUsed

	execution.Results[step.ID] = output

	return nil
}

func (e *DefaultPipelineExecutor) buildStepInput(execution *PipelineExecution, step *Step) string {
	input := execution.Input

	for _, dep := range step.Dependencies {
		if depResults, exists := execution.Results[dep]; exists {
			input = input + fmt.Sprintf("\n\n%s results: %v", dep, depResults)
		}
	}

	for _, artifactRef := range step.Memory.InjectArtifacts {
		if depResults, exists := execution.Results[artifactRef.Step]; exists {
			if artifactData, exists := depResults[artifactRef.Artifact]; exists {
				input = input + fmt.Sprintf("\n\n%s: %v", artifactRef.As, artifactData)
			}
		}
	}

	return input
}

func (e *DefaultPipelineExecutor) injectArtifacts(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return nil
}

func (e *DefaultPipelineExecutor) validateStepOutput(ctx context.Context, execution *PipelineExecution, step *Step) error {
	return nil
}

func (e *DefaultPipelineExecutor) Resume(ctx context.Context, pipelineID string, fromStep string) error {
	e.mu.RLock()
	execution, exists := e.pipelines[pipelineID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pipeline %q not found", pipelineID)
	}

	validator := &DAGValidator{}
	sortedSteps, err := validator.TopologicalSort(execution.Pipeline)
	if err != nil {
		return fmt.Errorf("failed to topologically sort steps: %w", err)
	}

	var startFromStep *Step
	for _, step := range sortedSteps {
		if step.ID == fromStep {
			startFromStep = step
			break
		}
	}

	if startFromStep == nil {
		return fmt.Errorf("step %q not found in pipeline", fromStep)
	}

	execution.Status.State = StateRunning

	resuming := false
	for _, step := range sortedSteps {
		if !resuming && step.ID == fromStep {
			resuming = true
		}

		if resuming {
			if execution.States[step.ID] != StateCompleted {
				if err := e.executeStep(ctx, execution, step); err != nil {
					execution.Status.State = StateFailed
					execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
					return fmt.Errorf("step %q failed: %w", step.ID, err)
				}
				execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
			}
		}
	}

	now := time.Now()
	execution.Status.CompletedAt = &now
	execution.Status.State = StateCompleted

	return nil
}

func (e *DefaultPipelineExecutor) GetStatus(pipelineID string) (*PipelineStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	execution, exists := e.pipelines[pipelineID]
	if !exists {
		return nil, fmt.Errorf("pipeline %q not found", pipelineID)
	}

	return execution.Status, nil
}

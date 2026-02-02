package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
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
	runner       adapter.AdapterRunner
	emitter      event.EventEmitter
	store        state.StateStore
	logger       audit.AuditLogger
	wsManager    workspace.WorkspaceManager
	relayMonitor *relay.RelayMonitor
	pipelines    map[string]*PipelineExecution
	mu           sync.RWMutex
	debug        bool
}

type ExecutorOption func(*DefaultPipelineExecutor)

func WithEmitter(e event.EventEmitter) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.emitter = e }
}

func WithStateStore(s state.StateStore) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.store = s }
}

func WithAuditLogger(l audit.AuditLogger) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.logger = l }
}

func WithDebug(debug bool) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.debug = debug }
}

func WithWorkspaceManager(w workspace.WorkspaceManager) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.wsManager = w }
}

func WithRelayMonitor(r *relay.RelayMonitor) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.relayMonitor = r }
}

type PipelineExecution struct {
	Pipeline       *Pipeline
	Manifest       *manifest.Manifest
	States         map[string]string
	Results        map[string]map[string]interface{}
	ArtifactPaths  map[string]string // "stepID:artifactName" -> filesystem path
	WorkspacePaths map[string]string // stepID -> workspace path
	Input          string
	Status         *PipelineStatus
}

func NewDefaultPipelineExecutor(runner adapter.AdapterRunner, opts ...ExecutorOption) *DefaultPipelineExecutor {
	ex := &DefaultPipelineExecutor{
		runner:    runner,
		pipelines: make(map[string]*PipelineExecution),
	}
	for _, opt := range opts {
		opt(ex)
	}
	return ex
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
		Pipeline:       p,
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Input:          input,
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

	if e.store != nil {
		e.store.SavePipelineState(pipelineID, StateRunning, input)
	}

	execution.Status.State = StateRunning

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      "started",
		Message:    fmt.Sprintf("input=%q steps=%d", input, len(p.Steps)),
	})

	// Ensure workspace root exists
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	os.MkdirAll(filepath.Join(wsRoot, pipelineID), 0755)

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      "started",
		Message:    fmt.Sprintf("workspace root: %s/%s/", wsRoot, pipelineID),
	})

	for _, step := range sortedSteps {
		if err := e.executeStep(ctx, execution, step); err != nil {
			execution.Status.State = StateFailed
			execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
			if e.store != nil {
				e.store.SavePipelineState(pipelineID, StateFailed, input)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "failed",
				Message:    err.Error(),
			})
			return fmt.Errorf("step %q failed: %w", step.ID, err)
		}
		execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
	}

	now := time.Now()
	execution.Status.CompletedAt = &now
	execution.Status.State = StateCompleted

	if e.store != nil {
		e.store.SavePipelineState(pipelineID, StateCompleted, input)
	}

	elapsed := time.Since(execution.Status.StartedAt).Milliseconds()
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      "completed",
		DurationMs: elapsed,
		Message:    fmt.Sprintf("%d steps completed", len(p.Steps)),
	})

	return nil
}

func (e *DefaultPipelineExecutor) executeStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Pipeline.Metadata.Name
	execution.States[step.ID] = StateRunning
	execution.Status.CurrentStep = step.ID

	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateRunning, "")
	}

	// Check if this step uses a matrix strategy
	if step.Strategy != nil && step.Strategy.Type == "matrix" {
		return e.executeMatrixStep(ctx, execution, step)
	}

	maxRetries := step.Handover.MaxRetries
	if maxRetries == 0 {
		if step.Handover.Contract.MaxRetries > 0 {
			maxRetries = step.Handover.Contract.MaxRetries
		} else {
			maxRetries = 1
		}
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			execution.States[step.ID] = StateRetrying
			if e.store != nil {
				e.store.SaveStepState(pipelineID, step.ID, state.StateRetrying, "")
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "retrying",
				Message:    fmt.Sprintf("attempt %d/%d", attempt, maxRetries),
			})
			time.Sleep(time.Second * time.Duration(attempt))
		}

		if err := e.runStepExecution(ctx, execution, step); err != nil {
			lastErr = err
			if attempt < maxRetries {
				continue
			}
			if e.store != nil {
				e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
			}
			return lastErr
		}

		execution.States[step.ID] = StateCompleted
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
		}
		return nil
	}

	return lastErr
}

// executeMatrixStep handles steps with matrix strategy using fan-out execution.
func (e *DefaultPipelineExecutor) executeMatrixStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Pipeline.Metadata.Name

	matrixExecutor := NewMatrixExecutor(e)
	err := matrixExecutor.Execute(ctx, execution, step)

	if err != nil {
		execution.States[step.ID] = StateFailed
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.States[step.ID] = StateCompleted
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}
	return nil
}

func (e *DefaultPipelineExecutor) runStepExecution(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Pipeline.Metadata.Name

	persona := execution.Manifest.GetPersona(step.Persona)
	if persona == nil {
		return fmt.Errorf("persona %q not found in manifest", step.Persona)
	}

	adapterDef := execution.Manifest.GetAdapter(persona.Adapter)
	if adapterDef == nil {
		return fmt.Errorf("adapter %q not found in manifest", persona.Adapter)
	}

	// Create workspace under .wave/workspaces/<pipeline>/<step>/
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	execution.WorkspacePaths[step.ID] = workspacePath

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "running",
		Persona:    step.Persona,
		Message:    fmt.Sprintf("Starting %s persona in %s", step.Persona, workspacePath),
	})

	// Inject artifacts from dependencies
	if err := e.injectArtifacts(execution, step, workspacePath); err != nil {
		return fmt.Errorf("failed to inject artifacts: %w", err)
	}

	prompt := e.buildStepPrompt(execution, step)

	if e.logger != nil {
		e.logger.LogToolCall(pipelineID, step.ID, "adapter.Run", fmt.Sprintf("persona=%s prompt_len=%d", step.Persona, len(prompt)))
	}

	timeout := execution.Manifest.Runtime.GetDefaultTimeout()

	// Load system prompt from persona file
	systemPrompt := ""
	if persona.SystemPromptFile != "" {
		if data, err := os.ReadFile(persona.SystemPromptFile); err == nil {
			systemPrompt = string(data)
		}
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:       adapterDef.Binary,
		Persona:       step.Persona,
		WorkspacePath: workspacePath,
		Prompt:        prompt,
		SystemPrompt:  systemPrompt,
		Timeout:       timeout,
		Temperature:   persona.Temperature,
		AllowedTools:  persona.Permissions.AllowedTools,
		DenyTools:     persona.Permissions.Deny,
		OutputFormat:  adapterDef.OutputFormat,
		Debug:         e.debug,
	}

	stepStart := time.Now()
	result, err := e.runner.Run(ctx, cfg)
	if err != nil {
		return fmt.Errorf("adapter execution failed: %w", err)
	}

	stepDuration := time.Since(stepStart).Milliseconds()

	output := make(map[string]interface{})
	stdoutData, err := io.ReadAll(result.Stdout)
	if err == nil {
		output["stdout"] = string(stdoutData)
	}
	output["exit_code"] = result.ExitCode
	output["tokens_used"] = result.TokensUsed
	output["workspace"] = workspacePath

	execution.Results[step.ID] = output

	// Write output artifacts to workspace
	// Use ResultContent if available (extracted from adapter response)
	// Don't fall back to raw stdout as it contains JSON wrapper, not actual content
	if result.ResultContent != "" {
		artifactContent := []byte(result.ResultContent)
		e.writeOutputArtifacts(execution, step, workspacePath, artifactContent)
	} else {
		// Skip writing artifacts when ResultContent is empty to avoid overwriting
		// existing artifacts with empty content during relay compaction or parsing failures
		if e.debug {
			fmt.Printf("[DEBUG] Warning: ResultContent is empty, skipping artifact write to preserve existing content\n")
		}
	}

	// Check relay/compaction threshold (FR-009)
	if err := e.checkRelayCompaction(ctx, execution, step, result.TokensUsed, workspacePath, string(stdoutData)); err != nil {
		// Log the error but don't fail the step - compaction is best-effort
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    fmt.Sprintf("relay compaction failed: %v", err),
		})
	}

	// Validate handover contract if configured
	if step.Handover.Contract.Type != "" {
		contractCfg := contract.ContractConfig{
			Type:       step.Handover.Contract.Type,
			Source:     step.Handover.Contract.Source,
			Schema:     step.Handover.Contract.Schema,
			SchemaPath: step.Handover.Contract.SchemaPath,
			Command:    step.Handover.Contract.Command,
			StrictMode: step.Handover.Contract.MustPass,
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "validating",
			Message:    fmt.Sprintf("Validating %s contract", step.Handover.Contract.Type),
		})

		if err := contract.Validate(contractCfg, workspacePath); err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "contract_failed",
				Message:    err.Error(),
			})
			return fmt.Errorf("contract validation failed: %w", err)
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "contract_passed",
			Message:    fmt.Sprintf("%s contract validated", step.Handover.Contract.Type),
		})
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "completed",
		Persona:    step.Persona,
		DurationMs: stepDuration,
		TokensUsed: result.TokensUsed,
		Artifacts:  result.Artifacts,
		Message:    fmt.Sprintf("%dk tokens", result.TokensUsed/1000),
	})

	return nil
}

func (e *DefaultPipelineExecutor) createStepWorkspace(execution *PipelineExecution, step *Step) (string, error) {
	pipelineID := execution.Pipeline.Metadata.Name
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	if e.wsManager != nil && len(step.Workspace.Mount) > 0 {
		templateVars := map[string]string{
			"pipeline_id": pipelineID,
			"step_id":     step.ID,
		}
		return e.wsManager.Create(workspace.WorkspaceConfig{
			Root:  wsRoot,
			Mount: toWorkspaceMounts(step.Workspace.Mount),
		}, templateVars)
	}

	// Create directory under .wave/workspaces/<pipeline>/<step>/
	wsPath := filepath.Join(wsRoot, pipelineID, step.ID)
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}
	return wsPath, nil
}

func toWorkspaceMounts(mounts []Mount) []workspace.Mount {
	result := make([]workspace.Mount, len(mounts))
	for i, m := range mounts {
		result[i] = workspace.Mount{
			Source: m.Source,
			Target: m.Target,
			Mode:   m.Mode,
		}
	}
	return result
}

func (e *DefaultPipelineExecutor) buildStepPrompt(execution *PipelineExecution, step *Step) string {
	prompt := step.Exec.Source

	// Replace {{ input }} template variable
	if execution.Input != "" {
		for _, pattern := range []string{"{{ input }}", "{{input}}", "{{ input}}", "{{input }}"} {
			for idx := indexOf(prompt, pattern); idx != -1; idx = indexOf(prompt, pattern) {
				prompt = prompt[:idx] + execution.Input + prompt[idx+len(pattern):]
			}
		}
	}

	return prompt
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (e *DefaultPipelineExecutor) injectArtifacts(execution *PipelineExecution, step *Step, workspacePath string) error {
	if len(step.Memory.InjectArtifacts) == 0 {
		return nil
	}

	artifactsDir := filepath.Join(workspacePath, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts dir: %w", err)
	}

	pipelineID := execution.Pipeline.Metadata.Name

	for _, ref := range step.Memory.InjectArtifacts {
		artName := ref.As
		if artName == "" {
			artName = ref.Artifact
		}
		destPath := filepath.Join(artifactsDir, artName)

		// Try registered artifact path first
		key := ref.Step + ":" + ref.Artifact
		if artifactPath, ok := execution.ArtifactPaths[key]; ok {
			if srcData, err := os.ReadFile(artifactPath); err == nil {
				os.WriteFile(destPath, srcData, 0644)
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "running",
					Message:    fmt.Sprintf("injected artifact %s from %s (%s)", artName, ref.Step, artifactPath),
				})
				continue
			}
		}

		// Fallback: use stdout from previous step
		if result, exists := execution.Results[ref.Step]; exists {
			if stdout, ok := result["stdout"].(string); ok {
				os.WriteFile(destPath, []byte(stdout), 0644)
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "running",
					Message:    fmt.Sprintf("injected artifact %s from step %s stdout", artName, ref.Step),
				})
			}
		}
	}

	return nil
}

func (e *DefaultPipelineExecutor) writeOutputArtifacts(execution *PipelineExecution, step *Step, workspacePath string, stdout []byte) {
	for _, art := range step.OutputArtifacts {
		artPath := filepath.Join(workspacePath, art.Path)
		os.MkdirAll(filepath.Dir(artPath), 0755)
		os.WriteFile(artPath, stdout, 0644)
		key := step.ID + ":" + art.Name
		execution.ArtifactPaths[key] = artPath
	}
}

func (e *DefaultPipelineExecutor) emit(ev event.Event) {
	if e.emitter != nil {
		e.emitter.Emit(ev)
	}
}

// checkRelayCompaction monitors token usage and triggers compaction when threshold is exceeded.
// This implements FR-009: System MUST monitor agent context utilization and trigger relay/compaction.
func (e *DefaultPipelineExecutor) checkRelayCompaction(ctx context.Context, execution *PipelineExecution, step *Step, tokensUsed int, workspacePath string, chatHistory string) error {
	if e.relayMonitor == nil {
		return nil // No relay monitor configured
	}

	relayConfig := execution.Manifest.Runtime.Relay
	thresholdPercent := relayConfig.TokenThresholdPercent

	// Allow step-level override via handover.compaction.trigger
	if step.Handover.Compaction.Trigger != "" {
		// Parse trigger like "token_limit_80%"
		var pct int
		if _, err := fmt.Sscanf(step.Handover.Compaction.Trigger, "token_limit_%d%%", &pct); err == nil {
			thresholdPercent = pct
		}
	}

	if thresholdPercent == 0 {
		// No threshold configured, skip compaction check
		return nil
	}

	// Check if we should compact based on token usage
	if !e.relayMonitor.ShouldCompact(tokensUsed, thresholdPercent) {
		return nil
	}

	pipelineID := execution.Pipeline.Metadata.Name

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "compacting",
		TokensUsed: tokensUsed,
		Message:    fmt.Sprintf("Token threshold exceeded (%d tokens, %d%% threshold), triggering compaction", tokensUsed, thresholdPercent),
	})

	// Get summarizer persona from config (step-level takes precedence over runtime-level)
	summarizerName := relayConfig.SummarizerPersona
	if step.Handover.Compaction.Persona != "" {
		summarizerName = step.Handover.Compaction.Persona
	}
	if summarizerName == "" {
		summarizerName = "summarizer" // Default fallback
	}

	// Load summarizer persona for system prompt
	summarizerPersona := execution.Manifest.GetPersona(summarizerName)
	systemPrompt := ""
	compactPrompt := "Summarize this conversation history concisely, preserving key context, decisions, and progress:"

	if summarizerPersona != nil {
		if summarizerPersona.SystemPromptFile != "" {
			if data, err := os.ReadFile(summarizerPersona.SystemPromptFile); err == nil {
				systemPrompt = string(data)
			}
		}
	}

	// Trigger compaction
	summary, err := e.relayMonitor.Compact(ctx, chatHistory, systemPrompt, compactPrompt, workspacePath)
	if err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "compacted",
		Message:    fmt.Sprintf("Checkpoint written to %s/checkpoint.md (%d chars)", workspacePath, len(summary)),
	})

	if e.logger != nil {
		e.logger.LogToolCall(pipelineID, step.ID, "relay.Compact", fmt.Sprintf("tokens=%d summary_len=%d persona=%s", tokensUsed, len(summary), summarizerName))
	}

	return nil
}

// injectCheckpointIfExists checks for a checkpoint.md file in the workspace and
// prepends checkpoint context to the prompt if found.
func (e *DefaultPipelineExecutor) injectCheckpointIfExists(workspacePath string, prompt string) string {
	checkpointPrompt, err := relay.InjectCheckpointPrompt(workspacePath)
	if err != nil {
		// No checkpoint found or error reading it - that's fine, just use original prompt
		return prompt
	}

	// Prepend checkpoint context to the prompt
	return checkpointPrompt + "\n\n" + prompt
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

	found := false
	for _, step := range sortedSteps {
		if step.ID == fromStep {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("step %q not found in pipeline", fromStep)
	}

	execution.Status.State = StateRunning

	resuming := false
	for _, step := range sortedSteps {
		if !resuming && step.ID == fromStep {
			resuming = true
		}
		if resuming && execution.States[step.ID] != StateCompleted {
			if err := e.executeStep(ctx, execution, step); err != nil {
				execution.Status.State = StateFailed
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
				return fmt.Errorf("step %q failed: %w", step.ID, err)
			}
			execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
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

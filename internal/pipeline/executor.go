package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/security"
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
	runner         adapter.AdapterRunner
	emitter        event.EventEmitter
	store          state.StateStore
	logger         audit.AuditLogger
	wsManager      workspace.WorkspaceManager
	relayMonitor   *relay.RelayMonitor
	pipelines      map[string]*PipelineExecution
	mu             sync.RWMutex
	debug          bool
	// Security infrastructure
	securityConfig *security.SecurityConfig
	pathValidator  *security.PathValidator
	inputSanitizer *security.InputSanitizer
	securityLogger *security.SecurityLogger
	// Deliverable tracking
	deliverableTracker *deliverable.Tracker
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
	Context        *PipelineContext  // Dynamic template variables
}

func NewDefaultPipelineExecutor(runner adapter.AdapterRunner, opts ...ExecutorOption) *DefaultPipelineExecutor {
	// Initialize security configuration with secure defaults
	securityConfig := security.DefaultSecurityConfig()
	securityLogger := security.NewSecurityLogger(securityConfig.LoggingEnabled)

	ex := &DefaultPipelineExecutor{
		runner:             runner,
		pipelines:          make(map[string]*PipelineExecution),
		securityConfig:     securityConfig,
		pathValidator:      security.NewPathValidator(*securityConfig, securityLogger),
		inputSanitizer:     security.NewInputSanitizer(*securityConfig, securityLogger),
		securityLogger:     securityLogger,
		deliverableTracker: deliverable.NewTracker(""),
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
	pipelineContext := NewPipelineContext(pipelineID, "")

	// Initialize deliverable tracker for this pipeline (only if not already set)
	if e.deliverableTracker == nil {
		e.deliverableTracker = deliverable.NewTracker(pipelineID)
	} else {
		// Update pipeline ID if tracker already exists
		e.deliverableTracker.SetPipelineID(pipelineID)
	}
	execution := &PipelineExecution{
		Pipeline:       p,
		Manifest:       m,
		States:         make(map[string]string),
		Results:        make(map[string]map[string]interface{}),
		ArtifactPaths:  make(map[string]string),
		WorkspacePaths: make(map[string]string),
		Input:          input,
		Context:        pipelineContext,
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
		Timestamp:      time.Now(),
		PipelineID:     pipelineID,
		State:          "started",
		Message:        fmt.Sprintf("input=%q steps=%d", input, len(p.Steps)),
		TotalSteps:     len(p.Steps),
		CompletedSteps: 0,
	})

	// Ensure workspace root exists and is clean for this pipeline run
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	pipelineWsPath := filepath.Join(wsRoot, pipelineID)
	// Clean previous run artifacts to ensure fresh state
	if err := os.RemoveAll(pipelineWsPath); err != nil {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			State:      "warning",
			Message:    fmt.Sprintf("failed to clean workspace: %v", err),
		})
	}
	if err := os.MkdirAll(pipelineWsPath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      "started",
		Message:    fmt.Sprintf("workspace root: %s/%s/", wsRoot, pipelineID),
	})

	for stepIdx, step := range sortedSteps {
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
			// Clean up failed pipeline from in-memory storage to prevent memory leak
			e.cleanupCompletedPipeline(pipelineID)
			return fmt.Errorf("step %q failed: %w", step.ID, err)
		}
		execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)

		// Emit overall pipeline progress after each step
		completedCount := stepIdx + 1
		e.emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     pipelineID,
			State:          "running",
			TotalSteps:     len(p.Steps),
			CompletedSteps: completedCount,
			Progress:       (completedCount * 100) / len(p.Steps),
			Message:        fmt.Sprintf("%d/%d steps completed", completedCount, len(p.Steps)),
		})
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

	// Clean up completed pipeline from in-memory storage to prevent memory leak
	e.cleanupCompletedPipeline(pipelineID)

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

		// Start progress ticker for smooth animation updates during step execution
		cancelTicker := e.startProgressTicker(ctx, pipelineID, step.ID)

		err := e.runStepExecution(ctx, execution, step)

		// Stop progress ticker when step completes
		cancelTicker()

		if err != nil {
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

		// Track deliverables from completed step
		e.trackStepDeliverables(execution, step)

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

	// Track deliverables from completed matrix step
	e.trackStepDeliverables(execution, step)

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
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "running",
		Persona:       step.Persona,
		Message:       fmt.Sprintf("Starting %s persona in %s", step.Persona, workspacePath),
		CurrentAction: "Initializing",
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
		Model:         persona.Model,
		AllowedTools:  persona.Permissions.AllowedTools,
		DenyTools:     persona.Permissions.Deny,
		OutputFormat:  adapterDef.OutputFormat,
		Debug:         e.debug,
	}

	// Emit step progress: executing
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       step.Persona,
		Progress:      25,
		CurrentAction: "Executing agent",
	})

	stepStart := time.Now()
	result, err := e.runner.Run(ctx, cfg)
	if err != nil {
		return fmt.Errorf("adapter execution failed: %w", err)
	}

	stepDuration := time.Since(stepStart).Milliseconds()

	// Emit step progress: processing results
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       step.Persona,
		Progress:      75,
		CurrentAction: "Processing results",
		TokensUsed:    result.TokensUsed,
	})

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
		// Resolve contract source path using pipeline context
		resolvedSource := execution.Context.ResolveContractSource(step.Handover.Contract)

		contractCfg := contract.ContractConfig{
			Type:       step.Handover.Contract.Type,
			Source:     resolvedSource,
			Schema:     step.Handover.Contract.Schema,
			SchemaPath: step.Handover.Contract.SchemaPath,
			Command:    step.Handover.Contract.Command,
			StrictMode: step.Handover.Contract.MustPass,
			MustPass:   step.Handover.Contract.MustPass,
			MaxRetries: step.Handover.Contract.MaxRetries,
		}

		e.emit(event.Event{
			Timestamp:       time.Now(),
			PipelineID:      pipelineID,
			StepID:          step.ID,
			State:           "validating",
			Message:         fmt.Sprintf("Validating %s contract", step.Handover.Contract.Type),
			CurrentAction:   "Validating contract",
			ValidationPhase: step.Handover.Contract.Type,
		})

		if err := contract.Validate(contractCfg, workspacePath); err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "contract_failed",
				Message:    err.Error(),
			})

			// Check if we should fail the step or allow soft failure
			if contractCfg.StrictMode {
				return fmt.Errorf("contract validation failed: %w", err)
			} else {
				// Soft failure: log the validation error but continue execution
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "contract_soft_failure",
					Message:    fmt.Sprintf("contract validation failed but continuing (must_pass: false): %s", err.Error()),
				})
			}
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
		// Update pipeline context with current step
		execution.Context.StepID = step.ID

		// Use pipeline context for template variables
		templateVars := execution.Context.ToTemplateVars()

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

	// Determine the input value to use (sanitized if provided, empty string if not)
	var sanitizedInput string
	if execution.Input != "" {
		// SECURITY FIX: Sanitize user input for prompt injection
		sanitizationRecord, tmpInput, sanitizeErr := e.inputSanitizer.SanitizeInput(execution.Input, "task_description")
		if sanitizeErr != nil {
			// Security violation detected - log and reject
			e.securityLogger.LogViolation(
				string(security.ViolationPromptInjection),
				string(security.SourceUserInput),
				fmt.Sprintf("User input sanitization failed for step %s", step.ID),
				security.SeverityCritical,
				true,
			)
			// In strict mode, this would cause the step to fail
			// For now, we'll use empty input to prevent the injection
			sanitizedInput = "[INPUT SANITIZED FOR SECURITY]"
		} else {
			// Log sanitization details
			if sanitizationRecord.ChangesDetected {
				e.securityLogger.LogViolation(
					string(security.ViolationPromptInjection),
					string(security.SourceUserInput),
					fmt.Sprintf("User input sanitized for step %s (risk score: %d)", step.ID, sanitizationRecord.RiskScore),
					security.SeverityMedium,
					false,
				)
			}
			sanitizedInput = tmpInput
		}
	} else {
		// No input provided - use empty string
		sanitizedInput = ""
	}

	// Replace template variables with sanitized input (even if empty)
	for _, pattern := range []string{"{{ input }}", "{{input}}", "{{ input}}", "{{input }}"} {
		for idx := indexOf(prompt, pattern); idx != -1; idx = indexOf(prompt, pattern) {
			prompt = prompt[:idx] + sanitizedInput + prompt[idx+len(pattern):]
		}
	}


	// Inject schema information for json_schema contracts with security validation
	if step.Handover.Contract.Type == "json_schema" {
		var schemaContent string
		var err error

		// Load schema from file or inline with security validation
		if step.Handover.Contract.SchemaPath != "" {
			// SECURITY FIX: Validate path for traversal attacks
			validationResult, pathErr := e.pathValidator.ValidatePath(step.Handover.Contract.SchemaPath)
			if pathErr != nil {
				// Security violation detected - log and use safe fallback
				e.securityLogger.LogViolation(
					string(security.ViolationPathTraversal),
					string(security.SourceSchemaPath),
					fmt.Sprintf("Schema path validation failed for step %s", step.ID),
					security.SeverityCritical,
					true,
				)
				err = fmt.Errorf("schema path validation failed: %w", pathErr)
			} else if validationResult.IsValid {
				// Path is safe - read the file using validated path
				data, readErr := os.ReadFile(validationResult.ValidatedPath)
				if readErr == nil {
					// SECURITY FIX: Sanitize schema content for prompt injection
					sanitizedContent, sanitizationActions, sanitizeErr := e.inputSanitizer.SanitizeSchemaContent(string(data))
					if sanitizeErr != nil {
						e.securityLogger.LogViolation(
							string(security.ViolationInputValidation),
							string(security.SourceSchemaPath),
							fmt.Sprintf("Schema content sanitization failed for step %s", step.ID),
							security.SeverityHigh,
							true,
						)
						err = fmt.Errorf("schema content sanitization failed: %w", sanitizeErr)
					} else {
						schemaContent = sanitizedContent
						// Log sanitization actions if any were taken
						if len(sanitizationActions) > 0 {
							e.securityLogger.LogViolation(
								string(security.ViolationPromptInjection),
								string(security.SourceSchemaPath),
								fmt.Sprintf("Schema content sanitized for step %s: %v", step.ID, sanitizationActions),
								security.SeverityMedium,
								false,
							)
						}
					}
				} else {
					err = readErr
				}
			}
		} else if step.Handover.Contract.Schema != "" {
			// SECURITY FIX: Sanitize inline schema content
			sanitizedContent, sanitizationActions, sanitizeErr := e.inputSanitizer.SanitizeSchemaContent(step.Handover.Contract.Schema)
			if sanitizeErr != nil {
				e.securityLogger.LogViolation(
					string(security.ViolationInputValidation),
					string(security.SourceSchemaPath),
					fmt.Sprintf("Inline schema sanitization failed for step %s", step.ID),
					security.SeverityHigh,
					true,
				)
				err = fmt.Errorf("inline schema sanitization failed: %w", sanitizeErr)
			} else {
				schemaContent = sanitizedContent
				// Log sanitization actions if any were taken
				if len(sanitizationActions) > 0 {
					e.securityLogger.LogViolation(
						string(security.ViolationPromptInjection),
						string(security.SourceSchemaPath),
						fmt.Sprintf("Inline schema sanitized for step %s: %v", step.ID, sanitizationActions),
						security.SeverityMedium,
						false,
					)
				}
			}
		}

		// Inject schema guidance if available and safe
		if schemaContent != "" && err == nil {
			prompt += "\n\nOUTPUT REQUIREMENTS:\n"
			prompt += "After completing all required tool calls (Bash, Read, Write, etc.), save your final output to artifact.json.\n"
			prompt += "The artifact.json must be valid JSON matching this schema:\n```json\n"
			prompt += schemaContent
			prompt += "\n```\n\n"
			prompt += "IMPORTANT:\n"
			prompt += "- First, execute any tool calls needed to gather data\n"
			prompt += "- Then, use the Write tool to save valid JSON to artifact.json\n"
			prompt += "- The JSON must match every required field in the schema\n"
		}
	}

	// Resolve remaining template variables using pipeline context
	if execution.Context != nil {
		prompt = execution.Context.ResolvePlaceholders(prompt)
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
		// Resolve artifact path using pipeline context
		resolvedPath := execution.Context.ResolveArtifactPath(art)
		artPath := filepath.Join(workspacePath, resolvedPath)
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

// startProgressTicker starts a background ticker to emit periodic progress events
// during step execution to ensure smooth animation updates
func (e *DefaultPipelineExecutor) startProgressTicker(ctx context.Context, pipelineID string, stepID string) context.CancelFunc {
	tickerCtx, cancel := context.WithCancel(ctx)

	if e.emitter != nil {
		go func() {
			ticker := time.NewTicker(1000 * time.Millisecond) // 1 FPS for progress updates
			defer ticker.Stop()

			for {
				select {
				case <-tickerCtx.Done():
					return
				case <-ticker.C:
					// Emit a progress heartbeat to keep the display updating
					e.emit(event.Event{
						PipelineID: pipelineID,
						StepID:     stepID,
						State:      event.StateStepProgress,
						Timestamp:  time.Now(),
					})
				}
			}
		}()
	}

	return cancel
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

// trackStepDeliverables automatically tracks deliverables produced by a completed step
func (e *DefaultPipelineExecutor) trackStepDeliverables(execution *PipelineExecution, step *Step) {
	if e.deliverableTracker == nil {
		return
	}

	// Get workspace path for this step
	workspacePath, exists := execution.WorkspacePaths[step.ID]
	if !exists {
		return
	}

	// Track workspace files automatically
	e.deliverableTracker.AddWorkspaceFiles(step.ID, workspacePath)

	// Track explicit output artifacts
	for _, artifact := range step.OutputArtifacts {
		resolvedPath := execution.Context.ResolveArtifactPath(artifact)
		artifactPath := filepath.Join(workspacePath, resolvedPath)

		// Get absolute path
		absPath, err := filepath.Abs(artifactPath)
		if err != nil {
			absPath = artifactPath
		}

		e.deliverableTracker.AddFile(step.ID, artifact.Name, absPath, artifact.Type)
	}

	// Check for common deliverable patterns
	e.trackCommonDeliverables(step.ID, workspacePath, execution)
}

// trackCommonDeliverables looks for common deliverable patterns like PR links, deployment URLs, etc.
func (e *DefaultPipelineExecutor) trackCommonDeliverables(stepID, workspacePath string, execution *PipelineExecution) {
	// Check step results for URLs, PRs, deployments
	if results, exists := execution.Results[stepID]; exists {
		// Look for PR URLs in results
		if prURL, ok := results["pr_url"].(string); ok && prURL != "" {
			e.deliverableTracker.AddPR(stepID, "Pull Request", prURL, "Generated pull request")
		}

		// Look for deployment URLs in results
		if deployURL, ok := results["deploy_url"].(string); ok && deployURL != "" {
			e.deliverableTracker.AddDeployment(stepID, "Deployment", deployURL, "Deployed application")
		}

		// Look for any URLs in results
		for key, value := range results {
			if strValue, ok := value.(string); ok {
				if strings.HasPrefix(strValue, "http://") || strings.HasPrefix(strValue, "https://") {
					e.deliverableTracker.AddURL(stepID, key, strValue, fmt.Sprintf("URL from %s", key))
				}
			}
		}
	}

	// Check for log files
	logFiles := []string{"step.log", "execution.log", "debug.log", "output.log"}
	for _, logFile := range logFiles {
		logPath := filepath.Join(workspacePath, logFile)
		if _, err := os.Stat(logPath); err == nil {
			absPath, _ := filepath.Abs(logPath)
			e.deliverableTracker.AddLog(stepID, logFile, absPath, "Step execution log")
		}
	}

	// Check for contract artifacts
	contractFiles := []string{"contract.json", "schema.json", "api-spec.yaml", "openapi.yaml"}
	for _, contractFile := range contractFiles {
		contractPath := filepath.Join(workspacePath, contractFile)
		if _, err := os.Stat(contractPath); err == nil {
			absPath, _ := filepath.Abs(contractPath)
			e.deliverableTracker.AddContract(stepID, contractFile, absPath, "Contract artifact")
		}
	}
}

// GetDeliverables returns the deliverables summary for the completed pipeline
func (e *DefaultPipelineExecutor) GetDeliverables() string {
	if e.deliverableTracker == nil {
		return ""
	}
	return e.deliverableTracker.FormatSummary()
}

// GetDeliverableTracker returns the deliverable tracker for external access
func (e *DefaultPipelineExecutor) GetDeliverableTracker() *deliverable.Tracker {
	return e.deliverableTracker
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
				// Clean up failed pipeline from in-memory storage to prevent memory leak
				e.cleanupCompletedPipeline(execution.Pipeline.Metadata.Name)
				return fmt.Errorf("step %q failed: %w", step.ID, err)
			}
			execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
		}
	}

	now := time.Now()
	execution.Status.CompletedAt = &now
	execution.Status.State = StateCompleted

	// Clean up completed pipeline from in-memory storage to prevent memory leak
	e.cleanupCompletedPipeline(execution.Pipeline.Metadata.Name)

	return nil
}

func (e *DefaultPipelineExecutor) GetStatus(pipelineID string) (*PipelineStatus, error) {
	// First check in-memory storage for running pipelines
	e.mu.RLock()
	execution, exists := e.pipelines[pipelineID]
	e.mu.RUnlock()

	if exists {
		return execution.Status, nil
	}

	// Fall back to persistent storage for completed pipelines
	if e.store != nil {
		stateRecord, err := e.store.GetPipelineState(pipelineID)
		if err != nil {
			return nil, fmt.Errorf("pipeline %q not found", pipelineID)
		}

		// Convert StateStore record to PipelineStatus
		status := &PipelineStatus{
			ID:             stateRecord.PipelineID,
			State:          stateRecord.Status,
			CurrentStep:    "", // Not tracked in legacy state store
			CompletedSteps: []string{}, // Would need step states to populate
			FailedSteps:    []string{}, // Would need step states to populate
			StartedAt:      stateRecord.CreatedAt,
		}

		// Set completion time if pipeline is completed
		if stateRecord.Status == StateCompleted || stateRecord.Status == StateFailed {
			status.CompletedAt = &stateRecord.UpdatedAt
		}

		// Optionally populate step information from step states
		stepStates, stepErr := e.store.GetStepStates(pipelineID)
		if stepErr == nil {
			for _, stepState := range stepStates {
				switch stepState.State {
				case StateCompleted:
					status.CompletedSteps = append(status.CompletedSteps, stepState.StepID)
				case StateFailed:
					status.FailedSteps = append(status.FailedSteps, stepState.StepID)
				case StateRunning, StateRetrying:
					status.CurrentStep = stepState.StepID
				}
			}
		}

		return status, nil
	}

	return nil, fmt.Errorf("pipeline %q not found", pipelineID)
}

// cleanupCompletedPipeline removes a completed or failed pipeline from in-memory storage
// to prevent memory leaks. This is safe to call because completed pipeline status
// can be retrieved from persistent storage via GetStatus.
func (e *DefaultPipelineExecutor) cleanupCompletedPipeline(pipelineID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.pipelines, pipelineID)
}

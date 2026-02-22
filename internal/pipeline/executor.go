package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/worktree"
	"github.com/recinq/wave/internal/workspace"
)


type PipelineExecutor interface {
	Execute(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error
	Resume(ctx context.Context, pipelineID string, fromStep string) error
	GetStatus(pipelineID string) (*PipelineStatus, error)
}

type PipelineStatus struct {
	ID             string // Runtime ID with hash suffix (e.g., "my-pipeline-a3b2c1d4")
	PipelineName   string // Logical pipeline name from Metadata.Name
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
	// Pre-generated run ID (optional — if empty, Execute generates one)
	runID string
	// Per-step timeout override (from CLI --timeout flag)
	stepTimeoutOverride time.Duration
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

func WithRunID(id string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.runID = id }
}

func WithStepTimeout(d time.Duration) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.stepTimeoutOverride = d }
}

// createRunID generates a run ID, preferring the state store's CreateRun()
// so the run appears in the dashboard. Falls back to GenerateRunID() if
// the store is unavailable or the call fails.
func (e *DefaultPipelineExecutor) createRunID(name string, hashLen int, input string) string {
	if e.store != nil {
		if id, err := e.store.CreateRun(name, input); err == nil {
			return id
		}
	}
	return GenerateRunID(name, hashLen)
}

// WorktreeInfo tracks a shared worktree created for a specific branch.
// Multiple steps using the same branch reuse the same worktree.
type WorktreeInfo struct {
	AbsPath  string // Absolute path to the worktree directory
	RepoRoot string // Repository root for cleanup
}

type PipelineExecution struct {
	Pipeline       *Pipeline
	Manifest       *manifest.Manifest
	States         map[string]string
	Results        map[string]map[string]interface{}
	ArtifactPaths  map[string]string          // "stepID:artifactName" -> filesystem path
	WorkspacePaths map[string]string          // stepID -> workspace path
	WorktreePaths  map[string]*WorktreeInfo   // resolved branch -> worktree info
	Input          string
	Status         *PipelineStatus
	Context        *PipelineContext  // Dynamic template variables
}

func NewDefaultPipelineExecutor(runner adapter.AdapterRunner, opts ...ExecutorOption) *DefaultPipelineExecutor {
	ex := &DefaultPipelineExecutor{
		runner:             runner,
		pipelines:          make(map[string]*PipelineExecution),
		deliverableTracker: deliverable.NewTracker(""),
	}
	for _, opt := range opts {
		opt(ex)
	}

	// Initialize security after options so logging respects --debug
	securityConfig := security.DefaultSecurityConfig()
	securityLogger := security.NewSecurityLogger(securityConfig.LoggingEnabled && ex.debug)
	ex.securityConfig = securityConfig
	ex.pathValidator = security.NewPathValidator(*securityConfig, securityLogger)
	ex.inputSanitizer = security.NewInputSanitizer(*securityConfig, securityLogger)
	ex.securityLogger = securityLogger

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

	// Preflight validation: check required tools and skills before execution
	if p.Requires != nil {
		checker := preflight.NewChecker(m.Skills)
		var tools, skills []string
		if len(p.Requires.Tools) > 0 {
			tools = p.Requires.Tools
		}
		if len(p.Requires.Skills) > 0 {
			skills = p.Requires.Skills
		}
		if len(tools) > 0 || len(skills) > 0 {
			results, err := checker.Run(tools, skills)
			for _, r := range results {
				e.emit(event.Event{
					Timestamp: time.Now(),
					State:     "preflight",
					Message:   r.Message,
				})
			}
			if err != nil {
				return fmt.Errorf("preflight check failed: %w", err)
			}
		}
	}

	pipelineName := p.Metadata.Name
	pipelineID := e.runID
	if pipelineID == "" {
		pipelineID = GenerateRunID(pipelineName, m.Runtime.PipelineIDHashLength)
	}
	pipelineContext := newContextWithProject(pipelineID, pipelineName, "", m)

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
		WorktreePaths:  make(map[string]*WorktreeInfo),
		Input:          input,
		Context:        pipelineContext,
		Status: &PipelineStatus{
			ID:             pipelineID,
			PipelineName:   pipelineName,
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
			return &StepError{StepID: step.ID, Err: err}
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
	pipelineID := execution.Status.ID
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
			// Don't retry if the parent context is already cancelled
			if ctx.Err() != nil {
				return fmt.Errorf("context cancelled, skipping retry: %w", lastErr)
			}
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
	pipelineID := execution.Status.ID

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
	pipelineID := execution.Status.ID

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
		Model:         persona.Model,
		Adapter:       adapterDef.Binary,
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
	if e.stepTimeoutOverride > 0 {
		timeout = e.stepTimeoutOverride
	}

	// Load system prompt from persona file
	systemPrompt := ""
	if persona.SystemPromptFile != "" {
		if data, err := os.ReadFile(persona.SystemPromptFile); err == nil {
			systemPrompt = string(data)
		}
	}

	// Auto-grant Write permissions for declared output artifact paths
	allowedTools := persona.Permissions.AllowedTools
	for _, art := range step.OutputArtifacts {
		dir := filepath.Dir(art.Path)
		if dir == "." {
			allowedTools = append(allowedTools, "Write("+art.Path+")")
		} else {
			allowedTools = append(allowedTools, "Write("+dir+"/*)")
		}
	}

	// Resolve sandbox config — all gated on runtime.sandbox.enabled
	sandboxEnabled := execution.Manifest.Runtime.Sandbox.Enabled
	var sandboxDomains []string
	var envPassthrough []string
	if sandboxEnabled {
		if persona.Sandbox != nil && len(persona.Sandbox.AllowedDomains) > 0 {
			sandboxDomains = persona.Sandbox.AllowedDomains
		} else if len(execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains) > 0 {
			sandboxDomains = execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains
		}
		envPassthrough = execution.Manifest.Runtime.Sandbox.EnvPassthrough
	}

	// Resolve skill commands directory for provisioning
	var skillCommandsDir string
	if execution.Pipeline.Requires != nil && len(execution.Pipeline.Requires.Skills) > 0 && len(execution.Manifest.Skills) > 0 {
		provisioner := skill.NewProvisioner(execution.Manifest.Skills, "")
		commands, _ := provisioner.DiscoverCommands(execution.Pipeline.Requires.Skills)
		// If we found any commands, provision them into a temp dir that the adapter can use
		if len(commands) > 0 {
			tmpDir := filepath.Join(workspacePath, ".wave-skill-commands")
			if err := provisioner.Provision(tmpDir, execution.Pipeline.Requires.Skills); err != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "warning",
					Message:    fmt.Sprintf("skill provisioning failed: %v", err),
				})
			} else {
				skillCommandsDir = filepath.Join(tmpDir, ".claude", "commands")
			}
		}
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:          adapterDef.Binary,
		Persona:          step.Persona,
		WorkspacePath:    workspacePath,
		Prompt:           prompt,
		SystemPrompt:     systemPrompt,
		Timeout:          timeout,
		Temperature:      persona.Temperature,
		Model:            persona.Model,
		AllowedTools:     allowedTools,
		DenyTools:        persona.Permissions.Deny,
		OutputFormat:     adapterDef.OutputFormat,
		Debug:            e.debug,
		SandboxEnabled:   sandboxEnabled,
		AllowedDomains:   sandboxDomains,
		EnvPassthrough:   envPassthrough,
		SkillCommandsDir: skillCommandsDir,
		OnStreamEvent: func(evt adapter.StreamEvent) {
			if evt.Type == "tool_use" && evt.ToolName != "" {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStreamActivity,
					Persona:    step.Persona,
					ToolName:   evt.ToolName,
					ToolTarget: evt.ToolInput,
				})
			}
		},
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
		// Let the higher-level executor emit a single failure event; just
		// propagate the error with context so enriched details (e.g. from
		// *adapter.StepError) remain available to callers.
		return fmt.Errorf("adapter execution failed: %w", err)
	}

	// Warn on non-zero exit code — adapter process may have crashed, but
	// work may still have been completed (e.g. Claude Code JS error after
	// tool calls finished). Let contract validation decide the outcome.
	if result.ExitCode != 0 {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    fmt.Sprintf("adapter exited with code %d (process may have crashed)", result.ExitCode),
		})
	}

	// Fail immediately on rate limit — the result content is an error message,
	// not useful work product. Proceeding would write the error as an artifact.
	if result.FailureReason == adapter.FailureReasonRateLimit {
		return fmt.Errorf("adapter rate limited: %s", result.ResultContent)
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

		// Resolve {{ project.* }} placeholders in contract command
		resolvedCommand := step.Handover.Contract.Command
		if execution.Context != nil && resolvedCommand != "" {
			resolvedCommand = execution.Context.ResolvePlaceholders(resolvedCommand)
		}

		contractCfg := contract.ContractConfig{
			Type:       step.Handover.Contract.Type,
			Source:     resolvedSource,
			Schema:     step.Handover.Contract.Schema,
			SchemaPath: step.Handover.Contract.SchemaPath,
			Command:    resolvedCommand,
			Dir:        step.Handover.Contract.Dir,
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
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Handle workspace ref — share another step's workspace
	if step.Workspace.Ref != "" {
		refPath, ok := execution.WorkspacePaths[step.Workspace.Ref]
		if !ok {
			return "", fmt.Errorf("referenced workspace step %q has not been executed yet", step.Workspace.Ref)
		}
		return refPath, nil
	}

	// Handle worktree workspace type
	if step.Workspace.Type == "worktree" {
		// Resolve branch name from template variables
		branch := step.Workspace.Branch
		if execution.Context != nil && branch != "" {
			branch = execution.Context.ResolvePlaceholders(branch)
		}

		// Resolve base ref from template variables
		base := step.Workspace.Base
		if execution.Context != nil && base != "" {
			base = execution.Context.ResolvePlaceholders(base)
		}

		if branch == "" && base == "" {
			// Fall back to pipeline context branch or generate one
			branch = execution.Context.BranchName
			if branch == "" {
				branch = fmt.Sprintf("wave/%s/%s", pipelineID, step.ID)
			}
		}

		// Reuse existing worktree for the same branch
		if info, ok := execution.WorktreePaths[branch]; ok {
			execution.WorkspacePaths[step.ID+"__worktree_repo_root"] = info.RepoRoot
			return info.AbsPath, nil
		}

		// Branch-keyed path for sharing across steps
		sanitized := sanitizeBranchName(branch)
		wtKey := "__wt_" + sanitized
		wsPath := filepath.Join(wsRoot, pipelineID, wtKey)

		absPath, err := filepath.Abs(wsPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve workspace path: %w", err)
		}

		mgr, err := worktree.NewManager("")
		if err != nil {
			return "", fmt.Errorf("failed to create worktree manager: %w", err)
		}

		if err := mgr.Create(absPath, branch, base); err != nil {
			return "", fmt.Errorf("failed to create worktree workspace: %w", err)
		}

		// Register for reuse and cleanup
		execution.WorktreePaths[branch] = &WorktreeInfo{AbsPath: absPath, RepoRoot: mgr.RepoRoot()}

		// Record branch creation as a deliverable for outcome tracking
		e.deliverableTracker.AddBranch(step.ID, branch, absPath, "Feature branch")
		execution.WorkspacePaths[step.ID+"__worktree_repo_root"] = mgr.RepoRoot()

		// Mark CLAUDE.md as skip-worktree so prepareWorkspace() changes
		// don't get staged by git add -A in implement steps
		exec.Command("git", "-C", absPath, "update-index", "--skip-worktree", "CLAUDE.md").Run()

		return absPath, nil
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
	// Handle slash_command exec type
	if step.Exec.Type == "slash_command" && step.Exec.Command != "" {
		args := step.Exec.Args
		if execution.Context != nil {
			args = execution.Context.ResolvePlaceholders(args)
		}
		// Replace {{ input }} in args
		if execution.Input != "" {
			for _, pattern := range []string{"{{ input }}", "{{input}}", "{{ input}}", "{{input }}"} {
				args = strings.ReplaceAll(args, pattern, execution.Input)
			}
		}
		return skill.FormatSkillCommandPrompt(step.Exec.Command, args)
	}

	prompt := step.Exec.Source

	// Load prompt from external file if source_path is set
	if step.Exec.SourcePath != "" {
		if e.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Loading prompt from source_path: %s\n", step.Exec.SourcePath)
		}
		data, err := os.ReadFile(step.Exec.SourcePath)
		if err != nil {
			if e.debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Failed to read prompt from %s: %v\n", step.Exec.SourcePath, err)
			}
		} else {
			prompt = string(data)
			if e.debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Loaded prompt: %d bytes\n", len(prompt))
			}
		}
	} else if e.debug && step.Exec.Source == "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Warning: step %s has neither source nor source_path set\n", step.ID)
	}

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
			prompt += "After completing all required tool calls (Bash, Read, Write, etc.), save your final output to .wave/artifact.json.\n"
			prompt += "The .wave/artifact.json must be valid JSON matching this schema:\n```json\n"
			prompt += schemaContent
			prompt += "\n```\n\n"
			prompt += "IMPORTANT:\n"
			prompt += "- First, execute any tool calls needed to gather data\n"
			prompt += "- Then, use the Write tool to save valid JSON to .wave/artifact.json\n"
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

	// Always inject into the workspace (agent's working directory) so the
	// agent can find artifacts at relative paths like ".wave/artifacts/<name>".
	// Do NOT redirect to the sidecar — the agent runs in workspacePath.
	artifactsDir := filepath.Join(workspacePath, ".wave", "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts dir: %w", err)
	}

	pipelineID := execution.Status.ID

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
		key := step.ID + ":" + art.Name

		// If the persona already wrote the file, trust it and don't overwrite
		if _, err := os.Stat(artPath); err == nil {
			execution.ArtifactPaths[key] = artPath
			if e.debug {
				fmt.Printf("[DEBUG] Artifact %s already exists at %s, preserving persona-written file\n", art.Name, artPath)
			}
		} else {
			// Fall back to writing ResultContent
			os.MkdirAll(filepath.Dir(artPath), 0755)
			os.WriteFile(artPath, stdout, 0644)
			execution.ArtifactPaths[key] = artPath
		}

		// Register artifact in DB for web dashboard visibility
		if e.store != nil {
			var size int64
			if info, err := os.Stat(artPath); err == nil {
				size = info.Size()
			}
			e.store.RegisterArtifact(execution.Status.ID, step.ID, art.Name, artPath, art.Type, size)
		}
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

	pipelineID := execution.Status.ID

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

	// Track explicit output artifacts (declared in pipeline YAML only)
	for _, artifact := range step.OutputArtifacts {
		resolvedPath := execution.Context.ResolveArtifactPath(artifact)
		artifactPath := filepath.Join(workspacePath, resolvedPath)

		// Get absolute path
		absPath, err := filepath.Abs(artifactPath)
		if err != nil {
			absPath = artifactPath
		}

		e.deliverableTracker.AddFile(step.ID, artifact.Name, absPath, artifact.Type)

		// Register artifact in DB for web dashboard visibility
		if e.store != nil {
			var size int64
			if info, statErr := os.Stat(absPath); statErr == nil {
				size = info.Size()
			}
			e.store.RegisterArtifact(execution.Status.ID, step.ID, artifact.Name, absPath, artifact.Type, size)
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

// GetTotalTokens returns the sum of tokens used across all completed steps.
func (e *DefaultPipelineExecutor) GetTotalTokens() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var total int
	for _, execution := range e.pipelines {
		for _, result := range execution.Results {
			if tokens, ok := result["tokens_used"].(int); ok {
				total += tokens
			}
		}
	}
	return total
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
				e.cleanupCompletedPipeline(execution.Status.ID)
				return &StepError{StepID: step.ID, Err: err}
			}
			execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
		}
	}

	now := time.Now()
	execution.Status.CompletedAt = &now
	execution.Status.State = StateCompleted

	// Clean up completed pipeline from in-memory storage to prevent memory leak
	e.cleanupCompletedPipeline(execution.Status.ID)

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

// cleanupWorktrees removes any git worktrees created during pipeline execution.
func (e *DefaultPipelineExecutor) cleanupWorktrees(execution *PipelineExecution, pipelineID string) {
	cleaned := map[string]bool{}
	for key, repoRoot := range execution.WorkspacePaths {
		if !strings.HasSuffix(key, "__worktree_repo_root") {
			continue
		}
		stepID := strings.TrimSuffix(key, "__worktree_repo_root")
		wsPath := execution.WorkspacePaths[stepID]
		if wsPath == "" {
			continue
		}
		// Skip already-cleaned paths (shared worktrees used by multiple steps)
		if cleaned[wsPath] {
			continue
		}
		cleaned[wsPath] = true
		mgr, err := worktree.NewManager(repoRoot)
		if err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     stepID,
				State:      "warning",
				Message:    fmt.Sprintf("worktree cleanup skipped: %v", err),
			})
			continue
		}
		if err := mgr.Remove(wsPath); err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     stepID,
				State:      "warning",
				Message:    fmt.Sprintf("worktree cleanup failed: %v", err),
			})
		}
	}
}

// cleanupCompletedPipeline removes a completed or failed pipeline from in-memory storage
// to prevent memory leaks. This is safe to call because completed pipeline status
// can be retrieved from persistent storage via GetStatus.
func (e *DefaultPipelineExecutor) cleanupCompletedPipeline(pipelineID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.pipelines, pipelineID)
}

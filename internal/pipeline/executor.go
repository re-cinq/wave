package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/audit"
	"github.com/recinq/wave/internal/cost"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
)

// maxStdoutTailChars is the maximum number of characters to retain from
// stdout when passing output to retry/rework context. Keeps payloads
// small while preserving the most recent (and usually most relevant) output.
const maxStdoutTailChars = 2000

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
	emitterMixin
	runner       adapter.AdapterRunner // Deprecated: use registry for per-step resolution
	registry     *adapter.AdapterRegistry
	store        state.StateStore
	logger       audit.AuditLogger
	wsManager    workspace.WorkspaceManager
	relayMonitor *relay.RelayMonitor
	pipelines    map[string]*PipelineExecution
	mu           sync.RWMutex
	debug        bool
	// Security layer: path/input/schema sanitization, skill ref validation
	sec *securityLayer
	// Outcome tracking (in-memory cache + state-store persistence)
	outcomeTracker *state.OutcomeTracker
	// Pre-generated run ID (optional — if empty, Execute generates one)
	runID string
	// Workspace run ID override (used by resume to point at the original
	// run's workspace tree while persisting state under the new resume run).
	// When empty, defaults to runID.
	workspaceRunID string
	// Per-step timeout override (from CLI --timeout flag)
	stepTimeoutOverride time.Duration
	// Model override (from CLI --model flag)
	modelOverride   string
	forceModel      bool
	adapterOverride string
	// Cross-pipeline artifacts from prior stages in a sequence
	crossPipelineArtifacts map[string]map[string][]byte // pipelineName -> artifactName -> data
	// ETA calculator for remaining pipeline time estimates
	etaCalculator *ETACalculator
	// Preserve workspace from previous run (skip cleanup for debugging)
	preserveWorkspace bool
	// Step filter for selective step execution (--steps / --exclude)
	stepFilter *StepFilter
	// Skill store for DirectoryStore-based skill provisioning
	skillStore skill.Store
	// Most recent execution for child state access
	lastExecution *PipelineExecution
	// Base branch override for stacked matrix execution (set by parent matrix executor)
	stackedBaseBranch string
	// Debug tracer for structured NDJSON trace file output (enabled by --debug)
	debugTracer *audit.DebugTracer
	// Accumulated token count across all steps (survives pipeline cleanup)
	totalTokens int
	// Lifecycle hook runner for pipeline-level hooks
	hookRunner hooks.HookRunner
	// Auto-approve mode: skip all approval gates using default choices
	autoApprove bool
	// Gate handler for interactive approval gates (CLI, TUI, WebUI)
	gateHandler GateHandler
	// Parent artifact paths injected from a parent sub-pipeline step
	parentArtifactPaths map[string]string
	// Parent workspace path for workspace.ref: parent resolution
	parentWorkspacePath string
	// Parent env vars injected from a parent sub-pipeline step's Config.Env.
	// Seeded into PipelineContext.CustomVariables as env.<key> so child
	// templates resolve {{ env.<key> }} via ResolvePlaceholders.
	parentEnv map[string]string
	// Retrospective generator for post-run analysis
	retroGenerator *retro.Generator
	// Cost ledger for per-run cost tracking and budget enforcement
	costLedger *cost.Ledger
	// Webhook runner for dynamic webhook delivery (non-blocking)
	webhookRunner *hooks.WebhookRunner
	// Task-level complexity from classifier (empty = no task-aware routing)
	taskComplexity string
	// Ontology service — bounded-context injection, staleness, lineage.
	// Required at construction time: callers must supply via
	// WithOntologyService (use ontology.NoOp{} to opt out explicitly). The
	// constructor panics on nil so a forgotten dependency surfaces loudly
	// instead of silently disabling lineage tracking. The Execute path may
	// still promote a NoOp to a real service when the manifest declares
	// ontology contexts.
	ontology ontology.Service
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

func WithDebugTracer(t *audit.DebugTracer) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.debugTracer = t }
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

// WithWorkspaceRunID overrides the run ID used to compute step workspace paths.
// Resume uses this to keep the resumed executor reading from the original run's
// workspace tree (where prior steps wrote artifacts) while state and new
// artifacts are persisted under the new resume run ID.
func WithWorkspaceRunID(id string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.workspaceRunID = id }
}

func WithStepTimeout(d time.Duration) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.stepTimeoutOverride = d }
}

func WithModelOverride(model string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.modelOverride = model }
}

// WithTaskComplexity sets the task-level complexity from the classifier.
// When set, it adjusts model routing: simple tasks cap at balanced,
// complex/architectural tasks floor at balanced.
func WithTaskComplexity(complexity string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.taskComplexity = complexity }
}

func WithForceModel(force bool) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.forceModel = force }
}

func WithAdapterOverride(adapter string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.adapterOverride = adapter }
}

// WithCrossPipelineArtifacts injects artifacts from prior pipeline stages
// for cross-pipeline artifact references.
func WithCrossPipelineArtifacts(artifacts map[string]map[string][]byte) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.crossPipelineArtifacts = artifacts }
}

// WithPreserveWorkspace skips workspace cleanup at pipeline start,
// preserving the workspace from a previous run for debugging purposes.
func WithPreserveWorkspace(preserve bool) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.preserveWorkspace = preserve }
}

// WithStepFilter sets the step filter for selective step execution.
func WithStepFilter(f *StepFilter) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.stepFilter = f }
}

// WithSkillStore sets the skill store for DirectoryStore-based skill provisioning.
func WithSkillStore(s skill.Store) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.skillStore = s }
}

// withSkillStore is an internal alias kept for child executor propagation.
func withSkillStore(s skill.Store) ExecutorOption { return WithSkillStore(s) }

// withHookRunner sets the lifecycle hook runner for pipeline events.
func withHookRunner(r hooks.HookRunner) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.hookRunner = r }
}

// WithAutoApprove enables auto-approve mode where all approval gates use their
// default choice without human interaction. Required for --detach and CI mode.
func WithAutoApprove(auto bool) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.autoApprove = auto }
}

// WithGateHandler sets the interactive handler for approval gates with choices.
func WithGateHandler(h GateHandler) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.gateHandler = h }
}

// WithParentArtifactPaths injects artifact paths from a parent pipeline execution
// into the child executor. These are registered in the child's PipelineContext
// at execution start, making them available via {{ artifacts.<name> }} templates.
func WithParentArtifactPaths(paths map[string]string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.parentArtifactPaths = paths }
}

// WithParentWorkspacePath sets the parent step's workspace path so child steps
// can reference it via workspace.ref: parent.
func WithParentWorkspacePath(path string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.parentWorkspacePath = path }
}

// WithParentEnv injects sub-pipeline-supplied env vars from a parent step's
// Config.Env into the child executor. Seeded into the child's PipelineContext
// as custom variables keyed env.<name>, making them available to all template
// resolution in the child via {{ env.<name> }}. Distinct from process env.
func WithParentEnv(env map[string]string) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.parentEnv = env }
}

// WithRetroGenerator sets the retrospective generator for post-run analysis.
func WithRetroGenerator(g *retro.Generator) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.retroGenerator = g }
}

// WithOntologyService injects a pre-constructed ontology Service. When not
// set, Execute auto-constructs one from the manifest via
// ontology.EnabledFromManifest — wiring the store, emitter, and audit sink
// from this executor.
func WithOntologyService(svc ontology.Service) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.ontology = svc }
}

// WithRegistry sets the adapter registry for per-step adapter resolution.
func WithRegistry(r *adapter.AdapterRegistry) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.registry = r }
}

// workspaceRunIDFor returns the run ID used to compute step workspace paths.
// When WithWorkspaceRunID has been set (resume), the override wins so the
// resumed executor reads from the original run's workspace tree. Otherwise it
// falls back to the caller's pipelineID (typically execution.Status.ID),
// which preserves prior behaviour for fresh runs.
func (e *DefaultPipelineExecutor) workspaceRunIDFor(pipelineID string) string {
	if e.workspaceRunID != "" {
		return e.workspaceRunID
	}
	return pipelineID
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
	mu                sync.Mutex // protects map writes during concurrent steps
	Pipeline          *Pipeline
	Manifest          *manifest.Manifest
	States            map[string]string
	Results           map[string]map[string]interface{}
	ArtifactPaths     map[string]string        // "stepID:artifactName" -> filesystem path
	WorkspacePaths    map[string]string        // stepID -> workspace path
	WorktreePaths     map[string]*WorktreeInfo // resolved branch -> worktree info
	Input             string
	Status            *PipelineStatus
	Context           *PipelineContext           // Dynamic template variables
	AttemptContexts   map[string]*AttemptContext // stepID -> current retry context (nil on first attempt)
	ReworkTransitions map[string]string          // failedStepID -> reworkStepID (for resume support)
	ThreadManager     *ThreadManager             // Thread conversation continuity manager
	CircuitBreaker    *CircuitBreaker            // Failure fingerprint tracking for circuit breaking
	Watchdog          *StallWatchdog             // Current step's stall watchdog (set during step execution)
}

// stepRunResources holds resolved values needed to dispatch a single step to an adapter.
// Produced by resolveStepResources; consumed by buildStepAdapterConfig and the adapter dispatch.
type stepRunResources struct {
	pipelineID          string
	resolvedPersona     string
	persona             *manifest.Persona
	adapterDef          *manifest.Adapter
	resolvedAdapterName string
	stepRunner          adapter.AdapterRunner
	workspacePath       string
	resolvedModel       string
	configuredModel     string
	prompt              string
}

// pipelineSetup holds the results of pipeline preflight validation.
// Produced by validatePipelineAndCreateContext; consumed by subsequent setup phases.
type pipelineSetup struct {
	pipelineID             string
	pipelineName           string
	sortedSteps            []*Step
	pipelineContext        *PipelineContext
	forgeInfo              forge.ForgeInfo
	resolvedPipelineSkills []string
}

func NewDefaultPipelineExecutor(runner adapter.AdapterRunner, opts ...ExecutorOption) *DefaultPipelineExecutor {
	ex := &DefaultPipelineExecutor{
		runner:    runner,
		pipelines: make(map[string]*PipelineExecution),
	}
	for _, opt := range opts {
		opt(ex)
	}
	// Ontology is a required dependency. Callers that don't need lineage
	// tracking must opt out explicitly via WithOntologyService(ontology.NoOp{}).
	// This avoids the silent no-op trap where forgetting to wire the service
	// produces a successful run with no lineage data and no warning.
	if ex.ontology == nil {
		panic("pipeline: ontology service is required; pass WithOntologyService(ontology.NoOp{}) to opt out explicitly")
	}
	ex.outcomeTracker = state.NewOutcomeTracker("", ex.store)

	// If no registry was provided via WithRegistry, wrap the single runner
	if ex.registry == nil {
		ex.registry = adapter.NewSingleRunnerRegistry(runner)
	}

	// Initialize security layer after options so logging respects --debug
	ex.sec = newSecurityLayer(ex)

	return ex
}

// NewChildExecutor creates a fresh executor that shares the same adapter runner,
// event emitter, workspace manager, and configuration, but has independent
// execution state. Used for child pipeline invocation within matrix strategies.
func (e *DefaultPipelineExecutor) NewChildExecutor() *DefaultPipelineExecutor {
	child := &DefaultPipelineExecutor{
		emitterMixin:           emitterMixin{emitter: e.emitter},
		runner:                 e.runner,
		registry:               e.registry,
		adapterOverride:        e.adapterOverride,
		store:                  e.store,
		logger:                 e.logger,
		wsManager:              e.wsManager,
		relayMonitor:           e.relayMonitor,
		pipelines:              make(map[string]*PipelineExecution),
		debug:                  e.debug,
		modelOverride:          e.modelOverride,
		sec:                    e.sec,
		outcomeTracker:         state.NewOutcomeTracker("", e.store),
		crossPipelineArtifacts: e.crossPipelineArtifacts,
		preserveWorkspace:      e.preserveWorkspace,
		skillStore:             e.skillStore,
		hookRunner:             e.hookRunner,
		autoApprove:            e.autoApprove,
		gateHandler:            e.gateHandler,
		retroGenerator:         e.retroGenerator,
		ontology:               e.ontology,
	}
	// Share parent security layer's collaborators so child sees identical
	// path/sanitization config but with its own back-pointer.
	if e.sec != nil {
		child.sec = &securityLayer{
			e:              child,
			securityConfig: e.sec.securityConfig,
			pathValidator:  e.sec.pathValidator,
			inputSanitizer: e.sec.inputSanitizer,
			securityLogger: e.sec.securityLogger,
		}
	} else {
		child.sec = newSecurityLayer(child)
	}
	return child
}

// LastExecution returns the most recently executed pipeline's execution state.
func (e *DefaultPipelineExecutor) LastExecution() *PipelineExecution {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastExecution
}

func (e *DefaultPipelineExecutor) Execute(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
	// Initialize cost ledger from manifest config
	if e.costLedger == nil {
		costCfg := m.Runtime.Cost
		if costCfg.Enabled || costCfg.BudgetCeiling > 0 {
			e.costLedger = cost.NewLedger(costCfg.BudgetCeiling, costCfg.WarnAt)
		}
	}

	// Detect graph-mode pipelines (edges or conditional steps present)
	if isGraphPipeline(p) {
		return e.executeGraphPipeline(ctx, p, m, input)
	}

	// Phase 1: Validate pipeline structure and create execution context
	setup, err := e.validatePipelineAndCreateContext(p, m, input)
	if err != nil {
		return err
	}

	// Phase 2: Preflight checks (skills, tools, forge, token scopes)
	if err := e.checkPipelinePreflights(ctx, setup, p, m); err != nil {
		return err
	}

	// Phase 3: Initialize execution state (context, deliverables, execution struct)
	execution, runCtx, cancel := e.initPipelineExecution(ctx, setup, p, m, input)
	defer cancel()

	// Phase 4: Prepare workspace, hooks, and fire run_start
	if err := e.setupPipelineRun(runCtx, execution, p, m); err != nil {
		return err
	}

	// Phase 5: Schedule and execute steps
	schedulableSteps, err := e.runSchedulingLoop(runCtx, execution, setup.sortedSteps)
	if err != nil {
		return err
	}

	// Phase 6: Finalize (status, terminal hooks, retro, cleanup)
	e.finalizePipelineExecution(runCtx, execution, schedulableSteps)
	return nil
}

// validatePipelineAndCreateContext validates the pipeline structure (DAG, threads, sort,
// retry policies, step filter, ETA) and creates the pipeline context and forge info.
// It is the first phase of Execute — run before any state is allocated.
func (e *DefaultPipelineExecutor) trace(eventType, stepID string, durationMs int64, metadata map[string]string) {
	if e.debugTracer == nil {
		return
	}
	_ = e.debugTracer.Emit(audit.TraceEvent{
		EventType:  eventType,
		StepID:     stepID,
		DurationMs: durationMs,
		Metadata:   metadata,
	})
}

// startProgressTicker starts a background ticker to emit periodic progress events
// during step execution to ensure smooth animation updates
func (e *DefaultPipelineExecutor) GetOutcomesSummary() string {
	if e.outcomeTracker == nil {
		return ""
	}
	return e.outcomeTracker.FormatSummary()
}

// GetOutcomeTracker returns the outcome tracker for external access.
func (e *DefaultPipelineExecutor) GetOutcomeTracker() *state.OutcomeTracker {
	return e.outcomeTracker
}

// GetRunID returns the run ID assigned to this executor (empty if not yet started).
func (e *DefaultPipelineExecutor) GetRunID() string {
	return e.runID
}

// GetTotalTokens returns the sum of tokens used across all completed steps.
func (e *DefaultPipelineExecutor) GetTotalTokens() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.totalTokens
}

// GetCostSummary returns a human-readable cost summary for the run, or empty if no cost tracking.
func (e *DefaultPipelineExecutor) GetCostSummary() string {
	if e.costLedger == nil {
		return ""
	}
	return e.costLedger.Summary()
}

// GetTotalCost returns the cumulative USD cost of the run, or 0 if cost tracking is disabled.
func (e *DefaultPipelineExecutor) GetTotalCost() float64 {
	if e.costLedger == nil {
		return 0
	}
	return e.costLedger.TotalCost()
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

	execution.Status.State = stateRunning

	resuming := false
	for _, step := range sortedSteps {
		if !resuming && step.ID == fromStep {
			resuming = true
		}
		if resuming && execution.States[step.ID] != stateCompleted {
			if err := e.executeStep(ctx, execution, step); err != nil {
				execution.Status.State = stateFailed
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
				// Clean up failed pipeline from in-memory storage to prevent memory leak
				e.cleanupCompletedPipeline(execution.Status.ID)
				return &StepExecutionError{StepID: step.ID, Err: err}
			}
			execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
		}
	}

	now := time.Now()
	execution.Status.CompletedAt = &now
	execution.Status.State = stateCompleted

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
			CurrentStep:    "",         // Not tracked in pipeline_state table
			CompletedSteps: []string{}, // Would need step states to populate
			FailedSteps:    []string{}, // Would need step states to populate
			StartedAt:      stateRecord.CreatedAt,
		}

		// Set completion time if pipeline is completed
		if stateRecord.Status == stateCompleted || stateRecord.Status == stateFailed {
			status.CompletedAt = &stateRecord.UpdatedAt
		}

		// Optionally populate step information from step states
		stepStates, stepErr := e.store.GetStepStates(pipelineID)
		if stepErr == nil {
			for _, stepState := range stepStates {
				switch stepState.State {
				case state.StateCompleted:
					status.CompletedSteps = append(status.CompletedSteps, stepState.StepID)
				case state.StateFailed:
					status.FailedSteps = append(status.FailedSteps, stepState.StepID)
				case state.StateRunning, state.StateRetrying:
					status.CurrentStep = stepState.StepID
				}
			}
		}

		return status, nil
	}

	return nil, fmt.Errorf("pipeline %q not found", pipelineID)
}

// cleanupWorktrees removes any git worktrees created during pipeline execution.
func (e *DefaultPipelineExecutor) cleanupCompletedPipeline(pipelineID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.pipelines, pipelineID)
}

// ResumeWithValidation resumes a pipeline with full validation and error handling.
// When force is true, phase validation and stale artifact checks are skipped.
// When priorRunID is provided, artifact paths are resolved from that specific run's
// workspace directory instead of scanning for the most recent match.
func (e *DefaultPipelineExecutor) ResumeWithValidation(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string, fromStep string, force bool, priorRunID ...string) error {
	manager := NewResumeManager(e)
	return manager.ResumeFromStep(ctx, p, m, input, fromStep, force, priorRunID...)
}

// fireWebhooks sends an event to all matching dynamic webhooks (non-blocking).

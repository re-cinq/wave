package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/recinq/wave/internal/cost"
	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/scope"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/workspace"
	"github.com/recinq/wave/internal/worktree"
	"golang.org/x/sync/errgroup"
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
	// Retrospective generator for post-run analysis
	retroGenerator *retro.Generator
	// Cost ledger for per-run cost tracking and budget enforcement
	costLedger *cost.Ledger
	// Webhook runner for dynamic webhook delivery (non-blocking)
	webhookRunner *hooks.WebhookRunner
	// Task-level complexity from classifier (empty = no task-aware routing)
	taskComplexity string
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

// WithRetroGenerator sets the retrospective generator for post-run analysis.
func WithRetroGenerator(g *retro.Generator) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.retroGenerator = g }
}

// WithRegistry sets the adapter registry for per-step adapter resolution.
func WithRegistry(r *adapter.AdapterRegistry) ExecutorOption {
	return func(ex *DefaultPipelineExecutor) { ex.registry = r }
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
		runner:             runner,
		pipelines:          make(map[string]*PipelineExecution),
		deliverableTracker: deliverable.NewTracker(""),
	}
	for _, opt := range opts {
		opt(ex)
	}

	// If no registry was provided via WithRegistry, wrap the single runner
	if ex.registry == nil {
		ex.registry = adapter.NewSingleRunnerRegistry(runner)
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

// NewChildExecutor creates a fresh executor that shares the same adapter runner,
// event emitter, workspace manager, and configuration, but has independent
// execution state. Used for child pipeline invocation within matrix strategies.
func (e *DefaultPipelineExecutor) NewChildExecutor() *DefaultPipelineExecutor {
	return &DefaultPipelineExecutor{
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
		securityConfig:         e.securityConfig,
		pathValidator:          e.pathValidator,
		inputSanitizer:         e.inputSanitizer,
		securityLogger:         e.securityLogger,
		deliverableTracker:     deliverable.NewTracker(""),
		crossPipelineArtifacts: e.crossPipelineArtifacts,
		preserveWorkspace:      e.preserveWorkspace,
		skillStore:             e.skillStore,
		hookRunner:             e.hookRunner,
		autoApprove:            e.autoApprove,
		gateHandler:            e.gateHandler,
		retroGenerator:         e.retroGenerator,
	}
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
func (e *DefaultPipelineExecutor) validatePipelineAndCreateContext(p *Pipeline, m *manifest.Manifest, input string) (*pipelineSetup, error) {
	validator := &DAGValidator{}
	if err := validator.ValidateDAG(p); err != nil {
		return nil, fmt.Errorf("invalid pipeline DAG: %w", err)
	}

	// Validate thread/fidelity fields
	if errs := ValidateThreadFields(p); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, err := range errs {
			msgs[i] = err.Error()
		}
		return nil, fmt.Errorf("thread validation failed:\n  %s", strings.Join(msgs, "\n  "))
	}

	sortedSteps, err := validator.TopologicalSort(p)
	if err != nil {
		return nil, fmt.Errorf("failed to topologically sort steps: %w", err)
	}

	// Resolve named retry policies into concrete values before execution
	if err := ResolvePipelineRetryPolicies(p); err != nil {
		return nil, fmt.Errorf("retry policy resolution: %w", err)
	}

	// Apply step filter (--steps / --exclude) to the sorted step list
	if e.stepFilter != nil && e.stepFilter.IsActive() {
		if err := e.stepFilter.Validate(p); err != nil {
			return nil, err
		}
		sortedSteps = e.stepFilter.Apply(sortedSteps)
		if len(sortedSteps) == 0 {
			return nil, fmt.Errorf("step filter produced no runnable steps")
		}
	}

	// Initialize ETA calculator from historical step performance data
	stepIDs := make([]string, len(sortedSteps))
	for i, step := range sortedSteps {
		stepIDs[i] = step.ID
	}
	e.etaCalculator = NewETACalculator(e.store, p.Metadata.Name, stepIDs)

	// Create pipeline context early so forge variables are available for preflight tool resolution
	pipelineName := p.Metadata.Name
	pipelineID := e.runID
	if pipelineID == "" {
		pipelineID = GenerateRunID(pipelineName, m.Runtime.PipelineIDHashLength)
	}
	pipelineContext := newContextWithProject(pipelineID, pipelineName, "", m)
	pipelineContext.Input = input

	// Inject forge variables for unified pipeline template resolution
	forgeInfo := forge.DetectFromGitRemotesWithOverride(m.Metadata.Forge)
	InjectForgeVariables(pipelineContext, forgeInfo)

	// Resolve template placeholders in pipeline skills before validation.
	// Skills like "{{ project.skill }}" must be resolved to their actual values
	// (or empty string) before validateSkillRefs checks them against the store.
	// Build a new slice (do NOT mutate p.Skills — the Pipeline may be shared
	// across concurrent matrix child executors).
	resolvedPipelineSkills := make([]string, 0, len(p.Skills))
	for _, s := range p.Skills {
		r := pipelineContext.ResolvePlaceholders(s)
		if r != "" {
			resolvedPipelineSkills = append(resolvedPipelineSkills, r)
		}
	}

	return &pipelineSetup{
		pipelineID:             pipelineID,
		pipelineName:           pipelineName,
		sortedSteps:            sortedSteps,
		pipelineContext:        pipelineContext,
		forgeInfo:              forgeInfo,
		resolvedPipelineSkills: resolvedPipelineSkills,
	}, nil
}

// checkPipelinePreflights runs skill validation, tool/skill preflight checks, forge preflight
// checks, and token scope validation. It is the second phase of Execute.
func (e *DefaultPipelineExecutor) checkPipelinePreflights(_ context.Context, setup *pipelineSetup, p *Pipeline, m *manifest.Manifest) error {
	// Validate skill references at manifest and pipeline scopes (after template resolution)
	if errs := e.validateSkillRefs(setup.resolvedPipelineSkills, p.Metadata.Name, m); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, err := range errs {
			msgs[i] = err.Error()
		}
		return fmt.Errorf("skill validation failed:\n  %s", strings.Join(msgs, "\n  "))
	}

	// Preflight validation: check required tools and skills before execution
	if p.Requires != nil {
		checker := preflight.NewChecker(p.Requires.Skills)
		var tools []string
		if len(p.Requires.Tools) > 0 {
			for _, tool := range p.Requires.Tools {
				resolved := setup.pipelineContext.ResolvePlaceholders(tool)
				if resolved != "" {
					tools = append(tools, resolved)
				}
			}
		}
		skillNames := p.Requires.SkillNames()
		if len(tools) > 0 || len(skillNames) > 0 {
			results, err := checker.Run(tools, skillNames)
			for _, r := range results {
				e.emit(event.Event{
					Timestamp: time.Now(),
					State:     "preflight",
					Message:   r.Message,
				})
			}
			if err != nil {
				return err
			}
		}
	}

	// Forge preflight: block forge-dependent steps when no forge is configured
	if setup.forgeInfo.Type == forge.ForgeLocal {
		// Check pipeline name for forge prefix
		if ferr := preflight.CheckForgePipelineName(setup.forgeInfo, p.Metadata.Name); ferr != nil {
			e.emit(event.Event{
				Timestamp: time.Now(),
				State:     "preflight",
				Message:   ferr.Error(),
			})
			return ferr
		}

		// Build step inputs for forge dependency scanning
		forgeStepInputs := make([]preflight.ForgeStepInput, 0, len(setup.sortedSteps))
		for _, step := range setup.sortedSteps {
			var personaTools []string
			resolvedPersona := setup.pipelineContext.ResolvePlaceholders(step.Persona)
			if persona := m.GetPersona(resolvedPersona); persona != nil {
				personaTools = persona.Permissions.AllowedTools
			}
			forgeStepInputs = append(forgeStepInputs, preflight.ForgeStepInput{
				StepID:       step.ID,
				PersonaTools: personaTools,
				PromptSource: step.Exec.Source,
			})
		}
		if ferr := preflight.CheckForgeSteps(setup.forgeInfo, forgeStepInputs); ferr != nil {
			for _, s := range ferr.Steps {
				e.emit(event.Event{
					Timestamp: time.Now(),
					State:     "preflight",
					Message:   s.Reason,
				})
			}
			return ferr
		}
	}

	// Token scope validation: check persona token requirements before execution
	if setup.forgeInfo.Type != forge.ForgeUnknown {
		resolver := scope.NewResolver(setup.forgeInfo.Type)
		introspector := scope.NewIntrospector(setup.forgeInfo.Type)
		scopeValidator := scope.NewValidator(resolver, introspector, setup.forgeInfo, m.Runtime.Sandbox.EnvPassthrough)

		// Build persona scope map from manifest personas used in this pipeline
		personaScopes := make(map[string][]string)
		for _, step := range setup.sortedSteps {
			resolvedName := setup.pipelineContext.ResolvePlaceholders(step.Persona)
			if persona := m.GetPersona(resolvedName); persona != nil && len(persona.TokenScopes) > 0 {
				personaScopes[resolvedName] = persona.TokenScopes
			}
		}

		if len(personaScopes) > 0 {
			scopeResult, scopeErr := scopeValidator.ValidatePersonas(personaScopes)
			if scopeErr != nil {
				return fmt.Errorf("token scope validation error: %w", scopeErr)
			}
			for _, w := range scopeResult.Warnings {
				e.emit(event.Event{
					Timestamp: time.Now(),
					State:     "preflight",
					Message:   fmt.Sprintf("token scope warning: %s", w),
				})
			}
			if scopeResult.HasViolations() {
				return fmt.Errorf("%s", scopeResult.Error())
			}
		}
	}

	return nil
}

// initPipelineExecution creates the PipelineExecution object, starts the cancellation
// poller, initialises the deliverable tracker and compaction adapter, and registers
// the execution in the executor's pipeline map. It is the third phase of Execute.
// The returned context and cancel func must be used for all subsequent execution phases.
func (e *DefaultPipelineExecutor) initPipelineExecution(
	ctx context.Context,
	setup *pipelineSetup,
	p *Pipeline,
	m *manifest.Manifest,
	input string,
) (*PipelineExecution, context.Context, context.CancelFunc) {
	pipelineID := setup.pipelineID

	// Ontology staleness check: warn if ontology is older than latest commit
	if m.Ontology != nil && len(m.Ontology.Contexts) > 0 {
		if msg := e.checkOntologyStaleness(); msg != "" {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				State:      "warning",
				Message:    msg,
			})
		}
	}

	// Start cancellation poller for cross-process cancel support.
	// When another process (TUI, webui) writes a cancellation record to the DB,
	// this goroutine detects it and cancels the executor's context.
	runCtx := ctx
	cancel := context.CancelFunc(func() {})
	if e.store != nil && pipelineID != "" {
		runCtx, cancel = context.WithCancel(ctx)
		go e.pollCancellation(runCtx, pipelineID, cancel)
	}

	// Initialize deliverable tracker for this pipeline (only if not already set)
	if e.deliverableTracker == nil {
		e.deliverableTracker = deliverable.NewTracker(pipelineID)
	} else {
		// Update pipeline ID if tracker already exists
		e.deliverableTracker.SetPipelineID(pipelineID)
	}

	// Build compaction adapter for thread summary fidelity (reuse relay monitor's adapter if available)
	var threadCompactionAdapter relay.CompactionAdapter
	if e.relayMonitor != nil {
		threadCompactionAdapter = e.relayMonitor.Adapter()
	}

	execution := &PipelineExecution{
		Pipeline:          p,
		Manifest:          m,
		States:            make(map[string]string),
		Results:           make(map[string]map[string]interface{}),
		ArtifactPaths:     make(map[string]string),
		WorkspacePaths:    make(map[string]string),
		WorktreePaths:     make(map[string]*WorktreeInfo),
		AttemptContexts:   make(map[string]*AttemptContext),
		ReworkTransitions: make(map[string]string),
		ThreadManager:     NewThreadManager(threadCompactionAdapter),
		CircuitBreaker:    NewCircuitBreaker(m.Runtime.CircuitBreaker.Limit, m.Runtime.CircuitBreaker.TrackedClasses),
		Input:             input,
		Context:           setup.pipelineContext,
		Status: &PipelineStatus{
			ID:             pipelineID,
			PipelineName:   setup.pipelineName,
			State:          statePending,
			CompletedSteps: []string{},
			FailedSteps:    []string{},
			StartedAt:      time.Now(),
		},
	}

	for _, step := range p.Steps {
		execution.States[step.ID] = statePending
	}

	e.mu.Lock()
	e.pipelines[pipelineID] = execution
	e.lastExecution = execution
	e.mu.Unlock()

	return execution, runCtx, cancel
}

// setupPipelineRun seeds parent artifacts, saves initial DB state, sets up workspace,
// initialises hook/webhook runners, and fires run_start hooks.
// It is the fourth phase of Execute.
func (e *DefaultPipelineExecutor) setupPipelineRun(ctx context.Context, execution *PipelineExecution, p *Pipeline, m *manifest.Manifest) error {
	pipelineID := execution.Status.ID
	input := execution.Input

	// Seed parent artifact paths into child execution context.
	// When this executor is a child of a sub-pipeline step, the parent passes
	// artifact paths so child steps can resolve {{ artifacts.<name> }} templates.
	if e.parentArtifactPaths != nil {
		for name, path := range e.parentArtifactPaths {
			execution.Context.SetArtifactPath(name, path)
		}
	}

	if e.store != nil {
		_ = e.store.SavePipelineState(pipelineID, stateRunning, input)
	}

	execution.Status.State = stateRunning

	e.emit(event.Event{
		Timestamp:       time.Now(),
		PipelineID:      pipelineID,
		State:           "started",
		Message:         fmt.Sprintf("input=%q steps=%d", input, len(p.Steps)),
		TotalSteps:      len(p.Steps),
		CompletedSteps:  0,
		Adapter:         e.adapterOverride,
		ConfiguredModel: e.modelOverride,
	})

	// Ensure workspace root exists and is clean for this pipeline run
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	pipelineWsPath := filepath.Join(wsRoot, pipelineID)
	// Clean previous run artifacts to ensure fresh state (unless --preserve-workspace is set)
	if e.preserveWorkspace {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			State:      "warning",
			Message:    "--preserve-workspace active: stale workspace state may cause non-reproducible results",
		})
	} else {
		if err := os.RemoveAll(pipelineWsPath); err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				State:      "warning",
				Message:    fmt.Sprintf("failed to clean workspace: %v", err),
			})
		}
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

	// Initialize hook runner from manifest + pipeline hooks if not already set.
	// Pipeline-level hooks are appended after manifest-level hooks so they run
	// in addition to (and after) the global manifest hooks.
	if e.hookRunner == nil {
		merged := append([]hooks.LifecycleHookDef{}, m.Hooks...)
		merged = append(merged, p.Hooks...)
		if len(merged) > 0 {
			runner, err := hooks.NewHookRunner(merged, e.emitter)
			if err != nil {
				return fmt.Errorf("failed to initialize hook runner: %w (pipeline: %s)", err, pipelineID)
			}
			e.hookRunner = runner
		}
	}

	// Initialize webhook runner from state store (dynamic webhooks)
	if e.webhookRunner == nil && e.store != nil {
		webhooks, err := e.store.ListWebhooks()
		if err == nil && len(webhooks) > 0 {
			records := make([]hooks.WebhookRecord, len(webhooks))
			for i, wh := range webhooks {
				records[i] = hooks.WebhookRecord{
					ID:      wh.ID,
					Name:    wh.Name,
					URL:     wh.URL,
					Events:  wh.Events,
					Matcher: wh.Matcher,
					Headers: wh.Headers,
					Secret:  wh.Secret,
					Active:  wh.Active,
				}
			}
			e.webhookRunner = hooks.NewWebhookRunner(records, &webhookStoreAdapter{store: e.store})
		}
	}

	// Run run_start hooks
	startEvt := hooks.HookEvent{
		Type:       hooks.EventRunStart,
		PipelineID: pipelineID,
		Input:      input,
	}
	if e.hookRunner != nil {
		if _, err := e.hookRunner.RunHooks(ctx, startEvt); err != nil {
			return fmt.Errorf("run_start hook failed: %w", err)
		}
	}
	e.fireWebhooks(ctx, startEvt)

	return nil
}

// runSchedulingLoop iterates the topologically-sorted step list, finding and executing
// ready batches until all schedulable steps complete or an unrecoverable error occurs.
// Returns (schedulableSteps, error) — schedulableSteps is needed by finalizePipelineExecution.
func (e *DefaultPipelineExecutor) runSchedulingLoop(ctx context.Context, execution *PipelineExecution, sortedSteps []*Step) (int, error) {
	pipelineID := execution.Status.ID

	completed := make(map[string]bool, len(sortedSteps))
	completedCount := 0

	// Count schedulable steps (excludes rework-only steps which run only via rework trigger)
	schedulableSteps := 0
	for _, step := range sortedSteps {
		if !step.ReworkOnly {
			schedulableSteps++
		}
	}

	for completedCount < schedulableSteps {
		ready := e.findReadySteps(sortedSteps, completed)
		if len(ready) == 0 {
			e.cleanupCompletedPipeline(pipelineID)
			pending := make([]string, 0)
			for _, step := range sortedSteps {
				if !completed[step.ID] && !step.ReworkOnly {
					pending = append(pending, step.ID)
				}
			}
			return 0, fmt.Errorf("deadlock: %d step(s) stuck waiting for dependencies — pending: %v", len(pending), pending)
		}

		if err := e.executeStepBatch(ctx, execution, ready); err != nil {
			// reQueueError means gate routing reset steps to pending — re-enter the scheduling loop
			var reQueueErr *reQueueError
			if errors.As(err, &reQueueErr) {
				// Remove re-queued steps from the completed map so findReadySteps can re-schedule them
				for stepID := range reQueueErr.ResetSteps {
					if completed[stepID] {
						delete(completed, stepID)
						completedCount--
					}
				}
				// Mark the gate step itself as completed (it already set its state)
				for _, step := range ready {
					execution.mu.Lock()
					stepState := execution.States[step.ID]
					execution.mu.Unlock()
					if (stepState == stateCompleted || stepState == stateCompletedEmpty) && !completed[step.ID] {
						completed[step.ID] = true
						completedCount++
					}
				}
				continue
			}
			execution.Status.State = stateFailed
			// Identify which step(s) failed from the batch
			var failedStepID string
			for _, step := range ready {
				execution.mu.Lock()
				stepState := execution.States[step.ID]
				execution.mu.Unlock()
				if stepState == stateFailed || stepState == stateRunning || stepState == stateRetrying {
					execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
					if failedStepID == "" {
						failedStepID = step.ID
					}
				}
			}
			// Fallback: if no step matched expected states, use the first step
			// in the batch — an error was returned so at least one step failed.
			if failedStepID == "" && len(ready) > 0 {
				failedStepID = ready[0].ID
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, failedStepID)
			}
			if e.store != nil {
				_ = e.store.SavePipelineState(pipelineID, stateFailed, execution.Input)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     failedStepID,
				State:      stateFailed,
				Message:    err.Error(),
			})
			// Generate retrospective for failed runs — these are the most valuable
			if e.retroGenerator != nil {
				e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
			}
			e.cleanupCompletedPipeline(pipelineID)
			return 0, &StepExecutionError{StepID: failedStepID, Err: err}
		}

		// Process batch results: steps may have completed, failed (optional), or been skipped
		for _, step := range ready {
			completed[step.ID] = true
			completedCount++

			execution.mu.Lock()
			stepState := execution.States[step.ID]
			execution.mu.Unlock()

			if stepState == stateFailed || stepState == stateSkipped {
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
			} else {
				execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
			}
		}

		// Sync rework-only steps that were triggered during this batch
		for _, step := range sortedSteps {
			if !step.ReworkOnly || completed[step.ID] {
				continue
			}
			execution.mu.Lock()
			stepState := execution.States[step.ID]
			execution.mu.Unlock()
			if stepState == stateCompleted || stepState == stateCompletedEmpty || stepState == stateFailed {
				completed[step.ID] = true
			}
		}

		// Skip steps whose dependencies include failed/skipped steps (transitive propagation)
		e.skipDependentSteps(execution, sortedSteps, completed, &completedCount)

		e.emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     pipelineID,
			State:          stateRunning,
			TotalSteps:     schedulableSteps,
			CompletedSteps: completedCount,
			Progress:       (completedCount * 100) / schedulableSteps,
			Message:        fmt.Sprintf("%d/%d steps completed", completedCount, schedulableSteps),
		})
	}

	return schedulableSteps, nil
}

// finalizePipelineExecution records completion status, fires terminal hooks, generates
// a retrospective, and cleans up in-memory state. It is the final phase of Execute.
func (e *DefaultPipelineExecutor) finalizePipelineExecution(_ context.Context, execution *PipelineExecution, schedulableSteps int) {
	pipelineID := execution.Status.ID
	input := execution.Input

	now := time.Now()
	execution.Status.CompletedAt = &now

	// Pipeline succeeds if no required steps failed
	if e.hasRequiredFailures(execution) {
		execution.Status.State = stateFailed
		if e.store != nil {
			_ = e.store.SavePipelineState(pipelineID, stateFailed, input)
		}
		// Run run_failed hooks with detached context (non-blocking by default).
		e.runTerminalHooks(hooks.HookEvent{
			Type:       hooks.EventRunFailed,
			PipelineID: pipelineID,
			Input:      input,
		})
	} else {
		// If every step is either completed_empty or non-worktree, the pipeline
		// itself is completed_empty — the run produced no code changes.
		pipelineState := stateCompleted
		execution.mu.Lock()
		allEmpty := true
		hasWorktreeStep := false
		for _, st := range execution.States {
			switch st {
			case stateCompletedEmpty:
				hasWorktreeStep = true
			case stateCompleted:
				allEmpty = false
			}
		}
		execution.mu.Unlock()
		if hasWorktreeStep && allEmpty {
			pipelineState = stateCompletedEmpty
		}

		execution.Status.State = pipelineState
		if e.store != nil {
			_ = e.store.SavePipelineState(pipelineID, pipelineState, input)
		}
		// Run run_completed hooks with detached context (non-blocking by default).
		e.runTerminalHooks(hooks.HookEvent{
			Type:       hooks.EventRunCompleted,
			PipelineID: pipelineID,
			Input:      input,
		})
	}

	elapsed := time.Since(execution.Status.StartedAt).Milliseconds()
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      execution.Status.State,
		DurationMs: elapsed,
		Message:    fmt.Sprintf("%d steps completed", schedulableSteps),
	})

	// Generate retrospective (non-blocking)
	if e.retroGenerator != nil {
		e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
	}

	// Clean up completed pipeline from in-memory storage to prevent memory leak
	e.cleanupCompletedPipeline(pipelineID)
}

// executeGraphPipeline runs a graph-mode pipeline using edge-following execution
// instead of topological sort. Activated when steps define edges or conditional types.
func (e *DefaultPipelineExecutor) executeGraphPipeline(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string) error {
	validator := &DAGValidator{}
	if err := validator.ValidateGraph(p); err != nil {
		return fmt.Errorf("invalid graph pipeline: %w", err)
	}

	// Create pipeline context (shared setup with DAG mode)
	pipelineName := p.Metadata.Name
	pipelineID := e.runID
	if pipelineID == "" {
		pipelineID = GenerateRunID(pipelineName, m.Runtime.PipelineIDHashLength)
	}
	pipelineContext := newContextWithProject(pipelineID, pipelineName, "", m)
	pipelineContext.Input = input

	// Inject forge variables
	forgeInfo := forge.DetectFromGitRemotesWithOverride(m.Metadata.Forge)
	InjectForgeVariables(pipelineContext, forgeInfo)

	// Forge preflight: block forge-dependent steps when no forge is configured
	if forgeInfo.Type == forge.ForgeLocal {
		if ferr := preflight.CheckForgePipelineName(forgeInfo, p.Metadata.Name); ferr != nil {
			return ferr
		}
		forgeStepInputs := make([]preflight.ForgeStepInput, 0, len(p.Steps))
		for i := range p.Steps {
			step := &p.Steps[i]
			var personaTools []string
			resolvedPersona := pipelineContext.ResolvePlaceholders(step.Persona)
			if persona := m.GetPersona(resolvedPersona); persona != nil {
				personaTools = persona.Permissions.AllowedTools
			}
			forgeStepInputs = append(forgeStepInputs, preflight.ForgeStepInput{
				StepID:       step.ID,
				PersonaTools: personaTools,
				PromptSource: step.Exec.Source,
			})
		}
		if ferr := preflight.CheckForgeSteps(forgeInfo, forgeStepInputs); ferr != nil {
			return ferr
		}
	}

	// Initialize deliverable tracker
	if e.deliverableTracker == nil {
		e.deliverableTracker = deliverable.NewTracker(pipelineID)
	} else {
		e.deliverableTracker.SetPipelineID(pipelineID)
	}

	// Build compaction adapter for thread summary fidelity (reuse relay monitor's adapter if available)
	var threadCompactionAdapter relay.CompactionAdapter
	if e.relayMonitor != nil {
		threadCompactionAdapter = e.relayMonitor.Adapter()
	}

	execution := &PipelineExecution{
		Pipeline:          p,
		Manifest:          m,
		States:            make(map[string]string),
		Results:           make(map[string]map[string]interface{}),
		ArtifactPaths:     make(map[string]string),
		WorkspacePaths:    make(map[string]string),
		WorktreePaths:     make(map[string]*WorktreeInfo),
		AttemptContexts:   make(map[string]*AttemptContext),
		ReworkTransitions: make(map[string]string),
		ThreadManager:     NewThreadManager(threadCompactionAdapter),
		CircuitBreaker:    NewCircuitBreaker(m.Runtime.CircuitBreaker.Limit, m.Runtime.CircuitBreaker.TrackedClasses),
		Input:             input,
		Context:           pipelineContext,
		Status: &PipelineStatus{
			ID:             pipelineID,
			PipelineName:   pipelineName,
			State:          statePending,
			CompletedSteps: []string{},
			FailedSteps:    []string{},
			StartedAt:      time.Now(),
		},
	}

	for _, step := range p.Steps {
		execution.States[step.ID] = statePending
	}

	e.mu.Lock()
	e.pipelines[pipelineID] = execution
	e.lastExecution = execution
	e.mu.Unlock()

	// Seed parent artifact paths into child execution context.
	// When this executor is a child of a sub-pipeline step, the parent passes
	// artifact paths so child steps can resolve {{ artifacts.<name> }} templates.
	if e.parentArtifactPaths != nil {
		for name, path := range e.parentArtifactPaths {
			execution.Context.SetArtifactPath(name, path)
		}
	}

	if e.store != nil {
		_ = e.store.SavePipelineState(pipelineID, stateRunning, input)
	}

	execution.Status.State = stateRunning

	e.emit(event.Event{
		Timestamp:       time.Now(),
		PipelineID:      pipelineID,
		State:           "started",
		Message:         fmt.Sprintf("graph-mode pipeline input=%q steps=%d", input, len(p.Steps)),
		TotalSteps:      len(p.Steps),
		CompletedSteps:  0,
		Adapter:         e.adapterOverride,
		ConfiguredModel: e.modelOverride,
	})

	// Initialize hook runner from manifest + pipeline hooks if not already set.
	if e.hookRunner == nil {
		merged := append([]hooks.LifecycleHookDef{}, m.Hooks...)
		merged = append(merged, p.Hooks...)
		if len(merged) > 0 {
			runner, err := hooks.NewHookRunner(merged, e.emitter)
			if err != nil {
				return fmt.Errorf("failed to initialize hook runner: %w (pipeline: %s)", err, pipelineID)
			}
			e.hookRunner = runner
		}
	}

	// Ensure workspace root exists
	wsRoot := m.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}
	pipelineWsPath := filepath.Join(wsRoot, pipelineID)
	if !e.preserveWorkspace {
		_ = os.RemoveAll(pipelineWsPath)
	}
	if err := os.MkdirAll(pipelineWsPath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create and run graph walker
	gw := NewGraphWalker(p)

	// Load initial visit counts from state store (resume support)
	var initialVisitCounts map[string]int
	if e.store != nil {
		initialVisitCounts = make(map[string]int)
		for _, step := range p.Steps {
			count, err := e.store.GetStepVisitCount(pipelineID, step.ID)
			if err == nil && count > 0 {
				initialVisitCounts[step.ID] = count
			}
		}
	}

	// Define the step executor callback
	stepExecutor := func(ctx context.Context, step *Step) (*StepResult, error) {
		// Handle command steps
		if step.Type == StepTypeCommand || step.Script != "" {
			result, err := e.executeCommandStep(ctx, execution, step)
			if err != nil {
				return result, err
			}
			// Run handover contract validation for command steps.
			// Command steps run in the project root (or mount target), so resolve
			// contract sources against the command's actual working directory.
			contractDir := resolveCommandWorkDir(execution.WorkspacePaths[step.ID], step)
			adapterResult := &adapter.AdapterResult{}
			if cErr := e.validateStepContracts(ctx, execution, step, contractDir, nil, execution.Status.ID, "", time.Now(), adapterResult); cErr != nil {
				return result, cErr
			}
			return result, nil
		}

		// Execute regular steps via the existing step execution path
		err := e.executeStep(ctx, execution, step)

		result := &StepResult{
			StepID:  step.ID,
			Context: make(map[string]string),
		}

		if err != nil {
			result.Outcome = "failure"
			result.Error = err
			return result, err
		}

		result.Outcome = "success"
		execution.mu.Lock()
		execution.Status.CompletedSteps = append(execution.Status.CompletedSteps, step.ID)
		execution.mu.Unlock()
		return result, nil
	}

	err := gw.Walk(ctx, stepExecutor, initialVisitCounts)

	// Persist final visit counts
	if e.store != nil {
		for stepID, count := range gw.VisitCounts() {
			if vcErr := e.store.SaveStepVisitCount(pipelineID, stepID, count); vcErr != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     stepID,
					State:      "warning",
					Message:    fmt.Sprintf("failed to persist visit count for step %q: %v", stepID, vcErr),
				})
			}
		}
	}

	now := time.Now()
	execution.Status.CompletedAt = &now

	if err != nil {
		execution.Status.State = stateFailed
		if e.store != nil {
			_ = e.store.SavePipelineState(pipelineID, stateFailed, input)
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			State:      stateFailed,
			Message:    err.Error(),
		})
		// Generate retrospective for failed runs — these are the most valuable
		if e.retroGenerator != nil {
			e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
		}
		e.cleanupCompletedPipeline(pipelineID)
		return err
	}

	execution.Status.State = stateCompleted
	if e.store != nil {
		_ = e.store.SavePipelineState(pipelineID, stateCompleted, input)
	}

	elapsed := time.Since(execution.Status.StartedAt).Milliseconds()
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      stateCompleted,
		DurationMs: elapsed,
		Message:    fmt.Sprintf("graph pipeline completed: %d steps visited", gw.totalVisits),
	})

	// Generate retrospective (non-blocking)
	if e.retroGenerator != nil {
		e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
	}

	e.cleanupCompletedPipeline(pipelineID)
	return nil
}

// executeCommandStep runs a shell script command step and captures its output.
// executeCommandStep runs a shell script command step and captures its output.
// Command steps don’t use adapters — they execute scripts directly via os/exec.
//
// Security: The resolved script is sanitized via InputSanitizer to detect prompt
// injection and shell metacharacter abuse. The subprocess environment is filtered
// to only include variables listed in Runtime.Sandbox.EnvPassthrough.
func (e *DefaultPipelineExecutor) executeCommandStep(ctx context.Context, execution *PipelineExecution, step *Step) (*StepResult, error) {
	pipelineID := execution.Status.ID

	execution.mu.Lock()
	execution.States[step.ID] = stateRunning
	execution.Status.CurrentStep = step.ID
	execution.mu.Unlock()

	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateRunning, "")
	}

	// Audit log: command step start
	if e.logger != nil {
		_ = e.logger.LogStepStart(pipelineID, step.ID, "command", nil)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      stateRunning,
		Message:    fmt.Sprintf("executing command step: %s", step.Script),
	})

	// Resolve template placeholders in the script
	script := step.Script
	if execution.Context != nil {
		script = execution.Context.ResolvePlaceholders(script)
	}

	// SECURITY: Reject command step execution when no sanitizer is configured.
	// Template resolution can introduce user-controlled content that must be
	// sanitized before shell execution.
	if e.inputSanitizer == nil {
		return nil, fmt.Errorf("command step %q: refusing to execute without input sanitizer", step.ID)
	}

	// SECURITY: Sanitize the resolved script to detect injection attempts.
	// Template resolution can introduce user-controlled content (e.g. issue titles,
	// branch names) that could contain shell metacharacters or injection payloads.
	if e.inputSanitizer != nil {
		record, sanitized, err := e.inputSanitizer.SanitizeInput(script, "command_script")
		if err != nil {
			// Sanitization rejected the input (strict mode / prompt injection detected)
			if e.securityLogger != nil {
				e.securityLogger.LogViolation(
					string(security.ViolationPromptInjection),
					string(security.SourceUserInput),
					fmt.Sprintf("command step %q script rejected by sanitizer: %v", step.ID, err),
					security.SeverityCritical,
					true,
				)
			}
			return nil, fmt.Errorf("command step %q: script sanitization failed: %w", step.ID, err)
		}
		if record != nil && record.ChangesDetected {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      stateRunning,
				Message:    fmt.Sprintf("command script sanitized (risk_score=%d, rules=%v)", record.RiskScore, record.SanitizationRules),
			})
		}
		script = sanitized
	}

	// Create workspace for the step
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace for step %q: %w", step.ID, err)
	}
	execution.mu.Lock()
	execution.WorkspacePaths[step.ID] = workspacePath
	execution.mu.Unlock()

	// Resolve the working directory for the command. For mount-based
	// workspaces the project files live under the mount target (e.g.
	// workspacePath/project/), so we set CWD to the project mount
	// directory rather than the bare workspace root.
	cmdDir := resolveCommandWorkDir(workspacePath, step)

	// Execute the script
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Dir = cmdDir

	// SECURITY: Filter environment to only EnvPassthrough variables.
	// Prevents leaking secrets, API keys, or other sensitive environment
	// variables into the command subprocess.
	cmd.Env = filterEnvPassthrough(execution.Manifest.Runtime.Sandbox.EnvPassthrough)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Audit log: tool call (the shell command)
	if e.logger != nil {
		_ = e.logger.LogToolCall(pipelineID, step.ID, "sh", script)
	}

	execErr := cmd.Run()
	duration := time.Since(startTime)

	result := &StepResult{
		StepID:  step.ID,
		Stdout:  stdout.String(),
		Context: make(map[string]string),
	}

	// Store stdout as a result
	execution.mu.Lock()
	if execution.Results[step.ID] == nil {
		execution.Results[step.ID] = make(map[string]interface{})
	}
	execution.Results[step.ID]["stdout"] = stdout.String()
	execution.Results[step.ID]["stderr"] = stderr.String()
	execution.mu.Unlock()

	if execErr != nil {
		result.Outcome = "failure"
		result.Error = execErr

		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, execErr.Error())
		}

		// Audit log: step end with failure
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		if e.logger != nil {
			_ = e.logger.LogStepEnd(pipelineID, step.ID, stateFailed, duration, exitCode, len(stdout.String()), 0, execErr.Error())
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      stateFailed,
			Message:    fmt.Sprintf("command failed: %v\nstderr: %s", execErr, stderr.String()),
		})

		return result, execErr
	}

	result.Outcome = "success"

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	// Audit log: step end with success
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if e.logger != nil {
		_ = e.logger.LogStepEnd(pipelineID, step.ID, stateCompleted, duration, exitCode, len(stdout.String()), 0, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      stateCompleted,
		Message:    "command completed successfully",
	})

	return result, nil
}

// filterEnvPassthrough builds a minimal environment containing only the
// variables named in the passthrough list. This prevents command steps from
// inheriting the full parent environment which may contain secrets.
// PATH is always included to ensure basic command resolution works.
func filterEnvPassthrough(passthrough []string) []string {
	// Always include PATH and essential build/runtime vars that commands need.
	essentials := []string{"PATH", "HOME", "USER", "TMPDIR",
		"GOPATH", "GOMODCACHE", "GOCACHE", "GOROOT",
		"XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME"}
	allowed := make(map[string]bool, len(passthrough)+len(essentials))
	for _, name := range essentials {
		allowed[name] = true
	}
	for _, name := range passthrough {
		allowed[name] = true
	}

	var filtered []string
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if ok && allowed[name] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// findReadySteps returns all steps whose dependencies are satisfied (all deps in completed set).
// Rework-only steps are excluded from normal DAG scheduling — they only run via rework trigger.
func (e *DefaultPipelineExecutor) findReadySteps(steps []*Step, completed map[string]bool) []*Step {
	var ready []*Step
	for _, step := range steps {
		if completed[step.ID] {
			continue
		}
		if step.ReworkOnly {
			continue
		}
		allDepsReady := true
		for _, dep := range step.Dependencies {
			if !completed[dep] {
				allDepsReady = false
				break
			}
		}
		if allDepsReady {
			ready = append(ready, step)
		}
	}
	return ready
}

// skipDependentSteps finds steps whose dependencies include a failed or skipped step
// and marks them as skipped. Propagates transitively until no more steps are affected.
func (e *DefaultPipelineExecutor) skipDependentSteps(execution *PipelineExecution, allSteps []*Step, completed map[string]bool, completedCount *int) {
	pipelineID := execution.Status.ID
	changed := true
	for changed {
		changed = false
		for _, step := range allSteps {
			if completed[step.ID] {
				continue
			}
			// Check if all dependencies are in the completed set
			allDepsComplete := true
			hasFailedDep := false
			for _, dep := range step.Dependencies {
				if !completed[dep] {
					allDepsComplete = false
					break
				}
				execution.mu.Lock()
				depState := execution.States[dep]
				execution.mu.Unlock()
				if depState == stateFailed || depState == stateSkipped {
					hasFailedDep = true
				}
			}
			if allDepsComplete && hasFailedDep {
				execution.mu.Lock()
				execution.States[step.ID] = stateSkipped
				execution.mu.Unlock()
				if e.store != nil {
					_ = e.store.SaveStepState(pipelineID, step.ID, state.StateSkipped, "dependency failed")
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateSkipped,
					Message:    "skipped: dependency failed",
				})
				completed[step.ID] = true
				*completedCount++
				execution.Status.FailedSteps = append(execution.Status.FailedSteps, step.ID)
				changed = true
			}
		}
	}
}

// hasRequiredFailures returns true if any non-optional step has failed.
func (e *DefaultPipelineExecutor) hasRequiredFailures(execution *PipelineExecution) bool {
	// Build a lookup of step optional status
	stepOptional := make(map[string]bool, len(execution.Pipeline.Steps))
	for i := range execution.Pipeline.Steps {
		stepOptional[execution.Pipeline.Steps[i].ID] = execution.Pipeline.Steps[i].Optional
	}

	execution.mu.Lock()
	defer execution.mu.Unlock()
	for stepID, stepState := range execution.States {
		if stepState == stateFailed {
			// A failed step is a required failure if the step itself is not optional
			// AND it was not skipped due to dependency propagation
			if !stepOptional[stepID] {
				return true
			}
		}
	}
	return false
}

// executeStepBatch runs a batch of ready steps. If the batch has a single step,
// it runs directly to avoid goroutine overhead. Otherwise, it launches concurrent
// goroutines via errgroup and returns the first error (cancelling remaining steps).
func (e *DefaultPipelineExecutor) executeStepBatch(ctx context.Context, execution *PipelineExecution, steps []*Step) error {
	if len(steps) == 1 {
		return e.executeStep(ctx, execution, steps[0])
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, step := range steps {
		step := step
		g.Go(func() error {
			return e.executeStep(gctx, execution, step)
		})
	}
	return g.Wait()
}

func (e *DefaultPipelineExecutor) executeStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID
	execution.mu.Lock()
	execution.States[step.ID] = stateRunning
	execution.Status.CurrentStep = step.ID
	execution.mu.Unlock()

	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateRunning, "")
	}

	// Check if this step uses concurrency (before matrix check — mutually exclusive)
	if step.Concurrency > 1 {
		return e.executeConcurrentStep(ctx, execution, step)
	}

	// Check if this step uses a matrix strategy
	if step.Strategy != nil && step.Strategy.Type == "matrix" {
		return e.executeMatrixStep(ctx, execution, step)
	}

	// Composition step: delegate to sub-pipeline execution
	if step.IsCompositionStep() {
		return e.executeCompositionStep(ctx, execution, step)
	}

	// Command step: execute shell script directly (no adapter/persona needed).
	// This mirrors the graph walker dispatch in executeGraphPipeline.
	if step.Type == StepTypeCommand || step.Script != "" {
		result, err := e.executeCommandStep(ctx, execution, step)
		if err != nil {
			return err
		}
		if result != nil && result.Outcome == "failure" {
			return result.Error
		}
		// Run handover contract validation (same as persona steps).
		// Resolve against the command's actual working directory, not the workspace root.
		contractDir := resolveCommandWorkDir(execution.WorkspacePaths[step.ID], step)
		adapterResult := &adapter.AdapterResult{}
		if cErr := e.validateStepContracts(ctx, execution, step, contractDir, nil, pipelineID, "", time.Now(), adapterResult); cErr != nil {
			return cErr
		}
		return nil
	}

	// Run step_start hooks
	stepStartEvt := hooks.HookEvent{
		Type:       hooks.EventStepStart,
		PipelineID: pipelineID,
		StepID:     step.ID,
		Input:      execution.Input,
	}
	if e.hookRunner != nil {
		if _, err := e.hookRunner.RunHooks(ctx, stepStartEvt); err != nil {
			execution.mu.Lock()
			execution.States[step.ID] = stateFailed
			execution.mu.Unlock()
			if e.store != nil {
				_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
			}
			return fmt.Errorf("step_start hook failed: %w", err)
		}
	}
	e.fireWebhooks(ctx, stepStartEvt)

	maxAttempts := step.Retry.EffectiveMaxAttempts()

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Don't retry if the parent context is already cancelled
			if ctx.Err() != nil {
				return fmt.Errorf("context cancelled, skipping retry: %w", lastErr)
			}
			execution.mu.Lock()
			execution.States[step.ID] = stateRetrying
			execution.mu.Unlock()
			if e.store != nil {
				_ = e.store.SaveStepState(pipelineID, step.ID, state.StateRetrying, "")
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      stateRetrying,
				Message:    fmt.Sprintf("attempt %d/%d", attempt, maxAttempts),
			})
			// Run step_retrying hooks (non-blocking by default)
			if e.hookRunner != nil {
				e.hookRunner.RunHooks(ctx, hooks.HookEvent{
					Type:       hooks.EventStepRetrying,
					PipelineID: pipelineID,
					StepID:     step.ID,
					Input:      execution.Input,
				})
			}
			time.Sleep(step.Retry.ComputeDelay(attempt))
		}

		// Record attempt start
		attemptStart := time.Now()
		if e.store != nil {
			_ = e.store.RecordStepAttempt(&state.StepAttemptRecord{
				RunID:     pipelineID,
				StepID:    step.ID,
				Attempt:   attempt,
				State:     stateRunning,
				StartedAt: attemptStart,
			})
		}

		// Start progress ticker for smooth animation updates during step execution
		cancelTicker := e.startProgressTicker(ctx, pipelineID, step.ID)

		// Start stall watchdog if configured
		stepCtx := ctx
		var watchdog *StallWatchdog
		if stallTimeout := e.parseStallTimeout(execution.Manifest); stallTimeout > 0 {
			watchdog = NewStallWatchdog(stallTimeout)
			stepCtx = watchdog.Start(stepCtx)
		}

		// Store watchdog on execution so runStepExecution can wire NotifyActivity
		execution.mu.Lock()
		execution.Watchdog = watchdog
		execution.mu.Unlock()

		err := e.runStepExecution(stepCtx, execution, step)

		// Stop stall watchdog and clear reference
		if watchdog != nil {
			watchdog.Stop()
		}
		execution.mu.Lock()
		execution.Watchdog = nil
		execution.mu.Unlock()

		// Stop progress ticker when step completes
		cancelTicker()

		attemptDuration := time.Since(attemptStart)

		if err != nil {
			lastErr = err

			// Classify the failure for intelligent retry decisions.
			// Use stepCtx (watchdog-derived) so stall cancellation is detected.
			failureClass := ClassifyStepFailure(err, nil, stepCtx.Err())

			// Record failed attempt with pipeline-level failure class
			if e.store != nil {
				completedAt := time.Now()
				_ = e.store.RecordStepAttempt(&state.StepAttemptRecord{
					RunID:        pipelineID,
					StepID:       step.ID,
					Attempt:      attempt,
					State:        stateFailed,
					ErrorMessage: err.Error(),
					FailureClass: failureClass,
					DurationMs:   attemptDuration.Milliseconds(),
					StartedAt:    attemptStart,
					CompletedAt:  &completedAt,
				})
			}

			// Check circuit breaker — if same failure fingerprint repeats too many times, stop
			if execution.CircuitBreaker != nil {
				fp := NormalizeFingerprint(step.ID, failureClass, err.Error())
				if execution.CircuitBreaker.Record(fp, failureClass) {
					e.emit(event.Event{
						Timestamp:    time.Now(),
						PipelineID:   pipelineID,
						StepID:       step.ID,
						State:        event.StateFailed,
						FailureClass: failureClass,
						Message:      fmt.Sprintf("circuit breaker tripped: same failure repeated %d times", execution.CircuitBreaker.Limit()),
					})
					// Fall through to on_failure handling below by exhausting attempts
					attempt = maxAttempts
				}
			}

			// Skip remaining retries for non-retryable failure classes
			if !IsRetryable(failureClass) && attempt < maxAttempts {
				e.emit(event.Event{
					Timestamp:    time.Now(),
					PipelineID:   pipelineID,
					StepID:       step.ID,
					State:        event.StateFailed,
					FailureClass: failureClass,
					Message:      fmt.Sprintf("non-retryable failure class %q, skipping remaining retries", failureClass),
				})
				attempt = maxAttempts
			}

			if attempt < maxAttempts {
				// Record retry decision
				e.recordDecision(pipelineID, step.ID, "retry",
					fmt.Sprintf("retrying step %s (attempt %d/%d)", step.ID, attempt+1, maxAttempts),
					fmt.Sprintf("failure class %q is retryable, attempts remaining", failureClass),
					map[string]interface{}{
						"attempt":       attempt,
						"max_attempts":  maxAttempts,
						"failure_class": failureClass,
						"error":         err.Error(),
					},
				)
				// Always inject failure context into the next retry attempt.
				// Previously gated behind AdaptPrompt, but contract failures
				// are the most common retry trigger and agents need to know
				// *what* failed to avoid starting from scratch.
				{
					errMsg := err.Error()
					// Capture stdout tail from results if available
					stdoutTail := ""
					execution.mu.Lock()
					if result, ok := execution.Results[step.ID]; ok {
						if stdout, ok := result["stdout"].(string); ok {
							if len(stdout) > maxStdoutTailChars {
								stdoutTail = stdout[len(stdout)-maxStdoutTailChars:]
							} else {
								stdoutTail = stdout
							}
						}
					}
					execution.mu.Unlock()

					// Extract contract-specific errors when the failure came
					// from contract validation so the agent gets actionable
					// detail about which contract failed and why.
					var contractErrors []string
					if strings.Contains(errMsg, "contract validation failed") {
						inner := err
						for uw := errors.Unwrap(inner); uw != nil; uw = errors.Unwrap(inner) {
							inner = uw
						}
						contractErrors = append(contractErrors, inner.Error())
					}

					execution.mu.Lock()
					execution.AttemptContexts[step.ID] = &AttemptContext{
						Attempt:        attempt + 1,
						MaxAttempts:    maxAttempts,
						PriorError:     errMsg,
						FailureClass:   failureClass,
						PriorStdout:    stdoutTail,
						ContractErrors: contractErrors,
					}
					execution.mu.Unlock()
				}
				continue
			}

			// All attempts exhausted — apply on_failure policy
			e.recordDecision(pipelineID, step.ID, "retry",
				fmt.Sprintf("all %d attempts exhausted for step %s", maxAttempts, step.ID),
				fmt.Sprintf("applying on_failure policy after %d failed attempts", maxAttempts),
				map[string]interface{}{
					"max_attempts":  maxAttempts,
					"failure_class": failureClass,
					"last_error":    err.Error(),
				},
			)
			onFailure := step.Retry.OnFailure
			if onFailure == "" {
				if step.Optional {
					onFailure = OnFailureContinue
				} else {
					onFailure = OnFailureFail
				}
			}

			switch onFailure {
			case OnFailureSkip:
				execution.mu.Lock()
				execution.States[step.ID] = stateSkipped
				execution.mu.Unlock()
				if e.store != nil {
					_ = e.store.SaveStepState(pipelineID, step.ID, state.StateSkipped, err.Error())
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateSkipped,
					Message:    fmt.Sprintf("step skipped after %d failed attempts: %s", maxAttempts, err.Error()),
				})
				e.recordOntologyUsage(execution, step, "skipped")
				return nil

			case OnFailureContinue:
				execution.mu.Lock()
				execution.States[step.ID] = stateFailed
				execution.mu.Unlock()
				if e.store != nil {
					_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateFailed,
					Message:    fmt.Sprintf("step failed after %d attempts but pipeline continues: %s", maxAttempts, err.Error()),
				})
				e.recordOntologyUsage(execution, step, "failed")
				return nil

			case OnFailureRework:
				return e.executeReworkStep(ctx, execution, step, lastErr, attemptDuration)

			default: // OnFailureFail
				execution.mu.Lock()
				execution.States[step.ID] = stateFailed
				execution.mu.Unlock()
				if e.store != nil {
					_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
				}
				// Run step_failed hooks and webhooks (non-blocking by default)
				stepFailedEvt := hooks.HookEvent{
					Type:       hooks.EventStepFailed,
					PipelineID: pipelineID,
					StepID:     step.ID,
					Input:      execution.Input,
					Error:      lastErr.Error(),
				}
				if e.hookRunner != nil {
					e.hookRunner.RunHooks(ctx, stepFailedEvt)
				}
				e.fireWebhooks(ctx, stepFailedEvt)
				e.recordOntologyUsage(execution, step, "failed")
				return lastErr
			}
		}

		// Record successful attempt
		if e.store != nil {
			completedAt := time.Now()
			_ = e.store.RecordStepAttempt(&state.StepAttemptRecord{
				RunID:       pipelineID,
				StepID:      step.ID,
				Attempt:     attempt,
				State:       "succeeded",
				DurationMs:  attemptDuration.Milliseconds(),
				StartedAt:   attemptStart,
				CompletedAt: &completedAt,
			})
		}

		// Clear attempt context on success
		execution.mu.Lock()
		delete(execution.AttemptContexts, step.ID)
		execution.mu.Unlock()

		execution.mu.Lock()
		execution.States[step.ID] = stateCompleted
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
		}

		// Record checkpoint for fork/rewind support
		if e.store != nil {
			stepIndex := -1
			for i, s := range execution.Pipeline.Steps {
				if s.ID == step.ID {
					stepIndex = i
					break
				}
			}
			recorder := &CheckpointRecorder{store: e.store}
			recorder.Record(execution, step, stepIndex)
		}

		// Record step completion for ETA calculation
		if e.etaCalculator != nil {
			e.etaCalculator.RecordStepCompletion(step.ID, attemptDuration.Milliseconds())
			e.emit(event.Event{
				Timestamp:       time.Now(),
				PipelineID:      pipelineID,
				StepID:          step.ID,
				State:           event.StateETAUpdated,
				EstimatedTimeMs: e.etaCalculator.RemainingMs(),
			})
		}

		// Track deliverables from completed step
		e.trackStepDeliverables(execution, step)

		// Extract declared outcomes from step artifacts
		e.processStepOutcomes(execution, step)

		// Record ontology usage for decision lineage tracking
		e.recordOntologyUsage(execution, step, "success")

		return nil
	}

	return lastErr
}

// checkOntologyStaleness returns a warning message if the ontology may be out of date.
// Two signals: (1) .wave/.ontology-stale sentinel exists (set by git post-merge hook),
// (2) wave.yaml is older than the most recent git commit.
func (e *DefaultPipelineExecutor) checkOntologyStaleness() string {
	// Check sentinel file first (cheapest — single stat call)
	if _, err := os.Stat(".wave/.ontology-stale"); err == nil {
		// Remove sentinel so warning fires once per merge, not every run
		_ = os.Remove(".wave/.ontology-stale")
		return "ontology may be stale (post-merge changes detected) — run 'wave analyze' to refresh"
	}

	// Compare wave.yaml mtime against last commit that touched ontology-relevant files.
	// Using ANY commit causes false positives when unrelated files are committed.
	manifestStat, err := os.Stat("wave.yaml")
	if err != nil {
		return ""
	}

	out, err := exec.Command("git", "log", "-1", "--format=%cI", "--", "wave.yaml", "internal/defaults/pipelines/", "internal/defaults/personas/").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return ""
	}
	lastCommit, err := time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
	if err != nil {
		return ""
	}

	if lastCommit.After(manifestStat.ModTime()) {
		return "ontology may be stale (wave.yaml older than latest commit) — run 'wave analyze' to refresh"
	}
	return ""
}

// recordOntologyUsage and buildOntologySection are defined in
// ontology_enabled.go (with "ontology" build tag) and
// ontology_disabled.go (without it).

// executeReworkStep handles on_failure=rework: marks the failed step, builds failure context,
// executes the rework target step, and re-registers its artifacts under the original step's ID.
func (e *DefaultPipelineExecutor) executeReworkStep(ctx context.Context, execution *PipelineExecution, failedStep *Step, failErr error, failDuration time.Duration) error {
	pipelineID := execution.Status.ID
	reworkStepID := failedStep.Retry.ReworkStep

	// Mark the failed step
	execution.mu.Lock()
	execution.States[failedStep.ID] = stateFailed
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, failedStep.ID, state.StateFailed, failErr.Error())
	}

	// Find the rework target step in the pipeline
	var reworkStep *Step
	for i := range execution.Pipeline.Steps {
		if execution.Pipeline.Steps[i].ID == reworkStepID {
			reworkStep = &execution.Pipeline.Steps[i]
			break
		}
	}
	if reworkStep == nil {
		return fmt.Errorf("rework step %q not found in pipeline (referenced by step %q)", reworkStepID, failedStep.ID)
	}

	// Build enhanced failure context for the rework step
	attemptCtx := &AttemptContext{
		Attempt:      failedStep.Retry.EffectiveMaxAttempts(),
		MaxAttempts:  failedStep.Retry.EffectiveMaxAttempts(),
		PriorError:   failErr.Error(),
		StepDuration: failDuration,
		FailedStepID: failedStep.ID,
	}

	// Capture stdout tail from results if available
	execution.mu.Lock()
	if result, ok := execution.Results[failedStep.ID]; ok {
		if stdout, ok := result["stdout"].(string); ok {
			if len(stdout) > maxStdoutTailChars {
				attemptCtx.PriorStdout = stdout[len(stdout)-maxStdoutTailChars:]
			} else {
				attemptCtx.PriorStdout = stdout
			}
		}
	}
	execution.mu.Unlock()

	// Scan workspace for partial artifacts (use relative paths to avoid exposing directory structure)
	execution.mu.Lock()
	wsPath := execution.WorkspacePaths[failedStep.ID]
	execution.mu.Unlock()
	if wsPath != "" && len(failedStep.OutputArtifacts) > 0 {
		partialArtifacts := make(map[string]string)
		for _, art := range failedStep.OutputArtifacts {
			artPath := filepath.Join(wsPath, art.Path)
			if _, err := os.Stat(artPath); err == nil {
				partialArtifacts[art.Name] = art.Path // relative path, not absolute
			}
		}
		if len(partialArtifacts) > 0 {
			attemptCtx.PartialArtifacts = partialArtifacts
		}
	}

	// Inject failure context into rework step
	execution.mu.Lock()
	execution.AttemptContexts[reworkStep.ID] = attemptCtx
	execution.mu.Unlock()

	// Emit reworking event
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     failedStep.ID,
		State:      event.StateReworking,
		Message:    fmt.Sprintf("rework: executing step %q after %q failed", reworkStepID, failedStep.ID),
	})
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, reworkStepID, state.StateReworking, "")
	}

	// Execute the rework step
	reworkStart := time.Now()
	reworkErr := e.runStepExecution(ctx, execution, reworkStep)
	reworkDuration := time.Since(reworkStart)
	if reworkErr != nil {
		execution.mu.Lock()
		execution.States[reworkStep.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, reworkStep.ID, state.StateFailed, reworkErr.Error())
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     reworkStepID,
			State:      event.StateFailed,
			Message:    fmt.Sprintf("rework step %q also failed: %s", reworkStepID, reworkErr.Error()),
		})
		return reworkErr
	}

	// Rework succeeded — replace failed step's artifacts with rework step's artifacts
	execution.mu.Lock()
	// Copy workspace path
	if rwPath, ok := execution.WorkspacePaths[reworkStep.ID]; ok {
		execution.WorkspacePaths[failedStep.ID] = rwPath
	}
	// Copy artifact paths: register rework step's artifacts under the original step's keys
	for _, art := range reworkStep.OutputArtifacts {
		reworkKey := fmt.Sprintf("%s:%s", reworkStep.ID, art.Name)
		if artPath, ok := execution.ArtifactPaths[reworkKey]; ok {
			originalKey := fmt.Sprintf("%s:%s", failedStep.ID, art.Name)
			execution.ArtifactPaths[originalKey] = artPath
		}
	}
	execution.States[reworkStep.ID] = stateCompleted
	// Mark the failed step as completed so downstream steps are not skipped
	execution.States[failedStep.ID] = stateCompleted
	// Record the rework transition for resume support
	execution.ReworkTransitions[failedStep.ID] = reworkStep.ID
	execution.mu.Unlock()

	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, reworkStep.ID, state.StateCompleted, "")
		_ = e.store.SaveStepState(pipelineID, failedStep.ID, state.StateCompleted, "reworked by "+reworkStepID)
	}

	// Record step attempt for audit trail
	if e.store != nil {
		completedAt := time.Now()
		_ = e.store.RecordStepAttempt(&state.StepAttemptRecord{
			RunID:       pipelineID,
			StepID:      reworkStep.ID,
			Attempt:     1,
			State:       "succeeded",
			DurationMs:  reworkDuration.Milliseconds(),
			StartedAt:   reworkStart,
			CompletedAt: &completedAt,
		})
	}

	// Record step completion for ETA calculation
	if e.etaCalculator != nil {
		e.etaCalculator.RecordStepCompletion(reworkStep.ID, reworkDuration.Milliseconds())
	}

	// Track deliverables from rework step
	e.trackStepDeliverables(execution, reworkStep)

	// Extract declared outcomes from rework step
	e.processStepOutcomes(execution, reworkStep)

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     reworkStepID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("rework step %q completed, artifacts replaced for %q", reworkStepID, failedStep.ID),
	})

	// Clear attempt context
	execution.mu.Lock()
	delete(execution.AttemptContexts, reworkStep.ID)
	execution.mu.Unlock()

	return nil
}

// validateStepContracts runs all contracts in EffectiveContracts() order.
// Each contract gets its own on_failure policy. When agent_review contracts fail
// with on_failure: rework, feedback is written as artifact and the rework step is
// executed; afterward all contracts re-run from the beginning (bounded by max_retries).
//
// Backward compatibility: a step with only the singular 'contract' field behaves
// identically to a single-element 'contracts' list — same events, tracing, and pass/fail.
func (e *DefaultPipelineExecutor) validateStepContracts(
	ctx context.Context,
	execution *PipelineExecution,
	step *Step,
	workspacePath string,
	stepRunner adapter.AdapterRunner,
	pipelineID string,
	resolvedPersona string,
	stepStart time.Time,
	result *adapter.AdapterResult,
) error {
	contracts := step.Handover.EffectiveContracts()
	if len(contracts) == 0 {
		if e.logger != nil {
			e.logger.LogContractResult(pipelineID, step.ID, "none", "skip")
		}
		return nil
	}

	// Build artifact paths map for agent_review context sources.
	// execution.ArtifactPaths keys are "stepID:artifactName"; build a name→path map
	// so artifact context sources can look up by name alone.
	artifactPaths := make(map[string]string)
	execution.mu.Lock()
	for k, v := range execution.ArtifactPaths {
		// k is "stepID:artifactName" — extract artifact name (part after last ":")
		if idx := strings.LastIndex(k, ":"); idx >= 0 {
			artifactName := k[idx+1:]
			// Keep the last-seen path for each artifact name
			artifactPaths[artifactName] = v
		} else {
			artifactPaths[k] = v
		}
	}
	execution.mu.Unlock()

	// maxRounds limits how many full contract-list re-runs can happen due to rework.
	// We use the max max_retries across all contracts that have on_failure: rework.
	maxRounds := 1
	var convergenceTracker *ConvergenceTracker
	for _, c := range contracts {
		if c.OnFailure == OnFailureRework && c.MaxRetries > maxRounds {
			maxRounds = c.MaxRetries
		}
		// Initialize convergence tracker from first rework contract with settings
		if c.OnFailure == OnFailureRework && convergenceTracker == nil {
			window := c.ConvergenceWindow
			if window == 0 {
				window = 3
			}
			minImprove := c.ConvergenceMinImprovement
			if minImprove == 0 {
				minImprove = 0.05
			}
			convergenceTracker = NewConvergenceTracker(window, minImprove)
		}
	}

	for round := 0; round <= maxRounds; round++ {
		reworkTriggered := false

		for _, c := range contracts {
			cErr := e.runSingleContract(ctx, execution, step, c, workspacePath, stepRunner, artifactPaths, pipelineID, resolvedPersona, stepStart, result)
			if cErr == nil {
				continue
			}

			reworkTriggered, policyErr := e.applyContractOnFailure(
				ctx, execution, step, c, cErr,
				round, maxRounds, convergenceTracker,
				pipelineID, resolvedPersona, stepStart, result, workspacePath,
			)
			if errors.Is(policyErr, errContractSkip) {
				return nil
			}
			if policyErr != nil {
				return policyErr
			}
			if reworkTriggered {
				break
			}
		}

		if !reworkTriggered {
			// All contracts passed (or continued) — we're done
			break
		}
		// Rework completed — re-run all contracts in next round
	}

	// Emit overall contract_passed if we get here without returning an error
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "contract_passed",
		Message:    fmt.Sprintf("all %d contract(s) validated", len(contracts)),
	})
	if e.logger != nil {
		// Log the primary contract type (first in list) for backward compat
		primaryType := contracts[0].Type
		e.logger.LogContractResult(pipelineID, step.ID, primaryType, "pass")
	}
	e.recordDecision(pipelineID, step.ID, "contract",
		fmt.Sprintf("contract validation passed for step %s", step.ID),
		fmt.Sprintf("all %d contract(s) validated successfully", len(contracts)),
		map[string]interface{}{"contract_count": len(contracts)},
	)
	// Run contract_validated hooks and webhooks (non-blocking by default)
	contractEvt := hooks.HookEvent{
		Type:       hooks.EventContractValidated,
		PipelineID: pipelineID,
		StepID:     step.ID,
		Workspace:  workspacePath,
	}
	if e.hookRunner != nil {
		e.hookRunner.RunHooks(ctx, contractEvt)
	}
	e.fireWebhooks(ctx, contractEvt)
	return nil
}

// errContractSkip is returned by applyContractOnFailure when the on_failure: skip
// policy is applied. validateStepContracts interprets this as "halt contract
// processing and return nil" — the step is treated as passing.
var errContractSkip = errors.New("contract: skip policy applied")

// applyContractOnFailure applies the configured on_failure policy for a failed contract.
// Returns (reworkTriggered, err):
//   - reworkTriggered=true: a rework step was triggered; caller should break the inner contract loop
//   - err == errContractSkip: skip policy applied; caller should return nil
//   - err != nil: hard failure; caller should return the error
//   - (false, nil): soft policy (continue/warn); caller resumes the next contract
func (e *DefaultPipelineExecutor) applyContractOnFailure(
	ctx context.Context,
	execution *PipelineExecution,
	step *Step,
	c ContractConfig,
	cErr error,
	round, maxRounds int,
	convergenceTracker *ConvergenceTracker,
	pipelineID, resolvedPersona string,
	stepStart time.Time,
	result *adapter.AdapterResult,
	workspacePath string,
) (reworkTriggered bool, err error) {
	// Determine on_failure policy (contract-level takes precedence, then legacy must_pass).
	// Default is fail — a contract that doesn't specify on_failure should not silently pass.
	onFailure := c.OnFailure
	if onFailure == "" {
		onFailure = OnFailureFail
	}

	switch onFailure {
	case OnFailureFail:
		if e.logger != nil {
			e.logger.LogContractResult(pipelineID, step.ID, c.Type, "fail")
			_ = e.logger.LogStepEnd(pipelineID, step.ID, stateFailed, time.Since(stepStart), result.ExitCode, 0, result.TokensUsed, cErr.Error())
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      resolvedPersona,
				StartedAt:    stepStart,
				CompletedAt:  &completedAt,
				DurationMs:   time.Since(stepStart).Milliseconds(),
				TokensUsed:   result.TokensUsed,
				Success:      false,
				ErrorMessage: "contract validation failed: " + cErr.Error(),
			})
		}
		e.recordDecision(pipelineID, step.ID, "contract",
			fmt.Sprintf("contract validation failed (hard) for step %s", step.ID),
			fmt.Sprintf("on_failure is 'fail', failing the step: %s", cErr.Error()),
			map[string]interface{}{"contract_type": c.Type, "error": cErr.Error()},
		)
		return false, fmt.Errorf("contract validation failed: %w", cErr)

	case OnFailureSkip:
		// Halt contract processing; the step is treated as passing (return nil upstream)
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "contract_skip",
			Message:    fmt.Sprintf("%s contract failed, skipping remaining contracts: %s", c.Type, cErr.Error()),
		})
		return false, errContractSkip

	case OnFailureContinue:
		// Log soft failure, continue to next contract
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "contract_soft_failure",
			Message:    fmt.Sprintf("contract validation failed but continuing (on_failure: continue): %s", cErr.Error()),
		})
		if e.logger != nil {
			e.logger.LogContractResult(pipelineID, step.ID, c.Type, "soft_fail")
		}
		e.recordDecision(pipelineID, step.ID, "contract",
			fmt.Sprintf("contract soft-failed for step %s", step.ID),
			"on_failure is 'continue', proceeding",
			map[string]interface{}{"contract_type": c.Type, "error": cErr.Error()},
		)
		return false, nil

	case OnFailureWarn:
		// Log warning, continue to next contract (same as continue but with explicit warning)
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "contract_warning",
			Message:    fmt.Sprintf("contract validation warning (on_failure: warn): %s", cErr.Error()),
		})
		if e.logger != nil {
			e.logger.LogContractResult(pipelineID, step.ID, c.Type, "warn")
		}
		e.recordDecision(pipelineID, step.ID, "contract",
			fmt.Sprintf("contract warning for step %s", step.ID),
			fmt.Sprintf("on_failure is 'warn', proceeding: %s", cErr.Error()),
			map[string]interface{}{"contract_type": c.Type, "error": cErr.Error()},
		)
		return false, nil

	case OnFailureRework:
		// Track convergence: extract score from error and check for stall
		if convergenceTracker != nil {
			if score, ok := ExtractScoreFromError(cErr.Error()); ok {
				convergenceTracker.RecordScore(score)
				if convergenceTracker.IsStalled() {
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "convergence_stalled",
						Message:    fmt.Sprintf("rework loop stalled at %s — aborting to save tokens", convergenceTracker.Summary()),
					})
					e.recordDecision(pipelineID, step.ID, "contract",
						fmt.Sprintf("convergence stalled for step %s", step.ID),
						fmt.Sprintf("score plateaued at %s, no improvement over %d rounds", convergenceTracker.Summary(), convergenceTracker.Rounds()),
						map[string]interface{}{"contract_type": c.Type, "scores": convergenceTracker.scores},
					)
					return false, fmt.Errorf("contract rework stalled (no convergence): %w", cErr)
				}
			}
		}

		if round >= maxRounds {
			// Retries exhausted — fall back to fail
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "contract_failed",
				Message:    fmt.Sprintf("%s contract: max rework retries (%d) exhausted: %s", c.Type, maxRounds, cErr.Error()),
			})
			return false, fmt.Errorf("contract validation failed after %d rework attempt(s): %w", maxRounds, cErr)
		}
		// Write feedback artifact and trigger rework
		feedbackPath, reworkErr := e.triggerContractRework(ctx, execution, step, c, cErr, workspacePath, pipelineID)
		if reworkErr != nil {
			return false, reworkErr
		}
		_ = feedbackPath
		return true, nil

	default:
		// Unknown on_failure — treat as fail
		return false, fmt.Errorf("contract validation failed: %w", cErr)
	}
}

// runSingleContract validates one contract and emits lifecycle events.
// For agent_review, it calls ValidateWithRunner; for all others, it calls contract.Validate.
func (e *DefaultPipelineExecutor) runSingleContract(
	_ context.Context,
	execution *PipelineExecution,
	step *Step,
	c ContractConfig,
	workspacePath string,
	stepRunner adapter.AdapterRunner,
	artifactPaths map[string]string,
	pipelineID string,
	_ string,
	_ time.Time,
	_ *adapter.AdapterResult,
) error {
	// Resolve source path
	resolvedSource := ""
	if c.Source != "" {
		// Explicit source: use as-is
		resolvedSource = execution.Context.ResolveContractSource(c)
	} else if len(step.OutputArtifacts) > 0 {
		// No explicit source: use output_artifacts[0].Path directly (root path)
		resolvedSource = step.OutputArtifacts[0].Path
	}

	// Resolve {{ project.* }} placeholders in command
	resolvedCommand := c.Command
	if execution.Context != nil {
		resolvedCommand = execution.Context.ResolvePlaceholders(c.Command)
	}

	// Display name for tracing
	contractDisplayName := c.Type
	if c.SchemaPath != "" {
		contractDisplayName = filepath.Base(c.SchemaPath)
	}

	// Build contract display name with schema info
	// Build contract display name with schema info
	contractDisplay := c.Type
	if c.SchemaPath != "" {
		contractDisplay = filepath.Base(c.SchemaPath)
	} else if c.Schema != "" {
		contractDisplay = "json_schema"
	}
	// Legacy: remove unused variable warning
	_ = contractDisplayName
	e.emit(event.Event{
		Timestamp:       time.Now(),
		PipelineID:      pipelineID,
		StepID:          step.ID,
		State:           "validating",
		Message:         fmt.Sprintf("Validating %s contract", contractDisplay),
		CurrentAction:   "Validating contract",
		ValidationPhase: contractDisplay,
	})

	e.trace("contract_validation_start", step.ID, 0, map[string]string{
		"type":   c.Type,
		"source": resolvedSource,
	})
	contractStart := time.Now()

	contractCfg := c
	contractCfg.Source = resolvedSource
	contractCfg.Command = resolvedCommand
	contractCfg.ArtifactPaths = artifactPaths

	var valErr error
	switch c.Type {
	case "agent_review":
		// Emit review_started event
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "review_started",
			Message:    fmt.Sprintf("agent review started (persona: %s)", c.Persona),
		})

		feedback, err := contract.ValidateWithRunner(contractCfg, workspacePath, stepRunner, execution.Manifest)
		switch {
		case err != nil:
			valErr = err
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "review_failed",
				Message:    fmt.Sprintf("agent review failed: %s", err.Error()),
			})
		case feedback != nil && feedback.Verdict == "fail":
			valErr = fmt.Errorf("agent review verdict: fail — %s", feedback.Summary)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "review_failed",
				Message:    fmt.Sprintf("agent review failed: verdict=%s issues=%d", feedback.Verdict, len(feedback.Issues)),
			})
		default:
			verdict := "pass"
			issueCount := 0
			if feedback != nil {
				verdict = feedback.Verdict
				issueCount = len(feedback.Issues)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "review_completed",
				Message:    fmt.Sprintf("agent review completed: verdict=%s issues=%d reviewer=%s", verdict, issueCount, c.Persona),
			})
		}
	case "event_contains":
		// Query event log for this run+step and validate patterns
		if e.store != nil {
			storeEvents, evErr := e.store.GetEvents(pipelineID, state.EventQueryOptions{Limit: 5000})
			if evErr != nil {
				valErr = fmt.Errorf("event_contains: failed to query events: %w", evErr)
			} else {
				records := make([]contract.EventRecord, len(storeEvents))
				for i, ev := range storeEvents {
					records[i] = contract.EventRecord{
						State:   ev.State,
						StepID:  ev.StepID,
						Message: ev.Message,
					}
				}
				valErr = contract.ValidateEventContains(contractCfg, step.ID, records)
				if valErr == nil {
					// Emit what was matched so the operator can see evidence
					for _, pattern := range contractCfg.Events {
						detail := pattern.State
						if pattern.Contains != "" {
							detail += " containing " + fmt.Sprintf("%q", pattern.Contains)
						}
						e.emit(event.Event{
							Timestamp:  time.Now(),
							PipelineID: pipelineID,
							StepID:     step.ID,
							State:      "contract_evidence",
							Message:    fmt.Sprintf("event_contains matched: %s", detail),
						})
					}
				}
			}
		} else {
			valErr = fmt.Errorf("event_contains: no state store available")
		}
	default:
		valErr = contract.Validate(contractCfg, workspacePath)
	}

	if valErr != nil {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "contract_failed",
			Message:    valErr.Error(),
		})
		e.trace("contract_validation_end", step.ID, time.Since(contractStart).Milliseconds(), map[string]string{
			"type":   c.Type,
			"result": "fail",
			"error":  valErr.Error(),
		})
		return valErr
	}

	e.trace("contract_validation_end", step.ID, time.Since(contractStart).Milliseconds(), map[string]string{
		"type":   c.Type,
		"result": "pass",
	})
	return nil
}

// triggerContractRework writes review feedback to .wave/artifacts/review_feedback.json,
// injects the feedback path into the rework step's context, and executes the rework step.
func (e *DefaultPipelineExecutor) triggerContractRework(
	ctx context.Context,
	execution *PipelineExecution,
	step *Step,
	c ContractConfig,
	contractErr error,
	workspacePath string,
	pipelineID string,
) (string, error) {
	reworkStepID := c.ReworkStep
	if reworkStepID == "" {
		reworkStepID = step.Retry.ReworkStep
	}
	if reworkStepID == "" {
		return "", fmt.Errorf("agent_review contract has on_failure: rework but no rework_step configured")
	}

	// Write review feedback as artifact
	feedbackPath := filepath.Join(workspacePath, ".wave", "artifacts", fmt.Sprintf("review_feedback_%s.json", step.ID))
	if err := os.MkdirAll(filepath.Dir(feedbackPath), 0o750); err != nil {
		return "", fmt.Errorf("failed to create artifacts dir for review feedback: %w", err)
	}
	feedbackPayload := map[string]interface{}{
		"contract_type": c.Type,
		"error":         contractErr.Error(),
	}
	feedbackBytes, _ := json.Marshal(feedbackPayload)
	if err := os.WriteFile(feedbackPath, feedbackBytes, 0o640); err != nil {
		return "", fmt.Errorf("failed to write review feedback artifact: %w", err)
	}

	// Find the rework step
	var reworkStep *Step
	for i := range execution.Pipeline.Steps {
		if execution.Pipeline.Steps[i].ID == reworkStepID {
			reworkStep = &execution.Pipeline.Steps[i]
			break
		}
	}
	if reworkStep == nil {
		return "", fmt.Errorf("rework step %q not found (referenced by contract in step %q)", reworkStepID, step.ID)
	}

	// Build attempt context with review feedback path
	attemptCtx := &AttemptContext{
		Attempt:            1,
		MaxAttempts:        c.MaxRetries + 1,
		PriorError:         contractErr.Error(),
		FailedStepID:       step.ID,
		ReviewFeedbackPath: feedbackPath,
	}
	execution.mu.Lock()
	execution.AttemptContexts[reworkStep.ID] = attemptCtx
	execution.mu.Unlock()

	// Emit reworking event
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateReworking,
		Message:    fmt.Sprintf("contract rework: executing step %q after review failed for step %q", reworkStepID, step.ID),
	})

	// Execute the rework step
	if reworkErr := e.runStepExecution(ctx, execution, reworkStep); reworkErr != nil {
		execution.mu.Lock()
		execution.States[reworkStep.ID] = stateFailed
		execution.mu.Unlock()
		return "", fmt.Errorf("rework step %q failed: %w", reworkStepID, reworkErr)
	}

	execution.mu.Lock()
	execution.States[reworkStep.ID] = stateCompleted
	delete(execution.AttemptContexts, reworkStep.ID)
	execution.mu.Unlock()

	return feedbackPath, nil
}

// executeMatrixStep handles steps with matrix strategy using fan-out execution.
func (e *DefaultPipelineExecutor) executeMatrixStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	matrixExecutor := NewMatrixExecutor(e)
	err := matrixExecutor.Execute(ctx, execution, step)

	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	// Track deliverables from completed matrix step
	e.trackStepDeliverables(execution, step)

	// Extract declared outcomes from matrix step artifacts
	e.processStepOutcomes(execution, step)

	return nil
}

// executeConcurrentStep handles steps with concurrency > 1 using parallel agent execution.
func (e *DefaultPipelineExecutor) executeConcurrentStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	concurrencyExecutor := NewConcurrencyExecutor(e)
	err := concurrencyExecutor.Execute(ctx, execution, step)

	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	// Track deliverables from completed concurrent step
	e.trackStepDeliverables(execution, step)

	// Extract declared outcomes from concurrent step artifacts
	e.processStepOutcomes(execution, step)

	return nil
}

// runStepExecution orchestrates the four phases of a single step run:
// resource resolution → config assembly → adapter dispatch → result processing.
func (e *DefaultPipelineExecutor) runStepExecution(ctx context.Context, execution *PipelineExecution, step *Step) error {
	// Phase A: Resolve persona, adapter, workspace, model, artifacts, and build base prompt
	res, err := e.resolveStepResources(ctx, execution, step)
	if err != nil {
		return err
	}

	// Phase B: Build AdapterRunConfig (timeout, system prompt, sandbox, skills, contract prompt)
	cfg, err := e.buildStepAdapterConfig(ctx, execution, step, res)
	if err != nil {
		return err
	}

	// Emit step progress: executing
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    res.pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       res.resolvedPersona,
		Progress:      25,
		CurrentAction: "Executing agent",
	})

	// Iron Rule: estimate prompt size and check against context window
	promptBytes := len(cfg.Prompt) + len(cfg.OntologySection)
	if promptBytes > 0 && res.resolvedModel != "" {
		ironStatus, ironMsg := cost.CheckIronRule(res.resolvedModel, promptBytes)
		switch ironStatus {
		case cost.IronRuleWarning:
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: res.pipelineID,
				StepID:     step.ID,
				State:      "iron_rule_warning",
				Message:    ironMsg,
			})
		case cost.IronRuleFail:
			return fmt.Errorf("iron rule: %s", ironMsg)
		}
	}

	// Phase C: Dispatch to adapter
	stepStart := time.Now()
	e.trace("adapter_start", step.ID, 0, map[string]string{
		"persona": res.resolvedPersona,
		"adapter": res.resolvedAdapterName,
		"model":   res.resolvedModel,
	})
	result, adapterErr := res.stepRunner.Run(ctx, cfg)
	adapterDurationMs := time.Since(stepStart).Milliseconds()

	if adapterErr != nil {
		e.trace("adapter_end", step.ID, adapterDurationMs, map[string]string{
			"status": stateFailed,
			"error":  adapterErr.Error(),
		})
		if e.logger != nil {
			_ = e.logger.LogStepEnd(res.pipelineID, step.ID, stateFailed, time.Since(stepStart), 0, 0, 0, adapterErr.Error())
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        res.pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      res.resolvedPersona,
				StartedAt:    stepStart,
				CompletedAt:  &completedAt,
				DurationMs:   time.Since(stepStart).Milliseconds(),
				Success:      false,
				ErrorMessage: adapterErr.Error(),
			})
		}
		return fmt.Errorf("adapter execution failed: %w", adapterErr)
	}

	// Warn on non-zero exit code — adapter process may have crashed, but
	// work may still have been completed (e.g. Claude Code JS error after
	// tool calls finished). Let contract validation decide the outcome.
	if result.ExitCode != 0 {
		msg := fmt.Sprintf("adapter exited with code %d", result.ExitCode)
		if result.FailureReason != "" {
			msg += ": " + result.FailureReason
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: res.pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    msg,
		})
	}

	// Fail immediately on rate limit — the result content is an error message,
	// not useful work product. Proceeding would write the error as an artifact.
	if result.FailureReason == adapter.FailureReasonRateLimit {
		if e.logger != nil {
			_ = e.logger.LogStepEnd(res.pipelineID, step.ID, stateFailed, time.Since(stepStart), result.ExitCode, 0, result.TokensUsed, "rate limited: "+result.ResultContent)
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        res.pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      res.resolvedPersona,
				StartedAt:    stepStart,
				CompletedAt:  &completedAt,
				DurationMs:   time.Since(stepStart).Milliseconds(),
				TokensUsed:   result.TokensUsed,
				Success:      false,
				ErrorMessage: "rate limited: " + result.ResultContent,
			})
		}
		return fmt.Errorf("adapter rate limited: %s", result.ResultContent)
	}

	e.trace("adapter_end", step.ID, adapterDurationMs, map[string]string{
		"status":      "success",
		"exit_code":   fmt.Sprintf("%d", result.ExitCode),
		"tokens_used": fmt.Sprintf("%d", result.TokensUsed),
	})

	// Phase D: Process adapter result (stdout, tokens, cost, artifacts, contracts, hooks)
	return e.processAdapterResult(ctx, execution, step, res, result, stepStart)
}

// resolveStepResources resolves the persona, adapter, workspace, and model for a step,
// injects dependent artifacts, and builds the base step prompt.
// It is Phase A of runStepExecution.
func (e *DefaultPipelineExecutor) resolveStepResources(ctx context.Context, execution *PipelineExecution, step *Step) (*stepRunResources, error) {
	pipelineID := execution.Status.ID

	resolvedPersona := step.Persona
	if execution.Context != nil {
		resolvedPersona = execution.Context.ResolvePlaceholders(step.Persona)
	}
	persona := execution.Manifest.GetPersona(resolvedPersona)
	if persona == nil {
		return nil, fmt.Errorf("persona %q not found in manifest", resolvedPersona)
	}

	// Resolve adapter name (strongest to weakest):
	// 1. CLI --adapter flag  2. Step-level adapter  3. Persona-level adapter  4. Adapter defaults
	resolvedAdapterName := persona.Adapter
	if step.Adapter != "" {
		resolvedAdapterName = step.Adapter
	}
	if e.adapterOverride != "" {
		resolvedAdapterName = e.adapterOverride
	}

	adapterDef := execution.Manifest.GetAdapter(resolvedAdapterName)
	if adapterDef == nil {
		available := make([]string, 0, len(execution.Manifest.Adapters))
		for name := range execution.Manifest.Adapters {
			available = append(available, name)
		}
		return nil, fmt.Errorf("adapter %q not found in manifest for step %q (available: %v)", resolvedAdapterName, step.ID, available)
	}

	// Resolve adapter runner from registry for per-step dispatch
	stepRunner := e.registry.ResolveWithFallback(resolvedAdapterName)

	// Create workspace under .wave/workspaces/<pipeline>/<step>/
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	execution.mu.Lock()
	execution.WorkspacePaths[step.ID] = workspacePath
	execution.mu.Unlock()

	// Run workspace_created hooks (non-blocking by default)
	if e.hookRunner != nil {
		e.hookRunner.RunHooks(ctx, hooks.HookEvent{
			Type:       hooks.EventWorkspaceCreated,
			PipelineID: pipelineID,
			StepID:     step.ID,
			Workspace:  workspacePath,
		})
	}

	// Pre-create .wave/output/ so personas without Bash can write artifacts
	if len(step.OutputArtifacts) > 0 {
		outputDir := filepath.Join(workspacePath, ".wave", "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output dir: %w", err)
		}
	}

	var adapterTierModels map[string]string
	if adapterDef != nil {
		adapterTierModels = adapterDef.TierModels
	}
	resolvedModel := e.resolveModel(step, persona, &execution.Manifest.Runtime.Routing, resolvedPersona, adapterTierModels)

	// When no model was resolved, fall back to adapter's default_model
	if resolvedModel == "" && adapterDef != nil && adapterDef.DefaultModel != "" {
		resolvedModel = adapterDef.DefaultModel
	}

	// When resolved adapter differs from persona's adapter and no explicit model was set,
	// fall back to the target adapter's default model (avoids cross-ecosystem model IDs)
	if resolvedAdapterName != persona.Adapter && e.modelOverride == "" && step.Model == "" {
		if adapterDef != nil && adapterDef.DefaultModel != "" {
			resolvedModel = adapterDef.DefaultModel
		} else {
			resolvedModel = ""
		}
	}

	// Determine the configured (pre-resolution) model for provenance tracking
	configuredModel := step.Model
	if configuredModel == "" {
		configuredModel = persona.Model
	}

	e.emit(event.Event{
		Timestamp:       time.Now(),
		PipelineID:      pipelineID,
		StepID:          step.ID,
		State:           stateRunning,
		Persona:         resolvedPersona,
		Message:         fmt.Sprintf("Starting %s persona in %s", resolvedPersona, workspacePath),
		CurrentAction:   "Initializing",
		Model:           resolvedModel,
		ConfiguredModel: configuredModel,
		Adapter:         resolvedAdapterName,
		Temperature:     persona.Temperature,
	})

	// Record model routing decision
	{
		var rationale string
		switch {
		case e.modelOverride != "":
			rationale = "CLI --model flag override"
		case step.Model != "":
			rationale = "per-step model pinning in pipeline YAML"
		case persona.Model != "":
			rationale = "per-persona model configuration"
		case execution.Manifest.Runtime.Routing.AutoRoute:
			rationale = "auto-routed based on step complexity"
		default:
			rationale = "adapter default (no override)"
		}
		modelDisplay := resolvedModel
		if modelDisplay == "" {
			modelDisplay = "(adapter default)"
		}
		e.recordDecision(pipelineID, step.ID, "model_routing",
			fmt.Sprintf("selected model %s for step %s", modelDisplay, step.ID),
			rationale,
			map[string]interface{}{
				"model":   resolvedModel,
				"persona": resolvedPersona,
				"adapter": resolvedAdapterName,
			},
		)
	}

	// Inject artifacts from dependencies
	artifactInjectStart := time.Now()
	if err := e.injectArtifacts(execution, step, workspacePath); err != nil {
		return nil, fmt.Errorf("failed to inject artifacts: %w", err)
	}
	e.trace("artifact_injection", step.ID, time.Since(artifactInjectStart).Milliseconds(), map[string]string{
		"workspace": workspacePath,
		"count":     fmt.Sprintf("%d", len(step.Memory.InjectArtifacts)),
	})

	// Audit: log step start with injected artifact names
	if e.logger != nil {
		var artifactNames []string
		for _, ref := range step.Memory.InjectArtifacts {
			name := ref.As
			if name == "" {
				name = ref.Artifact
			}
			artifactNames = append(artifactNames, name)
		}
		_ = e.logger.LogStepStartWithAdapter(pipelineID, step.ID, resolvedPersona, resolvedAdapterName, resolvedModel, artifactNames)
	}

	prompt := e.buildStepPrompt(execution, step)
	if e.logger != nil {
		_ = e.logger.LogToolCall(pipelineID, step.ID, "adapter.Run", fmt.Sprintf("persona=%s prompt_len=%d", resolvedPersona, len(prompt)))
	}

	return &stepRunResources{
		pipelineID:          pipelineID,
		resolvedPersona:     resolvedPersona,
		persona:             persona,
		adapterDef:          adapterDef,
		resolvedAdapterName: resolvedAdapterName,
		stepRunner:          stepRunner,
		workspacePath:       workspacePath,
		resolvedModel:       resolvedModel,
		configuredModel:     configuredModel,
		prompt:              prompt,
	}, nil
}

// buildStepAdapterConfig assembles the adapter.AdapterRunConfig for a step.
// It resolves timeout, system prompt, sandbox settings, skills, contract prompt,
// and ontology section. It is Phase B of runStepExecution.
func (e *DefaultPipelineExecutor) buildStepAdapterConfig(_ context.Context, execution *PipelineExecution, step *Step, res *stepRunResources) (adapter.AdapterRunConfig, error) {
	pipelineID := res.pipelineID
	prompt := res.prompt

	// Resolve timeout with four-tier precedence:
	// 1. Step-level timeout_minutes (pipeline YAML) — most specific
	// 2. CLI --timeout flag (stepTimeoutOverride)
	// 3. runtime.default_timeout_minutes (manifest)
	// 4. Hardcoded fallback (5 minutes)
	timeout := execution.Manifest.Runtime.GetDefaultTimeout()
	if e.stepTimeoutOverride > 0 {
		timeout = e.stepTimeoutOverride
	}
	if stepTimeout := step.GetTimeout(); stepTimeout > 0 {
		timeout = stepTimeout
	}

	// Load system prompt from persona file
	systemPrompt := ""
	if res.persona.SystemPromptFile != "" {
		if data, err := os.ReadFile(res.persona.SystemPromptFile); err == nil {
			systemPrompt = string(data)
		}
	}

	// Resolve sandbox config using the new backend-aware resolution
	sandboxBackend := execution.Manifest.Runtime.Sandbox.ResolveBackend()
	sandboxEnabled := sandboxBackend != "none"
	var sandboxDomains []string
	var envPassthrough []string
	if sandboxEnabled {
		if res.persona.Sandbox != nil && len(res.persona.Sandbox.AllowedDomains) > 0 {
			sandboxDomains = res.persona.Sandbox.AllowedDomains
		} else if len(execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains) > 0 {
			sandboxDomains = execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains
		}
		envPassthrough = execution.Manifest.Runtime.Sandbox.EnvPassthrough
	}

	// Resolve skills from all three scopes: global, persona, pipeline
	// Pipeline scope includes both pipeline.Skills and requires.skills keys.
	var pipelineSkills []string
	for _, s := range execution.Pipeline.Skills {
		r := execution.Context.ResolvePlaceholders(s)
		if r != "" {
			pipelineSkills = append(pipelineSkills, r)
		}
	}
	if execution.Pipeline.Requires != nil {
		pipelineSkills = append(pipelineSkills, execution.Pipeline.Requires.SkillNames()...)
	}
	resolvedSkills := skill.ResolveSkills(execution.Manifest.Skills, res.persona.Skills, pipelineSkills)

	// Provision skill commands from requires.skills (SkillConfig-backed skills)
	var skillCommandsDir string
	if execution.Pipeline.Requires != nil && len(execution.Pipeline.Requires.Skills) > 0 {
		skillNames := execution.Pipeline.Requires.SkillNames()
		provisioner := skill.NewProvisioner(execution.Pipeline.Requires.Skills, "")
		commands, _ := provisioner.DiscoverCommands(skillNames)
		if len(commands) > 0 {
			tmpDir := filepath.Join(res.workspacePath, ".wave-skill-commands")
			if err := provisioner.Provision(tmpDir, skillNames); err != nil {
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

	// Resolve DirectoryStore skills (name-only references not in requires.skills).
	// Read metadata from the store but do NOT write files here — each adapter's
	// prepareWorkspace handles file provisioning to the correct location
	// (e.g. .claude/skills/ for claude, text injection for others).
	var resolvedSkillRefs []adapter.SkillRef
	if e.skillStore != nil && len(resolvedSkills) > 0 {
		requiresSkills := make(map[string]bool)
		if execution.Pipeline.Requires != nil {
			for name := range execution.Pipeline.Requires.Skills {
				requiresSkills[name] = true
			}
		}
		for _, name := range resolvedSkills {
			if requiresSkills[name] {
				continue
			}
			s, readErr := e.skillStore.Read(name)
			if readErr != nil {
				if errors.Is(readErr, skill.ErrNotFound) {
					// skill not found in store, skip silently
					continue
				}
				return adapter.AdapterRunConfig{}, fmt.Errorf("skill %q: %w", name, readErr)
			}
			resolvedSkillRefs = append(resolvedSkillRefs, adapter.SkillRef{
				Name:        s.Name,
				Description: s.Description,
				SourcePath:  s.SourcePath,
			})
		}
	}

	// Auto-generate contract compliance section. Appended directly to the user prompt
	// so the model sees it alongside the task instructions (system prompt injection was unreliable).
	contractPrompt := e.buildContractPrompt(step, execution.Context)
	if contractPrompt != "" {
		prompt = prompt + "\n\n" + contractPrompt
	}

	// Build ontology section from manifest for AGENTS.md injection (requires ontology build tag)
	ontologySection := e.buildOntologySection(execution, step, pipelineID)

	cfg := adapter.AdapterRunConfig{
		Adapter:             res.resolvedAdapterName,
		Persona:             res.resolvedPersona,
		WorkspacePath:       res.workspacePath,
		Prompt:              prompt,
		SystemPrompt:        systemPrompt,
		Timeout:             timeout,
		Temperature:         res.persona.Temperature,
		Model:               res.resolvedModel,
		AllowedTools:        res.persona.Permissions.AllowedTools,
		DenyTools:           res.persona.Permissions.Deny,
		OutputFormat:        res.adapterDef.OutputFormat,
		Debug:               e.debug,
		SandboxEnabled:      sandboxEnabled,
		AllowedDomains:      sandboxDomains,
		EnvPassthrough:      envPassthrough,
		SandboxBackend:      sandboxBackend,
		DockerImage:         execution.Manifest.Runtime.Sandbox.GetDockerImage(),
		SkillCommandsDir:    skillCommandsDir,
		ResolvedSkills:      resolvedSkillRefs,
		OntologySection:     ontologySection,
		MaxConcurrentAgents: step.MaxConcurrentAgents,
		OnStreamEvent: func(evt adapter.StreamEvent) {
			if evt.Type == "tool_use" && evt.ToolName != "" {
				execution.mu.Lock()
				wd := execution.Watchdog
				execution.mu.Unlock()
				if wd != nil {
					wd.NotifyActivity()
					if IsProgressTool(evt.ToolName) {
						wd.NotifyProgress()
					}
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStreamActivity,
					Persona:    res.resolvedPersona,
					ToolName:   evt.ToolName,
					ToolTarget: evt.ToolInput,
				})
			}
		},
	}

	return cfg, nil
}

// processAdapterResult handles the result from a successful adapter run:
// reads stdout, accumulates tokens and cost, writes artifacts, runs relay compaction,
// validates contracts, fires completion hooks, and records performance metrics.
// It is Phase D of runStepExecution.
func (e *DefaultPipelineExecutor) processAdapterResult(
	ctx context.Context,
	execution *PipelineExecution,
	step *Step,
	res *stepRunResources,
	result *adapter.AdapterResult,
	stepStart time.Time,
) error {
	pipelineID := res.pipelineID
	stepDuration := time.Since(stepStart).Milliseconds()

	// Emit step progress: processing results
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       res.resolvedPersona,
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
	output["workspace"] = res.workspacePath

	execution.mu.Lock()
	execution.Results[step.ID] = output
	execution.mu.Unlock()

	// Append step output to thread transcript when the step is part of a thread group
	if step.Thread != "" && execution.ThreadManager != nil {
		content := result.ResultContent
		if content == "" {
			content = string(stdoutData)
		}
		if content != "" {
			execution.ThreadManager.AppendTranscript(step.Thread, step.ID, content)
			e.trace(audit.TraceThreadAppend, step.ID, int64(len(content)), map[string]string{
				"thread": step.Thread,
				"size":   fmt.Sprintf("%d", len(content)),
			})
		}
	}

	// Accumulate tokens at executor level (survives pipeline cleanup)
	if result.TokensUsed > 0 {
		e.mu.Lock()
		e.totalTokens += result.TokensUsed
		e.mu.Unlock()
	}

	// Record cost and enforce budget
	if e.costLedger != nil && (result.TokensIn > 0 || result.TokensOut > 0) {
		_, budgetStatus := e.costLedger.Record(pipelineID, step.ID, res.resolvedModel, result.TokensIn, result.TokensOut, result.TokensUsed)
		switch budgetStatus {
		case cost.BudgetWarning:
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "budget_warning",
				Message:    fmt.Sprintf("Cost warning: %s", e.costLedger.Summary()),
			})
		case cost.BudgetExceeded:
			return fmt.Errorf("budget exceeded: %s", e.costLedger.Summary())
		}
	}

	// Check for stdout artifacts and validate size limits
	hasStdoutArtifacts := false
	for _, art := range step.OutputArtifacts {
		if art.IsStdoutArtifact() {
			hasStdoutArtifacts = true
			break
		}
	}

	if hasStdoutArtifacts {
		maxSize := execution.Manifest.Runtime.Artifacts.GetMaxStdoutSize()
		if int64(len(stdoutData)) > maxSize {
			return fmt.Errorf("stdout artifact size (%d bytes) exceeds limit (%d bytes); consider reducing output or increasing runtime.artifacts.max_stdout_size",
				len(stdoutData), maxSize)
		}
		e.writeOutputArtifacts(execution, step, res.workspacePath, stdoutData)
	}

	// Write file-based output artifacts
	// Use ResultContent if available; don't fall back to raw stdout (contains JSON wrapper)
	if result.ResultContent != "" && !hasStdoutArtifacts {
		e.writeOutputArtifacts(execution, step, res.workspacePath, []byte(result.ResultContent))
	} else if !hasStdoutArtifacts {
		// ResultContent is empty — check whether the persona wrote artifact files to disk
		// (e.g. via Write/Bash). Without this, persona-written files are never registered
		// in ArtifactPaths and contract validation fails on missing files.
		e.writeOutputArtifacts(execution, step, res.workspacePath, nil)
	}

	// Check relay/compaction threshold (FR-009)
	if cErr := e.checkRelayCompaction(ctx, execution, step, result.TokensUsed, res.workspacePath, string(stdoutData)); cErr != nil {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    fmt.Sprintf("relay compaction failed: %v", cErr),
		})
	}

	// Validate handover contracts — iterate over EffectiveContracts() which returns
	// the plural 'contracts' list if set, otherwise wraps the singular 'contract' field.
	if err := e.validateStepContracts(ctx, execution, step, res.workspacePath, res.stepRunner, pipelineID, res.resolvedPersona, stepStart, result); err != nil {
		return err
	}

	// Populate artifact paths from step's OutputArtifacts when the adapter
	// doesn't report them (e.g. Claude adapter never populates Artifacts).
	stepArtifacts := result.Artifacts
	if len(stepArtifacts) == 0 && len(step.OutputArtifacts) > 0 && execution.Context != nil {
		for _, art := range step.OutputArtifacts {
			stepArtifacts = append(stepArtifacts, execution.Context.ResolveArtifactPath(art))
		}
	}

	// Run artifact_created hooks (non-blocking by default)
	if e.hookRunner != nil && len(stepArtifacts) > 0 {
		e.hookRunner.RunHooks(ctx, hooks.HookEvent{
			Type:       hooks.EventArtifactCreated,
			PipelineID: pipelineID,
			StepID:     step.ID,
			Workspace:  res.workspacePath,
			Artifacts:  stepArtifacts,
		})
	}

	// Run step_completed hooks (blocking by default)
	stepCompletedEvt := hooks.HookEvent{
		Type:       hooks.EventStepCompleted,
		PipelineID: pipelineID,
		StepID:     step.ID,
		Input:      execution.Input,
		Workspace:  res.workspacePath,
	}
	if e.hookRunner != nil {
		if _, err := e.hookRunner.RunHooks(ctx, stepCompletedEvt); err != nil {
			return fmt.Errorf("step_completed hook failed: %w", err)
		}
	}
	e.fireWebhooks(ctx, stepCompletedEvt)

	// Detect zero-diff worktree steps: step completed but produced no code changes.
	// This gives the UI an honest signal — "completed_empty" with warning colors
	// instead of a misleading green "completed" checkmark.
	finalState := stateCompleted
	if step.Workspace.Type == "worktree" && isWorktreeClean(res.workspacePath) {
		finalState = stateCompletedEmpty
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      finalState,
		Persona:    res.resolvedPersona,
		DurationMs: stepDuration,
		TokensUsed: result.TokensUsed,
		Artifacts:  stepArtifacts,
		TokensIn:   result.TokensIn,
		TokensOut:  result.TokensOut,
	})

	if e.logger != nil {
		_ = e.logger.LogStepEnd(pipelineID, step.ID, finalState, time.Since(stepStart), result.ExitCode, len(stdoutData), result.TokensUsed, "")
	}

	// Record performance metric for TUI step breakdown
	if e.store != nil {
		completedAt := time.Now()
		e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
			RunID:              pipelineID,
			StepID:             step.ID,
			PipelineName:       execution.Status.PipelineName,
			Persona:            res.resolvedPersona,
			StartedAt:          stepStart,
			CompletedAt:        &completedAt,
			DurationMs:         stepDuration,
			TokensUsed:         result.TokensUsed,
			ArtifactsGenerated: len(stepArtifacts),
			Success:            true,
		})
	}

	return nil
}

// isWorktreeClean checks whether a worktree workspace has uncommitted or
// unstaged changes. Returns true when the worktree is identical to its HEAD
// (zero diff) — meaning the agent produced no code changes.
func isWorktreeClean(workspacePath string) bool {
	// Find the worktree directory: it's typically a __wt_* subdirectory
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return false
	}
	wtDir := ""
	for _, e := range entries {
		if e.IsDir() && len(e.Name()) > 5 && e.Name()[:5] == "__wt_" {
			wtDir = filepath.Join(workspacePath, e.Name())
			break
		}
	}
	if wtDir == "" {
		return false // not a worktree workspace or can't find it
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = wtDir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(out)) == 0
}

// resolveModel applies model precedence:
//
// When --model is a tier name (cheapest/balanced/strongest):
//
//	The effective tier is the CHEAPER of the CLI tier and the step/persona tier.
//	This means --model balanced + step model: cheapest → cheapest (step wins).
//	The CLI flag sets a ceiling, not a floor.
//
// When --model is a literal model name (e.g., "claude-sonnet-4"):
//
//	The literal model is used for all steps regardless of YAML tiers.
//
// When --force-model is set:
//
//	The CLI model overrides everything unconditionally.
//
// Otherwise: step model > persona model > auto-route > adapter tier_models > global routing > adapter default.
func (e *DefaultPipelineExecutor) resolveModel(step *Step, persona *manifest.Persona, routing *manifest.RoutingConfig, personaName string, adapterTierModels map[string]string) string {
	// Force override — bypasses all tier logic
	if e.forceModel {
		if e.modelOverride != "" {
			return e.modelOverride
		}
	}

	// Determine step-level tier (if any)
	stepTier := ""
	if step != nil && step.Model != "" {
		stepTier = step.Model
	} else if persona.Model != "" {
		stepTier = persona.Model
	}

	if e.modelOverride != "" {
		overrideRank := TierRank(e.modelOverride)
		if overrideRank >= 0 && stepTier != "" {
			// Both are tiers — use the cheaper one
			effectiveTier := CheaperTier(e.modelOverride, stepTier)
			if resolved, isTier := resolveTierModel(effectiveTier, routing, adapterTierModels); isTier {
				return resolved
			}
			return effectiveTier
		}
		// CLI is a literal model name — use it directly
		return e.modelOverride
	}

	// No CLI override — use step, persona, auto-route
	if step != nil && step.Model != "" {
		if resolved, isTier := resolveTierModel(step.Model, routing, adapterTierModels); isTier {
			return resolved
		}
		return step.Model
	}
	if persona.Model != "" {
		if resolved, isTier := resolveTierModel(persona.Model, routing, adapterTierModels); isTier {
			return resolved
		}
		return persona.Model
	}
	if routing != nil && routing.AutoRoute {
		tier := ClassifyStepComplexity(step, persona, personaName)
		if e.taskComplexity != "" {
			tier = AdjustTierForTaskComplexity(tier, e.taskComplexity)
		}
		if resolved, isTier := resolveTierModel(tier, routing, adapterTierModels); isTier {
			return resolved
		}
	}
	return ""
}

// resolveTierModel checks if a model string is a tier name (cheapest/balanced/strongest)
// and resolves it to an actual model via:
//  1. Adapter-specific tier_models (highest priority)
//  2. Global routing complexity_map
//
// Returns (resolved model, true) if input is a tier name, or ("", false) if it's a literal model.
func resolveTierModel(model string, routing *manifest.RoutingConfig, adapterTierModels map[string]string) (string, bool) {
	switch model {
	case TierCheapest, TierBalanced, TierStrongest:
		// Priority 1: adapter-specific tier_models
		if adapterTierModels != nil {
			if m, ok := adapterTierModels[model]; ok && m != "" {
				return m, true
			}
		}
		// Priority 2: global routing complexity_map
		return routing.ResolveComplexityModel(model), true
	default:
		return "", false
	}
}

// recordDecision records a structured decision to the state store.
// It is a no-op if the store is nil.
func (e *DefaultPipelineExecutor) recordDecision(runID, stepID, category, decision, rationale string, ctx map[string]interface{}) {
	if e.store == nil {
		return
	}
	contextJSON := "{}"
	if ctx != nil {
		if data, err := json.Marshal(ctx); err == nil {
			contextJSON = string(data)
		}
	}
	_ = e.store.RecordDecision(&state.DecisionRecord{
		RunID:     runID,
		StepID:    stepID,
		Timestamp: time.Now(),
		Category:  category,
		Decision:  decision,
		Rationale: rationale,
		Context:   contextJSON,
	})
}

func (e *DefaultPipelineExecutor) createStepWorkspace(execution *PipelineExecution, step *Step) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Handle workspace ref — share another step's workspace
	if step.Workspace.Ref != "" {
		// Special "parent" ref: use the parent sub-pipeline step's workspace
		if step.Workspace.Ref == "parent" && e.parentWorkspacePath != "" {
			return e.parentWorkspacePath, nil
		}
		execution.mu.Lock()
		refPath, ok := execution.WorkspacePaths[step.Workspace.Ref]
		execution.mu.Unlock()
		if !ok {
			return "", fmt.Errorf("referenced workspace step %q has not been executed yet", step.Workspace.Ref)
		}
		return refPath, nil
	}

	// Handle worktree workspace type
	if step.Workspace.Type == "worktree" {
		// Resolve branch name from template variables.
		// Step output references ({{ steps.X.artifacts.Y.field }}) are resolved first
		// so that branch names can be derived from prior step outputs (e.g. PR head branch).
		branch := step.Workspace.Branch
		if branch != "" {
			resolved, err := e.resolveWorkspaceStepRefs(branch, execution)
			if err != nil {
				return "", fmt.Errorf("workspace branch template %q: %w", branch, err)
			}
			branch = resolved
		}
		if execution.Context != nil && branch != "" {
			branch = execution.Context.ResolvePlaceholders(branch)
		}

		// Resolve base ref from template variables (same two-pass resolution).
		base := step.Workspace.Base
		if base != "" {
			resolved, err := e.resolveWorkspaceStepRefs(base, execution)
			if err != nil {
				return "", fmt.Errorf("workspace base template %q: %w", base, err)
			}
			base = resolved
		}
		if execution.Context != nil && base != "" {
			base = execution.Context.ResolvePlaceholders(base)
		}
		// Stacked matrix execution: override base branch from parent tier
		if e.stackedBaseBranch != "" && base == "" {
			base = e.stackedBaseBranch
		}

		if branch == "" && base == "" {
			// Fall back to pipeline context branch or generate one
			branch = execution.Context.BranchName
			if branch == "" {
				branch = fmt.Sprintf("wave/%s/%s", pipelineID, step.ID)
			}
		}

		// Reuse existing worktree for the same branch
		execution.mu.Lock()
		info, ok := execution.WorktreePaths[branch]
		if ok {
			execution.WorkspacePaths[step.ID+"__worktree_repo_root"] = info.RepoRoot
		}
		execution.mu.Unlock()
		if ok {
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
		execution.mu.Lock()
		execution.WorktreePaths[branch] = &WorktreeInfo{AbsPath: absPath, RepoRoot: mgr.RepoRoot()}
		execution.WorkspacePaths[step.ID+"__worktree_repo_root"] = mgr.RepoRoot()
		execution.mu.Unlock()

		// Persist worktree branch name for TUI header display
		if e.store != nil {
			if branchErr := e.store.UpdateRunBranch(e.runID, branch); branchErr != nil {
				// Log warning but don't fail the step — branch display is non-critical
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "warn",
					Message:    fmt.Sprintf("failed to persist branch name: %v", branchErr),
				})
			}
		}

		// Record branch creation as a deliverable for outcome tracking
		e.deliverableTracker.AddBranch(step.ID, branch, absPath, "Feature branch")

		// Mark CLAUDE.md as skip-worktree so prepareWorkspace() changes
		// don't get staged by git add -A in implement steps
		_ = exec.Command("git", "-C", absPath, "update-index", "--skip-worktree", "AGENTS.md").Run()

		// Run skill init commands inside the worktree (only on first creation)
		if execution.Pipeline.Requires != nil {
			for _, skillName := range execution.Pipeline.Requires.SkillNames() {
				cfg := execution.Pipeline.Requires.Skills[skillName]
				if cfg.Init == "" {
					continue
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "skill_init",
					Message:    fmt.Sprintf("running init for skill %q in worktree", skillName),
				})
				initCmd := exec.Command("sh", "-c", cfg.Init)
				initCmd.Dir = absPath
				if out, err := initCmd.CombinedOutput(); err != nil {
					return "", fmt.Errorf("skill %q init failed in worktree: %w\noutput: %s", skillName, err, string(out))
				}
			}
		}

		return absPath, nil
	}

	if e.wsManager != nil && len(step.Workspace.Mount) > 0 {
		// Update pipeline context with current step
		execution.Context.StepID = step.ID

		// Use pipeline context for template variables
		templateVars := execution.Context.ToTemplateVars()

		wsPath, err := e.wsManager.Create(workspace.WorkspaceConfig{
			Root:  wsRoot,
			Mount: toWorkspaceMounts(step.Workspace.Mount),
		}, templateVars)
		if err != nil {
			return "", err
		}

		// Anchor Claude Code path resolution to the workspace root.
		// Without .git, Claude Code walks up the directory tree and resolves
		// relative paths against the project root instead of the workspace.
		_ = exec.Command("git", "init", "-q", wsPath).Run()
		return wsPath, nil
	}

	// Create directory under .wave/workspaces/<pipeline>/<step>/
	wsPath := filepath.Join(wsRoot, pipelineID, step.ID)
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}
	// Anchor Claude Code path resolution (see mount-based workspace above)
	_ = exec.Command("git", "init", "-q", wsPath).Run()
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

// resolveCommandWorkDir determines the working directory for a command step.
// When the step uses mount-based workspaces, the project files live under the
// mount target directory (e.g. workspacePath/project/) rather than the bare
// workspace root. This function finds the first mount whose source is "./"
// (the project root) and returns the corresponding target path inside the
// workspace. If no project-root mount is found, or the step has no mounts,
// the original workspace path is returned unchanged.
func resolveCommandWorkDir(workspacePath string, step *Step) string {
	// For mount-based workspaces, find the project root mount
	for _, m := range step.Workspace.Mount {
		if m.Source == "./" || m.Source == "." {
			target := strings.TrimPrefix(m.Target, "/")
			if target == "" {
				continue
			}
			candidate := filepath.Join(workspacePath, target)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
	}

	// For worktree workspaces, look for a __wt_ directory inside the workspace
	if entries, err := os.ReadDir(workspacePath); err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "__wt_") {
				return filepath.Join(workspacePath, e.Name())
			}
		}
	}

	// If the workspace is bare (no source files) and looks empty,
	// fall back to the project root so commands like "go test ./..." find packages.
	// Check for common project markers to distinguish a real workspace from bare.
	projectMarkers := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "Makefile"}
	hasMarker := false
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(workspacePath, marker)); err == nil {
			hasMarker = true
			break
		}
	}
	if !hasMarker {
		if cwd, err := os.Getwd(); err == nil {
			// Only fall back if CWD has a project marker
			for _, marker := range projectMarkers {
				if _, err := os.Stat(filepath.Join(cwd, marker)); err == nil {
					return cwd
				}
			}
		}
	}

	return workspacePath
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

	// Resolve source_path through template variables (e.g., {{ forge.prefix }})
	sourcePath := step.Exec.SourcePath
	if execution.Context != nil && sourcePath != "" {
		sourcePath = execution.Context.ResolvePlaceholders(sourcePath)
	}

	// Load prompt from external file if source_path is set
	if sourcePath != "" {
		e.trace(audit.TracePromptLoad, step.ID, 0, map[string]string{
			"source_path": sourcePath,
		})
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			e.trace(audit.TracePromptLoadError, step.ID, 0, map[string]string{
				"source_path": sourcePath,
				"error":       err.Error(),
			})
		} else {
			prompt = string(data)
			e.trace(audit.TracePromptLoad, step.ID, 0, map[string]string{
				"source_path": sourcePath,
				"size":        fmt.Sprintf("%d", len(prompt)),
			})
		}
	} else if e.debug && step.Exec.Source == "" {
		e.trace(audit.TracePromptLoadError, step.ID, 0, map[string]string{
			"error": "step has neither source nor source_path set",
		})
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
		for idx := strings.Index(prompt, pattern); idx != -1; idx = strings.Index(prompt, pattern) {
			prompt = prompt[:idx] + sanitizedInput + prompt[idx+len(pattern):]
		}
	}

	// NOTE: Schema injection for json_schema contracts is handled exclusively by
	// buildContractPrompt → appended to user prompt (-p argument). Do NOT duplicate it here.
	// See: buildContractPrompt() which uses the correct output path from OutputArtifacts.

	// Resolve remaining template variables using pipeline context
	if execution.Context != nil {
		prompt = execution.Context.ResolvePlaceholders(prompt)
	}

	// Inject retry failure context when adapt_prompt is enabled
	execution.mu.Lock()
	attemptCtx := execution.AttemptContexts[step.ID]
	execution.mu.Unlock()

	if attemptCtx != nil {
		var sb strings.Builder
		if attemptCtx.FailedStepID != "" {
			// Rework context — this step is a rework target for a failed step
			sb.WriteString("## REWORK CONTEXT\n\n")
			fmt.Fprintf(&sb, "You are executing as a rework step for failed step %q.\n", attemptCtx.FailedStepID)
			fmt.Fprintf(&sb, "The original step failed after %d attempt(s) (ran for %s).\n\n", attemptCtx.Attempt, attemptCtx.StepDuration.Round(time.Second))
		} else {
			sb.WriteString("## RETRY CONTEXT\n\n")
			fmt.Fprintf(&sb, "This is attempt %d of %d. The previous attempt failed.\n\n", attemptCtx.Attempt, attemptCtx.MaxAttempts)
		}
		if attemptCtx.PriorError != "" {
			sb.WriteString("### Previous Error\n```\n")
			sb.WriteString(attemptCtx.PriorError)
			sb.WriteString("\n```\n\n")
		}
		if len(attemptCtx.ContractErrors) > 0 {
			sb.WriteString("### Contract Validation Errors\n")
			for _, ce := range attemptCtx.ContractErrors {
				sb.WriteString(fmt.Sprintf("- %s\n", ce))
			}
			sb.WriteString("\n")
		}
		if attemptCtx.PriorStdout != "" {
			sb.WriteString(fmt.Sprintf("### Previous Output (last %d chars)\n```\n", maxStdoutTailChars))
			sb.WriteString(attemptCtx.PriorStdout)
			sb.WriteString("\n```\n\n")
		}
		if len(attemptCtx.PartialArtifacts) > 0 {
			sb.WriteString("### Partial Artifacts from Failed Step\n")
			for name, path := range attemptCtx.PartialArtifacts {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", name, path))
			}
			sb.WriteString("\n")
		}
		if attemptCtx.ReviewFeedbackPath != "" {
			sb.WriteString("### Agent Review Feedback\n\n")
			sb.WriteString(fmt.Sprintf("A review agent found issues with the previous implementation. Structured feedback is available at: `%s`\n", attemptCtx.ReviewFeedbackPath))
			sb.WriteString("Read this file to understand the specific issues and suggestions before making changes.\n\n")
		}
		if len(attemptCtx.ContractErrors) > 0 {
			sb.WriteString("Fix the specific failure above. Do not start from scratch.\n\n---\n\n")
		} else {
			sb.WriteString("Please address the issues from the previous attempt and try a different approach if needed.\n\n---\n\n")
		}
		sb.WriteString(prompt)
		prompt = sb.String()
	}

	// Inject thread conversation context when the step is part of a thread group
	if step.Thread != "" && execution.ThreadManager != nil {
		fidelity := step.EffectiveFidelity()
		transcript := execution.ThreadManager.GetTranscript(context.Background(), step.Thread, fidelity)
		if transcript != "" {
			var sb strings.Builder
			sb.WriteString("## THREAD CONTEXT\n\n")
			sb.WriteString("The following is conversation history from prior steps in this thread group.\n\n")
			sb.WriteString(transcript)
			sb.WriteString("\n---\n\n")
			sb.WriteString(prompt)
			prompt = sb.String()

			e.trace(audit.TraceThreadInject, step.ID, int64(len(transcript)), map[string]string{
				"thread":   step.Thread,
				"fidelity": fidelity,
				"size":     fmt.Sprintf("%d", len(transcript)),
			})
		}
	}

	// Inject output artifact paths so the persona knows where to write artifacts
	if len(step.OutputArtifacts) > 0 {
		var sb strings.Builder
		sb.WriteString("\n## Output Artifacts\n\n")
		sb.WriteString("Write the requested artifacts to these paths (in workspace root):\n\n")
		for _, art := range step.OutputArtifacts {
			// Use Path if specified, otherwise just the Name
			artPath := art.Path
			if artPath == "" {
				artPath = art.Name
			}
			sb.WriteString(fmt.Sprintf("- `%s` (as: %s)\n", artPath, art.Name))
		}
		sb.WriteString("\nThe pipeline will validate these artifacts. Write to the exact paths above.\n\n")
		sb.WriteString(prompt)
		prompt = sb.String()
	}

	return prompt
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

	// Build artifact type map for validation
	artifactTypes := e.buildArtifactTypeMap(execution)

	for _, ref := range step.Memory.InjectArtifacts {
		artName := ref.As
		if artName == "" {
			artName = ref.Artifact
		}
		destPath := filepath.Join(artifactsDir, artName)

		// Cross-pipeline artifact reference: look up from prior pipeline outputs
		if ref.Pipeline != "" && e.crossPipelineArtifacts != nil {
			pipelineArtifacts, hasPipeline := e.crossPipelineArtifacts[ref.Pipeline]
			if !hasPipeline || pipelineArtifacts == nil {
				if ref.Optional {
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "step_progress",
						Message:    fmt.Sprintf("optional cross-pipeline artifact '%s' from pipeline '%s' not found, skipping", ref.Artifact, ref.Pipeline),
					})
					continue
				}
				return fmt.Errorf("cross-pipeline artifact '%s' from pipeline '%s' not found", ref.Artifact, ref.Pipeline)
			}
			data, hasArtifact := pipelineArtifacts[ref.Artifact]
			if !hasArtifact {
				if ref.Optional {
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "step_progress",
						Message:    fmt.Sprintf("optional cross-pipeline artifact '%s' from pipeline '%s' not found, skipping", ref.Artifact, ref.Pipeline),
					})
					continue
				}
				return fmt.Errorf("cross-pipeline artifact '%s' not found in pipeline '%s' outputs", ref.Artifact, ref.Pipeline)
			}
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write artifact '%s': %w", artName, err)
			}
			execution.Context.SetArtifactPath(artName, destPath)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("injected cross-pipeline artifact %s from pipeline %s", artName, ref.Pipeline),
			})

			// Type validation (if specified)
			if ref.Type != "" {
				key := ref.Pipeline + ":" + ref.Artifact
				declaredType := artifactTypes[key]
				if declaredType != "" && declaredType != ref.Type {
					return fmt.Errorf("artifact '%s' type mismatch: expected %s, got %s", ref.Artifact, ref.Type, declaredType)
				}
			}

			// Schema validation for input artifacts (if schema_path is specified)
			if ref.SchemaPath != "" {
				if err := contract.ValidateInputArtifact(artName, ref.SchemaPath, workspacePath); err != nil {
					return fmt.Errorf("input artifact '%s' schema validation failed: %w", artName, err)
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("validated artifact %s against schema %s", artName, ref.SchemaPath),
				})
			}
			continue
		}

		// Try registered artifact path first
		key := ref.Step + ":" + ref.Artifact
		execution.mu.Lock()
		artifactPath, ok := execution.ArtifactPaths[key]
		execution.mu.Unlock()

		// Existence validation
		if !ok {
			// Try fallback: check if we have stdout results from the step
			execution.mu.Lock()
			result, exists := execution.Results[ref.Step]
			execution.mu.Unlock()
			if exists {
				if stdout, ok := result["stdout"].(string); ok {
					// Type validation (if specified)
					if ref.Type != "" {
						declaredType := artifactTypes[key]
						if declaredType != "" && declaredType != ref.Type {
							return fmt.Errorf("artifact '%s' type mismatch: expected %s, got %s", ref.Artifact, ref.Type, declaredType)
						}
					}
					if err := os.WriteFile(destPath, []byte(stdout), 0644); err != nil {
						return fmt.Errorf("failed to write artifact '%s': %w", artName, err)
					}
					// Register artifact path in context for template resolution
					execution.Context.SetArtifactPath(artName, destPath)
					e.emit(event.Event{
						Timestamp:  time.Now(),
						PipelineID: pipelineID,
						StepID:     step.ID,
						State:      "step_progress",
						Message:    fmt.Sprintf("injected artifact %s from step %s stdout", artName, ref.Step),
					})
					continue
				}
			}

			// Artifact not found - check if optional
			if ref.Optional {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("optional artifact '%s' from step '%s' not found, skipping", ref.Artifact, ref.Step),
				})
				continue
			}
			return fmt.Errorf("required artifact '%s' from step '%s' not found", ref.Artifact, ref.Step)
		}

		// Type validation (if specified)
		if ref.Type != "" {
			declaredType := artifactTypes[key]
			if declaredType != "" && declaredType != ref.Type {
				return fmt.Errorf("artifact '%s' type mismatch: expected %s, got %s", ref.Artifact, ref.Type, declaredType)
			}
		}

		srcData, err := os.ReadFile(artifactPath)
		if err != nil {
			if ref.Optional {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("optional artifact '%s' could not be read, skipping: %v", ref.Artifact, err),
				})
				continue
			}
			return fmt.Errorf("failed to read required artifact '%s': %w", ref.Artifact, err)
		}

		if err := os.WriteFile(destPath, srcData, 0644); err != nil {
			return fmt.Errorf("failed to write artifact '%s': %w", artName, err)
		}
		// Register artifact path in context for template resolution
		execution.Context.SetArtifactPath(artName, destPath)
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "step_progress",
			Message:    fmt.Sprintf("injected artifact %s from %s (%s)", artName, ref.Step, artifactPath),
		})

		// Schema validation for input artifacts (if schema_path is specified)
		if ref.SchemaPath != "" {
			if err := contract.ValidateInputArtifact(artName, ref.SchemaPath, workspacePath); err != nil {
				return fmt.Errorf("input artifact '%s' schema validation failed: %w", artName, err)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("validated artifact %s against schema %s", artName, ref.SchemaPath),
			})
		}
	}

	return nil
}

// buildArtifactTypeMap builds a map of artifact keys to their declared types
func (e *DefaultPipelineExecutor) buildArtifactTypeMap(execution *PipelineExecution) map[string]string {
	types := make(map[string]string)
	for _, step := range execution.Pipeline.Steps {
		for _, art := range step.OutputArtifacts {
			key := step.ID + ":" + art.Name
			types[key] = art.Type
		}
	}
	return types
}

func (e *DefaultPipelineExecutor) writeOutputArtifacts(execution *PipelineExecution, step *Step, workspacePath string, stdout []byte) {
	// Get artifact directory for stdout artifacts
	artifactDir := execution.Manifest.Runtime.Artifacts.GetDefaultArtifactDir()

	for _, art := range step.OutputArtifacts {
		key := step.ID + ":" + art.Name
		var artPath string

		// Handle stdout artifacts differently
		if art.IsStdoutArtifact() {
			// Stdout artifacts go to .wave/artifacts/<step-id>/<name>
			artPath = filepath.Join(workspacePath, artifactDir, step.ID, art.Name)
			_ = os.MkdirAll(filepath.Dir(artPath), 0755)

			// Write stdout content to artifact
			if err := os.WriteFile(artPath, stdout, 0644); err != nil {
				e.trace(audit.TraceArtifactWrite, step.ID, 0, map[string]string{
					"artifact": art.Name,
					"path":     artPath,
					"error":    err.Error(),
				})
			}
			execution.mu.Lock()
			execution.ArtifactPaths[key] = artPath
			execution.mu.Unlock()

			e.trace(audit.TraceArtifactWrite, step.ID, 0, map[string]string{
				"artifact": art.Name,
				"path":     artPath,
				"size":     fmt.Sprintf("%d", len(stdout)),
			})
		} else {
			// File-based artifacts: resolve path using pipeline context
			resolvedPath := execution.Context.ResolveArtifactPath(art)
			artPath = filepath.Join(workspacePath, resolvedPath)

			// If the persona already wrote the file, trust it and don't overwrite
			if _, err := os.Stat(artPath); err == nil {
				execution.mu.Lock()
				execution.ArtifactPaths[key] = artPath
				execution.mu.Unlock()
				e.trace(audit.TraceArtifactPreserved, step.ID, 0, map[string]string{
					"artifact": art.Name,
					"path":     artPath,
				})
			} else if len(stdout) > 0 {
				// Fall back to writing ResultContent (skip when nil/empty
				// to avoid creating zero-byte files from empty adapter output)
				_ = os.MkdirAll(filepath.Dir(artPath), 0755)
				_ = os.WriteFile(artPath, stdout, 0644)
				execution.mu.Lock()
				execution.ArtifactPaths[key] = artPath
				execution.mu.Unlock()
			}
		}

		// Archive artifact to a step-specific path so shared-worktree steps
		// don't all point at the same file in the DB. The injection system
		// keeps using artPath (the workspace-relative location), but the DB
		// gets the archived copy which survives subsequent steps overwriting
		// the same relative path.
		registeredPath := artPath
		if !art.IsStdoutArtifact() {
			archiveDir := filepath.Join(workspacePath, ".wave", "artifacts", step.ID)
			archiveName := art.Name
			if art.Type == "json" && !strings.HasSuffix(archiveName, ".json") {
				archiveName += ".json"
			}
			archivePath := filepath.Join(archiveDir, archiveName)
			if data, readErr := os.ReadFile(artPath); readErr == nil {
				if mkErr := os.MkdirAll(archiveDir, 0755); mkErr == nil {
					if writeErr := os.WriteFile(archivePath, data, 0644); writeErr == nil {
						registeredPath = archivePath
					}
				}
			}
		}

		// Register artifact in DB for web dashboard visibility
		if e.store != nil {
			var size int64
			if info, err := os.Stat(registeredPath); err == nil {
				size = info.Size()
			}
			_ = e.store.RegisterArtifact(execution.Status.ID, step.ID, art.Name, registeredPath, art.Type, size)
		}
	}
}

// validateSkillRefs validates skill references at manifest (global + persona)
// and pipeline scopes against the skill store. Returns nil if no store is set.
// pipelineSkills should be the already-resolved (template-expanded) skill list
// so that callers never need to mutate the shared Pipeline struct.
func (e *DefaultPipelineExecutor) validateSkillRefs(pipelineSkills []string, pipelineName string, m *manifest.Manifest) []error {
	if e.skillStore == nil {
		return nil
	}

	// Validate manifest-level skills (global + persona scopes)
	var personas []skill.PersonaSkills
	for name, persona := range m.Personas {
		if len(persona.Skills) > 0 {
			personas = append(personas, skill.PersonaSkills{Name: name, Skills: persona.Skills})
		}
	}
	errs := skill.ValidateManifestSkills(m.Skills, personas, e.skillStore)

	// Validate pipeline-level skills (already template-resolved by caller)
	if len(pipelineSkills) > 0 {
		errs = append(errs, skill.ValidateSkillRefs(pipelineSkills, "pipeline:"+pipelineName, e.skillStore)...)
	}

	return errs
}

// parseStallTimeout parses the stall timeout from the manifest runtime config.
// Returns 0 if not configured or invalid.
func (e *DefaultPipelineExecutor) parseStallTimeout(m *manifest.Manifest) time.Duration {
	if m == nil || m.Runtime.StallTimeout == "" {
		return 0
	}
	d, err := time.ParseDuration(m.Runtime.StallTimeout)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

// terminalHookTimeout is the maximum time terminal hooks (run_completed, run_failed)
// are allowed to run with a detached context.
const terminalHookTimeout = 30 * time.Second

// runTerminalHooks executes lifecycle hooks with a fresh, detached context.
// Terminal events fire after the pipeline context may already be cancelled.
func (e *DefaultPipelineExecutor) runTerminalHooks(evt hooks.HookEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), terminalHookTimeout)
	defer cancel()
	if e.hookRunner != nil {
		e.hookRunner.RunHooks(ctx, evt)
	}
	e.fireWebhooks(ctx, evt)
}

// trace emits a structured NDJSON trace event when debug tracing is enabled.
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
					var etaMs int64
					if e.etaCalculator != nil {
						etaMs = e.etaCalculator.RemainingMs()
					}
					e.emit(event.Event{
						PipelineID:      pipelineID,
						StepID:          stepID,
						State:           event.StateStepProgress,
						Timestamp:       time.Now(),
						EstimatedTimeMs: etaMs,
					})
				}
			}
		}()
	}

	return cancel
}

// pollCancellation checks the database for cross-process cancellation requests.
// When another process (TUI, webui, CLI) writes a cancellation record, this
// goroutine detects it and cancels the executor's context to stop execution.
func (e *DefaultPipelineExecutor) pollCancellation(ctx context.Context, runID string, cancel context.CancelFunc) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if rec, err := e.store.CheckCancellation(runID); err == nil && rec != nil {
				cancel()
				return
			}
		}
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
		_ = e.logger.LogToolCall(pipelineID, step.ID, "relay.Compact", fmt.Sprintf("tokens=%d summary_len=%d persona=%s", tokensUsed, len(summary), summarizerName))
	}

	return nil
}

// trackStepDeliverables automatically tracks deliverables produced by a completed step
func (e *DefaultPipelineExecutor) trackStepDeliverables(execution *PipelineExecution, step *Step) {
	if e.deliverableTracker == nil {
		return
	}

	// Get workspace path for this step
	execution.mu.Lock()
	workspacePath, exists := execution.WorkspacePaths[step.ID]
	execution.mu.Unlock()
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
		// NOTE: DB registration is handled by writeOutputArtifacts (with archiving).
		// Do NOT duplicate it here.
	}

}

// buildContractPrompt generates a contract compliance section that is appended
// to the user prompt (-p argument) at execution time. This tells the persona
// exactly what format the output must be in, so pipeline authors don't need to
// repeat format requirements in their prompts.
//
// This is the SINGLE source of truth for schema injection — it includes security
// validation (path traversal, content sanitization) and the full schema content.
func (e *DefaultPipelineExecutor) buildContractPrompt(step *Step, ctx *PipelineContext) string {
	var b strings.Builder

	// ── Output artifact guidance ──────────────────────────────────────
	// Always generated when the step has output_artifacts, regardless of
	// whether a handover contract exists. This is the SINGLE source of
	// truth for telling the persona what to write and where.
	if len(step.OutputArtifacts) > 0 {
		b.WriteString("## Output Requirements\n\n")
		for _, artifact := range step.OutputArtifacts {
			path := artifact.Path
			if ctx != nil {
				path = ctx.ResolveArtifactPath(artifact)
			}

			switch artifact.Type {
			case "json":
				b.WriteString(fmt.Sprintf("Write valid JSON to `%s` using the Write tool.\n", path))
				b.WriteString("The file must contain ONLY a JSON object — no markdown, no explanatory text, no code fences.\n\n")
			case "markdown":
				b.WriteString(fmt.Sprintf("Write your output as Markdown to `%s` using the Write tool.\n\n", path))
			default:
				b.WriteString(fmt.Sprintf("Write your output to `%s` using the Write tool.\n\n", path))
			}
		}
	}

	// ── Contract compliance (formal schema validation) ────────────────
	// Additional guidance when a handover contract is defined.
	switch step.Handover.Contract.Type {
	case "json_schema":
		b.WriteString("### Contract Schema\n\n")
		b.WriteString("**CRITICAL**: This step will FAIL validation if the output is not valid JSON conforming to the schema below.\n\n")

		// Load and security-validate schema content
		schemaContent := e.loadSecureSchemaContent(step)
		if schemaContent != "" {
			// Include the full schema for the persona to reference
			b.WriteString("**Schema** (your output must conform to this):\n```json\n")
			b.WriteString(schemaContent)
			b.WriteString("\n```\n\n")

			// Also extract required fields and build a skeleton example
			var schema struct {
				Required   []string                  `json:"required"`
				Properties map[string]map[string]any `json:"properties"`
			}
			if json.Unmarshal([]byte(schemaContent), &schema) == nil && len(schema.Required) > 0 {
				b.WriteString(fmt.Sprintf("**Required fields**: `%s`\n\n", strings.Join(schema.Required, "`, `")))

				// Build a concrete JSON skeleton from required fields
				b.WriteString("**Example structure** (populate with real data):\n```json\n{\n")
				for i, field := range schema.Required {
					placeholder := schemaFieldPlaceholder(field, schema.Properties[field])
					if i < len(schema.Required)-1 {
						b.WriteString(fmt.Sprintf("  %q: %s,\n", field, placeholder))
					} else {
						b.WriteString(fmt.Sprintf("  %q: %s\n", field, placeholder))
					}
				}
				b.WriteString("}\n```\n")
			}
		}

	case "test_suite":
		b.WriteString("### Test Validation\n\n")
		cmd := step.Handover.Contract.Command
		if cmd != "" {
			b.WriteString(fmt.Sprintf("After you complete your work, the following command will be run to validate your output:\n```\n%s\n```\n", cmd))
		} else {
			b.WriteString("After you complete your work, a test suite will be run to validate your output.\n")
		}
		b.WriteString("If tests fail, the step fails.\n")

	case "llm_judge":
		b.WriteString("### LLM Judge Evaluation\n\n")
		b.WriteString("After you complete your work, an LLM judge will evaluate your output against the following criteria:\n\n")
		for _, criterion := range step.Handover.Contract.Criteria {
			b.WriteString(fmt.Sprintf("- %s\n", criterion))
		}
		threshold := step.Handover.Contract.Threshold
		if threshold <= 0 {
			threshold = 1.0
		}
		b.WriteString(fmt.Sprintf("\nYou must satisfy at least %.0f%% of these criteria to pass.\n", threshold*100))

	case "agent_review":
		b.WriteString("### Agent Review Validation\n\n")
		b.WriteString("After you complete your work, a separate review agent will evaluate your output.\n")
		// Use EffectiveContracts to handle both singular and plural config
		for _, c := range step.Handover.EffectiveContracts() {
			if c.Type == "agent_review" {
				if c.CriteriaPath != "" {
					b.WriteString(fmt.Sprintf("Review criteria are loaded from: `%s`\n", c.CriteriaPath))
				}
				if c.Persona != "" {
					b.WriteString(fmt.Sprintf("Reviewer persona: `%s`\n", c.Persona))
				}
			}
		}
		b.WriteString("The reviewer will return a structured verdict (pass/fail/warn) with specific issues and suggestions.\n")
		b.WriteString("If the verdict is 'fail', the step fails.\n")
	}

	// ── Injected artifact guidance ────────────────────────────────────
	// Always generated when the step has inject_artifacts, regardless of
	// whether a handover contract exists. Tells the persona where to read.
	if len(step.Memory.InjectArtifacts) > 0 {
		b.WriteString("\n## Available Artifacts\n\n")
		b.WriteString("The following artifacts have been injected into your workspace:\n\n")
		for _, ref := range step.Memory.InjectArtifacts {
			name := ref.As
			if name == "" {
				name = ref.Artifact
			}
			b.WriteString(fmt.Sprintf("- `%s` → `.wave/artifacts/%s`\n", name, name))
		}
		b.WriteString("\nThese artifacts contain ALL data you need from prior pipeline steps. ")
		b.WriteString("Read these files instead of fetching equivalent data from external sources.\n")
	}

	if b.Len() == 0 {
		return ""
	}
	return b.String()
}

// loadSecureSchemaContent loads schema content with security validation
// (path traversal prevention, content sanitization). Returns empty string
// if the schema is unavailable, invalid, or fails security checks.
func (e *DefaultPipelineExecutor) loadSecureSchemaContent(step *Step) string {
	if step.Handover.Contract.SchemaPath != "" {
		// Validate path for traversal attacks
		if e.pathValidator != nil {
			validationResult, pathErr := e.pathValidator.ValidatePath(step.Handover.Contract.SchemaPath)
			if pathErr != nil {
				e.securityLogger.LogViolation(
					string(security.ViolationPathTraversal),
					string(security.SourceSchemaPath),
					fmt.Sprintf("Schema path validation failed for step %s", step.ID),
					security.SeverityCritical,
					true,
				)
				return ""
			}
			if !validationResult.IsValid {
				return ""
			}
			data, readErr := os.ReadFile(validationResult.ValidatedPath)
			if readErr != nil {
				return ""
			}
			return e.sanitizeSchemaContent(step, string(data))
		}
		// No path validator (e.g. in tests) — read directly
		data, err := os.ReadFile(step.Handover.Contract.SchemaPath)
		if err != nil {
			return ""
		}
		return string(data)
	}

	if step.Handover.Contract.Schema != "" {
		return e.sanitizeSchemaContent(step, step.Handover.Contract.Schema)
	}

	return ""
}

// sanitizeSchemaContent applies prompt injection sanitization to schema content.
// Returns the sanitized content, or empty string if sanitization fails.
func (e *DefaultPipelineExecutor) sanitizeSchemaContent(step *Step, content string) string {
	if e.inputSanitizer == nil {
		return content
	}
	sanitized, sanitizationActions, err := e.inputSanitizer.SanitizeSchemaContent(content)
	if err != nil {
		e.securityLogger.LogViolation(
			string(security.ViolationInputValidation),
			string(security.SourceSchemaPath),
			fmt.Sprintf("Schema content sanitization failed for step %s", step.ID),
			security.SeverityHigh,
			true,
		)
		return ""
	}
	if len(sanitizationActions) > 0 {
		e.securityLogger.LogViolation(
			string(security.ViolationPromptInjection),
			string(security.SourceSchemaPath),
			fmt.Sprintf("Schema content sanitized for step %s: %v", step.ID, sanitizationActions),
			security.SeverityMedium,
			false,
		)
	}
	return sanitized
}

// schemaFieldPlaceholder returns a JSON placeholder value for a schema property,
// used in the contract compliance example skeleton.
func schemaFieldPlaceholder(_ string, prop map[string]any) string {
	if prop == nil {
		return "\"...\""
	}
	t, _ := prop["type"].(string)
	switch t {
	case "string":
		return "\"...\""
	case "integer", "number":
		return "0"
	case "boolean":
		return "false"
	case "array":
		return "[...]"
	case "object":
		return "{...}"
	default:
		return "\"...\""
	}
}

// processStepOutcomes extracts declared outcomes from step artifacts and registers
// them with the deliverable tracker for display in the pipeline output summary.
// Errors are logged as warnings — outcome extraction never fails a step.
//
// When a json_path contains [*] wildcard syntax, all array elements are extracted
// and each is registered as a separate deliverable. The optional json_path_label
// field provides per-item labels; when absent, items are labeled with their index.
func (e *DefaultPipelineExecutor) processStepOutcomes(execution *PipelineExecution, step *Step) {
	if e.deliverableTracker == nil || len(step.Outcomes) == 0 {
		return
	}

	pipelineID := execution.Status.ID
	execution.mu.Lock()
	workspacePath := execution.WorkspacePaths[step.ID]
	execution.mu.Unlock()
	if workspacePath == "" {
		return
	}

	for _, outcome := range step.Outcomes {
		artifactPath := filepath.Clean(filepath.Join(workspacePath, outcome.ExtractFrom))
		cleanWorkspace := filepath.Clean(workspacePath) + string(filepath.Separator)
		if !strings.HasPrefix(artifactPath, cleanWorkspace) {
			msg := fmt.Sprintf("[%s] outcome: path %q escapes workspace, skipping", step.ID, outcome.ExtractFrom)
			e.deliverableTracker.AddOutcomeWarning(msg)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "warning",
				Message:    msg,
			})
			continue
		}
		data, err := os.ReadFile(artifactPath)
		if err != nil {
			msg := fmt.Sprintf("[%s] outcome: cannot read %s: %v", step.ID, outcome.ExtractFrom, err)
			e.deliverableTracker.AddOutcomeWarning(msg)
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "warning",
				Message:    msg,
			})
			continue
		}

		// file/artifact types: use the artifact path directly as the deliverable value
		if outcome.Type == "file" || outcome.Type == "artifact" {
			label := outcome.Label
			if label == "" {
				label = outcome.Type
			}
			e.registerOutcomeDeliverable(step.ID, outcome.Type, label, artifactPath, fmt.Sprintf("Produced by step %s", step.ID))
			if e.store != nil {
				_ = e.store.RecordOutcome(pipelineID, step.ID, outcome.Type, label, artifactPath)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      stateRunning,
				Message:    fmt.Sprintf("outcome: %s = %s", label, artifactPath),
			})
			continue
		}

		// Wildcard path: extract all array elements as separate deliverables
		if ContainsWildcard(outcome.JSONPath) {
			e.processWildcardOutcome(execution, step, outcome, data)
			continue
		}

		value, err := ExtractJSONPath(data, outcome.JSONPath)
		if err != nil {
			var emptyErr *emptyArrayError
			if errors.As(err, &emptyErr) {
				// Empty array is a "no results" condition, not an error.
				// Show a friendly message in the summary only — skip the real-time warning event.
				msg := fmt.Sprintf("[%s] outcome: no items in %s — skipping %s extraction from %s", step.ID, emptyErr.Field, outcome.JSONPath, outcome.ExtractFrom)
				e.deliverableTracker.AddOutcomeWarning(msg)
			} else {
				msg := fmt.Sprintf("[%s] outcome: %s at %s: %v", step.ID, outcome.JSONPath, outcome.ExtractFrom, err)
				e.deliverableTracker.AddOutcomeWarning(msg)
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "warning",
					Message:    msg,
				})
			}
			continue
		}

		label := outcome.Label
		if label == "" {
			label = outcome.Type
		}
		desc := fmt.Sprintf("Extracted from %s at %s", outcome.ExtractFrom, outcome.JSONPath)

		e.registerOutcomeDeliverable(step.ID, outcome.Type, label, value, desc)

		// Persist outcome in state DB so it survives worktree cleanup
		if e.store != nil {
			if err := e.store.RecordOutcome(pipelineID, step.ID, outcome.Type, label, value); err != nil {
				if e.logger != nil {
					_ = e.logger.LogToolCall(pipelineID, step.ID, "recordOutcome",
						fmt.Sprintf("type=%s label=%s err=%v", outcome.Type, label, err))
				}
			}
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      stateRunning,
			Message:    fmt.Sprintf("outcome: %s = %s", label, value),
		})
	}
}

// processWildcardOutcome handles outcome definitions with [*] wildcard paths,
// extracting all array elements and registering each as a separate deliverable.
func (e *DefaultPipelineExecutor) processWildcardOutcome(execution *PipelineExecution, step *Step, outcome OutcomeDef, data []byte) {
	pipelineID := execution.Status.ID

	values, err := ExtractJSONPathAll(data, outcome.JSONPath)
	if err != nil {
		msg := fmt.Sprintf("[%s] outcome: %s at %s: %v", step.ID, outcome.JSONPath, outcome.ExtractFrom, err)
		e.deliverableTracker.AddOutcomeWarning(msg)
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      "warning",
			Message:    msg,
		})
		return
	}

	// Empty array — log friendly message and skip
	if len(values) == 0 {
		msg := fmt.Sprintf("[%s] outcome: empty array at %s — skipping extraction from %s", step.ID, outcome.JSONPath, outcome.ExtractFrom)
		e.deliverableTracker.AddOutcomeWarning(msg)
		return
	}

	// Extract per-item labels if json_path_label is set
	var labels []string
	if outcome.JSONPathLabel != "" && ContainsWildcard(outcome.JSONPathLabel) {
		labels, _ = ExtractJSONPathAll(data, outcome.JSONPathLabel)
	}

	baseLabel := outcome.Label
	if baseLabel == "" {
		baseLabel = outcome.Type
	}

	total := len(values)
	for i, value := range values {
		var label string
		if i < len(labels) && labels[i] != "" {
			label = fmt.Sprintf("%s: %s", baseLabel, labels[i])
		} else {
			label = fmt.Sprintf("%s (%d/%d)", baseLabel, i+1, total)
		}

		desc := fmt.Sprintf("Extracted from %s at %s [%d]", outcome.ExtractFrom, outcome.JSONPath, i)
		e.registerOutcomeDeliverable(step.ID, outcome.Type, label, value, desc)

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      stateRunning,
			Message:    fmt.Sprintf("outcome: %s = %s", label, value),
		})
	}
}

// registerOutcomeDeliverable registers a single outcome value with the deliverable tracker
// based on the outcome type.
func (e *DefaultPipelineExecutor) registerOutcomeDeliverable(stepID, outcomeType, label, value, desc string) {
	switch outcomeType {
	case "pr":
		e.deliverableTracker.AddPR(stepID, label, value, desc)
	case "issue":
		e.deliverableTracker.AddIssue(stepID, label, value, desc)
	case "deployment":
		e.deliverableTracker.AddDeployment(stepID, label, value, desc)
	case "file":
		e.deliverableTracker.AddFile(stepID, label, value, desc)
	case "artifact":
		e.deliverableTracker.AddArtifact(stepID, label, value, desc)
	default:
		// "url" or any unknown type → generic URL
		e.deliverableTracker.AddURL(stepID, label, value, desc)
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

// executeCompositionStep handles steps that reference sub-pipelines (via the
// `pipeline:` field) rather than executing a persona directly. It loads the
// referenced pipeline YAML, resolves the step's input template, and delegates
// execution to a fresh DefaultPipelineExecutor instance.
func (e *DefaultPipelineExecutor) executeCompositionStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	// Route gate steps to the gate executor
	if step.Gate != nil {
		return e.executeGateInDAG(ctx, execution, step)
	}

	// Route composition primitives
	if step.Iterate != nil {
		return e.executeIterateInDAG(ctx, execution, step)
	}
	if step.Aggregate != nil {
		return e.executeAggregateInDAG(ctx, execution, step)
	}
	if step.Branch != nil {
		return e.executeBranchInDAG(ctx, execution, step)
	}
	if step.Loop != nil {
		return e.executeLoopInDAG(ctx, execution, step)
	}

	// Fall through: bare sub-pipeline step
	input := e.resolveSubPipelineInput(execution, step)
	return e.runNamedSubPipeline(ctx, execution, step, step.SubPipeline, input)
}

// resolveSubPipelineInput resolves the input string for a composition step,
// using the step's SubInput template or falling back to the pipeline input.
func (e *DefaultPipelineExecutor) resolveSubPipelineInput(execution *PipelineExecution, step *Step) string {
	input := execution.Input
	if step.SubInput != "" {
		// Resolve {{ input }} directly (not handled by PipelineContext.ResolvePlaceholders)
		resolved := strings.ReplaceAll(step.SubInput, "{{ input }}", execution.Input)
		resolved = strings.ReplaceAll(resolved, "{{input}}", execution.Input)
		// Then resolve remaining pipeline context variables
		if execution.Context != nil {
			resolved = execution.Context.ResolvePlaceholders(resolved)
		}
		if resolved != "" {
			input = resolved
		}
	}
	return input
}

// runNamedSubPipeline loads a pipeline by name from disk and executes it as a
// child of the current execution. It handles timeout, artifact injection/extraction,
// context merging, and parent-child state linking.
func (e *DefaultPipelineExecutor) runNamedSubPipeline(ctx context.Context, execution *PipelineExecution, step *Step, pipelineName, input string) error {
	pipelineID := execution.Status.ID

	// Apply lifecycle timeout from sub-pipeline config
	execCtx, cancel := subPipelineTimeout(ctx, step.Config)
	defer cancel()

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("composition: loading sub-pipeline %q", pipelineName),
	})

	// Load the sub-pipeline from disk
	loader := &YAMLPipelineLoader{}
	subPipelinePath := filepath.Join(".wave", "pipelines", pipelineName+".yaml")
	subPipeline, err := loader.Load(subPipelinePath)
	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return fmt.Errorf("failed to load sub-pipeline %q: %w", pipelineName, err)
	}

	// Build executor options for the child pipeline, inheriting configuration
	// from the parent but generating a fresh run ID.
	childOpts := []ExecutorOption{
		WithDebug(e.debug),
	}
	if e.emitter != nil {
		childOpts = append(childOpts, WithEmitter(e.emitter))
	}
	if e.store != nil {
		childOpts = append(childOpts, WithStateStore(e.store))
	}
	if e.modelOverride != "" {
		childOpts = append(childOpts, WithModelOverride(e.modelOverride))
	}
	if e.adapterOverride != "" {
		childOpts = append(childOpts, WithAdapterOverride(e.adapterOverride))
	}
	if e.stepTimeoutOverride > 0 {
		childOpts = append(childOpts, WithStepTimeout(e.stepTimeoutOverride))
	}
	if e.debugTracer != nil {
		childOpts = append(childOpts, WithDebugTracer(e.debugTracer))
	}
	if e.logger != nil {
		childOpts = append(childOpts, WithAuditLogger(e.logger))
	}
	if e.wsManager != nil {
		childOpts = append(childOpts, WithWorkspaceManager(e.wsManager))
	}
	if e.relayMonitor != nil {
		childOpts = append(childOpts, WithRelayMonitor(e.relayMonitor))
	}
	if e.skillStore != nil {
		childOpts = append(childOpts, withSkillStore(e.skillStore))
	}

	// Inject parent artifacts into child executor when config specifies injection
	if step.Config != nil && len(step.Config.Inject) > 0 {
		paths := make(map[string]string, len(step.Config.Inject))
		for _, name := range step.Config.Inject {
			path := execution.Context.GetArtifactPath(name)
			if path == "" {
				return fmt.Errorf("artifact %q not found in parent context for sub-pipeline injection", name)
			}
			paths[name] = path
		}
		childOpts = append(childOpts, WithParentArtifactPaths(paths))
	}

	// Pass parent workspace path if this step has a resolved workspace
	execution.mu.Lock()
	if ws, ok := execution.WorkspacePaths[step.ID]; ok && ws != "" {
		childOpts = append(childOpts, WithParentWorkspacePath(ws))
	}
	execution.mu.Unlock()

	// Propagate max_cycles to child pipeline's loop config
	if step.Config != nil && step.Config.MaxCycles > 0 {
		for i := range subPipeline.Steps {
			if subPipeline.Steps[i].Loop != nil && subPipeline.Steps[i].Loop.MaxIterations == 0 {
				subPipeline.Steps[i].Loop.MaxIterations = step.Config.MaxCycles
			}
		}
	}

	// Create a run record for the child pipeline so it appears in the dashboard.
	// Link to parent immediately so the WebUI can nest it from the start.
	if e.store != nil {
		childRunID := e.createRunID(pipelineName, 4, input)
		childOpts = append(childOpts, WithRunID(childRunID))
		_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
	}

	childOpts = append(childOpts, WithRegistry(e.registry))

	// Propagate step-level adapter override to child sub-pipeline
	if step.Adapter != "" {
		childOpts = append(childOpts, WithAdapterOverride(step.Adapter))
	} else if e.adapterOverride != "" {
		// Propagate CLI --adapter flag to sub-pipelines
		childOpts = append(childOpts, WithAdapterOverride(e.adapterOverride))
	}

	childExecutor := NewDefaultPipelineExecutor(e.runner, childOpts...)

	// Mark child run as running in the dashboard
	childRunID := childExecutor.runID
	if e.store != nil && childRunID != "" {
		_ = e.store.UpdateRunStatus(childRunID, "running", "", 0)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("composition: executing sub-pipeline %q", pipelineName),
	})

	if err := childExecutor.Execute(execCtx, subPipeline, execution.Manifest, input); err != nil {
		// Link parent-child state and update status even on failure
		if e.store != nil && childRunID != "" {
			_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
			_ = e.store.UpdateRunStatus(childRunID, "failed", err.Error(), childExecutor.GetTotalTokens())
		}
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return fmt.Errorf("sub-pipeline %q failed: %w", pipelineName, err)
	}

	// Link parent-child state and mark completed
	if e.store != nil && childRunID != "" {
		_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
		_ = e.store.UpdateRunStatus(childRunID, "completed", "", childExecutor.GetTotalTokens())

		// Propagate child's worktree branch to parent so the diff endpoint works
		if childRun, err := e.store.GetRun(childRunID); err == nil && childRun.BranchName != "" {
			_ = e.store.UpdateRunBranch(pipelineID, childRun.BranchName)
		}
	}

	// Extract child artifacts and merge context variables
	childExec := childExecutor.LastExecution()
	if childExec != nil {
		if step.Config != nil {
			// Determine parent workspace for artifact extraction
			parentWorkspace := "."
			execution.mu.Lock()
			if ws, ok := execution.WorkspacePaths[step.ID]; ok && ws != "" {
				parentWorkspace = ws
			}
			execution.mu.Unlock()

			if len(step.Config.Extract) > 0 {
				if err := extractSubPipelineArtifacts(step.Config, childExec.Context, pipelineName, execution.Context, parentWorkspace); err != nil {
					return fmt.Errorf("failed to extract sub-pipeline artifacts: %w", err)
				}
			}

			// Evaluate stop condition if configured
			if step.Config.StopCondition != "" {
				if evaluateStopCondition(step.Config.StopCondition, childExec.Context) {
					execution.Context.SetCustomVariable("sub_pipeline_stop", "true")
				}
			}
		}

		// Merge child context variables into parent (last-writer-wins)
		execution.Context.MergeFrom(childExec.Context, pipelineName)

		// Propagate child execution-level artifact paths into the parent context
		// so that iterate steps can collect outputs from all child sub-pipelines.
		childExec.mu.Lock()
		childArtifacts := make(map[string]string, len(childExec.ArtifactPaths))
		for k, v := range childExec.ArtifactPaths {
			childArtifacts[k] = v
		}
		childExec.mu.Unlock()

		execution.mu.Lock()
		for key, path := range childArtifacts {
			// Namespaced key for iterate/aggregate: "audit-security.scan:findings"
			nsKey := pipelineName + "." + key
			execution.Context.SetArtifactPath(nsKey, path)

			// Register under composition step ID so injectArtifacts can find
			// "audit-security:findings" when a persona step references this
			// sub-pipeline's output. Only for bare sub-pipeline steps (not
			// iterate/aggregate which have their own output collection).
			if step.Iterate == nil && step.Aggregate == nil {
				if _, artName, ok := strings.Cut(key, ":"); ok {
					parentKey := step.ID + ":" + artName
					execution.ArtifactPaths[parentKey] = path
				}
			}
		}
		execution.mu.Unlock()
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("composition: sub-pipeline %q completed", pipelineName),
	})

	return nil
}

// executeIterateInDAG fans out over a JSON array, running a sub-pipeline per item.
func (e *DefaultPipelineExecutor) executeIterateInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the items array — may contain step output references like
	// {{ fetch-children.output.child_urls }} or pipeline context vars.
	itemsExpr := step.Iterate.Over
	itemsExpr = e.resolveStepOutputRef(itemsExpr, execution)
	if execution.Context != nil {
		itemsExpr = execution.Context.ResolvePlaceholders(itemsExpr)
	}

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(itemsExpr), &items); err != nil {
		return fmt.Errorf("iterate.over did not resolve to a JSON array: %w (raw: %s)", err, itemsExpr)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("iterate: %d items (mode: %s)", len(items), step.Iterate.Mode),
	})

	pipelineNameTmpl := step.SubPipeline

	if step.Iterate.Mode == "parallel" {
		return e.executeIterateParallelInDAG(ctx, execution, step, pipelineNameTmpl, items)
	}

	// Sequential iterate — track resolved pipeline names so we can collect outputs.
	resolvedNames := make([]string, 0, len(items))

	for i, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tmplCtx := NewTemplateContext(execution.Input, "")
		tmplCtx.Item = item

		// Resolve pipeline name (e.g. "{{ item }}" → "audit-security")
		resolvedName, err := ResolveTemplate(pipelineNameTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("iterate item %d: failed to resolve pipeline name: %w", i, err)
		}
		resolvedNames = append(resolvedNames, resolvedName)

		// Resolve input
		input := execution.Input
		if step.SubInput != "" {
			input, err = ResolveTemplate(step.SubInput, tmplCtx)
			if err != nil {
				return fmt.Errorf("iterate item %d: failed to resolve input: %w", i, err)
			}
		}

		e.emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     pipelineID,
			StepID:         step.ID,
			State:          event.StateIterationProgress,
			Message:        fmt.Sprintf("iterate item %d/%d: %s", i+1, len(items), resolvedName),
			TotalSteps:     len(items),
			CompletedSteps: i,
		})

		if err := e.runNamedSubPipeline(ctx, execution, step, resolvedName, input); err != nil {
			return fmt.Errorf("iterate item %d (%s): %w", i, resolvedName, err)
		}
	}

	// Collect outputs from all child sub-pipelines and register under the
	// iterate step's ID so downstream steps can reference {{ stepID.output }}.
	if err := e.collectIterateOutputs(execution, step, resolvedNames); err != nil {
		return fmt.Errorf("iterate: failed to collect outputs: %w", err)
	}

	e.emit(event.Event{
		Timestamp:      time.Now(),
		PipelineID:     pipelineID,
		StepID:         step.ID,
		State:          event.StateIterationCompleted,
		Message:        fmt.Sprintf("iterate: all %d items completed", len(items)),
		TotalSteps:     len(items),
		CompletedSteps: len(items),
	})

	return nil
}

func (e *DefaultPipelineExecutor) executeIterateParallelInDAG(ctx context.Context, execution *PipelineExecution, step *Step, pipelineNameTmpl string, items []json.RawMessage) error {
	pipelineID := execution.Status.ID

	maxConcurrent := step.Iterate.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = len(items)
	}

	// Pre-resolve all pipeline names so we can track them for output collection.
	resolvedNames := make([]string, len(items))
	resolvedInputs := make([]string, len(items))
	for i, item := range items {
		tmplCtx := NewTemplateContext(execution.Input, "")
		tmplCtx.Item = item

		name, err := ResolveTemplate(pipelineNameTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("iterate item %d: failed to resolve pipeline name: %w", i, err)
		}
		resolvedNames[i] = name

		input := execution.Input
		if step.SubInput != "" {
			input, err = ResolveTemplate(step.SubInput, tmplCtx)
			if err != nil {
				return fmt.Errorf("iterate item %d: failed to resolve input: %w", i, err)
			}
		}
		resolvedInputs[i] = input
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i := range items {
		resolvedName := resolvedNames[i]
		input := resolvedInputs[i]

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      event.StateRunning,
			Message:    fmt.Sprintf("iterate parallel item %d/%d: %s", i+1, len(items), resolvedName),
		})

		g.Go(func() error {
			return e.runNamedSubPipeline(gctx, execution, step, resolvedName, input)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Collect outputs from all child sub-pipelines and register under the
	// iterate step's ID so downstream steps can reference {{ stepID.output }}.
	if err := e.collectIterateOutputs(execution, step, resolvedNames); err != nil {
		return fmt.Errorf("iterate: failed to collect outputs: %w", err)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("iterate: all %d items completed (parallel)", len(items)),
	})

	return nil
}

// collectIterateOutputs scans the parent execution's context for artifacts
// merged from child sub-pipelines (via MergeFrom) and assembles them into a
// JSON array. The collected output is written to .wave/output/<stepID>-collected.json
// and registered in execution.ArtifactPaths so {{ stepID.output }} resolves to
// the combined result for downstream aggregate steps.
func (e *DefaultPipelineExecutor) collectIterateOutputs(execution *PipelineExecution, step *Step, resolvedNames []string) error {
	if execution.Context == nil {
		return nil
	}

	// For each child pipeline name, find the first artifact path that was
	// merged under its namespace (e.g. "audit-alpha.scan:output") and read
	// its content. Order matches the original items array.
	collected := make([]json.RawMessage, 0, len(resolvedNames))
	execution.Context.mu.Lock()
	artifactSnapshot := make(map[string]string, len(execution.Context.ArtifactPaths))
	for k, v := range execution.Context.ArtifactPaths {
		artifactSnapshot[k] = v
	}
	execution.Context.mu.Unlock()

	for _, name := range resolvedNames {
		prefix := name + "."
		var artPath string
		for key, path := range artifactSnapshot {
			if strings.HasPrefix(key, prefix) {
				artPath = path
				break
			}
		}

		if artPath == "" {
			// No artifact found for this child — include null placeholder
			// to keep the array aligned with the items array.
			collected = append(collected, json.RawMessage("null"))
			continue
		}

		data, err := os.ReadFile(artPath)
		if err != nil {
			collected = append(collected, json.RawMessage("null"))
			continue
		}

		// If the content is valid JSON, include it raw; otherwise wrap as a string.
		if json.Valid(data) {
			collected = append(collected, json.RawMessage(data))
		} else {
			quoted, _ := json.Marshal(string(data))
			collected = append(collected, json.RawMessage(quoted))
		}
	}

	arrayBytes, err := json.Marshal(collected)
	if err != nil {
		return fmt.Errorf("failed to marshal collected outputs: %w", err)
	}

	// Write to .wave/output/<stepID>-collected.json
	outputDir := filepath.Join(".wave", "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, step.ID+"-collected.json")
	if err := os.WriteFile(outputPath, arrayBytes, 0644); err != nil {
		return fmt.Errorf("failed to write collected output: %w", err)
	}

	// Register in execution.ArtifactPaths so resolveStepOutputRef can find it
	// via the standard "stepID:<artifactName>" key convention.
	execution.mu.Lock()
	execution.ArtifactPaths[step.ID+":collected-output"] = outputPath
	execution.mu.Unlock()

	return nil
}

// executeAggregateInDAG collects outputs from prior steps and writes them to a file.
func (e *DefaultPipelineExecutor) executeAggregateInDAG(_ context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the source expression — step output references like
	// {{ run-audits.output }} must be resolved before context placeholders.
	sourceExpr := step.Aggregate.From
	sourceExpr = e.resolveStepOutputRef(sourceExpr, execution)
	if execution.Context != nil {
		sourceExpr = execution.Context.ResolvePlaceholders(sourceExpr)
	}

	var result string
	var err error

	switch step.Aggregate.Strategy {
	case "concat":
		result = sourceExpr
	case "merge_arrays":
		result, err = mergeJSONArrays(sourceExpr, step.Aggregate.Key)
		if err != nil {
			return fmt.Errorf("merge_arrays failed: %w", err)
		}
	case "reduce":
		result = sourceExpr
	default:
		return fmt.Errorf("unknown aggregate strategy: %q", step.Aggregate.Strategy)
	}

	// Write result to file
	outputPath := step.Aggregate.Into
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write aggregate output: %w", err)
	}

	// Derive the artifact name from the output filename (without extension)
	// so inject_artifacts can find it via the "stepID:artifactName" key.
	artifactName := strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))

	// Register in execution.ArtifactPaths (used by injectArtifacts lookup)
	execution.mu.Lock()
	execution.ArtifactPaths[step.ID+":"+artifactName] = outputPath
	execution.mu.Unlock()

	// Also register in context for template resolution
	if execution.Context != nil {
		execution.Context.SetArtifactPath(artifactName, outputPath)
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("aggregate: wrote %s (strategy: %s)", outputPath, step.Aggregate.Strategy),
	})

	return nil
}

// executeBranchInDAG evaluates a condition and runs the matching pipeline.
func (e *DefaultPipelineExecutor) executeBranchInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the branch condition — may reference step outputs
	value := step.Branch.On
	value = e.resolveStepOutputRef(value, execution)
	if execution.Context != nil {
		value = execution.Context.ResolvePlaceholders(value)
	}

	pipelineName, ok := step.Branch.Cases[value]
	if !ok {
		pipelineName, ok = step.Branch.Cases["default"]
		if !ok {
			return fmt.Errorf("branch value %q has no matching case and no default", value)
		}
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("branch %q → %s", value, pipelineName),
	})

	if pipelineName == "skip" {
		execution.mu.Lock()
		execution.States[step.ID] = stateCompleted
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "skipped by branch")
		}
		return nil
	}

	input := e.resolveSubPipelineInput(execution, step)
	return e.runNamedSubPipeline(ctx, execution, step, pipelineName, input)
}

// executeLoopInDAG runs sub-steps/sub-pipelines repeatedly until a condition is
// met or max iterations reached.
func (e *DefaultPipelineExecutor) executeLoopInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	if step.Loop.MaxIterations <= 0 {
		return fmt.Errorf("loop step %q: max_iterations must be > 0", step.ID)
	}

	for i := 0; i < step.Loop.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      event.StateRunning,
			Message:    fmt.Sprintf("loop iteration %d/%d", i+1, step.Loop.MaxIterations),
		})

		// Execute sub-pipeline if specified at the loop step level
		if step.SubPipeline != "" {
			input := e.resolveSubPipelineInput(execution, step)
			if err := e.runNamedSubPipeline(ctx, execution, step, step.SubPipeline, input); err != nil {
				return fmt.Errorf("loop iteration %d: %w", i, err)
			}
		}

		// Execute loop sub-steps
		for j := range step.Loop.Steps {
			subStep := &step.Loop.Steps[j]
			if subStep.IsCompositionStep() {
				if err := e.executeCompositionStep(ctx, execution, subStep); err != nil {
					return fmt.Errorf("loop iteration %d, step %q: %w", i, subStep.ID, err)
				}
			}
		}

		// Check loop termination condition
		if step.Loop.Until != "" {
			condResult := step.Loop.Until
			if execution.Context != nil {
				condResult = execution.Context.ResolvePlaceholders(condResult)
			}
			condResult = strings.TrimSpace(condResult)
			if condResult == "true" || condResult == "done" || condResult == "yes" {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateCompleted,
					Message:    fmt.Sprintf("loop terminated: condition met at iteration %d", i+1),
				})
				execution.mu.Lock()
				execution.States[step.ID] = stateCompleted
				execution.mu.Unlock()
				if e.store != nil {
					_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "loop condition met")
				}
				return nil
			}
		}
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("loop completed: max iterations (%d) reached", step.Loop.MaxIterations),
	})

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "max iterations reached")
	}

	return nil
}

// executeGateInDAG handles gate steps within a DAG pipeline.
// It delegates to GateExecutor, stores the decision in PipelineContext,
// writes freeform text as an artifact, and returns routing information via error type.
func (e *DefaultPipelineExecutor) executeGateInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateRunning,
		Message:    fmt.Sprintf("gate step: %s", step.Gate.Type),
	})

	// Select the appropriate handler
	var handler GateHandler
	if e.autoApprove {
		handler = &AutoApproveHandler{}
	} else if e.gateHandler != nil {
		handler = e.gateHandler
	}
	// If no handler and gate has choices, fall back to CLI handler
	if handler == nil && len(step.Gate.Choices) > 0 {
		handler = &CLIGateHandler{}
	}

	gate := NewGateExecutorWithHandler(e.emitter, e.store, &execution.Manifest.Runtime.Timeouts, handler)

	// Annotate the gate config with the step ID so downstream handlers
	// (e.g. WebUI) can associate the pending gate with a specific step.
	step.Gate.RuntimeStepID = step.ID

	decision, err := gate.ExecuteWithDecision(ctx, step.Gate, nil)
	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = stateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	// Store the gate decision in the pipeline context for template resolution
	if decision != nil && execution.Context != nil {
		execution.Context.SetGateDecision(step.ID, decision)

		// Write freeform text as artifact if provided
		if decision.Text != "" {
			wsRoot := execution.Manifest.Runtime.WorkspaceRoot
			if wsRoot == "" {
				wsRoot = ".wave/workspaces"
			}
			artifactPath := filepath.Join(wsRoot, pipelineID, ".wave", "artifacts", fmt.Sprintf("gate-%s-text", step.ID))
			if mkErr := os.MkdirAll(filepath.Dir(artifactPath), 0755); mkErr != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStepProgress,
					Message:    fmt.Sprintf("warning: failed to create artifact directory for gate freeform text: %v", mkErr),
				})
			} else if writeErr := os.WriteFile(artifactPath, []byte(decision.Text), 0644); writeErr != nil {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStepProgress,
					Message:    fmt.Sprintf("warning: failed to write gate freeform text artifact: %v", writeErr),
				})
			}
		}
	}

	execution.mu.Lock()
	execution.States[step.ID] = stateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		_ = e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    "gate step completed",
	})

	// Handle choice routing
	if decision != nil && decision.Target == "_fail" {
		return &gateAbortError{StepID: step.ID, Choice: decision.Label}
	}

	// If the target is a step ID that has already been completed, re-queue it
	// (revision loop pattern). If the target is a pending/future step, just
	// continue normal DAG flow — the scheduler will reach it naturally.
	if decision != nil && decision.Target != "" && decision.Target != "_fail" {
		execution.mu.Lock()
		targetState := execution.States[decision.Target]
		execution.mu.Unlock()

		if targetState == stateCompleted || targetState == stateFailed || targetState == stateSkipped {
			return e.reQueueStep(execution, decision.Target)
		}
		// Target is pending/running — normal flow continues
	}

	return nil
}

// reQueueStep resets a step and all its dependents to pending state,
// allowing the main loop to re-execute them. This implements the revision loop pattern.
func (e *DefaultPipelineExecutor) reQueueStep(execution *PipelineExecution, targetStepID string) error {
	pipelineID := execution.Status.ID

	// Find the target step
	var targetStep *Step
	for i := range execution.Pipeline.Steps {
		if execution.Pipeline.Steps[i].ID == targetStepID {
			targetStep = &execution.Pipeline.Steps[i]
			break
		}
	}
	if targetStep == nil {
		return fmt.Errorf("gate routing: target step %q not found in pipeline", targetStepID)
	}

	// Reset the target step and all steps that depend on it (transitively)
	toReset := map[string]bool{targetStepID: true}
	changed := true
	for changed {
		changed = false
		for i := range execution.Pipeline.Steps {
			s := &execution.Pipeline.Steps[i]
			if toReset[s.ID] {
				continue
			}
			for _, dep := range s.Dependencies {
				if toReset[dep] {
					toReset[s.ID] = true
					changed = true
					break
				}
			}
		}
	}

	execution.mu.Lock()
	for stepID := range toReset {
		execution.States[stepID] = statePending
	}
	execution.mu.Unlock()

	for stepID := range toReset {
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, stepID, state.StatePending, "re-queued by gate routing")
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     stepID,
			State:      statePending,
			Message:    fmt.Sprintf("re-queued by gate routing to %q", targetStepID),
		})
	}

	return &reQueueError{TargetStepID: targetStepID, ResetSteps: toReset}
}

// reQueueError signals that steps have been re-queued via gate routing.
// The main executor loop should handle this by re-entering the scheduling loop.
type reQueueError struct {
	TargetStepID string
	ResetSteps   map[string]bool
}

func (e *reQueueError) Error() string {
	return fmt.Sprintf("gate routing: re-queued step %q and %d dependents", e.TargetStepID, len(e.ResetSteps)-1)
}

// cleanupCompletedPipeline removes a completed or failed pipeline from in-memory storage
// to prevent memory leaks. This is safe to call because completed pipeline status
// can be retrieved from persistent storage via GetStatus.
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
func (e *DefaultPipelineExecutor) fireWebhooks(ctx context.Context, evt hooks.HookEvent) {
	if e.webhookRunner != nil {
		e.webhookRunner.FireWebhooks(ctx, evt)
	}
}

// webhookStoreAdapter bridges the hooks.WebhookStore interface to the state store,
// avoiding a direct state→hooks import cycle.
type webhookStoreAdapter struct {
	store state.StateStore
}

func (a *webhookStoreAdapter) RecordWebhookDeliveryResult(d *hooks.WebhookDeliveryRecord) error {
	return a.store.RecordWebhookDelivery(&state.WebhookDelivery{
		WebhookID:      d.WebhookID,
		RunID:          d.RunID,
		Event:          d.Event,
		StatusCode:     d.StatusCode,
		ResponseTimeMs: d.ResponseTimeMs,
		Error:          d.Error,
	})
}

// resolveWorkspaceStepRefs resolves {{ steps.<step-id>.artifacts.<artifact-name>.<json-path> }}
// and {{ steps.<step-id>.output.<field> }} references in a workspace config field.
// This is called just before workspace creation so that branch/base fields can reference
// outputs from prior steps (e.g. a PR's headRefName fetched by a preceding step).
//
// Supported patterns:
//   - {{ steps.STEP_ID.artifacts.ARTIFACT_NAME.json.path }} — read a JSON field from a named artifact
//   - {{ steps.STEP_ID.output.json.path }} — read a JSON field from the first artifact of the step
//
// Returns an error if a referenced step/artifact does not exist or the JSON path fails.
func (e *DefaultPipelineExecutor) resolveWorkspaceStepRefs(tmpl string, execution *PipelineExecution) (string, error) {
	var resolveErr error

	result := templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		if resolveErr != nil {
			return match
		}

		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Only handle {{ steps.* }} references here.
		if !strings.HasPrefix(expr, "steps.") {
			return match
		}

		// steps.STEP_ID.artifacts.ARTIFACT_NAME[.JSON_PATH]
		// steps.STEP_ID.output[.JSON_PATH]
		rest := expr[len("steps."):]
		parts := strings.SplitN(rest, ".", 3) // [STEP_ID, "artifacts"|"output", rest]
		if len(parts) < 2 {
			resolveErr = fmt.Errorf("workspace template %q: expected steps.<step-id>.artifacts.<name> or steps.<step-id>.output.<field>", match)
			return match
		}

		stepID := parts[0]
		segment := parts[1]

		execution.mu.Lock()
		artifactsCopy := make(map[string]string, len(execution.ArtifactPaths))
		for k, v := range execution.ArtifactPaths {
			artifactsCopy[k] = v
		}
		execution.mu.Unlock()

		switch segment {
		case "artifacts":
			// steps.STEP_ID.artifacts.ARTIFACT_NAME[.JSON_PATH]
			if len(parts) < 3 {
				resolveErr = fmt.Errorf("workspace template %q: missing artifact name after 'artifacts'", match)
				return match
			}
			// parts[2] = "ARTIFACT_NAME" or "ARTIFACT_NAME.json.path"
			artAndPath := parts[2]
			dotIdx := strings.Index(artAndPath, ".")
			var artifactName, jsonPath string
			if dotIdx == -1 {
				artifactName = artAndPath
				jsonPath = ""
			} else {
				artifactName = artAndPath[:dotIdx]
				jsonPath = artAndPath[dotIdx+1:]
			}

			key := stepID + ":" + artifactName
			artPath, ok := artifactsCopy[key]
			if !ok {
				resolveErr = fmt.Errorf("workspace template %q: artifact %q from step %q not found (step may not have completed yet)", match, artifactName, stepID)
				return match
			}

			data, err := os.ReadFile(artPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: failed to read artifact %q: %w", match, artPath, err)
				return match
			}

			if jsonPath == "" {
				return strings.TrimSpace(string(data))
			}

			val, err := ExtractJSONPath(data, "."+jsonPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: JSON path %q in artifact %q: %w", match, jsonPath, artifactName, err)
				return match
			}
			return val

		case "output":
			// steps.STEP_ID.output[.JSON_PATH]
			// Find the first artifact for this step.
			var artPath string
			for k, v := range artifactsCopy {
				if strings.HasPrefix(k, stepID+":") {
					artPath = v
					break
				}
			}
			if artPath == "" {
				resolveErr = fmt.Errorf("workspace template %q: no output found for step %q (step may not have completed yet)", match, stepID)
				return match
			}

			data, err := os.ReadFile(artPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: failed to read output for step %q: %w", match, stepID, err)
				return match
			}

			if len(parts) < 3 {
				return strings.TrimSpace(string(data))
			}

			jsonPath := parts[2]
			val, err := ExtractJSONPath(data, "."+jsonPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: JSON path %q in step %q output: %w", match, jsonPath, stepID, err)
				return match
			}
			return val

		default:
			resolveErr = fmt.Errorf("workspace template %q: unknown segment %q (expected 'artifacts' or 'output')", match, segment)
			return match
		}
	})

	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}

// resolveStepOutputRef resolves {{ stepID.output }} and {{ stepID.output.field }}
// references in a template string by reading the step's artifact file from disk.
// This bridges composition steps (which use TemplateContext-style references) with
// the DAG executor (which stores artifacts in execution.ArtifactPaths).
func (e *DefaultPipelineExecutor) resolveStepOutputRef(tmpl string, execution *PipelineExecution) string {
	return templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Must be stepID.output or stepID.output.field
		parts := strings.SplitN(expr, ".", 3)
		if len(parts) < 2 || parts[1] != "output" {
			return match // not a step output ref
		}

		stepID := parts[0]

		// Find the artifact file for this step
		execution.mu.Lock()
		var artPath string
		for key, path := range execution.ArtifactPaths {
			if strings.HasPrefix(key, stepID+":") {
				artPath = path
				break
			}
		}
		execution.mu.Unlock()

		if artPath == "" {
			return match // no artifact found
		}

		data, err := os.ReadFile(artPath)
		if err != nil {
			return match
		}

		if len(parts) == 2 {
			// {{ stepID.output }} → full file content
			return string(data)
		}

		// {{ stepID.output.field }} → extract JSON field
		field := parts[2]
		val, err := ExtractJSONPath(data, "."+field)
		if err != nil {
			return match
		}
		return val
	})
}

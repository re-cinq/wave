package pipeline

import (
	"context"
	"errors"
	"encoding/json"
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
	"github.com/recinq/wave/internal/cost"
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/scope"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/retro"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/worktree"
	"github.com/recinq/wave/internal/workspace"
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
	runner         adapter.AdapterRunner  // Deprecated: use registry for per-step resolution
	registry       *adapter.AdapterRegistry
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
	// Model override (from CLI --model flag)
	modelOverride string
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

// WithHookRunner sets the lifecycle hook runner for pipeline events.
func WithHookRunner(r hooks.HookRunner) ExecutorOption {
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
	mu              sync.Mutex                 // protects map writes during concurrent steps
	Pipeline        *Pipeline
	Manifest        *manifest.Manifest
	States          map[string]string
	Results         map[string]map[string]interface{}
	ArtifactPaths   map[string]string          // "stepID:artifactName" -> filesystem path
	WorkspacePaths  map[string]string          // stepID -> workspace path
	WorktreePaths   map[string]*WorktreeInfo   // resolved branch -> worktree info
	Input           string
	Status          *PipelineStatus
	Context         *PipelineContext  // Dynamic template variables
	AttemptContexts    map[string]*AttemptContext  // stepID -> current retry context (nil on first attempt)
	ReworkTransitions  map[string]string           // failedStepID -> reworkStepID (for resume support)
	ThreadManager      *ThreadManager              // Thread conversation continuity manager
	CircuitBreaker     *CircuitBreaker             // Failure fingerprint tracking for circuit breaking
	Watchdog           *StallWatchdog              // Current step's stall watchdog (set during step execution)
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
		runner:                 e.runner,
		registry:               e.registry,
		emitter:                e.emitter,
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
	if IsGraphPipeline(p) {
		return e.executeGraphPipeline(ctx, p, m, input)
	}

	validator := &DAGValidator{}
	if err := validator.ValidateDAG(p); err != nil {
		return fmt.Errorf("invalid pipeline DAG: %w", err)
	}

	// Validate thread/fidelity fields
	if errs := ValidateThreadFields(p); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, err := range errs {
			msgs[i] = err.Error()
		}
		return fmt.Errorf("thread validation failed:\n  %s", strings.Join(msgs, "\n  "))
	}

	sortedSteps, err := validator.TopologicalSort(p)
	if err != nil {
		return fmt.Errorf("failed to topologically sort steps: %w", err)
	}

	// Resolve named retry policies into concrete values before execution
	if err := ResolvePipelineRetryPolicies(p); err != nil {
		return fmt.Errorf("retry policy resolution: %w", err)
	}

	// Apply step filter (--steps / --exclude) to the sorted step list
	if e.stepFilter != nil && e.stepFilter.IsActive() {
		if err := e.stepFilter.Validate(p); err != nil {
			return err
		}
		sortedSteps = e.stepFilter.Apply(sortedSteps)
		if len(sortedSteps) == 0 {
			return fmt.Errorf("step filter produced no runnable steps")
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

	// Inject forge variables for unified pipeline template resolution
	forgeInfo, _ := forge.DetectFromGitRemotes()
	InjectForgeVariables(pipelineContext, forgeInfo)

	// Resolve template placeholders in pipeline skills before validation.
	// Skills like "{{ project.skill }}" must be resolved to their actual values
	// (or empty string) before ValidateSkillRefs checks them against the store.
	// Build a new slice (do NOT mutate p.Skills — the Pipeline may be shared
	// across concurrent matrix child executors).
	resolvedPipelineSkills := make([]string, 0, len(p.Skills))
	for _, s := range p.Skills {
		r := pipelineContext.ResolvePlaceholders(s)
		if r != "" {
			resolvedPipelineSkills = append(resolvedPipelineSkills, r)
		}
	}

	// Validate skill references at manifest and pipeline scopes (after template resolution)
	if errs := e.validateSkillRefs(resolvedPipelineSkills, p.Metadata.Name, m); len(errs) > 0 {
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
				resolved := pipelineContext.ResolvePlaceholders(tool)
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
	if forgeInfo.Type == forge.ForgeLocal {
		// Check pipeline name for forge prefix
		if ferr := preflight.CheckForgePipelineName(forgeInfo, p.Metadata.Name); ferr != nil {
			e.emit(event.Event{
				Timestamp: time.Now(),
				State:     "preflight",
				Message:   ferr.Error(),
			})
			return ferr
		}

		// Build step inputs for forge dependency scanning
		forgeStepInputs := make([]preflight.ForgeStepInput, 0, len(sortedSteps))
		for _, step := range sortedSteps {
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
	if forgeInfo.Type != forge.ForgeUnknown {
		resolver := scope.NewResolver(forgeInfo.Type)
		introspector := scope.NewIntrospector(forgeInfo.Type)
		scopeValidator := scope.NewValidator(resolver, introspector, forgeInfo, m.Runtime.Sandbox.EnvPassthrough)

		// Build persona scope map from manifest personas used in this pipeline
		personaScopes := make(map[string][]string)
		for _, step := range sortedSteps {
			resolvedName := pipelineContext.ResolvePlaceholders(step.Persona)
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
	if e.store != nil && pipelineID != "" {
		var pollCancel context.CancelFunc
		ctx, pollCancel = context.WithCancel(ctx)
		defer pollCancel()
		go e.pollCancellation(ctx, pipelineID, pollCancel)
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
		Pipeline:        p,
		Manifest:        m,
		States:          make(map[string]string),
		Results:         make(map[string]map[string]interface{}),
		ArtifactPaths:   make(map[string]string),
		WorkspacePaths:  make(map[string]string),
		WorktreePaths:   make(map[string]*WorktreeInfo),
		AttemptContexts:   make(map[string]*AttemptContext),
		ReworkTransitions: make(map[string]string),
		ThreadManager:     NewThreadManager(threadCompactionAdapter),
		CircuitBreaker:    NewCircuitBreaker(m.Runtime.CircuitBreaker.Limit, m.Runtime.CircuitBreaker.TrackedClasses),
		Input:             input,
		Context:         pipelineContext,
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
				return fmt.Errorf("failed to initialize hook runner: %w", err)
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
			return fmt.Errorf("deadlock: no steps ready but %d remain", len(sortedSteps)-completedCount)
		}

		if err := e.executeStepBatch(ctx, execution, ready); err != nil {
			// ReQueueError means gate routing reset steps to pending — re-enter the scheduling loop
			var reQueueErr *ReQueueError
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
					if stepState == StateCompleted && !completed[step.ID] {
						completed[step.ID] = true
						completedCount++
					}
				}
				continue
			}
			execution.Status.State = StateFailed
			// Identify which step(s) failed from the batch
			var failedStepID string
			for _, step := range ready {
				execution.mu.Lock()
				stepState := execution.States[step.ID]
				execution.mu.Unlock()
				if stepState == StateFailed || stepState == StateRunning || stepState == StateRetrying {
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
				e.store.SavePipelineState(pipelineID, StateFailed, input)
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     failedStepID,
				State:      StateFailed,
				Message:    err.Error(),
			})
			// Generate retrospective for failed runs — these are the most valuable
			if e.retroGenerator != nil {
				e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
			}
			e.cleanupCompletedPipeline(pipelineID)
			return &StepError{StepID: failedStepID, Err: err}
		}

		// Process batch results: steps may have completed, failed (optional), or been skipped
		for _, step := range ready {
			completed[step.ID] = true
			completedCount++

			execution.mu.Lock()
			stepState := execution.States[step.ID]
			execution.mu.Unlock()

			if stepState == StateFailed || stepState == StateSkipped {
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
			if stepState == StateCompleted || stepState == StateFailed {
				completed[step.ID] = true
			}
		}

		// Skip steps whose dependencies include failed/skipped steps (transitive propagation)
		e.skipDependentSteps(execution, sortedSteps, completed, &completedCount)

		e.emit(event.Event{
			Timestamp:      time.Now(),
			PipelineID:     pipelineID,
			State:          StateRunning,
			TotalSteps:     schedulableSteps,
			CompletedSteps: completedCount,
			Progress:       (completedCount * 100) / schedulableSteps,
			Message:        fmt.Sprintf("%d/%d steps completed", completedCount, schedulableSteps),
		})
	}

	now := time.Now()
	execution.Status.CompletedAt = &now

	// Pipeline succeeds if no required steps failed
	if e.hasRequiredFailures(execution) {
		execution.Status.State = StateFailed
		if e.store != nil {
			e.store.SavePipelineState(pipelineID, StateFailed, input)
		}
		// Run run_failed hooks with detached context (non-blocking by default).
		e.runTerminalHooks(hooks.HookEvent{
			Type:       hooks.EventRunFailed,
			PipelineID: pipelineID,
			Input:      input,
		})
	} else {
		execution.Status.State = StateCompleted
		if e.store != nil {
			e.store.SavePipelineState(pipelineID, StateCompleted, input)
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
		State:      StateCompleted,
		DurationMs: elapsed,
		Message:    fmt.Sprintf("%d steps completed", schedulableSteps),
	})

	// Generate retrospective (non-blocking)
	if e.retroGenerator != nil {
		e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
	}

	// Clean up completed pipeline from in-memory storage to prevent memory leak
	e.cleanupCompletedPipeline(pipelineID)

	return nil
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

	// Inject forge variables
	forgeInfo, _ := forge.DetectFromGitRemotes()
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
		e.store.SavePipelineState(pipelineID, StateRunning, input)
	}

	execution.Status.State = StateRunning

	e.emit(event.Event{
		Timestamp:      time.Now(),
		PipelineID:     pipelineID,
		State:          "started",
		Message:        fmt.Sprintf("graph-mode pipeline input=%q steps=%d", input, len(p.Steps)),
		TotalSteps:     len(p.Steps),
		CompletedSteps: 0,
	})

	// Initialize hook runner from manifest + pipeline hooks if not already set.
	if e.hookRunner == nil {
		merged := append([]hooks.LifecycleHookDef{}, m.Hooks...)
		merged = append(merged, p.Hooks...)
		if len(merged) > 0 {
			runner, err := hooks.NewHookRunner(merged, e.emitter)
			if err != nil {
				return fmt.Errorf("failed to initialize hook runner: %w", err)
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
		os.RemoveAll(pipelineWsPath)
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
			return e.executeCommandStep(ctx, execution, step)
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
		execution.Status.State = StateFailed
		if e.store != nil {
			e.store.SavePipelineState(pipelineID, StateFailed, input)
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			State:      StateFailed,
			Message:    err.Error(),
		})
		// Generate retrospective for failed runs — these are the most valuable
		if e.retroGenerator != nil {
			e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
		}
		e.cleanupCompletedPipeline(pipelineID)
		return err
	}

	execution.Status.State = StateCompleted
	if e.store != nil {
		e.store.SavePipelineState(pipelineID, StateCompleted, input)
	}

	elapsed := time.Since(execution.Status.StartedAt).Milliseconds()
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      StateCompleted,
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
	execution.States[step.ID] = StateRunning
	execution.Status.CurrentStep = step.ID
	execution.mu.Unlock()

	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateRunning, "")
	}

	// Audit log: command step start
	if e.logger != nil {
		e.logger.LogStepStart(pipelineID, step.ID, "command", nil)
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      StateRunning,
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
				State:      StateRunning,
				Message:    fmt.Sprintf("command script sanitized (risk_score=%d, rules=%v)", record.RiskScore, record.SanitizationRules),
			})
		}
		script = sanitized
	}

	// Create workspace for the step
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
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
		e.logger.LogToolCall(pipelineID, step.ID, "sh", script)
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
		execution.States[step.ID] = StateFailed
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, execErr.Error())
		}

		// Audit log: step end with failure
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		if e.logger != nil {
			e.logger.LogStepEnd(pipelineID, step.ID, StateFailed, duration, exitCode, len(stdout.String()), 0, execErr.Error())
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      StateFailed,
			Message:    fmt.Sprintf("command failed: %v\nstderr: %s", execErr, stderr.String()),
		})

		return result, execErr
	}

	result.Outcome = "success"

	execution.mu.Lock()
	execution.States[step.ID] = StateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
	}

	// Audit log: step end with success
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if e.logger != nil {
		e.logger.LogStepEnd(pipelineID, step.ID, StateCompleted, duration, exitCode, len(stdout.String()), 0, "")
	}

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      StateCompleted,
		Message:    "command completed successfully",
	})

	return result, nil
}

// filterEnvPassthrough builds a minimal environment containing only the
// variables named in the passthrough list. This prevents command steps from
// inheriting the full parent environment which may contain secrets.
// PATH is always included to ensure basic command resolution works.
func filterEnvPassthrough(passthrough []string) []string {
	allowed := make(map[string]bool, len(passthrough)+1)
	allowed["PATH"] = true
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
				if depState == StateFailed || depState == StateSkipped {
					hasFailedDep = true
				}
			}
			if allDepsComplete && hasFailedDep {
				execution.mu.Lock()
				execution.States[step.ID] = StateSkipped
				execution.mu.Unlock()
				if e.store != nil {
					e.store.SaveStepState(pipelineID, step.ID, state.StateSkipped, "dependency failed")
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
		stepOptional[execution.Pipeline.Steps[i].ID] = execution.Pipeline.Steps[i].IsOptional()
	}

	execution.mu.Lock()
	defer execution.mu.Unlock()
	for stepID, stepState := range execution.States {
		if stepState == StateFailed {
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
	execution.States[step.ID] = StateRunning
	execution.Status.CurrentStep = step.ID
	execution.mu.Unlock()

	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateRunning, "")
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
		return nil
	}

	// Run step_start hooks
	if e.hookRunner != nil {
		evt := hooks.HookEvent{
			Type:       hooks.EventStepStart,
			PipelineID: pipelineID,
			StepID:     step.ID,
			Input:      execution.Input,
		}
		if _, err := e.hookRunner.RunHooks(ctx, evt); err != nil {
			execution.mu.Lock()
			execution.States[step.ID] = StateFailed
			execution.mu.Unlock()
			if e.store != nil {
				e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
			}
			return fmt.Errorf("step_start hook failed: %w", err)
		}
	}

	maxAttempts := step.Retry.EffectiveMaxAttempts()

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Don't retry if the parent context is already cancelled
			if ctx.Err() != nil {
				return fmt.Errorf("context cancelled, skipping retry: %w", lastErr)
			}
			execution.mu.Lock()
			execution.States[step.ID] = StateRetrying
			execution.mu.Unlock()
			if e.store != nil {
				e.store.SaveStepState(pipelineID, step.ID, state.StateRetrying, "")
			}
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      StateRetrying,
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
			e.store.RecordStepAttempt(&state.StepAttemptRecord{
				RunID:     pipelineID,
				StepID:    step.ID,
				Attempt:   attempt,
				State:     StateRunning,
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
				e.store.RecordStepAttempt(&state.StepAttemptRecord{
					RunID:        pipelineID,
					StepID:       step.ID,
					Attempt:      attempt,
					State:        StateFailed,
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
				// Set up attempt context for prompt adaptation on next retry
				if step.Retry.AdaptPrompt {
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

					execution.mu.Lock()
					execution.AttemptContexts[step.ID] = &AttemptContext{
						Attempt:      attempt + 1,
						MaxAttempts:  maxAttempts,
						PriorError:   errMsg,
						FailureClass: failureClass,
						PriorStdout:  stdoutTail,
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
				if step.IsOptional() {
					onFailure = OnFailureContinue
				} else {
					onFailure = OnFailureFail
				}
			}

			switch onFailure {
			case OnFailureSkip:
				execution.mu.Lock()
				execution.States[step.ID] = StateSkipped
				execution.mu.Unlock()
				if e.store != nil {
					e.store.SaveStepState(pipelineID, step.ID, state.StateSkipped, err.Error())
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
				execution.States[step.ID] = StateFailed
				execution.mu.Unlock()
				if e.store != nil {
					e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
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
				execution.States[step.ID] = StateFailed
				execution.mu.Unlock()
				if e.store != nil {
					e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
				}
				// Run step_failed hooks (non-blocking by default)
				if e.hookRunner != nil {
					e.hookRunner.RunHooks(ctx, hooks.HookEvent{
						Type:       hooks.EventStepFailed,
						PipelineID: pipelineID,
						StepID:     step.ID,
						Input:      execution.Input,
						Error:      lastErr.Error(),
					})
				}
				e.recordOntologyUsage(execution, step, "failed")
				return lastErr
			}
		}

		// Record successful attempt
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordStepAttempt(&state.StepAttemptRecord{
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
		execution.States[step.ID] = StateCompleted
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
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
		os.Remove(".wave/.ontology-stale")
		return "ontology may be stale (post-merge changes detected) — run 'wave analyze' to refresh"
	}

	// Compare wave.yaml mtime against latest commit time
	manifestStat, err := os.Stat("wave.yaml")
	if err != nil {
		return ""
	}

	out, err := exec.Command("git", "log", "-1", "--format=%cI").Output()
	if err != nil {
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

// recordOntologyUsage records ontology context usage for a step after its final status is determined.
// This enables decision lineage tracking across pipeline runs.
func (e *DefaultPipelineExecutor) recordOntologyUsage(execution *PipelineExecution, step *Step, stepStatus string) {
	if e.store == nil || execution.Manifest.Ontology == nil || len(execution.Manifest.Ontology.Contexts) == 0 {
		return
	}

	// Determine which contexts were injected: if step declares contexts, only those;
	// otherwise all contexts are injected (RenderMarkdown with empty filter passes all).
	// Only record targeted (explicitly declared) contexts — bulk injection inflates stats.
	injectedContexts := step.Contexts
	if len(injectedContexts) == 0 {
		// Step didn't declare specific contexts — skip ontology tracking.
		// All contexts are injected by default, but recording all of them
		// makes every context show identical stats, which is meaningless.
		return
	}

	// Determine contract pass/fail: nil if no contract, true/false based on step outcome
	var contractPassed *bool
	if step.Handover.Contract.Type != "" {
		passed := stepStatus == "success"
		contractPassed = &passed
	}

	for _, ctxName := range injectedContexts {
		invariantCount := 0
		for _, ctx := range execution.Manifest.Ontology.Contexts {
			if ctx.Name == ctxName {
				invariantCount = len(ctx.Invariants)
				break
			}
		}
		if err := e.store.RecordOntologyUsage(
			execution.Status.ID, step.ID, ctxName,
			invariantCount, stepStatus, contractPassed,
		); err != nil {
			if e.logger != nil {
				_ = e.logger.LogToolCall(execution.Status.ID, step.ID, "recordOntologyUsage",
					fmt.Sprintf("context=%s err=%v", ctxName, err))
			}
		} else {
			if e.logger != nil {
				_ = e.logger.LogOntologyLineage(execution.Status.ID, step.ID, ctxName, stepStatus, invariantCount)
			}
			e.emit(event.Event{
				PipelineID: execution.Status.ID,
				StepID:     step.ID,
				State:      event.StateOntologyLineage,
				Message:    fmt.Sprintf("context=%s status=%s invariants=%d", ctxName, stepStatus, invariantCount),
				Timestamp:  time.Now(),
			})
		}
	}
}

// executeReworkStep handles on_failure=rework: marks the failed step, builds failure context,
// executes the rework target step, and re-registers its artifacts under the original step's ID.
func (e *DefaultPipelineExecutor) executeReworkStep(ctx context.Context, execution *PipelineExecution, failedStep *Step, failErr error, failDuration time.Duration) error {
	pipelineID := execution.Status.ID
	reworkStepID := failedStep.Retry.ReworkStep

	// Mark the failed step
	execution.mu.Lock()
	execution.States[failedStep.ID] = StateFailed
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
		execution.States[reworkStep.ID] = StateFailed
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
	execution.States[reworkStep.ID] = StateCompleted
	// Mark the failed step as completed so downstream steps are not skipped
	execution.States[failedStep.ID] = StateCompleted
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

// executeMatrixStep handles steps with matrix strategy using fan-out execution.
func (e *DefaultPipelineExecutor) executeMatrixStep(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	matrixExecutor := NewMatrixExecutor(e)
	err := matrixExecutor.Execute(ctx, execution, step)

	if err != nil {
		execution.mu.Lock()
		execution.States[step.ID] = StateFailed
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.mu.Lock()
	execution.States[step.ID] = StateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
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
		execution.States[step.ID] = StateFailed
		execution.mu.Unlock()
		if e.store != nil {
			_ = e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return err
	}

	execution.mu.Lock()
	execution.States[step.ID] = StateCompleted
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

func (e *DefaultPipelineExecutor) runStepExecution(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	resolvedPersona := step.Persona
	if execution.Context != nil {
		resolvedPersona = execution.Context.ResolvePlaceholders(step.Persona)
	}
	persona := execution.Manifest.GetPersona(resolvedPersona)
	if persona == nil {
		return fmt.Errorf("persona %q not found in manifest", resolvedPersona)
	}

	// Resolve adapter name: step.Adapter > persona.Adapter (3-tier precedence)
	resolvedAdapterName := persona.Adapter
	if step.Adapter != "" {
		resolvedAdapterName = step.Adapter
	}

	adapterDef := execution.Manifest.GetAdapter(resolvedAdapterName)
	if adapterDef == nil {
		return fmt.Errorf("adapter %q not found in manifest", resolvedAdapterName)
	}

	// Resolve adapter runner from registry for per-step dispatch
	stepRunner := e.registry.ResolveWithFallback(resolvedAdapterName)

	// Create workspace under .wave/workspaces/<pipeline>/<step>/
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
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
			return fmt.Errorf("failed to create output dir: %w", err)
		}
	}

	resolvedModel := e.resolveModel(step, persona, &execution.Manifest.Runtime.Routing, resolvedPersona)
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         StateRunning,
		Persona:       resolvedPersona,
		Message:       fmt.Sprintf("Starting %s persona in %s", resolvedPersona, workspacePath),
		CurrentAction: "Initializing",
		Model:            resolvedModel,
		Adapter:       adapterDef.Binary,
		Temperature:   persona.Temperature,
	})

	// Record model routing decision
	{
		rationale := "adapter default (no override)"
		if e.modelOverride != "" {
			rationale = "CLI --model flag override"
		} else if step.Model != "" {
			rationale = "per-step model pinning in pipeline YAML"
		} else if persona.Model != "" {
			rationale = "per-persona model configuration"
		} else if execution.Manifest.Runtime.Routing.AutoRoute {
			rationale = "auto-routed based on step complexity"
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
				"adapter": adapterDef.Binary,
			},
		)
	}

	// Inject artifacts from dependencies
	artifactInjectStart := time.Now()
	if err := e.injectArtifacts(execution, step, workspacePath); err != nil {
		return fmt.Errorf("failed to inject artifacts: %w", err)
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
		_ = e.logger.LogStepStart(pipelineID, step.ID, resolvedPersona, artifactNames)
	}

	prompt := e.buildStepPrompt(execution, step)

	if e.logger != nil {
		_ = e.logger.LogToolCall(pipelineID, step.ID, "adapter.Run", fmt.Sprintf("persona=%s prompt_len=%d", resolvedPersona, len(prompt)))
	}

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
	if persona.SystemPromptFile != "" {
		if data, err := os.ReadFile(persona.SystemPromptFile); err == nil {
			systemPrompt = string(data)
		}
	}


	// Resolve sandbox config using the new backend-aware resolution
	sandboxBackend := execution.Manifest.Runtime.Sandbox.ResolveBackend()
	sandboxEnabled := sandboxBackend != "none"
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

	// Resolve skills from all three scopes: global, persona, pipeline
	// Pipeline scope includes both pipeline.Skills and requires.skills keys.
	// Template-resolve pipeline skills (e.g. "{{ project.skill }}") before merging.
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
	resolvedSkills := skill.ResolveSkills(execution.Manifest.Skills, persona.Skills, pipelineSkills)

	// Provision skill commands from requires.skills (SkillConfig-backed skills)
	var skillCommandsDir string
	if execution.Pipeline.Requires != nil && len(execution.Pipeline.Requires.Skills) > 0 {
		skillNames := execution.Pipeline.Requires.SkillNames()
		provisioner := skill.NewProvisioner(execution.Pipeline.Requires.Skills, "")
		commands, _ := provisioner.DiscoverCommands(skillNames)
		if len(commands) > 0 {
			tmpDir := filepath.Join(workspacePath, ".wave-skill-commands")
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

	// Provision DirectoryStore skills (name-only references not in requires.skills)
	var resolvedSkillRefs []adapter.SkillRef
	if e.skillStore != nil && len(resolvedSkills) > 0 {
		requiresSkills := make(map[string]bool)
		if execution.Pipeline.Requires != nil {
			for name := range execution.Pipeline.Requires.Skills {
				requiresSkills[name] = true
			}
		}
		// Filter out skills already handled by requires.skills
		var storeSkills []string
		for _, name := range resolvedSkills {
			if !requiresSkills[name] {
				storeSkills = append(storeSkills, name)
			}
		}
		if len(storeSkills) > 0 {
			infos, err := skill.ProvisionFromStore(e.skillStore, workspacePath, storeSkills)
			if err != nil {
				return fmt.Errorf("skill provisioning failed: %w", err)
			}
			for _, info := range infos {
				resolvedSkillRefs = append(resolvedSkillRefs, adapter.SkillRef{
					Name:        info.Name,
					Description: info.Description,
				})
			}
		}
	}

	// Auto-generate contract compliance section. This is appended directly
	// to the user prompt (not the system prompt / agent .md) so the model
	// sees it alongside the task instructions where it has the strongest
	// signal. System prompt injection was unreliable — models would ignore
	// the schema buried in the agent .md body among other context.
	contractPrompt := e.buildContractPrompt(step, execution.Context)
	if contractPrompt != "" {
		prompt = prompt + "\n\n" + contractPrompt
	}

	// Build ontology section from manifest for CLAUDE.md injection
	ontologySection := ""
	if execution.Manifest.Ontology != nil {
		ontologySection = execution.Manifest.Ontology.RenderMarkdown(step.Contexts)
		if ontologySection != "" {
			injected := step.Contexts
			if len(injected) == 0 {
				for _, ctx := range execution.Manifest.Ontology.Contexts {
					injected = append(injected, ctx.Name)
				}
			}
			totalInvariants := 0
			for _, ctx := range execution.Manifest.Ontology.Contexts {
				for _, name := range injected {
					if ctx.Name == name {
						totalInvariants += len(ctx.Invariants)
						break
					}
				}
			}
			if e.logger != nil {
				_ = e.logger.LogOntologyInject(pipelineID, step.ID, injected, totalInvariants)
			}
			e.emit(event.Event{
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      event.StateOntologyInject,
				Message:    fmt.Sprintf("contexts=[%s] invariants=%d", strings.Join(injected, ","), totalInvariants),
				Timestamp:  time.Now(),
			})
		}
	}

	cfg := adapter.AdapterRunConfig{
		Adapter:          adapterDef.Binary,
		Persona:          resolvedPersona,
		WorkspacePath:    workspacePath,
		Prompt:           prompt,
		SystemPrompt:     systemPrompt,
		Timeout:          timeout,
		Temperature:      persona.Temperature,
		Model:            resolvedModel,
		AllowedTools:     persona.Permissions.AllowedTools,
		DenyTools:        persona.Permissions.Deny,
		OutputFormat:     adapterDef.OutputFormat,
		Debug:            e.debug,
		SandboxEnabled:   sandboxEnabled,
		AllowedDomains:   sandboxDomains,
		EnvPassthrough:   envPassthrough,
		SandboxBackend:   sandboxBackend,
		DockerImage:      execution.Manifest.Runtime.Sandbox.GetDockerImage(),
		SkillCommandsDir:    skillCommandsDir,
		ResolvedSkills:      resolvedSkillRefs,
		OntologySection:     ontologySection,
		ContractPrompt:      "", // Contract now in user prompt, not system prompt
		MaxConcurrentAgents: step.MaxConcurrentAgents,
		OnStreamEvent: func(evt adapter.StreamEvent) {
			if evt.Type == "tool_use" && evt.ToolName != "" {
				// Notify watchdog of activity to prevent stall timeout
				execution.mu.Lock()
				wd := execution.Watchdog
				execution.mu.Unlock()
				if wd != nil {
					wd.NotifyActivity()
				}

				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      event.StateStreamActivity,
					Persona:    resolvedPersona,
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
		Persona:       resolvedPersona,
		Progress:      25,
		CurrentAction: "Executing agent",
	})

	// Iron Rule: estimate prompt size and check against context window
	promptBytes := len(cfg.Prompt) + len(cfg.OntologySection) + len(cfg.ContractPrompt)
	if promptBytes > 0 && resolvedModel != "" {
		ironStatus, ironMsg := cost.CheckIronRule(resolvedModel, promptBytes)
		switch ironStatus {
		case cost.IronRuleWarning:
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "iron_rule_warning",
				Message:    ironMsg,
			})
		case cost.IronRuleFail:
			return fmt.Errorf("iron rule: %s", ironMsg)
		}
	}

	stepStart := time.Now()
	e.trace("adapter_start", step.ID, 0, map[string]string{
		"persona": resolvedPersona,
		"adapter": adapterDef.Binary,
		"model":   resolvedModel,
	})
	result, err := stepRunner.Run(ctx, cfg)
	adapterDurationMs := time.Since(stepStart).Milliseconds()
	if err != nil {
		e.trace("adapter_end", step.ID, adapterDurationMs, map[string]string{
			"status": StateFailed,
			"error":  err.Error(),
		})
		// Let the higher-level executor emit a single failure event; just
		// propagate the error with context so enriched details (e.g. from
		// *adapter.StepError) remain available to callers.
		if e.logger != nil {
			e.logger.LogStepEnd(pipelineID, step.ID, StateFailed, time.Since(stepStart), 0, 0, 0, err.Error())
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
				Success:      false,
				ErrorMessage: err.Error(),
			})
		}
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
		if e.logger != nil {
			e.logger.LogStepEnd(pipelineID, step.ID, StateFailed, time.Since(stepStart), result.ExitCode, 0, result.TokensUsed, "rate limited: "+result.ResultContent)
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

	stepDuration := time.Since(stepStart).Milliseconds()

	// Emit step progress: processing results
	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "step_progress",
		Persona:       resolvedPersona,
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
		_, budgetStatus := e.costLedger.Record(pipelineID, step.ID, resolvedModel, result.TokensIn, result.TokensOut, result.TokensUsed)
		switch budgetStatus {
		case cost.BudgetWarning:
			e.emit(event.Event{
				Timestamp:     time.Now(),
				PipelineID:    pipelineID,
				StepID:        step.ID,
				State:         "budget_warning",
				Message:       fmt.Sprintf("Cost warning: %s", e.costLedger.Summary()),
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
		// Validate stdout size limit
		maxSize := execution.Manifest.Runtime.Artifacts.GetMaxStdoutSize()
		if int64(len(stdoutData)) > maxSize {
			return fmt.Errorf("stdout artifact size (%d bytes) exceeds limit (%d bytes); consider reducing output or increasing runtime.artifacts.max_stdout_size",
				len(stdoutData), maxSize)
		}

		// Write stdout artifacts using raw stdout data
		e.writeOutputArtifacts(execution, step, workspacePath, stdoutData)
	}

	// Write file-based output artifacts to workspace
	// Use ResultContent if available (extracted from adapter response)
	// Don't fall back to raw stdout as it contains JSON wrapper, not actual content
	if result.ResultContent != "" && !hasStdoutArtifacts {
		artifactContent := []byte(result.ResultContent)
		e.writeOutputArtifacts(execution, step, workspacePath, artifactContent)
	} else if !hasStdoutArtifacts {
		// Skip writing artifacts when ResultContent is empty to avoid overwriting
		// existing artifacts with empty content during relay compaction or parsing failures
		e.trace(audit.TraceArtifactSkipEmpty, step.ID, 0, map[string]string{
			"reason": "ResultContent is empty, preserving existing content",
		})
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
		// Resolve contract source path using pipeline context.
		// If no explicit source is set, infer from the step's first output artifact
		// so the contract validates the file the persona actually writes to.
		resolvedSource := execution.Context.ResolveContractSource(step.Handover.Contract)
		if resolvedSource == "" && len(step.OutputArtifacts) > 0 {
			resolvedSource = execution.Context.ResolveArtifactPath(step.OutputArtifacts[0])
		}

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
			MustPass:   step.Handover.Contract.MustPass,
			MaxRetries: step.Handover.Contract.MaxRetries,
			Model:      step.Handover.Contract.Model,
			Criteria:   step.Handover.Contract.Criteria,
			Threshold:  step.Handover.Contract.Threshold,
		}

		// Use schema filename for display when available, fall back to contract type
		contractDisplayName := step.Handover.Contract.Type
		if step.Handover.Contract.SchemaPath != "" {
			contractDisplayName = filepath.Base(step.Handover.Contract.SchemaPath)
		}

		e.emit(event.Event{
			Timestamp:       time.Now(),
			PipelineID:      pipelineID,
			StepID:          step.ID,
			State:           "validating",
			Message:         fmt.Sprintf("Validating %s contract", step.Handover.Contract.Type),
			CurrentAction:   "Validating contract",
			ValidationPhase: contractDisplayName,
		})

		e.trace("contract_validation_start", step.ID, 0, map[string]string{
			"type":   step.Handover.Contract.Type,
			"source": resolvedSource,
		})
		contractStart := time.Now()
		if err := contract.Validate(contractCfg, workspacePath); err != nil {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "contract_failed",
				Message:    err.Error(),
			})

			// Check if we should fail the step or allow soft failure
			if contractCfg.MustPass {
				e.trace("contract_validation_end", step.ID, time.Since(contractStart).Milliseconds(), map[string]string{
					"type":   step.Handover.Contract.Type,
					"result": "fail",
					"error":  err.Error(),
				})
				if e.logger != nil {
					e.logger.LogContractResult(pipelineID, step.ID, step.Handover.Contract.Type, "fail")
					e.logger.LogStepEnd(pipelineID, step.ID, StateFailed, time.Since(stepStart), result.ExitCode, len(stdoutData), result.TokensUsed, err.Error())
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
						ErrorMessage: "contract validation failed: " + err.Error(),
					})
				}
				e.recordDecision(pipelineID, step.ID, "contract",
					fmt.Sprintf("contract validation failed (hard) for step %s", step.ID),
					"must_pass is true, failing the step",
					map[string]interface{}{
						"contract_type": step.Handover.Contract.Type,
						"must_pass":     true,
						"error":         err.Error(),
					},
				)
				return fmt.Errorf("contract validation failed: %w", err)
			}
			e.recordDecision(pipelineID, step.ID, "contract",
				fmt.Sprintf("contract validation soft-failed for step %s", step.ID),
				"must_pass is false, continuing despite validation failure",
				map[string]interface{}{
					"contract_type": step.Handover.Contract.Type,
					"must_pass":     false,
					"error":         err.Error(),
				},
			)
			e.trace("contract_validation_end", step.ID, time.Since(contractStart).Milliseconds(), map[string]string{
				"type":   step.Handover.Contract.Type,
				"result": "soft_fail",
			})
			// Soft failure: log the validation error but continue execution
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "contract_soft_failure",
				Message:    fmt.Sprintf("contract validation failed but continuing (must_pass: false): %s", err.Error()),
			})
			if e.logger != nil {
				e.logger.LogContractResult(pipelineID, step.ID, step.Handover.Contract.Type, "soft_fail")
			}
		} else {
			e.trace("contract_validation_end", step.ID, time.Since(contractStart).Milliseconds(), map[string]string{
				"type":   step.Handover.Contract.Type,
				"result": "pass",
			})
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "contract_passed",
				Message:    fmt.Sprintf("%s contract validated", step.Handover.Contract.Type),
			})
			if e.logger != nil {
				e.logger.LogContractResult(pipelineID, step.ID, step.Handover.Contract.Type, "pass")
			}
			e.recordDecision(pipelineID, step.ID, "contract",
				fmt.Sprintf("contract validation passed for step %s", step.ID),
				fmt.Sprintf("%s contract validated successfully", step.Handover.Contract.Type),
				map[string]interface{}{
					"contract_type": step.Handover.Contract.Type,
				},
			)
			// Run contract_validated hooks (non-blocking by default)
			if e.hookRunner != nil {
				e.hookRunner.RunHooks(ctx, hooks.HookEvent{
					Type:       hooks.EventContractValidated,
					PipelineID: pipelineID,
					StepID:     step.ID,
					Workspace:  workspacePath,
				})
			}
		}
	}
	if step.Handover.Contract.Type == "" && e.logger != nil {
		e.logger.LogContractResult(pipelineID, step.ID, "none", "skip")
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
			Workspace:  workspacePath,
			Artifacts:  stepArtifacts,
		})
	}

	// Run step_completed hooks (blocking by default)
	stepCompletedEvt := hooks.HookEvent{
		Type:       hooks.EventStepCompleted,
		PipelineID: pipelineID,
		StepID:     step.ID,
		Input:      execution.Input,
		Workspace:  workspacePath,
	}
	if e.hookRunner != nil {
		if _, err := e.hookRunner.RunHooks(ctx, stepCompletedEvt); err != nil {
			return fmt.Errorf("step_completed hook failed: %w", err)
		}
	}
	e.fireWebhooks(ctx, stepCompletedEvt)

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      StateCompleted,
		Persona:    resolvedPersona,
		DurationMs: stepDuration,
		TokensUsed: result.TokensUsed,
		Artifacts:  stepArtifacts,
		TokensIn:   result.TokensIn,
		TokensOut:  result.TokensOut,
	})

	if e.logger != nil {
		e.logger.LogStepEnd(pipelineID, step.ID, "success", time.Since(stepStart), result.ExitCode, len(stdoutData), result.TokensUsed, "")
	}

	// Record performance metric for TUI step breakdown
	if e.store != nil {
		completedAt := time.Now()
		e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
			RunID:              pipelineID,
			StepID:             step.ID,
			PipelineName:       execution.Status.PipelineName,
			Persona:            resolvedPersona,
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

// resolveModel applies five-tier model precedence:
// 1. CLI --model flag override (highest — explicit user intent)
// 2. Per-step model pinning (step.Model in pipeline YAML)
// 3. Per-persona model pinning
// 4. Auto-routing based on complexity heuristics (when routing.auto_route is enabled)
// 5. Adapter default (empty string)
func (e *DefaultPipelineExecutor) resolveModel(step *Step, persona *manifest.Persona, routing *manifest.RoutingConfig, personaName string) string {
	if e.modelOverride != "" {
		return e.modelOverride
	}
	if step != nil && step.Model != "" {
		return step.Model
	}
	if persona.Model != "" {
		return persona.Model
	}
	// Tier 4: auto-route based on complexity heuristics
	if routing != nil && routing.AutoRoute {
		tier := ClassifyStepComplexity(step, persona, personaName)
		if model := routing.ResolveComplexityModel(tier); model != "" {
			return model
		}
	}
	return ""
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
	e.store.RecordDecision(&state.DecisionRecord{
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
		exec.Command("git", "-C", absPath, "update-index", "--skip-worktree", "CLAUDE.md").Run()

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
		exec.Command("git", "init", "-q", wsPath).Run()
		return wsPath, nil
	}

	// Create directory under .wave/workspaces/<pipeline>/<step>/
	wsPath := filepath.Join(wsRoot, pipelineID, step.ID)
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", err
	}
	// Anchor Claude Code path resolution (see mount-based workspace above)
	exec.Command("git", "init", "-q", wsPath).Run()
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
	if len(step.Workspace.Mount) == 0 {
		return workspacePath
	}
	for _, m := range step.Workspace.Mount {
		if m.Source == "./" || m.Source == "." {
			// Mount target is stored as e.g. "/project" — strip leading
			// slash to get a relative path within the workspace.
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
		for idx := indexOf(prompt, pattern); idx != -1; idx = indexOf(prompt, pattern) {
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
		sb.WriteString("Please address the issues from the previous attempt and try a different approach if needed.\n\n---\n\n")
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
			os.MkdirAll(filepath.Dir(artPath), 0755)

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
			} else {
				// Fall back to writing ResultContent
				os.MkdirAll(filepath.Dir(artPath), 0755)
				os.WriteFile(artPath, stdout, 0644)
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

func (e *DefaultPipelineExecutor) emit(ev event.Event) {
	if e.emitter != nil {
		e.emitter.Emit(ev)
	}
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
	e.debugTracer.Emit(audit.TraceEvent{
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
func schemaFieldPlaceholder(field string, prop map[string]any) string {
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

		// Wildcard path: extract all array elements as separate deliverables
		if ContainsWildcard(outcome.JSONPath) {
			e.processWildcardOutcome(execution, step, outcome, data)
			continue
		}

		value, err := ExtractJSONPath(data, outcome.JSONPath)
		if err != nil {
			var emptyErr *EmptyArrayError
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

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      StateRunning,
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
			State:      StateRunning,
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
			CurrentStep:    "", // Not tracked in pipeline_state table
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
	if step.SubInput != "" && execution.Context != nil {
		resolved := execution.Context.ResolvePlaceholders(step.SubInput)
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
		execution.States[step.ID] = StateFailed
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
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
	if e.stepTimeoutOverride > 0 {
		childOpts = append(childOpts, WithStepTimeout(e.stepTimeoutOverride))
	}
	if e.debugTracer != nil {
		childOpts = append(childOpts, WithDebugTracer(e.debugTracer))
	}
	if e.logger != nil {
		childOpts = append(childOpts, WithAuditLogger(e.logger))
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
		execution.States[step.ID] = StateFailed
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
		}
		return fmt.Errorf("sub-pipeline %q failed: %w", pipelineName, err)
	}

	// Link parent-child state and mark completed
	if e.store != nil && childRunID != "" {
		_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
		_ = e.store.UpdateRunStatus(childRunID, "completed", "", childExecutor.GetTotalTokens())
	}

	// Extract child artifacts and merge context variables
	childExec := childExecutor.LastExecution()
	if step.Config != nil && childExec != nil {
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

		// Merge child context variables into parent (last-writer-wins)
		execution.Context.MergeFrom(childExec.Context, pipelineName)

		// Evaluate stop condition if configured
		if step.Config.StopCondition != "" {
			if evaluateStopCondition(step.Config.StopCondition, childExec.Context) {
				execution.Context.SetCustomVariable("sub_pipeline_stop", "true")
			}
		}
	}

	execution.mu.Lock()
	execution.States[step.ID] = StateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
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

	// Resolve the items array — may contain template expressions
	itemsExpr := step.Iterate.Over
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

	// Sequential iterate
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

		// Resolve input
		input := execution.Input
		if step.SubInput != "" {
			input, err = ResolveTemplate(step.SubInput, tmplCtx)
			if err != nil {
				return fmt.Errorf("iterate item %d: failed to resolve input: %w", i, err)
			}
		}

		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      event.StateIterationProgress,
			Message:    fmt.Sprintf("iterate item %d/%d: %s", i+1, len(items), resolvedName),
			TotalSteps: len(items),
			CompletedSteps: i,
		})

		if err := e.runNamedSubPipeline(ctx, execution, step, resolvedName, input); err != nil {
			return fmt.Errorf("iterate item %d (%s): %w", i, resolvedName, err)
		}
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

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for i, item := range items {
		tmplCtx := NewTemplateContext(execution.Input, "")
		tmplCtx.Item = item

		resolvedName, err := ResolveTemplate(pipelineNameTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("iterate item %d: failed to resolve pipeline name: %w", i, err)
		}

		input := execution.Input
		if step.SubInput != "" {
			input, err = ResolveTemplate(step.SubInput, tmplCtx)
			if err != nil {
				return fmt.Errorf("iterate item %d: failed to resolve input: %w", i, err)
			}
		}

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

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      event.StateCompleted,
		Message:    fmt.Sprintf("iterate: all %d items completed (parallel)", len(items)),
	})

	return nil
}

// executeAggregateInDAG collects outputs from prior steps and writes them to a file.
func (e *DefaultPipelineExecutor) executeAggregateInDAG(ctx context.Context, execution *PipelineExecution, step *Step) error {
	pipelineID := execution.Status.ID

	// Resolve the source expression
	sourceExpr := step.Aggregate.From
	if execution.Context != nil {
		sourceExpr = execution.Context.ResolvePlaceholders(sourceExpr)
	}

	var result string
	var err error

	switch step.Aggregate.Strategy {
	case "concat":
		result = sourceExpr
	case "merge_arrays":
		result, err = mergeJSONArrays(sourceExpr)
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

	// Register artifact in execution context
	if execution.Context != nil {
		execution.Context.SetArtifactPath("merged-findings", outputPath)
	}

	execution.mu.Lock()
	execution.States[step.ID] = StateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
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

	// Resolve the branch condition
	value := step.Branch.On
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
		execution.States[step.ID] = StateCompleted
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "skipped by branch")
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
				execution.States[step.ID] = StateCompleted
				execution.mu.Unlock()
				if e.store != nil {
					e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "loop condition met")
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
	execution.States[step.ID] = StateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "max iterations reached")
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
		execution.States[step.ID] = StateFailed
		execution.mu.Unlock()
		if e.store != nil {
			e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
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
	execution.States[step.ID] = StateCompleted
	execution.mu.Unlock()
	if e.store != nil {
		e.store.SaveStepState(pipelineID, step.ID, state.StateCompleted, "")
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
		return &GateAbortError{StepID: step.ID, Choice: decision.Label}
	}

	// If the target is a step ID that has already been completed, re-queue it
	// (revision loop pattern). If the target is a pending/future step, just
	// continue normal DAG flow — the scheduler will reach it naturally.
	if decision != nil && decision.Target != "" && decision.Target != "_fail" {
		execution.mu.Lock()
		targetState := execution.States[decision.Target]
		execution.mu.Unlock()

		if targetState == StateCompleted || targetState == StateFailed || targetState == StateSkipped {
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
		execution.States[stepID] = StatePending
	}
	execution.mu.Unlock()

	for stepID := range toReset {
		if e.store != nil {
			e.store.SaveStepState(pipelineID, stepID, state.StatePending, "re-queued by gate routing")
		}
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     stepID,
			State:      StatePending,
			Message:    fmt.Sprintf("re-queued by gate routing to %q", targetStepID),
		})
	}

	return &ReQueueError{TargetStepID: targetStepID, ResetSteps: toReset}
}

// ReQueueError signals that steps have been re-queued via gate routing.
// The main executor loop should handle this by re-entering the scheduling loop.
type ReQueueError struct {
	TargetStepID string
	ResetSteps   map[string]bool
}

func (e *ReQueueError) Error() string {
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

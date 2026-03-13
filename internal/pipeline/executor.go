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
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/skill"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/worktree"
	"github.com/recinq/wave/internal/workspace"
	"golang.org/x/sync/errgroup"
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

// NewChildExecutor creates a fresh executor that shares the same adapter runner,
// event emitter, workspace manager, and configuration, but has independent
// execution state. Used for child pipeline invocation within matrix strategies.
func (e *DefaultPipelineExecutor) NewChildExecutor() *DefaultPipelineExecutor {
	return &DefaultPipelineExecutor{
		runner:                 e.runner,
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
				State:      "failed",
				Message:    err.Error(),
			})
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
			State:          "running",
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
	} else {
		execution.Status.State = StateCompleted
		if e.store != nil {
			e.store.SavePipelineState(pipelineID, StateCompleted, input)
		}
	}

	elapsed := time.Since(execution.Status.StartedAt).Milliseconds()
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		State:      "completed",
		DurationMs: elapsed,
		Message:    fmt.Sprintf("%d steps completed", schedulableSteps),
	})

	// Clean up completed pipeline from in-memory storage to prevent memory leak
	e.cleanupCompletedPipeline(pipelineID)

	return nil
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

	// Check if this step uses a matrix strategy
	if step.Strategy != nil && step.Strategy.Type == "matrix" {
		return e.executeMatrixStep(ctx, execution, step)
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
				State:      "retrying",
				Message:    fmt.Sprintf("attempt %d/%d", attempt, maxAttempts),
			})
			time.Sleep(step.Retry.ComputeDelay(attempt))
		}

		// Record attempt start
		attemptStart := time.Now()
		if e.store != nil {
			e.store.RecordStepAttempt(&state.StepAttemptRecord{
				RunID:     pipelineID,
				StepID:    step.ID,
				Attempt:   attempt,
				State:     "running",
				StartedAt: attemptStart,
			})
		}

		// Start progress ticker for smooth animation updates during step execution
		cancelTicker := e.startProgressTicker(ctx, pipelineID, step.ID)

		err := e.runStepExecution(ctx, execution, step)

		// Stop progress ticker when step completes
		cancelTicker()

		attemptDuration := time.Since(attemptStart)

		if err != nil {
			lastErr = err

			// Record failed attempt
			if e.store != nil {
				completedAt := time.Now()
				e.store.RecordStepAttempt(&state.StepAttemptRecord{
					RunID:        pipelineID,
					StepID:       step.ID,
					Attempt:      attempt,
					State:        "failed",
					ErrorMessage: err.Error(),
					DurationMs:   attemptDuration.Milliseconds(),
					StartedAt:    attemptStart,
					CompletedAt:  &completedAt,
				})
			}

			if attempt < maxAttempts {
				// Set up attempt context for prompt adaptation on next retry
				if step.Retry.AdaptPrompt {
					errMsg := err.Error()
					// Capture stdout tail from results if available
					stdoutTail := ""
					execution.mu.Lock()
					if result, ok := execution.Results[step.ID]; ok {
						if stdout, ok := result["stdout"].(string); ok {
							if len(stdout) > 2000 {
								stdoutTail = stdout[len(stdout)-2000:]
							} else {
								stdoutTail = stdout
							}
						}
					}
					execution.mu.Unlock()

					execution.mu.Lock()
					execution.AttemptContexts[step.ID] = &AttemptContext{
						Attempt:     attempt + 1,
						MaxAttempts: maxAttempts,
						PriorError:  errMsg,
						PriorStdout: stdoutTail,
					}
					execution.mu.Unlock()
				}
				continue
			}

			// All attempts exhausted — apply on_failure policy
			onFailure := step.Retry.OnFailure
			if onFailure == "" {
				if step.IsOptional() {
					onFailure = "continue"
				} else {
					onFailure = "fail"
				}
			}

			switch onFailure {
			case "skip":
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
				return nil

			case "continue":
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
				return nil

			case "rework":
				return e.executeReworkStep(ctx, execution, step, lastErr, attemptDuration)

			default: // "fail"
				execution.mu.Lock()
				execution.States[step.ID] = StateFailed
				execution.mu.Unlock()
				if e.store != nil {
					e.store.SaveStepState(pipelineID, step.ID, state.StateFailed, err.Error())
				}
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

		return nil
	}

	return lastErr
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
			if len(stdout) > 2000 {
				attemptCtx.PriorStdout = stdout[len(stdout)-2000:]
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

	adapterDef := execution.Manifest.GetAdapter(persona.Adapter)
	if adapterDef == nil {
		return fmt.Errorf("adapter %q not found in manifest", persona.Adapter)
	}

	// Create workspace under .wave/workspaces/<pipeline>/<step>/
	workspacePath, err := e.createStepWorkspace(execution, step)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	execution.mu.Lock()
	execution.WorkspacePaths[step.ID] = workspacePath
	execution.mu.Unlock()

	// Pre-create .wave/output/ so personas without Bash can write artifacts
	if len(step.OutputArtifacts) > 0 {
		outputDir := filepath.Join(workspacePath, ".wave", "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output dir: %w", err)
		}
	}

	e.emit(event.Event{
		Timestamp:     time.Now(),
		PipelineID:    pipelineID,
		StepID:        step.ID,
		State:         "running",
		Persona:       step.Persona,
		Message:       fmt.Sprintf("Starting %s persona in %s", step.Persona, workspacePath),
		CurrentAction: "Initializing",
		Model:            e.resolveModel(persona),
		Adapter:       adapterDef.Binary,
		Temperature:   persona.Temperature,
	})

	// Inject artifacts from dependencies
	if err := e.injectArtifacts(execution, step, workspacePath); err != nil {
		return fmt.Errorf("failed to inject artifacts: %w", err)
	}

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
		e.logger.LogStepStart(pipelineID, step.ID, step.Persona, artifactNames)
	}

	prompt := e.buildStepPrompt(execution, step)

	if e.logger != nil {
		e.logger.LogToolCall(pipelineID, step.ID, "adapter.Run", fmt.Sprintf("persona=%s prompt_len=%d", step.Persona, len(prompt)))
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
	if execution.Pipeline.Requires != nil && len(execution.Pipeline.Requires.Skills) > 0 {
		skillNames := execution.Pipeline.Requires.SkillNames()
		provisioner := skill.NewProvisioner(execution.Pipeline.Requires.Skills, "")
		commands, _ := provisioner.DiscoverCommands(skillNames)
		// If we found any commands, provision them into a temp dir that the adapter can use
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

	// Auto-generate contract compliance prompt for CLAUDE.md
	contractPrompt := e.buildContractPrompt(step, execution.Context)

	cfg := adapter.AdapterRunConfig{
		Adapter:          adapterDef.Binary,
		Persona:          step.Persona,
		WorkspacePath:    workspacePath,
		Prompt:           prompt,
		SystemPrompt:     systemPrompt,
		Timeout:          timeout,
		Temperature:      persona.Temperature,
		Model:            e.resolveModel(persona),
		AllowedTools:     persona.Permissions.AllowedTools,
		DenyTools:        persona.Permissions.Deny,
		OutputFormat:     adapterDef.OutputFormat,
		Debug:            e.debug,
		SandboxEnabled:   sandboxEnabled,
		AllowedDomains:   sandboxDomains,
		EnvPassthrough:   envPassthrough,
		SkillCommandsDir:    skillCommandsDir,
		ContractPrompt:      contractPrompt,
		MaxConcurrentAgents: step.MaxConcurrentAgents,
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
		if e.logger != nil {
			e.logger.LogStepEnd(pipelineID, step.ID, "failed", time.Since(stepStart), 0, 0, 0, err.Error())
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      step.Persona,
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
			e.logger.LogStepEnd(pipelineID, step.ID, "failed", time.Since(stepStart), result.ExitCode, 0, result.TokensUsed, "rate limited: "+result.ResultContent)
		}
		if e.store != nil {
			completedAt := time.Now()
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      step.Persona,
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

	execution.mu.Lock()
	execution.Results[step.ID] = output
	execution.mu.Unlock()

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
				if e.logger != nil {
					e.logger.LogContractResult(pipelineID, step.ID, step.Handover.Contract.Type, "fail")
					e.logger.LogStepEnd(pipelineID, step.ID, "failed", time.Since(stepStart), result.ExitCode, len(stdoutData), result.TokensUsed, err.Error())
				}
				if e.store != nil {
					completedAt := time.Now()
					e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
						RunID:        pipelineID,
						StepID:       step.ID,
						PipelineName: execution.Status.PipelineName,
						Persona:      step.Persona,
						StartedAt:    stepStart,
						CompletedAt:  &completedAt,
						DurationMs:   time.Since(stepStart).Milliseconds(),
						TokensUsed:   result.TokensUsed,
						Success:      false,
						ErrorMessage: "contract validation failed: " + err.Error(),
					})
				}
				return fmt.Errorf("contract validation failed: %w", err)
			}
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

	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: pipelineID,
		StepID:     step.ID,
		State:      "completed",
		Persona:    step.Persona,
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
			Persona:            step.Persona,
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

// resolveModel applies three-tier model precedence:
// 1. CLI --model flag override (highest — explicit user intent)
// 2. Per-persona model pinning
// 3. Adapter default (empty string)
func (e *DefaultPipelineExecutor) resolveModel(persona *manifest.Persona) string {
	if e.modelOverride != "" {
		return e.modelOverride
	}
	if persona.Model != "" {
		return persona.Model
	}
	return ""
}

func (e *DefaultPipelineExecutor) createStepWorkspace(execution *PipelineExecution, step *Step) (string, error) {
	pipelineID := execution.Status.ID
	wsRoot := execution.Manifest.Runtime.WorkspaceRoot
	if wsRoot == "" {
		wsRoot = ".wave/workspaces"
	}

	// Handle workspace ref — share another step's workspace
	if step.Workspace.Ref != "" {
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
		if e.debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Loading prompt from source_path: %s\n", sourcePath)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			if e.debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Failed to read prompt from %s: %v\n", sourcePath, err)
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

	// NOTE: Schema injection for json_schema contracts is handled exclusively by
	// buildContractPrompt → ContractPrompt → CLAUDE.md. Do NOT duplicate it here.
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
			sb.WriteString("### Previous Output (last 2000 chars)\n```\n")
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
			if err := os.WriteFile(artPath, stdout, 0644); err != nil && e.debug {
				fmt.Printf("[DEBUG] Failed to write stdout artifact %s: %v\n", art.Name, err)
			}
			execution.mu.Lock()
			execution.ArtifactPaths[key] = artPath
			execution.mu.Unlock()

			if e.debug {
				fmt.Printf("[DEBUG] Wrote stdout artifact %s to %s (%d bytes)\n", art.Name, artPath, len(stdout))
			}
		} else {
			// File-based artifacts: resolve path using pipeline context
			resolvedPath := execution.Context.ResolveArtifactPath(art)
			artPath = filepath.Join(workspacePath, resolvedPath)

			// If the persona already wrote the file, trust it and don't overwrite
			if _, err := os.Stat(artPath); err == nil {
				execution.mu.Lock()
				execution.ArtifactPaths[key] = artPath
				execution.mu.Unlock()
				if e.debug {
					fmt.Printf("[DEBUG] Artifact %s already exists at %s, preserving persona-written file\n", art.Name, artPath)
				}
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
			e.store.RegisterArtifact(execution.Status.ID, step.ID, art.Name, registeredPath, art.Type, size)
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

// buildContractPrompt generates a contract compliance section for CLAUDE.md
// based on the step's contract definition and output artifacts. This tells the
// persona exactly what format the output must be in, so pipeline authors don't
// need to repeat format requirements in their prompts.
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
			State:      "running",
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
			State:      "running",
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

// ResumeWithValidation resumes a pipeline with full validation and error handling.
// When force is true, phase validation and stale artifact checks are skipped.
// When priorRunID is provided, artifact paths are resolved from that specific run's
// workspace directory instead of scanning for the most recent match.
func (e *DefaultPipelineExecutor) ResumeWithValidation(ctx context.Context, p *Pipeline, m *manifest.Manifest, input string, fromStep string, force bool, priorRunID ...string) error {
	manager := NewResumeManager(e)
	return manager.ResumeFromStep(ctx, p, m, input, fromStep, force, priorRunID...)
}

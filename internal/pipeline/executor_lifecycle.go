package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/preflight"
	"github.com/recinq/wave/internal/relay"
	"github.com/recinq/wave/internal/scope"
	"github.com/recinq/wave/internal/state"
)

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

	// Emit Wave Lego Protocol (ADR-011) load-time warnings collected by the
	// YAML loader, plus any DAG validator warnings (fidelity, mixed-persona
	// threads). These are non-fatal deprecation / style notices.
	for _, w := range p.Warnings {
		e.emit(event.Event{
			Timestamp: time.Now(),
			State:     "warning",
			Message:   w,
		})
	}
	for _, w := range validator.Warnings {
		e.emit(event.Event{
			Timestamp: time.Now(),
			State:     "warning",
			Message:   w,
		})
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
	seenSkill := make(map[string]bool)
	addSkill := func(name string) {
		if name == "" || seenSkill[name] {
			return
		}
		seenSkill[name] = true
		resolvedPipelineSkills = append(resolvedPipelineSkills, name)
	}
	for _, s := range p.Skills {
		addSkill(pipelineContext.ResolvePlaceholders(s))
	}
	// Per #1113: pipeline-level skill set is the union of step-level
	// skills declarations. Preflight catches missing skills before any
	// step starts, with a single resolved name suggesting `wave skills add`.
	for i := range p.Steps {
		for _, s := range p.Steps[i].Skills {
			addSkill(pipelineContext.ResolvePlaceholders(s))
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
	if errs := e.sec.validateSkillRefs(setup.resolvedPipelineSkills, p.Metadata.Name, m); len(errs) > 0 {
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

		// Build step inputs for forge dependency scanning. Use the resolved
		// per-step permission set so a step-level overlay (e.g. adding Write
		// to a navigator step) does not trip the forge dependency scanner.
		forgeStepInputs := make([]preflight.ForgeStepInput, 0, len(setup.sortedSteps))
		for _, step := range setup.sortedSteps {
			var personaTools []string
			resolvedPersona := setup.pipelineContext.ResolvePlaceholders(step.Persona)
			if persona := m.GetPersona(resolvedPersona); persona != nil {
				adapterName := persona.Adapter
				if step.Adapter != "" {
					adapterName = step.Adapter
				}
				personaTools = ResolveStepPermissions(step, persona, m.GetAdapter(adapterName)).AllowedTools
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

	// Auto-wire the ontology service if the caller did not inject one. The
	// real Service is active iff the manifest declares at least one
	// ontology context; otherwise the NoOp keeps call sites cheap.
	if _, isNoop := e.ontology.(ontology.NoOp); isNoop {
		var auditSink ontology.AuditSink
		if e.logger != nil {
			auditSink = e.logger
		}
		e.ontology = ontology.New(
			ontology.Config{Enabled: ontology.EnabledFromManifest(m)},
			ontology.Deps{
				Manifest:  m,
				Store:     e.store,
				Emitter:   e.emitter,
				AuditSink: auditSink,
			},
		)
	}

	// Ontology staleness check: warn if ontology is older than latest commit.
	if msg := e.ontology.CheckStaleness(); msg != "" {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			State:      "warning",
			Message:    msg,
		})
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

	// Initialize outcome tracker for this pipeline (only if not already set)
	if e.outcomeTracker == nil {
		e.outcomeTracker = state.NewOutcomeTracker(pipelineID, e.store)
	} else {
		// Update pipeline ID if tracker already exists
		e.outcomeTracker.SetPipelineID(pipelineID)
		e.outcomeTracker.SetStore(e.store)
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

	// Seed parent env vars as custom variables so child templates resolve
	// {{ env.<key> }} via ResolvePlaceholders.
	if len(e.parentEnv) > 0 {
		for k, v := range e.parentEnv {
			execution.Context.SetCustomVariable("env."+k, v)
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
		wsRoot = ".agents/workspaces"
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
			// Design rejection: a step's contract with on_failure: rejected
			// fired. Halt the pipeline in the dedicated `rejected` terminal
			// state — distinct from `failed` so UIs render it without the
			// red "this is broken" signal. The error is preserved so callers
			// (CLI runOnce, webui) can inspect it via errors.As.
			var rejectionErr *ContractRejectionError
			if errors.As(err, &rejectionErr) {
				execution.Status.State = stateRejected
				rejectedStepID := rejectionErr.StepID
				if rejectedStepID == "" && len(ready) > 0 {
					rejectedStepID = ready[0].ID
				}
				if rejectedStepID != "" {
					execution.Status.FailedSteps = append(execution.Status.FailedSteps, rejectedStepID)
				}
				if e.store != nil {
					_ = e.store.SavePipelineState(pipelineID, stateRejected, execution.Input)
				}
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     rejectedStepID,
					State:      stateRejected,
					Message:    err.Error(),
				})
				if e.retroGenerator != nil {
					e.retroGenerator.Generate(pipelineID, execution.Pipeline.Metadata.Name)
				}
				e.cleanupCompletedPipeline(pipelineID)
				return 0, &StepExecutionError{StepID: rejectedStepID, Err: err}
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
				adapterName := persona.Adapter
				if step.Adapter != "" {
					adapterName = step.Adapter
				}
				personaTools = ResolveStepPermissions(step, persona, m.GetAdapter(adapterName)).AllowedTools
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

	// Initialize outcome tracker
	if e.outcomeTracker == nil {
		e.outcomeTracker = state.NewOutcomeTracker(pipelineID, e.store)
	} else {
		e.outcomeTracker.SetPipelineID(pipelineID)
		e.outcomeTracker.SetStore(e.store)
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

	// Seed parent env vars as custom variables so child templates resolve
	// {{ env.<key> }} via ResolvePlaceholders.
	if len(e.parentEnv) > 0 {
		for k, v := range e.parentEnv {
			execution.Context.SetCustomVariable("env."+k, v)
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
		wsRoot = ".agents/workspaces"
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
			// Register output artifacts in ArtifactPaths so downstream
			// inject_artifacts can find the files the script wrote (#1490).
			workspacePath := execution.WorkspacePaths[step.ID]
			e.writeOutputArtifacts(execution, step, workspacePath, nil)
			// Run handover contract validation for command steps.
			// Command steps run in the project root (or mount target), so resolve
			// contract sources against the command's actual working directory.
			contractDir := resolveCommandWorkDir(workspacePath, step)
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

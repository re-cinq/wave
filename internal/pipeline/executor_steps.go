package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/security"
	"github.com/recinq/wave/internal/state"
	"golang.org/x/sync/errgroup"
)

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
	if e.sec == nil || e.sec.inputSanitizer == nil {
		return nil, fmt.Errorf("command step %q: refusing to execute without input sanitizer", step.ID)
	}

	// SECURITY: Sanitize the resolved script to detect injection attempts.
	// Template resolution can introduce user-controlled content (e.g. issue titles,
	// branch names) that could contain shell metacharacters or injection payloads.
	if e.sec.inputSanitizer != nil {
		record, sanitized, err := e.sec.inputSanitizer.SanitizeInput(script, "command_script")
		if err != nil {
			// Sanitization rejected the input (strict mode / prompt injection detected)
			if e.sec.securityLogger != nil {
				e.sec.securityLogger.LogViolation(
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

	// Auto-inject declared dependency artifacts (issue #1452). Command
	// scripts can read upstream outputs at .agents/artifacts/<dep>/<name>
	// or the back-compat alias .agents/output/<name> without any
	// workspace.mount or memory.inject_artifacts boilerplate.
	depArtifacts, err := e.injectDependencyArtifacts(execution, step, workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-inject dep artifacts for step %q: %w", step.ID, err)
	}

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

	// Append WAVE_DEP_<DEP>_<NAME>=<canonical path> + WAVE_DEPS_DIR for
	// every auto-injected upstream artifact. Issue #1452 phase 3.
	cmd.Env = append(cmd.Env, BuildDepEnvVars(depArtifacts, workspacePath)...)

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

	// Strategy dispatch: concurrency, matrix, and all composition primitives
	// route through the StrategyExecutor registry in strategy.go. When the
	// step is a regular persona / command step the registry returns nil and
	// we fall through to the standard adapter pipeline.
	if strategy := selectStrategy(e, step); strategy != nil {
		return strategy.Execute(ctx, execution, step)
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
		// Register output artifacts so downstream `inject_artifacts` lookups
		// in `injectArtifacts` (executor.go: ArtifactPaths[step.ID+":"+name])
		// resolve to the on-disk file the script wrote. Without this,
		// command-step outputs were silently delivered as 0-byte blobs to
		// downstream personas — see #1490.
		workspacePath := execution.WorkspacePaths[step.ID]
		e.writeOutputArtifacts(execution, step, workspacePath, nil)
		// Run handover contract validation (same as persona steps).
		// Resolve against the command's actual working directory, not the workspace root.
		contractDir := resolveCommandWorkDir(workspacePath, step)
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

		// Start stall watchdog if configured. Composition steps (iterate /
		// aggregate / branch / loop / sub_pipeline) do not produce their
		// own stream events — their work happens in spawned child
		// pipelines under separate run IDs. Wiring a stall watchdog to
		// them would fire after the configured timeout regardless of
		// whether children are healthy. Skip the watchdog for those step
		// kinds; each child pipeline owns its own stall watchdog.
		stepCtx := ctx
		var watchdog *StallWatchdog
		isCompositionStep := step.Iterate != nil || step.Aggregate != nil ||
			step.Branch != nil || step.Loop != nil || step.SubPipeline != ""
		if !isCompositionStep {
			if stallTimeout := e.parseStallTimeout(execution.Manifest); stallTimeout > 0 {
				w, err := NewStallWatchdog(stallTimeout)
				if err != nil {
					cancelTicker()
					return fmt.Errorf("step %s: stall watchdog setup: %w", step.ID, err)
				}
				watchdog = w
				stepCtx = watchdog.Start(stepCtx)
			}
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
				e.recordStepOntologyUsage(execution, step, "skipped")
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
				e.recordStepOntologyUsage(execution, step, "failed")
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
				e.recordStepOntologyUsage(execution, step, "failed")
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
		e.recordStepOntologyUsage(execution, step, "success")

		return nil
	}

	return lastErr
}

// recordStepOntologyUsage is a thin adapter that projects the pipeline.Step
// and PipelineExecution into the primitives the ontology.Service expects.
// It keeps executor.go decoupled from the Service's call shape so future
// Step/Execution refactors don't force ontology API changes.
func (e *DefaultPipelineExecutor) recordStepOntologyUsage(execution *PipelineExecution, step *Step, stepStatus string) {
	if e.ontology == nil {
		return
	}
	hasContract := step.Handover.Contract.Type != ""
	e.ontology.RecordUsage(execution.Status.ID, step.ID, step.Contexts, hasContract, stepStatus)
}

// executeReworkStep handles on_failure=rework: marks the failed step, builds failure context,
// executes the rework target step, and re-registers its artifacts under the original step's ID.
func (e *DefaultPipelineExecutor) executeReworkStep(ctx context.Context, execution *PipelineExecution, failedStep *Step, failErr error, failDuration time.Duration) error {
	pipelineID := execution.Status.ID
	reworkStepID := failedStep.Retry.ReworkStep

	// Short-circuit when the parent context has already been cancelled or
	// timed out: launching the rework subprocess would only produce a
	// duplicate "context canceled" error and a misleading failure event.
	if err := ctx.Err(); err != nil {
		return err
	}

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

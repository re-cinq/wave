package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/hooks"
	"github.com/recinq/wave/internal/state"
)

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

	case OnFailureRejected:
		// Design rejection: contract failed because the persona output
		// deliberately signalled no-op (e.g. `implementable: false`). This
		// is NOT a runtime failure — the run terminates in the dedicated
		// `rejected` state and the CLI exits 0. Mark step state as
		// rejected so step-level UIs render it distinctly.
		execution.mu.Lock()
		execution.States[step.ID] = stateRejected
		execution.mu.Unlock()
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      stateRejected,
			Message:    fmt.Sprintf("design rejection: %s contract reported non-actionable result (%s)", c.Type, cErr.Error()),
		})
		if e.logger != nil {
			e.logger.LogContractResult(pipelineID, step.ID, c.Type, "rejected")
			_ = e.logger.LogStepEnd(pipelineID, step.ID, stateRejected, time.Since(stepStart), result.ExitCode, 0, result.TokensUsed, "design rejection: "+cErr.Error())
		}
		if e.store != nil {
			completedAt := time.Now()
			// A rejection is not a runtime failure; treat it as a
			// successful no-op for performance metrics so dashboards
			// don't double-count it as a failure.
			e.store.RecordPerformanceMetric(&state.PerformanceMetricRecord{
				RunID:        pipelineID,
				StepID:       step.ID,
				PipelineName: execution.Status.PipelineName,
				Persona:      resolvedPersona,
				StartedAt:    stepStart,
				CompletedAt:  &completedAt,
				DurationMs:   time.Since(stepStart).Milliseconds(),
				TokensUsed:   result.TokensUsed,
				Success:      true,
				ErrorMessage: "design rejection: " + cErr.Error(),
			})
		}
		e.recordDecision(pipelineID, step.ID, "contract",
			fmt.Sprintf("contract reported design rejection for step %s", step.ID),
			fmt.Sprintf("on_failure is 'rejected' — pipeline halted with non-actionable verdict: %s", cErr.Error()),
			map[string]interface{}{"contract_type": c.Type, "error": cErr.Error()},
		)
		return false, &ContractRejectionError{
			StepID:       step.ID,
			ContractType: c.Type,
			Reason:       cErr.Error(),
		}

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

// triggerContractRework writes review feedback to .agents/artifacts/review_feedback.json,
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
	feedbackPath := filepath.Join(workspacePath, ".agents", "artifacts", fmt.Sprintf("review_feedback_%s.json", step.ID))
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

		// Load and security-validate schema content. Errors are swallowed
		// here: buildContractPrompt is advisory (it drives persona guidance),
		// not authoritative — actual schema enforcement happens at validation
		// time, which surfaces the real error.
		schemaContent, _ := e.sec.loadSecureSchemaContent(step)
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
			b.WriteString(fmt.Sprintf("- `%s` → `.agents/artifacts/%s`\n", name, name))
		}
		b.WriteString("\nThese artifacts contain ALL data you need from prior pipeline steps. ")
		b.WriteString("Read these files instead of fetching equivalent data from external sources.\n")
	}

	if b.Len() == 0 {
		return ""
	}
	return b.String()
}

// processStepOutcomes extracts declared outcomes from step artifacts and registers
// them with the deliverable tracker for display in the pipeline output summary.
// Errors are logged as warnings — outcome extraction never fails a step.
//
// When a json_path contains [*] wildcard syntax, all array elements are extracted
// and each is registered as a separate deliverable. The optional json_path_label
// field provides per-item labels; when absent, items are labeled with their index.

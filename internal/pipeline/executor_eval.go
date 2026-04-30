// Package pipeline — executor_eval.go houses the post-run hook that emits
// EvalSignal observations to the pipeline_eval table. Phase 3.1 of the
// evolution-loop epic (#1565); see internal/contract/eval_signal.go for the
// signal types and aggregator.
//
// The hook fires in two places:
//
//   - recordStepEval — called from executor_steps.go on each terminal step
//     state transition (completed, completed_empty, failed). Skipped /
//     reworking states emit no signal.
//   - recordPipelineEval — called from executor_lifecycle.go after the final
//     SavePipelineState in finalizePipelineExecution. Aggregates the
//     per-execution SignalSet and persists one PipelineEvalRecord via
//     state.EvolutionStore.RecordEval.
//
// Persistence failures are logged via the executor's emit channel and never
// bubble up: evolution-loop telemetry must not block pipeline completion.
package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/state"
)

// EvolutionTrigger decides whether a pipeline's accumulated eval rows merit
// firing pipeline-evolve. Defined consumer-side so tests can stub the
// executor's trigger field without depending on internal/evolution.
//
// ShouldEvolve is consulted from recordPipelineEval after RecordEval
// succeeds. A true return causes an advisory "evolution_proposed" event
// to be emitted; errors are logged as warnings and never fail the run.
type EvolutionTrigger interface {
	ShouldEvolve(pipelineName string) (bool, string, error)
}

// signalSetFor returns (creating if needed) the per-run SignalSet on the
// executor. Safe for concurrent calls from parallel step goroutines.
func (e *DefaultPipelineExecutor) signalSetFor(runID string) *contract.SignalSet {
	if runID == "" {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.evalCollectors == nil {
		e.evalCollectors = make(map[string]*contract.SignalSet)
	}
	set, ok := e.evalCollectors[runID]
	if !ok {
		set = contract.NewSignalSet()
		e.evalCollectors[runID] = set
	}
	return set
}

// recordStepEval emits step-level signals for a terminal step state.
// stepState must be one of stateCompleted / stateCompletedEmpty / stateFailed —
// other states are no-ops by contract (skipped / reworking emit no signal).
// stepErr is the terminal error (nil on success). stepDuration is the
// wall-clock duration of the step (across all attempts).
func (e *DefaultPipelineExecutor) recordStepEval(execution *PipelineExecution, step *Step, stepState string, stepErr error, stepDuration time.Duration) {
	if execution == nil || step == nil {
		return
	}
	runID := execution.Status.ID
	set := e.signalSetFor(runID)
	if set == nil {
		return
	}

	now := time.Now()
	switch stepState {
	case stateCompleted, stateCompletedEmpty:
		set.Add(contract.Signal{Kind: contract.SignalSuccess, StepID: step.ID, Timestamp: now})
	case stateFailed:
		kind := contract.SignalFailure
		if stepErr != nil && isContractFailure(stepErr) {
			kind = contract.SignalContractFailure
		}
		set.Add(contract.Signal{Kind: kind, StepID: step.ID, Timestamp: now})
	default:
		// stateSkipped / stateReworking / stateRejected: no-op
		return
	}

	if stepDuration > 0 {
		set.Add(contract.Signal{
			Kind:      contract.SignalDuration,
			StepID:    step.ID,
			Value:     float64(stepDuration.Milliseconds()),
			Timestamp: now,
		})
	}
}

// recordStepRetry bumps the retry counter on the per-run SignalSet.
func (e *DefaultPipelineExecutor) recordStepRetry(execution *PipelineExecution) {
	if execution == nil {
		return
	}
	if set := e.signalSetFor(execution.Status.ID); set != nil {
		set.RecordRetry()
	}
}

// recordPipelineEval aggregates the per-run SignalSet and persists one
// PipelineEvalRecord. Called from finalizePipelineExecution after the final
// SavePipelineState, before terminal hooks. Persistence is best-effort: the
// (pipeline_name, run_id) primary key on pipeline_eval can reject duplicate
// inserts on resume, which we swallow.
func (e *DefaultPipelineExecutor) recordPipelineEval(execution *PipelineExecution) {
	if execution == nil || execution.Status == nil {
		return
	}
	if execution.Pipeline == nil {
		return
	}

	evolStore, ok := e.store.(state.EvolutionStore)
	if !ok || evolStore == nil {
		return
	}

	runID := execution.Status.ID
	set := e.signalSetFor(runID)
	if set == nil {
		return
	}

	pipelineName := execution.Pipeline.Metadata.Name
	rec := set.Aggregate(runID, pipelineName, execution.Status.StartedAt)

	if e.costLedger != nil {
		if total := e.costLedger.TotalCost(); total > 0 {
			rec.CostDollars = &total
		}
	}

	if err := evolStore.RecordEval(rec); err != nil {
		if isUniqueConstraintErr(err) {
			// Resume safety: a previous run already wrote this row. Log and
			// move on — duplicate eval rows are not a runtime failure.
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: runID,
				State:      "warning",
				Message:    fmt.Sprintf("eval signal: duplicate row for %s/%s — skipped", pipelineName, runID),
			})
			return
		}
		// Non-fatal: emit a warning but don't fail the pipeline.
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: runID,
			State:      "warning",
			Message:    fmt.Sprintf("eval signal: RecordEval failed: %v", err),
		})
		return
	}

	// Phase 3.3: consult the evolution trigger now that the row has landed.
	// Best-effort — errors warn, never block finalize.
	e.maybeEmitEvolutionProposed(runID, pipelineName)
}

// maybeEmitEvolutionProposed runs the configured EvolutionTrigger (if any)
// and emits an advisory event when ShouldEvolve fires. Trigger errors emit
// a warning but never bubble up. Nil trigger is a no-op.
func (e *DefaultPipelineExecutor) maybeEmitEvolutionProposed(runID, pipelineName string) {
	if e.evolutionTrigger == nil {
		return
	}
	fire, reason, err := e.evolutionTrigger.ShouldEvolve(pipelineName)
	if err != nil {
		e.emit(event.Event{
			Timestamp:  time.Now(),
			PipelineID: runID,
			State:      "warning",
			Message:    fmt.Sprintf("evolution trigger: %v", err),
		})
		return
	}
	if !fire {
		return
	}
	msg := pipelineName
	if reason != "" {
		msg = fmt.Sprintf("%s: %s", pipelineName, reason)
	}
	e.emit(event.Event{
		Timestamp:  time.Now(),
		PipelineID: runID,
		State:      "evolution_proposed",
		Message:    msg,
	})
}

// isContractFailure reports whether err originates from the contract validator
// path. Used to map a generic step failure to SignalContractFailure when the
// proximate cause was a contract check, not a runtime error.
//
// The contract validator wraps failures with the literal prefix
// "contract validation failed" — see internal/pipeline/executor_contract.go
// applyContractOnFailure. String matching is sufficient and avoids exporting
// a sentinel from contract internals into this package.
func isContractFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "contract validation failed") ||
		strings.Contains(msg, "contract failed")
}

// isUniqueConstraintErr reports whether err is a SQLite UNIQUE / PRIMARY KEY
// constraint violation. Resume can re-fire the post-run hook for a run that
// already has a pipeline_eval row; in that case the second insert returns
// "UNIQUE constraint failed: pipeline_eval.pipeline_name, run_id" which is
// the safe-to-swallow case.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "PRIMARY KEY constraint failed") ||
		strings.Contains(msg, "constraint failed: pipeline_eval")
}


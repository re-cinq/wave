// Package contract — eval_signal.go defines the typed signals that the pipeline
// executor's post-run hook (issue #1606, Phase 3.1 of evolution-loop epic #1565)
// emits per step and aggregates per run before persisting via
// state.EvolutionStore.RecordEval.
//
// SignalKind enumerates the recognised producers (success, failure,
// contract_failure, judge_score, duration, cost). Signal carries a single
// observation. SignalSet aggregates observations across a run and produces a
// state.PipelineEvalRecord.
package contract

import (
	"sync"
	"time"

	"github.com/recinq/wave/internal/state"
)

// SignalKind is the discriminator for an EvalSignal observation.
type SignalKind string

// SignalKind constants — string values are the wire form persisted alongside
// pipeline_eval rows for downstream evolution analysis.
const (
	SignalSuccess         SignalKind = "success"
	SignalFailure         SignalKind = "failure"
	SignalContractFailure SignalKind = "contract_failure"
	SignalJudgeScore      SignalKind = "judge_score"
	SignalDuration        SignalKind = "duration"
	SignalCost            SignalKind = "cost"
)

// Signal is a single typed observation produced during pipeline execution.
// Step-level signals carry StepID; pipeline-level aggregates leave it empty.
//
// Value carries the numeric payload for kinds that have one (judge_score,
// duration_ms, cost_dollars). For boolean-shaped kinds it is unused.
type Signal struct {
	Kind      SignalKind
	StepID    string
	Value     float64
	Timestamp time.Time
}

// SignalSet is a per-run aggregator: step hooks Add() observations, then the
// pipeline-finalize hook calls Aggregate() to derive a PipelineEvalRecord for
// state.EvolutionStore.RecordEval. Safe for concurrent use.
type SignalSet struct {
	mu      sync.Mutex
	signals []Signal
	// retryCount accumulates from RecordRetry; not derivable from signal kinds
	// alone because retried-then-succeeded steps emit only a single success.
	retryCount int
}

// NewSignalSet returns an empty SignalSet ready for Add().
func NewSignalSet() *SignalSet {
	return &SignalSet{}
}

// Add records one signal. Concurrency-safe; safe to call from goroutines that
// run parallel steps.
func (s *SignalSet) Add(sig Signal) {
	if sig.Timestamp.IsZero() {
		sig.Timestamp = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signals = append(s.signals, sig)
}

// RecordRetry increments the retry counter. Step-level retry decisions are
// already known to the executor, so we don't reconstruct them from signals.
func (s *SignalSet) RecordRetry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retryCount++
}

// Len returns the number of signals collected. Useful for tests.
func (s *SignalSet) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.signals)
}

// FailureClass derives the dominant failure class with priority
// contract_failure > failure > "" (success / empty). Pipeline-level signals
// (StepID == "") and step-level signals are treated identically — any
// contract_failure or failure anywhere in the run promotes the class.
func (s *SignalSet) FailureClass() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	hasFailure := false
	for _, sig := range s.signals {
		if sig.Kind == SignalContractFailure {
			return "contract_failure"
		}
		if sig.Kind == SignalFailure {
			hasFailure = true
		}
	}
	if hasFailure {
		return "failure"
	}
	return ""
}

// Aggregate folds the collected signals into a PipelineEvalRecord ready for
// state.EvolutionStore.RecordEval. Caller passes runID, pipelineName, and the
// run's start time so DurationMs is computed against a consistent clock.
//
// Field rules:
//   - JudgeScore   — average of all SignalJudgeScore values; nil when none.
//   - ContractPass — true when no contract_failure signal exists, AND at
//     least one judge / contract / success signal proved the run actually
//     ran. nil when the set is empty (no observations to assert about).
//   - RetryCount   — the integer accumulated via RecordRetry; nil when zero.
//   - FailureClass — see FailureClass().
//   - DurationMs   — time.Since(startedAt) in milliseconds; nil when zero.
//   - CostDollars  — caller-supplied via WithCost (left nil otherwise).
func (s *SignalSet) Aggregate(runID, pipelineName string, startedAt time.Time) state.PipelineEvalRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec := state.PipelineEvalRecord{
		PipelineName: pipelineName,
		RunID:        runID,
		FailureClass: s.failureClassLocked(),
		RecordedAt:   time.Now(),
	}

	// Judge score average
	var judgeSum float64
	var judgeCount int
	hasObservation := false
	hasFailure := false
	for _, sig := range s.signals {
		switch sig.Kind {
		case SignalJudgeScore:
			judgeSum += sig.Value
			judgeCount++
			hasObservation = true
		case SignalContractFailure, SignalFailure:
			hasFailure = true
			hasObservation = true
		case SignalSuccess:
			hasObservation = true
		}
	}
	if judgeCount > 0 {
		avg := judgeSum / float64(judgeCount)
		rec.JudgeScore = &avg
	}

	// ContractPass = "did the run pass" — true when no failure signals fired,
	// false when any failure (contract or step) occurred. Nil only when the
	// set is empty, meaning we have no observations to assert about.
	if hasObservation {
		pass := !hasFailure
		rec.ContractPass = &pass
	}

	if s.retryCount > 0 {
		rc := s.retryCount
		rec.RetryCount = &rc
	}

	if !startedAt.IsZero() {
		ms := time.Since(startedAt).Milliseconds()
		if ms > 0 {
			rec.DurationMs = &ms
		}
	}

	return rec
}

// failureClassLocked is FailureClass without the mutex (caller already holds).
func (s *SignalSet) failureClassLocked() string {
	hasFailure := false
	for _, sig := range s.signals {
		if sig.Kind == SignalContractFailure {
			return "contract_failure"
		}
		if sig.Kind == SignalFailure {
			hasFailure = true
		}
	}
	if hasFailure {
		return "failure"
	}
	return ""
}

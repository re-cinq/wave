package evolution

import (
	"fmt"
	"sort"
	"time"

	"github.com/recinq/wave/internal/state"
)

// Store is the slim subset of state.EvolutionStore that Service needs.
// Defining it here keeps the package importable from tests with a fake
// store and avoids a hard coupling to the full aggregate interface.
type Store interface {
	GetEvalsForPipeline(pipelineName string, limit int) ([]state.PipelineEvalRecord, error)
	LastProposalAt(pipelineName string) (time.Time, bool, error)
}

// Service decides whether a pipeline's accumulated eval rows merit a
// pipeline-evolve run. It is created per-process by the CLI and reused
// across pipeline runs.
type Service struct {
	store Store
	cfg   Config
}

// NewService constructs a Service. A nil store yields a usable zero-value
// service whose ShouldEvolve always returns (false, "", nil) — convenient
// for tests and for environments where evolution telemetry is opt-out.
func NewService(store Store, cfg Config) *Service {
	return &Service{store: store, cfg: cfg}
}

// ShouldEvolve evaluates the three configured heuristics in order
// (every-N → drift → retry) and returns on the first match. It returns
// (false, "", nil) when no heuristic fires, when evolution is disabled,
// or when the service has no store.
//
// Heuristic precedence is by definition order, not severity. Phase 3.4
// will replace the first-match return with a richer SignalSummary.
func (s *Service) ShouldEvolve(pipelineName string) (bool, string, error) {
	if s == nil || s.store == nil {
		return false, "", nil
	}
	if !s.cfg.Enabled {
		return false, "", nil
	}
	if pipelineName == "" {
		return false, "", nil
	}

	limit := s.cfg.maxWindow()
	if limit <= 0 {
		return false, "", nil
	}

	evals, err := s.store.GetEvalsForPipeline(pipelineName, limit)
	if err != nil {
		return false, "", fmt.Errorf("evolution: load evals for %s: %w", pipelineName, err)
	}
	if len(evals) == 0 {
		return false, "", nil
	}

	lastProposal, _, err := s.store.LastProposalAt(pipelineName)
	if err != nil {
		return false, "", fmt.Errorf("evolution: last proposal for %s: %w", pipelineName, err)
	}

	if fire, reason := everyNJudgeDrop(evals, lastProposal, s.cfg); fire {
		return true, reason, nil
	}
	if fire, reason := contractPassDrift(evals, s.cfg); fire {
		return true, reason, nil
	}
	if fire, reason := retryRateSpike(evals, s.cfg); fire {
		return true, reason, nil
	}
	return false, "", nil
}

// everyNJudgeDrop fires when at least 2*EveryNWindow scored evals exist since
// the last proposal AND the median judge_score across the more-recent half
// has dropped by ≥EveryNJudgeDrop versus the older half.
//
// evals is expected newest-first (the order returned by GetEvalsForPipeline).
func everyNJudgeDrop(evals []state.PipelineEvalRecord, lastProposal time.Time, cfg Config) (bool, string) {
	if cfg.EveryNWindow <= 0 || cfg.EveryNJudgeDrop <= 0 {
		return false, ""
	}

	// Filter to scored rows recorded after the last proposal. Empty
	// lastProposal → all scored rows count.
	scored := make([]float64, 0, len(evals))
	for _, e := range evals {
		if e.JudgeScore == nil {
			continue
		}
		if !lastProposal.IsZero() && !e.RecordedAt.After(lastProposal) {
			continue
		}
		scored = append(scored, *e.JudgeScore)
	}

	need := cfg.EveryNWindow * 2
	if len(scored) < need {
		return false, ""
	}

	// scored is newest-first; the leading EveryNWindow rows are the recent
	// window, the next EveryNWindow are the prior window.
	recent := scored[:cfg.EveryNWindow]
	prior := scored[cfg.EveryNWindow:need]

	recentMed := medianFloats(recent)
	priorMed := medianFloats(prior)
	drop := priorMed - recentMed
	if drop >= cfg.EveryNJudgeDrop {
		return true, fmt.Sprintf(
			"every-N: median judge_score dropped %.3f over %d evals (was %.3f, now %.3f)",
			drop, cfg.EveryNWindow, priorMed, recentMed,
		)
	}
	return false, ""
}

// contractPassDrift fires when the contract_pass rate over the most recent
// DriftWindow rows has dropped by more than DriftPassDrop versus the prior
// DriftWindow rows. Both halves require DriftWindow rows with a non-nil
// ContractPass field; insufficient data → no fire.
func contractPassDrift(evals []state.PipelineEvalRecord, cfg Config) (bool, string) {
	if cfg.DriftWindow <= 0 || cfg.DriftPassDrop <= 0 {
		return false, ""
	}
	w := cfg.DriftWindow
	withPass := make([]state.PipelineEvalRecord, 0, len(evals))
	for _, e := range evals {
		if e.ContractPass != nil {
			withPass = append(withPass, e)
		}
	}
	if len(withPass) < 2*w {
		return false, ""
	}
	recent := passRate(withPass[:w])
	prior := passRate(withPass[w : 2*w])
	drop := prior - recent
	if drop > cfg.DriftPassDrop {
		return true, fmt.Sprintf(
			"drift: contract_pass rate dropped %.1f%% over %d evals (was %.1f%%, now %.1f%%)",
			drop*100, w, prior*100, recent*100,
		)
	}
	return false, ""
}

// retryRateSpike fires when the average retry_count over the most recent
// RetryWindow rows exceeds RetryAvgThreshold. Rows without a retry_count
// are treated as zero retries.
func retryRateSpike(evals []state.PipelineEvalRecord, cfg Config) (bool, string) {
	if cfg.RetryWindow <= 0 || cfg.RetryAvgThreshold <= 0 {
		return false, ""
	}
	w := cfg.RetryWindow
	if len(evals) < w {
		return false, ""
	}
	avg := avgRetry(evals[:w])
	if avg > cfg.RetryAvgThreshold {
		return true, fmt.Sprintf(
			"retry-rate: avg retry_count %.2f over %d evals exceeds %.2f",
			avg, w, cfg.RetryAvgThreshold,
		)
	}
	return false, ""
}

// medianFloats returns the median of a slice of float64 values. The input
// is copied before sorting so the caller's order is preserved. Empty
// input returns 0.
func medianFloats(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := make([]float64, len(xs))
	copy(cp, xs)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

// passRate returns the fraction of rows where ContractPass is non-nil and
// true. Rows with nil ContractPass are counted as failures — callers that
// want a strict pass/total ratio should pre-filter, as contractPassDrift does.
func passRate(rows []state.PipelineEvalRecord) float64 {
	if len(rows) == 0 {
		return 0
	}
	pass := 0
	for _, r := range rows {
		if r.ContractPass != nil && *r.ContractPass {
			pass++
		}
	}
	return float64(pass) / float64(len(rows))
}

// avgRetry returns the average retry_count across rows. Nil retry counts
// count as 0.
func avgRetry(rows []state.PipelineEvalRecord) float64 {
	if len(rows) == 0 {
		return 0
	}
	sum := 0
	for _, r := range rows {
		if r.RetryCount != nil {
			sum += *r.RetryCount
		}
	}
	return float64(sum) / float64(len(rows))
}

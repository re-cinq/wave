package retro

import (
	"fmt"
	"time"

	"github.com/recinq/wave/internal/state"
)

// Collector gathers quantitative execution data from the state store
// and assembles it into a Retrospective.
type Collector struct {
	store state.StateStore
}

// NewCollector creates a Collector backed by the given state store.
func NewCollector(store state.StateStore) *Collector {
	return &Collector{store: store}
}

// Collect builds a Retrospective for the given run by querying step
// states, step attempts, and performance metrics from the state store.
// The returned Retrospective has Quantitative data filled in and
// Narrative set to nil.
func (c *Collector) Collect(runID string, pipelineName string) (*Retrospective, error) {
	if c.store == nil {
		return nil, fmt.Errorf("collector state store is nil")
	}

	// 1. Get step states for the run.
	stepStates, err := c.store.GetStepStates(runID)
	if err != nil {
		return nil, fmt.Errorf("get step states: %w", err)
	}

	// 2. Get performance metrics for all steps in the run.
	metrics, err := c.store.GetPerformanceMetrics(runID, "")
	if err != nil {
		return nil, fmt.Errorf("get performance metrics: %w", err)
	}

	// Index performance metrics by step ID for fast lookup.
	metricsByStep := make(map[string]state.PerformanceMetricRecord, len(metrics))
	for _, m := range metrics {
		metricsByStep[m.StepID] = m
	}

	// 3. Build per-step metrics.
	var (
		steps        []StepMetrics
		totalDur     int64
		successCount int
		failureCount int
		totalRetries int
	)

	for _, ss := range stepStates {
		// Get retry count from step attempts.
		attempts, err := c.store.GetStepAttempts(runID, ss.StepID)
		if err != nil {
			return nil, fmt.Errorf("get step attempts for %s: %w", ss.StepID, err)
		}
		retries := len(attempts) - 1
		if retries < 0 {
			retries = 0
		}
		totalRetries += retries

		// Determine status string.
		metric, hasMetric := metricsByStep[ss.StepID]
		status := stepStatus(ss.State, hasMetric, metric.Success)

		if status == "success" {
			successCount++
		} else {
			failureCount++
		}

		// Extract timing and token data from performance metric.
		var durationMs int64
		var tokensUsed int
		var persona string
		var exitCode int
		if hasMetric {
			durationMs = metric.DurationMs
			tokensUsed = metric.TokensUsed
			persona = metric.Persona
			if !metric.Success {
				exitCode = 1
			}
		}
		totalDur += durationMs

		sm := StepMetrics{
			Name:       ss.StepID,
			DurationMs: durationMs,
			Retries:    retries,
			Status:     status,
			Adapter:    "", // not tracked in performance metrics
			Model:      "", // not tracked in performance metrics
			ExitCode:   exitCode,
			TokensUsed: tokensUsed,
		}
		// Use persona as a proxy for adapter when no dedicated field exists.
		if persona != "" {
			sm.Adapter = persona
		}

		steps = append(steps, sm)
	}

	retro := &Retrospective{
		RunID:    runID,
		Pipeline: pipelineName,
		Quantitative: QuantitativeData{
			TotalDurationMs: totalDur,
			TotalSteps:      len(stepStates),
			SuccessCount:    successCount,
			FailureCount:    failureCount,
			TotalRetries:    totalRetries,
			Steps:           steps,
		},
		Timestamp: time.Now(),
		Narrative: nil,
	}

	return retro, nil
}

// stepStatus derives a human-readable status string.
func stepStatus(ss state.StepState, hasMetric bool, metricSuccess bool) string {
	// If we have a performance metric, prefer its success flag.
	if hasMetric {
		if metricSuccess {
			return "success"
		}
		return "failed"
	}
	// Fall back to step state.
	switch ss {
	case state.StateCompleted:
		return "success"
	case state.StateFailed:
		return "failed"
	case state.StateSkipped:
		return "skipped"
	default:
		return string(ss)
	}
}

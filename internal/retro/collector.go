package retro

import (
	"fmt"
	"log"
	"time"

	"github.com/recinq/wave/internal/state"
)

// StateQuerier is the subset of state.RunStore needed by the Collector.
type StateQuerier interface {
	GetRun(runID string) (*state.RunRecord, error)
	GetPerformanceMetrics(runID string, stepID string) ([]state.PerformanceMetricRecord, error)
	GetStepAttempts(runID string, stepID string) ([]state.StepAttemptRecord, error)
}

// Collector builds quantitative retrospective data from the state store.
type Collector struct {
	store StateQuerier
}

// NewCollector creates a new Collector.
func NewCollector(store StateQuerier) *Collector {
	return &Collector{store: store}
}

// Collect gathers quantitative data for a completed pipeline run.
func (c *Collector) Collect(runID string) (*QuantitativeData, error) {
	run, err := c.store.GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run %s: %w", runID, err)
	}

	// Calculate total duration
	var totalDurationMs int64
	if run.CompletedAt != nil {
		totalDurationMs = run.CompletedAt.Sub(run.StartedAt).Milliseconds()
	} else {
		totalDurationMs = time.Since(run.StartedAt).Milliseconds()
	}

	// Get performance metrics for all steps
	metrics, err := c.store.GetPerformanceMetrics(runID, "")
	if err != nil {
		log.Printf("[retro] warning: failed to get performance metrics for run %s: %v", runID, err)
		metrics = nil
	}

	// Build per-step metrics
	var steps []StepMetrics
	var successCount, failureCount, totalRetries, totalTokens int

	// Track unique steps (metrics may have multiple entries per step due to retries)
	stepMap := make(map[string]*StepMetrics)
	for _, m := range metrics {
		sm, ok := stepMap[m.StepID]
		if !ok {
			status := "success"
			if !m.Success {
				status = "failed"
			}
			sm = &StepMetrics{
				Name:         m.StepID,
				DurationMs:   m.DurationMs,
				Status:       status,
				Adapter:      m.Persona, // persona is the closest to adapter info we have
				FilesChanged: m.FilesModified,
				TokensUsed:   m.TokensUsed,
			}
			stepMap[m.StepID] = sm
		} else {
			// Accumulate for retries — keep the last status
			sm.DurationMs += m.DurationMs
			sm.TokensUsed += m.TokensUsed
			sm.FilesChanged += m.FilesModified
			if m.Success {
				sm.Status = "success"
			} else {
				sm.Status = "failed"
			}
		}
		totalTokens += m.TokensUsed
	}

	// Count retries per step from step_attempt
	for stepID, sm := range stepMap {
		attempts, err := c.store.GetStepAttempts(runID, stepID)
		if err == nil && len(attempts) > 1 {
			sm.Retries = len(attempts) - 1
			totalRetries += sm.Retries
		}
	}

	// Convert map to slice and count successes/failures
	for _, sm := range stepMap {
		if sm.Status == "success" {
			successCount++
		} else {
			failureCount++
		}
		steps = append(steps, *sm)
	}

	// If no performance metrics exist, use run-level data
	if len(steps) == 0 {
		totalTokens = run.TotalTokens
	}

	return &QuantitativeData{
		TotalDurationMs: totalDurationMs,
		TotalSteps:      len(steps),
		SuccessCount:    successCount,
		FailureCount:    failureCount,
		TotalRetries:    totalRetries,
		TotalTokens:     totalTokens,
		Steps:           steps,
	}, nil
}

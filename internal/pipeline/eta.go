package pipeline

import (
	"sync"
	"time"

	"github.com/recinq/wave/internal/state"
)

// ETACalculator estimates remaining pipeline time using historical step durations
// from the state store combined with actual durations from the current run.
// It is safe for concurrent use.
type ETACalculator struct {
	mu               sync.Mutex
	historicalAvg    map[string]int64 // stepID -> historical average duration (ms)
	currentDurations map[string]int64 // stepID -> actual duration from current run (ms)
	completedSteps   map[string]bool  // stepID -> true if completed in current run
	stepOrder        []string         // ordered step IDs
}

// NewETACalculator creates an ETACalculator by querying historical step performance
// from the state store. If store is nil or no historical data exists, the calculator
// gracefully degrades — returning 0 for all estimates.
func NewETACalculator(store state.StateStore, pipelineName string, stepIDs []string) *ETACalculator {
	calc := &ETACalculator{
		historicalAvg:    make(map[string]int64, len(stepIDs)),
		currentDurations: make(map[string]int64),
		completedSteps:   make(map[string]bool),
		stepOrder:        stepIDs,
	}

	if store == nil {
		return calc
	}

	// Query historical averages for each step. Use all-time data (since epoch).
	for _, stepID := range stepIDs {
		stats, err := store.GetStepPerformanceStats(pipelineName, stepID, time.Time{})
		if err != nil || stats == nil {
			continue
		}
		if stats.AvgDurationMs > 0 {
			calc.historicalAvg[stepID] = stats.AvgDurationMs
		}
	}

	return calc
}

// RecordStepCompletion records the actual duration of a completed step in the current run.
func (c *ETACalculator) RecordStepCompletion(stepID string, durationMs int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.completedSteps[stepID] = true
	c.currentDurations[stepID] = durationMs
}

// RemainingMs returns the estimated remaining time in milliseconds for all
// incomplete steps. For each incomplete step, it uses the current-run actual
// duration if the step already ran (e.g., for ETA recalculation), otherwise
// the historical average. Returns 0 if no estimates are available.
func (c *ETACalculator) RemainingMs() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	var remaining int64
	for _, stepID := range c.stepOrder {
		if c.completedSteps[stepID] {
			continue
		}
		// Use historical average for incomplete steps
		if avg, ok := c.historicalAvg[stepID]; ok {
			remaining += avg
		}
	}
	return remaining
}

// AverageStepMs returns the average duration across all steps that have data
// (either from the current run or historical). Returns 0 if no data.
func (c *ETACalculator) AverageStepMs() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	var total int64
	var count int64

	for _, stepID := range c.stepOrder {
		// Prefer current-run actual duration
		if dur, ok := c.currentDurations[stepID]; ok {
			total += dur
			count++
			continue
		}
		// Fall back to historical average
		if avg, ok := c.historicalAvg[stepID]; ok {
			total += avg
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / count
}

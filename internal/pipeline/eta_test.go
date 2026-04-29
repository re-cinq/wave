package pipeline

import (
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/metrics"
)

// mockStoreForETA satisfies the statsLookup surface NewETACalculator needs.
// It returns preconfigured performance stats per step.
type mockStoreForETA struct {
	stats map[string]*metrics.StepPerformanceStats
}

func (m *mockStoreForETA) GetStepPerformanceStats(_ string, stepID string, _ time.Time) (*metrics.StepPerformanceStats, error) {
	if s, ok := m.stats[stepID]; ok {
		return s, nil
	}
	return &metrics.StepPerformanceStats{StepID: stepID}, nil
}

func TestETACalculator_NoHistory(t *testing.T) {
	calc := NewETACalculator(nil, "test-pipeline", []string{"step-1", "step-2", "step-3"})

	if got := calc.RemainingMs(); got != 0 {
		t.Errorf("RemainingMs() with no history = %d, want 0", got)
	}
	if got := calc.AverageStepMs(); got != 0 {
		t.Errorf("AverageStepMs() with no history = %d, want 0", got)
	}
}

func TestETACalculator_WithHistory(t *testing.T) {
	store := &mockStoreForETA{
		stats: map[string]*metrics.StepPerformanceStats{
			"step-1": {StepID: "step-1", AvgDurationMs: 10000},
			"step-2": {StepID: "step-2", AvgDurationMs: 20000},
			"step-3": {StepID: "step-3", AvgDurationMs: 30000},
		},
	}

	calc := NewETACalculator(store, "test-pipeline", []string{"step-1", "step-2", "step-3"})

	// All steps pending — remaining = sum of all averages
	if got := calc.RemainingMs(); got != 60000 {
		t.Errorf("RemainingMs() all pending = %d, want 60000", got)
	}

	// Average across all 3 steps
	if got := calc.AverageStepMs(); got != 20000 {
		t.Errorf("AverageStepMs() = %d, want 20000", got)
	}
}

func TestETACalculator_StepCompletionReducesRemaining(t *testing.T) {
	store := &mockStoreForETA{
		stats: map[string]*metrics.StepPerformanceStats{
			"step-1": {StepID: "step-1", AvgDurationMs: 10000},
			"step-2": {StepID: "step-2", AvgDurationMs: 20000},
			"step-3": {StepID: "step-3", AvgDurationMs: 30000},
		},
	}

	calc := NewETACalculator(store, "test-pipeline", []string{"step-1", "step-2", "step-3"})

	// Complete step-1 with actual duration
	calc.RecordStepCompletion("step-1", 12000)

	// Remaining should be step-2 + step-3 historical averages
	if got := calc.RemainingMs(); got != 50000 {
		t.Errorf("RemainingMs() after step-1 complete = %d, want 50000", got)
	}

	// Average should now include actual duration for step-1
	// (12000 + 20000 + 30000) / 3 = 20666
	if got := calc.AverageStepMs(); got != 20666 {
		t.Errorf("AverageStepMs() after step-1 complete = %d, want 20666", got)
	}

	// Complete step-2
	calc.RecordStepCompletion("step-2", 18000)

	// Remaining should be only step-3 historical average
	if got := calc.RemainingMs(); got != 30000 {
		t.Errorf("RemainingMs() after step-2 complete = %d, want 30000", got)
	}

	// Complete all steps
	calc.RecordStepCompletion("step-3", 25000)

	if got := calc.RemainingMs(); got != 0 {
		t.Errorf("RemainingMs() all complete = %d, want 0", got)
	}
}

func TestETACalculator_PartialHistory(t *testing.T) {
	// Only step-2 has historical data
	store := &mockStoreForETA{
		stats: map[string]*metrics.StepPerformanceStats{
			"step-2": {StepID: "step-2", AvgDurationMs: 15000},
		},
	}

	calc := NewETACalculator(store, "test-pipeline", []string{"step-1", "step-2", "step-3"})

	// Only step-2 has an estimate
	if got := calc.RemainingMs(); got != 15000 {
		t.Errorf("RemainingMs() partial history = %d, want 15000", got)
	}

	// Average only counts steps with data
	if got := calc.AverageStepMs(); got != 15000 {
		t.Errorf("AverageStepMs() partial history = %d, want 15000", got)
	}
}

func TestETACalculator_SingleStep(t *testing.T) {
	store := &mockStoreForETA{
		stats: map[string]*metrics.StepPerformanceStats{
			"only-step": {StepID: "only-step", AvgDurationMs: 5000},
		},
	}

	calc := NewETACalculator(store, "test-pipeline", []string{"only-step"})

	if got := calc.RemainingMs(); got != 5000 {
		t.Errorf("RemainingMs() single step = %d, want 5000", got)
	}

	calc.RecordStepCompletion("only-step", 4500)

	if got := calc.RemainingMs(); got != 0 {
		t.Errorf("RemainingMs() after completion = %d, want 0", got)
	}
}

func TestETACalculator_ConcurrentAccess(t *testing.T) {
	store := &mockStoreForETA{
		stats: map[string]*metrics.StepPerformanceStats{
			"step-1": {StepID: "step-1", AvgDurationMs: 10000},
			"step-2": {StepID: "step-2", AvgDurationMs: 20000},
			"step-3": {StepID: "step-3", AvgDurationMs: 30000},
		},
	}

	calc := NewETACalculator(store, "test-pipeline", []string{"step-1", "step-2", "step-3"})

	var wg sync.WaitGroup
	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			stepID := []string{"step-1", "step-2", "step-3"}[n%3]
			calc.RecordStepCompletion(stepID, int64(n*1000))
		}(i)
	}
	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			calc.RemainingMs()
			calc.AverageStepMs()
		}()
	}
	wg.Wait()

	// No race condition detected is the test — the race detector will catch issues
}

func TestETACalculator_NilStore(t *testing.T) {
	calc := NewETACalculator(nil, "test-pipeline", []string{"step-1", "step-2"})

	if got := calc.RemainingMs(); got != 0 {
		t.Errorf("RemainingMs() nil store = %d, want 0", got)
	}
	if got := calc.AverageStepMs(); got != 0 {
		t.Errorf("AverageStepMs() nil store = %d, want 0", got)
	}

	// Should not panic on completion recording
	calc.RecordStepCompletion("step-1", 5000)

	if got := calc.RemainingMs(); got != 0 {
		t.Errorf("RemainingMs() after completion with nil store = %d, want 0", got)
	}
}

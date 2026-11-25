package pipeline

import (
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

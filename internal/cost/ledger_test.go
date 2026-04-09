package cost

import (
	"strings"
	"testing"
)

func TestLookupPricing(t *testing.T) {
	tests := []struct {
		model   string
		wantIn  float64
		wantOut float64
	}{
		{"claude-opus", 15.0, 75.0},
		{"claude-sonnet", 3.0, 15.0},
		{"claude-haiku", 0.25, 1.25},
		{"claude-opus-4-6", 15.0, 75.0},  // prefix match
		{"claude-sonnet-4-6", 3.0, 15.0}, // prefix match
		{"claude-haiku-4-5", 0.25, 1.25}, // prefix match
		{"gpt-4o", 2.5, 10.0},
		{"gpt-4o-mini", 0.15, 0.6},
		{"unknown-model", 0, 0},
		{"", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			p := LookupPricing(tt.model)
			if p.InputPerMillion != tt.wantIn {
				t.Errorf("InputPerMillion = %v, want %v", p.InputPerMillion, tt.wantIn)
			}
			if p.OutputPerMillion != tt.wantOut {
				t.Errorf("OutputPerMillion = %v, want %v", p.OutputPerMillion, tt.wantOut)
			}
		})
	}
}

func TestComputeCost(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantCost     float64
		tolerance    float64
	}{
		{
			name:         "opus 1M input + 100k output",
			model:        "claude-opus",
			inputTokens:  1_000_000,
			outputTokens: 100_000,
			wantCost:     15.0 + 7.5, // $15 input + $7.50 output
			tolerance:    0.01,
		},
		{
			name:         "haiku small usage",
			model:        "claude-haiku",
			inputTokens:  10_000,
			outputTokens: 1_000,
			wantCost:     0.0025 + 0.00125,
			tolerance:    0.0001,
		},
		{
			name:         "unknown model returns zero",
			model:        "unknown",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantCost:     0,
			tolerance:    0,
		},
		{
			name:         "zero tokens",
			model:        "claude-opus",
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0,
			tolerance:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCost(tt.model, tt.inputTokens, tt.outputTokens)
			diff := got - tt.wantCost
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("ComputeCost() = %v, want %v (tolerance %v)", got, tt.wantCost, tt.tolerance)
			}
		})
	}
}

func TestLedger_Record(t *testing.T) {
	l := NewLedger(0, 0) // no budget
	entry, status := l.Record("run-1", "step-1", "claude-haiku", 100_000, 10_000, 110_000)

	if status != BudgetOK {
		t.Errorf("expected BudgetOK, got %d", status)
	}
	if entry.RunID != "run-1" {
		t.Errorf("entry.RunID = %q", entry.RunID)
	}
	if entry.Cost <= 0 {
		t.Errorf("expected positive cost, got %v", entry.Cost)
	}
	if l.TotalCost() != entry.Cost {
		t.Errorf("TotalCost = %v, want %v", l.TotalCost(), entry.Cost)
	}
}

func TestLedger_BudgetExceeded(t *testing.T) {
	l := NewLedger(0.01, 0) // very low ceiling

	// First call with opus should exceed immediately
	_, status := l.Record("run-1", "step-1", "claude-opus", 100_000, 10_000, 110_000)
	if status != BudgetExceeded {
		t.Errorf("expected BudgetExceeded, got %d (total: $%.4f)", status, l.TotalCost())
	}
}

func TestLedger_BudgetWarning(t *testing.T) {
	l := NewLedger(100.0, 0.001) // warn at $0.001

	// haiku with small tokens
	_, status := l.Record("run-1", "step-1", "claude-haiku", 10_000, 1_000, 11_000)
	if status != BudgetWarning {
		t.Errorf("expected BudgetWarning, got %d (total: $%.6f)", status, l.TotalCost())
	}

	// Second call should be OK (warning already fired)
	_, status2 := l.Record("run-1", "step-2", "claude-haiku", 10_000, 1_000, 11_000)
	if status2 != BudgetOK {
		t.Errorf("expected BudgetOK after warning, got %d", status2)
	}
}

func TestLedger_Entries(t *testing.T) {
	l := NewLedger(0, 0)
	l.Record("run-1", "step-1", "claude-opus", 1000, 100, 1100)
	l.Record("run-1", "step-2", "claude-haiku", 500, 50, 550)

	entries := l.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].StepID != "step-1" {
		t.Errorf("first entry step = %q", entries[0].StepID)
	}
	if entries[1].StepID != "step-2" {
		t.Errorf("second entry step = %q", entries[1].StepID)
	}
}

func TestLedger_Summary(t *testing.T) {
	l := NewLedger(50.0, 0)
	l.Record("run-1", "step-1", "claude-opus", 100_000, 10_000, 110_000)

	summary := l.Summary()
	if !strings.Contains(summary, "Cost: $") {
		t.Errorf("expected cost in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "budget") {
		t.Errorf("expected budget info in summary, got: %s", summary)
	}
}

func TestLedger_EmptySummary(t *testing.T) {
	l := NewLedger(0, 0)
	summary := l.Summary()
	if summary != "No cost data recorded" {
		t.Errorf("expected empty summary, got: %s", summary)
	}
}

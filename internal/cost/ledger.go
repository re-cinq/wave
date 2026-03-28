// Package cost provides token cost tracking and budget enforcement for pipeline runs.
package cost

import (
	"fmt"
	"strings"
	"sync"
)

// ModelPricing defines per-token costs for a model in USD.
type ModelPricing struct {
	InputPerMillion  float64 // cost per million input tokens
	OutputPerMillion float64 // cost per million output tokens
}

// DefaultPricing contains known model pricing (Anthropic, as of 2025).
// Models are matched by prefix — "claude-opus" matches "claude-opus-4-6".
var DefaultPricing = map[string]ModelPricing{
	"claude-opus":   {InputPerMillion: 15.0, OutputPerMillion: 75.0},
	"claude-sonnet": {InputPerMillion: 3.0, OutputPerMillion: 15.0},
	"claude-haiku":  {InputPerMillion: 0.25, OutputPerMillion: 1.25},
	// OpenAI
	"gpt-4o":      {InputPerMillion: 2.5, OutputPerMillion: 10.0},
	"gpt-4o-mini": {InputPerMillion: 0.15, OutputPerMillion: 0.6},
	"o3":          {InputPerMillion: 10.0, OutputPerMillion: 40.0},
	"o3-mini":     {InputPerMillion: 1.1, OutputPerMillion: 4.4},
	"o4-mini":     {InputPerMillion: 1.1, OutputPerMillion: 4.4},
	// Google
	"gemini-2.5-pro":   {InputPerMillion: 1.25, OutputPerMillion: 10.0},
	"gemini-2.5-flash": {InputPerMillion: 0.15, OutputPerMillion: 0.6},
}

// LookupPricing returns the pricing for a model name, matching by prefix.
// Returns zero pricing if no match found.
func LookupPricing(model string) ModelPricing {
	model = strings.ToLower(model)
	// Exact match first
	if p, ok := DefaultPricing[model]; ok {
		return p
	}
	// Prefix match (e.g. "claude-opus-4-6" matches "claude-opus")
	for prefix, p := range DefaultPricing {
		if strings.HasPrefix(model, prefix) {
			return p
		}
	}
	return ModelPricing{}
}

// ComputeCost calculates the USD cost for a given token usage.
func ComputeCost(model string, inputTokens, outputTokens int) float64 {
	pricing := LookupPricing(model)
	if pricing.InputPerMillion == 0 && pricing.OutputPerMillion == 0 {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000.0 * pricing.InputPerMillion
	outputCost := float64(outputTokens) / 1_000_000.0 * pricing.OutputPerMillion
	return inputCost + outputCost
}

// Entry represents a single cost ledger entry for a step execution.
type Entry struct {
	RunID        string
	StepID       string
	Model        string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64
}

// Ledger tracks cumulative costs for a pipeline run.
type Ledger struct {
	mu            sync.Mutex
	entries       []Entry
	totalCost     float64
	budgetCeiling float64 // 0 = unlimited
	warnAt        float64 // 0 = no warning
	warned        bool
}

// NewLedger creates a new cost ledger with optional budget ceiling and warning threshold.
func NewLedger(budgetCeiling, warnAt float64) *Ledger {
	return &Ledger{
		budgetCeiling: budgetCeiling,
		warnAt:        warnAt,
	}
}

// BudgetStatus represents the result of a budget check.
type BudgetStatus int

const (
	BudgetOK       BudgetStatus = iota
	BudgetWarning               // cost exceeded warn_at threshold
	BudgetExceeded              // cost exceeded budget_ceiling
)

// Record adds a cost entry and returns the budget status.
func (l *Ledger) Record(runID, stepID, model string, inputTokens, outputTokens, totalTokens int) (Entry, BudgetStatus) {
	cost := ComputeCost(model, inputTokens, outputTokens)
	entry := Entry{
		RunID:        runID,
		StepID:       stepID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		Cost:         cost,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, entry)
	l.totalCost += cost

	if l.budgetCeiling > 0 && l.totalCost >= l.budgetCeiling {
		return entry, BudgetExceeded
	}
	if l.warnAt > 0 && l.totalCost >= l.warnAt && !l.warned {
		l.warned = true
		return entry, BudgetWarning
	}
	return entry, BudgetOK
}

// TotalCost returns the cumulative cost across all entries.
func (l *Ledger) TotalCost() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.totalCost
}

// Entries returns a copy of all ledger entries.
func (l *Ledger) Entries() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]Entry, len(l.entries))
	copy(result, l.entries)
	return result
}

// Summary returns a human-readable cost summary.
func (l *Ledger) Summary() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) == 0 {
		return "No cost data recorded"
	}

	var totalInput, totalOutput, totalTokens int
	for _, e := range l.entries {
		totalInput += e.InputTokens
		totalOutput += e.OutputTokens
		totalTokens += e.TotalTokens
	}

	s := fmt.Sprintf("Cost: $%.4f (%d steps, %d tokens — %d in / %d out)",
		l.totalCost, len(l.entries), totalTokens, totalInput, totalOutput)

	if l.budgetCeiling > 0 {
		pct := (l.totalCost / l.budgetCeiling) * 100
		s += fmt.Sprintf(" [%.1f%% of $%.2f budget]", pct, l.budgetCeiling)
	}
	return s
}

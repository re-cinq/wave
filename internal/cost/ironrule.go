package cost

import "fmt"

// ModelContextWindow maps model names to their maximum context window size in tokens.
// Matched by prefix (e.g. "claude-opus" matches "claude-opus-4-6").
var ModelContextWindow = map[string]int{
	"claude-opus":      1_000_000,
	"claude-sonnet":    200_000,
	"claude-haiku":     200_000,
	"gpt-4o":           128_000,
	"gpt-4o-mini":      128_000,
	"o3":               200_000,
	"o3-mini":          200_000,
	"o4-mini":          200_000,
	"gemini-2.5-pro":   1_000_000,
	"gemini-2.5-flash": 1_000_000,
}

// DefaultContextWindow is used when model is unknown.
const DefaultContextWindow = 200_000

// EstimateTokens estimates token count from byte length using the 4 bytes/token heuristic.
func EstimateTokens(byteLen int) int {
	return byteLen / 4
}

// LookupContextWindow returns the context window for a model.
func LookupContextWindow(model string) int {
	if model == "" {
		return DefaultContextWindow
	}
	for prefix, window := range ModelContextWindow {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return window
		}
	}
	return DefaultContextWindow
}

// IronRuleStatus represents the result of a context window check.
type IronRuleStatus int

const (
	IronRuleOK      IronRuleStatus = iota
	IronRuleWarning                // prompt exceeds 80% of context window
	IronRuleFail                   // prompt exceeds 95% of context window
)

// CheckIronRule validates that the estimated prompt size fits within the model's context window.
// Returns OK, Warning (>80%), or Fail (>95%).
func CheckIronRule(model string, promptBytes int) (IronRuleStatus, string) {
	contextWindow := LookupContextWindow(model)
	estimatedTokens := EstimateTokens(promptBytes)

	ratio := float64(estimatedTokens) / float64(contextWindow)

	if ratio >= 0.95 {
		return IronRuleFail, fmt.Sprintf(
			"iron rule violation: estimated prompt size %d tokens exceeds 95%% of %s context window (%d tokens). Split this step into smaller units",
			estimatedTokens, model, contextWindow)
	}
	if ratio >= 0.80 {
		return IronRuleWarning, fmt.Sprintf(
			"iron rule warning: estimated prompt size %d tokens is %.0f%% of %s context window (%d tokens)",
			estimatedTokens, ratio*100, model, contextWindow)
	}
	return IronRuleOK, ""
}

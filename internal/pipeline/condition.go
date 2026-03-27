package pipeline

import (
	"fmt"
	"strings"
)

// ConditionExpr represents a parsed condition expression.
type ConditionExpr struct {
	Namespace string // "outcome", "context", or "" (unconditional)
	Key       string // The key within the namespace (e.g., "success" for outcome, "tests_passed" for context)
	Value     string // The expected value
	Raw       string // Original expression string
}

// IsUnconditional returns true if this condition always matches (empty expression).
func (c ConditionExpr) IsUnconditional() bool {
	return c.Namespace == ""
}

// ParseCondition parses a condition expression string into a ConditionExpr.
// Supported forms:
//   - "" (empty) — unconditional, always matches
//   - "outcome=success" — matches step outcome
//   - "outcome=failure" — matches step outcome
//   - "context.KEY=VALUE" — matches context key-value pair
func ParseCondition(expr string) (ConditionExpr, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ConditionExpr{Raw: expr}, nil
	}

	eqIdx := strings.Index(expr, "=")
	if eqIdx < 0 {
		return ConditionExpr{}, fmt.Errorf("invalid condition expression %q: missing '=' operator", expr)
	}

	left := strings.TrimSpace(expr[:eqIdx])
	right := strings.TrimSpace(expr[eqIdx+1:])

	if left == "" {
		return ConditionExpr{}, fmt.Errorf("invalid condition expression %q: empty left-hand side", expr)
	}
	if right == "" {
		return ConditionExpr{}, fmt.Errorf("invalid condition expression %q: empty right-hand side", expr)
	}

	// Check for context.KEY=VALUE form
	if strings.HasPrefix(left, "context.") {
		key := strings.TrimPrefix(left, "context.")
		if key == "" {
			return ConditionExpr{}, fmt.Errorf("invalid condition expression %q: empty context key", expr)
		}
		return ConditionExpr{
			Namespace: "context",
			Key:       key,
			Value:     right,
			Raw:       expr,
		}, nil
	}

	// Check for outcome=VALUE form
	if left == "outcome" {
		if right != "success" && right != "failure" {
			return ConditionExpr{}, fmt.Errorf("invalid condition expression %q: outcome must be 'success' or 'failure'", expr)
		}
		return ConditionExpr{
			Namespace: "outcome",
			Key:       right,
			Value:     right,
			Raw:       expr,
		}, nil
	}

	return ConditionExpr{}, fmt.Errorf("invalid condition expression %q: unknown namespace %q (expected 'outcome' or 'context.KEY')", expr, left)
}

// StepContext holds the evaluation context for condition expressions.
type StepContext struct {
	Outcome string            // "success" or "failure"
	Context map[string]string // Key-value pairs from step execution
}

// EvaluateCondition evaluates a condition expression against a step context.
// Returns true if the condition matches.
func EvaluateCondition(expr ConditionExpr, ctx *StepContext) bool {
	if expr.IsUnconditional() {
		return true
	}

	switch expr.Namespace {
	case "outcome":
		return ctx.Outcome == expr.Value
	case "context":
		val, ok := ctx.Context[expr.Key]
		return ok && val == expr.Value
	default:
		return false
	}
}

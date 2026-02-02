package pipeline

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

// Router matches work items against routing rules to select pipelines.
type Router struct {
	config manifest.RoutingConfig
	// sortedRules is a copy of rules sorted by priority (descending)
	sortedRules []manifest.RoutingRule
}

// NewRouter creates a new Router from the given configuration.
func NewRouter(config manifest.RoutingConfig) *Router {
	r := &Router{
		config: config,
	}
	r.sortedRules = r.sortRulesByPriority(config.Rules)
	return r
}

// NewRouterFromManifest creates a Router from a manifest's routing configuration.
func NewRouterFromManifest(m *manifest.Manifest) *Router {
	return NewRouter(m.Runtime.Routing)
}

// sortRulesByPriority returns rules sorted by priority in descending order.
// Rules with equal priority maintain their original order (stable sort).
func (r *Router) sortRulesByPriority(rules []manifest.RoutingRule) []manifest.RoutingRule {
	if len(rules) == 0 {
		return nil
	}

	// Create indexed copy to maintain original order for equal priorities
	type indexedRule struct {
		rule  manifest.RoutingRule
		index int
	}

	indexed := make([]indexedRule, len(rules))
	for i, rule := range rules {
		indexed[i] = indexedRule{rule: rule, index: i}
	}

	// Stable sort by priority descending
	sort.SliceStable(indexed, func(i, j int) bool {
		return indexed[i].rule.Priority > indexed[j].rule.Priority
	})

	sorted := make([]manifest.RoutingRule, len(rules))
	for i, ir := range indexed {
		sorted[i] = ir.rule
	}

	return sorted
}

// Route matches the input and optional labels against routing rules.
// Returns the pipeline name to execute.
// Rules are evaluated in priority order (highest first).
// Falls back to the default pipeline if no rule matches.
func (r *Router) Route(input string, labels map[string]string) string {
	for _, rule := range r.sortedRules {
		if r.matchRule(rule, input, labels) {
			return rule.Pipeline
		}
	}

	return r.config.Default
}

// matchRule checks if a single rule matches the input and labels.
func (r *Router) matchRule(rule manifest.RoutingRule, input string, labels map[string]string) bool {
	// Check input pattern match
	if rule.Pattern != "" {
		if !r.matchPattern(rule.Pattern, input) {
			return false
		}
	}

	// Check label matches (all must match)
	if len(rule.MatchLabels) > 0 {
		if !r.matchLabels(rule.MatchLabels, labels) {
			return false
		}
	}

	// If rule has neither pattern nor labels, it matches everything
	// This allows catch-all rules with just a pipeline name
	return true
}

// matchPattern checks if the input matches the glob pattern.
// Uses filepath.Match for glob matching.
func (r *Router) matchPattern(pattern, input string) bool {
	// Normalize input for matching (trim whitespace, lowercase for case-insensitive)
	normalizedInput := strings.TrimSpace(input)

	// Try exact match first
	matched, err := filepath.Match(pattern, normalizedInput)
	if err == nil && matched {
		return true
	}

	// Try case-insensitive match
	matched, err = filepath.Match(strings.ToLower(pattern), strings.ToLower(normalizedInput))
	if err == nil && matched {
		return true
	}

	// For patterns without wildcards, check substring containment
	if !strings.ContainsAny(pattern, "*?[") {
		return strings.Contains(strings.ToLower(normalizedInput), strings.ToLower(pattern))
	}

	return false
}

// matchLabels checks if all required labels match.
func (r *Router) matchLabels(required map[string]string, actual map[string]string) bool {
	if actual == nil {
		return len(required) == 0
	}

	for key, pattern := range required {
		value, exists := actual[key]
		if !exists {
			return false
		}

		// Match label value against pattern
		if !r.matchPattern(pattern, value) {
			return false
		}
	}

	return true
}

// GetDefaultPipeline returns the default pipeline name.
func (r *Router) GetDefaultPipeline() string {
	return r.config.Default
}

// GetRules returns the sorted routing rules.
func (r *Router) GetRules() []manifest.RoutingRule {
	return r.sortedRules
}

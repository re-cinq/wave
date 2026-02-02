package pipeline

import (
	"testing"

	"github.com/recinq/wave/internal/manifest"
)

func TestRouter_EmptyConfig(t *testing.T) {
	config := manifest.RoutingConfig{}
	router := NewRouter(config)

	result := router.Route("any input", nil)
	if result != "" {
		t.Errorf("Expected empty string for empty config, got: %s", result)
	}
}

func TestRouter_DefaultFallback(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules:   []manifest.RoutingRule{},
	}
	router := NewRouter(config)

	result := router.Route("any input", nil)
	if result != "default-pipeline" {
		t.Errorf("Expected 'default-pipeline', got: %s", result)
	}
}

func TestRouter_ExactPatternMatch(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "fix bug", Pipeline: "bugfix-pipeline"},
		},
	}
	router := NewRouter(config)

	result := router.Route("fix bug", nil)
	if result != "bugfix-pipeline" {
		t.Errorf("Expected 'bugfix-pipeline', got: %s", result)
	}
}

func TestRouter_GlobPatternMatch(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "fix*", Pipeline: "bugfix-pipeline"},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"fix", "bugfix-pipeline"},
		{"fix bug", "bugfix-pipeline"}, // glob * matches any sequence including spaces
		{"fixing", "bugfix-pipeline"},
		{"something else", "default-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, nil)
		if result != tt.expected {
			t.Errorf("Route(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRouter_SubstringMatch(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "bug", Pipeline: "bugfix-pipeline"},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"fix the bug", "bugfix-pipeline"},
		{"bug in code", "bugfix-pipeline"},
		{"debugging", "bugfix-pipeline"},
		{"feature request", "default-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, nil)
		if result != tt.expected {
			t.Errorf("Route(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRouter_CaseInsensitiveMatch(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "BUG", Pipeline: "bugfix-pipeline"},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"bug", "bugfix-pipeline"},
		{"BUG", "bugfix-pipeline"},
		{"Bug", "bugfix-pipeline"},
		{"fix the BUG", "bugfix-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, nil)
		if result != tt.expected {
			t.Errorf("Route(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRouter_PriorityOrder(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "bug", Pipeline: "low-priority", Priority: 1},
			{Pattern: "bug", Pipeline: "high-priority", Priority: 10},
			{Pattern: "bug", Pipeline: "medium-priority", Priority: 5},
		},
	}
	router := NewRouter(config)

	result := router.Route("fix a bug", nil)
	if result != "high-priority" {
		t.Errorf("Expected 'high-priority' (highest priority), got: %s", result)
	}
}

func TestRouter_SamePriorityMaintainsOrder(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "bug", Pipeline: "first-rule", Priority: 5},
			{Pattern: "bug", Pipeline: "second-rule", Priority: 5},
			{Pattern: "bug", Pipeline: "third-rule", Priority: 5},
		},
	}
	router := NewRouter(config)

	result := router.Route("bug", nil)
	if result != "first-rule" {
		t.Errorf("Expected 'first-rule' (first in definition order), got: %s", result)
	}
}

func TestRouter_LabelMatching(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{
				Pipeline: "urgent-pipeline",
				MatchLabels: map[string]string{
					"priority": "high",
				},
			},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		labels   map[string]string
		expected string
	}{
		{"task", map[string]string{"priority": "high"}, "urgent-pipeline"},
		{"task", map[string]string{"priority": "low"}, "default-pipeline"},
		{"task", map[string]string{}, "default-pipeline"},
		{"task", nil, "default-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, tt.labels)
		if result != tt.expected {
			t.Errorf("Route(%q, %v) = %q, want %q", tt.input, tt.labels, result, tt.expected)
		}
	}
}

func TestRouter_LabelGlobPatterns(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{
				Pipeline: "feature-pipeline",
				MatchLabels: map[string]string{
					"type": "feature*",
				},
			},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		labels   map[string]string
		expected string
	}{
		{map[string]string{"type": "feature"}, "feature-pipeline"},
		{map[string]string{"type": "feature-request"}, "feature-pipeline"},
		{map[string]string{"type": "bug"}, "default-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route("task", tt.labels)
		if result != tt.expected {
			t.Errorf("Route with labels %v = %q, want %q", tt.labels, result, tt.expected)
		}
	}
}

func TestRouter_MultipleLabelRequirements(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{
				Pipeline: "special-pipeline",
				MatchLabels: map[string]string{
					"priority": "high",
					"type":     "bug",
				},
			},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		labels   map[string]string
		expected string
	}{
		{map[string]string{"priority": "high", "type": "bug"}, "special-pipeline"},
		{map[string]string{"priority": "high"}, "default-pipeline"},
		{map[string]string{"type": "bug"}, "default-pipeline"},
		{map[string]string{"priority": "low", "type": "bug"}, "default-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route("task", tt.labels)
		if result != tt.expected {
			t.Errorf("Route with labels %v = %q, want %q", tt.labels, result, tt.expected)
		}
	}
}

func TestRouter_PatternAndLabelCombined(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{
				Pattern:  "urgent",
				Pipeline: "urgent-feature",
				MatchLabels: map[string]string{
					"type": "feature",
				},
			},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		labels   map[string]string
		expected string
	}{
		{"urgent task", map[string]string{"type": "feature"}, "urgent-feature"},
		{"urgent task", map[string]string{"type": "bug"}, "default-pipeline"},
		{"normal task", map[string]string{"type": "feature"}, "default-pipeline"},
		{"normal task", map[string]string{"type": "bug"}, "default-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, tt.labels)
		if result != tt.expected {
			t.Errorf("Route(%q, %v) = %q, want %q", tt.input, tt.labels, result, tt.expected)
		}
	}
}

func TestRouter_CatchAllRule(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "specific", Pipeline: "specific-pipeline", Priority: 10},
			{Pipeline: "catch-all-pipeline", Priority: 1}, // No pattern or labels
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"specific task", "specific-pipeline"},
		{"any other task", "catch-all-pipeline"},
		{"random input", "catch-all-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, nil)
		if result != tt.expected {
			t.Errorf("Route(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRouter_WhitespaceHandling(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "default-pipeline",
		Rules: []manifest.RoutingRule{
			{Pattern: "bug", Pipeline: "bugfix-pipeline"},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		input    string
		expected string
	}{
		{"  bug  ", "bugfix-pipeline"},
		{"\tbug\n", "bugfix-pipeline"},
		{"  fix bug  ", "bugfix-pipeline"},
	}

	for _, tt := range tests {
		result := router.Route(tt.input, nil)
		if result != tt.expected {
			t.Errorf("Route(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRouter_GetDefaultPipeline(t *testing.T) {
	config := manifest.RoutingConfig{
		Default: "my-default",
	}
	router := NewRouter(config)

	if router.GetDefaultPipeline() != "my-default" {
		t.Errorf("GetDefaultPipeline() = %q, want 'my-default'", router.GetDefaultPipeline())
	}
}

func TestRouter_GetRules(t *testing.T) {
	config := manifest.RoutingConfig{
		Rules: []manifest.RoutingRule{
			{Pattern: "a", Pipeline: "pipeline-a", Priority: 1},
			{Pattern: "b", Pipeline: "pipeline-b", Priority: 3},
			{Pattern: "c", Pipeline: "pipeline-c", Priority: 2},
		},
	}
	router := NewRouter(config)

	rules := router.GetRules()
	if len(rules) != 3 {
		t.Fatalf("Expected 3 rules, got %d", len(rules))
	}

	// Rules should be sorted by priority descending
	if rules[0].Pipeline != "pipeline-b" {
		t.Errorf("First rule should be pipeline-b (priority 3), got %s", rules[0].Pipeline)
	}
	if rules[1].Pipeline != "pipeline-c" {
		t.Errorf("Second rule should be pipeline-c (priority 2), got %s", rules[1].Pipeline)
	}
	if rules[2].Pipeline != "pipeline-a" {
		t.Errorf("Third rule should be pipeline-a (priority 1), got %s", rules[2].Pipeline)
	}
}

func TestNewRouterFromManifest(t *testing.T) {
	m := &manifest.Manifest{
		Runtime: manifest.Runtime{
			Routing: manifest.RoutingConfig{
				Default: "manifest-default",
				Rules: []manifest.RoutingRule{
					{Pattern: "test", Pipeline: "test-pipeline"},
				},
			},
		},
	}

	router := NewRouterFromManifest(m)

	if router.GetDefaultPipeline() != "manifest-default" {
		t.Errorf("Expected default 'manifest-default', got %q", router.GetDefaultPipeline())
	}

	result := router.Route("test input", nil)
	if result != "test-pipeline" {
		t.Errorf("Expected 'test-pipeline', got %q", result)
	}
}

func TestRouter_ComplexScenario(t *testing.T) {
	// Simulate a real-world routing configuration
	config := manifest.RoutingConfig{
		Default: "general-pipeline",
		Rules: []manifest.RoutingRule{
			// High priority: security issues
			{
				Pattern:  "security",
				Pipeline: "security-pipeline",
				Priority: 100,
			},
			// Medium priority: type-based routing
			{
				Pipeline: "feature-pipeline",
				Priority: 50,
				MatchLabels: map[string]string{
					"type": "feature",
				},
			},
			{
				Pipeline: "bugfix-pipeline",
				Priority: 50,
				MatchLabels: map[string]string{
					"type": "bug",
				},
			},
			// Low priority: keyword matching
			{
				Pattern:  "refactor",
				Pipeline: "refactor-pipeline",
				Priority: 10,
			},
			{
				Pattern:  "docs",
				Pipeline: "docs-pipeline",
				Priority: 10,
			},
		},
	}
	router := NewRouter(config)

	tests := []struct {
		name     string
		input    string
		labels   map[string]string
		expected string
	}{
		{
			name:     "security issue takes precedence",
			input:    "fix security vulnerability",
			labels:   map[string]string{"type": "bug"},
			expected: "security-pipeline",
		},
		{
			name:     "feature by label",
			input:    "add new feature",
			labels:   map[string]string{"type": "feature"},
			expected: "feature-pipeline",
		},
		{
			name:     "bug by label",
			input:    "fix crash",
			labels:   map[string]string{"type": "bug"},
			expected: "bugfix-pipeline",
		},
		{
			name:     "refactor by keyword",
			input:    "refactor authentication module",
			labels:   nil,
			expected: "refactor-pipeline",
		},
		{
			name:     "docs by keyword",
			input:    "update docs for API",
			labels:   nil,
			expected: "docs-pipeline",
		},
		{
			name:     "fallback to default",
			input:    "some random task",
			labels:   nil,
			expected: "general-pipeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.Route(tt.input, tt.labels)
			if result != tt.expected {
				t.Errorf("Route(%q, %v) = %q, want %q", tt.input, tt.labels, result, tt.expected)
			}
		})
	}
}

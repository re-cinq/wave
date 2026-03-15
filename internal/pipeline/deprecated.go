package pipeline

import "strings"

// taxonomyMappings maps old pipeline names to their new taxonomy-prefixed names.
var taxonomyMappings = map[string]string{
	"adr":                "plan-adr",
	"changelog":          "doc-changelog",
	"dead-code":          "audit-dead-code",
	"debug":              "ops-debug",
	"doc-audit":          "audit-doc",
	"explain":            "doc-explain",
	"feature":            "impl-feature",
	"hello-world":        "ops-hello-world",
	"hotfix":             "impl-hotfix",
	"improve":            "impl-improve",
	"onboard":            "doc-onboard",
	"plan":               "plan-task",
	"prototype":          "impl-prototype",
	"recinq":             "impl-recinq",
	"refactor":           "impl-refactor",
	"security-scan":      "audit-security",
	"smoke-test":         "test-smoke",
	"speckit-flow":       "impl-speckit",
	"plan-speckit":       "impl-speckit",
	"plan-prototype":     "impl-prototype",
	"consolidate":        "audit-consolidate",
	"ops-consolidate":    "audit-consolidate",
	"supervise":          "ops-supervise",
	"research-implement": "impl-research",
	"implement":          "impl-issue",
	"refresh":            "ops-refresh",
	"research":           "plan-research",
	"rewrite":            "ops-rewrite",
	"scope":              "plan-scope",
	"pr-review":          "ops-pr-review",
}

// ResolveDeprecatedName checks if a pipeline name uses a legacy forge-prefixed
// format or a pre-taxonomy name, and returns the current name with a deprecation flag.
func ResolveDeprecatedName(name string) (resolved string, deprecated bool) {
	// Check taxonomy mappings first (exact match)
	if newName, ok := taxonomyMappings[name]; ok {
		return newName, true
	}

	// Check forge-prefixed names (gh-*, gl-*, bb-*, gt-*)
	prefixes := []string{"gh-", "gl-", "bb-", "gt-"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			stripped := name[len(p):]
			// Check if the stripped name also has a taxonomy mapping
			if newName, ok := taxonomyMappings[stripped]; ok {
				return newName, true
			}
			return stripped, true
		}
	}

	return name, false
}

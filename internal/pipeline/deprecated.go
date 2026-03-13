package pipeline

import "strings"

// deprecatedPipelineNames maps old forge-prefixed pipeline names to their unified equivalents.
var deprecatedPipelineNames = map[string]string{
	"gh-implement": "implement",
	"gl-implement": "implement",
	"bb-implement": "implement",
	"gt-implement": "implement",

	"gh-scope": "scope",
	"gl-scope": "scope",
	"bb-scope": "scope",
	"gt-scope": "scope",

	"gh-research": "research",
	"gl-research": "research",
	"bb-research": "research",
	"gt-research": "research",

	"gh-rewrite": "rewrite",
	"gl-rewrite": "rewrite",
	"bb-rewrite": "rewrite",
	"gt-rewrite": "rewrite",

	"gh-refresh": "refresh",
	"gl-refresh": "refresh",
	"bb-refresh": "refresh",
	"gt-refresh": "refresh",

	"gh-pr-review": "pr-review",
	"gl-pr-review": "pr-review",
	"bb-pr-review": "pr-review",
	"gt-pr-review": "pr-review",

	"gh-implement-epic": "implement-epic",
	"gl-implement-epic": "implement-epic",
	"bb-implement-epic": "implement-epic",
	"gt-implement-epic": "implement-epic",
}

// ResolveDeprecatedName checks if a pipeline name is a deprecated forge-prefixed name
// and returns the unified name. The second return value indicates whether the name was
// deprecated (true) or already current (false).
func ResolveDeprecatedName(name string) (string, bool) {
	if unified, ok := deprecatedPipelineNames[name]; ok {
		return unified, true
	}
	return name, false
}

// IsDeprecatedPipelineName returns true if the given name is a known deprecated forge-prefixed pipeline name.
func IsDeprecatedPipelineName(name string) bool {
	_, deprecated := deprecatedPipelineNames[name]
	return deprecated
}

// StripForgePrefix removes the forge prefix from a pipeline name if present.
// Returns the base name and whether a prefix was stripped.
func StripForgePrefix(name string) (string, bool) {
	prefixes := []string{"gh-", "gl-", "bb-", "gt-"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return strings.TrimPrefix(name, p), true
		}
	}
	return name, false
}

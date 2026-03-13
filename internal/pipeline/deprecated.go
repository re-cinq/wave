package pipeline

import "strings"

// ResolveDeprecatedName checks if a pipeline name uses a legacy forge-prefixed
// format and returns the unified name with a deprecation flag.
func ResolveDeprecatedName(name string) (resolved string, deprecated bool) {
	prefixes := []string{"gh-", "gl-", "bb-", "gt-"}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return name[len(p):], true
		}
	}
	return name, false
}

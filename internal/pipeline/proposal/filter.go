package proposal

import "strings"

// forgePrefixes maps pipeline name prefixes to forge types.
var forgePrefixes = map[string]ForgeType{
	"gh-": ForgeGitHub,
	"gl-": ForgeGitLab,
	"gt-": ForgeGitea,
	"bb-": ForgeBitBkt,
}

// DetectForgeFromPrefix classifies a pipeline by its name prefix.
// Pipelines without a recognized prefix are considered forge-agnostic
// and return ForgeUnknown.
func DetectForgeFromPrefix(name string) ForgeType {
	for prefix, forge := range forgePrefixes {
		if strings.HasPrefix(name, prefix) {
			return forge
		}
	}
	return ForgeUnknown
}

// IsForgeAgnostic returns true if the pipeline name has no forge prefix,
// meaning it is compatible with any forge.
func IsForgeAgnostic(name string) bool {
	return DetectForgeFromPrefix(name) == ForgeUnknown
}

// FilterByForge returns only the catalog entries that match the given
// forge type or are forge-agnostic. If forgeType is ForgeUnknown, all
// entries are returned (no filtering).
func FilterByForge(entries []CatalogEntry, forgeType ForgeType) []CatalogEntry {
	if forgeType == ForgeUnknown {
		out := make([]CatalogEntry, len(entries))
		copy(out, entries)
		return out
	}

	var result []CatalogEntry
	for _, e := range entries {
		pipelineForge := DetectForgeFromPrefix(e.Name)
		if pipelineForge == forgeType || pipelineForge == ForgeUnknown {
			result = append(result, e)
		}
	}
	return result
}

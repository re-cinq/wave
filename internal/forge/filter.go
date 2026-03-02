package forge

import "strings"

// forgePrefixes lists all recognized forge pipeline prefixes.
var forgePrefixes = []string{"gh-", "gl-", "bb-", "gt-"}

// FilterPipelines returns only pipelines that match the given forge type
// plus all non-prefixed (universal) pipelines.
// If forgeType is Unknown, all pipelines are returned.
func FilterPipelines(forgeType ForgeType, pipelineNames []string) []string {
	if forgeType == Unknown {
		return pipelineNames
	}

	prefix := forgeType.Prefix()
	var result []string
	for _, name := range pipelineNames {
		if strings.HasPrefix(name, prefix) {
			result = append(result, name)
			continue
		}
		if !hasForgePrefix(name) {
			result = append(result, name)
		}
	}
	return result
}

// hasForgePrefix returns true if the pipeline name starts with any known forge prefix.
func hasForgePrefix(name string) bool {
	for _, p := range forgePrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

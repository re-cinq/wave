package skill

import "sort"

// ResolveSkills merges skill names from three configuration scopes:
// pipeline (highest precedence), persona, and global (lowest precedence).
// It deduplicates entries and returns the result sorted alphabetically.
// Returns nil when all inputs are empty.
func ResolveSkills(global, persona, pipeline []string) []string {
	if len(global) == 0 && len(persona) == 0 && len(pipeline) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []string

	// Iterate pipeline first (highest precedence), then persona, then global.
	// Precedence order determines which scope "wins" for deduplication tracking,
	// but since we only collect unique names the final output is the union.
	for _, scope := range [][]string{pipeline, persona, global} {
		for _, name := range scope {
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}

	sort.Strings(result)
	return result
}

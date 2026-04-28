package listing

import (
	"os/exec"
	"sort"
)

// ListAdapters converts a manifest's adapter map into a sorted slice of
// AdapterInfo records, including a binary-on-PATH availability check.
func ListAdapters(adapters map[string]ManifestAdapter) []AdapterInfo {
	if len(adapters) == 0 {
		return nil
	}

	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]AdapterInfo, 0, len(names))
	for _, name := range names {
		a := adapters[name]
		available := true
		if _, err := exec.LookPath(a.Binary); err != nil {
			available = false
		}
		result = append(result, AdapterInfo{
			Name:         name,
			Binary:       a.Binary,
			Mode:         a.Mode,
			OutputFormat: a.OutputFormat,
			Available:    available,
		})
	}
	return result
}

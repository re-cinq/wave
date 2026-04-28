package listing

import "sort"

// ListPersonas converts a manifest's persona map into a sorted slice of
// PersonaInfo records.
func ListPersonas(personas map[string]ManifestPersona) []PersonaInfo {
	if len(personas) == 0 {
		return nil
	}

	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]PersonaInfo, 0, len(names))
	for _, name := range names {
		p := personas[name]
		result = append(result, PersonaInfo{
			Name:         name,
			Adapter:      p.Adapter,
			Description:  p.Description,
			Temperature:  p.Temperature,
			AllowedTools: p.Permissions.AllowedTools,
			DeniedTools:  p.Permissions.Deny,
		})
	}
	return result
}

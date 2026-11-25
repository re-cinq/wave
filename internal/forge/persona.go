package forge

import "strings"

// PersonaForgeTypes are the forge types that ship forge-specific personas
// (`<type>-analyst`, `<type>-commenter`, ...). A `{{ forge.type }}` persona
// reference resolves to one of these once a forge remote is known.
var PersonaForgeTypes = []ForgeType{ForgeGitHub, ForgeGitLab, ForgeGitea, ForgeBitbucket}

// IsForgeTemplatedPersona reports whether a persona reference contains a
// `{{ forge.type }}` placeholder (spaced or compact form).
func IsForgeTemplatedPersona(persona string) bool {
	return strings.Contains(persona, "{{ forge.type }}") || strings.Contains(persona, "{{forge.type}}")
}

// ResolvePersonaCandidates expands `{{ forge.type }}` in a persona reference.
// A reference without the placeholder is returned unchanged. With a detected
// forge it resolves to exactly the persona the executor will use; for a local
// or undetected forge (no remote yet) the persona is only chosen once a remote
// exists, so every shipped forge variant is returned.
func ResolvePersonaCandidates(persona string, fi ForgeInfo) []string {
	if !IsForgeTemplatedPersona(persona) {
		return []string{persona}
	}
	if fi.Type != ForgeUnknown && fi.Type != ForgeLocal && fi.Type != "" {
		return []string{replaceForgeType(persona, fi.Type)}
	}
	results := make([]string, 0, len(PersonaForgeTypes))
	for _, ft := range PersonaForgeTypes {
		results = append(results, replaceForgeType(persona, ft))
	}
	return results
}

func replaceForgeType(persona string, ft ForgeType) string {
	return strings.NewReplacer("{{ forge.type }}", string(ft), "{{forge.type}}", string(ft)).Replace(persona)
}

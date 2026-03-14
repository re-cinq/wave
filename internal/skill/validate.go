package skill

import "fmt"

// PersonaSkills holds the skill list for a persona (used by ValidateManifestSkills).
type PersonaSkills struct {
	Name   string
	Skills []string
}

// ValidateSkillRefs validates a list of skill name references.
// It checks name format via ValidateName() and existence via store.Read() if store is non-nil.
// Returns all errors aggregated (not fail-fast). Each error includes scope context.
func ValidateSkillRefs(names []string, scope string, store Store) []error {
	var errs []error
	for _, name := range names {
		if err := ValidateName(name); err != nil {
			errs = append(errs, fmt.Errorf("%s: invalid skill name %q: %w", scope, name, err))
			continue
		}
		if store != nil {
			if _, err := store.Read(name); err != nil {
				errs = append(errs, fmt.Errorf("%s: skill %q not found in store", scope, name))
			}
		}
	}
	return errs
}

// ValidateManifestSkills validates all skill references across global and persona scopes.
func ValidateManifestSkills(globalSkills []string, personas []PersonaSkills, store Store) []error {
	var errs []error
	errs = append(errs, ValidateSkillRefs(globalSkills, "global", store)...)
	for _, p := range personas {
		errs = append(errs, ValidateSkillRefs(p.Skills, "persona:"+p.Name, store)...)
	}
	return errs
}

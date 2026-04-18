package skill

import "fmt"

// PersonaSkills holds the skill list for a persona (used by ValidateManifestSkills).
type PersonaSkills struct {
	Name   string
	Skills []string
}

// ValidationIssue represents a single validation finding.
type ValidationIssue struct {
	Field   string
	Message string
}

// ValidationReport contains validation results for a SKILL.md file.
type ValidationReport struct {
	Errors   []ValidationIssue
	Warnings []ValidationIssue
}

// Valid returns true if there are no blocking errors.
func (r *ValidationReport) Valid() bool {
	return len(r.Errors) == 0
}

// ValidateForPublish validates a skill for publishing.
// Required fields produce errors; optional fields produce warnings.
func ValidateForPublish(s Skill) ValidationReport {
	var report ValidationReport

	if s.Name == "" {
		report.Errors = append(report.Errors, ValidationIssue{
			Field:   "name",
			Message: "name is required",
		})
	} else if err := ValidateName(s.Name); err != nil {
		report.Errors = append(report.Errors, ValidationIssue{
			Field:   "name",
			Message: err.Error(),
		})
	}

	if s.Description == "" {
		report.Errors = append(report.Errors, ValidationIssue{
			Field:   "description",
			Message: "description is required",
		})
	}

	if s.License == "" {
		report.Warnings = append(report.Warnings, ValidationIssue{
			Field:   "license",
			Message: "license is recommended for published skills",
		})
	}

	if s.Compatibility == "" {
		report.Warnings = append(report.Warnings, ValidationIssue{
			Field:   "compatibility",
			Message: "compatibility is recommended for published skills",
		})
	}

	if len(s.AllowedTools) == 0 {
		report.Warnings = append(report.Warnings, ValidationIssue{
			Field:   "allowed-tools",
			Message: "allowed-tools is recommended for published skills",
		})
	}

	return report
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
				errs = append(errs, fmt.Errorf("%s: skill %q not found in store — install with `wave skills add <path-or-url>` or place SKILL.md under .agents/skills/%s/", scope, name, name))
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

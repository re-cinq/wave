package manifest

import (
	"fmt"
	"strings"
)

// ValidateOntology is the exported form of validateOntology for consumers
// that hold a *Manifest but not an internal reference (e.g. the ontology
// Service). It returns []error for symmetry with other manifest validators.
func ValidateOntology(o *Ontology, filePath string) []error {
	return validateOntology(o, filePath)
}

// validateOntology performs shape validation on the ontology section of a
// manifest: non-empty and unique context names. This is plain struct
// validation — independent of whether the ontology feature is enabled — so
// it lives alongside other manifest validators rather than behind a feature
// gate.
//
// Behavioral ontology logic (staleness, injection, lineage, doctor checks)
// lives in internal/ontology.
func validateOntology(o *Ontology, filePath string) []error {
	if o == nil {
		return nil
	}
	var errs []error
	seen := make(map[string]bool)
	for i, ctx := range o.Contexts {
		if strings.TrimSpace(ctx.Name) == "" {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("ontology.contexts[%d].name", i),
				Reason:     "is required",
				Suggestion: "Each bounded context must have a name",
			})
			continue
		}
		if seen[ctx.Name] {
			errs = append(errs, &ValidationError{
				File:       filePath,
				Field:      fmt.Sprintf("ontology.contexts[%d].name", i),
				Reason:     fmt.Sprintf("duplicate context name %q", ctx.Name),
				Suggestion: "Each bounded context name must be unique",
			})
		}
		seen[ctx.Name] = true
	}
	return errs
}

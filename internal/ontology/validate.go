package ontology

import "github.com/recinq/wave/internal/manifest"

// ValidateManifest delegates to manifest.ValidateOntology. Shape validation
// of the ontology section lives in manifest (plain struct validation), but
// the Service exposes it so callers can stay feature-gated.
func (s *realService) ValidateManifest(m *manifest.Manifest) []error {
	return ValidateManifestShape(m)
}

// ValidateManifestShape is the package-level form of Service.ValidateManifest,
// suitable for callers that do not hold a Service (e.g. tests). It delegates
// to manifest.ValidateOntology so the shape rules remain single-sourced.
func ValidateManifestShape(m *manifest.Manifest) []error {
	if m == nil {
		return nil
	}
	return manifest.ValidateOntology(m.Ontology, "")
}

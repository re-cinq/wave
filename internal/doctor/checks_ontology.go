package doctor

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
)

// checkOntology reports on the ontology section of the manifest and its
// supporting on-disk artifacts (context skills, staleness sentinel). It is
// always compiled; absence of ontology in the manifest surfaces as a WARN.
func checkOntology(opts *Options, m *manifest.Manifest) []CheckResult {
	if m == nil || m.Ontology == nil {
		return []CheckResult{{
			Name:     "Ontology",
			Category: "ontology",
			Status:   StatusWarn,
			Message:  "No ontology defined in wave.yaml",
			Fix:      "Run 'wave analyze' to generate project ontology",
		}}
	}

	var results []CheckResult

	// Telos
	if m.Ontology.Telos == "" {
		results = append(results, CheckResult{
			Name:     "Ontology Telos",
			Category: "ontology",
			Status:   StatusWarn,
			Message:  "No telos (project purpose) defined",
			Fix:      "Add 'telos' under 'ontology' in wave.yaml",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "Ontology Telos",
			Category: "ontology",
			Status:   StatusOK,
			Message:  "Telos defined",
		})
	}

	// Contexts
	if len(m.Ontology.Contexts) == 0 {
		results = append(results, CheckResult{
			Name:     "Ontology Contexts",
			Category: "ontology",
			Status:   StatusWarn,
			Message:  "No bounded contexts defined",
			Fix:      "Run 'wave analyze --deep' to generate bounded contexts",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "Ontology Contexts",
			Category: "ontology",
			Status:   StatusOK,
			Message:  fmt.Sprintf("%d bounded contexts defined", len(m.Ontology.Contexts)),
		})
	}

	// Context skills provisioned?
	skillsDir := opts.WaveDir + "/skills"
	missing := 0
	for _, ctx := range m.Ontology.Contexts {
		skillPath := skillsDir + "/wave-ctx-" + ctx.Name + "/SKILL.md"
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			missing++
		}
	}
	if missing > 0 {
		results = append(results, CheckResult{
			Name:     "Ontology Skills",
			Category: "ontology",
			Status:   StatusWarn,
			Message:  fmt.Sprintf("%d context skills not provisioned", missing),
			Fix:      "Run 'wave analyze --deep' to generate context skills",
		})
	} else if len(m.Ontology.Contexts) > 0 {
		results = append(results, CheckResult{
			Name:     "Ontology Skills",
			Category: "ontology",
			Status:   StatusOK,
			Message:  fmt.Sprintf("All %d context skills provisioned", len(m.Ontology.Contexts)),
		})
	}

	// Staleness — delegate sentinel path ownership to the ontology package.
	if ontology.IsStaleInDir(opts.WaveDir) {
		results = append(results, CheckResult{
			Name:     "Ontology Staleness",
			Category: "ontology",
			Status:   StatusWarn,
			Message:  "Ontology may be stale (changes detected since last analysis)",
			Fix:      "Run 'wave analyze' to refresh ontology",
		})
	} else {
		results = append(results, CheckResult{
			Name:     "Ontology Staleness",
			Category: "ontology",
			Status:   StatusOK,
			Message:  "Ontology is up to date",
		})
	}

	return results
}

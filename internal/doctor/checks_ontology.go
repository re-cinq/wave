package doctor

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/ontology"
	"github.com/recinq/wave/internal/suggest"
)

// checkOntology reports on the ontology section of the manifest and its
// supporting on-disk artifacts (context skills, staleness sentinel). It is
// always compiled; absence of ontology in the manifest surfaces as a WARN.
func checkOntology(opts *Options, m *manifest.Manifest) []suggest.CheckResult {
	if m == nil || m.Ontology == nil {
		return []suggest.CheckResult{{
			Name:     "Ontology",
			Category: "ontology",
			Status:   suggest.StatusWarn,
			Message:  "No ontology defined in wave.yaml",
			Fix:      "Run 'wave analyze' to generate project ontology",
		}}
	}

	var results []suggest.CheckResult

	// Telos
	if m.Ontology.Telos == "" {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Telos",
			Category: "ontology",
			Status:   suggest.StatusWarn,
			Message:  "No telos (project purpose) defined",
			Fix:      "Add 'telos' under 'ontology' in wave.yaml",
		})
	} else {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Telos",
			Category: "ontology",
			Status:   suggest.StatusOK,
			Message:  "Telos defined",
		})
	}

	// Contexts
	if len(m.Ontology.Contexts) == 0 {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Contexts",
			Category: "ontology",
			Status:   suggest.StatusWarn,
			Message:  "No bounded contexts defined",
			Fix:      "Run 'wave analyze --deep' to generate bounded contexts",
		})
	} else {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Contexts",
			Category: "ontology",
			Status:   suggest.StatusOK,
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
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Skills",
			Category: "ontology",
			Status:   suggest.StatusWarn,
			Message:  fmt.Sprintf("%d context skills not provisioned", missing),
			Fix:      "Run 'wave analyze --deep' to generate context skills",
		})
	} else if len(m.Ontology.Contexts) > 0 {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Skills",
			Category: "ontology",
			Status:   suggest.StatusOK,
			Message:  fmt.Sprintf("All %d context skills provisioned", len(m.Ontology.Contexts)),
		})
	}

	// Staleness — delegate sentinel path ownership to the ontology package.
	if ontology.IsStaleInDir(opts.WaveDir) {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Staleness",
			Category: "ontology",
			Status:   suggest.StatusWarn,
			Message:  "Ontology may be stale (changes detected since last analysis)",
			Fix:      "Run 'wave analyze' to refresh ontology",
		})
	} else {
		results = append(results, suggest.CheckResult{
			Name:     "Ontology Staleness",
			Category: "ontology",
			Status:   suggest.StatusOK,
			Message:  "Ontology is up to date",
		})
	}

	return results
}

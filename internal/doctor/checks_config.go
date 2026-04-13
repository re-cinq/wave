package doctor

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
)

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

	// Check telos
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

	// Check contexts
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

	// Check context skills are provisioned
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

	// Check staleness sentinel
	sentinelPath := opts.WaveDir + "/.ontology-stale"
	if _, err := os.Stat(sentinelPath); err == nil {
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

func checkAdapterRegistry(m *manifest.Manifest) CheckResult {
	if m == nil || len(m.Adapters) == 0 {
		return CheckResult{
			Name:     "Adapter Registry",
			Category: "capabilities",
			Status:   StatusOK,
			Message:  "No adapters registered",
		}
	}

	names := make([]string, 0, len(m.Adapters))
	for name := range m.Adapters {
		names = append(names, name)
	}
	sort.Strings(names)

	return CheckResult{
		Name:     "Adapter Registry",
		Category: "capabilities",
		Status:   StatusOK,
		Message:  fmt.Sprintf("Registered adapters: %s", strings.Join(names, ", ")),
	}
}

func checkRetryPolicies(opts *Options) []CheckResult {
	pipelines := loadAllPipelines(opts.PipelinesDir)
	if len(pipelines) == 0 {
		return []CheckResult{{
			Name:     "Retry Policies",
			Category: "capabilities",
			Status:   StatusOK,
			Message:  "No pipelines to check",
		}}
	}

	var rawSteps []string
	totalRetrySteps := 0
	policySteps := 0

	for _, pl := range pipelines {
		for _, step := range pl.Steps {
			if step.Retry.MaxAttempts > 1 || step.Retry.Policy != "" {
				totalRetrySteps++
				if step.Retry.Policy != "" {
					policySteps++
				} else {
					rawSteps = append(rawSteps, fmt.Sprintf("%s/%s", pl.Metadata.Name, step.ID))
				}
			}
		}
	}

	if totalRetrySteps == 0 {
		return []CheckResult{{
			Name:     "Retry Policies",
			Category: "capabilities",
			Status:   StatusOK,
			Message:  "No retry configurations found",
		}}
	}

	if len(rawSteps) == 0 {
		return []CheckResult{{
			Name:     "Retry Policies",
			Category: "capabilities",
			Status:   StatusOK,
			Message:  fmt.Sprintf("All %d retry steps use named policies", policySteps),
		}}
	}

	return []CheckResult{{
		Name:     "Retry Policies",
		Category: "capabilities",
		Status:   StatusWarn,
		Message:  fmt.Sprintf("%d of %d retry steps use raw max_attempts without a named policy", len(rawSteps), totalRetrySteps),
		Fix:      "Use named retry policies (standard, aggressive, patient) instead of raw max_attempts",
	}}
}

func checkEngineCapabilities() CheckResult {
	capabilities := []string{
		"graph loops",
		"gates",
		"hooks",
		"retro",
		"fork/rewind",
		"llm_judge",
		"thread continuity",
		"sub-pipelines",
	}

	return CheckResult{
		Name:     "Engine Capabilities",
		Category: "capabilities",
		Status:   StatusOK,
		Message:  fmt.Sprintf("Available: %s", strings.Join(capabilities, ", ")),
	}
}

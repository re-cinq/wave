package doctor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
)

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
	pipelines := pipeline.ScanPipelinesDir(opts.PipelinesDir)
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

package pipeline

import (
	"fmt"
	"strings"
)

// StepFilterConfig holds the configuration for step filtering.
// Include and Exclude are mutually exclusive.
type StepFilterConfig struct {
	Include []string // Run only these steps (--steps flag)
	Exclude []string // Skip these steps (-x/--exclude flag)
}

// IsEmpty returns true if no filtering is configured.
func (c StepFilterConfig) IsEmpty() bool {
	return len(c.Include) == 0 && len(c.Exclude) == 0
}

// ValidateFilterCombination checks that the filter config is valid in combination
// with the --from-step flag. Rules:
//   - --steps + --exclude = error (mutually exclusive)
//   - --from-step + --steps = error (conflicting semantics)
//   - --from-step + --exclude = ok
func ValidateFilterCombination(config StepFilterConfig, fromStep string) error {
	if len(config.Include) > 0 && len(config.Exclude) > 0 {
		return fmt.Errorf("--steps and --exclude are mutually exclusive; use one or the other")
	}
	if fromStep != "" && len(config.Include) > 0 {
		return fmt.Errorf("--from-step and --steps cannot be combined; use --from-step with --exclude instead")
	}
	return nil
}

// ValidateStepNames checks that all names exist in the pipeline.
// Returns an error listing available steps if any name is invalid.
func ValidateStepNames(names []string, p *Pipeline) error {
	available := make(map[string]bool, len(p.Steps))
	for _, step := range p.Steps {
		available[step.ID] = true
	}

	var invalid []string
	for _, name := range names {
		if !available[name] {
			invalid = append(invalid, name)
		}
	}

	if len(invalid) > 0 {
		stepIDs := make([]string, len(p.Steps))
		for i, step := range p.Steps {
			stepIDs[i] = step.ID
		}
		return fmt.Errorf("unknown step(s): %s (available: %s)",
			strings.Join(invalid, ", "),
			strings.Join(stepIDs, ", "))
	}
	return nil
}

// ApplyStepFilter filters the sorted step list according to the config.
// Returns the filtered steps, the IDs of skipped steps, and any error.
// An error is returned if the filter would result in zero steps to run.
func ApplyStepFilter(steps []*Step, config StepFilterConfig) ([]*Step, []string, error) {
	if config.IsEmpty() {
		return steps, nil, nil
	}

	var filtered []*Step
	var skipped []string

	if len(config.Include) > 0 {
		includeSet := make(map[string]bool, len(config.Include))
		for _, name := range config.Include {
			includeSet[name] = true
		}
		for _, step := range steps {
			if includeSet[step.ID] {
				filtered = append(filtered, step)
			} else {
				skipped = append(skipped, step.ID)
			}
		}
	} else if len(config.Exclude) > 0 {
		excludeSet := make(map[string]bool, len(config.Exclude))
		for _, name := range config.Exclude {
			excludeSet[name] = true
		}
		for _, step := range steps {
			if excludeSet[step.ID] {
				skipped = append(skipped, step.ID)
			} else {
				filtered = append(filtered, step)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, nil, fmt.Errorf("step filter would exclude all steps; nothing to run")
	}

	return filtered, skipped, nil
}

// ValidateFilteredArtifacts checks that steps remaining after filtering don't
// depend on artifacts from skipped steps unless those artifacts already exist
// on disk from prior runs. The workspaceRoot is used to locate prior artifacts.
func ValidateFilteredArtifacts(remaining []*Step, skippedIDs []string, p *Pipeline, workspaceRoot string) error {
	if len(skippedIDs) == 0 {
		return nil
	}

	skippedSet := make(map[string]bool, len(skippedIDs))
	for _, id := range skippedIDs {
		skippedSet[id] = true
	}

	var missing []string
	for _, step := range remaining {
		for _, art := range step.Memory.InjectArtifacts {
			if art.Step != "" && skippedSet[art.Step] && !art.Optional {
				missing = append(missing, fmt.Sprintf(
					"step %q requires artifact %q from skipped step %q",
					step.ID, art.Artifact, art.Step))
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("artifact dependency conflict:\n  %s\n\nHint: run the skipped steps first, or use --force to skip validation",
			strings.Join(missing, "\n  "))
	}

	return nil
}

// ParseStepList splits a comma-separated string into a slice of step names,
// trimming whitespace and filtering empty strings.
func ParseStepList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

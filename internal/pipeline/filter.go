package pipeline

import (
	"fmt"
	"strings"
)

// StepFilter controls which steps are included or excluded from pipeline execution.
// Include and Exclude are mutually exclusive — setting both is an error.
type StepFilter struct {
	Include []string
	Exclude []string
}

// Mode returns the active filter mode: "include", "exclude", or "none".
func (f *StepFilter) Mode() string {
	if len(f.Include) > 0 {
		return "include"
	}
	if len(f.Exclude) > 0 {
		return "exclude"
	}
	return "none"
}

// IsActive returns true if the filter has any include or exclude entries.
func (f *StepFilter) IsActive() bool {
	return f.Mode() != "none"
}

// Validate checks that the filter configuration is valid:
// - Include and Exclude are mutually exclusive
// - All named steps exist in the pipeline
func (f *StepFilter) Validate(steps []*Step) error {
	if len(f.Include) > 0 && len(f.Exclude) > 0 {
		return fmt.Errorf("--steps and --exclude are mutually exclusive: cannot use both at the same time")
	}

	if !f.IsActive() {
		return nil
	}

	// Build set of valid step names
	valid := make(map[string]bool, len(steps))
	for _, s := range steps {
		valid[s.ID] = true
	}

	// Validate the active list
	names := f.Include
	flag := "--steps"
	if f.Mode() == "exclude" {
		names = f.Exclude
		flag = "--exclude"
	}

	var invalid []string
	for _, name := range names {
		if !valid[name] {
			invalid = append(invalid, name)
		}
	}

	if len(invalid) > 0 {
		available := make([]string, 0, len(steps))
		for _, s := range steps {
			available = append(available, s.ID)
		}
		return fmt.Errorf("unknown step(s) in %s: %s (available: %s)",
			flag, strings.Join(invalid, ", "), strings.Join(available, ", "))
	}

	return nil
}

// Apply filters the topologically-sorted step list based on include/exclude mode.
// Returns an error if the filter would result in an empty step list.
func (f *StepFilter) Apply(steps []*Step) ([]*Step, error) {
	if !f.IsActive() {
		return steps, nil
	}

	var filtered []*Step

	switch f.Mode() {
	case "include":
		include := make(map[string]bool, len(f.Include))
		for _, name := range f.Include {
			include[name] = true
		}
		for _, s := range steps {
			if include[s.ID] {
				filtered = append(filtered, s)
			}
		}

	case "exclude":
		exclude := make(map[string]bool, len(f.Exclude))
		for _, name := range f.Exclude {
			exclude[name] = true
		}
		for _, s := range steps {
			if !exclude[s.ID] {
				filtered = append(filtered, s)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("step filter excluded all steps — nothing to execute")
	}

	return filtered, nil
}

// ValidateDependencies checks that all filtered steps have their dependency
// artifacts available — either from another step in the filtered set or from
// existing workspace artifacts (artifactPaths maps "stepID:artifactName" to a path).
func (f *StepFilter) ValidateDependencies(filtered []*Step, allSteps []*Step, artifactPaths map[string]string) error {
	if !f.IsActive() {
		return nil
	}

	// Build set of steps that will execute
	executing := make(map[string]bool, len(filtered))
	for _, s := range filtered {
		executing[s.ID] = true
	}

	// Build map of step ID → output artifact names for all steps
	stepArtifacts := make(map[string][]string)
	for _, s := range allSteps {
		for _, art := range s.OutputArtifacts {
			stepArtifacts[s.ID] = append(stepArtifacts[s.ID], art.Name)
		}
	}

	// For each filtered step, check that its inject_artifacts dependencies are satisfied
	var missing []string
	for _, step := range filtered {
		for _, inject := range step.Memory.InjectArtifacts {
			depStep := inject.Step
			if executing[depStep] {
				continue // dependency step is in the filtered set — will produce the artifact
			}

			// Check if the artifact exists from a prior run
			key := fmt.Sprintf("%s:%s", depStep, inject.Artifact)
			if _, ok := artifactPaths[key]; ok {
				continue // artifact available from workspace
			}

			missing = append(missing, fmt.Sprintf("%s (from skipped step '%s')", inject.Artifact, depStep))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing artifacts for filtered steps: %s\nRun the skipped steps first or use --from-step to resume with prior workspace artifacts",
			strings.Join(missing, ", "))
	}

	return nil
}

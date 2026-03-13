package pipeline

import (
	"fmt"
	"strings"
)

// StepFilter controls which steps are included or excluded from pipeline execution.
// Include and Exclude are mutually exclusive — setting both is an error.
type StepFilter struct {
	Include []string // Run only these steps (--steps flag)
	Exclude []string // Skip these steps (-x/--exclude flag)
}

// IsActive returns true if any filter criteria are set.
func (f *StepFilter) IsActive() bool {
	return f != nil && (len(f.Include) > 0 || len(f.Exclude) > 0)
}

// Validate checks that the filter is internally consistent and that all referenced
// step names exist in the pipeline. Returns an error with available step names
// if any are invalid.
func (f *StepFilter) Validate(p *Pipeline) error {
	if f == nil {
		return nil
	}

	// Mutual exclusivity check
	if len(f.Include) > 0 && len(f.Exclude) > 0 {
		return fmt.Errorf("--steps and --exclude are mutually exclusive; use one or the other")
	}

	// Build step name lookup
	validSteps := make(map[string]bool, len(p.Steps))
	for _, step := range p.Steps {
		validSteps[step.ID] = true
	}

	available := formatStepNames(p)

	// Validate include step names
	for _, name := range f.Include {
		if !validSteps[name] {
			return fmt.Errorf("unknown step %q in --steps; available steps: %s", name, available)
		}
	}

	// Validate exclude step names
	for _, name := range f.Exclude {
		if !validSteps[name] {
			return fmt.Errorf("unknown step %q in --exclude; available steps: %s", name, available)
		}
	}

	return nil
}

// ValidateCombinations checks that the step filter is compatible with other flags.
// Specifically, --from-step + --steps is rejected (conflicting semantics),
// while --from-step + --exclude is allowed.
func (f *StepFilter) ValidateCombinations(fromStep string) error {
	if f == nil || fromStep == "" {
		return nil
	}

	if len(f.Include) > 0 {
		return fmt.Errorf("--from-step and --steps are mutually exclusive; --from-step resumes from a point, --steps selects specific steps")
	}

	// --from-step + --exclude is valid: resume from a step, skip specific later steps
	return nil
}

// Apply filters the topologically sorted step list, returning only the steps
// that should be executed. When Include is set, only named steps are kept.
// When Exclude is set, named steps are removed.
func (f *StepFilter) Apply(steps []*Step) []*Step {
	if f == nil || (!f.IsActive()) {
		return steps
	}

	if len(f.Include) > 0 {
		include := make(map[string]bool, len(f.Include))
		for _, name := range f.Include {
			include[name] = true
		}
		var result []*Step
		for _, step := range steps {
			if include[step.ID] {
				result = append(result, step)
			}
		}
		return result
	}

	if len(f.Exclude) > 0 {
		exclude := make(map[string]bool, len(f.Exclude))
		for _, name := range f.Exclude {
			exclude[name] = true
		}
		var result []*Step
		for _, step := range steps {
			if !exclude[step.ID] {
				result = append(result, step)
			}
		}
		return result
	}

	return steps
}

// ValidateDependencies checks that every step in the filtered set has its
// declared dependencies satisfied either by another step in the filtered set
// or by existing workspace artifacts from a prior run. The artifactExists
// function is called to check whether a step's output artifacts are available
// on disk (same mechanism used by ResumeManager).
func (f *StepFilter) ValidateDependencies(filtered []*Step, p *Pipeline, artifactExists func(stepID string) bool) error {
	if f == nil || !f.IsActive() {
		return nil
	}

	// Build lookup of steps in the filtered set
	inFiltered := make(map[string]bool, len(filtered))
	for _, step := range filtered {
		inFiltered[step.ID] = true
	}

	// Build full step lookup from pipeline
	allSteps := make(map[string]*Step, len(p.Steps))
	for i := range p.Steps {
		allSteps[p.Steps[i].ID] = &p.Steps[i]
	}

	for _, step := range filtered {
		fullStep := allSteps[step.ID]
		if fullStep == nil {
			continue
		}
		for _, dep := range fullStep.Dependencies {
			if inFiltered[dep] {
				continue // dependency is in filtered set — will be executed
			}
			// Dependency is not in filtered set — check if artifacts exist from prior run
			if artifactExists != nil && artifactExists(dep) {
				continue
			}
			return fmt.Errorf("step %q depends on %q which is excluded and has no prior artifacts; either include %q or run it first",
				step.ID, dep, dep)
		}
	}

	return nil
}

// ShouldRun returns true if the given step ID should be executed under this filter.
func (f *StepFilter) ShouldRun(stepID string) bool {
	if f == nil || !f.IsActive() {
		return true
	}

	if len(f.Include) > 0 {
		for _, name := range f.Include {
			if name == stepID {
				return true
			}
		}
		return false
	}

	if len(f.Exclude) > 0 {
		for _, name := range f.Exclude {
			if name == stepID {
				return false
			}
		}
		return true
	}

	return true
}

// ParseStepFilter creates a StepFilter from comma-separated step name strings.
// Empty strings produce nil fields (no filter).
func ParseStepFilter(steps, exclude string) *StepFilter {
	f := &StepFilter{}
	if steps != "" {
		f.Include = splitAndTrim(steps)
	}
	if exclude != "" {
		f.Exclude = splitAndTrim(exclude)
	}
	if len(f.Include) == 0 && len(f.Exclude) == 0 {
		return nil
	}
	return f
}

// splitAndTrim splits a comma-separated string and trims whitespace from each element.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// formatStepNames returns a comma-separated list of step names from a pipeline.
func formatStepNames(p *Pipeline) string {
	names := make([]string, len(p.Steps))
	for i, step := range p.Steps {
		names[i] = step.ID
	}
	return strings.Join(names, ", ")
}

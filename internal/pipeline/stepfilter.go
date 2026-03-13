package pipeline

import (
	"fmt"
	"strings"
)

// StepFilter controls which steps in a pipeline are executed.
// Include and Exclude are mutually exclusive — setting both is an error.
type StepFilter struct {
	Include []string // Run only these steps
	Exclude []string // Skip these steps
}

// IsActive returns true if the filter has any include or exclude entries.
func (f *StepFilter) IsActive() bool {
	if f == nil {
		return false
	}
	return len(f.Include) > 0 || len(f.Exclude) > 0
}

// Validate checks that Include and Exclude are not both set.
func (f *StepFilter) Validate() error {
	if f == nil {
		return nil
	}
	if len(f.Include) > 0 && len(f.Exclude) > 0 {
		return fmt.Errorf("--steps and --exclude are mutually exclusive")
	}
	return nil
}

// ValidateWithFromStep checks that the filter is compatible with --from-step.
// --from-step + --steps is invalid (conflicting semantics).
// --from-step + --exclude is valid (resume, then skip specific steps).
func (f *StepFilter) ValidateWithFromStep(fromStep string) error {
	if f == nil || fromStep == "" {
		return nil
	}
	if len(f.Include) > 0 {
		return fmt.Errorf("--from-step and --steps are mutually exclusive")
	}
	return nil
}

// ValidateStepNames checks that all step names in Include and Exclude
// exist in the pipeline. Returns an error listing invalid names and
// all available step names.
func (f *StepFilter) ValidateStepNames(steps []Step) error {
	if f == nil {
		return nil
	}

	available := make(map[string]bool, len(steps))
	for _, s := range steps {
		available[s.ID] = true
	}

	var invalid []string
	for _, name := range f.Include {
		if !available[name] {
			invalid = append(invalid, name)
		}
	}
	for _, name := range f.Exclude {
		if !available[name] {
			invalid = append(invalid, name)
		}
	}

	if len(invalid) > 0 {
		names := make([]string, 0, len(steps))
		for _, s := range steps {
			names = append(names, s.ID)
		}
		return fmt.Errorf("unknown step(s): %s (available: %s)",
			strings.Join(invalid, ", "),
			strings.Join(names, ", "))
	}

	return nil
}

// Apply filters the topologically-sorted step list based on Include or Exclude.
// The returned slice preserves topological order.
func (f *StepFilter) Apply(sorted []*Step) []*Step {
	if f == nil || !f.IsActive() {
		return sorted
	}

	if len(f.Include) > 0 {
		include := make(map[string]bool, len(f.Include))
		for _, name := range f.Include {
			include[name] = true
		}
		var result []*Step
		for _, step := range sorted {
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
		for _, step := range sorted {
			if !exclude[step.ID] {
				result = append(result, step)
			}
		}
		return result
	}

	return sorted
}

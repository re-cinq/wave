package worksource

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// validateSpec rejects malformed BindingSpec values at the service boundary.
//
// Rules:
//   - Forge, RepoPattern, PipelineName must be non-empty.
//   - Trigger must be one of the documented constants.
//   - RepoPattern must be a valid path.Match glob; double-star (**) is
//     rejected because path.Match does not support it and silent acceptance
//     would surprise callers.
//   - LabelFilter entries must be non-empty.
func validateSpec(spec BindingSpec) error {
	if strings.TrimSpace(spec.Forge) == "" {
		return errors.New("worksource: forge required")
	}
	if strings.TrimSpace(spec.PipelineName) == "" {
		return errors.New("worksource: pipeline_name required")
	}
	if strings.TrimSpace(spec.RepoPattern) == "" {
		return errors.New("worksource: repo_pattern required")
	}
	if _, ok := triggerToState(spec.Trigger); !ok {
		return fmt.Errorf("worksource: unknown trigger %q (want on-demand|on-label|on-open|scheduled)", spec.Trigger)
	}
	if strings.Contains(spec.RepoPattern, "**") {
		return fmt.Errorf("worksource: repo_pattern %q uses ** which path.Match does not support", spec.RepoPattern)
	}
	// path.Match validates pattern syntax against an arbitrary input. A
	// malformed pattern returns ErrBadPattern regardless of the input.
	if _, err := path.Match(spec.RepoPattern, "probe"); err != nil {
		return fmt.Errorf("worksource: malformed repo_pattern %q: %w", spec.RepoPattern, err)
	}
	for i, lbl := range spec.LabelFilter {
		if strings.TrimSpace(lbl) == "" {
			return fmt.Errorf("worksource: label_filter[%d] is empty", i)
		}
	}
	return nil
}

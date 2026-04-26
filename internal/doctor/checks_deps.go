package doctor

import (
	"fmt"

	"github.com/recinq/wave/internal/suggest"
	"github.com/recinq/wave/internal/tools"
)

func checkRequiredTools(opts *Options) []suggest.CheckResult {
	required := collectRequiredTools(opts.PipelinesDir)
	if len(required) == 0 {
		return []suggest.CheckResult{{
			Name:     "Required Tools",
			Category: "system",
			Status:   suggest.StatusOK,
			Message:  "No tools required by pipelines",
		}}
	}

	checks := tools.CheckOnPath(opts.lookPath, required)
	results := make([]suggest.CheckResult, 0, len(checks))
	for _, c := range checks {
		if c.Found {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Tool: %s", c.Name),
				Category: "system",
				Status:   suggest.StatusOK,
				Message:  fmt.Sprintf("Tool %q available", c.Name),
			})
		} else {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Tool: %s", c.Name),
				Category: "system",
				Status:   suggest.StatusErr,
				Message:  fmt.Sprintf("Required tool %q not found on PATH", c.Name),
				Fix:      fmt.Sprintf("Install %s", c.Name),
			})
		}
	}
	return results
}

func checkRequiredSkills(opts *Options) []suggest.CheckResult {
	skills := collectRequiredSkills(opts.PipelinesDir)
	if len(skills) == 0 {
		return []suggest.CheckResult{{
			Name:     "Required Skills",
			Category: "system",
			Status:   suggest.StatusOK,
			Message:  "No skills required by pipelines",
		}}
	}

	var results []suggest.CheckResult
	for name, check := range skills {
		if check == "" {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   suggest.StatusWarn,
				Message:  fmt.Sprintf("Skill %q has no check command", name),
			})
			continue
		}

		err := opts.runCmd("sh", "-c", check)
		if err != nil {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   suggest.StatusErr,
				Message:  fmt.Sprintf("Skill %q not installed", name),
				Fix:      fmt.Sprintf("Install skill %q or run 'wave run' with auto-install", name),
			})
		} else {
			results = append(results, suggest.CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   suggest.StatusOK,
				Message:  fmt.Sprintf("Skill %q installed", name),
			})
		}
	}
	return results
}

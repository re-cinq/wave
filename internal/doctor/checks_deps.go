package doctor

import (
	"fmt"

	"github.com/recinq/wave/internal/checks"
	"github.com/recinq/wave/internal/tools"
)

func checkRequiredTools(opts *Options) []CheckResult {
	required := collectRequiredTools(opts.PipelinesDir)
	if len(required) == 0 {
		return []CheckResult{{
			Name:     "Required Tools",
			Category: "system",
			Status:   StatusOK,
			Message:  "No tools required by pipelines",
		}}
	}

	checks := tools.CheckOnPath(opts.lookPath, required)
	results := make([]CheckResult, 0, len(checks))
	for _, c := range checks {
		if c.Found {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Tool: %s", c.Name),
				Category: "system",
				Status:   StatusOK,
				Message:  fmt.Sprintf("Tool %q available", c.Name),
			})
		} else {
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Tool: %s", c.Name),
				Category: "system",
				Status:   StatusErr,
				Message:  fmt.Sprintf("Required tool %q not found on PATH", c.Name),
				Fix:      fmt.Sprintf("Install %s", c.Name),
			})
		}
	}
	return results
}

func checkRequiredSkills(opts *Options) []CheckResult {
	skills := collectRequiredSkills(opts.PipelinesDir)
	if len(skills) == 0 {
		return []CheckResult{{
			Name:     "Required Skills",
			Category: "system",
			Status:   StatusOK,
			Message:  "No skills required by pipelines",
		}}
	}

	var results []CheckResult
	for name, check := range skills {
		status := checks.Skill(opts.runCmd, check)
		switch {
		case !status.HasCheck:
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   StatusWarn,
				Message:  fmt.Sprintf("Skill %q has no check command", name),
			})
		case !status.Installed:
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   StatusErr,
				Message:  fmt.Sprintf("Skill %q not installed", name),
				Fix:      fmt.Sprintf("Install skill %q or run 'wave run' with auto-install", name),
			})
		default:
			results = append(results, CheckResult{
				Name:     fmt.Sprintf("Skill: %s", name),
				Category: "system",
				Status:   StatusOK,
				Message:  fmt.Sprintf("Skill %q installed", name),
			})
		}
	}
	return results
}

package commands

// performDryRun renders the execution plan for a pipeline without running
// any steps. Each step prints its persona/sub-pipeline, dependencies,
// adapter and tool permissions, mounted directories, memory strategy,
// inject/output artifacts, and contract policy. When a step filter is
// active, [RUN]/[SKIP] markers and downstream artifact warnings are shown.
// Finally, dry-run composition validation is delegated to
// pipeline.NewDryRunValidator and printed to stderr; an error is returned
// when the validator reports any errors so the caller can exit non-zero.

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
)

func performDryRun(p *pipeline.Pipeline, m *manifest.Manifest, filter *pipeline.StepFilter) error {
	fmt.Fprintf(os.Stderr, "Dry run for pipeline: %s\n", p.Metadata.Name)
	fmt.Fprintf(os.Stderr, "Description: %s\n", p.Metadata.Description)
	fmt.Fprintf(os.Stderr, "Steps: %d\n\n", len(p.Steps))
	fmt.Fprintf(os.Stderr, "Execution plan:\n")

	for i, step := range p.Steps {
		// Show [SKIP] or [RUN] status when a filter is active
		status := ""
		if filter != nil && filter.IsActive() {
			if filter.ShouldRun(step.ID) {
				status = " [RUN]"
			} else {
				status = " [SKIP]"
			}
		}
		if step.SubPipeline != "" {
			fmt.Fprintf(os.Stderr, "  %d. %s (pipeline: %s)%s\n", i+1, step.ID, step.SubPipeline, status)
		} else {
			fmt.Fprintf(os.Stderr, "  %d. %s (persona: %s)%s\n", i+1, step.ID, step.Persona, status)
		}

		if len(step.Dependencies) > 0 {
			fmt.Fprintf(os.Stderr, "     Dependencies: %v\n", step.Dependencies)
		}

		persona := m.GetPersona(step.Persona)
		if persona != nil {
			fmt.Fprintf(os.Stderr, "     Adapter: %s  Temp: %.1f\n", persona.Adapter, persona.Temperature)
			fmt.Fprintf(os.Stderr, "     System prompt: %s\n", persona.SystemPromptFile)
			if len(persona.Permissions.AllowedTools) > 0 {
				fmt.Fprintf(os.Stderr, "     Allowed tools: %v\n", persona.Permissions.AllowedTools)
			}
			if len(persona.Permissions.Deny) > 0 {
				fmt.Fprintf(os.Stderr, "     Denied tools: %v\n", persona.Permissions.Deny)
			}
		}

		if len(step.Workspace.Mount) > 0 {
			for _, mount := range step.Workspace.Mount {
				fmt.Fprintf(os.Stderr, "     Mount: %s → %s (%s)\n", mount.Source, mount.Target, mount.Mode)
			}
		}

		fmt.Fprintf(os.Stderr, "     Workspace: .agents/workspaces/%s/%s/\n", p.Metadata.Name, step.ID)

		if step.Memory.Strategy != "" {
			fmt.Fprintf(os.Stderr, "     Memory: %s\n", step.Memory.Strategy)
		}

		if len(step.Memory.InjectArtifacts) > 0 {
			for _, art := range step.Memory.InjectArtifacts {
				fmt.Fprintf(os.Stderr, "     Inject: %s:%s as %s\n", art.Step, art.Artifact, art.As)
			}
		}

		if len(step.OutputArtifacts) > 0 {
			for _, art := range step.OutputArtifacts {
				fmt.Fprintf(os.Stderr, "     Output: %s → %s (%s)\n", art.Name, art.Path, art.Type)
			}
		}

		if step.Handover.Contract.Type != "" {
			fmt.Fprintf(os.Stderr, "     Contract: %s", step.Handover.Contract.Type)
			if step.Handover.Contract.OnFailure != "" {
				fmt.Fprintf(os.Stderr, " (on_failure: %s, max_retries: %d)", step.Handover.Contract.OnFailure, step.Handover.Contract.MaxRetries)
			}
			fmt.Fprintln(os.Stderr)
		}

		fmt.Fprintln(os.Stderr)
	}

	// Show artifact warnings when a filter is active
	if filter != nil && filter.IsActive() {
		skippedSteps := make(map[string]bool)
		for _, step := range p.Steps {
			if !filter.ShouldRun(step.ID) {
				skippedSteps[step.ID] = true
			}
		}
		var warnings []string
		for _, step := range p.Steps {
			if !filter.ShouldRun(step.ID) {
				continue
			}
			for _, dep := range step.Dependencies {
				if skippedSteps[dep] {
					warnings = append(warnings, fmt.Sprintf("  ⚠ Step %q depends on skipped step %q — ensure prior artifacts exist", step.ID, dep))
				}
			}
		}
		if len(warnings) > 0 {
			fmt.Fprintln(os.Stderr, "Artifact warnings:")
			for _, w := range warnings {
				fmt.Fprintln(os.Stderr, w)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	// Run composition validation and report findings.
	validator := pipeline.NewDryRunValidator(pipelinesDir())
	report := validator.Validate(p, m)
	fmt.Fprint(os.Stderr, "\n")
	fmt.Fprint(os.Stderr, report.Format())

	if report.HasErrors() {
		return fmt.Errorf("dry-run validation found %d error(s) — pipeline is not safe to execute", report.ErrorCount())
	}
	return nil
}

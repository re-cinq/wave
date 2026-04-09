package preflight

import (
	"fmt"
	"strings"

	"github.com/recinq/wave/internal/forge"
)

// forgeCLITools is the set of CLI tool names that indicate a forge dependency.
var forgeCLITools = []string{"gh", "glab", "tea", "bb"}

// forgeTemplateVars is the set of template variables that indicate a forge dependency.
var forgeTemplateVars = []string{"{{ forge.cli_tool }}", "{{ forge.pr_command }}"}

// forgePipelinePrefixes is the set of pipeline name prefixes that indicate a forge-specific pipeline.
var forgePipelinePrefixes = []string{"gh-", "gl-", "bb-", "gt-"}

// ForgeStepCheck describes a step that was checked for forge dependencies.
type ForgeStepCheck struct {
	StepID string // Step ID
	Tool   string // Detected forge CLI tool (e.g. "gh")
	Reason string // Human-readable reason
}

// ForgeError represents a preflight failure due to forge-dependent steps running
// without a configured forge.
type ForgeError struct {
	Steps []ForgeStepCheck
}

// Error implements the error interface.
func (e *ForgeError) Error() string {
	if len(e.Steps) == 1 {
		return e.Steps[0].Reason
	}
	msgs := make([]string, len(e.Steps))
	for i, s := range e.Steps {
		msgs[i] = s.Reason
	}
	return strings.Join(msgs, "; ")
}

// ForgeStepInput contains the information needed to check a step for forge dependencies.
type ForgeStepInput struct {
	StepID       string
	PersonaTools []string // AllowedTools from the step's persona
	PromptSource string   // Inline prompt text (Exec.Source)
}

// CheckForgeSteps checks whether any pipeline steps depend on a forge CLI tool
// when the detected forge is ForgeLocal. Returns a ForgeError listing all
// forge-dependent steps, or nil if none are found.
//
// A step is considered forge-dependent if:
//   - Its persona has Bash(gh *), Bash(glab *), Bash(tea *), or Bash(bb *) in allowed_tools
//   - Its prompt source contains {{ forge.cli_tool }} or {{ forge.pr_command }}
//
// Pipeline-name-based detection (forge prefixes like gh-, gl-, bb-, gt-) is
// checked separately via CheckForgePipelineName.
func CheckForgeSteps(forgeInfo forge.ForgeInfo, steps []ForgeStepInput) *ForgeError {
	if forgeInfo.Type != forge.ForgeLocal {
		return nil
	}

	var failed []ForgeStepCheck
	for _, step := range steps {
		if tool, ok := detectForgeToolInAllowedTools(step.PersonaTools); ok {
			failed = append(failed, ForgeStepCheck{
				StepID: step.StepID,
				Tool:   tool,
				Reason: fmt.Sprintf("step %q requires forge CLI %q but no forge is configured (detected: local). Use a forge-independent pipeline or configure a git remote", step.StepID, tool),
			})
			continue
		}
		if tmplVar := detectForgeTemplateVar(step.PromptSource); tmplVar != "" {
			failed = append(failed, ForgeStepCheck{
				StepID: step.StepID,
				Reason: fmt.Sprintf("step %q uses forge template variable %q but no forge is configured (detected: local). Use a forge-independent pipeline or configure a git remote", step.StepID, tmplVar),
			})
		}
	}

	if len(failed) == 0 {
		return nil
	}
	return &ForgeError{Steps: failed}
}

// CheckForgePipelineName returns a ForgeError if the pipeline name starts with
// a forge-specific prefix (gh-, gl-, bb-, gt-) and the forge is local.
func CheckForgePipelineName(forgeInfo forge.ForgeInfo, pipelineName string) *ForgeError {
	if forgeInfo.Type != forge.ForgeLocal {
		return nil
	}

	for _, prefix := range forgePipelinePrefixes {
		if strings.HasPrefix(pipelineName, prefix) {
			return &ForgeError{
				Steps: []ForgeStepCheck{
					{
						Reason: fmt.Sprintf("pipeline %q has forge prefix %q but no forge is configured (detected: local). Use a forge-independent pipeline or configure a git remote", pipelineName, strings.TrimSuffix(prefix, "-")),
					},
				},
			}
		}
	}
	return nil
}

// detectForgeToolInAllowedTools checks if any allowed tool pattern references a forge CLI.
// Returns the tool name and true if found.
func detectForgeToolInAllowedTools(tools []string) (string, bool) {
	for _, tool := range tools {
		for _, cli := range forgeCLITools {
			// Match patterns like "Bash(gh *)", "Bash(gh)", "Bash(gh pr *)"
			if strings.Contains(tool, "Bash("+cli+" ") || strings.Contains(tool, "Bash("+cli+")") || tool == "Bash("+cli+"*)" {
				return cli, true
			}
		}
	}
	return "", false
}

// detectForgeTemplateVar checks if a prompt string contains forge template variables.
// Returns the first matched variable or empty string.
func detectForgeTemplateVar(prompt string) string {
	for _, v := range forgeTemplateVars {
		if strings.Contains(prompt, v) {
			return v
		}
	}
	return ""
}

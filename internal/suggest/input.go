package suggest

import (
	"regexp"
	"strings"
)

// InputType classifies the kind of input passed to `wave run`.
type InputType string

const (
	InputTypeIssueURL InputType = "issue_url" // GitHub/GitLab/Gitea issue URL
	InputTypePRURL    InputType = "pr_url"    // Pull request / merge request URL
	InputTypeRepoRef  InputType = "repo_ref"  // owner/repo #123 format
	InputTypeFreeText InputType = "free_text" // Everything else
)

// repoRefPattern matches "owner/repo #123" or "owner/repo 123" patterns.
var repoRefPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+\s+#?\d+$`)

// ClassifyInput determines the InputType for a given input string.
func ClassifyInput(input string) InputType {
	input = strings.TrimSpace(input)
	if input == "" {
		return InputTypeFreeText
	}

	lower := strings.ToLower(input)
	isHTTP := strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")

	// PR/MR URLs (more specific than issue URLs).
	if isHTTP && (strings.Contains(lower, "/pull/") ||
		strings.Contains(lower, "/pulls/") ||
		strings.Contains(lower, "/merge_requests/")) {
		return InputTypePRURL
	}

	// Issue URLs.
	if isHTTP && strings.Contains(lower, "/issues/") {
		return InputTypeIssueURL
	}

	// Repo ref pattern (owner/repo #123).
	if repoRefPattern.MatchString(input) {
		return InputTypeRepoRef
	}

	return InputTypeFreeText
}

// SuggestPipelineForInput returns recommended pipeline names based on input type.
// The first element is the highest-priority suggestion.
func SuggestPipelineForInput(inputType InputType) []string {
	switch inputType {
	case InputTypeIssueURL:
		return []string{"impl-issue", "plan-research"}
	case InputTypePRURL:
		return []string{"ops-pr-review"}
	case InputTypeRepoRef:
		return []string{"impl-issue"}
	default:
		return nil
	}
}

// InputMismatch describes a mismatch between the input type and the selected pipeline.
type InputMismatch struct {
	InputType       InputType
	Pipeline        string
	SuggestedReason string
}

// CheckInputPipelineMismatch checks whether the given input seems mismatched with
// the selected pipeline. Returns nil if no mismatch detected.
func CheckInputPipelineMismatch(input, pipelineName string) *InputMismatch {
	inputType := ClassifyInput(input)
	suggested := SuggestPipelineForInput(inputType)

	// No suggestions means nothing to compare against.
	if len(suggested) == 0 {
		return nil
	}

	// Check if the selected pipeline is among the suggestions.
	for _, s := range suggested {
		if s == pipelineName {
			return nil
		}
	}

	// Build a human-readable mismatch reason.
	var reason string
	// Only warn once per run for free text (not per step)
	switch inputType {
	case InputTypeIssueURL:
		// Suppress: issue URLs are commonly passed as context to any pipeline.
		reason = ""
	case InputTypePRURL:
		reason = "input looks like a PR URL — consider using: " + strings.Join(suggested, ", ")
	case InputTypeRepoRef:
		reason = "input looks like a repo reference — consider using: " + strings.Join(suggested, ", ")
	case InputTypeFreeText:
		// Skip warning - will be handled at run start, not per step warning
		reason = ""
	}

	// No message to show — caller suppressed this input type.
	if reason == "" {
		return nil
	}

	return &InputMismatch{
		InputType:       inputType,
		Pipeline:        pipelineName,
		SuggestedReason: reason,
	}
}

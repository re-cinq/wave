package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FormatValidator performs strict format validation beyond schema compliance
// Ensures outputs are production-ready and properly formatted
type FormatValidator struct{}

// FormatValidationConfig defines format validation rules
type FormatValidationConfig struct {
	Type           string                 `json:"type" yaml:"type"` // "github_issue", "github_pr", "code", etc.
	StrictFormat   bool                   `json:"strict_format" yaml:"strict_format"`
	RequiredFields []string               `json:"required_fields,omitempty" yaml:"required_fields,omitempty"`
	FormatRules    map[string]interface{} `json:"format_rules,omitempty" yaml:"format_rules,omitempty"`
}

// Validate performs format validation
func (v *FormatValidator) Validate(cfg ContractConfig, workspacePath string) error {
	outputPath := filepath.Join(workspacePath, cfg.Source)
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	// Parse output as JSON
	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		return &ValidationError{
			ContractType: "format",
			Message:      "output is not valid JSON",
			Details:      []string{err.Error()},
			Retryable:    true,
		}
	}

	// Get format type from config or infer from schema
	formatType := inferFormatType(cfg)

	switch formatType {
	case "github_issue":
		return v.validateGitHubIssueFormat(output)
	case "github_pr":
		return v.validateGitHubPRFormat(output)
	case "implementation_results":
		return v.validateImplementationResults(output)
	case "analysis":
		return v.validateAnalysisFormat(output)
	default:
		// Generic format validation
		return v.validateGenericFormat(output, cfg)
	}
}

// validateGitHubIssueFormat ensures GitHub issue creation format is correct
func (v *FormatValidator) validateGitHubIssueFormat(output map[string]interface{}) error {
	violations := []string{}

	// Check title format
	if title, ok := output["title"].(string); ok {
		if len(title) < 10 {
			violations = append(violations, "issue title too short (minimum 10 characters)")
		}
		if len(title) > 200 {
			violations = append(violations, "issue title too long (maximum 200 characters)")
		}
		// Check for placeholder text
		if strings.Contains(strings.ToLower(title), "[todo") || strings.Contains(strings.ToLower(title), "placeholder") {
			violations = append(violations, "issue title contains placeholder text")
		}
	} else {
		violations = append(violations, "missing or invalid title field")
	}

	// Check body format
	if body, ok := output["body"].(string); ok {
		if len(body) < 50 {
			violations = append(violations, "issue body too short (minimum 50 characters)")
		}

		// Check for required sections in markdown body
		requiredSections := []string{"Description", "Acceptance Criteria"}
		for _, section := range requiredSections {
			pattern := regexp.MustCompile(fmt.Sprintf(`(?i)##?\s*%s`, section))
			if !pattern.MatchString(body) {
				violations = append(violations, fmt.Sprintf("issue body missing '%s' section", section))
			}
		}

		// Check for placeholder content
		placeholders := []string{"[TODO", "[PLACEHOLDER", "TBD", "FIXME"}
		for _, ph := range placeholders {
			if strings.Contains(body, ph) {
				violations = append(violations, fmt.Sprintf("issue body contains placeholder: %s", ph))
			}
		}
	} else {
		violations = append(violations, "missing or invalid body field")
	}

	// Check labels
	if labels, ok := output["labels"].([]interface{}); ok {
		if len(labels) == 0 {
			violations = append(violations, "no labels specified (at least one recommended)")
		}
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "format",
			Message:      "GitHub issue format validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// validateGitHubPRFormat ensures pull request format is production-ready
func (v *FormatValidator) validateGitHubPRFormat(output map[string]interface{}) error {
	violations := []string{}

	// Check title
	if title, ok := output["title"].(string); ok {
		if len(title) < 10 {
			violations = append(violations, "PR title too short (minimum 10 characters)")
		}
		if len(title) > 72 {
			violations = append(violations, "PR title too long (maximum 72 characters for optimal display)")
		}

		// Check for conventional commit format (optional but recommended)
		conventionalPattern := regexp.MustCompile(`^(feat|fix|docs|style|refactor|test|chore)(\(.+\))?:\s+.+`)
		if !conventionalPattern.MatchString(title) {
			// Warning, not error
			violations = append(violations, "PR title should follow conventional commit format (feat/fix/docs/etc: description)")
		}

		// Check for placeholder text
		if strings.Contains(strings.ToLower(title), "[todo") {
			violations = append(violations, "PR title contains placeholder text")
		}
	}

	// Check body structure
	if body, ok := output["body"].(string); ok {
		if len(body) < 100 {
			violations = append(violations, "PR body too short (minimum 100 characters for clarity)")
		}

		// Check for required sections
		requiredSections := []string{"Summary", "Changes", "Testing"}
		for _, section := range requiredSections {
			pattern := regexp.MustCompile(fmt.Sprintf(`(?i)##?\s*%s`, section))
			if !pattern.MatchString(body) {
				violations = append(violations, fmt.Sprintf("PR body missing '%s' section", section))
			}
		}

		// Check for issue references (Closes #123 or Fixes #456)
		if !regexp.MustCompile(`(?i)(closes|fixes|resolves)\s+#\d+`).MatchString(body) {
			violations = append(violations, "PR body should reference related issues (Closes #123)")
		}

		// Check for checklist
		if !strings.Contains(body, "- [") && !strings.Contains(body, "* [") {
			violations = append(violations, "PR body should include a checklist")
		}
	}

	// Check branch names
	if head, ok := output["head"].(string); ok {
		if head == "main" || head == "master" {
			violations = append(violations, "cannot create PR from main/master branch")
		}
	}

	if base, ok := output["base"].(string); ok {
		if base == "" {
			violations = append(violations, "base branch is required")
		}
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "format",
			Message:      "GitHub PR format validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// validateImplementationResults ensures code implementation outputs are complete
func (v *FormatValidator) validateImplementationResults(output map[string]interface{}) error {
	violations := []string{}

	// Check files_changed array
	if filesChanged, ok := output["files_changed"].([]interface{}); ok {
		if len(filesChanged) == 0 {
			violations = append(violations, "no files changed - implementation appears empty")
		}
	} else {
		violations = append(violations, "missing files_changed array")
	}

	// Check test status
	if testsPassed, ok := output["tests_passed"].(bool); ok {
		if !testsPassed {
			violations = append(violations, "tests are failing - implementation must pass all tests")
		}
	}

	// Check build status
	if builds, ok := output["builds_successfully"].(bool); ok {
		if !builds {
			violations = append(violations, "code does not build - must compile/run successfully")
		}
	}

	// Check for implementation notes
	if notes, ok := output["implementation_notes"].(string); ok {
		if len(notes) < 20 {
			violations = append(violations, "implementation_notes too brief (minimum 20 characters)")
		}
	} else {
		violations = append(violations, "missing implementation_notes")
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "format",
			Message:      "implementation results validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// validateAnalysisFormat ensures analysis outputs are comprehensive
func (v *FormatValidator) validateAnalysisFormat(output map[string]interface{}) error {
	violations := []string{}

	// Check for minimum content in key analysis fields
	analysisFields := []string{"findings", "recommendations", "analysis", "summary"}
	foundField := false

	for _, field := range analysisFields {
		if value, ok := output[field]; ok {
			foundField = true
			// Check if it's a string with content
			if str, ok := value.(string); ok {
				if len(str) < 50 {
					violations = append(violations, fmt.Sprintf("%s field too brief (minimum 50 characters)", field))
				}
			}
			// Check if it's an array with items
			if arr, ok := value.([]interface{}); ok {
				if len(arr) == 0 {
					violations = append(violations, fmt.Sprintf("%s array is empty", field))
				}
			}
		}
	}

	if !foundField {
		violations = append(violations, "missing analysis content (expected findings, recommendations, or similar fields)")
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "format",
			Message:      "analysis format validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// validateGenericFormat performs basic format validation
func (v *FormatValidator) validateGenericFormat(output map[string]interface{}, cfg ContractConfig) error {
	violations := []string{}

	// Check that output is not empty
	if len(output) == 0 {
		violations = append(violations, "output object is empty")
	}

	// Check for common placeholder patterns in all string values
	placeholders := []string{"TODO", "FIXME", "XXX", "[placeholder]", "TBD"}
	for key, value := range output {
		if str, ok := value.(string); ok {
			for _, ph := range placeholders {
				if strings.Contains(str, ph) {
					violations = append(violations, fmt.Sprintf("field '%s' contains placeholder text: %s", key, ph))
				}
			}
		}
	}

	if len(violations) > 0 {
		return &ValidationError{
			ContractType: "format",
			Message:      "generic format validation failed",
			Details:      violations,
			Retryable:    true,
		}
	}

	return nil
}

// inferFormatType determines format type from schema path or config
func inferFormatType(cfg ContractConfig) string {
	if cfg.SchemaPath == "" {
		return "generic"
	}

	schemaName := filepath.Base(cfg.SchemaPath)
	schemaName = strings.TrimSuffix(schemaName, ".schema.json")

	// Map schema names to format types
	formatMap := map[string]string{
		"github-issue-analysis":  "github_issue",
		"github-pr-draft":        "github_pr",
		"github-pr-info":         "github_pr",
		"implementation-results": "implementation_results",
		"analysis":               "analysis",
		"findings":               "analysis",
	}

	if formatType, ok := formatMap[schemaName]; ok {
		return formatType
	}

	// Check for patterns in schema name
	if strings.Contains(schemaName, "issue") {
		return "github_issue"
	}
	if strings.Contains(schemaName, "pr") {
		return "github_pr"
	}
	if strings.Contains(schemaName, "implementation") {
		return "implementation_results"
	}

	return "generic"
}

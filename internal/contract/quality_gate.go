package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// QualityGate represents a validation gate that ensures output quality
type QualityGate interface {
	// Check runs the quality gate check and returns violations
	Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error)
	// Name returns the name of this quality gate
	Name() string
}

// QualityGateConfig defines configuration for quality gate validation
type QualityGateConfig struct {
	Type       string                 `json:"type" yaml:"type"`
	Target     string                 `json:"target,omitempty" yaml:"target,omitempty"`           // File or pattern to check
	Required   bool                   `json:"required,omitempty" yaml:"required,omitempty"`       // Is passing required?
	Threshold  int                    `json:"threshold,omitempty" yaml:"threshold,omitempty"`     // Minimum score (0-100)
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"` // Gate-specific parameters
}

// QualityViolation represents a quality gate failure
type QualityViolation struct {
	Gate        string   `json:"gate"`
	Severity    string   `json:"severity"`    // "error", "warning", "info"
	Message     string   `json:"message"`
	Details     []string `json:"details,omitempty"`
	Score       int      `json:"score,omitempty"`       // Actual score if applicable
	Threshold   int      `json:"threshold,omitempty"`   // Required threshold
	Suggestions []string `json:"suggestions,omitempty"` // How to fix
}

// QualityGateResult aggregates results from multiple quality gates
type QualityGateResult struct {
	Passed     bool               `json:"passed"`
	Violations []QualityViolation `json:"violations"`
	Score      int                `json:"score"`      // Overall quality score (0-100)
	Details    map[string]int     `json:"details"`    // Gate-specific scores
}

// FormatGuidance returns a human-readable guidance message for fixing violations
func (r *QualityGateResult) FormatGuidance() string {
	if r.Passed {
		return "All quality gates passed"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Quality gates failed (score: %d/100)\n\n", r.Score))

	errorCount := 0
	warningCount := 0

	for _, v := range r.Violations {
		if v.Severity == "error" {
			errorCount++
		} else if v.Severity == "warning" {
			warningCount++
		}
	}

	sb.WriteString(fmt.Sprintf("Summary: %d errors, %d warnings\n\n", errorCount, warningCount))
	sb.WriteString("Violations:\n")

	for i, v := range r.Violations {
		icon := "•"
		if v.Severity == "error" {
			icon = "✗"
		} else if v.Severity == "warning" {
			icon = "⚠"
		}

		sb.WriteString(fmt.Sprintf("%d. %s [%s] %s\n", i+1, icon, v.Gate, v.Message))

		if len(v.Details) > 0 {
			for _, detail := range v.Details {
				sb.WriteString(fmt.Sprintf("   - %s\n", detail))
			}
		}

		if len(v.Suggestions) > 0 {
			sb.WriteString("   Suggestions:\n")
			for _, suggestion := range v.Suggestions {
				sb.WriteString(fmt.Sprintf("   → %s\n", suggestion))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// QualityGateRunner runs multiple quality gates and aggregates results
type QualityGateRunner struct {
	gates []QualityGate
}

// NewQualityGateRunner creates a new quality gate runner with standard gates
func NewQualityGateRunner() *QualityGateRunner {
	return &QualityGateRunner{
		gates: []QualityGate{
			&LinkValidationGate{},
			&MarkdownStructureGate{},
			&JSONStructureGate{},
			&RequiredFieldsGate{},
			&ContentCompletenessGate{},
			&VerificationGate{},
		},
	}
}

// RunGates executes all quality gates and returns aggregated results
func (r *QualityGateRunner) RunGates(workspacePath string, configs []QualityGateConfig) (*QualityGateResult, error) {
	result := &QualityGateResult{
		Passed:     true,
		Violations: []QualityViolation{},
		Details:    make(map[string]int),
	}

	for _, config := range configs {
		gate := r.findGate(config.Type)
		if gate == nil {
			continue // Skip unknown gate types
		}

		violations, err := gate.Check(workspacePath, config)
		if err != nil {
			return nil, fmt.Errorf("gate %s failed: %w", config.Type, err)
		}

		// Filter violations by required status
		for _, v := range violations {
			if config.Required && v.Severity == "error" {
				result.Passed = false
			}
			result.Violations = append(result.Violations, v)
		}
	}

	// Calculate overall score
	result.Score = r.calculateOverallScore(result.Violations, configs)

	return result, nil
}

func (r *QualityGateRunner) findGate(gateType string) QualityGate {
	for _, gate := range r.gates {
		if gate.Name() == gateType {
			return gate
		}
	}
	return nil
}

func (r *QualityGateRunner) calculateOverallScore(violations []QualityViolation, configs []QualityGateConfig) int {
	if len(configs) == 0 {
		return 100
	}

	totalScore := 0
	for _, config := range configs {
		gateScore := 100 // Default to perfect
		for _, v := range violations {
			if v.Gate == config.Type && v.Score > 0 {
				gateScore = v.Score
				break
			}
		}
		totalScore += gateScore
	}

	return totalScore / len(configs)
}

// LinkValidationGate validates that all links in markdown files are valid
type LinkValidationGate struct{}

func (g *LinkValidationGate) Name() string {
	return "link_validation"
}

func (g *LinkValidationGate) Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error) {
	violations := []QualityViolation{}

	target := config.Target
	if target == "" {
		target = "*.md" // Default to all markdown files
	}

	pattern := filepath.Join(workspacePath, target)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Regex patterns for markdown links
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	refLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\[([^\]]*)\]`)

	brokenLinks := []string{}

	for _, file := range matches {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		text := string(content)

		// Check inline links
		inlineLinks := linkPattern.FindAllStringSubmatch(text, -1)
		for _, match := range inlineLinks {
			if len(match) > 2 {
				link := match[2]
				if !g.isValidLink(link, workspacePath) {
					brokenLinks = append(brokenLinks, fmt.Sprintf("%s: %s", filepath.Base(file), link))
				}
			}
		}

		// Check reference links
		refLinks := refLinkPattern.FindAllStringSubmatch(text, -1)
		for _, match := range refLinks {
			if len(match) > 2 {
				ref := match[2]
				if ref == "" {
					ref = match[1]
				}
				// Check if reference is defined
				refDefPattern := regexp.MustCompile(fmt.Sprintf(`\[%s\]:\s*(.+)`, regexp.QuoteMeta(ref)))
				if !refDefPattern.MatchString(text) {
					brokenLinks = append(brokenLinks, fmt.Sprintf("%s: undefined reference [%s]", filepath.Base(file), ref))
				}
			}
		}
	}

	if len(brokenLinks) > 0 {
		severity := "warning"
		if config.Required {
			severity = "error"
		}

		violations = append(violations, QualityViolation{
			Gate:     g.Name(),
			Severity: severity,
			Message:  fmt.Sprintf("Found %d broken or invalid links", len(brokenLinks)),
			Details:  brokenLinks,
			Suggestions: []string{
				"Verify all link targets exist",
				"Use relative paths for local files",
				"Define all reference links",
			},
		})
	}

	return violations, nil
}

func (g *LinkValidationGate) isValidLink(link string, workspacePath string) bool {
	// Skip external links (assume valid for now)
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return true
	}

	// Skip anchors
	if strings.HasPrefix(link, "#") {
		return true
	}

	// Check if local file exists
	targetPath := filepath.Join(workspacePath, link)
	if _, err := os.Stat(targetPath); err == nil {
		return true
	}

	return false
}

// MarkdownStructureGate validates markdown document structure
type MarkdownStructureGate struct{}

func (g *MarkdownStructureGate) Name() string {
	return "markdown_structure"
}

func (g *MarkdownStructureGate) Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error) {
	violations := []QualityViolation{}

	target := config.Target
	if target == "" {
		return violations, nil // No target specified
	}

	filePath := filepath.Join(workspacePath, target)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	lines := strings.Split(text, "\n")

	// Check for proper heading hierarchy
	headingPattern := regexp.MustCompile(`^(#{1,6})\s+(.+)`)
	lastLevel := 0
	headingIssues := []string{}

	for i, line := range lines {
		if match := headingPattern.FindStringSubmatch(line); match != nil {
			level := len(match[1])
			if lastLevel > 0 && level > lastLevel+1 {
				headingIssues = append(headingIssues, fmt.Sprintf("Line %d: Heading level jumped from %d to %d", i+1, lastLevel, level))
			}
			lastLevel = level
		}
	}

	if len(headingIssues) > 0 {
		violations = append(violations, QualityViolation{
			Gate:     g.Name(),
			Severity: "warning",
			Message:  "Markdown heading hierarchy issues detected",
			Details:  headingIssues,
			Suggestions: []string{
				"Use sequential heading levels (h1, h2, h3, not h1, h3)",
				"Start with h1 and increment by one level at a time",
			},
		})
	}

	// Check for required sections (if specified in parameters)
	if requiredSections, ok := config.Parameters["required_sections"].([]interface{}); ok {
		missingSections := []string{}
		for _, section := range requiredSections {
			sectionName := fmt.Sprintf("%v", section)
			sectionPattern := regexp.MustCompile(fmt.Sprintf(`(?i)^#+\s+.*%s`, regexp.QuoteMeta(sectionName)))
			found := false
			for _, line := range lines {
				if sectionPattern.MatchString(line) {
					found = true
					break
				}
			}
			if !found {
				missingSections = append(missingSections, sectionName)
			}
		}

		if len(missingSections) > 0 {
			severity := "warning"
			if config.Required {
				severity = "error"
			}

			violations = append(violations, QualityViolation{
				Gate:     g.Name(),
				Severity: severity,
				Message:  "Required sections missing from markdown document",
				Details:  missingSections,
				Suggestions: []string{
					"Add the missing sections to the document",
					"Ensure section headings match the required format",
				},
			})
		}
	}

	return violations, nil
}

// JSONStructureGate validates JSON structure and formatting
type JSONStructureGate struct{}

func (g *JSONStructureGate) Name() string {
	return "json_structure"
}

func (g *JSONStructureGate) Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error) {
	violations := []QualityViolation{}

	target := config.Target
	if target == "" {
		return violations, nil
	}

	filePath := filepath.Join(workspacePath, target)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Check if valid JSON
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		violations = append(violations, QualityViolation{
			Gate:     g.Name(),
			Severity: "error",
			Message:  "Invalid JSON format",
			Details:  []string{err.Error()},
			Suggestions: []string{
				"Ensure the file contains valid JSON",
				"Check for missing commas, brackets, or quotes",
				"Use a JSON validator to identify syntax errors",
			},
		})
		return violations, nil
	}

	// Check formatting (if properly formatted, should parse the same way)
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return violations, nil
	}

	// Compare normalized versions
	original := strings.TrimSpace(string(content))
	formatted := strings.TrimSpace(string(prettyJSON))

	if original != formatted {
		violations = append(violations, QualityViolation{
			Gate:     g.Name(),
			Severity: "info",
			Message:  "JSON formatting could be improved",
			Suggestions: []string{
				"Use consistent indentation (2 spaces recommended)",
				"Run through a JSON formatter",
			},
		})
	}

	return violations, nil
}

// RequiredFieldsGate validates that required fields are present
type RequiredFieldsGate struct{}

func (g *RequiredFieldsGate) Name() string {
	return "required_fields"
}

func (g *RequiredFieldsGate) Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error) {
	violations := []QualityViolation{}

	target := config.Target
	if target == "" {
		return violations, nil
	}

	filePath := filepath.Join(workspacePath, target)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return violations, nil // Skip if not valid JSON
	}

	// Check for required fields from parameters
	if requiredFields, ok := config.Parameters["fields"].([]interface{}); ok {
		missingFields := []string{}
		for _, field := range requiredFields {
			fieldName := fmt.Sprintf("%v", field)
			if _, exists := data[fieldName]; !exists {
				missingFields = append(missingFields, fieldName)
			}
		}

		if len(missingFields) > 0 {
			severity := "warning"
			if config.Required {
				severity = "error"
			}

			violations = append(violations, QualityViolation{
				Gate:     g.Name(),
				Severity: severity,
				Message:  "Required fields missing from JSON",
				Details:  missingFields,
				Suggestions: []string{
					"Add the missing required fields",
					"Check the contract schema for field definitions",
				},
			})
		}
	}

	return violations, nil
}

// ContentCompletenessGate checks content length and completeness
type ContentCompletenessGate struct{}

func (g *ContentCompletenessGate) Name() string {
	return "content_completeness"
}

func (g *ContentCompletenessGate) Check(workspacePath string, config QualityGateConfig) ([]QualityViolation, error) {
	violations := []QualityViolation{}

	target := config.Target
	if target == "" {
		return violations, nil
	}

	filePath := filepath.Join(workspacePath, target)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	wordCount := len(strings.Fields(text))
	_ = len(strings.Split(text, "\n")) // lineCount - reserved for future use

	// Check minimum word count (if specified)
	if minWords, ok := config.Parameters["min_words"].(float64); ok {
		if wordCount < int(minWords) {
			severity := "warning"
			if config.Required {
				severity = "error"
			}

			violations = append(violations, QualityViolation{
				Gate:     g.Name(),
				Severity: severity,
				Message:  fmt.Sprintf("Content too short: %d words (minimum: %d)", wordCount, int(minWords)),
				Suggestions: []string{
					"Expand the content to provide more detail",
					"Add examples and explanations",
					"Include all required sections",
				},
			})
		}
	}

	// Check for placeholder content
	placeholders := []string{"TODO", "FIXME", "XXX", "TBD", "[placeholder]"}
	foundPlaceholders := []string{}
	for _, placeholder := range placeholders {
		if strings.Contains(text, placeholder) {
			foundPlaceholders = append(foundPlaceholders, placeholder)
		}
	}

	if len(foundPlaceholders) > 0 {
		violations = append(violations, QualityViolation{
			Gate:     g.Name(),
			Severity: "warning",
			Message:  "Placeholder content detected",
			Details:  foundPlaceholders,
			Suggestions: []string{
				"Replace placeholder text with actual content",
				"Complete all TODO and FIXME items",
			},
		})
	}

	// Calculate completeness score
	score := 100
	if len(foundPlaceholders) > 0 {
		score -= len(foundPlaceholders) * 10
	}
	if wordCount < 100 {
		score -= (100 - wordCount) / 2
	}
	if score < 0 {
		score = 0
	}

	if score < config.Threshold {
		violations = append(violations, QualityViolation{
			Gate:      g.Name(),
			Severity:  "warning",
			Message:   "Content completeness below threshold",
			Score:     score,
			Threshold: config.Threshold,
		})
	}

	return violations, nil
}

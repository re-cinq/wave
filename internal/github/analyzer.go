package github

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// IssueAnalysis contains the analysis results for an issue
type IssueAnalysis struct {
	Issue           *Issue                 `json:"issue"`
	QualityScore    int                    `json:"quality_score"` // 0-100
	Problems        []string               `json:"problems"`
	Recommendations []string               `json:"recommendations"`
	SuggestedTitle  string                 `json:"suggested_title,omitempty"`
	SuggestedBody   string                 `json:"suggested_body,omitempty"`
	SuggestedLabels []string               `json:"suggested_labels,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Analyzer provides issue analysis functionality
type Analyzer struct {
	client *Client
}

// NewAnalyzer creates a new issue analyzer
func NewAnalyzer(client *Client) *Analyzer {
	return &Analyzer{client: client}
}

// AnalyzeIssue performs comprehensive analysis on a GitHub issue
func (a *Analyzer) AnalyzeIssue(ctx context.Context, issue *Issue) *IssueAnalysis {
	analysis := &IssueAnalysis{
		Issue:        issue,
		QualityScore: 100,
		Problems:     []string{},
		Recommendations: []string{},
		Metadata:     make(map[string]interface{}),
	}

	// Analyze title
	a.analyzeTitle(issue, analysis)

	// Analyze body
	a.analyzeBody(issue, analysis)

	// Analyze labels
	a.analyzeLabels(issue, analysis)

	// Analyze metadata
	a.analyzeMetadata(issue, analysis)

	return analysis
}

// analyzeTitle checks the issue title for quality issues
func (a *Analyzer) analyzeTitle(issue *Issue, analysis *IssueAnalysis) {
	title := strings.TrimSpace(issue.Title)
	titleLen := len(title)

	// Check title length
	if titleLen == 0 {
		analysis.Problems = append(analysis.Problems, "Title is empty")
		analysis.QualityScore -= 50
		analysis.Recommendations = append(analysis.Recommendations, "Add a descriptive title")
		return
	}

	if titleLen < 10 {
		analysis.Problems = append(analysis.Problems, "Title is too short (less than 10 characters)")
		analysis.QualityScore -= 20
		analysis.Recommendations = append(analysis.Recommendations, "Expand title with more context")
	}

	if titleLen > 200 {
		analysis.Problems = append(analysis.Problems, "Title is too long (over 200 characters)")
		analysis.QualityScore -= 10
		analysis.Recommendations = append(analysis.Recommendations, "Shorten title and move details to description")
	}

	// Check for all lowercase
	if title == strings.ToLower(title) && titleLen > 5 {
		analysis.Problems = append(analysis.Problems, "Title is all lowercase")
		analysis.QualityScore -= 5
		analysis.Recommendations = append(analysis.Recommendations, "Use proper capitalization")
	}

	// Check for all uppercase
	if title == strings.ToUpper(title) && titleLen > 5 {
		analysis.Problems = append(analysis.Problems, "Title is all uppercase")
		analysis.QualityScore -= 10
		analysis.Recommendations = append(analysis.Recommendations, "Use normal capitalization instead of all caps")
	}

	// Check for vague terms
	vagueTerms := []string{"issue", "problem", "help", "bug?", "question"}
	titleLower := strings.ToLower(title)
	for _, term := range vagueTerms {
		if strings.Contains(titleLower, term) && titleLen < 30 {
			analysis.Problems = append(analysis.Problems, fmt.Sprintf("Title contains vague term '%s' without specifics", term))
			analysis.QualityScore -= 10
			analysis.Recommendations = append(analysis.Recommendations, "Add specific details about what's affected")
			break
		}
	}

	// Count words
	words := strings.Fields(title)
	if len(words) < 3 {
		analysis.Problems = append(analysis.Problems, "Title has very few words")
		analysis.QualityScore -= 15
		analysis.Recommendations = append(analysis.Recommendations, "Use a more descriptive title with context")
	}

	// Store metadata
	analysis.Metadata["title_length"] = titleLen
	analysis.Metadata["title_word_count"] = len(words)
}

// analyzeBody checks the issue body for quality issues
func (a *Analyzer) analyzeBody(issue *Issue, analysis *IssueAnalysis) {
	body := strings.TrimSpace(issue.Body)
	bodyLen := len(body)

	// Check if body is empty
	if bodyLen == 0 {
		analysis.Problems = append(analysis.Problems, "Description is empty")
		analysis.QualityScore -= 40
		analysis.Recommendations = append(analysis.Recommendations, "Add a detailed description with context, steps to reproduce, and expected behavior")
		analysis.Metadata["body_length"] = 0
		analysis.Metadata["has_code_blocks"] = false
		analysis.Metadata["has_lists"] = false
		return
	}

	// Check for very short descriptions
	if bodyLen < 50 {
		analysis.Problems = append(analysis.Problems, "Description is very short (less than 50 characters)")
		analysis.QualityScore -= 25
		analysis.Recommendations = append(analysis.Recommendations, "Expand description with more details")
	} else if bodyLen < 100 {
		analysis.Problems = append(analysis.Problems, "Description is short (less than 100 characters)")
		analysis.QualityScore -= 10
		analysis.Recommendations = append(analysis.Recommendations, "Consider adding more context or examples")
	}

	// Check for code blocks
	hasCodeBlocks := strings.Contains(body, "```") || strings.Contains(body, "    ")
	analysis.Metadata["has_code_blocks"] = hasCodeBlocks

	// Check for lists
	hasLists := strings.Contains(body, "\n- ") || strings.Contains(body, "\n* ") ||
		strings.Contains(body, "\n1. ") || strings.Contains(body, "\n2. ")
	analysis.Metadata["has_lists"] = hasLists

	// Check for structure keywords
	structureKeywords := []string{
		"steps to reproduce", "expected behavior", "actual behavior",
		"environment", "version", "screenshot", "reproduction",
	}
	bodyLower := strings.ToLower(body)
	hasStructure := false
	for _, keyword := range structureKeywords {
		if strings.Contains(bodyLower, keyword) {
			hasStructure = true
			break
		}
	}

	if !hasStructure && bodyLen > 100 {
		analysis.Recommendations = append(analysis.Recommendations, "Consider adding structured sections (Steps to Reproduce, Expected Behavior, etc.)")
		analysis.QualityScore -= 5
	}

	// Check for single sentence descriptions
	sentences := strings.Count(body, ".") + strings.Count(body, "!") + strings.Count(body, "?")
	if sentences < 2 && bodyLen > 20 {
		analysis.Problems = append(analysis.Problems, "Description appears to be a single sentence")
		analysis.QualityScore -= 10
		analysis.Recommendations = append(analysis.Recommendations, "Expand with multiple sentences providing context")
	}

	// Store metadata
	analysis.Metadata["body_length"] = bodyLen
	analysis.Metadata["body_sentence_count"] = sentences
	analysis.Metadata["has_structure"] = hasStructure
}

// analyzeLabels checks if the issue has appropriate labels
func (a *Analyzer) analyzeLabels(issue *Issue, analysis *IssueAnalysis) {
	labelCount := len(issue.Labels)
	analysis.Metadata["label_count"] = labelCount

	if labelCount == 0 {
		analysis.Problems = append(analysis.Problems, "No labels assigned")
		analysis.QualityScore -= 10
		analysis.Recommendations = append(analysis.Recommendations, "Add relevant labels (bug, enhancement, documentation, etc.)")
	}

	// Extract label names for analysis
	labelNames := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labelNames[i] = label.Name
	}
	analysis.Metadata["labels"] = labelNames
}

// analyzeMetadata analyzes other issue metadata
func (a *Analyzer) analyzeMetadata(issue *Issue, analysis *IssueAnalysis) {
	// Check if issue has assignees
	if len(issue.Assignees) == 0 {
		analysis.Metadata["has_assignee"] = false
	} else {
		analysis.Metadata["has_assignee"] = true
	}

	// Check if issue has milestone
	analysis.Metadata["has_milestone"] = issue.Milestone != nil

	// Check issue age
	age := issue.UpdatedAt.Sub(issue.CreatedAt)
	analysis.Metadata["age_hours"] = int(age.Hours())
	analysis.Metadata["comment_count"] = issue.Comments

	// Check if issue has been updated since creation
	if age < time.Minute {
		analysis.Metadata["recently_created"] = true
	}

	// Ensure quality score doesn't go below 0
	if analysis.QualityScore < 0 {
		analysis.QualityScore = 0
	}
}

// FindPoorQualityIssues finds issues that need improvement
func (a *Analyzer) FindPoorQualityIssues(ctx context.Context, owner, repo string, threshold int) ([]*IssueAnalysis, error) {
	// List all open issues
	issues, err := a.client.ListIssues(ctx, owner, repo, ListIssuesOptions{
		State:   "open",
		PerPage: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	var poorQualityIssues []*IssueAnalysis
	for _, issue := range issues {
		// Skip pull requests
		if issue.IsPullRequest() {
			continue
		}

		analysis := a.AnalyzeIssue(ctx, issue)
		if analysis.QualityScore < threshold {
			poorQualityIssues = append(poorQualityIssues, analysis)
		}
	}

	return poorQualityIssues, nil
}

// GenerateEnhancementSuggestions creates enhancement suggestions for an issue
func (a *Analyzer) GenerateEnhancementSuggestions(issue *Issue, analysis *IssueAnalysis) {
	// Generate title suggestions if needed
	if len(issue.Title) < 10 || strings.ToLower(issue.Title) == issue.Title {
		// Capitalize first letter and key words
		words := strings.Fields(issue.Title)
		for i, word := range words {
			if i == 0 || len(word) > 3 {
				words[i] = capitalize(word)
			}
		}
		analysis.SuggestedTitle = strings.Join(words, " ")
	}

	// Generate body template if body is missing or very short
	if len(issue.Body) < 100 {
		template := generateBodyTemplate(issue)
		if template != "" {
			analysis.SuggestedBody = template
		}
	}

	// Suggest labels based on content
	suggestedLabels := suggestLabels(issue)
	if len(suggestedLabels) > 0 {
		analysis.SuggestedLabels = suggestedLabels
	}
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// generateBodyTemplate creates a template for issue body
func generateBodyTemplate(issue *Issue) string {
	// If there's already some content, enhance it
	if len(issue.Body) > 0 {
		return fmt.Sprintf(`## Description
%s

## Additional Context
Please provide:
- Steps to reproduce (if applicable)
- Expected behavior
- Actual behavior
- Environment details (OS, version, etc.)
- Any relevant screenshots or logs
`, issue.Body)
	}

	// Create a basic template
	return `## Description
[Please provide a clear description of the issue]

## Steps to Reproduce
1.
2.
3.

## Expected Behavior
[What should happen?]

## Actual Behavior
[What actually happens?]

## Environment
- OS:
- Version:
- Browser (if applicable):

## Additional Context
[Any other relevant information]
`
}

// suggestLabels suggests labels based on issue content
func suggestLabels(issue *Issue) []string {
	var labels []string
	combined := strings.ToLower(issue.Title + " " + issue.Body)

	// Bug indicators
	bugKeywords := []string{"bug", "error", "crash", "broken", "fails", "failing", "doesn't work"}
	for _, keyword := range bugKeywords {
		if strings.Contains(combined, keyword) {
			labels = append(labels, "bug")
			break
		}
	}

	// Enhancement indicators
	enhancementKeywords := []string{"feature", "enhancement", "improve", "add", "support for"}
	for _, keyword := range enhancementKeywords {
		if strings.Contains(combined, keyword) {
			labels = append(labels, "enhancement")
			break
		}
	}

	// Documentation indicators
	docKeywords := []string{"documentation", "docs", "readme", "example", "tutorial"}
	for _, keyword := range docKeywords {
		if strings.Contains(combined, keyword) {
			labels = append(labels, "documentation")
			break
		}
	}

	// Question indicators
	questionKeywords := []string{"how to", "how do", "question", "help"}
	for _, keyword := range questionKeywords {
		if strings.Contains(combined, keyword) {
			labels = append(labels, "question")
			break
		}
	}

	return labels
}

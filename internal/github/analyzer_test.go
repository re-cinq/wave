package github

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIssueFilter_Matches(t *testing.T) {
	tests := []struct {
		name   string
		filter IssueFilter
		issue  *Issue
		want   bool
	}{
		{
			name: "matches short title",
			filter: IssueFilter{
				MaxTitleLength: 20,
			},
			issue: &Issue{
				Title: "Short",
				Body:  "Some body",
			},
			want: true,
		},
		{
			name: "rejects long title",
			filter: IssueFilter{
				MaxTitleLength: 20,
			},
			issue: &Issue{
				Title: "This is a very long title that exceeds the limit",
				Body:  "Some body",
			},
			want: false,
		},
		{
			name: "requires minimum body length",
			filter: IssueFilter{
				MinBodyLength: 50,
			},
			issue: &Issue{
				Title: "Title",
				Body:  "Short body",
			},
			want: false,
		},
		{
			name: "requires body",
			filter: IssueFilter{
				RequireBody: true,
			},
			issue: &Issue{
				Title: "Title",
				Body:  "",
			},
			want: false,
		},
		{
			name: "filters by state",
			filter: IssueFilter{
				States: []string{"open"},
			},
			issue: &Issue{
				Title: "Title",
				Body:  "Body",
				State: "closed",
			},
			want: false,
		},
		{
			name: "filters by labels",
			filter: IssueFilter{
				Labels: []string{"bug"},
			},
			issue: &Issue{
				Title: "Title",
				Body:  "Body",
				Labels: []*Label{
					{Name: "enhancement"},
				},
			},
			want: false,
		},
		{
			name: "matches with labels",
			filter: IssueFilter{
				Labels: []string{"bug"},
			},
			issue: &Issue{
				Title: "Title",
				Body:  "Body",
				Labels: []*Label{
					{Name: "bug"},
					{Name: "enhancement"},
				},
			},
			want: true,
		},
		{
			name:   "rejects pull requests",
			filter: IssueFilter{},
			issue: &Issue{
				Title: "Title",
				Body:  "Body",
				PullRequest: &struct {
					URL     string `json:"url"`
					HTMLURL string `json:"html_url"`
				}{
					URL: "https://api.github.com/pulls/1",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(tt.issue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAnalyzer_AnalyzeIssue(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	tests := []struct {
		name             string
		issue            *Issue
		expectLowScore   bool
		expectProblems   bool
		minQualityScore  int
	}{
		{
			name: "high quality issue",
			issue: &Issue{
				Title: "Add GitHub Integration for Wave Pipeline Orchestrator",
				Body: `## Description
This PR adds comprehensive GitHub integration to Wave.

## Changes
- GitHub API client
- Issue analysis
- PR creation

## Test Plan
Run the test suite with go test ./...`,
				Labels: []*Label{
					{Name: "enhancement"},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectLowScore:  false,
			expectProblems:  false,
			minQualityScore: 80,
		},
		{
			name: "empty title",
			issue: &Issue{
				Title:     "",
				Body:      "Some body text",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectLowScore:  true,
			expectProblems:  true,
			minQualityScore: 0,
		},
		{
			name: "short title",
			issue: &Issue{
				Title:     "bug",
				Body:      "This is a bug report with some details",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectLowScore:  true,
			expectProblems:  true,
			minQualityScore: 0,
		},
		{
			name: "empty body",
			issue: &Issue{
				Title:     "Good Title Here",
				Body:      "",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectLowScore:  true,
			expectProblems:  true,
			minQualityScore: 0,
		},
		{
			name: "very short body",
			issue: &Issue{
				Title:     "Good Title Here",
				Body:      "Short",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectLowScore:  true,
			expectProblems:  true,
			minQualityScore: 0,
		},
		{
			name: "all lowercase title",
			issue: &Issue{
				Title:     "this is all lowercase",
				Body:      "This is a longer body with more details about the issue",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectProblems: true,
		},
		{
			name: "all uppercase title",
			issue: &Issue{
				Title:     "THIS IS ALL UPPERCASE",
				Body:      "This is a longer body with more details about the issue",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectProblems: true,
		},
		{
			name: "vague title",
			issue: &Issue{
				Title:     "help",
				Body:      "I need help",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectLowScore: true,
			expectProblems: true,
		},
		{
			name: "no labels",
			issue: &Issue{
				Title:     "Good Title Here",
				Body:      "This is a decent body with enough content to pass basic checks",
				Labels:    []*Label{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectProblems: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzer.AnalyzeIssue(context.Background(), tt.issue)

			assert.NotNil(t, analysis)
			assert.Equal(t, tt.issue, analysis.Issue)

			if tt.expectLowScore {
				assert.Less(t, analysis.QualityScore, 70, "Quality score should be low")
			}

			if tt.expectProblems {
				assert.NotEmpty(t, analysis.Problems, "Should identify problems")
			}

			if tt.minQualityScore > 0 {
				assert.GreaterOrEqual(t, analysis.QualityScore, tt.minQualityScore)
			}

			// Quality score should be 0-100
			assert.GreaterOrEqual(t, analysis.QualityScore, 0)
			assert.LessOrEqual(t, analysis.QualityScore, 100)

			// Should have metadata
			assert.NotNil(t, analysis.Metadata)
		})
	}
}

func TestAnalyzer_GenerateEnhancementSuggestions(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	tests := []struct {
		name                 string
		issue                *Issue
		expectTitleSuggestion bool
		expectBodySuggestion  bool
		expectLabelSuggestion bool
	}{
		{
			name: "poor quality issue needs all enhancements",
			issue: &Issue{
				Title: "bug",
				Body:  "it doesnt work",
			},
			expectTitleSuggestion: true,
			expectBodySuggestion:  true,
			expectLabelSuggestion: true,
		},
		{
			name: "lowercase title needs capitalization",
			issue: &Issue{
				Title: "this is lowercase",
				Body:  "This is a longer description with enough content",
			},
			expectTitleSuggestion: true,
		},
		{
			name: "short body needs template",
			issue: &Issue{
				Title: "Good Title",
				Body:  "Short",
			},
			expectBodySuggestion: true,
		},
		{
			name: "bug keywords suggest bug label",
			issue: &Issue{
				Title: "Application crashes on startup",
				Body:  "The application crashes immediately when I try to start it",
			},
			expectLabelSuggestion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzer.AnalyzeIssue(context.Background(), tt.issue)
			analyzer.GenerateEnhancementSuggestions(tt.issue, analysis)

			if tt.expectTitleSuggestion {
				assert.NotEmpty(t, analysis.SuggestedTitle, "Should suggest title improvement")
			}

			if tt.expectBodySuggestion {
				assert.NotEmpty(t, analysis.SuggestedBody, "Should suggest body template")
			}

			if tt.expectLabelSuggestion {
				assert.NotEmpty(t, analysis.SuggestedLabels, "Should suggest labels")
			}
		})
	}
}

func TestSuggestLabels(t *testing.T) {
	tests := []struct {
		name          string
		issue         *Issue
		expectedLabel string
	}{
		{
			name: "bug keywords",
			issue: &Issue{
				Title: "Application crashes",
				Body:  "The app crashes when I click the button",
			},
			expectedLabel: "bug",
		},
		{
			name: "enhancement keywords",
			issue: &Issue{
				Title: "Add new feature",
				Body:  "Would be great to have support for X",
			},
			expectedLabel: "enhancement",
		},
		{
			name: "documentation keywords",
			issue: &Issue{
				Title: "Update README",
				Body:  "The documentation needs updating",
			},
			expectedLabel: "documentation",
		},
		{
			name: "question keywords",
			issue: &Issue{
				Title: "How to configure X?",
				Body:  "I need help understanding how to set this up",
			},
			expectedLabel: "question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := suggestLabels(tt.issue)
			assert.Contains(t, labels, tt.expectedLabel)
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "Hello"},
		{"HELLO", "HELLO"},
		{"h", "H"},
		{"", ""},
		{"123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalize(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateBodyTemplate(t *testing.T) {
	tests := []struct {
		name        string
		issue       *Issue
		expectEnhancement bool
	}{
		{
			name: "empty body gets full template",
			issue: &Issue{
				Title: "Test Issue",
				Body:  "",
			},
			expectEnhancement: true,
		},
		{
			name: "short body gets enhanced",
			issue: &Issue{
				Title: "Test Issue",
				Body:  "Short description",
			},
			expectEnhancement: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := generateBodyTemplate(tt.issue)
			if tt.expectEnhancement {
				assert.NotEmpty(t, template)
				assert.Contains(t, template, "Description")
			}
		})
	}
}

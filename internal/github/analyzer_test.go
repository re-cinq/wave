package github

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzer_AnalyzeIssue(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	tests := []struct {
		name            string
		issue           *Issue
		expectLowScore  bool
		expectProblems  bool
		minQualityScore int
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

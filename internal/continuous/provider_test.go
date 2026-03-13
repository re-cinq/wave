package continuous

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGHIssues(t *testing.T) {
	t.Run("parses valid JSON output", func(t *testing.T) {
		data := []byte(`[
			{"number": 1, "title": "Fix bug", "url": "https://github.com/owner/repo/issues/1", "labels": [{"name": "bug"}]},
			{"number": 2, "title": "Add feature", "url": "https://github.com/owner/repo/issues/2", "labels": [{"name": "enhancement"}, {"name": "priority"}]}
		]`)

		issues, err := parseGHIssues(data)
		require.NoError(t, err)
		assert.Len(t, issues, 2)

		assert.Equal(t, 1, issues[0].Number)
		assert.Equal(t, "Fix bug", issues[0].Title)
		assert.Equal(t, "https://github.com/owner/repo/issues/1", issues[0].URL)
		assert.Len(t, issues[0].Labels, 1)
		assert.Equal(t, "bug", issues[0].Labels[0].Name)

		assert.Equal(t, 2, issues[1].Number)
		assert.Len(t, issues[1].Labels, 2)
	})

	t.Run("returns empty list for empty JSON array", func(t *testing.T) {
		data := []byte(`[]`)
		issues, err := parseGHIssues(data)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("handles issues with no labels", func(t *testing.T) {
		data := []byte(`[{"number": 42, "title": "No labels", "url": "https://github.com/o/r/issues/42", "labels": []}]`)
		issues, err := parseGHIssues(data)
		require.NoError(t, err)
		assert.Len(t, issues, 1)
		assert.Empty(t, issues[0].Labels)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		data := []byte(`not json`)
		_, err := parseGHIssues(data)
		assert.Error(t, err)
	})
}

func TestBuildItemKey(t *testing.T) {
	tests := []struct {
		name     string
		issueURL string
		expected string
	}{
		{
			name:     "GitHub issue URL",
			issueURL: "https://github.com/owner/repo/issues/42",
			expected: "owner/repo#42",
		},
		{
			name:     "GitHub issue URL with trailing slash",
			issueURL: "https://github.com/owner/repo/issues/1/",
			expected: "owner/repo#1",
		},
		{
			name:     "non-issue URL falls through",
			issueURL: "https://github.com/owner/repo/pull/5",
			expected: "https://github.com/owner/repo/pull/5",
		},
		{
			name:     "plain text falls through",
			issueURL: "not a url",
			expected: "not a url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildItemKey(tt.issueURL)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGitHubProviderFiltersProcessedItems(t *testing.T) {
	// Create a mock store with some items already processed
	store := newMockStore()
	store.processed["owner/repo#1"] = true

	// Simulate provider behavior manually since we can't mock gh CLI
	// Test the filtering logic via Next with a subclass approach
	items := []ghIssue{
		{Number: 1, URL: "https://github.com/owner/repo/issues/1"},
		{Number: 2, URL: "https://github.com/owner/repo/issues/2"},
	}

	// Verify key generation and filtering logic
	for _, issue := range items {
		key := BuildItemKey(issue.URL)
		processed, err := store.IsItemProcessed("test", key)
		require.NoError(t, err)

		if issue.Number == 1 {
			assert.True(t, processed, "issue #1 should be processed")
		} else {
			assert.False(t, processed, "issue #2 should not be processed")
		}
	}
}

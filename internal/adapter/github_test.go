//go:build integration

package adapter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubAdapter_ParseOperation(t *testing.T) {
	adapter := &GitHubAdapter{}

	tests := []struct {
		name        string
		prompt      string
		expectType  string
		expectError bool
	}{
		{
			name:       "JSON operation",
			prompt:     `{"type":"list_issues","owner":"test","repo":"repo"}`,
			expectType: "list_issues",
		},
		{
			name:       "list issues natural language",
			prompt:     "list issues for owner/repo",
			expectType: "list_issues",
		},
		{
			name:       "scan issues",
			prompt:     "scan issues in owner/repo",
			expectType: "list_issues",
		},
		{
			name:       "analyze issues",
			prompt:     "analyze issues for owner/repo",
			expectType: "analyze_issues",
		},
		{
			name:       "find poor quality issues",
			prompt:     "find poor quality issues in owner/repo",
			expectType: "analyze_issues",
		},
		{
			name:       "update issue",
			prompt:     "update issue in owner/repo",
			expectType: "update_issue",
		},
		{
			name:       "create PR",
			prompt:     "create pr for owner/repo",
			expectType: "create_pr",
		},
		{
			name:        "unknown operation",
			prompt:      "do something random",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, err := adapter.parseOperation(tt.prompt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectType, op.Type)
			}
		})
	}
}

func TestExtractRepoInfo(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantOwner  string
		wantRepo   string
	}{
		{
			name:      "simple owner/repo",
			text:      "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "in sentence",
			text:      "list issues for github/wave repository",
			wantOwner: "github",
			wantRepo:  "wave",
		},
		{
			name:      "URL format",
			text:      "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "no match",
			text:      "some random text",
			wantOwner: "unknown",
			wantRepo:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := extractRepoInfo(tt.text)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

func TestIsValidRepoName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid name", "my-repo", true},
		{"valid with underscore", "my_repo", true},
		{"valid with dot", "my.repo", true},
		{"valid numbers", "repo123", true},
		{"empty", "", false},
		{"too long", string(make([]byte, 101)), false},
		{"invalid chars", "repo@name", false},
		{"spaces", "repo name", false},
		{"special chars", "repo!name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRepoName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitHubAdapter_FormatResult(t *testing.T) {
	adapter := &GitHubAdapter{}

	data := map[string]interface{}{
		"test": "value",
		"number": 123,
	}

	result, err := adapter.formatResult(data)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.NotEmpty(t, result.ResultContent)

	// Verify JSON is valid
	var decoded map[string]interface{}
	err = json.Unmarshal([]byte(result.ResultContent), &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "value", decoded["test"])
	assert.Equal(t, float64(123), decoded["number"])
}

func TestGitHubAdapter_Structure(t *testing.T) {
	// Test that the adapter can be created
	adapter := &GitHubAdapter{}
	assert.NotNil(t, adapter)
}

package continuous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildItemKeyNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard GitHub issue URL",
			input:    "https://github.com/re-cinq/wave/issues/201",
			expected: "re-cinq/wave#201",
		},
		{
			name:     "GitHub issue URL with trailing slash",
			input:    "https://github.com/re-cinq/wave/issues/201/",
			expected: "re-cinq/wave#201",
		},
		{
			name:     "different owner/repo",
			input:    "https://github.com/foo/bar/issues/1",
			expected: "foo/bar#1",
		},
		{
			name:     "non-GitHub URL returns original",
			input:    "https://gitlab.com/foo/bar/issues/1",
			expected: "https://gitlab.com/foo/bar/issues/1",
		},
		{
			name:     "non-issue GitHub path returns original",
			input:    "https://github.com/foo/bar/pulls/1",
			expected: "https://github.com/foo/bar/pulls/1",
		},
		{
			name:     "plain text returns as-is",
			input:    "some plain text",
			expected: "some plain text",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildItemKey(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

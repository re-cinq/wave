package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMatcher(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		stepID      string
		shouldMatch bool
		wantErr     bool
	}{
		{
			name:        "exact match with anchors",
			pattern:     "^implement$",
			stepID:      "implement",
			shouldMatch: true,
		},
		{
			name:        "exact match rejects substring",
			pattern:     "^implement$",
			stepID:      "implementor",
			shouldMatch: false,
		},
		{
			name:        "exact match rejects prefix",
			pattern:     "^implement$",
			stepID:      "pre-implement",
			shouldMatch: false,
		},
		{
			name:        "alternation matches first",
			pattern:     "implement|fix",
			stepID:      "implement",
			shouldMatch: true,
		},
		{
			name:        "alternation matches second",
			pattern:     "implement|fix",
			stepID:      "fix",
			shouldMatch: true,
		},
		{
			name:        "alternation rejects non-matching",
			pattern:     "implement|fix",
			stepID:      "deploy",
			shouldMatch: false,
		},
		{
			name:        "wildcard matches anything",
			pattern:     ".*",
			stepID:      "anything-goes-here",
			shouldMatch: true,
		},
		{
			name:        "wildcard matches empty string",
			pattern:     ".*",
			stepID:      "",
			shouldMatch: true,
		},
		{
			name:        "empty pattern matches everything",
			pattern:     "",
			stepID:      "any-step-id",
			shouldMatch: true,
		},
		{
			name:        "empty pattern matches empty step",
			pattern:     "",
			stepID:      "",
			shouldMatch: true,
		},
		{
			name:        "partial match via substring regex",
			pattern:     "impl",
			stepID:      "implement",
			shouldMatch: true,
		},
		{
			name:        "partial match via substring in the middle",
			pattern:     "step",
			stepID:      "my-step-name",
			shouldMatch: true,
		},
		{
			name:        "partial match rejects non-matching",
			pattern:     "deploy",
			stepID:      "implement",
			shouldMatch: false,
		},
		{
			name:    "invalid regex returns error",
			pattern: "[",
			wantErr: true,
		},
		{
			name:    "invalid regex unclosed group",
			pattern: "(",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewMatcher(tc.pattern)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, m)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, m)
			assert.Equal(t, tc.shouldMatch, m.Match(tc.stepID))
		})
	}
}

func TestMatcherNilRegexMatchesAll(t *testing.T) {
	m, err := NewMatcher("")
	require.NoError(t, err)
	assert.Nil(t, m.re, "empty pattern should result in nil regex")

	// Verify it matches various inputs
	assert.True(t, m.Match("anything"))
	assert.True(t, m.Match(""))
	assert.True(t, m.Match("complex-step-id-123"))
}

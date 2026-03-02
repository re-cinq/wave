package proposal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectForgeFromPrefix(t *testing.T) {
	tests := []struct {
		name     string
		pipeline string
		want     ForgeType
	}{
		{"github", "gh-implement", ForgeGitHub},
		{"github scope", "gh-scope", ForgeGitHub},
		{"gitlab", "gl-implement", ForgeGitLab},
		{"gitlab research", "gl-research", ForgeGitLab},
		{"gitea", "gt-implement", ForgeGitea},
		{"gitea refresh", "gt-refresh", ForgeGitea},
		{"bitbucket", "bb-implement", ForgeBitBkt},
		{"bitbucket rewrite", "bb-rewrite", ForgeBitBkt},
		{"agnostic refactor", "refactor", ForgeUnknown},
		{"agnostic test-gen", "test-gen", ForgeUnknown},
		{"agnostic dead-code", "dead-code", ForgeUnknown},
		{"agnostic doc-fix", "doc-fix", ForgeUnknown},
		{"empty string", "", ForgeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DetectForgeFromPrefix(tt.pipeline))
		})
	}
}

func TestIsForgeAgnostic(t *testing.T) {
	assert.True(t, IsForgeAgnostic("refactor"))
	assert.True(t, IsForgeAgnostic("test-gen"))
	assert.True(t, IsForgeAgnostic("dead-code"))
	assert.False(t, IsForgeAgnostic("gh-implement"))
	assert.False(t, IsForgeAgnostic("gl-scope"))
	assert.False(t, IsForgeAgnostic("gt-research"))
	assert.False(t, IsForgeAgnostic("bb-implement"))
}

func TestFilterByForge(t *testing.T) {
	entries := []CatalogEntry{
		{Name: "gh-implement"},
		{Name: "gh-scope"},
		{Name: "gl-implement"},
		{Name: "gt-implement"},
		{Name: "bb-implement"},
		{Name: "refactor"},
		{Name: "test-gen"},
		{Name: "dead-code"},
	}

	t.Run("github", func(t *testing.T) {
		filtered := FilterByForge(entries, ForgeGitHub)
		names := extractNames(filtered)
		assert.Contains(t, names, "gh-implement")
		assert.Contains(t, names, "gh-scope")
		assert.Contains(t, names, "refactor")
		assert.Contains(t, names, "test-gen")
		assert.Contains(t, names, "dead-code")
		assert.NotContains(t, names, "gl-implement")
		assert.NotContains(t, names, "gt-implement")
		assert.NotContains(t, names, "bb-implement")
	})

	t.Run("gitlab", func(t *testing.T) {
		filtered := FilterByForge(entries, ForgeGitLab)
		names := extractNames(filtered)
		assert.Contains(t, names, "gl-implement")
		assert.Contains(t, names, "refactor")
		assert.NotContains(t, names, "gh-implement")
		assert.NotContains(t, names, "bb-implement")
	})

	t.Run("unknown returns all", func(t *testing.T) {
		filtered := FilterByForge(entries, ForgeUnknown)
		assert.Len(t, filtered, len(entries))
	})

	t.Run("empty entries", func(t *testing.T) {
		filtered := FilterByForge(nil, ForgeGitHub)
		assert.Empty(t, filtered)
	})
}

func TestFilterByForgeReturnsCopy(t *testing.T) {
	entries := []CatalogEntry{{Name: "refactor"}}
	filtered := FilterByForge(entries, ForgeUnknown)
	filtered[0].Name = "modified"
	assert.Equal(t, "refactor", entries[0].Name)
}

func extractNames(entries []CatalogEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}

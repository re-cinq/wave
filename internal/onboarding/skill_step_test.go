package onboarding

import (
	"context"
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillSelectionStep_Name(t *testing.T) {
	step := &SkillSelectionStep{}
	assert.Equal(t, "Skill Selection", step.Name())
}

func TestSkillSelectionStep_NonInteractive(t *testing.T) {
	step := &SkillSelectionStep{}
	cfg := &WizardConfig{Interactive: false}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	skills, ok := result.Data["skills"].([]string)
	require.True(t, ok)
	assert.Empty(t, skills)
}

func TestSkillSelectionStep_NonInteractiveWithExisting(t *testing.T) {
	step := &SkillSelectionStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing: &manifest.Manifest{
			Skills: []string{"golang", "spec-kit"},
		},
	}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Non-interactive always returns empty skills regardless of existing
	skills, ok := result.Data["skills"].([]string)
	require.True(t, ok)
	assert.Empty(t, skills)
}

func TestParseTesslOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []skillSearchResult
	}{
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "blank lines only",
			input:    "\n\n\n",
			expected: nil,
		},
		{
			name:  "single skill with rating",
			input: "golang 4.5 A Go development skill",
			expected: []skillSearchResult{
				{Name: "golang", Description: "A Go development skill"},
			},
		},
		{
			name:  "single skill without rating (two fields)",
			input: "golang description",
			expected: []skillSearchResult{
				{Name: "golang", Description: "description"},
			},
		},
		{
			name:  "three fields with rating",
			input: "golang 4.5 A Go skill",
			expected: []skillSearchResult{
				{Name: "golang", Description: "A Go skill"},
			},
		},
		{
			name: "multiple skills",
			input: `golang 4.5 A Go development skill
spec-kit 4.2 Specification tools
react 3.8 React component generation`,
			expected: []skillSearchResult{
				{Name: "golang", Description: "A Go development skill"},
				{Name: "spec-kit", Description: "Specification tools"},
				{Name: "react", Description: "React component generation"},
			},
		},
		{
			name:     "single field line (ignored)",
			input:    "justname",
			expected: nil,
		},
		{
			name: "mixed valid and invalid lines",
			input: `golang 4.5 A Go skill
bad
spec-kit 4.0 Specs`,
			expected: []skillSearchResult{
				{Name: "golang", Description: "A Go skill"},
				{Name: "spec-kit", Description: "Specs"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parseTesslOutput(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestFindEcosystem(t *testing.T) {
	tests := []struct {
		value    string
		found    bool
		expected string
	}{
		{"tessl", true, "Tessl"},
		{"bmad", true, "BMAD"},
		{"openspec", true, "OpenSpec"},
		{"speckit", true, "Spec-Kit"},
		{"unknown", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			eco := findEcosystem(tt.value)
			if tt.found {
				require.NotNil(t, eco)
				assert.Equal(t, tt.expected, eco.Name)
			} else {
				assert.Nil(t, eco)
			}
		})
	}
}

func TestMergeSkills(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		new      []string
		expected []string
	}{
		{
			name:     "both empty",
			existing: nil,
			new:      nil,
			expected: nil,
		},
		{
			name:     "existing only",
			existing: []string{"golang", "spec-kit"},
			new:      nil,
			expected: []string{"golang", "spec-kit"},
		},
		{
			name:     "new only",
			existing: nil,
			new:      []string{"react", "vue"},
			expected: []string{"react", "vue"},
		},
		{
			name:     "no overlap",
			existing: []string{"golang"},
			new:      []string{"react"},
			expected: []string{"golang", "react"},
		},
		{
			name:     "with duplicates",
			existing: []string{"golang", "spec-kit"},
			new:      []string{"spec-kit", "react"},
			expected: []string{"golang", "spec-kit", "react"},
		},
		{
			name:     "all duplicates",
			existing: []string{"golang", "react"},
			new:      []string{"golang", "react"},
			expected: []string{"golang", "react"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeSkills(tt.existing, tt.new)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEcosystemDefinitions(t *testing.T) {
	// Verify all ecosystems are properly defined
	assert.Len(t, ecosystems, 4)

	// Verify tessl
	tessl := findEcosystem("tessl")
	require.NotNil(t, tessl)
	assert.Equal(t, "tessl", tessl.Dep.Binary)
	assert.Equal(t, "npm i -g @tessl/cli", tessl.Dep.Instructions)
	assert.False(t, tessl.InstallAll)

	// Verify BMAD
	bmad := findEcosystem("bmad")
	require.NotNil(t, bmad)
	assert.Equal(t, "npx", bmad.Dep.Binary)
	assert.True(t, bmad.InstallAll)

	// Verify OpenSpec
	openspec := findEcosystem("openspec")
	require.NotNil(t, openspec)
	assert.Equal(t, "openspec", openspec.Dep.Binary)
	assert.True(t, openspec.InstallAll)

	// Verify Spec-Kit
	speckit := findEcosystem("speckit")
	require.NotNil(t, speckit)
	assert.Equal(t, "specify", speckit.Dep.Binary)
	assert.True(t, speckit.InstallAll)
}

func TestSkillSelectionStep_ReconfigureShowsExisting(t *testing.T) {
	step := &SkillSelectionStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing: &manifest.Manifest{
			Skills: []string{"golang", "spec-kit"},
		},
	}

	// Non-interactive with reconfigure should not error
	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	skills, ok := result.Data["skills"].([]string)
	require.True(t, ok)
	assert.Empty(t, skills, "non-interactive reconfigure returns empty skills")
}

func TestSkillSelectionStep_MissingCLI(t *testing.T) {
	step := &SkillSelectionStep{
		LookPath: func(binary string) (string, error) {
			return "", fmt.Errorf("not found: %s", binary)
		},
	}
	cfg := &WizardConfig{Interactive: false}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Non-interactive skips entirely regardless of CLI availability
	skills, ok := result.Data["skills"].([]string)
	require.True(t, ok)
	assert.Empty(t, skills)
}

func TestDefaultCommandRunner(t *testing.T) {
	// Test with a command that should succeed
	ctx := context.Background()
	output, err := defaultCommandRunner(ctx, "echo", "hello")
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
}

func TestDefaultCommandRunner_Failure(t *testing.T) {
	// Test with a command that doesn't exist
	ctx := context.Background()
	_, err := defaultCommandRunner(ctx, "nonexistent-binary-that-does-not-exist")
	require.Error(t, err)
}

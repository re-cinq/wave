package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/doctor"
	"github.com/recinq/wave/internal/manifest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyzeCmd_Structure(t *testing.T) {
	cmd := NewAnalyzeCmd()
	assert.Equal(t, "analyze", cmd.Use)

	deepFlag := cmd.Flags().Lookup("deep")
	require.NotNil(t, deepFlag)
	assert.Equal(t, "false", deepFlag.DefValue)

	evolveFlag := cmd.Flags().Lookup("evolve")
	require.NotNil(t, evolveFlag)
	assert.Equal(t, "false", evolveFlag.DefValue)
}

func TestBuildAnalyzeResult_NilOntology(t *testing.T) {
	m := &manifest.Manifest{}
	profile := &doctor.ProjectProfile{FilesScanned: 10}

	result := buildAnalyzeResult(m, profile)
	assert.Empty(t, result.Telos)
	assert.Empty(t, result.Contexts)
	assert.Equal(t, 10, result.Profile.FilesScanned)
}

func TestBuildAnalyzeResult_WithOntology(t *testing.T) {
	m := &manifest.Manifest{
		Ontology: &manifest.Ontology{
			Telos: "Build a CLI tool",
			Contexts: []manifest.OntologyContext{
				{Name: "auth", Description: "Authentication"},
			},
			Conventions: map[string]string{
				"naming": "snake_case",
			},
		},
	}
	profile := &doctor.ProjectProfile{FilesScanned: 42}

	result := buildAnalyzeResult(m, profile)
	assert.Equal(t, "Build a CLI tool", result.Telos)
	require.Len(t, result.Contexts, 1)
	assert.Equal(t, "auth", result.Contexts[0].Name)
	assert.Equal(t, "Authentication", result.Contexts[0].Description)
	assert.Equal(t, "snake_case", result.Conventions["naming"])
}

func TestGenerateSkillContent(t *testing.T) {
	ctx := AnalyzeContext{
		Name:        "identity",
		Description: "User identity management",
		Packages:    []string{"internal/auth"},
		FileCount:   5,
		HasTests:    true,
	}

	content := generateSkillContent(ctx, "Build a secure app")
	assert.Contains(t, content, "# identity Context")
	assert.Contains(t, content, "Project telos: Build a secure app")
	assert.Contains(t, content, "User identity management")
	assert.Contains(t, content, "`internal/auth`")
	assert.Contains(t, content, "5 files")
	assert.Contains(t, content, "Test files detected")
}

func TestWriteContextSkills(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	require.NoError(t, os.MkdirAll(".agents/skills", 0755))

	result := &AnalyzeResult{
		Telos: "Test project",
		Contexts: []AnalyzeContext{
			{Name: "auth", Description: "Authentication"},
			{Name: "billing", Description: "Billing system"},
		},
	}

	written, err := writeContextSkills(result)
	require.NoError(t, err)
	assert.Len(t, written, 2)

	// Verify files exist
	for _, path := range written {
		_, err := os.Stat(path)
		assert.NoError(t, err, "skill file should exist: %s", path)
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(".agents", "skills", "wave-ctx-auth", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "# auth Context")
	assert.Contains(t, string(data), "Test project")
}

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte(""), 0644)
	_ = os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	assert.Equal(t, 2, countFiles(tmpDir))
	assert.Equal(t, 0, countFiles(filepath.Join(tmpDir, "nonexistent")))
}

func TestMatchesContext(t *testing.T) {
	assert.True(t, matchesContext("internal/auth", "auth", "auth"))
	assert.True(t, matchesContext("src/user-management", "user-management", "user/management"))
	assert.True(t, matchesContext("internal/user_management", "user-management", "user/management"))
	assert.False(t, matchesContext("internal/other", "auth", "auth"))
}

func TestAnalyzeCmd_DeepFlagErrors(t *testing.T) {
	cmd := NewAnalyzeCmd()
	rootCmd := &cobra.Command{}
	rootCmd.PersistentFlags().String("manifest", "", "")
	rootCmd.PersistentFlags().Bool("debug", false, "")
	rootCmd.PersistentFlags().String("output", "auto", "")
	rootCmd.PersistentFlags().Bool("verbose", false, "")
	rootCmd.PersistentFlags().Bool("json", false, "")
	rootCmd.PersistentFlags().Bool("quiet", false, "")
	rootCmd.PersistentFlags().Bool("no-color", false, "")
	rootCmd.PersistentFlags().Bool("no-tui", false, "")
	rootCmd.AddCommand(cmd)

	rootCmd.SetArgs([]string{"analyze", "--deep"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestAnalyzeCmd_EvolveFlagErrorsWithoutDB(t *testing.T) {
	// --evolve now routes to the evolve implementation, but it still errors
	// when no wave.yaml or state.db exists.
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	cmd := NewAnalyzeCmd()
	rootCmd := &cobra.Command{}
	rootCmd.PersistentFlags().String("manifest", "", "")
	rootCmd.PersistentFlags().Bool("debug", false, "")
	rootCmd.PersistentFlags().String("output", "auto", "")
	rootCmd.PersistentFlags().Bool("verbose", false, "")
	rootCmd.PersistentFlags().Bool("json", false, "")
	rootCmd.PersistentFlags().Bool("quiet", false, "")
	rootCmd.PersistentFlags().Bool("no-color", false, "")
	rootCmd.PersistentFlags().Bool("no-tui", false, "")
	rootCmd.AddCommand(cmd)

	rootCmd.SetArgs([]string{"analyze", "--evolve"})
	err := rootCmd.Execute()
	// Errors because no wave.yaml manifest exists in temp directory
	assert.Error(t, err)
}

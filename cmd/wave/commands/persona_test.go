package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRunPersonaCreate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runPersonaCreate("my-nav", "navigator")
	require.NoError(t, err)

	// Check .md file was created
	mdPath := filepath.Join(".agents", "personas", "my-nav.md")
	mdData, err := os.ReadFile(mdPath)
	require.NoError(t, err)
	assert.Contains(t, string(mdData), "Navigator", "system prompt should contain persona content")

	// Check .yaml file was created
	yamlPath := filepath.Join(".agents", "personas", "my-nav.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	// Verify it's valid YAML
	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(yamlData, &config))

	// Should have a description field from the template
	_, hasDesc := config["description"]
	assert.True(t, hasDesc, "config should have description field")
}

func TestRunPersonaCreate_TemplateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runPersonaCreate("my-persona", "nonexistent-persona")
	require.Error(t, err)

	cliErr, ok := err.(*CLIError)
	require.True(t, ok, "expected CLIError, got %T", err)
	assert.Equal(t, CodeInvalidArgs, cliErr.Code)
	assert.Contains(t, cliErr.Message, "nonexistent-persona")
}

func TestRunPersonaCreate_NameCollision_MD(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create the first time
	err = runPersonaCreate("dup-persona", "navigator")
	require.NoError(t, err)

	// Try to create again — should fail
	err = runPersonaCreate("dup-persona", "navigator")
	require.Error(t, err)

	cliErr, ok := err.(*CLIError)
	require.True(t, ok, "expected CLIError, got %T", err)
	assert.Equal(t, CodeInvalidArgs, cliErr.Code)
	assert.Contains(t, cliErr.Message, "already exists")
}

func TestRunPersonaCreate_MissingName(t *testing.T) {
	err := runPersonaCreate("", "navigator")
	require.Error(t, err)

	cliErr, ok := err.(*CLIError)
	require.True(t, ok, "expected CLIError, got %T", err)
	assert.Equal(t, CodeInvalidArgs, cliErr.Code)
	assert.Contains(t, cliErr.Message, "--name is required")
}

func TestRunPersonaCreate_InvalidName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCode string
	}{
		{"path traversal", "../evil", CodeSecurityViolation},
		{"absolute path", "/etc/passwd", CodeSecurityViolation},
		{"slash in name", "foo/bar", CodeSecurityViolation},
		{"backslash in name", "foo\\bar", CodeSecurityViolation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runPersonaCreate(tt.input, "navigator")
			require.Error(t, err)

			cliErr, ok := err.(*CLIError)
			require.True(t, ok, "expected CLIError, got %T", err)
			assert.Equal(t, tt.wantCode, cliErr.Code)
		})
	}
}

func TestRunPersonaCreate_NoTemplateListsTemplates(t *testing.T) {
	// When template is empty, should list templates (no error)
	err := runPersonaCreate("some-name", "")
	assert.NoError(t, err)
}

func TestNewPersonaCmd_HasSubcommands(t *testing.T) {
	cmd := NewPersonaCmd()
	assert.Equal(t, "persona", cmd.Use)

	var subNames []string
	for _, sub := range cmd.Commands() {
		subNames = append(subNames, sub.Name())
	}
	assert.Contains(t, subNames, "create")
	assert.Contains(t, subNames, "list")
}

func TestPersonaListCmd_OutputsNames(t *testing.T) {
	cmd := NewPersonaCmd()
	var listCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "list" {
			listCmd = sub
			break
		}
	}
	require.NotNil(t, listCmd)

	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = listCmd.RunE(listCmd, nil)
	w.Close()
	os.Stdout = old
	require.NoError(t, err)

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should contain at least some known persona names
	assert.True(t, strings.Contains(output, "navigator") || strings.Contains(output, "craftsman"),
		"expected known persona names in output, got: %s", output)
}

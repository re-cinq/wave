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

func TestRunPipelineCreate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runPipelineCreate("my-custom", "ops-hello-world")
	require.NoError(t, err)

	outputPath := filepath.Join(".wave", "pipelines", "my-custom.yaml")
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Verify it's valid YAML
	var doc map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &doc))

	// Verify metadata.name was updated
	metadata, ok := doc["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata should be a map")
	assert.Equal(t, "my-custom", metadata["name"])

	// Verify the rest of the pipeline content is preserved (e.g., steps exist)
	steps, ok := doc["steps"]
	assert.True(t, ok, "steps should exist in the scaffolded pipeline")
	assert.NotNil(t, steps)
}

func TestRunPipelineCreate_TemplateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err = runPipelineCreate("my-pipeline", "nonexistent-template")
	require.Error(t, err)

	cliErr, ok := err.(*CLIError)
	require.True(t, ok, "expected CLIError, got %T", err)
	assert.Equal(t, CodePipelineNotFound, cliErr.Code)
	assert.Contains(t, cliErr.Message, "nonexistent-template")
}

func TestRunPipelineCreate_NameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create the first time
	err = runPipelineCreate("collision-test", "ops-hello-world")
	require.NoError(t, err)

	// Try to create again — should fail
	err = runPipelineCreate("collision-test", "ops-hello-world")
	require.Error(t, err)

	cliErr, ok := err.(*CLIError)
	require.True(t, ok, "expected CLIError, got %T", err)
	assert.Equal(t, CodeInvalidArgs, cliErr.Code)
	assert.Contains(t, cliErr.Message, "already exists")
}

func TestRunPipelineCreate_MissingName(t *testing.T) {
	err := runPipelineCreate("", "ops-hello-world")
	require.Error(t, err)

	cliErr, ok := err.(*CLIError)
	require.True(t, ok, "expected CLIError, got %T", err)
	assert.Equal(t, CodeInvalidArgs, cliErr.Code)
	assert.Contains(t, cliErr.Message, "--name is required")
}

func TestRunPipelineCreate_InvalidName(t *testing.T) {
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
			err := runPipelineCreate(tt.input, "ops-hello-world")
			require.Error(t, err)

			cliErr, ok := err.(*CLIError)
			require.True(t, ok, "expected CLIError, got %T", err)
			assert.Equal(t, tt.wantCode, cliErr.Code)
		})
	}
}

func TestRunPipelineCreate_NoTemplateListsTemplates(t *testing.T) {
	// When template is empty, runPipelineCreate should list templates (no error)
	err := runPipelineCreate("some-name", "")
	assert.NoError(t, err)
}

func TestCategoryPrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"impl prefix", "impl-issue", "impl"},
		{"plan prefix", "plan-scope", "plan"},
		{"ops prefix", "ops-hello-world", "ops"},
		{"audit prefix", "audit-security", "audit"},
		{"doc prefix", "doc-fix", "doc"},
		{"test prefix", "test-gen", "test"},
		{"wave prefix", "wave-evolve", "wave"},
		{"bench prefix", "bench-solve", "bench"},
		{"unknown prefix", "custom-pipeline", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, categoryPrefix(tt.input))
		})
	}
}

func TestUpdateMetadataName(t *testing.T) {
	input := `kind: WavePipeline
metadata:
  name: old-name
  description: "test"
steps: []`

	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(input), &doc))

	updateMetadataName(&doc, "new-name")

	out, err := yaml.Marshal(&doc)
	require.NoError(t, err)
	assert.Contains(t, string(out), "name: new-name")
	assert.NotContains(t, string(out), "old-name")
}

func TestNewPipelineCmd_HasSubcommands(t *testing.T) {
	cmd := NewPipelineCmd()
	assert.Equal(t, "pipeline", cmd.Use)

	var subNames []string
	for _, sub := range cmd.Commands() {
		subNames = append(subNames, sub.Name())
	}
	assert.Contains(t, subNames, "create")
	assert.Contains(t, subNames, "list")
}

func TestPipelineListCmd_OutputsNames(t *testing.T) {
	cmd := NewPipelineCmd()
	// Find the list subcommand
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

	// Should contain at least some known pipeline names
	assert.True(t, strings.Contains(output, "ops-hello-world") || strings.Contains(output, "impl-issue"),
		"expected known pipeline names in output, got: %s", output)
}

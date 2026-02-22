package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInputArtifact_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{"name": "test", "value": 42}`), 0644)
	require.NoError(t, err)

	// Create schema
	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["name", "value"],
		"properties": {
			"name": {"type": "string"},
			"value": {"type": "integer"}
		}
	}`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Validate
	err = ValidateInputArtifact("test-artifact", "schema.json", tmpDir)
	assert.NoError(t, err)
}

func TestValidateInputArtifact_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact with invalid JSON
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{"name": "test", "value": "not a number"}`), 0644)
	require.NoError(t, err)

	// Create schema expecting integer
	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["name", "value"],
		"properties": {
			"name": {"type": "string"},
			"value": {"type": "integer"}
		}
	}`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Validate
	err = ValidateInputArtifact("test-artifact", "schema.json", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed schema validation")
}

func TestValidateInputArtifact_NoSchemaSkipsValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact (no schema will be provided)
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{"anything": "works"}`), 0644)
	require.NoError(t, err)

	// Validate with empty schema path - should skip
	err = ValidateInputArtifact("test-artifact", "", tmpDir)
	assert.NoError(t, err)
}

func TestValidateInputArtifact_MissingArtifact(t *testing.T) {
	tmpDir := t.TempDir()

	// Create schema but no artifact
	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{"type": "object"}`
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Validate
	err = ValidateInputArtifact("nonexistent-artifact", "schema.json", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read input artifact")
}

func TestValidateInputArtifact_MissingSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact but no schema
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{}`), 0644)
	require.NoError(t, err)

	// Validate
	err = ValidateInputArtifact("test-artifact", "nonexistent-schema.json", tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read schema file")
}

func TestValidateInputArtifacts_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifacts directory
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	// Create two artifacts
	err := os.WriteFile(filepath.Join(artifactsDir, "artifact1"), []byte(`{"id": 1}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(artifactsDir, "artifact2"), []byte(`{"id": 2}`), 0644)
	require.NoError(t, err)

	// Create schema
	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["id"],
		"properties": {
			"id": {"type": "integer"}
		}
	}`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Validate both
	configs := []InputArtifactConfig{
		{Name: "artifact1", SchemaPath: "schema.json"},
		{Name: "artifact2", SchemaPath: "schema.json"},
	}

	results, err := ValidateInputArtifacts(configs, tmpDir)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Passed)
	assert.True(t, results[1].Passed)
}

func TestValidateInputArtifacts_FailsOnFirstError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifacts directory
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	// Create one valid, one invalid artifact
	err := os.WriteFile(filepath.Join(artifactsDir, "valid"), []byte(`{"id": 1}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(artifactsDir, "invalid"), []byte(`{"id": "not a number"}`), 0644)
	require.NoError(t, err)

	// Create schema
	schemaPath := filepath.Join(tmpDir, "schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["id"],
		"properties": {
			"id": {"type": "integer"}
		}
	}`
	err = os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Validate - invalid first
	configs := []InputArtifactConfig{
		{Name: "invalid", SchemaPath: "schema.json"},
		{Name: "valid", SchemaPath: "schema.json"},
	}

	results, err := ValidateInputArtifacts(configs, tmpDir)
	assert.Error(t, err)
	// Should have stopped after first failure
	assert.Len(t, results, 1)
	assert.False(t, results[0].Passed)
}

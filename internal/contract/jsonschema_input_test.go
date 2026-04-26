package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"required": ["name", "value"],
	"properties": {
		"name": {"type": "string"},
		"value": {"type": "integer"}
	}
}`

func TestValidateInputArtifactContent_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{"name": "test", "value": 42}`), 0644)
	require.NoError(t, err)

	err = ValidateInputArtifactContent("test-artifact", testSchema, artifactPath)
	assert.NoError(t, err)
}

func TestValidateInputArtifactContent_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact with invalid JSON
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{"name": "test", "value": "not a number"}`), 0644)
	require.NoError(t, err)

	err = ValidateInputArtifactContent("test-artifact", testSchema, artifactPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed schema validation")
}

func TestValidateInputArtifactContent_NoSchemaSkipsValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact (no schema will be provided)
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{"anything": "works"}`), 0644)
	require.NoError(t, err)

	err = ValidateInputArtifactContent("test-artifact", "", artifactPath)
	assert.NoError(t, err)
}

func TestValidateInputArtifactContent_MissingArtifact(t *testing.T) {
	err := ValidateInputArtifactContent("nonexistent-artifact", testSchema, "/nonexistent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read input artifact")
}

func TestValidateInputArtifactContent_InvalidSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact but no schema
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	artifactPath := filepath.Join(artifactsDir, "test-artifact")
	err := os.WriteFile(artifactPath, []byte(`{}`), 0644)
	require.NoError(t, err)

	err = ValidateInputArtifactContent("test-artifact", "not valid json {{{", artifactPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse schema")
}

func TestValidateInputArtifacts_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifacts directory
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	err := os.WriteFile(filepath.Join(artifactsDir, "artifact1"), []byte(`{"id": 1}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(artifactsDir, "artifact2"), []byte(`{"id": 2}`), 0644)
	require.NoError(t, err)

	idSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["id"],
		"properties": { "id": {"type": "integer"} }
	}`

	configs := []InputArtifactConfig{
		{Name: "artifact1", SchemaContent: idSchema},
		{Name: "artifact2", SchemaContent: idSchema},
	}

	results, err := ValidateInputArtifacts(configs, tmpDir)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Passed)
	assert.True(t, results[1].Passed)
}

func TestValidateInputArtifactContent_SharedFindingsSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifact matching shared-findings schema
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	validFindings := `{
		"findings": [
			{
				"type": "dead-code",
				"severity": "high",
				"file": "internal/foo/bar.go",
				"description": "Unused function"
			}
		],
		"summary": "Found 1 dead code item",
		"scan_type": "dead-code",
		"scanned_at": "2026-01-15T10:30:00Z"
	}`
	artifactPath := filepath.Join(artifactsDir, "findings")
	err := os.WriteFile(artifactPath, []byte(validFindings), 0644)
	require.NoError(t, err)

	// Write an inline fixture schema that covers the fields used above (not the real shared-findings schema)
	schemaDir := filepath.Join(tmpDir, ".agents", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["findings"],
		"properties": {
			"findings": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["type", "severity"],
					"properties": {
						"type": { "type": "string" },
						"severity": { "type": "string", "enum": ["critical", "high", "medium", "low", "info"] },
						"file": { "type": "string" },
						"description": { "type": "string" }
					}
				}
			},
			"summary": { "type": "string" },
			"scan_type": { "type": "string" },
			"scanned_at": { "type": "string", "format": "date-time" }
		},
		"additionalProperties": false
	}`

	_ = schemaDir
	// Valid findings should pass
	err = ValidateInputArtifactContent("findings", schemaContent, artifactPath)
	assert.NoError(t, err)

	// Invalid severity should fail
	invalidFindings := `{
		"findings": [
			{
				"type": "dead-code",
				"severity": "CRITICAL",
				"file": "internal/foo/bar.go"
			}
		]
	}`
	err = os.WriteFile(artifactPath, []byte(invalidFindings), 0644)
	require.NoError(t, err)

	err = ValidateInputArtifactContent("findings", schemaContent, artifactPath)
	assert.Error(t, err, "uppercase severity should fail validation against canonical enum")
}

func TestValidateInputArtifactContent_SharedReviewVerdictSchema(t *testing.T) {
	tmpDir := t.TempDir()

	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	validVerdict := `{
		"verdict": "APPROVE",
		"summary": "Code looks good, all tests pass",
		"findings": [],
		"pr_url": "https://github.com/org/repo/pull/42",
		"reviewed_at": "2026-01-15T10:30:00Z"
	}`
	err := os.WriteFile(filepath.Join(artifactsDir, "verdict"), []byte(validVerdict), 0644)
	require.NoError(t, err)

	schemaDir := filepath.Join(tmpDir, ".agents", "contracts")
	require.NoError(t, os.MkdirAll(schemaDir, 0755))

	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["verdict"],
		"properties": {
			"verdict": { "type": "string", "enum": ["APPROVE", "REQUEST_CHANGES", "COMMENT", "REJECT"] },
			"summary": { "type": "string" },
			"findings": { "type": "array" },
			"pr_url": { "type": "string" },
			"reviewed_at": { "type": "string" }
		},
		"additionalProperties": false
	}`

	_ = schemaDir
	artifactPath := filepath.Join(artifactsDir, "verdict")

	err = ValidateInputArtifactContent("verdict", schemaContent, artifactPath)
	assert.NoError(t, err)

	// Invalid verdict value should fail
	invalidVerdict := `{"verdict": "LGTM"}`
	err = os.WriteFile(artifactPath, []byte(invalidVerdict), 0644)
	require.NoError(t, err)

	err = ValidateInputArtifactContent("verdict", schemaContent, artifactPath)
	assert.Error(t, err, "invalid verdict enum value should fail validation")
}

func TestValidateInputArtifacts_FailsOnFirstError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create artifacts directory
	artifactsDir := filepath.Join(tmpDir, ".agents", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))

	err := os.WriteFile(filepath.Join(artifactsDir, "valid"), []byte(`{"id": 1}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(artifactsDir, "invalid"), []byte(`{"id": "not a number"}`), 0644)
	require.NoError(t, err)

	idSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["id"],
		"properties": { "id": {"type": "integer"} }
	}`

	configs := []InputArtifactConfig{
		{Name: "invalid", SchemaContent: idSchema},
		{Name: "valid", SchemaContent: idSchema},
	}

	results, err := ValidateInputArtifacts(configs, tmpDir)
	assert.Error(t, err)
	assert.Len(t, results, 1)
	assert.False(t, results[0].Passed)
}

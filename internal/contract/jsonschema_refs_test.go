package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDefsDir creates a _defs/ directory with shared schema files for testing.
func setupDefsDir(t *testing.T, contractsDir string) {
	t.Helper()
	defsDir := filepath.Join(contractsDir, "_defs")
	require.NoError(t, os.MkdirAll(defsDir, 0755))

	severitySchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"definitions": {
			"findings_severity": {
				"type": "string",
				"enum": ["critical", "high", "medium", "low", "info"]
			}
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(defsDir, "severity.schema.json"), []byte(severitySchema), 0644))
}

func TestPreloadSharedDefs_RefResolvesCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	contractsDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(contractsDir, 0755))
	setupDefsDir(t, contractsDir)

	// Create a schema that uses $ref to _defs/
	mainSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["severity"],
		"properties": {
			"severity": {
				"$ref": "_defs/severity.schema.json#/definitions/findings_severity"
			}
		}
	}`
	schemaPath := filepath.Join(contractsDir, "test.schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(mainSchema), 0644))

	// Create artifact
	waveDir := filepath.Join(tmpDir, ".wave")
	require.NoError(t, os.WriteFile(filepath.Join(waveDir, "artifact.json"), []byte(`{"severity": "high"}`), 0644))

	// Validate using the full validator path
	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:       "json_schema",
		SchemaPath: schemaPath,
	}
	err := v.Validate(cfg, tmpDir)
	assert.NoError(t, err)
}

func TestPreloadSharedDefs_InvalidDataFailsAgainstRefSchema(t *testing.T) {
	tmpDir := t.TempDir()
	contractsDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(contractsDir, 0755))
	setupDefsDir(t, contractsDir)

	mainSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["severity"],
		"properties": {
			"severity": {
				"$ref": "_defs/severity.schema.json#/definitions/findings_severity"
			}
		}
	}`
	schemaPath := filepath.Join(contractsDir, "test.schema.json")
	require.NoError(t, os.WriteFile(schemaPath, []byte(mainSchema), 0644))

	// Create artifact with invalid severity value
	waveDir := filepath.Join(tmpDir, ".wave")
	require.NoError(t, os.WriteFile(filepath.Join(waveDir, "artifact.json"), []byte(`{"severity": "INVALID"}`), 0644))

	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:       "json_schema",
		SchemaPath: schemaPath,
	}
	err := v.Validate(cfg, tmpDir)
	assert.Error(t, err)
}

func TestPreloadSharedDefs_MissingDefsDirectorySkipsSilently(t *testing.T) {
	// No _defs directory exists — preloadSharedDefs should return nil
	compiler := jsonschema.NewCompiler()
	err := preloadSharedDefs(compiler, "/nonexistent/schema.json", "/nonexistent")
	assert.NoError(t, err)
}

func TestPreloadSharedDefs_MalformedDefsFileProducesError(t *testing.T) {
	tmpDir := t.TempDir()
	defsDir := filepath.Join(tmpDir, "_defs")
	require.NoError(t, os.MkdirAll(defsDir, 0755))

	// Write malformed JSON
	require.NoError(t, os.WriteFile(filepath.Join(defsDir, "bad.schema.json"), []byte(`{not valid json`), 0644))

	compiler := jsonschema.NewCompiler()
	err := preloadSharedDefs(compiler, filepath.Join(tmpDir, "schema.json"), tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing shared def")
}

func TestPreloadSharedDefs_InputValidator_RefResolves(t *testing.T) {
	tmpDir := t.TempDir()
	contractsDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(contractsDir, 0755))
	setupDefsDir(t, contractsDir)

	// Create a schema that uses $ref
	mainSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["severity"],
		"properties": {
			"severity": {
				"$ref": "_defs/severity.schema.json#/definitions/findings_severity"
			}
		}
	}`
	relSchemaPath := filepath.Join(".wave", "contracts", "test.schema.json")
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, relSchemaPath), []byte(mainSchema), 0644))

	// Create artifact
	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(artifactsDir, "test-artifact"), []byte(`{"severity": "medium"}`), 0644))

	err := ValidateInputArtifact("test-artifact", relSchemaPath, tmpDir)
	assert.NoError(t, err)
}

func TestPreloadSharedDefs_InputValidator_RefValidationFails(t *testing.T) {
	tmpDir := t.TempDir()
	contractsDir := filepath.Join(tmpDir, ".wave", "contracts")
	require.NoError(t, os.MkdirAll(contractsDir, 0755))
	setupDefsDir(t, contractsDir)

	mainSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["severity"],
		"properties": {
			"severity": {
				"$ref": "_defs/severity.schema.json#/definitions/findings_severity"
			}
		}
	}`
	relSchemaPath := filepath.Join(".wave", "contracts", "test.schema.json")
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, relSchemaPath), []byte(mainSchema), 0644))

	artifactsDir := filepath.Join(tmpDir, ".wave", "artifacts")
	require.NoError(t, os.MkdirAll(artifactsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(artifactsDir, "test-artifact"), []byte(`{"severity": "BOGUS"}`), 0644))

	err := ValidateInputArtifact("test-artifact", relSchemaPath, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed schema validation")
}

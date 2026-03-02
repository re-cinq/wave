package proposal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/recinq/wave/internal/pipeline/proposal"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectRoot returns the project root by walking up from the test file location.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine test file location")
	// From internal/pipeline/proposal/ go up 3 levels
	return filepath.Join(filepath.Dir(filename), "..", "..", "..")
}

func TestIntegrationRealCatalog(t *testing.T) {
	root := projectRoot(t)
	pipelinesDir := filepath.Join(root, ".wave", "pipelines")

	// Verify the pipelines directory exists
	_, err := os.Stat(pipelinesDir)
	if os.IsNotExist(err) {
		t.Skip("pipelines directory not found — skipping integration test")
	}

	catalog, err := proposal.NewCatalog(pipelinesDir)
	require.NoError(t, err)
	require.Greater(t, catalog.Len(), 0, "expected at least one pipeline in catalog")

	// Create a synthetic health artifact with diverse signals
	health := proposal.HealthArtifact{
		Version:   "1.0",
		Timestamp: time.Now().UTC(),
		Signals: []proposal.HealthSignal{
			{Category: "test_failures", Severity: "high", Count: 3, Score: 0.7, Detail: "3 failing tests in auth package"},
			{Category: "security", Severity: "critical", Count: 1, Score: 1.0, Detail: "SQL injection vulnerability"},
			{Category: "dead_code", Severity: "low", Count: 8, Score: 0.2, Detail: "8 unused functions"},
			{Category: "doc_issues", Severity: "medium", Count: 5, Score: 0.5, Detail: "5 missing doc comments"},
		},
		ForgeType: proposal.ForgeGitHub,
		Summary:   "Integration test health artifact",
	}

	engine := proposal.NewEngine(catalog)
	prop, err := engine.Propose(health, proposal.ForgeGitHub)
	require.NoError(t, err)
	require.NotNil(t, prop)

	// Validate the proposal struct
	require.NoError(t, prop.Validate())

	// Should have proposals since we provided signals matching known pipelines
	assert.NotEmpty(t, prop.Proposals, "expected at least one pipeline proposal")
	assert.Equal(t, proposal.ForgeGitHub, prop.ForgeType)

	// Every item should have non-empty rationale and valid priority
	for _, item := range prop.Proposals {
		assert.NotEmpty(t, item.Pipeline)
		assert.NotEmpty(t, item.Rationale)
		assert.Greater(t, item.Priority, 0)
		assert.GreaterOrEqual(t, item.Score, 0.0)
		assert.LessOrEqual(t, item.Score, 1.0)
	}

	// Validate JSON output against the contract schema
	proposalJSON, err := json.Marshal(prop)
	require.NoError(t, err)

	schemaPath := filepath.Join(root, ".wave", "contracts", "pipeline-proposal.schema.json")
	validateJSONAgainstSchema(t, proposalJSON, schemaPath)
}

func TestIntegrationSchemaValidation(t *testing.T) {
	root := projectRoot(t)
	schemaPath := filepath.Join(root, ".wave", "contracts", "pipeline-proposal.schema.json")

	// Verify schema file exists
	_, err := os.Stat(schemaPath)
	if os.IsNotExist(err) {
		t.Skip("schema file not found — skipping schema validation test")
	}

	t.Run("valid proposal passes schema", func(t *testing.T) {
		prop := &proposal.Proposal{
			ForgeType: proposal.ForgeGitHub,
			Timestamp: time.Now().UTC(),
			Proposals: []proposal.ProposalItem{
				{
					Pipeline:      "test-gen",
					Rationale:     "test coverage is low",
					Priority:      1,
					Score:         0.9,
					ParallelGroup: 0,
					DependsOn:     nil,
					Category:      "testing",
				},
				{
					Pipeline:      "security-scan",
					Rationale:     "security vulnerability detected",
					Priority:      2,
					Score:         0.8,
					ParallelGroup: 0,
					DependsOn:     nil,
				},
			},
			HealthSummary: "2 issues found",
		}

		data, err := json.Marshal(prop)
		require.NoError(t, err)
		validateJSONAgainstSchema(t, data, schemaPath)
	})

	t.Run("empty proposals passes schema", func(t *testing.T) {
		prop := &proposal.Proposal{
			ForgeType: proposal.ForgeUnknown,
			Timestamp: time.Now().UTC(),
			Proposals: []proposal.ProposalItem{},
		}

		data, err := json.Marshal(prop)
		require.NoError(t, err)
		validateJSONAgainstSchema(t, data, schemaPath)
	})
}

func validateJSONAgainstSchema(t *testing.T, data []byte, schemaPath string) {
	t.Helper()

	schemaData, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "could not read schema file")

	// Parse schema
	var schemaDoc interface{}
	require.NoError(t, json.Unmarshal(schemaData, &schemaDoc))

	compiler := jsonschema.NewCompiler()
	require.NoError(t, compiler.AddResource("schema.json", schemaDoc))

	schema, err := compiler.Compile("schema.json")
	require.NoError(t, err, "could not compile JSON schema")

	// Parse instance
	var instance interface{}
	require.NoError(t, json.Unmarshal(data, &instance))

	// Validate
	err = schema.Validate(instance)
	assert.NoError(t, err, "JSON validation against schema failed: %v", err)
}

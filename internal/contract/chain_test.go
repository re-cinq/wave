package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateArtifactAgainstSchema validates a JSON artifact against a schema file,
// with _defs preloaded for $ref resolution.
func validateArtifactAgainstSchema(t *testing.T, artifact []byte, schemaPath string, contractsDir string) error {
	t.Helper()

	schemaData, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "reading schema file")

	var schemaDoc interface{}
	require.NoError(t, json.Unmarshal(schemaData, &schemaDoc), "parsing schema")

	compiler := jsonschema.NewCompiler()
	schemaURI := schemaPath
	require.NoError(t, compiler.AddResource(schemaURI, schemaDoc), "adding schema resource")

	// Preload shared defs for $ref resolution
	err = preloadSharedDefs(compiler, schemaURI, filepath.Dir(schemaPath))
	require.NoError(t, err, "preloading shared defs")

	schema, err := compiler.Compile(schemaURI)
	require.NoError(t, err, "compiling schema")

	var data interface{}
	require.NoError(t, json.Unmarshal(artifact, &data), "parsing artifact")

	return schema.Validate(data)
}

// TestChain_AssessmentToPlan validates that a typical assessment artifact
// produced by fetch-assess is compatible with what the plan step expects.
func TestChain_AssessmentToPlan(t *testing.T) {
	contractsDir := findContractsDir(t)
	schemaPath := filepath.Join(contractsDir, "issue-assessment.schema.json")

	// Sample assessment artifact matching the output schema
	artifact := []byte(`{
		"implementable": true,
		"issue": {
			"number": 42,
			"title": "Add feature X",
			"body": "We need feature X for performance",
			"repository": "owner/repo",
			"url": "https://github.com/owner/repo/issues/42",
			"labels": ["enhancement"],
			"state": "OPEN",
			"author": "dev1",
			"comments": []
		},
		"assessment": {
			"quality_score": 85,
			"complexity": "medium",
			"skip_steps": ["clarify"],
			"branch_name": "42-add-feature-x",
			"missing_info": [],
			"summary": "Well-specified feature request"
		}
	}`)

	err := validateArtifactAgainstSchema(t, artifact, schemaPath, contractsDir)
	assert.NoError(t, err, "assessment artifact should validate against assessment schema")
}

// TestChain_PlanToImplement validates that a typical impl-plan artifact
// produced by the plan step is compatible with what the implement step expects.
func TestChain_PlanToImplement(t *testing.T) {
	contractsDir := findContractsDir(t)
	schemaPath := filepath.Join(contractsDir, "issue-impl-plan.schema.json")

	artifact := []byte(`{
		"issue_number": 42,
		"branch_name": "42-add-feature-x",
		"feature_dir": "specs/42-add-feature-x",
		"spec_file": "specs/42-add-feature-x/spec.md",
		"plan_file": "specs/42-add-feature-x/plan.md",
		"tasks_file": "specs/42-add-feature-x/tasks.md",
		"tasks": [
			{
				"id": "1.1",
				"title": "Create handler",
				"description": "Create the HTTP handler for feature X",
				"file_changes": [
					{"path": "internal/handler/feature_x.go", "action": "create"}
				]
			}
		],
		"summary": "Implementation plan for feature X"
	}`)

	err := validateArtifactAgainstSchema(t, artifact, schemaPath, contractsDir)
	assert.NoError(t, err, "impl-plan artifact should validate against impl-plan schema")
}

// TestChain_DiffToReview validates that a typical diff-analysis artifact
// is compatible with the schema used by the review steps.
func TestChain_DiffToReview(t *testing.T) {
	contractsDir := findContractsDir(t)
	schemaPath := filepath.Join(contractsDir, "diff-analysis.schema.json")

	artifact := []byte(`{
		"pr_metadata": {
			"number": 123,
			"url": "https://github.com/owner/repo/pull/123",
			"head_branch": "feature-branch",
			"base_branch": "main"
		},
		"files_changed": [
			{
				"path": "internal/handler/feature.go",
				"change_type": "modified",
				"purpose": "Add new endpoint"
			}
		],
		"modules_affected": ["internal/handler"],
		"related_tests": ["internal/handler/feature_test.go"],
		"breaking_changes": []
	}`)

	err := validateArtifactAgainstSchema(t, artifact, schemaPath, contractsDir)
	assert.NoError(t, err, "diff-analysis artifact should validate against diff-analysis schema")
}

// TestChain_ReviewToTriage validates that a typical review-findings artifact
// is compatible with the triage-verdict schema for the review→triage chain.
func TestChain_ReviewToTriage(t *testing.T) {
	contractsDir := findContractsDir(t)

	// First, validate review-findings output
	reviewSchema := filepath.Join(contractsDir, "review-findings.schema.json")
	reviewArtifact := []byte(`{
		"pr_number": 123,
		"pr_url": "https://github.com/owner/repo/pull/123",
		"head_branch": "feature-branch",
		"findings": [
			{
				"severity": "high",
				"summary": "Missing nil check",
				"file": "handler.go",
				"line": 42,
				"detail": "os.ReadFile result not checked",
				"action": "fix"
			}
		],
		"fix_plan_summary": "Fix nil check in handler.go",
		"verdict": "changes_requested",
		"critical_count": 0,
		"major_count": 1
	}`)

	err := validateArtifactAgainstSchema(t, reviewArtifact, reviewSchema, contractsDir)
	assert.NoError(t, err, "review-findings artifact should validate against review-findings schema")

	// Then validate that a triage verdict can consume the PR metadata
	triageSchema := filepath.Join(contractsDir, "triage-verdict.schema.json")
	triageArtifact := []byte(`{
		"pr_number": 123,
		"pr_url": "https://github.com/owner/repo/pull/123",
		"head_branch": "feature-branch",
		"fixes": [
			{
				"severity": "major",
				"summary": "Missing nil check",
				"file": "handler.go",
				"line": 42,
				"action_detail": "Add nil check after os.ReadFile call"
			}
		],
		"rejected": [],
		"deferred": [],
		"summary": "1 fix accepted: nil check in handler.go"
	}`)

	err = validateArtifactAgainstSchema(t, triageArtifact, triageSchema, contractsDir)
	assert.NoError(t, err, "triage-verdict artifact should validate against triage-verdict schema")
}

// TestChain_SharedFindingsWithRef validates that the shared-findings schema
// works correctly with $ref to _defs/ after refactoring.
func TestChain_SharedFindingsWithRef(t *testing.T) {
	contractsDir := findContractsDir(t)
	schemaPath := filepath.Join(contractsDir, "shared-findings.schema.json")

	artifact := []byte(`{
		"findings": [
			{
				"type": "dead-code",
				"severity": "medium",
				"package": "internal/bench",
				"file": "bench.go",
				"line": 42,
				"item": "BenchmarkOld",
				"description": "Unused benchmark function",
				"evidence": "No callers found",
				"recommendation": "remove"
			},
			{
				"type": "security",
				"severity": "critical",
				"file": "handler.go",
				"description": "SQL injection vulnerability"
			}
		],
		"summary": "2 findings: 1 critical, 1 medium",
		"scan_type": "dead-code"
	}`)

	err := validateArtifactAgainstSchema(t, artifact, schemaPath, contractsDir)
	assert.NoError(t, err, "shared-findings artifact should validate with $ref to _defs/")
}

// TestChain_AggregatedFindingsWithRef validates the aggregated-findings schema
// with $ref severity.
func TestChain_AggregatedFindingsWithRef(t *testing.T) {
	contractsDir := findContractsDir(t)
	schemaPath := filepath.Join(contractsDir, "aggregated-findings.schema.json")

	artifact := []byte(`{
		"findings": [
			{
				"type": "dead-code",
				"severity": "medium",
				"source_audit": "audit-dead-code",
				"file": "bench.go",
				"description": "Unused function"
			}
		],
		"source_audits": [
			{"name": "audit-dead-code", "finding_count": 1, "status": "completed"}
		],
		"total_findings": 1,
		"summary": "1 finding from 1 audit"
	}`)

	err := validateArtifactAgainstSchema(t, artifact, schemaPath, contractsDir)
	assert.NoError(t, err, "aggregated-findings artifact should validate with $ref severity")
}

// TestChain_IssueReferenceWithRef validates schemas that use $ref for issue_reference.
func TestChain_IssueReferenceWithRef(t *testing.T) {
	contractsDir := findContractsDir(t)

	// comment-result with $ref issue_reference
	commentSchema := filepath.Join(contractsDir, "comment-result.schema.json")
	commentArtifact := []byte(`{
		"success": true,
		"issue_reference": {
			"issue_number": 42,
			"repository": "owner/repo",
			"issue_url": "https://github.com/owner/repo/issues/42"
		},
		"timestamp": "2026-04-14T00:00:00Z"
	}`)

	err := validateArtifactAgainstSchema(t, commentArtifact, commentSchema, contractsDir)
	assert.NoError(t, err, "comment-result should validate with $ref issue_reference")

	// research-findings with $ref issue_reference
	researchSchema := filepath.Join(contractsDir, "research-findings.schema.json")
	researchArtifact := []byte(`{
		"issue_reference": {
			"issue_number": 42,
			"repository": "owner/repo"
		},
		"findings_by_topic": [
			{
				"topic_id": "TOPIC-0001",
				"topic_title": "Performance",
				"findings": [
					{
						"id": "FINDING-0001",
						"summary": "This is a research finding about performance optimization techniques for Go applications",
						"source": {
							"url": "https://example.com/article",
							"title": "Go Performance Guide",
							"type": "blog_post"
						},
						"relevance_score": 0.9
					}
				],
				"confidence_level": "high"
			}
		],
		"research_metadata": {
			"started_at": "2026-04-14T00:00:00Z",
			"completed_at": "2026-04-14T01:00:00Z"
		}
	}`)

	err = validateArtifactAgainstSchema(t, researchArtifact, researchSchema, contractsDir)
	assert.NoError(t, err, "research-findings should validate with $ref issue_reference")
}

// TestChain_ReviewVerdictWithRef validates the shared-review-verdict schema
// with $ref review_severity.
func TestChain_ReviewVerdictWithRef(t *testing.T) {
	contractsDir := findContractsDir(t)
	schemaPath := filepath.Join(contractsDir, "shared-review-verdict.schema.json")

	artifact := []byte(`{
		"verdict": "REQUEST_CHANGES",
		"summary": "Found 1 critical security issue",
		"findings": [
			{
				"severity": "critical",
				"file": "handler.go",
				"line": 42,
				"description": "SQL injection vulnerability",
				"suggestion": "Use parameterized queries"
			}
		]
	}`)

	err := validateArtifactAgainstSchema(t, artifact, schemaPath, contractsDir)
	assert.NoError(t, err, "review verdict should validate with $ref review_severity")
}

// findContractsDir locates the .wave/contracts directory relative to the project root.
func findContractsDir(t *testing.T) string {
	t.Helper()

	// Walk up from the test file to find the project root
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		contractsPath := filepath.Join(dir, ".wave", "contracts")
		if _, err := os.Stat(contractsPath); err == nil {
			return contractsPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find .wave/contracts directory")
		}
		dir = parent
	}
}

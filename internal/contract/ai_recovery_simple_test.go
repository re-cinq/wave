package contract

import (
	"io/ioutil"
	"testing"
)

// TestJSONRecovery_RealWorldAIContent tests the JSON recovery with actual problematic AI content
func TestJSONRecovery_RealWorldAIContent(t *testing.T) {
	// Test with the actual content from the failing github-issue-enhancer pipeline
	realAIContent := `Based on the analysis I've performed on the re-cinq/wave repository and the existing artifact.json content, I can provide the clean JSON output that matches the required schema. Let me extract just the JSON portion from what I read:

{
  "repository": {
    "owner": "re-cinq",
    "name": "wave"
  },
  "total_issues": 10,
  "analyzed_count": 10,
  "poor_quality_issues": [
    {
      "number": 20,
      "title": "add scan poorly commented gh issues and extend and connect",
      "body": "",
      "quality_score": 25,
      "problems": [
        "Title uses lowercase and lacks proper capitalization"
      ],
      "recommendations": [
        "Rewrite title as 'Add GitHub Issue Scanner for Code Quality Analysis'"
      ],
      "labels": [],
      "url": "https://github.com/re-cinq/wave/issues/20"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T15:30:00Z"
}`

	parser := NewJSONRecoveryParser(ProgressiveRecovery)
	result, err := parser.ParseWithRecovery(realAIContent)

	if err != nil {
		t.Errorf("Expected successful recovery, got error: %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected valid result, got invalid. Applied fixes: %v", result.AppliedFixes)
	}

	if result.ParsedData == nil {
		t.Error("Expected parsed data, got nil")
	}

	// Check that some AI-related fix was applied
	if len(result.AppliedFixes) == 0 {
		t.Error("Expected some fixes to be applied for AI content")
	}

	t.Logf("Successfully recovered AI content with fixes: %v", result.AppliedFixes)
}

// TestJSONSchemaValidator_WithRealAIContent tests end-to-end schema validation with real AI content
func TestJSONSchemaValidator_WithRealAIContent(t *testing.T) {
	// Use the actual GitHub issue analysis schema
	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["repository", "total_issues", "poor_quality_issues"],
		"properties": {
			"repository": {
				"type": "object",
				"required": ["owner", "name"],
				"properties": {
					"owner": {"type": "string"},
					"name": {"type": "string"}
				}
			},
			"total_issues": {"type": "integer", "minimum": 0},
			"poor_quality_issues": {"type": "array"}
		}
	}`

	// Use real AI content with explanation text
	aiContent := `The current file has invalid JSON due to explanatory text. Since I need Write permission to fix the artifact.json file, let me provide the corrected JSON output directly:

{
  "repository": {
    "owner": "re-cinq",
    "name": "wave"
  },
  "total_issues": 5,
  "poor_quality_issues": [
    {
      "number": 20,
      "title": "test issue"
    }
  ]
}`

	config := ContractConfig{
		Type:                  "json_schema",
		Schema:                schema,
		AllowRecovery:         true,
		RecoveryLevel:         "progressive",
		ProgressiveValidation: false,
		MustPass:              true,
	}

	// Create temporary workspace
	workspacePath := t.TempDir()
	artifactPath := workspacePath + "/artifact.json"

	err := ioutil.WriteFile(artifactPath, []byte(aiContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test artifact: %v", err)
	}

	// Run validation
	validator := &jsonSchemaValidator{}
	err = validator.Validate(config, workspacePath)

	if err != nil {
		t.Errorf("Expected validation to succeed with AI content recovery, got error: %v", err)
	}
}
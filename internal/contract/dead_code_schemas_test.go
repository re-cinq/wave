package contract

import (
	"strings"
	"testing"
)

// Inline schema constants for dead code pipeline contracts

const deadCodeScanSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Dead Code Scan",
  "description": "Scan results for dead or redundant code",
  "type": "object",
  "required": ["target", "findings", "summary", "timestamp"],
  "properties": {
    "target": { "type": "string", "minLength": 1 },
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "type", "location", "description", "confidence", "safe_to_remove"],
        "properties": {
          "id": { "type": "string", "pattern": "^DC-[0-9]{3}$" },
          "type": {
            "type": "string",
            "enum": ["unused_export", "unreachable", "orphaned_file", "redundant", "stale_test", "unused_import", "commented_code", "duplicate", "stale_glue", "hardcoded_value"]
          },
          "location": { "type": "string", "minLength": 1 },
          "symbol": { "type": "string" },
          "description": { "type": "string", "minLength": 5 },
          "evidence": { "type": "string" },
          "confidence": { "type": "string", "enum": ["high", "medium", "low"] },
          "safe_to_remove": { "type": "boolean" },
          "removal_note": { "type": "string" },
          "line_range": {
            "type": "object",
            "properties": {
              "start": { "type": "integer", "minimum": 1 },
              "end": { "type": "integer", "minimum": 1 }
            },
            "required": ["start", "end"]
          },
          "suggested_action": {
            "type": "string",
            "enum": ["remove", "consolidate", "investigate", "configure"]
          }
        }
      }
    },
    "summary": {
      "type": "object",
      "required": ["total_findings"],
      "properties": {
        "total_findings": { "type": "integer", "minimum": 0 },
        "by_type": { "type": "object" },
        "high_confidence_count": { "type": "integer", "minimum": 0 },
        "estimated_lines_removable": { "type": "integer", "minimum": 0 }
      }
    },
    "timestamp": { "type": "string", "format": "date-time" }
  }
}`

const deadCodeReportSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Dead Code Report",
  "description": "Formatted markdown report of dead code scan findings",
  "type": "object",
  "required": ["title", "body", "findings_count", "categories_found"],
  "properties": {
    "title": { "type": "string", "minLength": 1 },
    "body": { "type": "string", "minLength": 1 },
    "findings_count": { "type": "integer", "minimum": 0 },
    "categories_found": {
      "type": "array",
      "items": { "type": "string" },
      "minItems": 0
    }
  }
}`

const deadCodePRResultSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Dead Code PR Result",
  "description": "Result of posting dead code findings as a PR comment",
  "type": "object",
  "required": ["comment_url", "pr_number", "findings_summary"],
  "properties": {
    "comment_url": { "type": "string", "format": "uri" },
    "pr_number": { "type": "integer", "minimum": 1 },
    "findings_summary": {
      "type": "object",
      "required": ["total", "by_category"],
      "properties": {
        "total": { "type": "integer", "minimum": 0 },
        "by_category": { "type": "object" },
        "high_confidence": { "type": "integer", "minimum": 0 }
      }
    }
  }
}`

const deadCodeIssueResultSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Dead Code Issue Result",
  "description": "Result of creating a GitHub issue with dead code findings",
  "type": "object",
  "required": ["issue_url", "issue_number", "findings_summary"],
  "properties": {
    "issue_url": { "type": "string", "format": "uri" },
    "issue_number": { "type": "integer", "minimum": 1 },
    "findings_summary": {
      "type": "object",
      "required": ["total", "by_category"],
      "properties": {
        "total": { "type": "integer", "minimum": 0 },
        "by_category": { "type": "object" },
        "high_confidence": { "type": "integer", "minimum": 0 }
      }
    }
  }
}`

// Helper to build a minimal valid scan finding JSON fragment with overrides.
func minimalFinding(overrides string) string {
	// Base finding with all required fields; overrides are merged at the end.
	base := `{
		"id": "DC-001",
		"type": "unused_export",
		"location": "internal/foo.go",
		"description": "Exported symbol is never imported",
		"confidence": "high",
		"safe_to_remove": true`
	if overrides != "" {
		base += "," + overrides
	}
	base += "}"
	return base
}

// wrapScan wraps a finding JSON string into a full scan payload.
func wrapScan(findingJSON string) []byte {
	return []byte(`{
		"target": "internal/",
		"findings": [` + findingJSON + `],
		"summary": { "total_findings": 1 },
		"timestamp": "2026-02-23T10:00:00Z"
	}`)
}

// ---------------------------------------------------------------------------
// 1. TestDeadCodeScanSchema_NewCategories
// ---------------------------------------------------------------------------

func TestDeadCodeScanSchema_NewCategories(t *testing.T) {
	tests := []struct {
		name     string
		category string
	}{
		{"duplicate category", "duplicate"},
		{"stale_glue category", "stale_glue"},
		{"hardcoded_value category", "hardcoded_value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finding := `{
				"id": "DC-010",
				"type": "` + tt.category + `",
				"location": "internal/example.go",
				"description": "Finding with new category type",
				"confidence": "medium",
				"safe_to_remove": false
			}`
			artifact := wrapScan(finding)

			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:                    "json_schema",
				Schema:                  deadCodeScanSchema,
				DisableWrapperDetection: true,
				AllowRecovery:           false,
			}

			ws := t.TempDir()
			writeTestArtifact(t, ws, artifact)

			if err := v.Validate(cfg, ws); err != nil {
				t.Errorf("expected valid scan with category %q, got error: %v", tt.category, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. TestDeadCodeScanSchema_NewFields
// ---------------------------------------------------------------------------

func TestDeadCodeScanSchema_NewFields(t *testing.T) {
	tests := []struct {
		name     string
		artifact []byte
	}{
		{
			name: "finding with line_range",
			artifact: wrapScan(minimalFinding(`
				"line_range": { "start": 10, "end": 25 }
			`)),
		},
		{
			name: "suggested_action remove",
			artifact: wrapScan(minimalFinding(`
				"suggested_action": "remove"
			`)),
		},
		{
			name: "suggested_action consolidate",
			artifact: wrapScan(minimalFinding(`
				"suggested_action": "consolidate"
			`)),
		},
		{
			name: "suggested_action investigate",
			artifact: wrapScan(minimalFinding(`
				"suggested_action": "investigate"
			`)),
		},
		{
			name: "suggested_action configure",
			artifact: wrapScan(minimalFinding(`
				"suggested_action": "configure"
			`)),
		},
		{
			name: "all new optional fields together",
			artifact: wrapScan(`{
				"id": "DC-042",
				"type": "duplicate",
				"location": "cmd/wave/main.go",
				"symbol": "initConfig",
				"description": "Duplicated configuration initialiser across two commands",
				"evidence": "Same body in cmd/wave/run.go:initConfig",
				"confidence": "high",
				"safe_to_remove": true,
				"removal_note": "Consolidate into shared helper",
				"line_range": { "start": 5, "end": 30 },
				"suggested_action": "consolidate"
			}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:                    "json_schema",
				Schema:                  deadCodeScanSchema,
				DisableWrapperDetection: true,
				AllowRecovery:           false,
			}

			ws := t.TempDir()
			writeTestArtifact(t, ws, tt.artifact)

			if err := v.Validate(cfg, ws); err != nil {
				t.Errorf("expected valid scan payload, got error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. TestDeadCodeScanSchema_BackwardCompatibility
// ---------------------------------------------------------------------------

func TestDeadCodeScanSchema_BackwardCompatibility(t *testing.T) {
	// Old-style payload: no line_range, no suggested_action, only original categories.
	oldPayload := []byte(`{
		"target": "internal/pipeline",
		"findings": [
			{
				"id": "DC-001",
				"type": "unused_export",
				"location": "internal/pipeline/runner.go",
				"symbol": "OldHelper",
				"description": "Exported function that is never imported elsewhere",
				"evidence": "grep -r shows zero imports",
				"confidence": "high",
				"safe_to_remove": true,
				"removal_note": "Safe to delete"
			},
			{
				"id": "DC-002",
				"type": "orphaned_file",
				"location": "internal/pipeline/legacy.go",
				"description": "File has no references from any other package",
				"confidence": "medium",
				"safe_to_remove": false
			}
		],
		"summary": {
			"total_findings": 2,
			"by_type": { "unused_export": 1, "orphaned_file": 1 },
			"high_confidence_count": 1,
			"estimated_lines_removable": 45
		},
		"timestamp": "2026-02-23T08:30:00Z"
	}`)

	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:                    "json_schema",
		Schema:                  deadCodeScanSchema,
		DisableWrapperDetection: true,
		AllowRecovery:           false,
	}

	ws := t.TempDir()
	writeTestArtifact(t, ws, oldPayload)

	if err := v.Validate(cfg, ws); err != nil {
		t.Errorf("old-style payload without new fields should still validate, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 4. TestDeadCodeReportSchema_Valid
// ---------------------------------------------------------------------------

func TestDeadCodeReportSchema_Valid(t *testing.T) {
	payload := []byte(`{
		"title": "Dead Code Report - Wave",
		"body": "## Summary\n\nFound 5 dead code items across 3 packages.",
		"findings_count": 5,
		"categories_found": ["unused_export", "duplicate", "stale_glue"]
	}`)

	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:                    "json_schema",
		Schema:                  deadCodeReportSchema,
		DisableWrapperDetection: true,
		AllowRecovery:           false,
	}

	ws := t.TempDir()
	writeTestArtifact(t, ws, payload)

	if err := v.Validate(cfg, ws); err != nil {
		t.Errorf("expected valid report payload, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5. TestDeadCodeReportSchema_Invalid
// ---------------------------------------------------------------------------

func TestDeadCodeReportSchema_Invalid(t *testing.T) {
	tests := []struct {
		name          string
		artifact      string
		errorContains string
	}{
		{
			name:          "missing title",
			artifact:      `{"body": "report body", "findings_count": 1, "categories_found": ["unused_export"]}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing body",
			artifact:      `{"title": "Report", "findings_count": 1, "categories_found": ["unused_export"]}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing findings_count",
			artifact:      `{"title": "Report", "body": "Some body text", "categories_found": ["unused_export"]}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing categories_found",
			artifact:      `{"title": "Report", "body": "Some body text", "findings_count": 2}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "empty title",
			artifact:      `{"title": "", "body": "Some body", "findings_count": 0, "categories_found": []}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "negative findings_count",
			artifact:      `{"title": "Report", "body": "Body", "findings_count": -1, "categories_found": []}`,
			errorContains: "contract validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:                    "json_schema",
				Schema:                  deadCodeReportSchema,
				DisableWrapperDetection: true,
				AllowRecovery:           false,
			}

			ws := t.TempDir()
			writeTestArtifact(t, ws, []byte(tt.artifact))

			err := v.Validate(cfg, ws)
			if err == nil {
				t.Error("expected validation error but got none")
				return
			}
			if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 6. TestDeadCodePRResultSchema_Valid
// ---------------------------------------------------------------------------

func TestDeadCodePRResultSchema_Valid(t *testing.T) {
	payload := []byte(`{
		"comment_url": "https://github.com/re-cinq/wave/pull/135#issuecomment-123456",
		"pr_number": 135,
		"findings_summary": {
			"total": 7,
			"by_category": { "unused_export": 3, "duplicate": 2, "stale_glue": 2 },
			"high_confidence": 4
		}
	}`)

	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:                    "json_schema",
		Schema:                  deadCodePRResultSchema,
		DisableWrapperDetection: true,
		AllowRecovery:           false,
	}

	ws := t.TempDir()
	writeTestArtifact(t, ws, payload)

	if err := v.Validate(cfg, ws); err != nil {
		t.Errorf("expected valid PR result payload, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7. TestDeadCodeIssueResultSchema_Valid
// ---------------------------------------------------------------------------

func TestDeadCodeIssueResultSchema_Valid(t *testing.T) {
	payload := []byte(`{
		"issue_url": "https://github.com/re-cinq/wave/issues/200",
		"issue_number": 200,
		"findings_summary": {
			"total": 3,
			"by_category": { "hardcoded_value": 1, "commented_code": 2 },
			"high_confidence": 1
		}
	}`)

	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:                    "json_schema",
		Schema:                  deadCodeIssueResultSchema,
		DisableWrapperDetection: true,
		AllowRecovery:           false,
	}

	ws := t.TempDir()
	writeTestArtifact(t, ws, payload)

	if err := v.Validate(cfg, ws); err != nil {
		t.Errorf("expected valid issue result payload, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 8. TestDeadCodePRResultSchema_Invalid
// ---------------------------------------------------------------------------

func TestDeadCodePRResultSchema_Invalid(t *testing.T) {
	tests := []struct {
		name          string
		artifact      string
		errorContains string
	}{
		{
			name:          "missing comment_url",
			artifact:      `{"pr_number": 1, "findings_summary": {"total": 0, "by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing pr_number",
			artifact:      `{"comment_url": "https://github.com/re-cinq/wave/pull/1#issuecomment-1", "findings_summary": {"total": 0, "by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing findings_summary",
			artifact:      `{"comment_url": "https://github.com/re-cinq/wave/pull/1#issuecomment-1", "pr_number": 1}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "pr_number zero",
			artifact:      `{"comment_url": "https://github.com/re-cinq/wave/pull/1#issuecomment-1", "pr_number": 0, "findings_summary": {"total": 0, "by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "findings_summary missing total",
			artifact:      `{"comment_url": "https://github.com/re-cinq/wave/pull/1#issuecomment-1", "pr_number": 1, "findings_summary": {"by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "findings_summary missing by_category",
			artifact:      `{"comment_url": "https://github.com/re-cinq/wave/pull/1#issuecomment-1", "pr_number": 1, "findings_summary": {"total": 0}}`,
			errorContains: "contract validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:                    "json_schema",
				Schema:                  deadCodePRResultSchema,
				DisableWrapperDetection: true,
				AllowRecovery:           false,
			}

			ws := t.TempDir()
			writeTestArtifact(t, ws, []byte(tt.artifact))

			err := v.Validate(cfg, ws)
			if err == nil {
				t.Error("expected validation error but got none")
				return
			}
			if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 9. TestDeadCodeIssueResultSchema_Invalid
// ---------------------------------------------------------------------------

func TestDeadCodeIssueResultSchema_Invalid(t *testing.T) {
	tests := []struct {
		name          string
		artifact      string
		errorContains string
	}{
		{
			name:          "missing issue_url",
			artifact:      `{"issue_number": 1, "findings_summary": {"total": 0, "by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing issue_number",
			artifact:      `{"issue_url": "https://github.com/re-cinq/wave/issues/1", "findings_summary": {"total": 0, "by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing findings_summary",
			artifact:      `{"issue_url": "https://github.com/re-cinq/wave/issues/1", "issue_number": 1}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "issue_number zero",
			artifact:      `{"issue_url": "https://github.com/re-cinq/wave/issues/1", "issue_number": 0, "findings_summary": {"total": 0, "by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "findings_summary missing total",
			artifact:      `{"issue_url": "https://github.com/re-cinq/wave/issues/1", "issue_number": 1, "findings_summary": {"by_category": {}}}`,
			errorContains: "contract validation failed",
		},
		{
			name:          "findings_summary missing by_category",
			artifact:      `{"issue_url": "https://github.com/re-cinq/wave/issues/1", "issue_number": 1, "findings_summary": {"total": 0}}`,
			errorContains: "contract validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:                    "json_schema",
				Schema:                  deadCodeIssueResultSchema,
				DisableWrapperDetection: true,
				AllowRecovery:           false,
			}

			ws := t.TempDir()
			writeTestArtifact(t, ws, []byte(tt.artifact))

			err := v.Validate(cfg, ws)
			if err == nil {
				t.Error("expected validation error but got none")
				return
			}
			if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
			}
		})
	}
}

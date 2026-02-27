package tests_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// projectRoot finds the project root by walking up from CWD to find go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.mod found in parent directories)")
		}
		dir = parent
	}
}

// TestAuditFindingsSchema_Valid verifies that the audit-findings.schema.json
// accepts valid audit output data for each audit type.
func TestAuditFindingsSchema_Valid(t *testing.T) {
	root := projectRoot(t)
	schemaPath := filepath.Join(root, ".wave", "contracts", "audit-findings.schema.json")
	schema := loadAndCompileSchema(t, schemaPath)

	tests := []struct {
		name string
		data string
	}{
		{
			name: "quality audit with findings",
			data: `{
				"target": "internal/pipeline",
				"audit_type": "quality",
				"findings": [
					{
						"id": "AQ-001",
						"title": "High cyclomatic complexity in executor",
						"severity": "HIGH",
						"category": "complexity",
						"location": "internal/pipeline/executor.go:42",
						"description": "Function Execute has a cyclomatic complexity of 25, exceeding the recommended threshold of 15"
					}
				],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 0, "HIGH": 1, "MEDIUM": 0, "LOW": 0},
					"by_category": {"complexity": 1},
					"risk_assessment": "Moderate risk: one high-complexity function identified"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "security audit with findings",
			data: `{
				"target": "internal/adapter",
				"audit_type": "security",
				"findings": [
					{
						"id": "AS-001",
						"title": "Potential command injection in subprocess execution",
						"severity": "CRITICAL",
						"category": "owasp",
						"location": "internal/adapter/claude.go:128",
						"description": "User-controlled input passed to exec.Command without sanitization",
						"evidence": "exec.Command(adapter.Binary, args...)",
						"recommendation": "Validate and sanitize all command arguments before execution"
					}
				],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 1, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
					"by_category": {"owasp": 1},
					"risk_assessment": "High risk: critical injection vulnerability found in adapter execution path"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "deps audit with findings",
			data: `{
				"target": "go.mod",
				"audit_type": "deps",
				"findings": [
					{
						"id": "AD-001",
						"title": "Outdated dependency: cobra v1.8.0",
						"severity": "LOW",
						"category": "outdated",
						"location": "go.mod",
						"description": "github.com/spf13/cobra is at v1.8.0, latest stable is v1.9.0",
						"details": {"current_version": "v1.8.0", "latest_version": "v1.9.0"}
					}
				],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 1},
					"by_category": {"outdated": 1},
					"risk_assessment": "Low risk: one minor version behind on a single dependency"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "flaws audit with findings",
			data: `{
				"target": "internal/",
				"audit_type": "flaws",
				"findings": [
					{
						"id": "AF-001",
						"title": "TODO comment without linked issue",
						"severity": "LOW",
						"category": "todo_fixme",
						"location": "internal/contract/jsonschema.go:276",
						"description": "TODO comment found without a linked issue reference for tracking"
					},
					{
						"id": "AF-002",
						"title": "Missing error handling in file read",
						"severity": "MEDIUM",
						"category": "error_handling",
						"location": "internal/workspace/setup.go:55",
						"description": "Error return from os.ReadFile is assigned to _ and silently ignored"
					}
				],
				"summary": {
					"total_findings": 2,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 1, "LOW": 1},
					"by_category": {"todo_fixme": 1, "error_handling": 1},
					"risk_assessment": "Low to moderate risk: minor error handling gaps and untracked TODOs"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "empty findings list",
			data: `{
				"target": "internal/",
				"audit_type": "quality",
				"findings": [],
				"summary": {
					"total_findings": 0,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
					"by_category": {},
					"risk_assessment": "No quality issues found — codebase is clean"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "finding with all optional fields",
			data: `{
				"target": "cmd/wave/",
				"audit_type": "security",
				"findings": [
					{
						"id": "AS-001",
						"title": "Hardcoded API token in test fixture",
						"severity": "MEDIUM",
						"category": "secrets",
						"location": "cmd/wave/commands/run_test.go:42",
						"description": "Test fixture contains what appears to be a hardcoded API token",
						"evidence": "const testToken = \"ghp_xxxx...\"",
						"recommendation": "Use environment variables or test fixtures for sensitive test data",
						"details": {"token_type": "github_pat", "is_valid": false}
					}
				],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 1, "LOW": 0},
					"by_category": {"secrets": 1},
					"risk_assessment": "Moderate risk: potential secret exposure in test fixtures"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data interface{}
			if err := json.Unmarshal([]byte(tt.data), &data); err != nil {
				t.Fatalf("failed to parse test data as JSON: %v", err)
			}
			if err := schema.Validate(data); err != nil {
				t.Errorf("expected valid data to pass schema validation, got error: %v", err)
			}
		})
	}
}

// TestAuditFindingsSchema_Invalid verifies that the audit-findings.schema.json
// rejects invalid audit output data.
func TestAuditFindingsSchema_Invalid(t *testing.T) {
	root := projectRoot(t)
	schemaPath := filepath.Join(root, ".wave", "contracts", "audit-findings.schema.json")
	schema := loadAndCompileSchema(t, schemaPath)

	tests := []struct {
		name string
		data string
	}{
		{
			name: "missing required field: target",
			data: `{
				"audit_type": "quality",
				"findings": [],
				"summary": {
					"total_findings": 0,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
					"by_category": {},
					"risk_assessment": "No findings detected"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "missing required field: audit_type",
			data: `{
				"target": "internal/",
				"findings": [],
				"summary": {
					"total_findings": 0,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
					"by_category": {},
					"risk_assessment": "No findings detected"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "invalid audit_type enum value",
			data: `{
				"target": "internal/",
				"audit_type": "performance",
				"findings": [],
				"summary": {
					"total_findings": 0,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
					"by_category": {},
					"risk_assessment": "No findings detected"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "invalid severity enum value",
			data: `{
				"target": "internal/",
				"audit_type": "quality",
				"findings": [{
					"id": "AQ-001",
					"title": "Test finding for invalid severity",
					"severity": "URGENT",
					"category": "lint",
					"location": "file.go:1",
					"description": "This is a test finding with invalid severity level"
				}],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 1},
					"by_category": {"lint": 1},
					"risk_assessment": "Low risk finding"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "invalid finding ID pattern",
			data: `{
				"target": "internal/",
				"audit_type": "quality",
				"findings": [{
					"id": "WRONG-001",
					"title": "Test finding with wrong ID prefix",
					"severity": "LOW",
					"category": "lint",
					"location": "file.go:1",
					"description": "This is a test finding with an invalid ID pattern"
				}],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 1},
					"by_category": {"lint": 1},
					"risk_assessment": "Low risk finding"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "missing required finding field: description",
			data: `{
				"target": "internal/",
				"audit_type": "quality",
				"findings": [{
					"id": "AQ-001",
					"title": "Finding without description",
					"severity": "LOW",
					"category": "lint",
					"location": "file.go:1"
				}],
				"summary": {
					"total_findings": 1,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 1},
					"by_category": {"lint": 1},
					"risk_assessment": "Low risk finding"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "missing summary by_severity",
			data: `{
				"target": "internal/",
				"audit_type": "quality",
				"findings": [],
				"summary": {
					"total_findings": 0,
					"by_category": {},
					"risk_assessment": "No findings detected"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
		{
			name: "negative total_findings",
			data: `{
				"target": "internal/",
				"audit_type": "quality",
				"findings": [],
				"summary": {
					"total_findings": -1,
					"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
					"by_category": {},
					"risk_assessment": "No findings detected"
				},
				"timestamp": "2026-02-27T18:00:00Z"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data interface{}
			if err := json.Unmarshal([]byte(tt.data), &data); err != nil {
				t.Fatalf("failed to parse test data as JSON: %v", err)
			}
			if err := schema.Validate(data); err == nil {
				t.Error("expected invalid data to fail schema validation, but it passed")
			}
		})
	}
}

// TestAuditFindingsSchema_IsValidJSON verifies the schema file itself is valid JSON
// and a valid JSON Schema.
func TestAuditFindingsSchema_IsValidJSON(t *testing.T) {
	root := projectRoot(t)
	schemaPath := filepath.Join(root, ".wave", "contracts", "audit-findings.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}

	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("schema file is not valid JSON: %v", err)
	}

	// Verify it compiles as a JSON Schema
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, parsed); err != nil {
		t.Fatalf("failed to add schema as resource: %v", err)
	}
	if _, err := compiler.Compile(schemaPath); err != nil {
		t.Fatalf("schema is not a valid JSON Schema: %v", err)
	}
}

// TestAuditFindingsSchema_ContractValidation verifies the schema works with
// Wave's contract validation engine.
func TestAuditFindingsSchema_ContractValidation(t *testing.T) {
	root := projectRoot(t)
	schemaPath := filepath.Join(root, ".wave", "contracts", "audit-findings.schema.json")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}

	validArtifact := `{
		"target": "internal/",
		"audit_type": "quality",
		"findings": [],
		"summary": {
			"total_findings": 0,
			"by_severity": {"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0},
			"by_category": {},
			"risk_assessment": "No quality issues found in the scanned codebase"
		},
		"timestamp": "2026-02-27T18:00:00Z"
	}`

	cfg := contract.ContractConfig{
		Type:   "json_schema",
		Schema: string(schemaData),
	}

	workspacePath := t.TempDir()
	waveDir := filepath.Join(workspacePath, ".wave")
	if err := os.MkdirAll(waveDir, 0755); err != nil {
		t.Fatalf("failed to create .wave directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(waveDir, "artifact.json"), []byte(validArtifact), 0644); err != nil {
		t.Fatalf("failed to write test artifact: %v", err)
	}

	validator := contract.NewValidator(cfg)
	if err := validator.Validate(cfg, workspacePath); err != nil {
		t.Errorf("expected valid artifact to pass contract validation, got: %v", err)
	}
}

// TestAuditPipelineYAML_Parseable verifies that each audit pipeline YAML file
// can be loaded and parsed by the pipeline loader.
func TestAuditPipelineYAML_Parseable(t *testing.T) {
	root := projectRoot(t)

	pipelines := []struct {
		name  string
		file  string
		steps int
	}{
		{"audit-quality", filepath.Join(root, ".wave", "pipelines", "audit-quality.yaml"), 3},
		{"audit-security", filepath.Join(root, ".wave", "pipelines", "audit-security.yaml"), 3},
		{"audit-deps", filepath.Join(root, ".wave", "pipelines", "audit-deps.yaml"), 3},
		{"audit-flaws", filepath.Join(root, ".wave", "pipelines", "audit-flaws.yaml"), 3},
	}

	loader := &pipeline.YAMLPipelineLoader{}

	for _, tt := range pipelines {
		t.Run(tt.name, func(t *testing.T) {
			p, err := loader.Load(tt.file)
			if err != nil {
				t.Fatalf("failed to load pipeline %s: %v", tt.file, err)
			}

			if p.Metadata.Name != tt.name {
				t.Errorf("pipeline name = %q, want %q", p.Metadata.Name, tt.name)
			}

			if p.Kind != "WavePipeline" {
				t.Errorf("pipeline kind = %q, want %q", p.Kind, "WavePipeline")
			}

			if len(p.Steps) != tt.steps {
				t.Errorf("pipeline step count = %d, want %d", len(p.Steps), tt.steps)
			}

			// Verify standard 3-step pattern: scan → verify → report
			if len(p.Steps) >= 3 {
				if p.Steps[0].ID != "scan" {
					t.Errorf("first step ID = %q, want %q", p.Steps[0].ID, "scan")
				}
				if p.Steps[1].ID != "verify" {
					t.Errorf("second step ID = %q, want %q", p.Steps[1].ID, "verify")
				}
				if p.Steps[2].ID != "report" {
					t.Errorf("third step ID = %q, want %q", p.Steps[2].ID, "report")
				}
			}

			// Verify personas follow expected pattern
			if p.Steps[0].Persona != "navigator" {
				t.Errorf("scan step persona = %q, want %q", p.Steps[0].Persona, "navigator")
			}
			if p.Steps[1].Persona != "auditor" {
				t.Errorf("verify step persona = %q, want %q", p.Steps[1].Persona, "auditor")
			}
			if p.Steps[2].Persona != "summarizer" {
				t.Errorf("report step persona = %q, want %q", p.Steps[2].Persona, "summarizer")
			}

			// Verify scan step has readonly mount
			if len(p.Steps[0].Workspace.Mount) == 0 {
				t.Error("scan step should have at least one mount")
			} else if p.Steps[0].Workspace.Mount[0].Mode != "readonly" {
				t.Errorf("scan step mount mode = %q, want %q", p.Steps[0].Workspace.Mount[0].Mode, "readonly")
			}

			// Verify all steps have contract validation pointing to audit-findings schema
			for _, step := range p.Steps {
				if step.Handover.Contract.SchemaPath != ".wave/contracts/audit-findings.schema.json" {
					t.Errorf("step %q contract schema_path = %q, want %q",
						step.ID, step.Handover.Contract.SchemaPath, ".wave/contracts/audit-findings.schema.json")
				}
			}

			// Verify dependencies form a DAG: verify depends on scan, report depends on verify
			if len(p.Steps[1].Dependencies) != 1 || p.Steps[1].Dependencies[0] != "scan" {
				t.Errorf("verify step dependencies = %v, want [scan]", p.Steps[1].Dependencies)
			}
			if len(p.Steps[2].Dependencies) != 1 || p.Steps[2].Dependencies[0] != "verify" {
				t.Errorf("report step dependencies = %v, want [verify]", p.Steps[2].Dependencies)
			}

			// Verify pipeline is marked as releasable
			if !p.Metadata.Release {
				t.Error("audit pipeline should be marked as release: true")
			}
		})
	}
}

// TestAuditPipelineYAML_DAGValid verifies that each audit pipeline has a valid DAG
// (no cycles, no missing dependencies).
func TestAuditPipelineYAML_DAGValid(t *testing.T) {
	root := projectRoot(t)

	pipelines := []string{
		filepath.Join(root, ".wave", "pipelines", "audit-quality.yaml"),
		filepath.Join(root, ".wave", "pipelines", "audit-security.yaml"),
		filepath.Join(root, ".wave", "pipelines", "audit-deps.yaml"),
		filepath.Join(root, ".wave", "pipelines", "audit-flaws.yaml"),
	}

	loader := &pipeline.YAMLPipelineLoader{}

	for _, file := range pipelines {
		t.Run(filepath.Base(file), func(t *testing.T) {
			p, err := loader.Load(file)
			if err != nil {
				t.Fatalf("failed to load pipeline: %v", err)
			}

			validator := &pipeline.DAGValidator{}
			if err := validator.ValidateDAG(p); err != nil {
				t.Errorf("DAG validation failed: %v", err)
			}
		})
	}
}

// loadAndCompileSchema loads a JSON Schema file and compiles it for validation.
func loadAndCompileSchema(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read schema file %s: %v", path, err)
	}

	var schemaDoc interface{}
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, schemaDoc); err != nil {
		t.Fatalf("failed to add schema resource: %v", err)
	}

	schema, err := compiler.Compile(path)
	if err != nil {
		t.Fatalf("failed to compile schema: %v", err)
	}

	return schema
}

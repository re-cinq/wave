package complexity

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestToSharedFindings_SeverityMapping(t *testing.T) {
	report := Report{
		ScannedAt: time.Now().UTC(),
		Scores: []FunctionScore{
			{File: "a.go", Function: "Pass", Cyclomatic: 5, Cognitive: 3},
			{File: "b.go", Function: "Warn", Cyclomatic: 12, Cognitive: 8},
			{File: "c.go", Function: "FailCyclo", Cyclomatic: 20, Cognitive: 5},
			{File: "d.go", Function: "FailCog", Cyclomatic: 5, Cognitive: 22},
			{File: "e.go", Function: "FailBoth", Cyclomatic: 30, Cognitive: 30},
		},
	}
	doc := ToSharedFindings(report, Options{})
	// Pass: no findings
	// Warn: 1 cyclomatic medium (cog 8 ≤ 10 warn)
	// FailCyclo: 1 high
	// FailCog: 1 high
	// FailBoth: 2 high
	if got, want := len(doc.Findings), 5; got != want {
		t.Fatalf("findings count = %d, want %d (%+v)", got, want, doc.Findings)
	}
	if !doc.HasBreach() {
		t.Fatalf("HasBreach = false, want true")
	}
	highs, mediums := 0, 0
	for _, f := range doc.Findings {
		if f.Type != "complexity" {
			t.Fatalf("finding type = %q, want complexity", f.Type)
		}
		switch f.Severity {
		case "high":
			highs++
		case "medium":
			mediums++
		default:
			t.Fatalf("unexpected severity %q", f.Severity)
		}
	}
	if highs != 4 || mediums != 1 {
		t.Fatalf("high=%d medium=%d, want 4/1", highs, mediums)
	}
}

func TestToSharedFindings_AllPass(t *testing.T) {
	report := Report{
		Scores: []FunctionScore{
			{File: "a.go", Function: "Easy", Cyclomatic: 2, Cognitive: 0},
		},
	}
	doc := ToSharedFindings(report, Options{})
	if len(doc.Findings) != 0 {
		t.Fatalf("expected no findings, got %+v", doc.Findings)
	}
	if doc.HasBreach() {
		t.Fatalf("HasBreach = true, want false")
	}
}

func TestToSharedFindings_ValidatesAgainstSchema(t *testing.T) {
	report := Report{
		ScannedAt: time.Now().UTC(),
		Scores: []FunctionScore{
			{File: "a.go", Package: "x", Function: "Big", Line: 10, Cyclomatic: 30, Cognitive: 30},
		},
	}
	doc := ToSharedFindings(report, Options{})
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Locate the shared-findings schema relative to the repo root.
	schemaPath := findSchemaFile(t)
	schemaSrc, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schemaDoc any
	if err := json.Unmarshal(schemaSrc, &schemaDoc); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("shared-findings.schema.json", schemaDoc); err != nil {
		t.Fatalf("add schema: %v", err)
	}
	schema, err := compiler.Compile("shared-findings.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	var artifact any
	if err := json.Unmarshal(body, &artifact); err != nil {
		t.Fatalf("parse findings: %v", err)
	}
	if err := schema.Validate(artifact); err != nil {
		t.Fatalf("findings JSON failed schema validation: %v", err)
	}
}

func findSchemaFile(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../../.agents/contracts/shared-findings.schema.json",
		"../../internal/defaults/contracts/shared-findings.schema.json",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Fatalf("shared-findings schema not found in %v", candidates)
	return ""
}

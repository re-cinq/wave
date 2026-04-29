package shared

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileWorkItemRefSchema loads the embedded work_item_ref schema and compiles
// it with the same validator the contract package uses at runtime so test
// behavior matches production validation.
func compileWorkItemRefSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	raw, ok := Lookup("work_item_ref")
	if !ok {
		t.Fatal("Lookup(\"work_item_ref\") = !ok")
	}

	var schemaDoc any
	if err := json.Unmarshal(raw, &schemaDoc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	const schemaURL = "wave://shared/work_item_ref"
	c := jsonschema.NewCompiler()
	if err := c.AddResource(schemaURL, schemaDoc); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	compiled, err := c.Compile(schemaURL)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return compiled
}

func mustParse(t *testing.T, raw string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("invalid fixture JSON: %v\n%s", err, raw)
	}
	return v
}

// TestWorkItemRef_PositiveFixtures validates one well-formed document for each
// value of the source enum: forge entries supply forge_host/owner/repo, while
// schedule and manual entries omit them.
func TestWorkItemRef_PositiveFixtures(t *testing.T) {
	schema := compileWorkItemRefSchema(t)

	cases := []struct {
		name string
		json string
	}{
		{
			name: "github_issue",
			json: `{
				"source": "github",
				"forge_host": "github.com",
				"owner": "re-cinq",
				"repo": "wave",
				"number": 1590,
				"url": "https://github.com/re-cinq/wave/issues/1590",
				"title": "Phase 2.1: work_item_ref shared schema + registry entry",
				"labels": ["enhancement", "ready-for-impl"],
				"state": "open",
				"created_at": "2026-04-29T10:15:00Z"
			}`,
		},
		{
			name: "gitea_pr_merged",
			json: `{
				"source": "gitea",
				"forge_host": "codeberg.org",
				"owner": "libretech",
				"repo": "wave-testing",
				"number": 42,
				"url": "https://codeberg.org/libretech/wave-testing/pulls/42",
				"title": "Add caching layer",
				"labels": [],
				"state": "merged",
				"created_at": "2026-03-10T08:00:00Z"
			}`,
		},
		{
			name: "gitlab_issue_closed",
			json: `{
				"source": "gitlab",
				"forge_host": "gitlab.example.com",
				"owner": "team",
				"repo": "service",
				"number": 7,
				"url": "https://gitlab.example.com/team/service/-/issues/7",
				"title": "Fix flaky test",
				"state": "closed",
				"created_at": "2026-02-01T12:30:45+02:00"
			}`,
		},
		{
			name: "bitbucket_no_number",
			json: `{
				"source": "bitbucket",
				"forge_host": "bitbucket.org",
				"owner": "acme",
				"repo": "platform",
				"url": "https://bitbucket.org/acme/platform/board/task-xyz",
				"title": "Cleanup dashboards",
				"labels": ["chore"],
				"state": "open",
				"created_at": "2026-04-15T18:00:00Z"
			}`,
		},
		{
			name: "schedule_trigger",
			json: `{
				"source": "schedule",
				"url": "wave://schedule/nightly-audit",
				"title": "Nightly audit run",
				"state": "open",
				"created_at": "2026-04-30T00:00:00Z"
			}`,
		},
		{
			name: "manual_trigger",
			json: `{
				"source": "manual",
				"url": "wave://manual/operator-2026-04-30-01",
				"title": "Operator: rerun failed contract suite",
				"labels": ["ops"],
				"state": "open",
				"created_at": "2026-04-30T01:48:11Z"
			}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParse(t, tc.json)
			if err := schema.Validate(doc); err != nil {
				t.Errorf("expected valid document, got error: %v", err)
			}
		})
	}
}

// TestWorkItemRef_NegativeFixtures asserts the schema rejects malformed
// documents. Each case isolates one violation: missing forge fields on a
// forge source, an unknown source, additional properties, an invalid state,
// and a malformed created_at timestamp.
func TestWorkItemRef_NegativeFixtures(t *testing.T) {
	schema := compileWorkItemRefSchema(t)

	cases := []struct {
		name        string
		json        string
		wantSubstr  string // optional fragment we expect somewhere in the error
		description string
	}{
		{
			name: "forge_missing_forge_host",
			json: `{
				"source": "github",
				"owner": "re-cinq",
				"repo": "wave",
				"url": "https://github.com/re-cinq/wave/issues/1",
				"title": "x",
				"state": "open",
				"created_at": "2026-04-30T00:00:00Z"
			}`,
			wantSubstr:  "forge_host",
			description: "github source must require forge_host via if/then",
		},
		{
			name: "unknown_source",
			json: `{
				"source": "carrier-pigeon",
				"url": "wave://manual/x",
				"title": "x",
				"state": "open",
				"created_at": "2026-04-30T00:00:00Z"
			}`,
			wantSubstr:  "source",
			description: "source enum must reject unknown values",
		},
		{
			name: "extra_property",
			json: `{
				"source": "manual",
				"url": "wave://manual/x",
				"title": "x",
				"state": "open",
				"created_at": "2026-04-30T00:00:00Z",
				"priority": "p0"
			}`,
			wantSubstr:  "priority",
			description: "additionalProperties:false must reject undeclared fields",
		},
		{
			name: "invalid_state",
			json: `{
				"source": "manual",
				"url": "wave://manual/x",
				"title": "x",
				"state": "wip",
				"created_at": "2026-04-30T00:00:00Z"
			}`,
			wantSubstr:  "state",
			description: "state enum must reject values outside open|closed|merged",
		},
		{
			name: "malformed_created_at",
			json: `{
				"source": "manual",
				"url": "wave://manual/x",
				"title": "x",
				"state": "open",
				"created_at": "yesterday"
			}`,
			wantSubstr:  "date-time",
			description: "created_at format:date-time must reject non-RFC3339 strings",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParse(t, tc.json)
			err := schema.Validate(doc)
			if err == nil {
				t.Fatalf("expected validation error (%s), got nil", tc.description)
			}
			if tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q does not mention %q (%s)", err.Error(), tc.wantSubstr, tc.description)
			}
		})
	}
}

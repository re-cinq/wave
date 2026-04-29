package defaults

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestOnboardProjectPipelineRegistered asserts the onboard-project pipeline
// is embedded and discoverable via the defaults registry, including all the
// prompts and the contract it depends on.
func TestOnboardProjectPipelineRegistered(t *testing.T) {
	pipelines, err := GetPipelines()
	if err != nil {
		t.Fatalf("GetPipelines() error: %v", err)
	}
	body, ok := pipelines["onboard-project.yaml"]
	if !ok {
		t.Fatal("expected onboard-project.yaml in default pipelines")
	}

	for _, want := range []string{
		"name: onboard-project",
		"persona: onboarder",
		"source_path: .agents/prompts/onboard/detect.md",
		"source_path: .agents/prompts/onboard/propose.md",
		"source_path: .agents/prompts/onboard/generate.md",
		"source_path: .agents/prompts/onboard/finalize.md",
		"schema_path: .agents/contracts/detection.schema.json",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("onboard-project.yaml missing required reference %q", want)
		}
	}

	prompts, err := GetPrompts()
	if err != nil {
		t.Fatalf("GetPrompts() error: %v", err)
	}
	for _, p := range []string{
		"onboard/detect.md",
		"onboard/propose.md",
		"onboard/generate.md",
		"onboard/finalize.md",
	} {
		if _, ok := prompts[p]; !ok {
			t.Errorf("expected prompt %q to be embedded", p)
		}
	}

	contracts, err := GetContracts()
	if err != nil {
		t.Fatalf("GetContracts() error: %v", err)
	}
	if _, ok := contracts["detection.schema.json"]; !ok {
		t.Error("expected detection.schema.json in default contracts")
	}
}

// TestOnboarderPersonaRegistered asserts both the markdown and yaml halves
// of the onboarder persona are embedded.
func TestOnboarderPersonaRegistered(t *testing.T) {
	personas, err := GetPersonas()
	if err != nil {
		t.Fatalf("GetPersonas() error: %v", err)
	}
	if _, ok := personas["onboarder.md"]; !ok {
		t.Fatal("expected onboarder.md in default personas")
	}

	configs, err := GetPersonaConfigs()
	if err != nil {
		t.Fatalf("GetPersonaConfigs() error: %v", err)
	}
	cfg, ok := configs["onboarder"]
	if !ok {
		t.Fatal("expected onboarder persona config")
	}
	if cfg.Description == "" {
		t.Error("onboarder persona config missing description")
	}
	if len(cfg.Permissions.AllowedTools) == 0 {
		t.Error("onboarder persona config missing allowed_tools")
	}
}

// TestDetectionSchemaValidatesFixtures runs the schema against valid + invalid
// fixtures to catch regressions in either the schema or the contract loader.
func TestDetectionSchemaValidatesFixtures(t *testing.T) {
	contracts, err := GetContracts()
	if err != nil {
		t.Fatalf("GetContracts() error: %v", err)
	}
	schemaText, ok := contracts["detection.schema.json"]
	if !ok {
		t.Fatal("detection.schema.json not embedded")
	}

	var schemaDoc interface{}
	if err := json.Unmarshal([]byte(schemaText), &schemaDoc); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("detection.schema.json", schemaDoc); err != nil {
		t.Fatalf("compiler.AddResource: %v", err)
	}
	schema, err := compiler.Compile("detection.schema.json")
	if err != nil {
		t.Fatalf("compiler.Compile: %v", err)
	}

	valid := []string{
		`{"flavour":"go","build_system":"go","test_command":"go test ./...","signals":["go.mod"],"package_managers":["go"]}`,
		`{"flavour":"node","signals":["package.json","package-lock.json"],"frameworks":["react"]}`,
		`{"flavour":"unknown","signals":[],"additional_signals":{"hint":"empty repo"}}`,
		`{"flavour":"rust","build_system":null,"test_command":null,"signals":["Cargo.toml"]}`,
	}
	for i, raw := range valid {
		var doc interface{}
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			t.Fatalf("valid fixture %d: unmarshal: %v", i, err)
		}
		if err := schema.Validate(doc); err != nil {
			t.Errorf("valid fixture %d failed validation: %v\nfixture: %s", i, err, raw)
		}
	}

	invalid := []string{
		// missing required flavour
		`{"signals":["go.mod"]}`,
		// missing required signals
		`{"flavour":"go"}`,
		// flavour wrong type
		`{"flavour":1,"signals":[]}`,
		// extra top-level property forbidden
		`{"flavour":"go","signals":[],"unexpected":"nope"}`,
	}
	for i, raw := range invalid {
		var doc interface{}
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			t.Fatalf("invalid fixture %d: unmarshal: %v", i, err)
		}
		if err := schema.Validate(doc); err == nil {
			t.Errorf("invalid fixture %d unexpectedly passed validation: %s", i, raw)
		}
	}
}

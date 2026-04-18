package pipeline

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestArtifactDef_YAMLParsing(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantName   string
		wantPath   string
		wantType   string
		wantSource string
		wantErr    bool
	}{
		{
			name:       "basic artifact with path",
			yaml:       "name: report\npath: .agents/output/report.json\ntype: json\n",
			wantName:   "report",
			wantPath:   ".agents/output/report.json",
			wantType:   "json",
			wantSource: "",
		},
		{
			name:       "stdout artifact without path",
			yaml:       "name: analysis\nsource: stdout\ntype: json\n",
			wantName:   "analysis",
			wantPath:   "",
			wantType:   "json",
			wantSource: "stdout",
		},
		{
			name:       "file source explicit",
			yaml:       "name: output\npath: result.md\nsource: file\ntype: markdown\n",
			wantName:   "output",
			wantPath:   "result.md",
			wantType:   "markdown",
			wantSource: "file",
		},
		{
			name:       "absent source defaults to empty (file)",
			yaml:       "name: data\npath: data.txt\ntype: text\n",
			wantName:   "data",
			wantPath:   "data.txt",
			wantType:   "text",
			wantSource: "",
		},
		{
			name:       "binary type supported",
			yaml:       "name: archive\npath: archive.tar.gz\ntype: binary\n",
			wantName:   "archive",
			wantPath:   "archive.tar.gz",
			wantType:   "binary",
			wantSource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var art ArtifactDef
			err := yaml.Unmarshal([]byte(tt.yaml), &art)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unmarshal error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if art.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", art.Name, tt.wantName)
			}
			if art.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", art.Path, tt.wantPath)
			}
			if art.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", art.Type, tt.wantType)
			}
			if art.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", art.Source, tt.wantSource)
			}
		})
	}
}

func TestArtifactRef_YAMLParsing(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		wantStep       string
		wantArtifact   string
		wantAs         string
		wantType       string
		wantSchemaPath string
		wantOptional   bool
		wantErr        bool
	}{
		{
			name:         "basic artifact ref",
			yaml:         "step: analyze\nartifact: report\nas: analysis\n",
			wantStep:     "analyze",
			wantArtifact: "report",
			wantAs:       "analysis",
		},
		{
			name:         "with type validation",
			yaml:         "step: produce\nartifact: data\nas: input\ntype: json\n",
			wantStep:     "produce",
			wantArtifact: "data",
			wantAs:       "input",
			wantType:     "json",
		},
		{
			name:           "with schema path",
			yaml:           "step: generate\nartifact: config\nas: cfg\nschema_path: ./schemas/config.json\n",
			wantStep:       "generate",
			wantArtifact:   "config",
			wantAs:         "cfg",
			wantSchemaPath: "./schemas/config.json",
		},
		{
			name:         "optional artifact",
			yaml:         "step: optional-step\nartifact: maybe\nas: opt\noptional: true\n",
			wantStep:     "optional-step",
			wantArtifact: "maybe",
			wantAs:       "opt",
			wantOptional: true,
		},
		{
			name:           "full specification",
			yaml:           "step: full\nartifact: complete\nas: all\ntype: json\nschema_path: schema.json\noptional: true\n",
			wantStep:       "full",
			wantArtifact:   "complete",
			wantAs:         "all",
			wantType:       "json",
			wantSchemaPath: "schema.json",
			wantOptional:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ref ArtifactRef
			err := yaml.Unmarshal([]byte(tt.yaml), &ref)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unmarshal error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if ref.Step != tt.wantStep {
				t.Errorf("Step = %q, want %q", ref.Step, tt.wantStep)
			}
			if ref.Artifact != tt.wantArtifact {
				t.Errorf("Artifact = %q, want %q", ref.Artifact, tt.wantArtifact)
			}
			if ref.As != tt.wantAs {
				t.Errorf("As = %q, want %q", ref.As, tt.wantAs)
			}
			if ref.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", ref.Type, tt.wantType)
			}
			if ref.SchemaPath != tt.wantSchemaPath {
				t.Errorf("SchemaPath = %q, want %q", ref.SchemaPath, tt.wantSchemaPath)
			}
			if ref.Optional != tt.wantOptional {
				t.Errorf("Optional = %v, want %v", ref.Optional, tt.wantOptional)
			}
		})
	}
}

func TestOutcomeDef_YAMLParsing(t *testing.T) {
	tests := []struct {
		name            string
		yaml            string
		wantType        string
		wantExtractFrom string
		wantJSONPath    string
		wantLabel       string
		wantErr         bool
	}{
		{
			name:            "PR outcome",
			yaml:            "type: pr\nextract_from: .agents/output/pr-result.json\njson_path: .pr_url\nlabel: Pull Request\n",
			wantType:        "pr",
			wantExtractFrom: ".agents/output/pr-result.json",
			wantJSONPath:    ".pr_url",
			wantLabel:       "Pull Request",
		},
		{
			name:            "URL outcome without label",
			yaml:            "type: url\nextract_from: output/publish-result.json\njson_path: .comment_url\n",
			wantType:        "url",
			wantExtractFrom: "output/publish-result.json",
			wantJSONPath:    ".comment_url",
			wantLabel:       "",
		},
		{
			name:            "deployment outcome",
			yaml:            "type: deployment\nextract_from: output/deploy.json\njson_path: .deploy_url\nlabel: Staging\n",
			wantType:        "deployment",
			wantExtractFrom: "output/deploy.json",
			wantJSONPath:    ".deploy_url",
			wantLabel:       "Staging",
		},
		{
			name:            "issue outcome with nested path",
			yaml:            "type: issue\nextract_from: output/result.json\njson_path: .github.issue_url\nlabel: Created Issue\n",
			wantType:        "issue",
			wantExtractFrom: "output/result.json",
			wantJSONPath:    ".github.issue_url",
			wantLabel:       "Created Issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var def OutcomeDef
			err := yaml.Unmarshal([]byte(tt.yaml), &def)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unmarshal error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if def.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", def.Type, tt.wantType)
			}
			if def.ExtractFrom != tt.wantExtractFrom {
				t.Errorf("ExtractFrom = %q, want %q", def.ExtractFrom, tt.wantExtractFrom)
			}
			if def.JSONPath != tt.wantJSONPath {
				t.Errorf("JSONPath = %q, want %q", def.JSONPath, tt.wantJSONPath)
			}
			if def.Label != tt.wantLabel {
				t.Errorf("Label = %q, want %q", def.Label, tt.wantLabel)
			}
		})
	}
}

func TestStep_OutcomesYAMLParsing(t *testing.T) {
	yamlContent := `
id: publish
persona: github-commenter
exec:
  type: prompt
  source: "post review"
output_artifacts:
  - name: result
    path: output/result.json
    type: json
outcomes:
  - type: url
    extract_from: output/result.json
    json_path: .comment_url
    label: "Review Comment"
  - type: pr
    extract_from: output/result.json
    json_path: .pr_url
`
	var step Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if len(step.Outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(step.Outcomes))
	}

	if step.Outcomes[0].Type != "url" {
		t.Errorf("Outcomes[0].Type = %q, want %q", step.Outcomes[0].Type, "url")
	}
	if step.Outcomes[0].Label != "Review Comment" {
		t.Errorf("Outcomes[0].Label = %q, want %q", step.Outcomes[0].Label, "Review Comment")
	}
	if step.Outcomes[1].Type != "pr" {
		t.Errorf("Outcomes[1].Type = %q, want %q", step.Outcomes[1].Type, "pr")
	}
	if step.Outcomes[1].Label != "" {
		t.Errorf("Outcomes[1].Label = %q, want empty", step.Outcomes[1].Label)
	}
}

func TestPipelineMetadata_YAMLParsing(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantRelease  bool
		wantDisabled bool
		wantErr      bool
	}{
		{
			name:        "release true parses correctly",
			yaml:        "name: test\nrelease: true\n",
			wantRelease: true,
		},
		{
			name:        "release false parses correctly",
			yaml:        "name: test\nrelease: false\n",
			wantRelease: false,
		},
		{
			name:        "absent release field defaults to false",
			yaml:        "name: test\n",
			wantRelease: false,
		},
		{
			name:         "disabled true is independent of release true",
			yaml:         "name: test\nrelease: true\ndisabled: true\n",
			wantRelease:  true,
			wantDisabled: true,
		},
		{
			name:         "disabled true with release false",
			yaml:         "name: test\nrelease: false\ndisabled: true\n",
			wantRelease:  false,
			wantDisabled: true,
		},
		{
			name:    "invalid release value produces error",
			yaml:    "name: test\nrelease: \"banana\"\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var meta PipelineMetadata
			err := yaml.Unmarshal([]byte(tt.yaml), &meta)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unmarshal error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if meta.Release != tt.wantRelease {
				t.Errorf("Release = %v, want %v", meta.Release, tt.wantRelease)
			}
			if meta.Disabled != tt.wantDisabled {
				t.Errorf("Disabled = %v, want %v", meta.Disabled, tt.wantDisabled)
			}
		})
	}
}

func TestOutcomeDef_JSONPathLabel_YAMLParsing(t *testing.T) {
	tests := []struct {
		name              string
		yaml              string
		wantJSONPathLabel string
	}{
		{
			name:              "json_path_label present",
			yaml:              "type: issue\nextract_from: output/result.json\njson_path: \".enhanced_issues[*].url\"\njson_path_label: \".enhanced_issues[*].issue_number\"\nlabel: Issues\n",
			wantJSONPathLabel: ".enhanced_issues[*].issue_number",
		},
		{
			name:              "json_path_label absent defaults to empty",
			yaml:              "type: url\nextract_from: output/result.json\njson_path: .comment_url\n",
			wantJSONPathLabel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var def OutcomeDef
			err := yaml.Unmarshal([]byte(tt.yaml), &def)
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if def.JSONPathLabel != tt.wantJSONPathLabel {
				t.Errorf("JSONPathLabel = %q, want %q", def.JSONPathLabel, tt.wantJSONPathLabel)
			}
		})
	}
}

func TestStep_TimeoutMinutes_YAMLParsing(t *testing.T) {
	tests := []struct {
		name               string
		yaml               string
		wantTimeoutMinutes int
	}{
		{
			name:               "timeout_minutes set to 90",
			yaml:               "id: implement\npersona: craftsman\ntimeout_minutes: 90\nexec:\n  type: prompt\n  source: \"do work\"\n",
			wantTimeoutMinutes: 90,
		},
		{
			name:               "timeout_minutes omitted defaults to zero",
			yaml:               "id: specify\npersona: navigator\nexec:\n  type: prompt\n  source: \"analyze\"\n",
			wantTimeoutMinutes: 0,
		},
		{
			name:               "timeout_minutes set to 5",
			yaml:               "id: quick-step\npersona: navigator\ntimeout_minutes: 5\nexec:\n  type: prompt\n  source: \"quick task\"\n",
			wantTimeoutMinutes: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var step Step
			err := yaml.Unmarshal([]byte(tt.yaml), &step)
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if step.TimeoutMinutes != tt.wantTimeoutMinutes {
				t.Errorf("TimeoutMinutes = %d, want %d", step.TimeoutMinutes, tt.wantTimeoutMinutes)
			}
		})
	}
}

func TestStep_GetTimeout(t *testing.T) {
	tests := []struct {
		name           string
		timeoutMinutes int
		wantDuration   time.Duration
	}{
		{
			name:           "zero returns zero duration",
			timeoutMinutes: 0,
			wantDuration:   0,
		},
		{
			name:           "positive value returns minutes duration",
			timeoutMinutes: 30,
			wantDuration:   30 * time.Minute,
		},
		{
			name:           "one minute",
			timeoutMinutes: 1,
			wantDuration:   1 * time.Minute,
		},
		{
			name:           "large value",
			timeoutMinutes: 120,
			wantDuration:   120 * time.Minute,
		},
		{
			name:           "negative value returns zero",
			timeoutMinutes: -5,
			wantDuration:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{TimeoutMinutes: tt.timeoutMinutes}
			got := step.GetTimeout()
			if got != tt.wantDuration {
				t.Errorf("GetTimeout() = %v, want %v", got, tt.wantDuration)
			}
		})
	}
}

func TestStep_MaxConcurrentAgents_YAMLParsing(t *testing.T) {
	tests := []struct {
		name                    string
		yaml                    string
		wantMaxConcurrentAgents int
	}{
		{
			name:                    "max_concurrent_agents set to 3",
			yaml:                    "id: implement\npersona: craftsman\nmax_concurrent_agents: 3\nexec:\n  type: prompt\n  source: \"do work\"\n",
			wantMaxConcurrentAgents: 3,
		},
		{
			name:                    "max_concurrent_agents omitted defaults to zero",
			yaml:                    "id: specify\npersona: navigator\nexec:\n  type: prompt\n  source: \"analyze\"\n",
			wantMaxConcurrentAgents: 0,
		},
		{
			name:                    "max_concurrent_agents set to 10",
			yaml:                    "id: parallel-step\npersona: implementer\nmax_concurrent_agents: 10\nexec:\n  type: prompt\n  source: \"parallel work\"\n",
			wantMaxConcurrentAgents: 10,
		},
		{
			name:                    "max_concurrent_agents set to 1",
			yaml:                    "id: single-step\npersona: navigator\nmax_concurrent_agents: 1\nexec:\n  type: prompt\n  source: \"single work\"\n",
			wantMaxConcurrentAgents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var step Step
			err := yaml.Unmarshal([]byte(tt.yaml), &step)
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if step.MaxConcurrentAgents != tt.wantMaxConcurrentAgents {
				t.Errorf("MaxConcurrentAgents = %d, want %d", step.MaxConcurrentAgents, tt.wantMaxConcurrentAgents)
			}
		})
	}
}

func TestArtifactRef_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ref     ArtifactRef
		wantErr bool
	}{
		{
			name:    "step only is valid",
			ref:     ArtifactRef{Step: "analyze", Artifact: "report", As: "input"},
			wantErr: false,
		},
		{
			name:    "pipeline only is valid",
			ref:     ArtifactRef{Pipeline: "other", Artifact: "report", As: "input"},
			wantErr: false,
		},
		{
			name:    "neither step nor pipeline is valid",
			ref:     ArtifactRef{Artifact: "report", As: "input"},
			wantErr: false,
		},
		{
			name:    "both step and pipeline is invalid",
			ref:     ArtifactRef{Step: "analyze", Pipeline: "other", Artifact: "report", As: "input"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.Validate("test-step", 0)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "mutually exclusive") {
					t.Errorf("error should mention mutual exclusivity, got: %v", err)
				}
			}
		})
	}
}

func TestStep_Optional_YAMLParsing(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantOptional bool
	}{
		{
			name:         "optional true",
			yaml:         "id: notify\npersona: notifier\noptional: true\nexec:\n  type: prompt\n  source: \"send notification\"\n",
			wantOptional: true,
		},
		{
			name:         "optional false (explicit)",
			yaml:         "id: build\npersona: builder\noptional: false\nexec:\n  type: prompt\n  source: \"build project\"\n",
			wantOptional: false,
		},
		{
			name:         "optional omitted defaults to false",
			yaml:         "id: deploy\npersona: deployer\nexec:\n  type: prompt\n  source: \"deploy\"\n",
			wantOptional: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var step Step
			err := yaml.Unmarshal([]byte(tt.yaml), &step)
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			if step.Optional != tt.wantOptional {
				t.Errorf("Optional = %v, want %v", step.Optional, tt.wantOptional)
			}

			if step.Optional != tt.wantOptional {
				t.Errorf("step.Optional = %v, want %v", step.Optional, tt.wantOptional)
			}

			// Round-trip: marshal and unmarshal
			out, err := yaml.Marshal(&step)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			var roundTrip Step
			err = yaml.Unmarshal(out, &roundTrip)
			if err != nil {
				t.Fatalf("unexpected unmarshal error on round-trip: %v", err)
			}

			if roundTrip.Optional != tt.wantOptional {
				t.Errorf("round-trip Optional = %v, want %v", roundTrip.Optional, tt.wantOptional)
			}
		})
	}
}

func TestStepSkillsYAMLRoundTrip(t *testing.T) {
	yamlStr := `kind: WavePipeline
metadata:
  name: test-pipeline
input:
  source: cli
steps:
  - id: implement
    persona: craftsman
    skills: [golang, gh-cli]
    exec:
      type: prompt
      source: "do work"
`
	var p Pipeline
	if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(p.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(p.Steps))
	}
	want := []string{"golang", "gh-cli"}
	if !reflect.DeepEqual(p.Steps[0].Skills, want) {
		t.Errorf("Step.Skills = %v, want %v", p.Steps[0].Skills, want)
	}
}

func TestPipelineSkillsYAMLRoundTrip(t *testing.T) {
	t.Run("pipeline with skills", func(t *testing.T) {
		yamlStr := `kind: WavePipeline
metadata:
  name: test-pipeline
skills:
  - golang
  - testing
input:
  source: cli
steps:
  - id: step1
    persona: agent1
    exec:
      type: prompt
      source: "do work"
`
		var p Pipeline
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		want := []string{"golang", "testing"}
		if !reflect.DeepEqual(p.Skills, want) {
			t.Errorf("Skills = %v, want %v", p.Skills, want)
		}

		// Round-trip
		out, err := yaml.Marshal(&p)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var p2 Pipeline
		if err := yaml.Unmarshal(out, &p2); err != nil {
			t.Fatalf("round-trip unmarshal error: %v", err)
		}

		if !reflect.DeepEqual(p2.Skills, want) {
			t.Errorf("round-trip Skills = %v, want %v", p2.Skills, want)
		}
	})

	t.Run("pipeline skills does not affect requires.skills", func(t *testing.T) {
		yamlStr := `kind: WavePipeline
metadata:
  name: test-pipeline
skills:
  - golang
  - testing
requires:
  skills:
    speckit:
      install: "wave skill install speckit"
      check: "test -d .agents/skills/speckit"
input:
  source: cli
steps:
  - id: step1
    persona: agent1
    exec:
      type: prompt
      source: "do work"
`
		var p Pipeline
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		// Verify top-level Skills
		wantSkills := []string{"golang", "testing"}
		if !reflect.DeepEqual(p.Skills, wantSkills) {
			t.Errorf("Skills = %v, want %v", p.Skills, wantSkills)
		}

		// Verify Requires.Skills map is independent
		if p.Requires == nil {
			t.Fatal("Requires should not be nil")
		}
		if len(p.Requires.Skills) != 1 {
			t.Fatalf("Requires.Skills length = %d, want 1", len(p.Requires.Skills))
		}
		sc, ok := p.Requires.Skills["speckit"]
		if !ok {
			t.Fatal("Requires.Skills[\"speckit\"] not found")
		}
		if sc.Install != "wave skill install speckit" {
			t.Errorf("Requires.Skills[\"speckit\"].Install = %q, want %q", sc.Install, "wave skill install speckit")
		}

		// Round-trip
		out, err := yaml.Marshal(&p)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var p2 Pipeline
		if err := yaml.Unmarshal(out, &p2); err != nil {
			t.Fatalf("round-trip unmarshal error: %v", err)
		}

		if !reflect.DeepEqual(p2.Skills, wantSkills) {
			t.Errorf("round-trip Skills = %v, want %v", p2.Skills, wantSkills)
		}
		if p2.Requires == nil || len(p2.Requires.Skills) != 1 {
			t.Errorf("round-trip Requires.Skills was lost or changed")
		}
	})

	t.Run("pipeline without skills key", func(t *testing.T) {
		yamlStr := `kind: WavePipeline
metadata:
  name: test-pipeline
input:
  source: cli
steps:
  - id: step1
    persona: agent1
    exec:
      type: prompt
      source: "do work"
`
		var p Pipeline
		if err := yaml.Unmarshal([]byte(yamlStr), &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if p.Skills != nil {
			t.Errorf("Skills should be nil when key is absent, got %v", p.Skills)
		}
	})
}

func TestHandoverConfig_EffectiveContracts(t *testing.T) {
	t.Run("plural contracts takes precedence over singular", func(t *testing.T) {
		h := HandoverConfig{
			Contract:  ContractConfig{Type: "json_schema"},
			Contracts: []ContractConfig{{Type: "test_suite"}, {Type: "agent_review"}},
		}
		got := h.EffectiveContracts()
		if len(got) != 2 {
			t.Fatalf("expected 2 contracts, got %d", len(got))
		}
		if got[0].Type != "test_suite" || got[1].Type != "agent_review" {
			t.Errorf("unexpected contracts: %v", got)
		}
	})

	t.Run("singular contract wrapped in slice when plural absent", func(t *testing.T) {
		h := HandoverConfig{
			Contract: ContractConfig{Type: "json_schema"},
		}
		got := h.EffectiveContracts()
		if len(got) != 1 {
			t.Fatalf("expected 1 contract, got %d", len(got))
		}
		if got[0].Type != "json_schema" {
			t.Errorf("expected json_schema, got %q", got[0].Type)
		}
	})

	t.Run("nil when neither set", func(t *testing.T) {
		h := HandoverConfig{}
		got := h.EffectiveContracts()
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("plural only when singular is empty", func(t *testing.T) {
		h := HandoverConfig{
			Contracts: []ContractConfig{{Type: "format"}},
		}
		got := h.EffectiveContracts()
		if len(got) != 1 || got[0].Type != "format" {
			t.Errorf("expected [format], got %v", got)
		}
	})
}

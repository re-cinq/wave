package pipeline

import (
	"testing"

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
			yaml:       "name: report\npath: .wave/output/report.json\ntype: json\n",
			wantName:   "report",
			wantPath:   ".wave/output/report.json",
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

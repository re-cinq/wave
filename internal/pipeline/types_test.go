package pipeline

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStep_OptionalYAMLParsing(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantOptional bool
	}{
		{
			name:         "optional true parses correctly",
			yaml:         "id: test\npersona: nav\noptional: true\n",
			wantOptional: true,
		},
		{
			name:         "optional false parses correctly",
			yaml:         "id: test\npersona: nav\noptional: false\n",
			wantOptional: false,
		},
		{
			name:         "absent optional defaults to false",
			yaml:         "id: test\npersona: nav\n",
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

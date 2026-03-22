package manifest

import (
	"testing"
)

func TestProjectVars(t *testing.T) {
	tests := []struct {
		name     string
		project  *Project
		expected map[string]string
	}{
		{
			name:     "nil project returns empty map",
			project:  nil,
			expected: map[string]string{},
		},
		{
			name:     "empty project returns empty map",
			project:  &Project{},
			expected: map[string]string{},
		},
		{
			name: "all fields populated",
			project: &Project{
				Language:      "go",
				Flavour:       "go",
				TestCommand:   "go test ./...",
				LintCommand:   "golangci-lint run ./...",
				BuildCommand:  "go build ./...",
				FormatCommand: "gofmt -l .",
				SourceGlob:    "*.go",
				Skill:         "golang",
			},
			expected: map[string]string{
				"project.language":       "go",
				"project.flavour":        "go",
				"project.test_command":   "go test ./...",
				"project.lint_command":   "golangci-lint run ./...",
				"project.build_command":  "go build ./...",
				"project.format_command": "gofmt -l .",
				"project.source_glob":    "*.go",
				"project.skill":          "golang",
			},
		},
		{
			name: "rust project",
			project: &Project{
				Language:      "rust",
				Flavour:       "rust",
				TestCommand:   "cargo test",
				LintCommand:   "cargo clippy -- -D warnings",
				BuildCommand:  "cargo build",
				FormatCommand: "cargo fmt -- --check",
				SourceGlob:    "*.rs",
				Skill:         "rust",
			},
			expected: map[string]string{
				"project.language":       "rust",
				"project.flavour":        "rust",
				"project.test_command":   "cargo test",
				"project.lint_command":   "cargo clippy -- -D warnings",
				"project.build_command":  "cargo build",
				"project.format_command": "cargo fmt -- --check",
				"project.source_glob":    "*.rs",
				"project.skill":          "rust",
			},
		},
		{
			name: "partial fields only emit populated keys",
			project: &Project{
				Language:    "typescript",
				TestCommand: "npm test",
				Skill:       "typescript",
			},
			expected: map[string]string{
				"project.language":     "typescript",
				"project.test_command": "npm test",
				"project.skill":       "typescript",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := tt.project.ProjectVars()

			if len(vars) != len(tt.expected) {
				t.Errorf("expected %d vars, got %d: %v", len(tt.expected), len(vars), vars)
			}

			for k, want := range tt.expected {
				got, ok := vars[k]
				if !ok {
					t.Errorf("expected key %q to exist", k)
					continue
				}
				if got != want {
					t.Errorf("vars[%q] = %q, want %q", k, got, want)
				}
			}

			// Verify no extra keys
			for k := range vars {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("unexpected key %q in vars", k)
				}
			}
		})
	}
}

func TestRuntimeSandboxResolveBackend(t *testing.T) {
	tests := []struct {
		name    string
		sandbox RuntimeSandbox
		want    string
	}{
		{
			name:    "backend set to docker",
			sandbox: RuntimeSandbox{Backend: "docker"},
			want:    "docker",
		},
		{
			name:    "backend set to bubblewrap",
			sandbox: RuntimeSandbox{Backend: "bubblewrap"},
			want:    "bubblewrap",
		},
		{
			name:    "backend set to none",
			sandbox: RuntimeSandbox{Backend: "none"},
			want:    "none",
		},
		{
			name:    "no backend enabled true returns bubblewrap (legacy)",
			sandbox: RuntimeSandbox{Enabled: true},
			want:    "bubblewrap",
		},
		{
			name:    "no backend enabled false returns none",
			sandbox: RuntimeSandbox{Enabled: false},
			want:    "none",
		},
		{
			name:    "backend wins over enabled",
			sandbox: RuntimeSandbox{Backend: "docker", Enabled: true},
			want:    "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sandbox.ResolveBackend()
			if got != tt.want {
				t.Errorf("ResolveBackend() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuntimeSandboxGetDockerImage(t *testing.T) {
	tests := []struct {
		name    string
		sandbox RuntimeSandbox
		want    string
	}{
		{
			name:    "custom image set",
			sandbox: RuntimeSandbox{DockerImage: "myorg/myimage:v2"},
			want:    "myorg/myimage:v2",
		},
		{
			name:    "empty returns default ubuntu:24.04",
			sandbox: RuntimeSandbox{},
			want:    "ubuntu:24.04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sandbox.GetDockerImage()
			if got != tt.want {
				t.Errorf("GetDockerImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

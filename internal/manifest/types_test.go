package manifest

import (
	"strings"
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
				"project.language":              "go",
				"project.flavour":               "go",
				"project.test_command":          "go test ./...",
				"project.contract_test_command": "go test ./...",
				"project.lint_command":          "golangci-lint run ./...",
				"project.build_command":         "go build ./...",
				"project.format_command":        "gofmt -l .",
				"project.source_glob":           "*.go",
				"project.skill":                 "golang",
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
				"project.language":              "rust",
				"project.flavour":               "rust",
				"project.test_command":          "cargo test",
				"project.contract_test_command": "cargo test",
				"project.lint_command":          "cargo clippy -- -D warnings",
				"project.build_command":         "cargo build",
				"project.format_command":        "cargo fmt -- --check",
				"project.source_glob":           "*.rs",
				"project.skill":                 "rust",
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
				"project.language":              "typescript",
				"project.test_command":          "npm test",
				"project.contract_test_command": "npm test",
				"project.skill":                 "typescript",
			},
		},
		{
			name: "explicit contract_test_command overrides fallback",
			project: &Project{
				Language:            "go",
				TestCommand:         "go test -race ./...",
				ContractTestCommand: "go test ./...",
			},
			expected: map[string]string{
				"project.language":              "go",
				"project.test_command":          "go test -race ./...",
				"project.contract_test_command": "go test ./...",
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

func TestProjectVars_Services(t *testing.T) {
	p := &Project{
		Language:    "go",
		TestCommand: "go test ./...",
		Services: map[string]ServiceConfig{
			"api": {
				Path:         "services/api",
				Language:     "go",
				TestCommand:  "go test ./services/api/...",
				BuildCommand: "go build ./services/api",
				SourceGlob:   "services/api/**/*.go",
			},
			"web": {
				Path:                "services/web",
				Language:            "typescript",
				TestCommand:         "npm test",
				ContractTestCommand: "npm run test:contracts",
			},
		},
	}

	vars := p.ProjectVars()

	// Root vars still work
	if vars["project.language"] != "go" {
		t.Errorf("expected project.language=go, got %q", vars["project.language"])
	}

	// API service vars
	if vars["project.services.api.path"] != "services/api" {
		t.Errorf("expected api.path, got %q", vars["project.services.api.path"])
	}
	if vars["project.services.api.language"] != "go" {
		t.Errorf("expected api.language=go, got %q", vars["project.services.api.language"])
	}
	if vars["project.services.api.test_command"] != "go test ./services/api/..." {
		t.Errorf("expected api.test_command, got %q", vars["project.services.api.test_command"])
	}
	if vars["project.services.api.build_command"] != "go build ./services/api" {
		t.Errorf("expected api.build_command, got %q", vars["project.services.api.build_command"])
	}
	// API has no explicit contract_test_command, should fall back to test_command
	if vars["project.services.api.contract_test_command"] != "go test ./services/api/..." {
		t.Errorf("expected api.contract_test_command fallback, got %q", vars["project.services.api.contract_test_command"])
	}

	// Web service vars
	if vars["project.services.web.language"] != "typescript" {
		t.Errorf("expected web.language=typescript, got %q", vars["project.services.web.language"])
	}
	// Web has explicit contract_test_command, should NOT fall back
	if vars["project.services.web.contract_test_command"] != "npm run test:contracts" {
		t.Errorf("expected web.contract_test_command explicit, got %q", vars["project.services.web.contract_test_command"])
	}
}

func TestOntologyVars(t *testing.T) {
	tests := []struct {
		name     string
		ontology *Ontology
		expected map[string]string
	}{
		{
			name:     "nil ontology returns empty map",
			ontology: nil,
			expected: map[string]string{},
		},
		{
			name:     "empty ontology returns empty map",
			ontology: &Ontology{},
			expected: map[string]string{},
		},
		{
			name: "telos only",
			ontology: &Ontology{
				Telos: "Build the best orchestrator",
			},
			expected: map[string]string{
				"ontology.telos": "Build the best orchestrator",
			},
		},
		{
			name: "full ontology",
			ontology: &Ontology{
				Telos: "Financial assistant for SMEs",
				Contexts: []OntologyContext{
					{Name: "billing", Description: "Payment processing"},
					{Name: "auth", Description: "Authentication and authorization"},
				},
				Conventions: map[string]string{
					"commits": "conventional",
					"testing": "table-driven",
				},
			},
			expected: map[string]string{
				"ontology.telos":              "Financial assistant for SMEs",
				"ontology.context.billing":    "Payment processing",
				"ontology.context.auth":       "Authentication and authorization",
				"ontology.convention.commits": "conventional",
				"ontology.convention.testing": "table-driven",
			},
		},
		{
			name: "context without description is omitted",
			ontology: &Ontology{
				Contexts: []OntologyContext{
					{Name: "billing"},
				},
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := tt.ontology.OntologyVars()

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

			for k := range vars {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("unexpected key %q in vars", k)
				}
			}
		})
	}
}

func TestOntologyRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		ontology *Ontology
		filter   []string
		contains []string
		absent   []string
	}{
		{
			name:     "nil ontology returns empty",
			ontology: nil,
			contains: nil,
		},
		{
			name: "renders telos",
			ontology: &Ontology{
				Telos: "Build great software",
			},
			contains: []string{"## Project Ontology", "**Purpose**: Build great software"},
		},
		{
			name: "renders contexts with invariants",
			ontology: &Ontology{
				Contexts: []OntologyContext{
					{
						Name:        "billing",
						Description: "Payment processing",
						Invariants:  []string{"Never write to Uniconta"},
					},
				},
			},
			contains: []string{"### billing", "Payment processing", "- Never write to Uniconta"},
		},
		{
			name: "filters contexts",
			ontology: &Ontology{
				Contexts: []OntologyContext{
					{Name: "billing", Description: "Payments"},
					{Name: "auth", Description: "Authentication"},
				},
			},
			filter:   []string{"billing"},
			contains: []string{"### billing"},
			absent:   []string{"### auth"},
		},
		{
			name: "renders conventions",
			ontology: &Ontology{
				Conventions: map[string]string{
					"commits": "conventional",
				},
			},
			contains: []string{"### Conventions", "**commits**: conventional"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ontology.RenderMarkdown(tt.filter)

			for _, s := range tt.contains {
				if !containsString(result, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, result)
				}
			}
			for _, s := range tt.absent {
				if containsString(result, s) {
					t.Errorf("expected output to NOT contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && strings.Contains(haystack, needle)
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

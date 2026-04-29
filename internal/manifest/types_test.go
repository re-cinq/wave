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

func TestDefaultComplexityMap(t *testing.T) {
	m := DefaultComplexityMap()

	if len(m) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(m))
	}

	tests := []struct {
		tier string
		want string
	}{
		{"cheapest", "claude-haiku-4-5"},
		{"balanced", ""},
		{"strongest", "claude-opus-4"},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got, ok := m[tt.tier]
			if !ok {
				t.Fatalf("missing key %q", tt.tier)
			}
			if got != tt.want {
				t.Errorf("DefaultComplexityMap()[%q] = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestResolveComplexityModel(t *testing.T) {
	tests := []struct {
		name    string
		routing *RoutingConfig
		tier    string
		want    string
	}{
		{
			name:    "nil receiver falls back to default cheapest",
			routing: nil,
			tier:    "cheapest",
			want:    "claude-haiku-4-5",
		},
		{
			name:    "nil receiver balanced returns empty",
			routing: nil,
			tier:    "balanced",
			want:    "",
		},
		{
			name:    "nil receiver strongest",
			routing: nil,
			tier:    "strongest",
			want:    "claude-opus-4",
		},
		{
			name:    "nil receiver unknown tier returns empty",
			routing: nil,
			tier:    "extreme",
			want:    "",
		},
		{
			name:    "empty ComplexityMap falls back to defaults",
			routing: &RoutingConfig{},
			tier:    "cheapest",
			want:    "claude-haiku-4-5",
		},
		{
			name: "custom ComplexityMap overrides tier",
			routing: &RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "my-custom-haiku",
				},
			},
			tier: "cheapest",
			want: "my-custom-haiku",
		},
		{
			name: "partial override falls through to default for missing tier",
			routing: &RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "my-haiku",
				},
			},
			tier: "balanced",
			want: "",
		},
		{
			name: "custom map overrides balanced to non-empty",
			routing: &RoutingConfig{
				ComplexityMap: map[string]string{
					"balanced": "claude-sonnet-4",
				},
			},
			tier: "balanced",
			want: "claude-sonnet-4",
		},
		{
			name: "unknown tier not in any map returns empty",
			routing: &RoutingConfig{
				ComplexityMap: map[string]string{
					"cheapest": "haiku",
				},
			},
			tier: "extreme",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.routing.ResolveComplexityModel(tt.tier)
			if got != tt.want {
				t.Errorf("ResolveComplexityModel(%q) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestEffectiveDefaultTier(t *testing.T) {
	tests := []struct {
		name    string
		routing *RoutingConfig
		want    string
	}{
		{
			name:    "nil receiver returns balanced",
			routing: nil,
			want:    "balanced",
		},
		{
			name:    "empty DefaultTier returns balanced",
			routing: &RoutingConfig{},
			want:    "balanced",
		},
		{
			name:    "explicit DefaultTier returned as-is",
			routing: &RoutingConfig{DefaultTier: "cheapest"},
			want:    "cheapest",
		},
		{
			name:    "non-standard value returned as-is",
			routing: &RoutingConfig{DefaultTier: "custom-tier"},
			want:    "custom-tier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.routing.EffectiveDefaultTier()
			if got != tt.want {
				t.Errorf("EffectiveDefaultTier() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRetrosConfig_IsEnabled(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name   string
		config *RetrosConfig
		want   bool
	}{
		{
			name:   "nil receiver defaults to true",
			config: nil,
			want:   true,
		},
		{
			name:   "nil Enabled field defaults to true",
			config: &RetrosConfig{},
			want:   true,
		},
		{
			name:   "explicit true",
			config: &RetrosConfig{Enabled: boolPtr(true)},
			want:   true,
		},
		{
			name:   "explicit false",
			config: &RetrosConfig{Enabled: boolPtr(false)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnabled()
			if got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetrosConfig_IsNarrateEnabled(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name   string
		config *RetrosConfig
		want   bool
	}{
		{
			name:   "nil receiver defaults to true",
			config: nil,
			want:   true,
		},
		{
			name:   "nil Narrate field defaults to true",
			config: &RetrosConfig{},
			want:   true,
		},
		{
			name:   "explicit true",
			config: &RetrosConfig{Narrate: boolPtr(true)},
			want:   true,
		},
		{
			name:   "explicit false",
			config: &RetrosConfig{Narrate: boolPtr(false)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsNarrateEnabled()
			if got != tt.want {
				t.Errorf("IsNarrateEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetrosConfig_GetNarrateModel(t *testing.T) {
	tests := []struct {
		name   string
		config *RetrosConfig
		want   string
	}{
		{
			name:   "nil receiver returns cheapest default",
			config: nil,
			want:   "claude-haiku-4-5",
		},
		{
			name:   "empty NarrateModel returns cheapest default",
			config: &RetrosConfig{},
			want:   "claude-haiku-4-5",
		},
		{
			name:   "explicit NarrateModel override",
			config: &RetrosConfig{NarrateModel: "claude-sonnet-4"},
			want:   "claude-sonnet-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetNarrateModel()
			if got != tt.want {
				t.Errorf("GetNarrateModel() = %q, want %q", got, tt.want)
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

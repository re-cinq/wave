package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAllowedTools(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "bare tools unchanged",
			input: []string{"Read", "Write", "Edit", "Bash"},
			want:  []string{"Read", "Write", "Edit", "Bash"},
		},
		{
			name:  "Write scoped entries normalized to bare Write",
			input: []string{"Read", "Write(output/*)", "Write(artifact.json)"},
			want:  []string{"Read", "Write"},
		},
		{
			name:  "deduplicates after normalization",
			input: []string{"Write(output/*)", "Write(artifact.json)", "Read"},
			want:  []string{"Write", "Read"},
		},
		{
			name:  "preserves Bash scoped entries",
			input: []string{"Bash(go test*)", "Bash(git log*)", "Read"},
			want:  []string{"Bash(go test*)", "Bash(git log*)", "Read"},
		},
		{
			name:  "mixed scoped and bare",
			input: []string{"Read", "Glob", "Grep", "WebSearch", "Write(output/*)", "Write"},
			want:  []string{"Read", "Glob", "Grep", "WebSearch", "Write"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  nil,
		},
		{
			name:  "bare Write preserved",
			input: []string{"Write"},
			want:  []string{"Write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAllowedTools(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("normalizeAllowedTools(%v) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("normalizeAllowedTools(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSettingsJSONFormat(t *testing.T) {
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:       "test",
		WorkspacePath: tmpDir,
		Model:         "sonnet",
		Temperature:   0.5,
		AllowedTools:  []string{"Read", "Write(output/*)", "Glob"},
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	// Verify it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}

	// Verify permissions.allow field exists (not allowed_tools)
	if _, ok := raw["allowed_tools"]; ok {
		t.Error("settings.json should not have 'allowed_tools' field")
	}

	perms, ok := raw["permissions"].(map[string]interface{})
	if !ok {
		t.Fatal("settings.json missing 'permissions' object")
	}

	allow, ok := perms["allow"].([]interface{})
	if !ok {
		t.Fatal("settings.json missing 'permissions.allow' array")
	}

	// Verify Write(output/*) was normalized to Write
	allowStrs := make([]string, len(allow))
	for i, v := range allow {
		allowStrs[i] = v.(string)
	}

	expected := []string{"Read", "Write", "Glob"}
	if len(allowStrs) != len(expected) {
		t.Fatalf("permissions.allow = %v, want %v", allowStrs, expected)
	}
	for i, want := range expected {
		if allowStrs[i] != want {
			t.Errorf("permissions.allow[%d] = %q, want %q", i, allowStrs[i], want)
		}
	}
}

func TestBuildArgsNormalizesAllowedTools(t *testing.T) {
	adapter := NewClaudeAdapter()
	cfg := AdapterRunConfig{
		AllowedTools: []string{"Read", "Write(output/*)", "Write(artifact.json)", "Glob"},
		Prompt:       "test",
	}

	args := adapter.buildArgs(cfg)

	// Find the --allowedTools value
	var allowedToolsArg string
	for i, arg := range args {
		if arg == "--allowedTools" && i+1 < len(args) {
			allowedToolsArg = args[i+1]
			break
		}
	}

	if allowedToolsArg == "" {
		t.Fatal("--allowedTools not found in args")
	}

	expected := "Read,Write,Glob"
	if allowedToolsArg != expected {
		t.Errorf("--allowedTools = %q, want %q", allowedToolsArg, expected)
	}
}

// --- Phase 2 tests: deny rules, sandbox settings, CLAUDE.md restrictions, env hygiene ---

func TestSettingsJSONDenyRules(t *testing.T) {
	tests := []struct {
		name     string
		deny     []string
		wantDeny []string
	}{
		{
			name:     "deny rules written to settings.json",
			deny:     []string{"Write(*)", "Edit(*)", "Bash(rm *)"},
			wantDeny: []string{"Write(*)", "Edit(*)", "Bash(rm *)"},
		},
		{
			name:     "empty deny omitted from JSON",
			deny:     nil,
			wantDeny: nil,
		},
		{
			name:     "single deny rule",
			deny:     []string{"Bash(rm -rf /*)"},
			wantDeny: []string{"Bash(rm -rf /*)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewClaudeAdapter()
			tmpDir := t.TempDir()

			cfg := AdapterRunConfig{
				Persona:      "navigator",
				Model:        "sonnet",
				AllowedTools: []string{"Read", "Glob", "Grep"},
				DenyTools:    tt.deny,
			}

			if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
				t.Fatalf("prepareWorkspace failed: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(tmpDir, ".claude", "settings.json"))
			if err != nil {
				t.Fatalf("failed to read settings.json: %v", err)
			}

			var settings ClaudeSettings
			if err := json.Unmarshal(data, &settings); err != nil {
				t.Fatalf("failed to parse settings.json: %v", err)
			}

			if tt.wantDeny == nil {
				if len(settings.Permissions.Deny) != 0 {
					t.Errorf("expected no deny rules, got %v", settings.Permissions.Deny)
				}
			} else {
				if len(settings.Permissions.Deny) != len(tt.wantDeny) {
					t.Fatalf("deny rules = %v, want %v", settings.Permissions.Deny, tt.wantDeny)
				}
				for i, want := range tt.wantDeny {
					if settings.Permissions.Deny[i] != want {
						t.Errorf("deny[%d] = %q, want %q", i, settings.Permissions.Deny[i], want)
					}
				}
			}
		})
	}
}

func TestSettingsJSONSandboxSettings(t *testing.T) {
	tests := []struct {
		name           string
		allowedDomains []string
		wantSandbox    bool
		wantDomains    []string
	}{
		{
			name:           "sandbox enabled with domains",
			allowedDomains: []string{"api.anthropic.com", "github.com", "*.github.com"},
			wantSandbox:    true,
			wantDomains:    []string{"api.anthropic.com", "github.com", "*.github.com"},
		},
		{
			name:           "no sandbox when no domains",
			allowedDomains: nil,
			wantSandbox:    false,
		},
		{
			name:           "single domain",
			allowedDomains: []string{"api.anthropic.com"},
			wantSandbox:    true,
			wantDomains:    []string{"api.anthropic.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewClaudeAdapter()
			tmpDir := t.TempDir()

			cfg := AdapterRunConfig{
				Persona:        "implementer",
				Model:          "opus",
				AllowedTools:   []string{"Read", "Write", "Edit", "Bash"},
				AllowedDomains: tt.allowedDomains,
			}

			if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
				t.Fatalf("prepareWorkspace failed: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(tmpDir, ".claude", "settings.json"))
			if err != nil {
				t.Fatalf("failed to read settings.json: %v", err)
			}

			var settings ClaudeSettings
			if err := json.Unmarshal(data, &settings); err != nil {
				t.Fatalf("failed to parse settings.json: %v", err)
			}

			if tt.wantSandbox {
				if settings.Sandbox == nil {
					t.Fatal("expected sandbox settings, got nil")
				}
				if !settings.Sandbox.Enabled {
					t.Error("expected sandbox.enabled = true")
				}
				if !settings.Sandbox.AutoAllowBashIfSandboxed {
					t.Error("expected sandbox.autoAllowBashIfSandboxed = true")
				}
				if settings.Sandbox.AllowUnsandboxedCommands {
					t.Error("expected sandbox.allowUnsandboxedCommands = false")
				}
				if settings.Sandbox.Network == nil {
					t.Fatal("expected sandbox.network, got nil")
				}
				if len(settings.Sandbox.Network.AllowedDomains) != len(tt.wantDomains) {
					t.Fatalf("allowedDomains = %v, want %v", settings.Sandbox.Network.AllowedDomains, tt.wantDomains)
				}
				for i, want := range tt.wantDomains {
					if settings.Sandbox.Network.AllowedDomains[i] != want {
						t.Errorf("allowedDomains[%d] = %q, want %q", i, settings.Sandbox.Network.AllowedDomains[i], want)
					}
				}
			} else {
				if settings.Sandbox != nil {
					t.Errorf("expected no sandbox settings, got %+v", settings.Sandbox)
				}
			}
		})
	}
}

func TestCLAUDEMDRestrictionSection(t *testing.T) {
	tests := []struct {
		name         string
		cfg          AdapterRunConfig
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "deny rules appear in CLAUDE.md",
			cfg: AdapterRunConfig{
				Persona:      "navigator",
				Model:        "sonnet",
				SystemPrompt: "# Navigator\n\nYou are the navigator.",
				AllowedTools: []string{"Read", "Glob", "Grep"},
				DenyTools:    []string{"Write(*)", "Edit(*)"},
			},
			wantContains: []string{
				"## Restrictions",
				"### Denied Tools",
				"- `Write(*)`",
				"- `Edit(*)`",
				"### Allowed Tools",
				"- `Read`",
				"- `Glob`",
				"- `Grep`",
				"# Navigator",
			},
		},
		{
			name: "network domains appear in CLAUDE.md",
			cfg: AdapterRunConfig{
				Persona:        "implementer",
				Model:          "opus",
				SystemPrompt:   "# Implementer",
				AllowedTools:   []string{"Read", "Write"},
				AllowedDomains: []string{"api.anthropic.com", "github.com"},
			},
			wantContains: []string{
				"### Network Access",
				"- `api.anthropic.com`",
				"- `github.com`",
			},
		},
		{
			name: "no restrictions when nothing configured",
			cfg: AdapterRunConfig{
				Persona:      "test",
				Model:        "sonnet",
				SystemPrompt: "# Test persona",
			},
			wantAbsent: []string{
				"## Restrictions",
				"### Denied Tools",
				"### Allowed Tools",
				"### Network Access",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewClaudeAdapter()
			tmpDir := t.TempDir()

			if err := a.prepareWorkspace(tmpDir, tt.cfg); err != nil {
				t.Fatalf("prepareWorkspace failed: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
			if err != nil {
				t.Fatalf("failed to read CLAUDE.md: %v", err)
			}
			content := string(data)

			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("CLAUDE.md missing %q\nGot:\n%s", want, content)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(content, absent) {
					t.Errorf("CLAUDE.md should not contain %q\nGot:\n%s", absent, content)
				}
			}
		})
	}
}

func TestBuildRestrictionSection(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AdapterRunConfig
		wantLen int // 0 means empty string
	}{
		{
			name:    "empty config produces empty section",
			cfg:     AdapterRunConfig{},
			wantLen: 0,
		},
		{
			name: "deny only",
			cfg: AdapterRunConfig{
				DenyTools: []string{"Bash(rm *)"},
			},
		},
		{
			name: "allow only",
			cfg: AdapterRunConfig{
				AllowedTools: []string{"Read"},
			},
		},
		{
			name: "domains only",
			cfg: AdapterRunConfig{
				AllowedDomains: []string{"api.anthropic.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRestrictionSection(tt.cfg)
			if tt.wantLen == 0 && len(tt.cfg.DenyTools) == 0 && len(tt.cfg.AllowedTools) == 0 && len(tt.cfg.AllowedDomains) == 0 {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
			} else {
				if result == "" {
					t.Error("expected non-empty restriction section")
				}
				if !strings.Contains(result, "## Restrictions") {
					t.Error("restriction section missing header")
				}
			}
		})
	}
}

func TestBuildEnvironmentCurated(t *testing.T) {
	a := NewClaudeAdapter()

	// Set a canary env var that should NOT leak
	t.Setenv("CANARY_SECRET_KEY", "super-secret-value")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-secret-12345")

	// Set a passthrough var that SHOULD appear
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	t.Setenv("GH_TOKEN", "ghp_test_token")

	cfg := AdapterRunConfig{
		EnvPassthrough: []string{"ANTHROPIC_API_KEY", "GH_TOKEN"},
		Env:            []string{"STEP_VAR=step-value"},
	}

	env := a.buildEnvironment(cfg)
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Base vars must be present
	for _, key := range []string{"HOME", "PATH", "TERM", "TMPDIR"} {
		if _, ok := envMap[key]; !ok {
			t.Errorf("missing required base env var %s", key)
		}
	}

	// Telemetry disabling must be present
	for _, key := range []string{"DISABLE_TELEMETRY", "DISABLE_ERROR_REPORTING", "CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY", "DISABLE_BUG_COMMAND"} {
		if envMap[key] != "1" {
			t.Errorf("expected %s=1, got %q", key, envMap[key])
		}
	}

	// Passthrough vars must appear
	if envMap["ANTHROPIC_API_KEY"] != "sk-ant-test-key" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want %q", envMap["ANTHROPIC_API_KEY"], "sk-ant-test-key")
	}
	if envMap["GH_TOKEN"] != "ghp_test_token" {
		t.Errorf("GH_TOKEN = %q, want %q", envMap["GH_TOKEN"], "ghp_test_token")
	}

	// Step-specific env must appear
	if envMap["STEP_VAR"] != "step-value" {
		t.Errorf("STEP_VAR = %q, want %q", envMap["STEP_VAR"], "step-value")
	}

	// Canary vars MUST NOT leak
	if _, ok := envMap["CANARY_SECRET_KEY"]; ok {
		t.Error("CANARY_SECRET_KEY leaked through curated environment")
	}
	if _, ok := envMap["AWS_SECRET_ACCESS_KEY"]; ok {
		t.Error("AWS_SECRET_ACCESS_KEY leaked through curated environment")
	}
}

func TestBuildEnvironmentEmptyPassthrough(t *testing.T) {
	a := NewClaudeAdapter()

	// Var exists in host env but not in passthrough list
	t.Setenv("SOME_VAR", "some-value")

	cfg := AdapterRunConfig{
		EnvPassthrough: nil,
	}

	env := a.buildEnvironment(cfg)
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if _, ok := envMap["SOME_VAR"]; ok {
		t.Error("SOME_VAR should not appear without explicit passthrough")
	}
}

func TestBuildEnvironmentMissingPassthroughVar(t *testing.T) {
	a := NewClaudeAdapter()

	// Request passthrough of a var that doesn't exist
	cfg := AdapterRunConfig{
		EnvPassthrough: []string{"NONEXISTENT_VAR"},
	}

	env := a.buildEnvironment(cfg)
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Should not appear since it doesn't exist in host env
	if _, ok := envMap["NONEXISTENT_VAR"]; ok {
		t.Error("NONEXISTENT_VAR should not appear when not set in host env")
	}
}

func TestSettingsJSONPerPersona(t *testing.T) {
	// Table-driven test: verify settings.json for different persona profiles
	tests := []struct {
		name           string
		persona        string
		allowedTools   []string
		denyTools      []string
		allowedDomains []string
		wantAllow      []string
		wantDeny       []string
		wantSandbox    bool
	}{
		{
			name:         "navigator: read-only with deny",
			persona:      "navigator",
			allowedTools: []string{"Read", "Glob", "Grep", "Bash(git log*)"},
			denyTools:    []string{"Write(*)", "Edit(*)"},
			wantAllow:    []string{"Read", "Glob", "Grep", "Bash(git log*)"},
			wantDeny:     []string{"Write(*)", "Edit(*)"},
			wantSandbox:  false,
		},
		{
			name:           "implementer: full access with sandbox",
			persona:        "implementer",
			allowedTools:   []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
			allowedDomains: []string{"api.anthropic.com", "github.com", "proxy.golang.org"},
			wantAllow:      []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
			wantSandbox:    true,
		},
		{
			name:           "reviewer: read-only with network",
			persona:        "reviewer",
			allowedTools:   []string{"Read", "Glob", "Grep"},
			denyTools:      []string{"Write(*)", "Edit(*)", "Bash(*)"},
			allowedDomains: []string{"api.anthropic.com"},
			wantAllow:      []string{"Read", "Glob", "Grep"},
			wantDeny:       []string{"Write(*)", "Edit(*)", "Bash(*)"},
			wantSandbox:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewClaudeAdapter()
			tmpDir := t.TempDir()

			cfg := AdapterRunConfig{
				Persona:        tt.persona,
				Model:          "opus",
				AllowedTools:   tt.allowedTools,
				DenyTools:      tt.denyTools,
				AllowedDomains: tt.allowedDomains,
			}

			if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
				t.Fatalf("prepareWorkspace failed: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(tmpDir, ".claude", "settings.json"))
			if err != nil {
				t.Fatalf("failed to read settings.json: %v", err)
			}

			var settings ClaudeSettings
			if err := json.Unmarshal(data, &settings); err != nil {
				t.Fatalf("failed to parse settings.json: %v", err)
			}

			// Verify allow
			if len(settings.Permissions.Allow) != len(tt.wantAllow) {
				t.Fatalf("allow = %v, want %v", settings.Permissions.Allow, tt.wantAllow)
			}
			for i, want := range tt.wantAllow {
				if settings.Permissions.Allow[i] != want {
					t.Errorf("allow[%d] = %q, want %q", i, settings.Permissions.Allow[i], want)
				}
			}

			// Verify deny
			if len(tt.wantDeny) > 0 {
				if len(settings.Permissions.Deny) != len(tt.wantDeny) {
					t.Fatalf("deny = %v, want %v", settings.Permissions.Deny, tt.wantDeny)
				}
				for i, want := range tt.wantDeny {
					if settings.Permissions.Deny[i] != want {
						t.Errorf("deny[%d] = %q, want %q", i, settings.Permissions.Deny[i], want)
					}
				}
			}

			// Verify sandbox
			if tt.wantSandbox && settings.Sandbox == nil {
				t.Error("expected sandbox settings, got nil")
			}
			if !tt.wantSandbox && settings.Sandbox != nil {
				t.Errorf("expected no sandbox, got %+v", settings.Sandbox)
			}
		})
	}
}

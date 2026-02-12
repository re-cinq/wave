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
		sandboxEnabled bool
		allowedDomains []string
		wantSandbox    bool
		wantDomains    []string
	}{
		{
			name:           "sandbox enabled with domains",
			sandboxEnabled: true,
			allowedDomains: []string{"api.anthropic.com", "github.com", "*.github.com"},
			wantSandbox:    true,
			wantDomains:    []string{"api.anthropic.com", "github.com", "*.github.com"},
		},
		{
			name:           "no sandbox when not enabled",
			sandboxEnabled: false,
			allowedDomains: nil,
			wantSandbox:    false,
		},
		{
			name:           "sandbox enabled without domains",
			sandboxEnabled: true,
			allowedDomains: nil,
			wantSandbox:    true,
		},
		{
			name:           "single domain",
			sandboxEnabled: true,
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
				SandboxEnabled: tt.sandboxEnabled,
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
				if len(tt.wantDomains) > 0 {
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
					if settings.Sandbox.Network != nil {
						t.Errorf("expected no network settings, got %+v", settings.Sandbox.Network)
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
		sandboxEnabled bool
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
			sandboxEnabled: true,
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
			sandboxEnabled: true,
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
				SandboxEnabled: tt.sandboxEnabled,
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

func TestExtractToolTarget(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    json.RawMessage
		want     string
	}{
		// Explicit tools
		{
			name:     "Read extracts file_path",
			toolName: "Read",
			input:    json.RawMessage(`{"file_path": "/home/user/main.go"}`),
			want:     "/home/user/main.go",
		},
		{
			name:     "Write extracts file_path",
			toolName: "Write",
			input:    json.RawMessage(`{"file_path": "/tmp/output.txt", "content": "hello"}`),
			want:     "/tmp/output.txt",
		},
		{
			name:     "Edit extracts file_path",
			toolName: "Edit",
			input:    json.RawMessage(`{"file_path": "/src/lib.go", "old_string": "a", "new_string": "b"}`),
			want:     "/src/lib.go",
		},
		{
			name:     "Glob extracts pattern",
			toolName: "Glob",
			input:    json.RawMessage(`{"pattern": "**/*.go"}`),
			want:     "**/*.go",
		},
		{
			name:     "Grep extracts pattern",
			toolName: "Grep",
			input:    json.RawMessage(`{"pattern": "func main", "path": "/src"}`),
			want:     "func main",
		},
		{
			name:     "Bash extracts command",
			toolName: "Bash",
			input:    json.RawMessage(`{"command": "go test ./..."}`),
			want:     "go test ./...",
		},
		{
			name:     "Task extracts description",
			toolName: "Task",
			input:    json.RawMessage(`{"description": "Find all TODO comments"}`),
			want:     "Find all TODO comments",
		},
		{
			name:     "WebFetch extracts url",
			toolName: "WebFetch",
			input:    json.RawMessage(`{"url": "https://example.com/api", "prompt": "summarize"}`),
			want:     "https://example.com/api",
		},
		{
			name:     "WebSearch extracts query",
			toolName: "WebSearch",
			input:    json.RawMessage(`{"query": "Go error handling best practices"}`),
			want:     "Go error handling best practices",
		},
		{
			name:     "NotebookEdit extracts notebook_path",
			toolName: "NotebookEdit",
			input:    json.RawMessage(`{"notebook_path": "/notebooks/analysis.ipynb", "cell_type": "code", "new_source": "print(1)"}`),
			want:     "/notebooks/analysis.ipynb",
		},
		// Bash truncation (at 200 chars, display layer truncates further based on terminal width)
		{
			name:     "Bash command under 200 chars is not truncated",
			toolName: "Bash",
			input:    json.RawMessage(`{"command": "find /very/long/path -name '*.go' -exec grep -l 'something' {} \\; | sort | uniq"}`),
			want:     `find /very/long/path -name '*.go' -exec grep -l 'something' {} \; | sort | uniq`,
		},
		// Generic heuristic: unknown tool with common fields
		{
			name:     "heuristic: unknown tool with file_path",
			toolName: "CustomTool",
			input:    json.RawMessage(`{"file_path": "/custom/file.txt"}`),
			want:     "/custom/file.txt",
		},
		{
			name:     "heuristic: unknown tool with url",
			toolName: "SomeWebTool",
			input:    json.RawMessage(`{"url": "https://api.example.com"}`),
			want:     "https://api.example.com",
		},
		{
			name:     "heuristic: unknown tool with query",
			toolName: "SearchTool",
			input:    json.RawMessage(`{"query": "latest news"}`),
			want:     "latest news",
		},
		{
			name:     "heuristic: file_path takes priority over url",
			toolName: "UnknownTool",
			input:    json.RawMessage(`{"url": "https://example.com", "file_path": "/priority/file.go"}`),
			want:     "/priority/file.go",
		},
		// Edge cases
		{
			name:     "unknown tool with no matching fields returns empty",
			toolName: "UnknownTool",
			input:    json.RawMessage(`{"foo": "bar", "baz": 42}`),
			want:     "",
		},
		{
			name:     "nil input returns empty without panic",
			toolName: "Read",
			input:    json.RawMessage(nil),
			want:     "",
		},
		{
			name:     "empty JSON object returns empty",
			toolName: "Read",
			input:    json.RawMessage(`{}`),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolTarget(tt.toolName, tt.input)
			if got != tt.want {
				t.Errorf("extractToolTarget(%q, %s) = %q, want %q", tt.toolName, string(tt.input), got, tt.want)
			}
		})
	}
}

func TestExtractTodoSummary(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
		want  string
	}{
		{
			name:  "returns first in_progress task content",
			input: json.RawMessage(`[{"content":"Writing tests","status":"in_progress"},{"content":"Review code","status":"pending"}]`),
			want:  "Writing tests",
		},
		{
			name:  "returns first in_progress when multiple in_progress exist",
			input: json.RawMessage(`[{"content":"Task A","status":"completed"},{"content":"Task B","status":"in_progress"},{"content":"Task C","status":"in_progress"}]`),
			want:  "Task B",
		},
		{
			name:  "returns done/total when no in_progress task",
			input: json.RawMessage(`[{"content":"Task A","status":"completed"},{"content":"Task B","status":"pending"},{"content":"Task C","status":"completed"}]`),
			want:  "2/3 tasks",
		},
		{
			name:  "returns 0/N when all pending",
			input: json.RawMessage(`[{"content":"Task A","status":"pending"},{"content":"Task B","status":"pending"}]`),
			want:  "0/2 tasks",
		},
		{
			name:  "returns N/N when all completed",
			input: json.RawMessage(`[{"content":"Task A","status":"completed"},{"content":"Task B","status":"completed"}]`),
			want:  "2/2 tasks",
		},
		{
			name:  "single completed task",
			input: json.RawMessage(`[{"content":"Only task","status":"completed"}]`),
			want:  "1/1 tasks",
		},
		{
			name:  "single in_progress task",
			input: json.RawMessage(`[{"content":"Doing stuff","status":"in_progress"}]`),
			want:  "Doing stuff",
		},
		{
			name:  "empty array returns empty string",
			input: json.RawMessage(`[]`),
			want:  "",
		},
		{
			name:  "nil input returns empty string",
			input: nil,
			want:  "",
		},
		{
			name:  "malformed JSON returns empty string",
			input: json.RawMessage(`not json`),
			want:  "",
		},
		{
			name:  "JSON object instead of array returns empty string",
			input: json.RawMessage(`{"content":"task","status":"in_progress"}`),
			want:  "",
		},
		{
			name:  "empty JSON object returns empty string",
			input: json.RawMessage(`{}`),
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTodoSummary(tt.input)
			if got != tt.want {
				t.Errorf("extractTodoSummary(%s) = %q, want %q", string(tt.input), got, tt.want)
			}
		})
	}
}

// T002: TestParseStreamLine — table-driven tests for all event types
func TestParseStreamLine(t *testing.T) {
	// Build a 1MB+ string for the extremely long line test
	longText := strings.Repeat("A", 1024*1024+1)
	longLineJSON := `{"type":"assistant","message":{"content":[{"type":"text","text":"` + longText + `"}],"usage":{}}}`

	tests := []struct {
		name      string
		line      []byte
		wantOK    bool
		wantEvent StreamEvent
	}{
		{
			name:   "system type",
			line:   []byte(`{"type":"system","subtype":"init"}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type: "system",
			},
		},
		{
			name:   "assistant with tool_use",
			line:   []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/foo.go"}}],"usage":{"input_tokens":100,"output_tokens":50}}}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:      "tool_use",
				ToolName:  "Read",
				ToolInput: "/tmp/foo.go",
				TokensIn:  100,
				TokensOut: 50,
			},
		},
		{
			name:   "assistant with text",
			line:   []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello world this is a test"}],"usage":{}}}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:    "text",
				Content: "Hello world this is a test",
			},
		},
		{
			name:   "tool_result type is skipped",
			line:   []byte(`{"type":"tool_result"}`),
			wantOK: false,
		},
		{
			name:   "result type with usage",
			line:   []byte(`{"type":"result","usage":{"input_tokens":1000,"output_tokens":500}}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:      "result",
				TokensIn:  1000,
				TokensOut: 500,
			},
		},
		{
			name:   "malformed JSON",
			line:   []byte(`not json at all`),
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   []byte(``),
			wantOK: false,
		},
		{
			name:   "unknown type",
			line:   []byte(`{"type":"unknown_type"}`),
			wantOK: false,
		},
		{
			name:   "extremely long line (1MB+)",
			line:   []byte(longLineJSON),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:    "text",
				Content: strings.Repeat("A", 200), // truncated to 200 chars at parse time
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseStreamLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("parseStreamLine() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got.Type != tt.wantEvent.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantEvent.Type)
			}
			if got.ToolName != tt.wantEvent.ToolName {
				t.Errorf("ToolName = %q, want %q", got.ToolName, tt.wantEvent.ToolName)
			}
			if got.ToolInput != tt.wantEvent.ToolInput {
				t.Errorf("ToolInput = %q, want %q", got.ToolInput, tt.wantEvent.ToolInput)
			}
			if got.Content != tt.wantEvent.Content {
				t.Errorf("Content = %q, want %q", got.Content, tt.wantEvent.Content)
			}
			if got.TokensIn != tt.wantEvent.TokensIn {
				t.Errorf("TokensIn = %d, want %d", got.TokensIn, tt.wantEvent.TokensIn)
			}
			if got.TokensOut != tt.wantEvent.TokensOut {
				t.Errorf("TokensOut = %d, want %d", got.TokensOut, tt.wantEvent.TokensOut)
			}
		})
	}
}

// T003: TestStreamEventCallback — verify OnStreamEvent invocation via parseStreamLine
func TestStreamEventCallback(t *testing.T) {
	lines := []struct {
		label    string
		data     []byte
		wantOK   bool
		wantType string
		toolName string
		toolIn   string
	}{
		{
			label:    "tool_use event returns true with correct fields",
			data:     []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}],"usage":{"input_tokens":200,"output_tokens":80}}}`),
			wantOK:   true,
			wantType: "tool_use",
			toolName: "Bash",
			toolIn:   "go test ./...",
		},
		{
			label:    "second tool_use event with Write",
			data:     []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/tmp/out.txt"}}],"usage":{"input_tokens":50,"output_tokens":20}}}`),
			wantOK:   true,
			wantType: "tool_use",
			toolName: "Write",
			toolIn:   "/tmp/out.txt",
		},
		{
			label:    "text event returns true with different Type",
			data:     []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Analyzing the code..."}],"usage":{}}}`),
			wantOK:   true,
			wantType: "text",
		},
		{
			label:  "tool_result event returns false",
			data:   []byte(`{"type":"tool_result"}`),
			wantOK: false,
		},
		{
			label:    "system event returns true with Type system",
			data:     []byte(`{"type":"system","subtype":"init"}`),
			wantOK:   true,
			wantType: "system",
		},
	}

	for _, tc := range lines {
		t.Run(tc.label, func(t *testing.T) {
			evt, ok := parseStreamLine(tc.data)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if evt.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", evt.Type, tc.wantType)
			}
			if tc.toolName != "" && evt.ToolName != tc.toolName {
				t.Errorf("ToolName = %q, want %q", evt.ToolName, tc.toolName)
			}
			if tc.toolIn != "" && evt.ToolInput != tc.toolIn {
				t.Errorf("ToolInput = %q, want %q", evt.ToolInput, tc.toolIn)
			}
		})
	}
}

// T004: TestResultAccumulation — verify result extraction from stream
func TestResultAccumulation(t *testing.T) {
	t.Run("result with standard token counts", func(t *testing.T) {
		line := []byte(`{"type":"result","usage":{"input_tokens":5000,"output_tokens":2000}}`)
		evt, ok := parseStreamLine(line)
		if !ok {
			t.Fatal("expected ok=true for result event")
		}
		if evt.TokensIn != 5000 {
			t.Errorf("TokensIn = %d, want 5000", evt.TokensIn)
		}
		if evt.TokensOut != 2000 {
			t.Errorf("TokensOut = %d, want 2000", evt.TokensOut)
		}
	})

	t.Run("result parses correctly after tool_use events", func(t *testing.T) {
		// Parse several tool_use events first — shouldn't affect result parsing
		toolLines := [][]byte{
			[]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/a.go"}}],"usage":{"input_tokens":10,"output_tokens":5}}}`),
			[]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Glob","input":{"pattern":"*.go"}}],"usage":{"input_tokens":20,"output_tokens":10}}}`),
		}
		for _, tl := range toolLines {
			_, ok := parseStreamLine(tl)
			if !ok {
				t.Fatal("expected ok=true for tool_use event")
			}
		}

		// Now parse the result event
		resultLine := []byte(`{"type":"result","usage":{"input_tokens":5000,"output_tokens":2000}}`)
		evt, ok := parseStreamLine(resultLine)
		if !ok {
			t.Fatal("expected ok=true for result event")
		}
		if evt.TokensIn != 5000 {
			t.Errorf("TokensIn = %d, want 5000", evt.TokensIn)
		}
		if evt.TokensOut != 2000 {
			t.Errorf("TokensOut = %d, want 2000", evt.TokensOut)
		}
	})

	t.Run("result with zero tokens", func(t *testing.T) {
		line := []byte(`{"type":"result","usage":{"input_tokens":0,"output_tokens":0}}`)
		evt, ok := parseStreamLine(line)
		if !ok {
			t.Fatal("expected ok=true for result event")
		}
		if evt.TokensIn != 0 {
			t.Errorf("TokensIn = %d, want 0", evt.TokensIn)
		}
		if evt.TokensOut != 0 {
			t.Errorf("TokensOut = %d, want 0", evt.TokensOut)
		}
	})
}

// T005: TestMidStreamTermination — verify no panic on incomplete stream
func TestMidStreamTermination(t *testing.T) {
	t.Run("only system event without result does not panic", func(t *testing.T) {
		line := []byte(`{"type":"system","subtype":"init"}`)
		evt, ok := parseStreamLine(line)
		if !ok {
			t.Fatal("expected ok=true for system event")
		}
		if evt.Type != "system" {
			t.Errorf("Type = %q, want %q", evt.Type, "system")
		}
		// No result follows — this should be fine
	})

	t.Run("only tool_use events without result does not panic", func(t *testing.T) {
		lines := [][]byte{
			[]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/a.go"}}],"usage":{"input_tokens":10,"output_tokens":5}}}`),
			[]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/b.go"}}],"usage":{"input_tokens":20,"output_tokens":10}}}`),
		}
		for _, line := range lines {
			evt, ok := parseStreamLine(line)
			if !ok {
				t.Fatal("expected ok=true for tool_use event")
			}
			if evt.Type != "tool_use" {
				t.Errorf("Type = %q, want %q", evt.Type, "tool_use")
			}
		}
	})

	t.Run("empty byte slice does not panic", func(t *testing.T) {
		_, ok := parseStreamLine([]byte{})
		if ok {
			t.Error("expected ok=false for empty byte slice")
		}
	})

	t.Run("nil does not panic", func(t *testing.T) {
		_, ok := parseStreamLine(nil)
		if ok {
			t.Error("expected ok=false for nil input")
		}
	})
}

func TestSkillCommandsCopied(t *testing.T) {
	adapter := NewClaudeAdapter()
	workspace := t.TempDir()

	// Create source skill commands directory
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "speckit.specify.md"), []byte("# Specify"), 0644)
	os.WriteFile(filepath.Join(srcDir, "speckit.plan.md"), []byte("# Plan"), 0644)
	os.WriteFile(filepath.Join(srcDir, "not-markdown.txt"), []byte("ignored"), 0644)

	cfg := AdapterRunConfig{
		Persona:          "implementer",
		AllowedTools:     []string{"Read", "Write"},
		SkillCommandsDir: srcDir,
	}

	if err := adapter.prepareWorkspace(workspace, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	// Verify skill commands were copied
	commandsDir := filepath.Join(workspace, ".claude", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}

	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}

	if len(mdFiles) != 2 {
		t.Fatalf("expected 2 md files, got %d: %v", len(mdFiles), mdFiles)
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(commandsDir, "speckit.specify.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Specify" {
		t.Errorf("unexpected content: %s", string(data))
	}

	// Verify non-md files were NOT copied
	if _, err := os.Stat(filepath.Join(commandsDir, "not-markdown.txt")); !os.IsNotExist(err) {
		t.Error("non-markdown file should not be copied")
	}
}

func TestSkillCommandsDir_Empty(t *testing.T) {
	adapter := NewClaudeAdapter()
	workspace := t.TempDir()

	// No SkillCommandsDir set - should work fine
	cfg := AdapterRunConfig{
		Persona:      "implementer",
		AllowedTools: []string{"Read", "Write"},
	}

	if err := adapter.prepareWorkspace(workspace, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	// .claude/commands/ should NOT exist (no skill commands to copy)
	commandsDir := filepath.Join(workspace, ".claude", "commands")
	if _, err := os.Stat(commandsDir); !os.IsNotExist(err) {
		t.Error("commands dir should not exist when SkillCommandsDir is empty")
	}
}

func TestSkillCommandsDir_NonExistent(t *testing.T) {
	adapter := NewClaudeAdapter()
	workspace := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:          "implementer",
		AllowedTools:     []string{"Read", "Write"},
		SkillCommandsDir: "/nonexistent/path",
	}

	// Should not error - silently skip non-existent source
	if err := adapter.prepareWorkspace(workspace, cfg); err != nil {
		t.Fatalf("prepareWorkspace should not fail for non-existent source: %v", err)
	}
}

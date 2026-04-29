package adapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testBaseProtocol = `# Wave Agent Protocol

You are operating within a Wave pipeline step.

## Operational Context

- **Fresh context**: You have no memory of prior steps. Each step starts clean.
- **Artifact I/O**: Read inputs from injected artifacts. Write outputs to artifact files.
- **Workspace isolation**: You are in an ephemeral worktree. Changes here do not affect the source repository directly.
- **Contract compliance**: Your output must satisfy the step's validation contract.
- **Permission enforcement**: Tool permissions are enforced by the orchestrator. Do not attempt to bypass restrictions listed below.
`

// setupBaseProtocol creates .agents/personas/base-protocol.md relative to the
// current working directory so that prepareWorkspace can find it. Returns a
// cleanup function that removes the created directory structure.
func setupBaseProtocol(t *testing.T) {
	t.Helper()
	dir := filepath.Join(".agents", "personas")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create .agents/personas: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base-protocol.md"), []byte(testBaseProtocol), 0644); err != nil {
		t.Fatalf("failed to write base-protocol.md: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(".agents")
	})
}

func TestNoSettingsJSONWhenSandboxDisabled(t *testing.T) {
	setupBaseProtocol(t)
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:       "test",
		WorkspacePath: tmpDir,
		Model:         "sonnet",
		AllowedTools:  []string{"Read", "Write(.agents/output/*)", "Glob"},
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("settings.json should not be created when sandbox is disabled")
	}
}

// --- Phase 2 tests: deny rules, sandbox settings, agent file restrictions, env hygiene ---

func TestDenyRulesInAgentFrontmatter(t *testing.T) {
	setupBaseProtocol(t)
	tests := []struct {
		name     string
		deny     []string
		wantDeny []string
	}{
		{
			name: "deny rules appear in agent frontmatter",
			deny: []string{"Write(*)", "Edit(*)", "Bash(rm *)"},
			// TodoWrite is auto-injected by prepareWorkspace
			wantDeny: []string{"Write(*)", "Edit(*)", "Bash(rm *)", "TodoWrite"},
		},
		{
			name: "empty deny gets only TodoWrite",
			deny: nil,
			// TodoWrite is auto-injected
			wantDeny: []string{"TodoWrite"},
		},
		{
			name:     "single deny rule plus auto TodoWrite",
			deny:     []string{"Bash(rm -rf /*)"},
			wantDeny: []string{"Bash(rm -rf /*)", "TodoWrite"},
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

			data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
			if err != nil {
				t.Fatalf("failed to read agent file: %v", err)
			}
			content := string(data)

			for _, want := range tt.wantDeny {
				if !strings.Contains(content, "  - "+want+"\n") {
					t.Errorf("agent frontmatter missing deny rule %q\nGot:\n%s", want, content)
				}
			}
		})
	}
}

func TestSettingsJSONSandboxSettings(t *testing.T) {
	setupBaseProtocol(t)
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
			name:           "no settings.json when sandbox disabled",
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

			settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")

			if !tt.wantSandbox {
				if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
					t.Error("settings.json should not exist when sandbox is disabled")
				}
				return
			}

			data, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("failed to read settings.json: %v", err)
			}

			var settings SandboxOnlySettings
			if err := json.Unmarshal(data, &settings); err != nil {
				t.Fatalf("failed to parse settings.json: %v", err)
			}

			// Verify settings.json contains ONLY sandbox field
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to parse settings.json as raw map: %v", err)
			}
			for key := range raw {
				if key != "sandbox" {
					t.Errorf("settings.json should only contain 'sandbox' field, found %q", key)
				}
			}

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
			} else if settings.Sandbox.Network != nil {
				t.Errorf("expected no network settings, got %+v", settings.Sandbox.Network)
			}
		})
	}
}

func TestAgentFileRestrictionSection(t *testing.T) {
	setupBaseProtocol(t)
	tests := []struct {
		name         string
		cfg          AdapterRunConfig
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "deny rules appear in agent file",
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
			name: "network domains appear in agent file",
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
			// TodoWrite is always auto-injected into DenyTools, so the
			// restriction section will contain "Denied Tools" with TodoWrite.
			wantContains: []string{
				"## Restrictions",
				"### Denied Tools",
				"`TodoWrite`",
			},
			wantAbsent: []string{
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

			data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
			if err != nil {
				t.Fatalf("failed to read agent file: %v", err)
			}
			content := string(data)

			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("agent file missing %q\nGot:\n%s", want, content)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(content, absent) {
					t.Errorf("agent file should not contain %q\nGot:\n%s", absent, content)
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

func TestBuildConcurrencyHint(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{
			name: "zero produces no hint",
			n:    0,
			want: "",
		},
		{
			name: "one produces no hint",
			n:    1,
			want: "",
		},
		{
			name: "three produces hint with 3",
			n:    3,
			want: "\n\n## Agent Concurrency\n\nYou may spawn up to 3 concurrent sub-agents or workers for this step.\n",
		},
		{
			name: "ten produces hint with 10",
			n:    10,
			want: "\n\n## Agent Concurrency\n\nYou may spawn up to 10 concurrent sub-agents or workers for this step.\n",
		},
		{
			name: "fifteen capped at 10",
			n:    15,
			want: "\n\n## Agent Concurrency\n\nYou may spawn up to 10 concurrent sub-agents or workers for this step.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConcurrencyHint(tt.n)
			if got != tt.want {
				t.Errorf("buildConcurrencyHint(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestConcurrencyHintInAgentFile(t *testing.T) {
	setupBaseProtocol(t)
	tests := []struct {
		name         string
		cfg          AdapterRunConfig
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "no hint when MaxConcurrentAgents is 0",
			cfg: AdapterRunConfig{
				Persona:      "test",
				Model:        "sonnet",
				SystemPrompt: "# Test",
			},
			wantAbsent: []string{"Agent Concurrency", "concurrent sub-agents"},
		},
		{
			name: "no hint when MaxConcurrentAgents is 1",
			cfg: AdapterRunConfig{
				Persona:             "test",
				Model:               "sonnet",
				SystemPrompt:        "# Test",
				MaxConcurrentAgents: 1,
			},
			wantAbsent: []string{"Agent Concurrency", "concurrent sub-agents"},
		},
		{
			name: "hint present when MaxConcurrentAgents is 3",
			cfg: AdapterRunConfig{
				Persona:             "test",
				Model:               "sonnet",
				SystemPrompt:        "# Test",
				MaxConcurrentAgents: 3,
			},
			wantContains: []string{
				"## Agent Concurrency",
				"You may spawn up to 3 concurrent sub-agents or workers for this step.",
			},
		},
		{
			name: "hint capped at 10 when MaxConcurrentAgents is 15",
			cfg: AdapterRunConfig{
				Persona:             "test",
				Model:               "sonnet",
				SystemPrompt:        "# Test",
				MaxConcurrentAgents: 15,
			},
			wantContains: []string{
				"You may spawn up to 10 concurrent sub-agents or workers for this step.",
			},
			wantAbsent: []string{
				"up to 15",
			},
		},
		{
			name: "hint appears before restrictions",
			cfg: AdapterRunConfig{
				Persona:             "test",
				Model:               "sonnet",
				SystemPrompt:        "# Test",
				MaxConcurrentAgents: 4,
				AllowedTools:        []string{"Read", "Bash"},
			},
			wantContains: []string{
				"Agent Concurrency",
				"## Restrictions",
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

			data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
			if err != nil {
				t.Fatalf("failed to read agent file: %v", err)
			}
			content := string(data)

			for _, want := range tt.wantContains {
				if !strings.Contains(content, want) {
					t.Errorf("agent file missing %q\nGot:\n%s", want, content)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(content, absent) {
					t.Errorf("agent file should not contain %q\nGot:\n%s", absent, content)
				}
			}
		})
	}
}

func TestConcurrencyHintOrdering(t *testing.T) {
	setupBaseProtocol(t)
	a := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:             "test",
		Model:               "sonnet",
		SystemPrompt:        "# Test Persona",
		MaxConcurrentAgents: 6,
		AllowedTools:        []string{"Read", "Write"},
		DenyTools:           []string{"Bash(rm *)"},
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
	if err != nil {
		t.Fatalf("failed to read agent file: %v", err)
	}
	content := string(data)

	concurrencyIdx := strings.Index(content, "## Agent Concurrency")
	restrictionIdx := strings.Index(content, "## Restrictions")

	if concurrencyIdx == -1 {
		t.Fatal("missing Agent Concurrency section")
	}
	if restrictionIdx == -1 {
		t.Fatal("missing Restrictions section")
	}

	if concurrencyIdx > restrictionIdx {
		t.Errorf("Agent Concurrency (pos %d) should appear before Restrictions (pos %d)", concurrencyIdx, restrictionIdx)
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

func TestAgentFilePerPersona(t *testing.T) {
	setupBaseProtocol(t)
	// Table-driven test: verify agent file generated per persona config
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

			// Verify agent file contains correct tools in frontmatter
			data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
			if err != nil {
				t.Fatalf("failed to read agent file: %v", err)
			}
			content := string(data)

			// Verify allowed tools
			for _, tool := range tt.wantAllow {
				if !strings.Contains(content, "  - "+tool+"\n") {
					t.Errorf("agent frontmatter missing allowed tool %q", tool)
				}
			}

			// Verify deny tools
			for _, tool := range tt.wantDeny {
				if !strings.Contains(content, "  - "+tool+"\n") {
					t.Errorf("agent frontmatter missing deny tool %q", tool)
				}
			}

			// Verify model
			if !strings.Contains(content, "model: opus\n") {
				t.Error("agent frontmatter missing model")
			}

			// Verify sandbox settings.json
			settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
			if tt.wantSandbox {
				if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
					t.Error("expected settings.json for sandbox, but not found")
				}
			} else {
				if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
					t.Error("settings.json should not exist when sandbox is disabled")
				}
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
			name:   "result type with subtype success",
			line:   []byte(`{"type":"result","subtype":"success","usage":{"input_tokens":2000,"output_tokens":800}}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:      "result",
				TokensIn:  2000,
				TokensOut: 800,
				Subtype:   "success",
			},
		},
		{
			name:   "result type with subtype error_max_turns",
			line:   []byte(`{"type":"result","subtype":"error_max_turns","usage":{"input_tokens":180000,"output_tokens":20000}}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:      "result",
				TokensIn:  180000,
				TokensOut: 20000,
				Subtype:   "error_max_turns",
			},
		},
		{
			name:   "result type with subtype error_during_execution",
			line:   []byte(`{"type":"result","subtype":"error_during_execution","usage":{"input_tokens":500,"output_tokens":100}}`),
			wantOK: true,
			wantEvent: StreamEvent{
				Type:      "result",
				TokensIn:  500,
				TokensOut: 100,
				Subtype:   "error_during_execution",
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
			if got.Subtype != tt.wantEvent.Subtype {
				t.Errorf("Subtype = %q, want %q", got.Subtype, tt.wantEvent.Subtype)
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
	setupBaseProtocol(t)
	adapter := NewClaudeAdapter()
	workspace := t.TempDir()

	// Create source skill commands directory
	srcDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(srcDir, "speckit.specify.md"), []byte("# Specify"), 0644)
	_ = os.WriteFile(filepath.Join(srcDir, "speckit.plan.md"), []byte("# Plan"), 0644)
	_ = os.WriteFile(filepath.Join(srcDir, "not-markdown.txt"), []byte("ignored"), 0644)

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
	setupBaseProtocol(t)
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
	setupBaseProtocol(t)
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

// TestParseOutputSubtype verifies that parseOutput extracts the subtype field
// from NDJSON result events for error classification.
func TestParseOutputSubtype(t *testing.T) {
	adapter := NewClaudeAdapter()

	tests := []struct {
		name        string
		data        string
		wantSubtype string
		wantTokens  int
	}{
		{
			name:        "result with success subtype",
			data:        `{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":1000,"output_tokens":500}}` + "\n",
			wantSubtype: "success",
			wantTokens:  1500,
		},
		{
			name:        "result with error_max_turns subtype",
			data:        `{"type":"result","subtype":"error_max_turns","result":"exceeded max turns","usage":{"input_tokens":180000,"output_tokens":20000}}` + "\n",
			wantSubtype: "error_max_turns",
			wantTokens:  200000,
		},
		{
			name:        "result with error_during_execution subtype",
			data:        `{"type":"result","subtype":"error_during_execution","result":"prompt is too long","usage":{"input_tokens":5000,"output_tokens":100}}` + "\n",
			wantSubtype: "error_during_execution",
			wantTokens:  5100,
		},
		{
			name:        "result without subtype field",
			data:        `{"type":"result","result":"done","usage":{"input_tokens":1000,"output_tokens":500}}` + "\n",
			wantSubtype: "",
			wantTokens:  1500,
		},
		{
			name:        "result excludes cache_read_input_tokens from total",
			data:        `{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":5000,"output_tokens":2000,"cache_read_input_tokens":1500000,"cache_creation_input_tokens":50000}}` + "\n",
			wantSubtype: "success",
			wantTokens:  57000, // 5000 + 2000 + 50000 (cache_read excluded)
		},
		{
			name:        "no result event falls back to assistant tokens",
			data:        `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":100,"output_tokens":50}}}` + "\n",
			wantSubtype: "",
			wantTokens:  150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := adapter.parseOutput([]byte(tt.data))
			if parsed.Subtype != tt.wantSubtype {
				t.Errorf("Subtype = %q, want %q", parsed.Subtype, tt.wantSubtype)
			}
			if parsed.Tokens != tt.wantTokens {
				t.Errorf("Tokens = %d, want %d", parsed.Tokens, tt.wantTokens)
			}
		})
	}
}

// T004: Base protocol is prepended before persona content in agent file
func TestBaseProtocolInAgentFile(t *testing.T) {
	setupBaseProtocol(t)
	a := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:      "test-persona",
		Model:        "sonnet",
		SystemPrompt: "# Test Persona\n\nYou are a test persona.",
		AllowedTools: []string{"Read"},
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
	if err != nil {
		t.Fatalf("failed to read agent file: %v", err)
	}
	content := string(data)

	// Base protocol heading must appear before persona content
	protocolIdx := strings.Index(content, "# Wave Agent Protocol")
	personaIdx := strings.Index(content, "# Test Persona")
	separatorIdx := strings.Index(content, "\n\n---\n\n")

	if protocolIdx == -1 {
		t.Fatal("agent file missing base protocol heading '# Wave Agent Protocol'")
	}
	if personaIdx == -1 {
		t.Fatal("agent file missing persona heading '# Test Persona'")
	}
	if separatorIdx == -1 {
		t.Fatal("agent file missing '---' separator between base protocol and persona")
	}
	if protocolIdx >= separatorIdx {
		t.Error("base protocol heading should appear before the separator")
	}
	if separatorIdx >= personaIdx {
		t.Error("separator should appear before persona content")
	}

	// Verify key base protocol content is present
	for _, want := range []string{
		"Fresh context",
		"Artifact I/O",
		"Workspace isolation",
		"Contract compliance",
		"Permission enforcement",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("agent file missing base protocol element %q", want)
		}
	}
}

// T005: prepareWorkspace returns error when base-protocol.md is missing
func TestBaseProtocolMissingError(t *testing.T) {
	// Do NOT call setupBaseProtocol — intentionally missing
	a := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:      "test",
		Model:        "sonnet",
		SystemPrompt: "# Test",
		AllowedTools: []string{"Read"},
	}

	err := a.prepareWorkspace(tmpDir, cfg)
	if err == nil {
		t.Fatal("expected error when base-protocol.md is missing, got nil")
	}
	if !strings.Contains(err.Error(), "base protocol") {
		t.Errorf("error should mention 'base protocol', got: %v", err)
	}
}

// T006: Base protocol is prepended even when SystemPrompt is set directly
func TestBaseProtocolWithInlinePromptInAgentFile(t *testing.T) {
	setupBaseProtocol(t)
	a := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:      "inline-test",
		Model:        "sonnet",
		SystemPrompt: "# Inline Persona\n\nDo something specific.",
		AllowedTools: []string{"Read", "Write"},
	}

	if err := a.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
	if err != nil {
		t.Fatalf("failed to read agent file: %v", err)
	}
	content := string(data)

	// Base protocol must still be present even with inline prompt
	if !strings.Contains(content, "# Wave Agent Protocol") {
		t.Error("agent file missing base protocol when using SystemPrompt")
	}
	if !strings.Contains(content, "# Inline Persona") {
		t.Error("agent file missing inline persona content")
	}
	// Verify ordering
	if strings.Index(content, "# Wave Agent Protocol") > strings.Index(content, "# Inline Persona") {
		t.Error("base protocol should appear before inline persona content")
	}
}

// Task 1.2: Token parsing with cache token fields
func TestParseStreamLine_CacheTokenExclusion(t *testing.T) {
	tests := []struct {
		name    string
		line    []byte
		wantIn  int
		wantOut int
	}{
		{
			name:    "result excludes cache_read_input_tokens",
			line:    []byte(`{"type":"result","usage":{"input_tokens":5000,"output_tokens":2000,"cache_read_input_tokens":1500000,"cache_creation_input_tokens":50000}}`),
			wantIn:  55000, // 5000 + 50000 (cache_read excluded)
			wantOut: 2000,
		},
		{
			name:    "result with cache_creation only",
			line:    []byte(`{"type":"result","usage":{"input_tokens":3000,"output_tokens":1000,"cache_creation_input_tokens":20000}}`),
			wantIn:  23000, // 3000 + 20000
			wantOut: 1000,
		},
		{
			name:    "result with cache_read only (excluded)",
			line:    []byte(`{"type":"result","usage":{"input_tokens":3000,"output_tokens":1000,"cache_read_input_tokens":500000}}`),
			wantIn:  3000, // cache_read excluded
			wantOut: 1000,
		},
		{
			name:    "result with no cache tokens",
			line:    []byte(`{"type":"result","usage":{"input_tokens":1000,"output_tokens":500}}`),
			wantIn:  1000,
			wantOut: 500,
		},
		{
			name:    "result with both cache fields zero",
			line:    []byte(`{"type":"result","usage":{"input_tokens":2000,"output_tokens":800,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}`),
			wantIn:  2000,
			wantOut: 800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, ok := parseStreamLine(tt.line)
			if !ok {
				t.Fatal("expected ok=true for result event")
			}
			if evt.Type != "result" {
				t.Errorf("Type = %q, want %q", evt.Type, "result")
			}
			if evt.TokensIn != tt.wantIn {
				t.Errorf("TokensIn = %d, want %d", evt.TokensIn, tt.wantIn)
			}
			if evt.TokensOut != tt.wantOut {
				t.Errorf("TokensOut = %d, want %d", evt.TokensOut, tt.wantOut)
			}
		})
	}
}

// Task 1.2: Verify parseStreamLine and parseOutput agree on cache token handling
func TestParseOutputAndStreamLineConsistency(t *testing.T) {
	// Same NDJSON payload parsed by both methods should produce the same total
	data := []byte(`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":5000,"output_tokens":2000,"cache_read_input_tokens":1500000,"cache_creation_input_tokens":50000}}` + "\n")

	adapter := NewClaudeAdapter()
	parsed := adapter.parseOutput(data)

	evt, ok := parseStreamLine(data[:len(data)-1]) // strip trailing newline for parseStreamLine
	if !ok {
		t.Fatal("parseStreamLine should return ok=true for result event")
	}

	streamTotal := evt.TokensIn + evt.TokensOut
	if parsed.Tokens != streamTotal {
		t.Errorf("parseOutput tokens (%d) != parseStreamLine total (%d); cache_read handling inconsistent",
			parsed.Tokens, streamTotal)
	}
}

// Task 1.3: Token fallback chain tests
func TestParseOutputFallbackChain(t *testing.T) {
	adapter := NewClaudeAdapter()

	t.Run("prefers result tokens over assistant tokens", func(t *testing.T) {
		data := []byte(
			`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":100,"output_tokens":50}}}` + "\n" +
				`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":5000,"output_tokens":2000}}` + "\n",
		)
		parsed := adapter.parseOutput(data)
		if parsed.Tokens != 7000 {
			t.Errorf("Tokens = %d, want 7000 (result tokens preferred)", parsed.Tokens)
		}
	})

	t.Run("falls back to assistant tokens when result is zero", func(t *testing.T) {
		data := []byte(
			`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":100,"output_tokens":50}}}` + "\n" +
				`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":0,"output_tokens":0}}` + "\n",
		)
		parsed := adapter.parseOutput(data)
		if parsed.Tokens != 150 {
			t.Errorf("Tokens = %d, want 150 (assistant fallback)", parsed.Tokens)
		}
	})

	t.Run("falls back to byte estimate when both are zero", func(t *testing.T) {
		// A result event with all-zero tokens and no assistant event
		data := []byte(`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":0,"output_tokens":0}}` + "\n")
		parsed := adapter.parseOutput(data)
		expected := len(data) / 4
		if parsed.Tokens != expected {
			t.Errorf("Tokens = %d, want %d (byte estimate fallback)", parsed.Tokens, expected)
		}
	})

	t.Run("falls back to byte estimate with no events", func(t *testing.T) {
		data := []byte("some raw output with no JSON\n")
		parsed := adapter.parseOutput(data)
		expected := len(data) / 4
		if parsed.Tokens != expected {
			t.Errorf("Tokens = %d, want %d (byte estimate for non-JSON)", parsed.Tokens, expected)
		}
	})
}

func TestParseOutput_ReturnsTokensInOut(t *testing.T) {
	a := &ClaudeAdapter{}
	// result event with input/output token breakdown
	data := []byte(`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":5000,"output_tokens":2000,"cache_creation_input_tokens":1000,"cache_read_input_tokens":500}}` + "\n")
	parsed := a.parseOutput(data)

	// TokensIn should be input_tokens + cache_creation_input_tokens = 5000 + 1000 = 6000
	if parsed.TokensIn != 6000 {
		t.Errorf("TokensIn = %d, want %d", parsed.TokensIn, 6000)
	}
	// TokensOut should be output_tokens = 2000
	if parsed.TokensOut != 2000 {
		t.Errorf("TokensOut = %d, want %d", parsed.TokensOut, 2000)
	}
	// Total should be input_tokens + output_tokens + cache_creation = 5000 + 2000 + 1000 = 8000
	if parsed.Tokens != 8000 {
		t.Errorf("Tokens = %d, want %d", parsed.Tokens, 8000)
	}
}

func TestBuildSkillSection(t *testing.T) {
	t.Run("empty skills returns empty string", func(t *testing.T) {
		result := buildSkillSection(nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
		result = buildSkillSection([]SkillRef{})
		if result != "" {
			t.Errorf("expected empty string for empty slice, got %q", result)
		}
	})

	t.Run("single skill", func(t *testing.T) {
		result := buildSkillSection([]SkillRef{
			{Name: "golang", Description: "Expert Go development"},
		})
		if !strings.Contains(result, "## Available Skills") {
			t.Error("missing Available Skills heading")
		}
		if !strings.Contains(result, "**golang**") {
			t.Error("missing skill name")
		}
		if !strings.Contains(result, "Expert Go development") {
			t.Error("missing skill description")
		}
		if !strings.Contains(result, ".agents/skills/golang/SKILL.md") {
			t.Error("missing SKILL.md path")
		}
	})

	t.Run("multiple skills", func(t *testing.T) {
		result := buildSkillSection([]SkillRef{
			{Name: "golang", Description: "Go development"},
			{Name: "cli", Description: "CLI patterns"},
		})
		if !strings.Contains(result, "**golang**") {
			t.Error("missing first skill")
		}
		if !strings.Contains(result, "**cli**") {
			t.Error("missing second skill")
		}
	})
}

func TestSkillSectionInAgentFile(t *testing.T) {
	setupBaseProtocol(t)
	adapter := NewClaudeAdapter()
	workspace := t.TempDir()

	t.Run("skills appear in agent file when ResolvedSkills non-empty", func(t *testing.T) {
		cfg := AdapterRunConfig{
			Persona:      "craftsman",
			SystemPrompt: "# Craftsman\n\nYou are a craftsman.",
			ResolvedSkills: []SkillRef{
				{Name: "golang", Description: "Go development"},
				{Name: "cli", Description: "CLI patterns"},
			},
		}

		if err := adapter.prepareWorkspace(workspace, cfg); err != nil {
			t.Fatalf("prepareWorkspace failed: %v", err)
		}

		agentPath := filepath.Join(workspace, agentFilePath)
		data, err := os.ReadFile(agentPath)
		if err != nil {
			t.Fatalf("failed to read agent file: %v", err)
		}
		content := string(data)

		if !strings.Contains(content, "## Available Skills") {
			t.Error("agent file missing Available Skills section")
		}
		if !strings.Contains(content, "**golang**") {
			t.Error("agent file missing golang skill")
		}
		if !strings.Contains(content, "**cli**") {
			t.Error("agent file missing cli skill")
		}
	})

	t.Run("no skill section when ResolvedSkills empty", func(t *testing.T) {
		cfg := AdapterRunConfig{
			Persona:      "craftsman",
			SystemPrompt: "# Craftsman\n\nYou are a craftsman.",
		}

		if err := adapter.prepareWorkspace(workspace, cfg); err != nil {
			t.Fatalf("prepareWorkspace failed: %v", err)
		}

		agentPath := filepath.Join(workspace, agentFilePath)
		data, err := os.ReadFile(agentPath)
		if err != nil {
			t.Fatalf("failed to read agent file: %v", err)
		}
		content := string(data)

		if strings.Contains(content, "## Available Skills") {
			t.Error("agent file should not have Available Skills section when no skills")
		}
	})

	t.Run("skill section appears after persona", func(t *testing.T) {
		cfg := AdapterRunConfig{
			Persona:      "craftsman",
			SystemPrompt: "# Craftsman\n\nYou are a craftsman.",
			ResolvedSkills: []SkillRef{
				{Name: "golang", Description: "Go development"},
			},
		}

		if err := adapter.prepareWorkspace(workspace, cfg); err != nil {
			t.Fatalf("prepareWorkspace failed: %v", err)
		}

		agentPath := filepath.Join(workspace, agentFilePath)
		data, err := os.ReadFile(agentPath)
		if err != nil {
			t.Fatalf("failed to read agent file: %v", err)
		}
		content := string(data)

		skillIdx := strings.Index(content, "## Available Skills")
		personaIdx := strings.Index(content, "# Craftsman")

		if skillIdx == -1 || personaIdx == -1 {
			t.Fatalf("missing sections: skill=%d persona=%d", skillIdx, personaIdx)
		}

		if personaIdx >= skillIdx {
			t.Errorf("wrong ordering: persona=%d should appear before skill=%d", personaIdx, skillIdx)
		}
	})
}

// ---------------------------------------------------------------------------
// PersonaToAgentMarkdown tests
// ---------------------------------------------------------------------------

func TestPersonaToAgentMarkdown(t *testing.T) {
	baseProtocol := "# Wave Agent Protocol\n\nYou are a Wave pipeline agent."
	systemPrompt := "# Navigator\n\nYou are a read-only codebase explorer."
	contractSection := "## Contract\n\nOutput must be valid JSON with key `files`."
	restrictions := "\n\n---\n\n## Restrictions\n\nDenied tools: Edit(*)"

	t.Run("frontmatter includes model when set", func(t *testing.T) {
		p := PersonaSpec{Model: "claude-opus-4-5"}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, contractSection, restrictions)
		if !strings.HasPrefix(out, "---\n") {
			t.Error("output should start with YAML frontmatter delimiter ---")
		}
		if !strings.Contains(out, "model: claude-opus-4-5\n") {
			t.Error("frontmatter should contain model field")
		}
	})

	t.Run("frontmatter omits model when empty", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if strings.Contains(out, "model:") {
			t.Error("frontmatter should not contain model field when persona has no model")
		}
	})

	t.Run("frontmatter includes tools list", func(t *testing.T) {
		p := PersonaSpec{AllowedTools: []string{"Read", "Glob", "Grep"}}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if !strings.Contains(out, "tools:\n") {
			t.Error("frontmatter should contain tools section")
		}
		if !strings.Contains(out, "  - Read\n") {
			t.Error("tools section should list Read")
		}
		if !strings.Contains(out, "  - Glob\n") {
			t.Error("tools section should list Glob")
		}
	})

	t.Run("frontmatter includes disallowedTools", func(t *testing.T) {
		p := PersonaSpec{DenyTools: []string{"Edit(*)", "Bash(git commit*)"}}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if !strings.Contains(out, "disallowedTools:\n") {
			t.Error("frontmatter should contain disallowedTools section")
		}
		if !strings.Contains(out, "  - Edit(*)\n") {
			t.Error("disallowedTools should list Edit(*)")
		}
	})

	t.Run("permissionMode is always bypassPermissions", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if !strings.Contains(out, "permissionMode: bypassPermissions\n") {
			t.Error("frontmatter should always include permissionMode: bypassPermissions")
		}
	})

	t.Run("body contains base protocol", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if !strings.Contains(out, baseProtocol) {
			t.Error("body should contain base protocol text")
		}
	})

	t.Run("body contains system prompt", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if !strings.Contains(out, systemPrompt) {
			t.Error("body should contain system prompt")
		}
	})

	t.Run("body contains contract section when provided", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, contractSection, "")
		if !strings.Contains(out, contractSection) {
			t.Error("body should contain contract section")
		}
	})

	t.Run("body omits contract section when empty", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", "")
		if strings.Contains(out, "## Contract") {
			t.Error("body should not contain contract section when empty")
		}
	})

	t.Run("body contains restrictions when provided", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, "", restrictions)
		if !strings.Contains(out, restrictions) {
			t.Error("body should contain restrictions section")
		}
	})

	t.Run("section ordering: base protocol before system prompt before contract", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, baseProtocol, systemPrompt, contractSection, "")
		baseIdx := strings.Index(out, "Wave Agent Protocol")
		promptIdx := strings.Index(out, "# Navigator")
		contractIdx := strings.Index(out, "## Contract")
		if baseIdx == -1 || promptIdx == -1 || contractIdx == -1 {
			t.Fatalf("missing sections: base=%d prompt=%d contract=%d", baseIdx, promptIdx, contractIdx)
		}
		if baseIdx >= promptIdx || promptIdx >= contractIdx {
			t.Errorf("wrong section order: base=%d prompt=%d contract=%d", baseIdx, promptIdx, contractIdx)
		}
	})

	t.Run("tools are passed through without normalization", func(t *testing.T) {
		p := PersonaSpec{AllowedTools: []string{"Write", "Write(.agents/output/*)"}}
		out := PersonaToAgentMarkdown(p, "", "", "", "")
		// Both bare Write and scoped Write should appear (no subsumption)
		if !strings.Contains(out, "  - Write\n") {
			t.Error("expected bare Write in tools")
		}
		if !strings.Contains(out, "  - Write(.agents/output/*)\n") {
			t.Error("expected scoped Write(.agents/output/*) in tools — no normalization")
		}
	})

	t.Run("empty persona produces minimal valid frontmatter", func(t *testing.T) {
		p := PersonaSpec{}
		out := PersonaToAgentMarkdown(p, "", "", "", "")
		if !strings.HasPrefix(out, "---\n") {
			t.Error("output must start with ---")
		}
		if !strings.Contains(out, "---\npermissionMode: bypassPermissions\n---\n") {
			t.Errorf("minimal frontmatter not found in output:\n%s", out)
		}
	})
}

func TestPrepareWorkspaceAgentMode(t *testing.T) {
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	// Setup base protocol in a real .agents/personas directory
	wavePersonasDir := filepath.Join(".agents", "personas")
	if err := os.MkdirAll(wavePersonasDir, 0755); err != nil {
		t.Fatalf("failed to create .agents/personas: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wavePersonasDir, "base-protocol.md"), []byte(testBaseProtocol), 0644); err != nil {
		t.Fatalf("failed to write base-protocol.md: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(".agents") })

	cfg := AdapterRunConfig{
		Persona:      "navigator",
		SystemPrompt: "# Navigator\n\nYou explore codebases.",
		AllowedTools: []string{"Read", "Glob"},
		DenyTools:    []string{"Edit(*)"},
		Model:        "sonnet",
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	t.Run("agent md file is written", func(t *testing.T) {
		agentPath := filepath.Join(tmpDir, agentFilePath)
		data, err := os.ReadFile(agentPath)
		if err != nil {
			t.Fatalf("agent .md file not found: %v", err)
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\n") {
			t.Error("agent .md should start with YAML frontmatter")
		}
		if !strings.Contains(content, "model: sonnet\n") {
			t.Error("agent .md should contain model")
		}
		if !strings.Contains(content, "permissionMode: bypassPermissions\n") {
			t.Error("agent .md should contain permissionMode: bypassPermissions")
		}
		if !strings.Contains(content, "# Navigator") {
			t.Error("agent .md should contain system prompt")
		}
	})

	t.Run("CLAUDE.md is not written", func(t *testing.T) {
		claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
		if _, err := os.Stat(claudeMdPath); !os.IsNotExist(err) {
			t.Error("CLAUDE.md should not be written")
		}
	})
}

// T028: Verify buildArgs produces --agent, no legacy flags
func TestBuildArgsAgentMode(t *testing.T) {
	adapter := NewClaudeAdapter()
	args := adapter.buildArgs()

	// --agent must be present
	hasAgent := false
	for i, arg := range args {
		if arg == "--agent" && i+1 < len(args) {
			hasAgent = true
			if args[i+1] != agentFilePath {
				t.Errorf("--agent value = %q, want %q", args[i+1], agentFilePath)
			}
		}
	}
	if !hasAgent {
		t.Error("--agent flag not found in args")
	}

	// Retained flags must be present
	argsStr := strings.Join(args, " ")
	for _, want := range []string{"--output-format", "--verbose", "--no-session-persistence", "--dangerously-skip-permissions"} {
		if !strings.Contains(argsStr, want) {
			t.Errorf("args missing retained flag %q", want)
		}
	}

	// Legacy flags must NOT be present (permissions now in frontmatter, model in frontmatter)
	for _, absent := range []string{"--allowedTools", "--disallowedTools", "--model"} {
		if strings.Contains(argsStr, absent) {
			t.Errorf("args should not contain legacy flag %q", absent)
		}
	}
}

// T029: Verify TodoWrite is injected into disallowedTools in agent frontmatter
func TestTodoWriteInjection(t *testing.T) {
	setupBaseProtocol(t)
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:      "test",
		Model:        "sonnet",
		AllowedTools: []string{"Read", "Write"},
		DenyTools:    []string{"Bash(rm *)"},
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
	if err != nil {
		t.Fatalf("failed to read agent file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "  - TodoWrite\n") {
		t.Error("agent frontmatter should contain TodoWrite in disallowedTools")
	}
	if !strings.Contains(content, "  - Bash(rm *)\n") {
		t.Error("agent frontmatter should contain original deny rule Bash(rm *)")
	}
}

// T030: Verify no duplicate when persona already denies TodoWrite
func TestTodoWriteNoDuplication(t *testing.T) {
	setupBaseProtocol(t)
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:      "test",
		Model:        "sonnet",
		AllowedTools: []string{"Read"},
		DenyTools:    []string{"TodoWrite", "Bash(sudo *)"},
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, agentFilePath))
	if err != nil {
		t.Fatalf("failed to read agent file: %v", err)
	}
	content := string(data)

	// TodoWrite should appear exactly once
	count := strings.Count(content, "  - TodoWrite\n")
	if count != 1 {
		t.Errorf("expected TodoWrite exactly once in disallowedTools, got %d", count)
	}
}

// T031: Verify agent frontmatter omits tools/disallowedTools when lists are empty
func TestEmptyToolLists(t *testing.T) {
	p := PersonaSpec{}
	out := PersonaToAgentMarkdown(p, "", "", "", "")

	if strings.Contains(out, "tools:") {
		t.Error("frontmatter should not contain 'tools:' when AllowedTools is empty")
	}
	if strings.Contains(out, "disallowedTools:") {
		t.Error("frontmatter should not contain 'disallowedTools:' when DenyTools is empty")
	}
	// Should still have permissionMode
	if !strings.Contains(out, "permissionMode: bypassPermissions") {
		t.Error("frontmatter should always contain permissionMode: bypassPermissions")
	}
}

// T032: Verify minimal settings.json with sandbox-only config
func TestSandboxOnlySettingsJSON(t *testing.T) {
	setupBaseProtocol(t)
	adapter := NewClaudeAdapter()
	tmpDir := t.TempDir()

	cfg := AdapterRunConfig{
		Persona:        "test",
		Model:          "sonnet",
		AllowedTools:   []string{"Read", "Write"},
		SandboxEnabled: true,
		AllowedDomains: []string{"api.anthropic.com"},
	}

	if err := adapter.prepareWorkspace(tmpDir, cfg); err != nil {
		t.Fatalf("prepareWorkspace failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}

	// Must contain only "sandbox" key
	if len(raw) != 1 {
		t.Errorf("settings.json should have exactly 1 top-level key, got %d: %v", len(raw), raw)
	}
	if _, ok := raw["sandbox"]; !ok {
		t.Error("settings.json missing 'sandbox' key")
	}

	// Must NOT contain model, temperature, permissions, or output_format
	for _, forbidden := range []string{"model", "temperature", "permissions", "output_format"} {
		if _, ok := raw[forbidden]; ok {
			t.Errorf("settings.json should not contain %q", forbidden)
		}
	}
}

package pipeline

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/forge"
)

// TestForgeIntegration_PromptResolution verifies that forge variables resolve
// correctly in prompt templates for all 4 forge types.
func TestForgeIntegration_PromptResolution(t *testing.T) {
	tests := []struct {
		name     string
		info     forge.ForgeInfo
		expected string
	}{
		{
			name: "GitHub",
			info: forge.ForgeInfo{
				Type: forge.ForgeGitHub, Host: "github.com",
				Owner: "recinq", Repo: "wave",
				CLITool: "gh", PipelinePrefix: "gh",
				PRTerm: "Pull Request", PRCommand: "pr",
			},
			expected: "Use gh pr create to make a Pull Request",
		},
		{
			name: "GitLab",
			info: forge.ForgeInfo{
				Type: forge.ForgeGitLab, Host: "gitlab.com",
				Owner: "myorg", Repo: "myrepo",
				CLITool: "glab", PipelinePrefix: "gl",
				PRTerm: "Merge Request", PRCommand: "mr",
			},
			expected: "Use glab mr create to make a Merge Request",
		},
		{
			name: "Bitbucket",
			info: forge.ForgeInfo{
				Type: forge.ForgeBitbucket, Host: "bitbucket.org",
				Owner: "team", Repo: "project",
				CLITool: "bb", PipelinePrefix: "bb",
				PRTerm: "Pull Request", PRCommand: "pr",
			},
			expected: "Use bb pr create to make a Pull Request",
		},
		{
			name: "Gitea",
			info: forge.ForgeInfo{
				Type: forge.ForgeGitea, Host: "gitea.example.com",
				Owner: "devs", Repo: "app",
				CLITool: "tea", PipelinePrefix: "gt",
				PRTerm: "Pull Request", PRCommand: "pr",
			},
			expected: "Use tea pr create to make a Pull Request",
		},
		{
			name: "Codeberg",
			info: forge.ForgeInfo{
				Type: forge.ForgeCodeberg, Host: "codeberg.org",
				Owner: "user", Repo: "project",
				CLITool: "tea", PipelinePrefix: "gt",
				PRTerm: "Pull Request", PRCommand: "pr",
			},
			expected: "Use tea pr create to make a Pull Request",
		},
	}

	template := "Use {{ forge.cli_tool }} {{ forge.pr_command }} create to make a {{ forge.pr_term }}"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewPipelineContext("test", "implement", "create-pr")
			InjectForgeVariables(ctx, tt.info)
			result := ctx.ResolvePlaceholders(template)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestForgeIntegration_PersonaResolution verifies persona name resolution
// with forge prefix variables for all 4 forge types.
func TestForgeIntegration_PersonaResolution(t *testing.T) {
	tests := []struct {
		name     string
		info     forge.ForgeInfo
		expected string
	}{
		{"GitHub", forge.Detect("https://github.com/o/r.git"), "gh-commenter"},
		{"GitLab", forge.Detect("https://gitlab.com/o/r.git"), "gl-commenter"},
		{"Bitbucket", forge.Detect("https://bitbucket.org/o/r.git"), "bb-commenter"},
		{"Gitea", forge.Detect("https://gitea.example.com/o/r.git"), "gt-commenter"},
		{"Codeberg", forge.Detect("https://codeberg.org/o/r.git"), "gt-commenter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewPipelineContext("test", "implement", "step")
			InjectForgeVariables(ctx, tt.info)
			result := ctx.ResolvePlaceholders("{{ forge.prefix }}-commenter")
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestForgeIntegration_ContractCommandResolution verifies forge variables
// resolve correctly in contract command templates.
func TestForgeIntegration_ContractCommandResolution(t *testing.T) {
	tests := []struct {
		name     string
		info     forge.ForgeInfo
		expected string
	}{
		{
			name:     "GitHub",
			info:     forge.Detect("https://github.com/o/r.git"),
			expected: "gh pr list --json state",
		},
		{
			name:     "GitLab",
			info:     forge.Detect("https://gitlab.com/o/r.git"),
			expected: "glab mr list --json state",
		},
		{
			name:     "Bitbucket",
			info:     forge.Detect("https://bitbucket.org/o/r.git"),
			expected: "bb pr list --json state",
		},
		{
			name:     "Gitea",
			info:     forge.Detect("https://gitea.example.com/o/r.git"),
			expected: "tea pr list --json state",
		},
		{
			name:     "Codeberg",
			info:     forge.Detect("https://codeberg.org/o/r.git"),
			expected: "tea pr list --json state",
		},
	}

	template := "{{ forge.cli_tool }} {{ forge.pr_command }} list --json state"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewPipelineContext("test", "impl", "step")
			InjectForgeVariables(ctx, tt.info)
			result := ctx.ResolvePlaceholders(template)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestForgeIntegration_SourcePathResolution verifies forge prefix resolution
// in source path templates.
func TestForgeIntegration_SourcePathResolution(t *testing.T) {
	tests := []struct {
		name     string
		info     forge.ForgeInfo
		expected string
	}{
		{"GitHub", forge.Detect("https://github.com/o/r.git"), ".wave/prompts/gh/create-pr.md"},
		{"GitLab", forge.Detect("https://gitlab.com/o/r.git"), ".wave/prompts/gl/create-pr.md"},
		{"Bitbucket", forge.Detect("https://bitbucket.org/o/r.git"), ".wave/prompts/bb/create-pr.md"},
		{"Gitea", forge.Detect("https://gitea.example.com/o/r.git"), ".wave/prompts/gt/create-pr.md"},
		{"Codeberg", forge.Detect("https://codeberg.org/o/r.git"), ".wave/prompts/gt/create-pr.md"},
	}

	template := ".wave/prompts/{{ forge.prefix }}/create-pr.md"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewPipelineContext("test", "impl", "step")
			InjectForgeVariables(ctx, tt.info)
			result := ctx.ResolvePlaceholders(template)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestForgeIntegration_FullExecutorFlow verifies the full executor flow uses
// forge variables correctly for all forge types using synthetic ForgeInfo.
func TestForgeIntegration_FullExecutorFlow(t *testing.T) {
	tests := []struct {
		name     string
		info     forge.ForgeInfo
		wantCLI  string
		wantPR   string
		wantType string
	}{
		{
			name:     "GitHub",
			info:     forge.ForgeInfo{Type: forge.ForgeGitHub, Host: "github.com", Owner: "org", Repo: "repo", CLITool: "gh", PipelinePrefix: "gh", PRTerm: "Pull Request", PRCommand: "pr"},
			wantCLI:  "gh pr create",
			wantPR:   "github-committer",
			wantType: "github",
		},
		{
			name:     "GitLab",
			info:     forge.ForgeInfo{Type: forge.ForgeGitLab, Host: "gitlab.com", Owner: "org", Repo: "repo", CLITool: "glab", PipelinePrefix: "gl", PRTerm: "Merge Request", PRCommand: "mr"},
			wantCLI:  "glab mr create",
			wantPR:   "gitlab-committer",
			wantType: "gitlab",
		},
		{
			name:     "Bitbucket",
			info:     forge.ForgeInfo{Type: forge.ForgeBitbucket, Host: "bitbucket.org", Owner: "org", Repo: "repo", CLITool: "bb", PipelinePrefix: "bb", PRTerm: "Pull Request", PRCommand: "pr"},
			wantCLI:  "bb pr create",
			wantPR:   "bitbucket-committer",
			wantType: "bitbucket",
		},
		{
			name:     "Gitea",
			info:     forge.ForgeInfo{Type: forge.ForgeGitea, Host: "gitea.example.com", Owner: "org", Repo: "repo", CLITool: "tea", PipelinePrefix: "gt", PRTerm: "Pull Request", PRCommand: "pr"},
			wantCLI:  "tea pr create",
			wantPR:   "gitea-committer",
			wantType: "gitea",
		},
		{
			name:     "Codeberg",
			info:     forge.ForgeInfo{Type: forge.ForgeCodeberg, Host: "codeberg.org", Owner: "org", Repo: "repo", CLITool: "tea", PipelinePrefix: "gt", PRTerm: "Pull Request", PRCommand: "pr"},
			wantCLI:  "tea pr create",
			wantPR:   "codeberg-committer",
			wantType: "codeberg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewPipelineContext("test", "implement", "step")
			InjectForgeVariables(ctx, tt.info)

			result := ctx.ResolvePlaceholders("{{ forge.cli_tool }} {{ forge.pr_command }} create")
			if result != tt.wantCLI {
				t.Errorf("CLI command = %q, want %q", result, tt.wantCLI)
			}

			persona := ctx.ResolvePlaceholders("{{ forge.type }}-committer")
			if strings.Contains(persona, "{{") {
				t.Errorf("unresolved placeholders in persona: %q", persona)
			}
			if persona != tt.wantPR {
				t.Errorf("persona = %q, want %q", persona, tt.wantPR)
			}
		})
	}
}

// TestForgeIntegration_LocalForge verifies that ForgeLocal resolves template
// variables gracefully — empty CLI tool and PR commands, "local" for type/prefix.
func TestForgeIntegration_LocalForge(t *testing.T) {
	ctx := NewPipelineContext("test", "wave-validate", "lint-step")
	info := forge.ForgeInfo{
		Type:           forge.ForgeLocal,
		PipelinePrefix: "local",
	}
	InjectForgeVariables(ctx, info)

	// forge.type resolves to "local"
	result := ctx.ResolvePlaceholders("{{ forge.type }}")
	if result != "local" {
		t.Errorf("forge.type = %q, want %q", result, "local")
	}

	// forge.cli_tool resolves to empty — no forge CLI available
	result = ctx.ResolvePlaceholders("{{ forge.cli_tool }}")
	if result != "" {
		t.Errorf("forge.cli_tool = %q, want empty", result)
	}

	// forge.pr_term resolves to empty — no PR concept in local mode
	result = ctx.ResolvePlaceholders("{{ forge.pr_term }}")
	if result != "" {
		t.Errorf("forge.pr_term = %q, want empty", result)
	}

	// Template with forge vars should not have unresolved placeholders
	template := "type={{ forge.type }} cli={{ forge.cli_tool }} prefix={{ forge.prefix }}"
	result = ctx.ResolvePlaceholders(template)
	if strings.Contains(result, "{{") {
		t.Errorf("unresolved placeholders in result: %q", result)
	}
	expected := "type=local cli= prefix=local"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

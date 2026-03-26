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

// TestForgeIntegration_FullExecutorFlow_GitHub verifies the full executor
// flow uses forge variables when running in a GitHub repo. This test only
// runs when the test environment has a GitHub remote.
func TestForgeIntegration_FullExecutorFlow_GitHub(t *testing.T) {
	info, err := forge.DetectFromGitRemotes()
	if err != nil || info.Type != forge.ForgeGitHub {
		t.Skip("test requires a git remote pointing to github.com")
	}

	ctx := NewPipelineContext("test", "implement", "step")
	InjectForgeVariables(ctx, info)

	// Verify the detected info produces valid resolutions
	result := ctx.ResolvePlaceholders("{{ forge.cli_tool }} {{ forge.pr_command }} create")
	if result != "gh pr create" {
		t.Errorf("got %q, want %q", result, "gh pr create")
	}

	// Verify no unresolved placeholders
	persona := ctx.ResolvePlaceholders("{{ forge.type }}-committer")
	if strings.Contains(persona, "{{") {
		t.Errorf("unresolved placeholders in persona: %q", persona)
	}
	if persona != "github-committer" {
		t.Errorf("persona = %q, want %q", persona, "github-committer")
	}
}

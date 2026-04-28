package commands

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossForgeInit validates that wave init produces correct behaviour for
// each supported forge type: detection, pipeline filtering, persona filtering,
// and template variable resolution. No network access required — all detection
// is URL-based.
func TestCrossForgeInit(t *testing.T) {
	forges := []struct {
		name              string
		remoteURL         string // empty → use DetectWithOverride for local
		forgeOverride     string // non-empty only for ForgeLocal
		wantForgeType     forge.ForgeType
		wantCLI           string
		wantPrefix        string
		wantPersonaPrefix string // "github", "gitlab", "gitea", "bitbucket", ""
		wantPRCommand     string
		wantPRTerm        string
	}{
		{
			name:              "GitHub",
			remoteURL:         "https://github.com/org/repo.git",
			wantForgeType:     forge.ForgeGitHub,
			wantCLI:           "gh",
			wantPrefix:        "gh",
			wantPersonaPrefix: "github",
			wantPRCommand:     "pr",
			wantPRTerm:        "Pull Request",
		},
		{
			name:              "GitLab",
			remoteURL:         "https://gitlab.com/org/repo.git",
			wantForgeType:     forge.ForgeGitLab,
			wantCLI:           "glab",
			wantPrefix:        "gl",
			wantPersonaPrefix: "gitlab",
			wantPRCommand:     "mr",
			wantPRTerm:        "Merge Request",
		},
		{
			name:              "Gitea",
			remoteURL:         "https://gitea.com/org/repo.git",
			wantForgeType:     forge.ForgeGitea,
			wantCLI:           "tea",
			wantPrefix:        "gt",
			wantPersonaPrefix: "gitea",
			wantPRCommand:     "pr",
			wantPRTerm:        "Pull Request",
		},
		{
			name:              "Codeberg",
			remoteURL:         "https://codeberg.org/org/repo.git",
			wantForgeType:     forge.ForgeCodeberg,
			wantCLI:           "tea",
			wantPrefix:        "gt",
			wantPersonaPrefix: "gitea", // Codeberg shares Gitea personas
			wantPRCommand:     "pr",
			wantPRTerm:        "Pull Request",
		},
		{
			name:              "Bitbucket",
			remoteURL:         "https://bitbucket.org/org/repo.git",
			wantForgeType:     forge.ForgeBitbucket,
			wantCLI:           "bb",
			wantPrefix:        "bb",
			wantPersonaPrefix: "bitbucket",
			wantPRCommand:     "pr",
			wantPRTerm:        "Pull Request",
		},
		{
			name:              "Local (no remote)",
			forgeOverride:     "local",
			wantForgeType:     forge.ForgeLocal,
			wantCLI:           "",
			wantPrefix:        "local",
			wantPersonaPrefix: "",
			wantPRCommand:     "",
			wantPRTerm:        "",
		},
	}

	for _, tt := range forges {
		t.Run(tt.name, func(t *testing.T) {
			// --- 1. Forge detection ---
			var info forge.ForgeInfo
			if tt.forgeOverride != "" {
				info = forge.DetectWithOverride(tt.remoteURL, tt.forgeOverride)
			} else {
				info = forge.Detect(tt.remoteURL)
			}
			assert.Equal(t, tt.wantForgeType, info.Type, "forge type mismatch")
			assert.Equal(t, tt.wantCLI, info.CLITool, "CLI tool mismatch")
			assert.Equal(t, tt.wantPrefix, info.PipelinePrefix, "pipeline prefix mismatch")
			assert.Equal(t, tt.wantPRCommand, info.PRCommand, "PR command mismatch")
			assert.Equal(t, tt.wantPRTerm, info.PRTerm, "PR term mismatch")

			// --- 2. Pipeline filtering ---
			// Mix of forge-prefixed and generic pipeline names.
			allPipelines := []string{
				"gh-implement", "gl-deploy", "gt-sync", "bb-build",
				"impl-issue", "audit-security",
			}
			filtered := forge.FilterPipelinesByForge(info.Type, allPipelines)

			// Every returned pipeline must either match this forge's prefix or have no forge prefix.
			for _, p := range filtered {
				hasForgePrefix := strings.HasPrefix(p, "gh-") ||
					strings.HasPrefix(p, "gl-") ||
					strings.HasPrefix(p, "gt-") ||
					strings.HasPrefix(p, "bb-") ||
					strings.HasPrefix(p, "local-")
				if hasForgePrefix {
					assert.True(t, strings.HasPrefix(p, tt.wantPrefix+"-"),
						"pipeline %q has wrong forge prefix for %s (want prefix %q)", p, tt.name, tt.wantPrefix)
				}
			}

			// Generic pipelines (no forge prefix) must always be included.
			assert.Contains(t, filtered, "impl-issue", "generic pipeline should be included")
			assert.Contains(t, filtered, "audit-security", "generic pipeline should be included")

			// Exactly one forge-prefixed pipeline should survive (the matching one),
			// unless this is a forge type whose prefix doesn't appear in the test list.
			forgePrefixed := filterByAnyForgePrefix(filtered)
			if tt.wantPrefix != "local" {
				// For non-local forges, exactly the matching prefixed pipeline survives.
				assert.Len(t, forgePrefixed, 1, "expected exactly 1 forge-prefixed pipeline for %s", tt.name)
			} else {
				// For local, no forge-prefixed pipelines should survive.
				assert.Empty(t, forgePrefixed, "local forge should exclude all forge-prefixed pipelines")
			}

			// --- 3. Persona filtering ---
			personaConfigs, err := defaults.GetPersonaConfigs()
			require.NoError(t, err, "loading persona configs")

			filteredPersonas := onboarding.FilterPersonasByForge(personaConfigs, info.Type)

			if tt.wantPersonaPrefix != "" {
				// For forges with a persona prefix, every forge-specific persona
				// in the result must match this forge.
				for name := range filteredPersonas {
					if isForgeSpecific(name) {
						assert.True(t, strings.HasPrefix(name, tt.wantPersonaPrefix+"-"),
							"persona %q has wrong forge prefix for %s (want %q-)", name, tt.name, tt.wantPersonaPrefix)
					}
				}
			} else {
				// For local/unknown forges, no filtering is applied — all personas
				// are returned (there's no forge prefix to filter on).
				assert.Equal(t, len(personaConfigs), len(filteredPersonas),
					"local forge should return all personas unfiltered")
			}

			// Generic personas (no forge prefix) should always survive filtering.
			genericCount := 0
			for name := range filteredPersonas {
				if !isForgeSpecific(name) {
					genericCount++
				}
			}
			allGenericCount := 0
			for name := range personaConfigs {
				if !isForgeSpecific(name) {
					allGenericCount++
				}
			}
			assert.Equal(t, allGenericCount, genericCount,
				"all generic personas should survive forge filtering for %s", tt.name)

			// --- 4. Template variable resolution ---
			ctx := pipeline.NewPipelineContext("test-run-abc123", "test-pipe", "test-step")
			pipeline.InjectForgeVariables(ctx, info)

			// Resolve a prompt that uses all forge template variables.
			prompt := "Use {{ forge.cli_tool }} {{ forge.pr_command }} to create a {{ forge.pr_term }}"
			resolved := ctx.ResolvePlaceholders(prompt)

			if tt.wantCLI != "" {
				assert.Contains(t, resolved, tt.wantCLI,
					"resolved prompt should contain CLI tool %q", tt.wantCLI)
				assert.Contains(t, resolved, tt.wantPRCommand,
					"resolved prompt should contain PR command %q", tt.wantPRCommand)
				assert.Contains(t, resolved, tt.wantPRTerm,
					"resolved prompt should contain PR term %q", tt.wantPRTerm)
			}
			// No unresolved {{ forge.* }} placeholders should remain.
			assert.NotContains(t, resolved, "{{ forge.",
				"unresolved forge placeholders remain in: %s", resolved)
			assert.NotContains(t, resolved, "{{forge.",
				"unresolved forge placeholders remain in: %s", resolved)

			// Verify individual forge variables resolve correctly.
			assert.Equal(t, tt.wantCLI, ctx.ResolvePlaceholders("{{ forge.cli_tool }}"))
			assert.Equal(t, string(tt.wantForgeType), ctx.ResolvePlaceholders("{{ forge.type }}"))
			assert.Equal(t, tt.wantPrefix, ctx.ResolvePlaceholders("{{ forge.prefix }}"))
			assert.Equal(t, tt.wantPRCommand, ctx.ResolvePlaceholders("{{ forge.pr_command }}"))
			assert.Equal(t, tt.wantPRTerm, ctx.ResolvePlaceholders("{{ forge.pr_term }}"))
		})
	}
}

// TestCrossForgeInit_SSHURLs verifies forge detection works with SSH remote URLs.
func TestCrossForgeInit_SSHURLs(t *testing.T) {
	tests := []struct {
		name      string
		sshURL    string
		wantType  forge.ForgeType
		wantOwner string
		wantRepo  string
	}{
		{"GitHub SSH", "git@github.com:org/repo.git", forge.ForgeGitHub, "org", "repo"},
		{"GitLab SSH", "git@gitlab.com:org/repo.git", forge.ForgeGitLab, "org", "repo"},
		{"Bitbucket SSH", "git@bitbucket.org:org/repo.git", forge.ForgeBitbucket, "org", "repo"},
		{"Codeberg SSH", "git@codeberg.org:org/repo.git", forge.ForgeCodeberg, "org", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := forge.Detect(tt.sshURL)
			assert.Equal(t, tt.wantType, info.Type)
			assert.Equal(t, tt.wantOwner, info.Owner)
			assert.Equal(t, tt.wantRepo, info.Repo)
		})
	}
}

// TestCrossForgeInit_PipelineFilterExclusion verifies that forge-specific
// pipelines from OTHER forges are correctly excluded.
func TestCrossForgeInit_PipelineFilterExclusion(t *testing.T) {
	allPipelines := []string{
		"gh-implement", "gh-scope",
		"gl-deploy", "gl-merge",
		"gt-sync", "gt-release",
		"bb-build", "bb-deploy",
		"impl-issue", "audit-security", "ops-pr-review",
	}

	tests := []struct {
		forgeType    forge.ForgeType
		wantIncluded []string
		wantExcluded []string
	}{
		{
			forge.ForgeGitHub,
			[]string{"gh-implement", "gh-scope", "impl-issue", "audit-security", "ops-pr-review"},
			[]string{"gl-deploy", "gl-merge", "gt-sync", "gt-release", "bb-build", "bb-deploy"},
		},
		{
			forge.ForgeGitLab,
			[]string{"gl-deploy", "gl-merge", "impl-issue", "audit-security", "ops-pr-review"},
			[]string{"gh-implement", "gh-scope", "gt-sync", "gt-release", "bb-build", "bb-deploy"},
		},
		{
			forge.ForgeGitea,
			[]string{"gt-sync", "gt-release", "impl-issue", "audit-security", "ops-pr-review"},
			[]string{"gh-implement", "gh-scope", "gl-deploy", "gl-merge", "bb-build", "bb-deploy"},
		},
		{
			forge.ForgeCodeberg, // shares gt- prefix with Gitea
			[]string{"gt-sync", "gt-release", "impl-issue", "audit-security", "ops-pr-review"},
			[]string{"gh-implement", "gh-scope", "gl-deploy", "gl-merge", "bb-build", "bb-deploy"},
		},
		{
			forge.ForgeBitbucket,
			[]string{"bb-build", "bb-deploy", "impl-issue", "audit-security", "ops-pr-review"},
			[]string{"gh-implement", "gh-scope", "gl-deploy", "gl-merge", "gt-sync", "gt-release"},
		},
		{
			forge.ForgeLocal,
			[]string{"impl-issue", "audit-security", "ops-pr-review"},
			[]string{"gh-implement", "gh-scope", "gl-deploy", "gl-merge", "gt-sync", "gt-release", "bb-build", "bb-deploy"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.forgeType), func(t *testing.T) {
			filtered := forge.FilterPipelinesByForge(tt.forgeType, allPipelines)
			for _, want := range tt.wantIncluded {
				assert.Contains(t, filtered, want, "should include %q for %s", want, tt.forgeType)
			}
			for _, excluded := range tt.wantExcluded {
				assert.NotContains(t, filtered, excluded, "should exclude %q for %s", excluded, tt.forgeType)
			}
		})
	}
}

// TestCrossForgeInit_ForgejoSharesGiteaPrefix verifies that Forgejo uses the
// same pipeline prefix and CLI tool as Gitea.
func TestCrossForgeInit_ForgejoSharesGiteaPrefix(t *testing.T) {
	// Forgejo isn't in the main table because there's no well-known hostname
	// for it (it's typically self-hosted). Use the override mechanism.
	info := forge.DetectWithOverride("https://my-forgejo.example.com/org/repo.git", "forgejo")
	assert.Equal(t, forge.ForgeForgejo, info.Type)
	assert.Equal(t, "tea", info.CLITool)
	assert.Equal(t, "gt", info.PipelinePrefix)
	assert.Equal(t, "pr", info.PRCommand)
	assert.Equal(t, "Pull Request", info.PRTerm)

	// Should filter the same as Gitea.
	pipelines := []string{"gh-implement", "gt-sync", "gl-deploy", "impl-issue"}
	filtered := forge.FilterPipelinesByForge(info.Type, pipelines)
	assert.Contains(t, filtered, "gt-sync")
	assert.Contains(t, filtered, "impl-issue")
	assert.NotContains(t, filtered, "gh-implement")
	assert.NotContains(t, filtered, "gl-deploy")
}

// filterByAnyForgePrefix returns only names that start with a known forge prefix.
func filterByAnyForgePrefix(names []string) []string {
	var result []string
	for _, n := range names {
		if strings.HasPrefix(n, "gh-") ||
			strings.HasPrefix(n, "gl-") ||
			strings.HasPrefix(n, "gt-") ||
			strings.HasPrefix(n, "bb-") ||
			strings.HasPrefix(n, "local-") {
			result = append(result, n)
		}
	}
	return result
}

// isForgeSpecific returns true if the persona name starts with a known forge prefix.
func isForgeSpecific(name string) bool {
	for _, prefix := range []string{"github-", "gitlab-", "bitbucket-", "gitea-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

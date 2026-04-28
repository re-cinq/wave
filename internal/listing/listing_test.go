package listing

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chdirToTemp switches into a fresh temp dir for the test and restores the
// original working directory afterwards.
func chdirToTemp(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return tmp
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{5 * time.Second, "5.0s"},
		{90 * time.Second, "1m30s"},
		{65 * time.Minute, "1h5m"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatDuration(tc.in))
		})
	}
}

func TestListPipelines_MissingDirectoryReturnsNil(t *testing.T) {
	chdirToTemp(t)
	pipelines, err := ListPipelines()
	require.NoError(t, err)
	assert.Nil(t, pipelines)
}

func TestListPipelines_ParsesAndSorts(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/pipelines/zebra.yaml", `metadata:
  description: Z
steps:
  - id: one
  - id: two
`)
	writeFile(t, ".agents/pipelines/alpha.yaml", `metadata:
  description: A
steps:
  - id: only
`)

	pipelines, err := ListPipelines()
	require.NoError(t, err)
	require.Len(t, pipelines, 2)
	assert.Equal(t, "alpha", pipelines[0].Name)
	assert.Equal(t, "zebra", pipelines[1].Name)
	assert.Equal(t, 1, pipelines[0].StepCount)
	assert.Equal(t, []string{"one", "two"}, pipelines[1].Steps)
}

func TestListPipelines_BadYAMLMarkedAsError(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/pipelines/broken.yaml", "{ not: valid: yaml")
	pipelines, err := ListPipelines()
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	assert.Equal(t, "broken", pipelines[0].Name)
	assert.Contains(t, pipelines[0].Description, "error parsing")
}

func TestListPersonas_SortsAndCopiesPermissions(t *testing.T) {
	in := map[string]ManifestPersona{
		"zeta":  {Adapter: "claude", Description: "Z", Temperature: 0.2},
		"alpha": {Adapter: "opencode", Description: "A", Temperature: 0.7},
	}
	in["alpha"] = ManifestPersona{
		Adapter:     "opencode",
		Description: "A",
		Temperature: 0.7,
		Permissions: struct {
			AllowedTools []string `yaml:"allowed_tools"`
			Deny         []string `yaml:"deny"`
		}{AllowedTools: []string{"Read"}, Deny: []string{"Write"}},
	}

	got := ListPersonas(in)
	require.Len(t, got, 2)
	assert.Equal(t, "alpha", got[0].Name)
	assert.Equal(t, []string{"Read"}, got[0].AllowedTools)
	assert.Equal(t, []string{"Write"}, got[0].DeniedTools)
	assert.Equal(t, "zeta", got[1].Name)
}

func TestListPersonas_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, ListPersonas(nil))
	assert.Nil(t, ListPersonas(map[string]ManifestPersona{}))
}

func TestListAdapters_DetectsAvailability(t *testing.T) {
	in := map[string]ManifestAdapter{
		"real":    {Binary: "ls", Mode: "headless", OutputFormat: "json"},
		"missing": {Binary: "definitely-not-a-real-binary-xyz123", Mode: "x", OutputFormat: "y"},
	}
	got := ListAdapters(in)
	require.Len(t, got, 2)
	// Sorted alphabetically: missing, real
	assert.Equal(t, "missing", got[0].Name)
	assert.False(t, got[0].Available)
	assert.Equal(t, "real", got[1].Name)
	assert.True(t, got[1].Available)
}

func TestExtractPipelineName_StripsRunIDSuffix(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/pipelines/gh-implement.yaml", "steps: []\n")
	assert.Equal(t, "gh-implement", ExtractPipelineName("gh-implement-abcd1234"))
	assert.Equal(t, "no-match", ExtractPipelineName("no-match"))
}

func TestListContracts_EmptyDirectoryReturnsNil(t *testing.T) {
	chdirToTemp(t)
	contracts, err := ListContracts()
	require.NoError(t, err)
	assert.Nil(t, contracts)
}

func TestListContracts_FindsUsageInPipelines(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/contracts/navigation.json", `{"type":"object"}`)
	writeFile(t, ".agents/contracts/orphan.schema.json", `{"type":"object"}`)
	writeFile(t, ".agents/pipelines/use-it.yaml", `steps:
  - id: navigate
    persona: navigator
    contract:
      schema_path: .agents/contracts/navigation.json
`)

	contracts, err := ListContracts()
	require.NoError(t, err)
	require.Len(t, contracts, 2)

	byName := map[string]ContractInfo{}
	for _, c := range contracts {
		byName[c.Name] = c
	}
	nav := byName["navigation"]
	require.Len(t, nav.UsedBy, 1)
	assert.Equal(t, "use-it", nav.UsedBy[0].Pipeline)
	assert.Equal(t, "navigate", nav.UsedBy[0].Step)
	assert.Equal(t, "navigator", nav.UsedBy[0].Persona)

	orphan := byName["orphan"]
	assert.Empty(t, orphan.UsedBy)
}

func TestCollectSkillsFromPipelines_FirstWins(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/pipelines/a-first.yaml", `requires:
  skills:
    speckit:
      check: "true"
      install: "first"
`)
	writeFile(t, ".agents/pipelines/b-second.yaml", `requires:
  skills:
    speckit:
      check: "false"
      install: "second"
`)
	merged := CollectSkillsFromPipelines()
	require.Contains(t, merged, "speckit")
	assert.Equal(t, "first", merged["speckit"].Install)
}

func TestCollectSkillPipelineUsage_SortedPerSkill(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/pipelines/zeta.yaml", `requires:
  skills:
    one:
      check: "true"
`)
	writeFile(t, ".agents/pipelines/alpha.yaml", `requires:
  skills:
    one:
      check: "true"
`)
	usage := CollectSkillPipelineUsage()
	assert.Equal(t, []string{"alpha", "zeta"}, usage["one"])
}

func TestListSkills_RunsCheckAndAttachesUsage(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/pipelines/p.yaml", `requires:
  skills:
    speckit:
      check: "true"
    linter:
      check: "false"
      install: "go install lint"
`)
	skills := CollectSkillsFromPipelines()
	got := ListSkills(skills)
	require.Len(t, got, 2)
	byName := map[string]SkillInfo{}
	for _, s := range got {
		byName[s.Name] = s
	}
	assert.True(t, byName["speckit"].Installed, "check=true should be installed")
	assert.False(t, byName["linter"].Installed, "check=false should not be installed")
	assert.Equal(t, []string{"p"}, byName["speckit"].UsedBy)
}

func TestListSkills_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, ListSkills(nil))
}

func TestLoadManifest_MissingFile(t *testing.T) {
	chdirToTemp(t)
	_, err := LoadManifest("does-not-exist.yaml")
	assert.Error(t, err)
}

func TestLoadManifest_ParsesAdaptersAndPersonas(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, "wave.yaml", `adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json
personas:
  navigator:
    adapter: claude
    description: nav
    temperature: 0.1
    permissions:
      allowed_tools: [Read]
      deny: [Write]
`)
	m, err := LoadManifest("wave.yaml")
	require.NoError(t, err)
	require.Contains(t, m.Adapters, "claude")
	assert.Equal(t, "claude", m.Adapters["claude"].Binary)
	require.Contains(t, m.Personas, "navigator")
	assert.Equal(t, []string{"Read"}, m.Personas["navigator"].Permissions.AllowedTools)
	assert.Equal(t, []string{"Write"}, m.Personas["navigator"].Permissions.Deny)
}

func TestListRuns_NoStateNoWorkspaces(t *testing.T) {
	chdirToTemp(t)
	runs, err := ListRuns(RunsOptions{Limit: 10})
	require.NoError(t, err)
	assert.Nil(t, runs)
}

func TestListRuns_WorkspaceFallback(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/workspaces/run-one/marker.txt", "x")
	writeFile(t, ".agents/workspaces/run-two/marker.txt", "y")
	runs, err := ListRuns(RunsOptions{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, runs, 2)
}

func TestListRuns_WorkspaceFallback_PipelineFilter(t *testing.T) {
	chdirToTemp(t)
	writeFile(t, ".agents/workspaces/target/marker.txt", "x")
	writeFile(t, ".agents/workspaces/other/marker.txt", "y")
	runs, err := ListRuns(RunsOptions{Limit: 10, Pipeline: "target"})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "target", runs[0].RunID)
}

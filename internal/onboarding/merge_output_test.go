package onboarding

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	wsroot "github.com/recinq/wave/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeManifests_ExistingValuesWinRecursively(t *testing.T) {
	defaults := map[string]interface{}{
		"apiVersion": "v1",
		"runtime": map[string]interface{}{
			"workspace_root": ".agents/workspaces",
			"relay":          map[string]interface{}{"strategy": "summarize", "token_threshold_percent": 80},
		},
		"personas": map[string]interface{}{"navigator": map[string]interface{}{"adapter": "claude"}},
	}
	existing := map[string]interface{}{
		"runtime": map[string]interface{}{
			"relay": map[string]interface{}{"token_threshold_percent": 60},
		},
		"personas": "not-a-map",
		"custom":   true,
	}

	merged := MergeManifests(defaults, existing)

	assert.Equal(t, "v1", merged["apiVersion"], "missing keys come from defaults")
	assert.Equal(t, true, merged["custom"], "user-only keys are kept")
	assert.Equal(t, "not-a-map", merged["personas"], "a scalar in existing replaces a default map")
	runtime := merged["runtime"].(map[string]interface{})
	assert.Equal(t, ".agents/workspaces", runtime["workspace_root"])
	relay := runtime["relay"].(map[string]interface{})
	assert.Equal(t, 60, relay["token_threshold_percent"], "existing nested value wins")
	assert.Equal(t, "summarize", relay["strategy"], "nested default fills the gap")
}

func TestComputeManifestDiff(t *testing.T) {
	defaults := map[string]interface{}{
		"kind":     "WaveManifest",
		"runtime":  map[string]interface{}{"workspace_root": ".agents/workspaces", "max_concurrent_workers": 5},
		"personas": map[string]interface{}{"navigator": map[string]interface{}{}},
	}
	existing := map[string]interface{}{
		"kind":    "WaveManifest",
		"runtime": map[string]interface{}{"workspace_root": "ws"},
		"project": map[string]interface{}{"test_command": "make test"},
	}

	merged, changes := ComputeManifestDiff(defaults, existing)

	assert.Equal(t, []ManifestChangeEntry{
		{KeyPath: "personas", Action: ManifestActionAdded},
		{KeyPath: "project", Action: ManifestActionPreserved},
		{KeyPath: "runtime.max_concurrent_workers", Action: ManifestActionAdded},
		{KeyPath: "runtime.workspace_root", Action: ManifestActionPreserved},
	}, changes, "sorted by key path; identical values are not reported")
	assert.Equal(t, "ws", merged["runtime"].(map[string]interface{})["workspace_root"])
	assert.Contains(t, merged, "personas")
}

func TestChangeSummary_ClassifyDisplayApply(t *testing.T) {
	chdirTemp(t)
	writeFile(t, ".agents/personas/navigator.md", "custom navigator")
	writeFile(t, ".agents/pipelines/hello.yaml", "kind: WavePipeline\n")

	assets := &AssetSet{
		Personas:  map[string]string{"navigator.md": "default navigator", "planner.md": "planner"},
		Pipelines: map[string]string{"hello.yaml": "kind: WavePipeline\n"},
		Contracts: map[string]string{"out.schema.json": "{}"},
		Prompts:   map[string]string{"gh/create-pr.md": "open a PR"},
	}
	existing := map[string]interface{}{"kind": "WaveManifest", "custom": 1}
	defaults := map[string]interface{}{"kind": "WaveManifest", "runtime": map[string]interface{}{}}

	summary := ComputeChangeSummary(assets, existing, defaults)

	status := map[string]FileStatus{}
	for _, f := range summary.Files {
		status[f.RelPath] = f.Status
	}
	assert.Equal(t, map[string]FileStatus{
		filepath.Join(".agents", "personas", "navigator.md"):      FileStatusPreserved,
		filepath.Join(".agents", "personas", "planner.md"):        FileStatusNew,
		filepath.Join(".agents", "pipelines", "hello.yaml"):       FileStatusUpToDate,
		filepath.Join(".agents", "contracts", "out.schema.json"):  FileStatusNew,
		filepath.Join(".agents", "prompts", "gh", "create-pr.md"): FileStatusNew,
	}, status)
	assert.False(t, summary.AlreadyUpToDate)
	assert.Same(t, assets, summary.Assets)

	var out bytes.Buffer
	DisplayChangeSummary(&out, summary)
	text := out.String()
	for _, want := range []string{
		"Change Summary:",
		"Personas:",
		"~ preserved    .agents/personas/navigator.md",
		"+ new          .agents/personas/planner.md",
		"= up to date   .agents/pipelines/hello.yaml",
		"Contracts:",
		"Prompts:",
		"Manifest (wave.yaml):",
		"+ added        runtime",
		"~ preserved    custom",
	} {
		assert.Contains(t, text, want)
	}

	require.NoError(t, ApplyChanges(summary, "wave.yaml"))
	for path, want := range map[string]string{
		".agents/personas/navigator.md":     "custom navigator", // preserved, untouched
		".agents/personas/planner.md":       "planner",
		".agents/contracts/out.schema.json": "{}",
		".agents/prompts/gh/create-pr.md":   "open a PR",
	} {
		data, err := os.ReadFile(path)
		require.NoError(t, err, path)
		assert.Equal(t, want, string(data), path)
	}
	m := readManifest(t, "wave.yaml")
	assert.Equal(t, 1, m["custom"])
	assert.Contains(t, m, "runtime")
	assert.DirExists(t, ".agents/traces")

	// Re-running against the applied state finds nothing new to add.
	again := ComputeChangeSummary(assets, m, defaults)
	assert.True(t, again.AlreadyUpToDate)
}

func TestValidateHint(t *testing.T) {
	assert.Equal(t, "wave validate", ValidateHint(""))
	assert.Equal(t, "wave validate", ValidateHint("./wave.yaml"))
	assert.Equal(t, "wave validate --manifest sub/wave.yaml", ValidateHint("sub/wave.yaml"))
}

func TestPrintInitSuccess(t *testing.T) {
	res := &GreenfieldResult{
		Assets: &AssetSet{
			PersonaConfigs: map[string]manifest.Persona{"navigator": {}, "craftsman": {}},
			Pipelines:      map[string]string{"plan-task.yaml": "", "ops-hello-world.yaml": ""},
			Contracts:      map[string]string{"a.json": ""},
			Prompts:        map[string]string{},
		},
		Adapter:          "codex",
		Model:            "gpt-6.2-sol",
		Project:          map[string]interface{}{"test_command": "go test ./..."},
		InstructionFiles: []string{"AGENTS.md", "CLAUDE.md"},
		GitignoreUpdated: true,
	}

	var out bytes.Buffer
	PrintInitSuccess(&out, "app/wave.yaml", "ws/", res)
	text := out.String()

	for _, want := range []string{
		"Project initialized successfully!",
		"Adapter: codex (model gpt-6.2-sol for every tier)",
		"Project root: app (paths below are relative to it)",
		"wave.yaml                  Main manifest",
		".agents/personas/          2 personas",
		".agents/pipelines/         2 pipelines",
		".agents/contracts/         1 contracts",
		".agents/prompts/           0 prompt templates",
		"ws/                        Ephemeral workspace root",
		"AGENTS.md                  Project instructions copied into every agent workspace",
		"CLAUDE.md                  Imports AGENTS.md for Claude Code",
		".gitignore                 Ignores Wave runtime state",
		"Pipelines: ops-hello-world, plan-task",
		"Missing project commands: lint, build",
		"git add wave.yaml .agents AGENTS.md CLAUDE.md .gitignore",
		"Run 'wave validate --manifest app/wave.yaml' to check configuration",
	} {
		assert.Contains(t, text, want)
	}
	assert.NotContains(t, text, "git commit")

	res.Committed = true
	res.Project = map[string]interface{}{"test_command": "t", "lint_command": "l", "build_command": "b"}
	out.Reset()
	PrintInitSuccess(&out, "wave.yaml", "", res)
	text = out.String()
	assert.Contains(t, text, "git commit                 chore: initialize wave project")
	assert.NotContains(t, text, "Ephemeral workspace root", "the default root is not created by init")
	assert.Contains(t, text, "Workspaces: "+wsroot.DefaultRoot()+"/ (outside the checkout")
	assert.NotContains(t, text, "Project root:")
	assert.NotContains(t, text, "Missing project commands")
	assert.NotContains(t, text, "git add", "committed setups need no manual commit hint")
	assert.Contains(t, text, "Run 'wave validate' to check configuration")
}

func TestPrintMergeAndReconfigureSuccess(t *testing.T) {
	var out bytes.Buffer
	PrintMergeSuccess(&out, "wave.yaml")
	assert.Contains(t, out.String(), "Configuration merged successfully!")
	assert.Contains(t, out.String(), "wave.yaml       Preserved your settings")

	out.Reset()
	PrintReconfigureSuccess(&out, "x/wave.yaml", "x/wave.yaml.bak", "claude", "sonnet")
	text := out.String()
	assert.Contains(t, text, "Configuration updated successfully!")
	assert.Contains(t, text, "Adapter: claude (model sonnet for every tier)")
	assert.Contains(t, text, "Previous manifest saved to x/wave.yaml.bak")
	assert.Contains(t, text, "wave validate --manifest x/wave.yaml")

	out.Reset()
	PrintReconfigureSuccess(&out, "wave.yaml", "", "codex", "o3")
	assert.NotContains(t, out.String(), "Previous manifest saved")
}

func TestSuggestFirstRun(t *testing.T) {
	var out bytes.Buffer
	SuggestFirstRun(&out, nil)
	assert.Contains(t, out.String(), `wave meta "<task>"`)

	out.Reset()
	SuggestFirstRun(&out, &FlavourInfo{SourceGlob: "*.go"})
	assert.Contains(t, out.String(), "wave run audit-architecture")
}

func TestBuildDefaultManifest(t *testing.T) {
	personas := map[string]manifest.Persona{
		"navigator": {
			Description: "maps the codebase",
			Temperature: 0.1,
			Model:       "haiku",
			Permissions: manifest.Permissions{AllowedTools: []string{"Read"}, Deny: []string{"Write(*)"}},
		},
		"craftsman": {Description: "writes code"},
	}
	m := BuildDefaultManifest("claude", "ws", map[string]interface{}{"flavour": "go"}, personas)

	assert.Equal(t, "WaveManifest", m["kind"])
	assert.Equal(t, map[string]interface{}{"flavour": "go"}, m["project"])
	assert.Equal(t, "ws", m["runtime"].(map[string]interface{})["workspace_root"])
	claude := m["adapters"].(map[string]interface{})["claude"].(map[string]interface{})
	assert.Equal(t, "sonnet", claude["default_model"])
	assert.Equal(t, []string{"AGENTS.md"}, claude["project_files"])

	ps := m["personas"].(map[string]interface{})
	nav := ps["navigator"].(map[string]interface{})
	assert.Equal(t, "claude", nav["adapter"])
	assert.Equal(t, "maps the codebase", nav["description"])
	assert.Equal(t, ".agents/personas/navigator.md", nav["system_prompt_file"])
	assert.Equal(t, 0.1, nav["temperature"])
	assert.Equal(t, "haiku", nav["model"])
	assert.Equal(t, map[string]interface{}{"allowed_tools": []string{"Read"}, "deny": []string{"Write(*)"}}, nav["permissions"])
	assert.NotContains(t, ps["craftsman"], "model", "no model key unless the persona pins one")

	unknown := BuildManifest(ManifestOptions{Adapter: "gemini"})
	gemini := unknown["adapters"].(map[string]interface{})["gemini"].(map[string]interface{})
	assert.NotContains(t, gemini, "default_model", "adapters wave init does not scaffold get no pinned model")
	assert.NotContains(t, gemini, "tier_models")
}

func TestFilterTransitiveDeps(t *testing.T) {
	pipelines := map[string]string{
		"impl.yaml": `kind: WavePipeline
metadata:
  name: impl
steps:
  - id: fetch
    persona: "{{ forge.type }}-analyst"
    exec:
      type: prompt
      source_path: .agents/prompts/impl/fetch.md
    handover:
      contract:
        type: json_schema
        schema_path: .agents/contracts/assessment.schema.json
      compaction:
        persona: summarizer
  - id: build
    persona: craftsman
    exec:
      type: prompt
      source_path: prompts/outside.md
    handover:
      contract:
        type: json_schema
        schema_path: .agents/contracts/missing.schema.json
`,
		"broken.yaml": "steps: [",
	}
	contracts := map[string]string{"assessment.schema.json": "{}", "unused.schema.json": "{}"}
	prompts := map[string]string{"impl/fetch.md": "fetch", "other.md": "other"}
	personas := map[string]manifest.Persona{}
	for _, n := range []string{"github-analyst", "gitlab-analyst", "craftsman", "navigator", "philosopher", "summarizer", "auditor"} {
		personas[n] = manifest.Persona{}
	}

	var warn bytes.Buffer
	gotContracts, gotPrompts, gotPersonas := FilterTransitiveDeps(&warn, pipelines, contracts, prompts, personas)

	assert.Equal(t, []string{"assessment.schema.json"}, sortedKeys(gotContracts))
	assert.Equal(t, []string{"impl/fetch.md"}, sortedKeys(gotPrompts), "only prompts under .agents/prompts/ are tracked")
	assert.Equal(t, []string{"craftsman", "github-analyst", "gitlab-analyst", "navigator", "philosopher", "summarizer"}, sortedKeys(gotPersonas),
		"forge-templated refs expand to every forge; system personas are always kept")
	assert.Contains(t, warn.String(), "failed to parse pipeline broken.yaml")
	assert.Contains(t, warn.String(), "references contract missing.schema.json which is not in embedded defaults")
}

func TestLoadAssets_ReleaseSubsetOfAll(t *testing.T) {
	release, err := LoadAssets(nil, AssetOptions{})
	require.NoError(t, err)
	all, err := LoadAssets(nil, AssetOptions{All: true})
	require.NoError(t, err)

	assert.NotEmpty(t, release.Pipelines)
	assert.Less(t, len(release.Pipelines), len(all.Pipelines), "bench-solve is not a release pipeline")
	for name := range release.Pipelines {
		assert.Contains(t, all.Pipelines, name)
	}
	assert.LessOrEqual(t, len(release.Contracts), len(all.Contracts))
	for _, sys := range []string{"navigator", "summarizer", "philosopher"} {
		assert.Contains(t, release.PersonaConfigs, sys)
	}
	assert.Equal(t, all.Personas, release.Personas, "prompt files are trimmed later by ScopeAssetsToForge")
}

func TestForgeScoping(t *testing.T) {
	configs := map[string]manifest.Persona{"github-analyst": {}, "gitlab-analyst": {}, "gitea-scoper": {}, "navigator": {}}

	assert.Equal(t, []string{"github-analyst", "navigator"}, sortedKeys(FilterPersonasByForge(configs, forge.ForgeGitHub)))
	assert.Equal(t, []string{"gitea-scoper", "navigator"}, sortedKeys(FilterPersonasByForge(configs, forge.ForgeCodeberg)), "Codeberg shares Gitea personas")
	assert.Equal(t, []string{"navigator"}, sortedKeys(FilterPersonasByForge(configs, forge.ForgeLocal)))
	assert.Equal(t, []string{"navigator"}, sortedKeys(FilterPersonasByForge(configs, "")))
	assert.Len(t, FilterPersonasByForge(configs, forge.ForgeUnknown), 4, "unknown hosts keep every variant")
	assert.Len(t, FilterPersonasByForge(configs, forge.ForgeForgejo), 4, "forges without a persona prefix keep every variant")

	assert.True(t, IsForgePersona("bitbucket-commenter"))
	assert.False(t, IsForgePersona("navigator"))

	assets := &AssetSet{
		PersonaConfigs: configs,
		Personas: map[string]string{
			"github-analyst.md": "", "gitlab-analyst.md": "", "navigator.md": "", "base-protocol.md": "", "orphan.md": "",
		},
	}
	ScopeAssetsToForge(assets, forge.ForgeGitLab, true)
	assert.Len(t, assets.PersonaConfigs, 4, "--all keeps every asset")

	ScopeAssetsToForge(assets, forge.ForgeGitLab, false)
	assert.Equal(t, []string{"gitlab-analyst", "navigator"}, sortedKeys(assets.PersonaConfigs))
	assert.Equal(t, []string{"base-protocol.md", "gitlab-analyst.md", "navigator.md"}, sortedKeys(assets.Personas))

	ScopeAssetsToForge(nil, forge.ForgeGitHub, false) // must not panic
}

func TestProjectCommandHelpers(t *testing.T) {
	assert.Nil(t, FlavourToProjectMap(nil))
	assert.Equal(t, map[string]interface{}{
		"flavour": "rust", "language": "rust", "test_command": "cargo test", "lint_command": "cargo clippy",
		"build_command": "cargo build", "format_command": "cargo fmt --check", "source_glob": "*.rs", "skill": "rust",
	}, FlavourToProjectMap(&FlavourInfo{
		Flavour: "rust", Language: "rust", TestCommand: "cargo test", LintCommand: "cargo clippy",
		BuildCommand: "cargo build", FormatCommand: "cargo fmt --check", SourceGlob: "*.rs", Skill: "rust",
	}))

	assert.Nil(t, ApplyProjectCommands(nil, ProjectCommands{}), "no commands leaves the project untouched")
	project := ApplyProjectCommands(nil, ProjectCommands{Test: "  make test ", Build: "make"})
	assert.Equal(t, map[string]interface{}{"test_command": "make test", "build_command": "make"}, project)
	assert.Equal(t, []string{"lint"}, MissingProjectCommands(project))
	assert.Equal(t, []string{"test", "lint", "build"}, MissingProjectCommands(map[string]interface{}{"test_command": "  "}))

	assert.Equal(t, "github", forgeMetadataValue(forge.ForgeGitHub))
	for _, ft := range []forge.ForgeType{forge.ForgeUnknown, forge.ForgeLocal, ""} {
		assert.Empty(t, forgeMetadataValue(ft))
	}

	dirs := waveDirsFor("tmp/ws")
	assert.Contains(t, dirs, "tmp/ws")
	assert.NotContains(t, dirs, ".agents/workspaces")
	assert.Len(t, dirs, len(DefaultWaveDirs)+1)
	assert.Equal(t, DefaultWaveDirs, waveDirsFor(""), "no --workspace: nothing extra is created in the checkout")
}

func TestSupportedInitAdapters_ReturnsCopy(t *testing.T) {
	got := SupportedInitAdapters()
	assert.Equal(t, []string{"claude", "codex"}, got)
	got[0] = "mutated"
	assert.Equal(t, "claude", SupportedInitAdapters()[0])
}

func TestExtractProjectMetadata_MoreEcosystems(t *testing.T) {
	cases := map[string]struct {
		file, content     string
		name, description string
	}{
		"composer": {"composer.json", `{"name": "acme/shop", "description": "A PHP shop"}`, "acme/shop", "A PHP shop"},
		"pubspec":  {"pubspec.yaml", "name: flutter_app\ndescription: 'A Flutter app'\ndependencies:\n  name: nested\n", "flutter_app", "A Flutter app"},
		"mix":      {"mix.exs", "def project do\n  [\n    app: :phoenix_app,\n    version: \"0.1.0\"\n  ]\nend\n", "phoenix_app", ""},
		"swift":    {"Package.swift", "let package = Package(\n    name: \"SwiftTool\",\n    targets: []\n)\n", "SwiftTool", ""},
	}
	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, tc.file), tc.content)
			meta := ExtractProjectMetadata(dir)
			assert.Equal(t, tc.name, meta.Name)
			assert.Equal(t, tc.description, meta.Description)
		})
	}

	// Malformed manifests fall back to the directory name.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "composer.json"), "{not json")
	writeFile(t, filepath.Join(dir, "mix.exs"), "app: \"string-not-atom\"\n")
	writeFile(t, filepath.Join(dir, "Package.swift"), "name: unquoted\nname: \"unterminated\n")
	assert.Equal(t, filepath.Base(dir), ExtractProjectMetadata(dir).Name)
}

func TestWriteAssetMap_NestedAndErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteAssetMap(dir, map[string]string{"gh/create-pr.md": "pr", "top.md": "top"}))
	data, err := os.ReadFile(filepath.Join(dir, "gh", "create-pr.md"))
	require.NoError(t, err)
	assert.Equal(t, "pr", string(data))

	// A file in place of the target directory makes writing fail.
	blocker := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	err = WriteAssetMap(blocker, map[string]string{"x.md": "x"})
	require.Error(t, err)

	err = EnsureWaveDirs([]string{filepath.Join(blocker, "sub")})
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "failed to create directory "))
}

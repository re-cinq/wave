package onboarding

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeLookPath(installed ...string) LookPathFunc {
	return func(bin string) (string, error) {
		for _, i := range installed {
			if i == bin {
				return "/bin/" + bin, nil
			}
		}
		return "", errors.New("not found")
	}
}

func TestDefaultInitAdapter(t *testing.T) {
	assert.Equal(t, "claude", DefaultInitAdapter(fakeLookPath()))
	assert.Equal(t, "codex", DefaultInitAdapter(fakeLookPath("codex")))
	assert.Equal(t, "claude", DefaultInitAdapter(fakeLookPath("codex", "claude")))
	assert.Equal(t, []string{"codex"}, DetectInstalledAdapters(fakeLookPath("codex", "opencode")))
}

// Regression: codex defaulted to o3 and opencode/gemini were always written.
func TestBuildManifest_SingleAdapterPinnedTiers(t *testing.T) {
	for adapter, model := range map[string]string{"claude": "sonnet", "codex": "gpt-6.2-sol"} {
		m := BuildManifest(ManifestOptions{Adapter: adapter})
		adapters := m["adapters"].(map[string]interface{})
		require.Len(t, adapters, 1, adapter)
		entry := adapters[adapter].(map[string]interface{})
		assert.Equal(t, model, entry["default_model"])
		assert.Equal(t, TierModelsFor(model), entry["tier_models"])
		_, hasProject := m["project"]
		assert.False(t, hasProject, "no empty project block")
	}
	assert.Error(t, ValidateInitAdapter("opencode"))
	assert.Error(t, ValidateInitAdapter("o3"))
	assert.NoError(t, ValidateInitAdapter("codex"))
}

func TestAdapterAndModelOfManifest(t *testing.T) {
	m := map[string]interface{}{
		"adapters": map[string]interface{}{"codex": map[string]interface{}{"default_model": "o3"}},
		"personas": map[string]interface{}{
			"a": map[string]interface{}{"adapter": "codex"},
			"b": map[string]interface{}{"adapter": "codex"},
			"c": map[string]interface{}{"adapter": "claude"},
		},
	}
	assert.Equal(t, "codex", AdapterOfManifest(m))
	assert.Equal(t, "o3", ModelOfManifest(m, "codex"))
	assert.Equal(t, "", AdapterOfManifest(map[string]interface{}{}))
}

func TestApplyReconfigure(t *testing.T) {
	m := map[string]interface{}{
		"adapters": map[string]interface{}{"codex": map[string]interface{}{"binary": "codex", "default_model": "o3"}},
		"personas": map[string]interface{}{
			"a":      map[string]interface{}{"adapter": "codex"},
			"custom": map[string]interface{}{"adapter": "my-llm"},
		},
	}
	out := ApplyReconfigure(m, "codex", "claude", "sonnet", ProjectCommands{Test: "make test"})
	claude := out["adapters"].(map[string]interface{})["claude"].(map[string]interface{})
	assert.Equal(t, "sonnet", claude["default_model"])
	personas := out["personas"].(map[string]interface{})
	assert.Equal(t, "claude", personas["a"].(map[string]interface{})["adapter"])
	assert.Equal(t, "my-llm", personas["custom"].(map[string]interface{})["adapter"], "personas on other adapters are left alone")
	assert.Equal(t, "make test", out["project"].(map[string]interface{})["test_command"])
	assert.NotContains(t, out["adapters"], "codex", "unused previous adapter is removed")
}

func TestEnsureGitignoreIdempotent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n.agents/traces/"), 0o644))

	changed, err := EnsureGitignore(dir, ".agents/workspaces")
	require.NoError(t, err)
	assert.True(t, changed)
	data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	content := string(data)
	assert.True(t, strings.HasPrefix(content, "node_modules/\n.agents/traces/\n"))
	assert.Equal(t, 1, strings.Count(content, ".agents/traces/"))
	assert.Contains(t, content, ".agents/workspaces/\n")

	changed, err = EnsureGitignore(dir, ".agents/workspaces")
	require.NoError(t, err)
	assert.False(t, changed, "second run must not change .gitignore")
}

func TestCreateProjectInstructionFilesKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("mine"), 0o644))
	created, err := CreateProjectInstructionFiles(dir, "codex", ProjectMetadata{Name: "x"}, nil)
	require.NoError(t, err)
	assert.Empty(t, created, "codex reads AGENTS.md natively; existing AGENTS.md is kept")
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	assert.Equal(t, "mine", string(data))
	for _, f := range []string{"CLAUDE.md", "GEMINI.md", "CODEX.md"} {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.True(t, os.IsNotExist(err), f)
	}
}

func TestExtractProjectMetadata_Descriptions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"tinypy\"\ndescription = \"A tiny Python package\"\n")
	meta := ExtractProjectMetadata(dir)
	assert.Equal(t, "tinypy", meta.Name)
	assert.Equal(t, "A tiny Python package", meta.Description)

	dir = t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/calc\n")
	writeFile(t, filepath.Join(dir, "README.md"), "# calc\n\n[![ci](x)](y)\n\nA tiny calculator module.\n")
	meta = ExtractProjectMetadata(dir)
	assert.Equal(t, "calc", meta.Name)
	assert.Equal(t, "A tiny calculator module.", meta.Description)
}

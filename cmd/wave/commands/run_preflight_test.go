package commands

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A missing adapter binary is reported before any step runs, with a hint to
// switch adapters (regression: fork/exec /usr/local/bin/claude mid-step).
func TestCheckRunAdapters(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()
	installed := map[string]bool{"codex": true}
	lookPath = func(bin string) (string, error) {
		if installed[bin] {
			return "/bin/" + bin, nil
		}
		return "", exec.ErrNotFound
	}

	m := &manifest.Manifest{
		Adapters: map[string]manifest.Adapter{"claude": {Binary: "claude"}, "codex": {Binary: "codex"}},
		Personas: map[string]manifest.Persona{"craftsman": {Adapter: "claude"}},
	}
	p := &pipeline.Pipeline{Steps: []pipeline.Step{{ID: "greet", Persona: "craftsman"}}}

	err := checkRunAdapters(p, m, "")
	require.Error(t, err)
	var cliErr *CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, CodePreflightFailed, cliErr.Code)
	assert.Contains(t, cliErr.Message, "claude")
	assert.Contains(t, cliErr.Suggestion, "--adapter codex")

	assert.NoError(t, checkRunAdapters(p, m, "codex"), "--adapter override replaces the persona adapter")
}

// Regression: the preflight hint suggested '--adapter claude or --adapter
// codex' even when claude was the missing binary and codex was not
// installed. Only installed alternatives may be suggested.
func TestCheckRunAdapters_SuggestsOnlyInstalledAlternatives(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()
	p := &pipeline.Pipeline{Steps: []pipeline.Step{{ID: "greet", Persona: "craftsman"}}}
	suggestion := func(t *testing.T, installed map[string]bool, adapters map[string]manifest.Adapter) string {
		t.Helper()
		lookPath = func(bin string) (string, error) {
			if installed[bin] {
				return "/bin/" + bin, nil
			}
			return "", exec.ErrNotFound
		}
		m := &manifest.Manifest{
			Adapters: adapters,
			Personas: map[string]manifest.Persona{"craftsman": {Adapter: "claude"}},
		}
		err := checkRunAdapters(p, m, "")
		require.Error(t, err)
		var cliErr *CLIError
		require.True(t, errors.As(err, &cliErr))
		return cliErr.Suggestion
	}
	both := map[string]manifest.Adapter{"claude": {Binary: "claude"}, "codex": {Binary: "codex"}}
	claudeOnly := map[string]manifest.Adapter{"claude": {Binary: "claude"}}

	// Nothing installed: only "install", no adapter switch at all.
	s := suggestion(t, map[string]bool{}, both)
	assert.NotContains(t, s, "--adapter claude")
	assert.NotContains(t, s, "--adapter codex")
	assert.Contains(t, s, "Install the adapter CLI")

	// codex installed and configured: suggest it, never the missing claude.
	s = suggestion(t, map[string]bool{"codex": true}, both)
	assert.Contains(t, s, "--adapter codex")
	assert.NotContains(t, s, "--adapter claude")

	// codex installed but not in wave.yaml: --adapter codex would fail at
	// dispatch, so point at reconfigure instead.
	s = suggestion(t, map[string]bool{"codex": true}, claudeOnly)
	assert.Contains(t, s, "wave init --reconfigure --adapter codex")
	assert.NotContains(t, s, "run with --adapter")
}

// init's missing-adapter warning names only installed alternatives.
func TestWarnIfAdapterMissing_OnlyInstalledAlternatives(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()

	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	var buf bytes.Buffer
	warnIfAdapterMissing(&buf, "claude")
	assert.Contains(t, buf.String(), `"claude" not found on PATH`)
	assert.NotContains(t, buf.String(), "--adapter codex")

	lookPath = func(bin string) (string, error) {
		if bin == "codex" {
			return "/bin/codex", nil
		}
		return "", exec.ErrNotFound
	}
	buf.Reset()
	warnIfAdapterMissing(&buf, "claude")
	assert.Contains(t, buf.String(), "re-run init with --adapter codex")
}

func TestForgeFooter(t *testing.T) {
	assert.Equal(t, "Forge: local", forgeFooter("local", ""))
	assert.Equal(t, "Forge: github (acme/widget)", forgeFooter("github", "acme/widget"))
}

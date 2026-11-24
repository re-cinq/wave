package commands

import (
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

func TestForgeFooter(t *testing.T) {
	assert.Equal(t, "Forge: local", forgeFooter("local", ""))
	assert.Equal(t, "Forge: github (acme/widget)", forgeFooter("github", "acme/widget"))
}

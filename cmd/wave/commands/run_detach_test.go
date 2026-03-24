package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCmdHasDetachFlag(t *testing.T) {
	cmd := NewRunCmd()
	f := cmd.Flags().Lookup("detach")
	require.NotNil(t, f, "--detach flag should be registered")
	assert.Equal(t, "", f.Shorthand, "no shorthand — -d is taken by --debug")
	assert.Equal(t, "false", f.DefValue, "default should be false")
}

func TestBuildDetachEnv(t *testing.T) {
	// Set a known env var to verify passthrough.
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/usr/bin")

	env := buildDetachEnv()

	envMap := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}

	assert.Equal(t, "/home/test", envMap["HOME"])
	// PATH should include $HOME/.local/bin prepended for tool manager binaries
	assert.Equal(t, "/home/test/.local/bin:/usr/bin", envMap["PATH"])
	assert.Equal(t, "sk-test-key", envMap["ANTHROPIC_API_KEY"])
}

func TestBuildDetachEnvNoPathDuplication(t *testing.T) {
	// When $HOME/.local/bin is already in PATH, don't duplicate it.
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/home/test/.local/bin:/usr/bin")

	env := buildDetachEnv()
	envMap := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}
	assert.Equal(t, "/home/test/.local/bin:/usr/bin", envMap["PATH"])
}

func TestBuildDetachEnvOmitsMissing(t *testing.T) {
	// Unset optional vars to ensure they're omitted, not empty.
	os.Unsetenv("CLAUDE_CODE_USE_BEDROCK")
	os.Unsetenv("AWS_PROFILE")

	env := buildDetachEnv()

	for _, e := range env {
		assert.NotContains(t, e, "CLAUDE_CODE_USE_BEDROCK=")
		assert.NotContains(t, e, "AWS_PROFILE=")
	}
}

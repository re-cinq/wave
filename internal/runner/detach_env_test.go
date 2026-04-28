package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDetachEnv(t *testing.T) {
	// Set a known env var to verify passthrough.
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/usr/bin")

	env := BuildDetachEnv()

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

	env := BuildDetachEnv()
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

	env := BuildDetachEnv()

	for _, e := range env {
		assert.NotContains(t, e, "CLAUDE_CODE_USE_BEDROCK=")
		assert.NotContains(t, e, "AWS_PROFILE=")
	}
}

func TestBuildDetachEnvForwardsExtraVars(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("GH_TOKEN", "ghp_test")
	t.Setenv("GITHUB_TOKEN", "github_test")

	env := BuildDetachEnv("GH_TOKEN", "GITHUB_TOKEN")

	envMap := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}
	assert.Equal(t, "ghp_test", envMap["GH_TOKEN"])
	assert.Equal(t, "github_test", envMap["GITHUB_TOKEN"])
}

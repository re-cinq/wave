package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envSliceToMap parses a `KEY=VALUE` slice (as produced by buildDetachEnv)
// into a lookup map. Splits on the first `=` so values may contain `=`.
func envSliceToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, e := range env {
		if i := strings.IndexByte(e, '='); i >= 0 {
			out[e[:i]] = e[i+1:]
		}
	}
	return out
}

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
	envMap := envSliceToMap(env)

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
	envMap := envSliceToMap(env)
	assert.Equal(t, "/home/test/.local/bin:/usr/bin", envMap["PATH"])
}

// TestBuildDetachEnvHomeUnset verifies that when HOME is empty the PATH
// prepend is skipped (no `/.local/bin` artifact produced from an empty HOME).
func TestBuildDetachEnvHomeUnset(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	os.Unsetenv("HOME")

	env := buildDetachEnv()
	envMap := envSliceToMap(env)

	assert.Equal(t, "", envMap["HOME"])
	assert.Equal(t, "/usr/bin", envMap["PATH"], "PATH must not be prefixed when HOME is empty")
	assert.NotContains(t, envMap["PATH"], "/.local/bin", "no spurious .local/bin from empty HOME")
}

// TestBuildDetachEnvPassthroughVars exercises each named API/AWS/XDG/system
// var: when present it must appear in the result; when absent it must be
// omitted entirely (not emitted as KEY=).
func TestBuildDetachEnvPassthroughVars(t *testing.T) {
	passthrough := []string{
		"ANTHROPIC_API_KEY",
		"CLAUDE_CODE_USE_BEDROCK",
		"AWS_PROFILE",
		"AWS_REGION",
		"TERM",
		"USER",
		"SHELL",
		"XDG_DATA_HOME",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
	}

	for _, key := range passthrough {
		key := key
		t.Run(key+"_present", func(t *testing.T) {
			// Reset all passthrough vars first so this subtest only asserts
			// on the one we care about.
			for _, k := range passthrough {
				os.Unsetenv(k)
			}
			t.Setenv("HOME", "/home/test")
			t.Setenv("PATH", "/usr/bin")
			t.Setenv(key, "value-for-"+key)

			env := buildDetachEnv()
			envMap := envSliceToMap(env)

			got, ok := envMap[key]
			require.Truef(t, ok, "%s should be present in detach env", key)
			assert.Equalf(t, "value-for-"+key, got, "%s value must passthrough verbatim", key)
		})

		t.Run(key+"_absent", func(t *testing.T) {
			for _, k := range passthrough {
				os.Unsetenv(k)
			}
			t.Setenv("HOME", "/home/test")
			t.Setenv("PATH", "/usr/bin")

			env := buildDetachEnv()
			envMap := envSliceToMap(env)

			_, ok := envMap[key]
			assert.Falsef(t, ok, "%s must be omitted (not emitted as empty) when unset", key)
		})
	}
}

// TestBuildDetachEnvEmptyValuePassesThrough confirms that an explicitly-set
// empty string is treated as "present" (LookupEnv contract) and forwarded,
// distinguishing it from an unset var.
func TestBuildDetachEnvEmptyValuePassesThrough(t *testing.T) {
	os.Unsetenv("AWS_PROFILE")
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("AWS_REGION", "")

	env := buildDetachEnv()
	envMap := envSliceToMap(env)

	val, ok := envMap["AWS_REGION"]
	require.True(t, ok, "AWS_REGION set to empty string is still 'present' per LookupEnv")
	assert.Equal(t, "", val)
	_, awsProfileSet := envMap["AWS_PROFILE"]
	assert.False(t, awsProfileSet, "unset AWS_PROFILE must remain absent")
}

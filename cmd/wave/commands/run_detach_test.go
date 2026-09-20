package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureDetachedLaunch(t *testing.T, format string) (string, string, error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	errR, errW, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = outW, errW
	callErr := printDetachedLaunch(format, "impl-issue", "impl-issue-123")
	require.NoError(t, outW.Close())
	require.NoError(t, errW.Close())
	os.Stdout, os.Stderr = oldOut, oldErr
	defer outR.Close()
	defer errR.Close()
	var stdout, stderr bytes.Buffer
	_, _ = io.Copy(&stdout, outR)
	_, _ = io.Copy(&stderr, errR)
	return stdout.String(), stderr.String(), callErr
}

func TestRunCmdHasDetachFlag(t *testing.T) {
	cmd := NewRunCmd()
	f := cmd.Flags().Lookup("detach")
	require.NotNil(t, f, "--detach flag should be registered")
	assert.Equal(t, "", f.Shorthand, "no shorthand — -d is taken by --debug")
	assert.Equal(t, "false", f.DefValue, "default should be false")
}

func TestDetachedLaunchJSONIsStructured(t *testing.T) {
	stdout, stderr, err := captureDetachedLaunch(t, OutputFormatJSON)
	require.NoError(t, err)
	assert.Empty(t, stderr)
	var result DetachedLaunchResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	assert.Equal(t, DetachedLaunchResult{
		RunID: "impl-issue-123", PipelineName: "impl-issue", Status: "running", Detached: true,
	}, result)
}

func TestDetachedLaunchQuietIsSilent(t *testing.T) {
	stdout, stderr, err := captureDetachedLaunch(t, OutputFormatQuiet)
	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)
}

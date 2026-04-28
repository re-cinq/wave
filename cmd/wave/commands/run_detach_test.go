package commands

import (
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

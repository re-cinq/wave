package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServeCmd(t *testing.T) {
	cmd := NewServeCmd()
	require.NotNil(t, cmd)

	assert.Equal(t, "serve", cmd.Use)
	assert.Contains(t, cmd.Short, "dashboard")

	// Check flags
	portFlag := cmd.Flags().Lookup("port")
	require.NotNil(t, portFlag)
	assert.Equal(t, "8080", portFlag.DefValue)

	bindFlag := cmd.Flags().Lookup("bind")
	require.NotNil(t, bindFlag)
	assert.Equal(t, "127.0.0.1", bindFlag.DefValue)
}

func TestRunServeInvalidPort(t *testing.T) {
	testCases := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 70000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runServe(tc.port, "127.0.0.1")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid port")
		})
	}
}

func TestRunServeMissingDB(t *testing.T) {
	// Running without a state database should fail
	err := runServe(8080, "127.0.0.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state database not found")
}

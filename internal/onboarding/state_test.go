package onboarding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsOnboarded(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		{
			name:     "returns false when directory does not exist",
			setup:    func(t *testing.T, dir string) {},
			expected: false,
		},
		{
			name: "returns false when state file is missing",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(dir, 0755))
			},
			expected: false,
		},
		{
			name: "returns false when state file is corrupt",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".onboarded"), []byte("not json"), 0644))
			},
			expected: false,
		},
		{
			name: "returns false when completed is false",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(dir, 0755))
				state := State{Completed: false, Version: 1}
				data, _ := json.Marshal(state)
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".onboarded"), data, 0644))
			},
			expected: false,
		},
		{
			name: "returns true when completed is true",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(dir, 0755))
				state := State{Completed: true, Version: 1}
				data, _ := json.Marshal(state)
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".onboarded"), data, 0644))
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), ".agents")
			tt.setup(t, dir)
			assert.Equal(t, tt.expected, IsOnboarded(dir))
		})
	}
}

func TestClearOnboardingMissingFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agents")

	// Clearing when no file exists should not error
	err := ClearOnboarding(dir)
	assert.NoError(t, err)
}

func TestClearOnboardingRemovesState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agents")
	require.NoError(t, os.MkdirAll(dir, 0755))
	data, err := json.Marshal(State{Completed: true, Version: 1})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".onboarded"), data, 0644))
	require.True(t, IsOnboarded(dir))

	require.NoError(t, ClearOnboarding(dir))
	assert.False(t, IsOnboarded(dir))
	assert.NoFileExists(t, filepath.Join(dir, ".onboarded"))
}

func TestClearOnboardingReportsRemoveFailure(t *testing.T) {
	// A non-empty directory in place of the state file cannot be removed.
	dir := filepath.Join(t.TempDir(), ".agents")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".onboarded", "child"), 0755))

	assert.Error(t, ClearOnboarding(dir))
}

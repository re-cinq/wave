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
			dir := filepath.Join(t.TempDir(), ".wave")
			tt.setup(t, dir)
			assert.Equal(t, tt.expected, IsOnboarded(dir))
		})
	}
}

func TestMarkOnboarded(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".wave")

	err := MarkOnboarded(dir)
	require.NoError(t, err)

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(dir, ".onboarded"))
	require.NoError(t, err)

	var state State
	require.NoError(t, json.Unmarshal(data, &state))
	assert.True(t, state.Completed)
	assert.Equal(t, 1, state.Version)
	assert.False(t, state.CompletedAt.IsZero())
}

func TestMarkOnboardedRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".wave")

	// Mark as onboarded
	require.NoError(t, MarkOnboarded(dir))
	assert.True(t, IsOnboarded(dir))

	// Read state
	state, err := ReadState(dir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.True(t, state.Completed)
	assert.Equal(t, 1, state.Version)
}

func TestClearOnboarding(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".wave")

	// Mark then clear
	require.NoError(t, MarkOnboarded(dir))
	assert.True(t, IsOnboarded(dir))

	require.NoError(t, ClearOnboarding(dir))
	assert.False(t, IsOnboarded(dir))
}

func TestClearOnboardingMissingFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".wave")

	// Clearing when no file exists should not error
	err := ClearOnboarding(dir)
	assert.NoError(t, err)
}

func TestReadState(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string)
		expectNil bool
		expectErr bool
	}{
		{
			name:      "returns nil for missing file",
			setup:     func(t *testing.T, dir string) {},
			expectNil: true,
		},
		{
			name: "returns error for corrupt file",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".onboarded"), []byte("{invalid"), 0644))
			},
			expectErr: true,
		},
		{
			name: "returns state for valid file",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, MarkOnboarded(dir))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), ".wave")
			tt.setup(t, dir)

			state, err := ReadState(dir)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.expectNil {
				assert.Nil(t, state)
			} else {
				require.NotNil(t, state)
				assert.True(t, state.Completed)
			}
		})
	}
}

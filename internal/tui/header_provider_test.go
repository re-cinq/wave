package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: DefaultMetadataProvider implements MetadataProvider.
var _ MetadataProvider = (*DefaultMetadataProvider)(nil)

func TestDefaultMetadataProvider_FetchGitState_InRepo(t *testing.T) {
	// We are running inside a git worktree, so git commands should succeed.
	p := &DefaultMetadataProvider{}
	state, err := p.FetchGitState()
	require.NoError(t, err)

	assert.NotEmpty(t, state.Branch)
	assert.NotEqual(t, "[no git]", state.Branch)
	assert.NotEmpty(t, state.CommitHash)
	assert.NotEqual(t, "[no git]", state.CommitHash)
	// RemoteName may or may not be set depending on the worktree config.
}

func TestDefaultMetadataProvider_FetchGitState_NotARepo(t *testing.T) {
	// Run from a temp directory that is not a git repo.
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	p := &DefaultMetadataProvider{}
	state, err := p.FetchGitState()
	require.NoError(t, err)

	assert.Equal(t, "[no git]", state.Branch)
	assert.Equal(t, "[no git]", state.CommitHash)
	assert.False(t, state.IsDirty)
	assert.Empty(t, state.RemoteName)
}

func TestDefaultMetadataProvider_FetchManifestInfo_ValidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	content := `apiVersion: wave/v1alpha1
kind: Manifest
metadata:
  name: test-project
  repo: owner/test-repo
adapters: {}
runtime:
  workspace_root: .agents/workspaces
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0644))

	p := &DefaultMetadataProvider{ManifestPath: manifestPath}
	info, err := p.FetchManifestInfo()
	require.NoError(t, err)

	assert.Equal(t, "test-project", info.ProjectName)
	assert.Equal(t, "owner/test-repo", info.RepoName)
}

func TestDefaultMetadataProvider_FetchManifestInfo_MissingFile(t *testing.T) {
	p := &DefaultMetadataProvider{ManifestPath: "/nonexistent/wave.yaml"}
	info, err := p.FetchManifestInfo()
	require.NoError(t, err)

	assert.Equal(t, "[no project]", info.ProjectName)
	assert.Empty(t, info.RepoName)
}

func TestDefaultMetadataProvider_FetchManifestInfo_EmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "wave.yaml")

	content := `apiVersion: wave/v1alpha1
kind: Manifest
metadata:
  name: ""
  description: "no name set"
adapters: {}
runtime:
  workspace_root: .agents/workspaces
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0644))

	p := &DefaultMetadataProvider{ManifestPath: manifestPath}
	info, err := p.FetchManifestInfo()
	require.NoError(t, err)

	assert.Equal(t, "[no project]", info.ProjectName)
}

func TestDefaultMetadataProvider_FetchManifestInfo_DefaultPath(t *testing.T) {
	// When ManifestPath is empty, it defaults to "wave.yaml" — which won't
	// exist in the test's working directory (t.TempDir()), so we get the
	// placeholder.
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	p := &DefaultMetadataProvider{}
	info, err := p.FetchManifestInfo()
	require.NoError(t, err)

	assert.Equal(t, "[no project]", info.ProjectName)
}

func TestDefaultMetadataProvider_FetchGitHubInfo_EmptyRepo(t *testing.T) {
	p := &DefaultMetadataProvider{}
	info, err := p.FetchGitHubInfo("")
	require.NoError(t, err)

	assert.Equal(t, GitHubNotConfigured, info.AuthState)
	assert.Equal(t, 0, info.IssuesCount)
}

func TestDefaultMetadataProvider_FetchPipelineHealth_NoFunc(t *testing.T) {
	p := &DefaultMetadataProvider{}
	health, err := p.FetchPipelineHealth()
	require.NoError(t, err)

	assert.Equal(t, HealthOK, health)
}

func TestDefaultMetadataProvider_FetchPipelineHealth_WithFunc_OK(t *testing.T) {
	p := &DefaultMetadataProvider{
		HealthCheckFunc: func() (HealthStatus, error) {
			return HealthOK, nil
		},
	}
	health, err := p.FetchPipelineHealth()
	require.NoError(t, err)

	assert.Equal(t, HealthOK, health)
}

func TestDefaultMetadataProvider_FetchPipelineHealth_WithFunc_Warn(t *testing.T) {
	p := &DefaultMetadataProvider{
		HealthCheckFunc: func() (HealthStatus, error) {
			return HealthWarn, nil
		},
	}
	health, err := p.FetchPipelineHealth()
	require.NoError(t, err)

	assert.Equal(t, HealthWarn, health)
}

func TestDefaultMetadataProvider_FetchPipelineHealth_WithFunc_Err(t *testing.T) {
	p := &DefaultMetadataProvider{
		HealthCheckFunc: func() (HealthStatus, error) {
			return HealthErr, nil
		},
	}
	health, err := p.FetchPipelineHealth()
	require.NoError(t, err)

	assert.Equal(t, HealthErr, health)
}

func TestDefaultMetadataProvider_FetchPipelineHealth_WithFunc_Error(t *testing.T) {
	expectedErr := errors.New("db connection failed")
	p := &DefaultMetadataProvider{
		HealthCheckFunc: func() (HealthStatus, error) {
			return HealthOK, expectedErr
		},
	}
	_, err := p.FetchPipelineHealth()
	assert.ErrorIs(t, err, expectedErr)
}

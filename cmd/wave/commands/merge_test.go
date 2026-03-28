package commands

import (
	"testing"

	"github.com/recinq/wave/internal/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMergeCmd(t *testing.T) {
	cmd := NewMergeCmd()

	assert.Equal(t, "merge <PR-URL-or-number>", cmd.Use)
	assert.Contains(t, cmd.Short, "Merge")

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("all"), "all flag should exist")
	assert.NotNil(t, flags.Lookup("yes"), "yes flag should exist")
}

func TestNewMergeCmd_ArgValidation(t *testing.T) {
	t.Run("no args without --all", func(t *testing.T) {
		cmd := NewMergeCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires exactly 1 argument")
	})

	t.Run("args with --all", func(t *testing.T) {
		cmd := NewMergeCmd()
		cmd.SetArgs([]string{"--all", "123"})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--all does not accept arguments")
	})
}

func TestParsePRInput(t *testing.T) {
	fi := forge.ForgeInfo{
		Type:  forge.ForgeGitHub,
		Host:  "github.com",
		Owner: "owner",
		Repo:  "repo",
	}

	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:  "plain number",
			input: "123",
			want:  123,
		},
		{
			name:  "plain number with whitespace",
			input: "  456  ",
			want:  456,
		},
		{
			name:  "github PR URL",
			input: "https://github.com/owner/repo/pull/789",
			want:  789,
		},
		{
			name:  "gitlab MR URL",
			input: "https://gitlab.com/owner/repo/merge_requests/42",
			want:  42,
		},
		{
			name:  "gitea pulls URL",
			input: "https://gitea.example.com/owner/repo/pulls/55",
			want:  55,
		},
		{
			name:    "zero",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "negative",
			input:   "-1",
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "URL without PR path",
			input:   "https://github.com/owner/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePRInput(tt.input, fi)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPRURLPattern(t *testing.T) {
	tests := []struct {
		url     string
		matched bool
		number  string
	}{
		{"https://github.com/owner/repo/pull/123", true, "123"},
		{"https://gitlab.com/owner/repo/merge_requests/456", true, "456"},
		{"https://gitea.example.com/owner/repo/pulls/789", true, "789"},
		{"http://forgejo.local/user/project/pull/1", true, "1"},
		{"https://github.com/owner/repo/issues/123", false, ""},
		{"https://github.com/owner/repo", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			m := prURLPattern.FindStringSubmatch(tt.url)
			if tt.matched {
				require.NotNil(t, m, "expected URL to match: %s", tt.url)
				assert.Equal(t, tt.number, m[1])
			} else {
				assert.Nil(t, m, "expected URL not to match: %s", tt.url)
			}
		})
	}
}

func TestMergePRViaCLI_UnsupportedForge(t *testing.T) {
	fi := forge.ForgeInfo{
		Type: forge.ForgeUnknown,
	}
	err := mergePRViaCLI(fi, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported forge type")
}

func TestMergePRViaAPI_NoToken(t *testing.T) {
	// Use GitLab which relies purely on env vars (no CLI fallback).
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GL_TOKEN", "")

	fi := forge.ForgeInfo{
		Type:  forge.ForgeGitLab,
		Host:  "gitlab.com",
		Owner: "owner",
		Repo:  "repo",
	}
	err := mergePRViaAPI(fi, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API token found")
}

func TestMergePRViaAPI_UnsupportedForge(t *testing.T) {
	t.Setenv("BITBUCKET_TOKEN", "fake-token")
	fi := forge.ForgeInfo{
		Type: forge.ForgeBitbucket,
	}
	err := mergePRViaAPI(fi, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API fallback")
}

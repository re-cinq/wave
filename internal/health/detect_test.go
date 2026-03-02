package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType ForgeType
		wantRepo string
	}{
		{
			name:     "GitHub HTTPS",
			url:      "https://github.com/owner/repo.git",
			wantType: GitHub,
			wantRepo: "owner/repo",
		},
		{
			name:     "GitHub HTTPS without .git",
			url:      "https://github.com/owner/repo",
			wantType: GitHub,
			wantRepo: "owner/repo",
		},
		{
			name:     "GitHub SSH",
			url:      "git@github.com:owner/repo.git",
			wantType: GitHub,
			wantRepo: "owner/repo",
		},
		{
			name:     "GitHub SSH without .git",
			url:      "git@github.com:owner/repo",
			wantType: GitHub,
			wantRepo: "owner/repo",
		},
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/group/project.git",
			wantType: GitLab,
			wantRepo: "group/project",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:group/project.git",
			wantType: GitLab,
			wantRepo: "group/project",
		},
		{
			name:     "Bitbucket HTTPS",
			url:      "https://bitbucket.org/team/repo.git",
			wantType: Bitbucket,
			wantRepo: "team/repo",
		},
		{
			name:     "Bitbucket SSH",
			url:      "git@bitbucket.org:team/repo.git",
			wantType: Bitbucket,
			wantRepo: "team/repo",
		},
		{
			name:     "Codeberg HTTPS",
			url:      "https://codeberg.org/user/repo.git",
			wantType: Gitea,
			wantRepo: "user/repo",
		},
		{
			name:     "Self-hosted Gitea",
			url:      "https://gitea.example.com/org/repo.git",
			wantType: Gitea,
			wantRepo: "org/repo",
		},
		{
			name:     "Unknown forge",
			url:      "https://example.com/user/project.git",
			wantType: Unknown,
			wantRepo: "user/project",
		},
		{
			name:     "Empty URL",
			url:      "",
			wantType: Unknown,
			wantRepo: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotRepo := parseRemoteURL(tt.url)
			assert.Equal(t, tt.wantType, gotType, "forge type mismatch")
			assert.Equal(t, tt.wantRepo, gotRepo, "repo identifier mismatch")
		})
	}
}

func TestDetectForge(t *testing.T) {
	// Test against the current repository which should be a valid git repo.
	forgeType, repoID, err := DetectForge(".")

	assert.NoError(t, err, "DetectForge should not error on a valid git repo")
	assert.NotEmpty(t, repoID, "repository identifier should not be empty")
	assert.Contains(t, []ForgeType{GitHub, GitLab, Bitbucket, Gitea, Unknown}, forgeType,
		"forge type should be a valid ForgeType")
}

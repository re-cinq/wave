package platform

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_GitHubURLs(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
	}{
		{
			name:  "HTTPS",
			url:   "https://github.com/re-cinq/wave",
			owner: "re-cinq",
			repo:  "wave",
		},
		{
			name:  "HTTPS with .git suffix",
			url:   "https://github.com/re-cinq/wave.git",
			owner: "re-cinq",
			repo:  "wave",
		},
		{
			name:  "SSH colon syntax",
			url:   "git@github.com:re-cinq/wave.git",
			owner: "re-cinq",
			repo:  "wave",
		},
		{
			name:  "SSH colon syntax without .git",
			url:   "git@github.com:re-cinq/wave",
			owner: "re-cinq",
			repo:  "wave",
		},
		{
			name:  "SSH scheme",
			url:   "ssh://git@github.com/re-cinq/wave.git",
			owner: "re-cinq",
			repo:  "wave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, PlatformGitHub, profile.Type)
			assert.Equal(t, tt.owner, profile.Owner)
			assert.Equal(t, tt.repo, profile.Repo)
			assert.Equal(t, "https://api.github.com", profile.APIURL)
			assert.Equal(t, "gh", profile.CLITool)
			assert.Equal(t, "gh", profile.PipelineFamily)
		})
	}
}

func TestDetect_GitLabURLs(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
	}{
		{
			name:  "HTTPS",
			url:   "https://gitlab.com/mygroup/myproject",
			owner: "mygroup",
			repo:  "myproject",
		},
		{
			name:  "HTTPS with .git suffix",
			url:   "https://gitlab.com/mygroup/myproject.git",
			owner: "mygroup",
			repo:  "myproject",
		},
		{
			name:  "SSH colon syntax",
			url:   "git@gitlab.com:mygroup/myproject.git",
			owner: "mygroup",
			repo:  "myproject",
		},
		{
			name:  "SSH scheme",
			url:   "ssh://git@gitlab.com/mygroup/myproject.git",
			owner: "mygroup",
			repo:  "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, PlatformGitLab, profile.Type)
			assert.Equal(t, tt.owner, profile.Owner)
			assert.Equal(t, tt.repo, profile.Repo)
			assert.Equal(t, "https://gitlab.com/api/v4", profile.APIURL)
			assert.Equal(t, "glab", profile.CLITool)
			assert.Equal(t, "gl", profile.PipelineFamily)
		})
	}
}

func TestDetect_GitLabSubgroups(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
	}{
		{
			name:  "HTTPS two-level subgroup",
			url:   "https://gitlab.com/group/subgroup/myproject.git",
			owner: "group/subgroup",
			repo:  "myproject",
		},
		{
			name:  "HTTPS three-level subgroup",
			url:   "https://gitlab.com/org/team/project/myrepo.git",
			owner: "org/team/project",
			repo:  "myrepo",
		},
		{
			name:  "SSH two-level subgroup",
			url:   "git@gitlab.com:group/subgroup/myproject.git",
			owner: "group/subgroup",
			repo:  "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, PlatformGitLab, profile.Type)
			assert.Equal(t, tt.owner, profile.Owner)
			assert.Equal(t, tt.repo, profile.Repo)
			assert.Equal(t, "gl", profile.PipelineFamily)
		})
	}
}

func TestDetect_BitbucketURLs(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
	}{
		{
			name:  "HTTPS",
			url:   "https://bitbucket.org/myteam/myrepo",
			owner: "myteam",
			repo:  "myrepo",
		},
		{
			name:  "HTTPS with .git suffix",
			url:   "https://bitbucket.org/myteam/myrepo.git",
			owner: "myteam",
			repo:  "myrepo",
		},
		{
			name:  "SSH colon syntax",
			url:   "git@bitbucket.org:myteam/myrepo.git",
			owner: "myteam",
			repo:  "myrepo",
		},
		{
			name:  "SSH scheme",
			url:   "ssh://git@bitbucket.org/myteam/myrepo.git",
			owner: "myteam",
			repo:  "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, PlatformBitbucket, profile.Type)
			assert.Equal(t, tt.owner, profile.Owner)
			assert.Equal(t, tt.repo, profile.Repo)
			assert.Equal(t, "https://api.bitbucket.org/2.0", profile.APIURL)
			assert.Equal(t, "bb", profile.CLITool)
			assert.Equal(t, "bb", profile.PipelineFamily)
		})
	}
}

func TestDetect_GiteaURLs(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
	}{
		{
			name:  "HTTPS with gitea in hostname",
			url:   "https://gitea.example.com/myuser/myrepo",
			owner: "myuser",
			repo:  "myrepo",
		},
		{
			name:  "HTTPS with gitea in hostname and .git suffix",
			url:   "https://gitea.example.com/myuser/myrepo.git",
			owner: "myuser",
			repo:  "myrepo",
		},
		{
			name:  "SSH with gitea in hostname",
			url:   "git@gitea.example.com:myuser/myrepo.git",
			owner: "myuser",
			repo:  "myrepo",
		},
		{
			name:  "HTTPS with try.gitea.io",
			url:   "https://try.gitea.io/myuser/myrepo",
			owner: "myuser",
			repo:  "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, PlatformGitea, profile.Type)
			assert.Equal(t, tt.owner, profile.Owner)
			assert.Equal(t, tt.repo, profile.Repo)
			assert.Equal(t, "tea", profile.CLITool)
			assert.Equal(t, "gt", profile.PipelineFamily)
		})
	}
}

func TestDetect_SelfHostedURLs(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType PlatformType
		family   string
	}{
		{
			name:     "self-hosted GitLab",
			url:      "https://gitlab.mycompany.com/team/project",
			wantType: PlatformGitLab,
			family:   "gl",
		},
		{
			name:     "self-hosted GitHub Enterprise",
			url:      "https://github.mycompany.com/team/project",
			wantType: PlatformGitHub,
			family:   "gh",
		},
		{
			name:     "self-hosted Bitbucket",
			url:      "https://bitbucket.mycompany.com/team/project",
			wantType: PlatformBitbucket,
			family:   "bb",
		},
		{
			name:     "self-hosted Gitea",
			url:      "https://gitea.mycompany.com/team/project",
			wantType: PlatformGitea,
			family:   "gt",
		},
		{
			name:     "unknown self-hosted",
			url:      "https://git.example.com/team/project",
			wantType: PlatformUnknown,
			family:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, tt.wantType, profile.Type)
			assert.Equal(t, tt.family, profile.PipelineFamily)
		})
	}
}

func TestDetect_PortNumbers(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		owner string
		repo  string
	}{
		{
			name:  "HTTPS with port",
			url:   "https://github.com:443/owner/repo",
			owner: "owner",
			repo:  "repo",
		},
		{
			name:  "SSH with port via scheme",
			url:   "ssh://git@github.com:2222/owner/repo.git",
			owner: "owner",
			repo:  "repo",
		},
		{
			name:  "HTTP with custom port",
			url:   "http://gitlab.example.com:8080/mygroup/myproject.git",
			owner: "mygroup",
			repo:  "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, tt.owner, profile.Owner)
			assert.Equal(t, tt.repo, profile.Repo)
		})
	}
}

func TestDetect_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType PlatformType
		owner    string
		repo     string
	}{
		{
			name:     "empty string",
			url:      "",
			wantType: PlatformUnknown,
		},
		{
			name:     "whitespace only",
			url:      "   ",
			wantType: PlatformUnknown,
		},
		{
			name:     "not a URL at all",
			url:      "not-a-url",
			wantType: PlatformUnknown,
		},
		{
			name:     "single segment path",
			url:      "https://github.com/just-owner",
			wantType: PlatformUnknown,
		},
		{
			name:     "URL with trailing whitespace",
			url:      "  https://github.com/owner/repo  ",
			wantType: PlatformGitHub,
			owner:    "owner",
			repo:     "repo",
		},
		{
			name:     "double .git suffix stripped once",
			url:      "https://github.com/owner/repo.git",
			wantType: PlatformGitHub,
			owner:    "owner",
			repo:     "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, tt.wantType, profile.Type)
			if tt.owner != "" {
				assert.Equal(t, tt.owner, profile.Owner)
			}
			if tt.repo != "" {
				assert.Equal(t, tt.repo, profile.Repo)
			}
		})
	}
}

func TestDetect_PipelineFamilyMapping(t *testing.T) {
	tests := []struct {
		url            string
		pipelineFamily string
	}{
		{"https://github.com/o/r", "gh"},
		{"https://gitlab.com/o/r", "gl"},
		{"https://bitbucket.org/o/r", "bb"},
		{"https://gitea.example.com/o/r", "gt"},
		{"https://git.example.com/o/r", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.pipelineFamily, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, tt.pipelineFamily, profile.PipelineFamily)
		})
	}
}

func TestDetectFromGit_OriginRemote(t *testing.T) {
	mockRunner := func(args ...string) ([]byte, error) {
		output := "origin\thttps://github.com/re-cinq/wave.git (fetch)\n" +
			"origin\thttps://github.com/re-cinq/wave.git (push)\n"
		return []byte(output), nil
	}

	profile, err := detectFromGitWith(mockRunner)
	require.NoError(t, err)
	assert.Equal(t, PlatformGitHub, profile.Type)
	assert.Equal(t, "re-cinq", profile.Owner)
	assert.Equal(t, "wave", profile.Repo)
	assert.Equal(t, "gh", profile.PipelineFamily)
	assert.Empty(t, profile.AdditionalRemotes)
}

func TestDetectFromGit_MultipleRemotes(t *testing.T) {
	mockRunner := func(args ...string) ([]byte, error) {
		output := "origin\thttps://github.com/re-cinq/wave.git (fetch)\n" +
			"origin\thttps://github.com/re-cinq/wave.git (push)\n" +
			"upstream\thttps://gitlab.com/upstream/wave.git (fetch)\n" +
			"upstream\thttps://gitlab.com/upstream/wave.git (push)\n" +
			"backup\tgit@bitbucket.org:backup/wave.git (fetch)\n" +
			"backup\tgit@bitbucket.org:backup/wave.git (push)\n"
		return []byte(output), nil
	}

	profile, err := detectFromGitWith(mockRunner)
	require.NoError(t, err)
	assert.Equal(t, PlatformGitHub, profile.Type)
	assert.Equal(t, "re-cinq", profile.Owner)
	assert.Equal(t, "wave", profile.Repo)
	assert.Len(t, profile.AdditionalRemotes, 2)

	// Verify additional remotes are correctly identified
	assert.Equal(t, "upstream", profile.AdditionalRemotes[0].Name)
	assert.Equal(t, PlatformGitLab, profile.AdditionalRemotes[0].Platform)
	assert.Equal(t, "backup", profile.AdditionalRemotes[1].Name)
	assert.Equal(t, PlatformBitbucket, profile.AdditionalRemotes[1].Platform)
}

func TestDetectFromGit_NoOriginFallsBackToFirstRemote(t *testing.T) {
	mockRunner := func(args ...string) ([]byte, error) {
		output := "upstream\thttps://gitlab.com/myorg/myrepo.git (fetch)\n" +
			"upstream\thttps://gitlab.com/myorg/myrepo.git (push)\n" +
			"fork\thttps://github.com/user/myrepo.git (fetch)\n" +
			"fork\thttps://github.com/user/myrepo.git (push)\n"
		return []byte(output), nil
	}

	profile, err := detectFromGitWith(mockRunner)
	require.NoError(t, err)
	assert.Equal(t, PlatformGitLab, profile.Type)
	assert.Equal(t, "myorg", profile.Owner)
	assert.Equal(t, "myrepo", profile.Repo)
	assert.Len(t, profile.AdditionalRemotes, 1)
	assert.Equal(t, "fork", profile.AdditionalRemotes[0].Name)
}

func TestDetectFromGit_NoRemotes(t *testing.T) {
	mockRunner := func(args ...string) ([]byte, error) {
		return []byte(""), nil
	}

	_, err := detectFromGitWith(mockRunner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no git remotes configured")
}

func TestDetectFromGit_GitCommandFails(t *testing.T) {
	mockRunner := func(args ...string) ([]byte, error) {
		return nil, fmt.Errorf("fatal: not a git repository")
	}

	profile, err := detectFromGitWith(mockRunner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list git remotes")
	assert.Equal(t, PlatformUnknown, profile.Type)
}

func TestDetect_APIURLs(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantAPI   string
	}{
		{
			name:    "GitHub uses fixed API URL",
			url:     "https://github.com/o/r",
			wantAPI: "https://api.github.com",
		},
		{
			name:    "GitLab uses host-based API URL",
			url:     "https://gitlab.com/o/r",
			wantAPI: "https://gitlab.com/api/v4",
		},
		{
			name:    "Self-hosted GitLab uses host-based API URL",
			url:     "https://gitlab.mycompany.com/o/r",
			wantAPI: "https://gitlab.mycompany.com/api/v4",
		},
		{
			name:    "Bitbucket uses fixed API URL",
			url:     "https://bitbucket.org/o/r",
			wantAPI: "https://api.bitbucket.org/2.0",
		},
		{
			name:    "Gitea uses host-based API URL",
			url:     "https://gitea.example.com/o/r",
			wantAPI: "https://gitea.example.com/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, tt.wantAPI, profile.APIURL)
		})
	}
}

func TestDetect_CLITools(t *testing.T) {
	tests := []struct {
		url      string
		wantTool string
	}{
		{"https://github.com/o/r", "gh"},
		{"https://gitlab.com/o/r", "glab"},
		{"https://bitbucket.org/o/r", "bb"},
		{"https://gitea.example.com/o/r", "tea"},
	}

	for _, tt := range tests {
		t.Run(tt.wantTool, func(t *testing.T) {
			profile := Detect(tt.url)
			assert.Equal(t, tt.wantTool, profile.CLITool)
		})
	}
}

func TestParseGitRemoteOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    []RemoteInfo
	}{
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
		{
			name: "single remote fetch and push",
			output: "origin\thttps://github.com/owner/repo.git (fetch)\n" +
				"origin\thttps://github.com/owner/repo.git (push)\n",
			want: []RemoteInfo{
				{Name: "origin", URL: "https://github.com/owner/repo.git", Platform: PlatformGitHub},
			},
		},
		{
			name: "multiple remotes",
			output: "origin\thttps://github.com/owner/repo.git (fetch)\n" +
				"origin\thttps://github.com/owner/repo.git (push)\n" +
				"upstream\thttps://gitlab.com/upstream/repo.git (fetch)\n" +
				"upstream\thttps://gitlab.com/upstream/repo.git (push)\n",
			want: []RemoteInfo{
				{Name: "origin", URL: "https://github.com/owner/repo.git", Platform: PlatformGitHub},
				{Name: "upstream", URL: "https://gitlab.com/upstream/repo.git", Platform: PlatformGitLab},
			},
		},
		{
			name:   "whitespace-only lines ignored",
			output: "  \n\norigin\thttps://github.com/o/r.git (fetch)\n  \n",
			want: []RemoteInfo{
				{Name: "origin", URL: "https://github.com/o/r.git", Platform: PlatformGitHub},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitRemoteOutput([]byte(tt.output))
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIdentifyPlatform(t *testing.T) {
	tests := []struct {
		host     string
		wantType PlatformType
	}{
		{"github.com", PlatformGitHub},
		{"gitlab.com", PlatformGitLab},
		{"bitbucket.org", PlatformBitbucket},
		{"gitlab.mycompany.com", PlatformGitLab},
		{"gitea.mycompany.com", PlatformGitea},
		{"bitbucket.mycompany.com", PlatformBitbucket},
		{"github.enterprise.com", PlatformGitHub},
		{"git.example.com", PlatformUnknown},
		{"my-server.local", PlatformUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := identifyPlatform(tt.host)
			assert.Equal(t, tt.wantType, got)
		})
	}
}

func TestCleanRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"repo", "repo"},
		{"repo.git", "repo"},
		{"  repo  ", "repo"},
		{"  repo.git  ", "repo"},
		{"repo.git.git", "repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, cleanRepoName(tt.input))
		})
	}
}

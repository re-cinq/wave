package forge

import (
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType ForgeType
		wantHost string
		wantOwner string
		wantRepo  string
		wantCLI   string
		wantPrefix string
	}{
		{
			name:       "GitHub HTTPS",
			url:        "https://github.com/recinq/wave.git",
			wantType:   ForgeGitHub,
			wantHost:   "github.com",
			wantOwner:  "recinq",
			wantRepo:   "wave",
			wantCLI:    "gh",
			wantPrefix: "gh",
		},
		{
			name:       "GitHub SSH",
			url:        "git@github.com:recinq/wave.git",
			wantType:   ForgeGitHub,
			wantHost:   "github.com",
			wantOwner:  "recinq",
			wantRepo:   "wave",
			wantCLI:    "gh",
			wantPrefix: "gh",
		},
		{
			name:       "GitHub HTTPS without .git",
			url:        "https://github.com/recinq/wave",
			wantType:   ForgeGitHub,
			wantHost:   "github.com",
			wantOwner:  "recinq",
			wantRepo:   "wave",
			wantCLI:    "gh",
			wantPrefix: "gh",
		},
		{
			name:       "GitLab HTTPS",
			url:        "https://gitlab.com/myorg/myrepo.git",
			wantType:   ForgeGitLab,
			wantHost:   "gitlab.com",
			wantOwner:  "myorg",
			wantRepo:   "myrepo",
			wantCLI:    "glab",
			wantPrefix: "gl",
		},
		{
			name:       "GitLab SSH",
			url:        "git@gitlab.com:myorg/myrepo.git",
			wantType:   ForgeGitLab,
			wantHost:   "gitlab.com",
			wantOwner:  "myorg",
			wantRepo:   "myrepo",
			wantCLI:    "glab",
			wantPrefix: "gl",
		},
		{
			name:       "Bitbucket HTTPS",
			url:        "https://bitbucket.org/team/project.git",
			wantType:   ForgeBitbucket,
			wantHost:   "bitbucket.org",
			wantOwner:  "team",
			wantRepo:   "project",
			wantCLI:    "bb",
			wantPrefix: "bb",
		},
		{
			name:       "Bitbucket SSH",
			url:        "git@bitbucket.org:team/project.git",
			wantType:   ForgeBitbucket,
			wantHost:   "bitbucket.org",
			wantOwner:  "team",
			wantRepo:   "project",
			wantCLI:    "bb",
			wantPrefix: "bb",
		},
		{
			name:       "Gitea self-hosted",
			url:        "https://gitea.example.com/user/repo.git",
			wantType:   ForgeGitea,
			wantHost:   "gitea.example.com",
			wantOwner:  "user",
			wantRepo:   "repo",
			wantCLI:    "tea",
			wantPrefix: "gt",
		},
		{
			name:       "Self-hosted unknown",
			url:        "https://git.corp.com/team/app.git",
			wantType:   ForgeUnknown,
			wantHost:   "git.corp.com",
			wantOwner:  "team",
			wantRepo:   "app",
			wantCLI:    "",
			wantPrefix: "",
		},
		{
			name:       "SSH with ssh:// prefix",
			url:        "ssh://git@github.com/recinq/wave.git",
			wantType:   ForgeGitHub,
			wantHost:   "github.com",
			wantOwner:  "recinq",
			wantRepo:   "wave",
			wantCLI:    "gh",
			wantPrefix: "gh",
		},
		{
			name:     "Empty URL",
			url:      "",
			wantType: ForgeUnknown,
		},
		{
			name:     "Malformed URL",
			url:      "not-a-url",
			wantType: ForgeUnknown,
		},
		{
			name:       "HTTP URL (upgrades)",
			url:        "http://github.com/recinq/wave.git",
			wantType:   ForgeGitHub,
			wantHost:   "github.com",
			wantOwner:  "recinq",
			wantRepo:   "wave",
			wantCLI:    "gh",
			wantPrefix: "gh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.url)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", got.Host, tt.wantHost)
			}
			if got.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.wantRepo)
			}
			if got.CLITool != tt.wantCLI {
				t.Errorf("CLITool = %q, want %q", got.CLITool, tt.wantCLI)
			}
			if got.PipelinePrefix != tt.wantPrefix {
				t.Errorf("PipelinePrefix = %q, want %q", got.PipelinePrefix, tt.wantPrefix)
			}
		})
	}
}

func TestForgeInfo_Slug(t *testing.T) {
	tests := []struct {
		name  string
		info  ForgeInfo
		want  string
	}{
		{
			name: "both set",
			info: ForgeInfo{Owner: "recinq", Repo: "wave"},
			want: "recinq/wave",
		},
		{
			name: "owner only",
			info: ForgeInfo{Owner: "recinq"},
			want: "",
		},
		{
			name: "repo only",
			info: ForgeInfo{Repo: "wave"},
			want: "",
		},
		{
			name: "neither set",
			info: ForgeInfo{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.Slug(); got != tt.want {
				t.Errorf("Slug() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterPipelinesByForge(t *testing.T) {
	pipelines := []string{
		"gh-implement",
		"gh-review",
		"gl-deploy",
		"bb-build",
		"gt-test",
		"plan-speckit",
		"wave-evolve",
		"impl-debug",
	}

	tests := []struct {
		name      string
		forgeType ForgeType
		want      []string
	}{
		{
			name:      "GitHub forge filters to gh- and generic",
			forgeType: ForgeGitHub,
			want:      []string{"gh-implement", "gh-review", "plan-speckit", "wave-evolve", "impl-debug"},
		},
		{
			name:      "GitLab forge filters to gl- and generic",
			forgeType: ForgeGitLab,
			want:      []string{"gl-deploy", "plan-speckit", "wave-evolve", "impl-debug"},
		},
		{
			name:      "Bitbucket forge filters to bb- and generic",
			forgeType: ForgeBitbucket,
			want:      []string{"bb-build", "plan-speckit", "wave-evolve", "impl-debug"},
		},
		{
			name:      "Gitea forge filters to gt- and generic",
			forgeType: ForgeGitea,
			want:      []string{"gt-test", "plan-speckit", "wave-evolve", "impl-debug"},
		},
		{
			name:      "Unknown forge returns all",
			forgeType: ForgeUnknown,
			want:      pipelines,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPipelinesByForge(tt.forgeType, pipelines)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d pipelines, want %d: %v vs %v", len(got), len(tt.want), got, tt.want)
			}
			for i, name := range got {
				if name != tt.want[i] {
					t.Errorf("pipeline[%d] = %q, want %q", i, name, tt.want[i])
				}
			}
		})
	}
}

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantHost  string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "HTTPS with .git",
			url:       "https://github.com/owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH colon format",
			url:       "git@github.com:owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH protocol prefix",
			url:       "ssh://git@github.com/owner/repo.git",
			wantHost:  "github.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:     "empty",
			url:      "",
			wantHost: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, owner, repo := parseRemoteURL(tt.url)
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

package forge

import (
	"fmt"
	"testing"
)

func TestExtractHost(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"SSH GitHub", "git@github.com:org/repo.git", "github.com"},
		{"SSH GitLab", "git@gitlab.com:org/repo.git", "gitlab.com"},
		{"SSH Bitbucket", "git@bitbucket.org:org/repo.git", "bitbucket.org"},
		{"HTTPS GitHub", "https://github.com/org/repo.git", "github.com"},
		{"HTTPS GitLab", "https://gitlab.com/org/repo.git", "gitlab.com"},
		{"HTTPS Bitbucket", "https://bitbucket.org/org/repo.git", "bitbucket.org"},
		{"SSH URL format", "ssh://git@github.com/org/repo.git", "github.com"},
		{"SSH URL with port", "ssh://git@github.com:22/org/repo.git", "github.com"},
		{"HTTPS with port", "https://gitlab.example.com:8443/org/repo.git", "gitlab.example.com"},
		{"Enterprise GitHub", "git@github.example.com:org/repo.git", "github.example.com"},
		{"Self-hosted GitLab", "https://git.mycompany.com/org/repo.git", "git.mycompany.com"},
		{"HTTP (no S)", "http://github.com/org/repo.git", "github.com"},
		{"Uppercase host", "https://GitHub.COM/org/repo.git", "github.com"},
		{"Empty string", "", ""},
		{"No protocol no SSH", "just-a-string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHost(tt.url)
			if got != tt.want {
				t.Errorf("extractHost(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestClassifyHost(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		cfg      *ForgeConfig
		want     ForgeType
	}{
		{"GitHub", "github.com", nil, GitHub},
		{"GitLab", "gitlab.com", nil, GitLab},
		{"Bitbucket", "bitbucket.org", nil, Bitbucket},
		{"Codeberg", "codeberg.org", nil, Gitea},
		{"Unknown host", "example.com", nil, Unknown},
		{"GitHub subdomain", "enterprise.github.com", nil, GitHub},
		{"GitLab subdomain", "git.gitlab.com", nil, GitLab},
		{"Gitea indicator", "gitea.mycompany.com", nil, Gitea},
		{"Forgejo indicator", "forgejo.example.org", nil, Gitea},
		{"Custom domain override", "git.internal.corp", &ForgeConfig{
			Domains: map[string]string{"git.internal.corp": "github"},
		}, GitHub},
		{"Custom domain GitLab", "code.company.com", &ForgeConfig{
			Domains: map[string]string{"code.company.com": "gitlab"},
		}, GitLab},
		{"Custom domain takes priority", "github.com", &ForgeConfig{
			Domains: map[string]string{"github.com": "gitlab"},
		}, GitLab},
		{"Custom domain invalid forge", "custom.example.com", &ForgeConfig{
			Domains: map[string]string{"custom.example.com": "invalid"},
		}, Unknown},
		{"Nil config", "unknown.com", nil, Unknown},
		{"Empty domains map", "unknown.com", &ForgeConfig{Domains: map[string]string{}}, Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyHost(tt.hostname, tt.cfg)
			if got != tt.want {
				t.Errorf("classifyHost(%q) = %q, want %q", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestParseRemotes(t *testing.T) {
	tests := []struct {
		name   string
		output string
		cfg    *ForgeConfig
		want   []ForgeDetection
	}{
		{
			name:   "Single GitHub remote",
			output: "origin\tgit@github.com:org/repo.git (fetch)\norigin\tgit@github.com:org/repo.git (push)\n",
			want: []ForgeDetection{
				{Type: GitHub, Remote: "git@github.com:org/repo.git", Hostname: "github.com", CLITool: "gh"},
			},
		},
		{
			name: "Multiple remotes same forge",
			output: "origin\thttps://github.com/org/repo.git (fetch)\n" +
				"origin\thttps://github.com/org/repo.git (push)\n" +
				"upstream\thttps://github.com/upstream/repo.git (fetch)\n" +
				"upstream\thttps://github.com/upstream/repo.git (push)\n",
			want: []ForgeDetection{
				{Type: GitHub, Remote: "https://github.com/org/repo.git", Hostname: "github.com", CLITool: "gh"},
			},
		},
		{
			name: "Multiple remotes different forges",
			output: "origin\tgit@github.com:org/repo.git (fetch)\n" +
				"origin\tgit@github.com:org/repo.git (push)\n" +
				"mirror\tgit@gitlab.com:org/repo.git (fetch)\n" +
				"mirror\tgit@gitlab.com:org/repo.git (push)\n",
			want: []ForgeDetection{
				{Type: GitHub, Remote: "git@github.com:org/repo.git", Hostname: "github.com", CLITool: "gh"},
				{Type: GitLab, Remote: "git@gitlab.com:org/repo.git", Hostname: "gitlab.com", CLITool: "glab"},
			},
		},
		{
			name:   "Empty output",
			output: "",
			want:   nil,
		},
		{
			name:   "Malformed lines",
			output: "garbage\n\n   \nnot-a-remote\n",
			want:   nil,
		},
		{
			name:   "Unknown forge",
			output: "origin\thttps://example.com/org/repo.git (fetch)\n",
			want: []ForgeDetection{
				{Type: Unknown, Remote: "https://example.com/org/repo.git", Hostname: "example.com", CLITool: ""},
			},
		},
		{
			name:   "Custom domain",
			output: "origin\thttps://git.corp.com/org/repo.git (fetch)\n",
			cfg:    &ForgeConfig{Domains: map[string]string{"git.corp.com": "github"}},
			want: []ForgeDetection{
				{Type: GitHub, Remote: "https://git.corp.com/org/repo.git", Hostname: "git.corp.com", CLITool: "gh"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRemotes(tt.output, tt.cfg)
			if len(got) != len(tt.want) {
				t.Fatalf("parseRemotes() returned %d detections, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Type != tt.want[i].Type {
					t.Errorf("detection[%d].Type = %q, want %q", i, got[i].Type, tt.want[i].Type)
				}
				if got[i].Hostname != tt.want[i].Hostname {
					t.Errorf("detection[%d].Hostname = %q, want %q", i, got[i].Hostname, tt.want[i].Hostname)
				}
				if got[i].CLITool != tt.want[i].CLITool {
					t.Errorf("detection[%d].CLITool = %q, want %q", i, got[i].CLITool, tt.want[i].CLITool)
				}
				if got[i].Remote != tt.want[i].Remote {
					t.Errorf("detection[%d].Remote = %q, want %q", i, got[i].Remote, tt.want[i].Remote)
				}
			}
		})
	}
}

func TestDetect(t *testing.T) {
	mockFn := func() (string, error) {
		return "origin\tgit@github.com:re-cinq/wave.git (fetch)\norigin\tgit@github.com:re-cinq/wave.git (push)\n", nil
	}

	detections, err := Detect(nil, mockFn)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	if detections[0].Type != GitHub {
		t.Errorf("expected GitHub, got %q", detections[0].Type)
	}
}

func TestDetectError(t *testing.T) {
	errFn := func() (string, error) {
		return "", fmt.Errorf("git not found")
	}

	_, err := Detect(nil, errFn)
	if err == nil {
		t.Error("expected error from Detect")
	}
}

func TestDetectPrimary(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantType    ForgeType
		wantCount   int
	}{
		{
			name:      "Single GitHub",
			output:    "origin\tgit@github.com:org/repo.git (fetch)\n",
			wantType:  GitHub,
			wantCount: 1,
		},
		{
			name:      "No remotes",
			output:    "",
			wantType:  Unknown,
			wantCount: 0,
		},
		{
			name: "Multi-forge",
			output: "origin\tgit@github.com:org/repo.git (fetch)\n" +
				"mirror\tgit@gitlab.com:org/repo.git (fetch)\n",
			wantType:  GitHub,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFn := func() (string, error) { return tt.output, nil }
			primary, all, err := DetectPrimary(nil, mockFn)
			if err != nil {
				t.Fatalf("DetectPrimary() error = %v", err)
			}
			if primary.Type != tt.wantType {
				t.Errorf("primary.Type = %q, want %q", primary.Type, tt.wantType)
			}
			if len(all) != tt.wantCount {
				t.Errorf("len(all) = %d, want %d", len(all), tt.wantCount)
			}
		})
	}
}

func TestIsAmbiguous(t *testing.T) {
	tests := []struct {
		name       string
		detections []ForgeDetection
		want       bool
	}{
		{
			name:       "Empty",
			detections: nil,
			want:       false,
		},
		{
			name: "Single forge",
			detections: []ForgeDetection{
				{Type: GitHub},
			},
			want: false,
		},
		{
			name: "Same forge twice",
			detections: []ForgeDetection{
				{Type: GitHub},
				{Type: GitHub},
			},
			want: false,
		},
		{
			name: "Different forges",
			detections: []ForgeDetection{
				{Type: GitHub},
				{Type: GitLab},
			},
			want: true,
		},
		{
			name: "Unknown plus known",
			detections: []ForgeDetection{
				{Type: Unknown},
				{Type: GitHub},
			},
			want: false,
		},
		{
			name: "All unknown",
			detections: []ForgeDetection{
				{Type: Unknown},
				{Type: Unknown},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAmbiguous(tt.detections)
			if got != tt.want {
				t.Errorf("IsAmbiguous() = %v, want %v", got, tt.want)
			}
		})
	}
}

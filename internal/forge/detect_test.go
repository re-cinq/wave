package forge

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// disableProbing stubs the tea CLI check and HTTP probe client so tests don't
// make real network calls. Returns a cleanup function to restore originals.
func disableProbing(t *testing.T) {
	t.Helper()

	origClient := probeHTTPClient
	origTeaFunc := checkTeaCLIFunc

	// HTTP client that always fails (no server listening on 127.0.0.1:1)
	probeHTTPClient = &http.Client{
		Transport: &http.Transport{},
		Timeout:   1, // 1ns — effectively instant timeout
	}
	checkTeaCLIFunc = func(string) ForgeType { return ForgeUnknown }

	t.Cleanup(func() {
		probeHTTPClient = origClient
		checkTeaCLIFunc = origTeaFunc
	})
}

func TestDetect(t *testing.T) {
	disableProbing(t)

	tests := []struct {
		name          string
		url           string
		wantType      ForgeType
		wantHost      string
		wantOwner     string
		wantRepo      string
		wantCLI       string
		wantPrefix    string
		wantPRTerm    string
		wantPRCommand string
	}{
		{
			name:          "GitHub HTTPS",
			url:           "https://github.com/recinq/wave.git",
			wantType:      ForgeGitHub,
			wantHost:      "github.com",
			wantOwner:     "recinq",
			wantRepo:      "wave",
			wantCLI:       "gh",
			wantPrefix:    "gh",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "GitHub SSH",
			url:           "git@github.com:recinq/wave.git",
			wantType:      ForgeGitHub,
			wantHost:      "github.com",
			wantOwner:     "recinq",
			wantRepo:      "wave",
			wantCLI:       "gh",
			wantPrefix:    "gh",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "GitHub HTTPS without .git",
			url:           "https://github.com/recinq/wave",
			wantType:      ForgeGitHub,
			wantHost:      "github.com",
			wantOwner:     "recinq",
			wantRepo:      "wave",
			wantCLI:       "gh",
			wantPrefix:    "gh",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "GitLab HTTPS",
			url:           "https://gitlab.com/myorg/myrepo.git",
			wantType:      ForgeGitLab,
			wantHost:      "gitlab.com",
			wantOwner:     "myorg",
			wantRepo:      "myrepo",
			wantCLI:       "glab",
			wantPrefix:    "gl",
			wantPRTerm:    "Merge Request",
			wantPRCommand: "mr",
		},
		{
			name:          "GitLab SSH",
			url:           "git@gitlab.com:myorg/myrepo.git",
			wantType:      ForgeGitLab,
			wantHost:      "gitlab.com",
			wantOwner:     "myorg",
			wantRepo:      "myrepo",
			wantCLI:       "glab",
			wantPrefix:    "gl",
			wantPRTerm:    "Merge Request",
			wantPRCommand: "mr",
		},
		{
			name:          "Bitbucket HTTPS",
			url:           "https://bitbucket.org/team/project.git",
			wantType:      ForgeBitbucket,
			wantHost:      "bitbucket.org",
			wantOwner:     "team",
			wantRepo:      "project",
			wantCLI:       "bb",
			wantPrefix:    "bb",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "Bitbucket SSH",
			url:           "git@bitbucket.org:team/project.git",
			wantType:      ForgeBitbucket,
			wantHost:      "bitbucket.org",
			wantOwner:     "team",
			wantRepo:      "project",
			wantCLI:       "bb",
			wantPrefix:    "bb",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "Gitea self-hosted",
			url:           "https://gitea.example.com/user/repo.git",
			wantType:      ForgeGitea,
			wantHost:      "gitea.example.com",
			wantOwner:     "user",
			wantRepo:      "repo",
			wantCLI:       "tea",
			wantPrefix:    "gt",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "Codeberg HTTPS",
			url:           "https://codeberg.org/user/repo.git",
			wantType:      ForgeCodeberg,
			wantHost:      "codeberg.org",
			wantOwner:     "user",
			wantRepo:      "repo",
			wantCLI:       "tea",
			wantPrefix:    "gt",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "Codeberg SSH",
			url:           "git@codeberg.org:user/repo.git",
			wantType:      ForgeCodeberg,
			wantHost:      "codeberg.org",
			wantOwner:     "user",
			wantRepo:      "repo",
			wantCLI:       "tea",
			wantPrefix:    "gt",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "Codeberg HTTPS without .git",
			url:           "https://codeberg.org/user/repo",
			wantType:      ForgeCodeberg,
			wantHost:      "codeberg.org",
			wantOwner:     "user",
			wantRepo:      "repo",
			wantCLI:       "tea",
			wantPrefix:    "gt",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
		},
		{
			name:          "Self-hosted unknown",
			url:           "https://git.corp.com/team/app.git",
			wantType:      ForgeUnknown,
			wantHost:      "git.corp.com",
			wantOwner:     "team",
			wantRepo:      "app",
			wantCLI:       "",
			wantPrefix:    "",
			wantPRTerm:    "",
			wantPRCommand: "",
		},
		{
			name:          "SSH with ssh:// prefix",
			url:           "ssh://git@github.com/recinq/wave.git",
			wantType:      ForgeGitHub,
			wantHost:      "github.com",
			wantOwner:     "recinq",
			wantRepo:      "wave",
			wantCLI:       "gh",
			wantPrefix:    "gh",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
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
			name:          "HTTP URL (upgrades)",
			url:           "http://github.com/recinq/wave.git",
			wantType:      ForgeGitHub,
			wantHost:      "github.com",
			wantOwner:     "recinq",
			wantRepo:      "wave",
			wantCLI:       "gh",
			wantPrefix:    "gh",
			wantPRTerm:    "Pull Request",
			wantPRCommand: "pr",
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
			if got.PRTerm != tt.wantPRTerm {
				t.Errorf("PRTerm = %q, want %q", got.PRTerm, tt.wantPRTerm)
			}
			if got.PRCommand != tt.wantPRCommand {
				t.Errorf("PRCommand = %q, want %q", got.PRCommand, tt.wantPRCommand)
			}
		})
	}
}

func TestForgeInfo_Slug(t *testing.T) {
	tests := []struct {
		name string
		info ForgeInfo
		want string
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
		"gt-sync",
		"speckit-flow",
		"wave-evolve",
		"debug",
	}

	tests := []struct {
		name      string
		forgeType ForgeType
		want      []string
	}{
		{
			name:      "GitHub forge filters to gh- and generic",
			forgeType: ForgeGitHub,
			want:      []string{"gh-implement", "gh-review", "speckit-flow", "wave-evolve", "debug"},
		},
		{
			name:      "GitLab forge filters to gl- and generic",
			forgeType: ForgeGitLab,
			want:      []string{"gl-deploy", "speckit-flow", "wave-evolve", "debug"},
		},
		{
			name:      "Bitbucket forge filters to bb- and generic",
			forgeType: ForgeBitbucket,
			want:      []string{"bb-build", "speckit-flow", "wave-evolve", "debug"},
		},
		{
			name:      "Gitea forge filters to gt- and generic",
			forgeType: ForgeGitea,
			want:      []string{"gt-test", "gt-sync", "speckit-flow", "wave-evolve", "debug"},
		},
		{
			name:      "Codeberg forge filters to gt- and generic",
			forgeType: ForgeCodeberg,
			want:      []string{"gt-test", "gt-sync", "speckit-flow", "wave-evolve", "debug"},
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

func TestFilterPipelinesByForge_EmptyInput(t *testing.T) {
	got := FilterPipelinesByForge(ForgeGitHub, nil)
	if got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}

	got = FilterPipelinesByForge(ForgeGitHub, []string{})
	if got != nil {
		t.Errorf("expected nil for empty slice, got %v", got)
	}
}

func TestFilterPipelinesByForge_NoPrefixedPipelines(t *testing.T) {
	generic := []string{"speckit-flow", "wave-evolve", "debug", "deploy"}
	for _, ft := range []ForgeType{ForgeGitHub, ForgeGitLab, ForgeBitbucket, ForgeGitea, ForgeCodeberg} {
		got := FilterPipelinesByForge(ft, generic)
		if len(got) != len(generic) {
			t.Errorf("forge %s: got %d pipelines, want %d (all generic should be returned)", ft, len(got), len(generic))
		}
	}
}

func TestDetect_SubdomainVariants(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType ForgeType
	}{
		{
			name:     "GitHub enterprise subdomain",
			url:      "git@enterprise.github.com:org/repo.git",
			wantType: ForgeGitHub,
		},
		{
			name:     "GitLab self-hosted subdomain",
			url:      "git@self-hosted.gitlab.com:org/repo.git",
			wantType: ForgeGitLab,
		},
		{
			name:     "Gitea with port in URL",
			url:      "https://gitea.company.com/org/repo.git",
			wantType: ForgeGitea,
		},
		{
			name:     "Bitbucket server subdomain",
			url:      "git@stash.bitbucket.org:team/project.git",
			wantType: ForgeBitbucket,
		},
		{
			name:     "Codeberg pages subdomain",
			url:      "git@pages.codeberg.org:user/repo.git",
			wantType: ForgeCodeberg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.url)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

func TestDetect_SSHWithPort(t *testing.T) {
	disableProbing(t)

	// SSH URLs with port: ssh://git@github.com:2222/org/repo.git
	// The port in the URL causes the host to include ":2222", so
	// classifyHost won't match "github.com" — this is a known limitation.
	info := Detect("ssh://git@github.com:2222/org/repo.git")
	// Host includes port, so forge detection falls back to unknown
	if info.Type != ForgeUnknown {
		t.Errorf("Type = %q, want %q (port in host breaks suffix matching)", info.Type, ForgeUnknown)
	}
	if info.Owner != "org" {
		t.Errorf("Owner = %q, want %q", info.Owner, "org")
	}
	if info.Repo != "repo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "repo")
	}
}

func TestForgeInfo_Slug_AllForges(t *testing.T) {
	// Verify Slug works correctly for all detected forge types
	urls := []string{
		"https://github.com/owner/repo.git",
		"https://gitlab.com/owner/repo.git",
		"https://bitbucket.org/owner/repo.git",
		"https://gitea.example.com/owner/repo.git",
		"https://codeberg.org/owner/repo.git",
	}

	for _, url := range urls {
		info := Detect(url)
		slug := info.Slug()
		if slug != "owner/repo" {
			t.Errorf("Detect(%q).Slug() = %q, want %q", url, slug, "owner/repo")
		}
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

// TestProbeForgeType verifies that probeForgeType correctly identifies forges
// by probing their well-known API endpoints.
func TestProbeForgeType(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		wantType ForgeType
	}{
		{
			name: "Forgejo version endpoint",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/forgejo/v1/version" {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"9.0.0"}`)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			wantType: ForgeForgejo,
		},
		{
			name: "Gitea version endpoint",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/version" {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"1.21.0"}`)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			wantType: ForgeGitea,
		},
		{
			name: "GitLab version endpoint",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v4/version" {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"16.0.0"}`)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			wantType: ForgeGitLab,
		},
		{
			name: "Bitbucket Server properties endpoint",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/rest/api/1.0/application-properties" {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"8.0.0","buildNumber":"8000000"}`)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			wantType: ForgeBitbucket,
		},
		{
			name: "Forgejo wins over Gitea when both endpoints respond",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/forgejo/v1/version":
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"9.0.0"}`)
				case "/api/v1/version":
					// Forgejo also serves the Gitea API
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"1.21.0"}`)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantType: ForgeForgejo,
		},
		{
			name: "All endpoints return 404",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantType: ForgeUnknown,
		},
		{
			name: "All endpoints return 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantType: ForgeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			origClient := probeHTTPClient
			probeHTTPClient = srv.Client()
			defer func() { probeHTTPClient = origClient }()

			// Extract host:port from test server URL (strip "http://")
			host := strings.TrimPrefix(srv.URL, "http://")

			// Override the probe to use http:// instead of https:// for test server.
			// We achieve this by temporarily wrapping the client transport to
			// rewrite URLs. Instead, we use a simpler approach: directly test
			// probeForgeType after monkey-patching the probeHTTPClient to point
			// at the test server.

			// probeForgeType builds "https://host/path" but our test server is
			// http. We work around this by intercepting with a custom RoundTripper.
			probeHTTPClient = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					// Rewrite https → http and point at test server
					req.URL.Scheme = "http"
					req.URL.Host = host
					return http.DefaultTransport.RoundTrip(req)
				}),
			}

			got := probeForgeType(host)
			if got != tt.wantType {
				t.Errorf("probeForgeType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

// roundTripFunc is an adapter to use a function as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestClassifyHost_UnknownFallsToProbe verifies that an unrecognized hostname
// triggers endpoint probing and returns the probed forge type.
func TestClassifyHost_UnknownFallsToProbe(t *testing.T) {
	// Set up a test server that responds to the GitLab API endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/version" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"version":"16.5.0"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")

	origClient := probeHTTPClient
	origTeaFunc := checkTeaCLIFunc
	probeHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = host
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
	checkTeaCLIFunc = func(string) ForgeType { return ForgeUnknown }
	defer func() {
		probeHTTPClient = origClient
		checkTeaCLIFunc = origTeaFunc
	}()

	// classifyHost should fall through hostname matching and detect GitLab via probe.
	got := classifyHost(host)
	if got != ForgeGitLab {
		t.Errorf("classifyHost(%q) = %q, want %q", host, got, ForgeGitLab)
	}
}

// TestClassifyHost_TeaCLIFallback verifies that checkTeaCLI is consulted
// before HTTP probing for unknown hosts.
func TestClassifyHost_TeaCLIFallback(t *testing.T) {
	disableProbing(t)

	origTeaFunc := checkTeaCLIFunc
	checkTeaCLIFunc = func(host string) ForgeType {
		if host == "code.company.com" {
			return ForgeGitea
		}
		return ForgeUnknown
	}
	defer func() { checkTeaCLIFunc = origTeaFunc }()

	got := classifyHost("code.company.com")
	if got != ForgeGitea {
		t.Errorf("classifyHost(code.company.com) = %q, want %q", got, ForgeGitea)
	}
}

// TestManifestForgeOverride verifies that DetectWithOverride uses the manifest
// forge override instead of hostname-based or probe-based detection.
func TestManifestForgeOverride(t *testing.T) {
	disableProbing(t)

	tests := []struct {
		name       string
		url        string
		override   string
		wantType   ForgeType
		wantCLI    string
		wantPRTerm string
	}{
		{
			name:       "Override unknown host to gitlab",
			url:        "https://git.corp.com/team/app.git",
			override:   "gitlab",
			wantType:   ForgeGitLab,
			wantCLI:    "glab",
			wantPRTerm: "Merge Request",
		},
		{
			name:       "Override unknown host to github",
			url:        "https://git.corp.com/team/app.git",
			override:   "github",
			wantType:   ForgeGitHub,
			wantCLI:    "gh",
			wantPRTerm: "Pull Request",
		},
		{
			name:       "Override unknown host to gitea",
			url:        "https://code.internal.net/team/app.git",
			override:   "gitea",
			wantType:   ForgeGitea,
			wantCLI:    "tea",
			wantPRTerm: "Pull Request",
		},
		{
			name:       "Override unknown host to forgejo",
			url:        "https://code.internal.net/team/app.git",
			override:   "forgejo",
			wantType:   ForgeForgejo,
			wantCLI:    "tea",
			wantPRTerm: "Pull Request",
		},
		{
			name:       "Override unknown host to codeberg",
			url:        "https://git.corp.com/team/app.git",
			override:   "codeberg",
			wantType:   ForgeCodeberg,
			wantCLI:    "tea",
			wantPRTerm: "Pull Request",
		},
		{
			name:       "Override unknown host to bitbucket",
			url:        "https://git.corp.com/team/app.git",
			override:   "bitbucket",
			wantType:   ForgeBitbucket,
			wantCLI:    "bb",
			wantPRTerm: "Pull Request",
		},
		{
			name:       "Override is case-insensitive",
			url:        "https://git.corp.com/team/app.git",
			override:   "GitLab",
			wantType:   ForgeGitLab,
			wantCLI:    "glab",
			wantPRTerm: "Merge Request",
		},
		{
			name:       "Override overrides hostname detection",
			url:        "https://github.com/owner/repo.git",
			override:   "gitlab",
			wantType:   ForgeGitLab,
			wantCLI:    "glab",
			wantPRTerm: "Merge Request",
		},
		{
			name:       "Empty override falls through to hostname",
			url:        "https://github.com/owner/repo.git",
			override:   "",
			wantType:   ForgeGitHub,
			wantCLI:    "gh",
			wantPRTerm: "Pull Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectWithOverride(tt.url, tt.override)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.CLITool != tt.wantCLI {
				t.Errorf("CLITool = %q, want %q", got.CLITool, tt.wantCLI)
			}
			if got.PRTerm != tt.wantPRTerm {
				t.Errorf("PRTerm = %q, want %q", got.PRTerm, tt.wantPRTerm)
			}
		})
	}
}

// TestDetect_ForgejoHostname verifies that hostnames containing "forgejo"
// are classified as ForgeForgejo.
func TestDetect_ForgejoHostname(t *testing.T) {
	disableProbing(t)

	info := Detect("https://forgejo.example.com/user/repo.git")
	if info.Type != ForgeForgejo {
		t.Errorf("Type = %q, want %q", info.Type, ForgeForgejo)
	}
	if info.CLITool != "tea" {
		t.Errorf("CLITool = %q, want %q", info.CLITool, "tea")
	}
}

// TestFilterPipelinesByForge_Forgejo verifies that Forgejo uses the same
// pipeline prefix as Gitea (gt-).
func TestFilterPipelinesByForge_Forgejo(t *testing.T) {
	pipelines := []string{"gh-implement", "gt-test", "speckit-flow"}
	got := FilterPipelinesByForge(ForgeForgejo, pipelines)
	want := []string{"gt-test", "speckit-flow"}
	if len(got) != len(want) {
		t.Fatalf("got %d pipelines, want %d: %v vs %v", len(got), len(want), got, want)
	}
	for i, name := range got {
		if name != want[i] {
			t.Errorf("pipeline[%d] = %q, want %q", i, name, want[i])
		}
	}
}

// TestFilterPipelinesByForge_Codeberg verifies that Codeberg uses its own
// pipeline prefix (gt-), same as Gitea/Forgejo.
func TestFilterPipelinesByForge_Codeberg(t *testing.T) {
	pipelines := []string{"gh-implement", "gl-review", "gt-test", "gt-sync", "speckit-flow"}
	got := FilterPipelinesByForge(ForgeCodeberg, pipelines)
	want := []string{"gt-test", "gt-sync", "speckit-flow"}
	if len(got) != len(want) {
		t.Fatalf("got %d pipelines, want %d: %v vs %v", len(got), len(want), got, want)
	}
	for i, name := range got {
		if name != want[i] {
			t.Errorf("pipeline[%d] = %q, want %q", i, name, want[i])
		}
	}
}

// TestDetectWithOverride_Local verifies that the "local" forge override
// returns ForgeLocal regardless of the remote URL, with empty CLI/PR fields.
func TestDetectWithOverride_Local(t *testing.T) {
	disableProbing(t)

	tests := []struct {
		name string
		url  string
	}{
		{"local override with GitHub URL", "https://github.com/owner/repo.git"},
		{"local override with empty URL", ""},
		{"local override with SSH URL", "git@github.com:owner/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectWithOverride(tt.url, "local")
			if got.Type != ForgeLocal {
				t.Errorf("Type = %q, want %q", got.Type, ForgeLocal)
			}
			if got.CLITool != "" {
				t.Errorf("CLITool = %q, want empty", got.CLITool)
			}
			if got.PipelinePrefix != "local" {
				t.Errorf("PipelinePrefix = %q, want %q", got.PipelinePrefix, "local")
			}
			if got.PRTerm != "" {
				t.Errorf("PRTerm = %q, want empty", got.PRTerm)
			}
			if got.PRCommand != "" {
				t.Errorf("PRCommand = %q, want empty", got.PRCommand)
			}
			// Host/Owner/Repo should be empty for local override
			if got.Host != "" {
				t.Errorf("Host = %q, want empty", got.Host)
			}
			if got.Owner != "" {
				t.Errorf("Owner = %q, want empty", got.Owner)
			}
			if got.Repo != "" {
				t.Errorf("Repo = %q, want empty", got.Repo)
			}
		})
	}
}

// TestDetectWithOverride_LocalCaseInsensitive verifies that the "local"
// override is case-insensitive.
func TestDetectWithOverride_LocalCaseInsensitive(t *testing.T) {
	disableProbing(t)

	for _, override := range []string{"local", "Local", "LOCAL", "LocAL"} {
		t.Run(override, func(t *testing.T) {
			got := DetectWithOverride("https://github.com/owner/repo.git", override)
			if got.Type != ForgeLocal {
				t.Errorf("DetectWithOverride with override %q: Type = %q, want %q", override, got.Type, ForgeLocal)
			}
		})
	}
}

// TestFilterPipelinesByForge_Local verifies that ForgeLocal includes only
// local-prefixed and non-forge-prefixed pipelines, excluding forge-specific ones.
func TestFilterPipelinesByForge_Local(t *testing.T) {
	pipelines := []string{
		"gh-implement",
		"gh-review",
		"gl-deploy",
		"bb-build",
		"gt-test",
		"gt-sync",
		"local-validate",
		"local-lint",
		"speckit-flow",
		"wave-evolve",
		"debug",
	}

	got := FilterPipelinesByForge(ForgeLocal, pipelines)
	want := []string{"local-validate", "local-lint", "speckit-flow", "wave-evolve", "debug"}
	if len(got) != len(want) {
		t.Fatalf("got %d pipelines, want %d: %v vs %v", len(got), len(want), got, want)
	}
	for i, name := range got {
		if name != want[i] {
			t.Errorf("pipeline[%d] = %q, want %q", i, name, want[i])
		}
	}
}

// TestFilterPipelinesByForge_LocalNoForgePrefix verifies that ForgeLocal
// includes all generic pipelines when no forge-prefixed pipelines exist.
func TestFilterPipelinesByForge_LocalNoForgePrefix(t *testing.T) {
	generic := []string{"speckit-flow", "wave-evolve", "debug", "deploy"}
	got := FilterPipelinesByForge(ForgeLocal, generic)
	if len(got) != len(generic) {
		t.Errorf("got %d pipelines, want %d (all generic should be returned)", len(got), len(generic))
	}
}

// TestForgeLocal_Slug verifies that ForgeLocal has empty slug.
func TestForgeLocal_Slug(t *testing.T) {
	info := ForgeInfo{Type: ForgeLocal}
	if slug := info.Slug(); slug != "" {
		t.Errorf("Slug() = %q, want empty for ForgeLocal", slug)
	}
}

// TestForgeLocal_Metadata verifies that ForgeLocal metadata has correct values.
func TestForgeLocal_Metadata(t *testing.T) {
	cli, prefix, prTerm, prCommand := forgeMetadata(ForgeLocal)
	if cli != "" {
		t.Errorf("cli = %q, want empty", cli)
	}
	if prefix != "local" {
		t.Errorf("prefix = %q, want %q", prefix, "local")
	}
	if prTerm != "" {
		t.Errorf("prTerm = %q, want empty", prTerm)
	}
	if prCommand != "" {
		t.Errorf("prCommand = %q, want empty", prCommand)
	}
}

// TestPickFetchRemoteURL verifies that pickFetchRemoteURL prefers "origin" over
// other remotes, and falls back to the first non-origin remote when origin is absent.
func TestPickFetchRemoteURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantURL string
	}{
		{
			name:    "single origin remote",
			input:   "origin\thttps://github.com/owner/repo.git (fetch)\norigin\thttps://github.com/owner/repo.git (push)\n",
			wantURL: "https://github.com/owner/repo.git",
		},
		{
			name: "origin preferred over other remotes",
			input: "github\thttps://github.com/owner/repo.git (fetch)\n" +
				"github\thttps://github.com/owner/repo.git (push)\n" +
				"origin\thttps://gitea.example.com/owner/repo.git (fetch)\n" +
				"origin\thttps://gitea.example.com/owner/repo.git (push)\n",
			wantURL: "https://gitea.example.com/owner/repo.git",
		},
		{
			name: "origin first in output",
			input: "origin\thttps://gitea.example.com/owner/repo.git (fetch)\n" +
				"origin\thttps://gitea.example.com/owner/repo.git (push)\n" +
				"github\thttps://github.com/owner/repo.git (fetch)\n" +
				"github\thttps://github.com/owner/repo.git (push)\n",
			wantURL: "https://gitea.example.com/owner/repo.git",
		},
		{
			name: "no origin — falls back to first remote",
			input: "upstream\thttps://gitlab.com/owner/repo.git (fetch)\n" +
				"upstream\thttps://gitlab.com/owner/repo.git (push)\n" +
				"fork\thttps://github.com/fork/repo.git (fetch)\n" +
				"fork\thttps://github.com/fork/repo.git (push)\n",
			wantURL: "https://gitlab.com/owner/repo.git",
		},
		{
			name:    "empty output",
			input:   "",
			wantURL: "",
		},
		{
			name:    "only push remotes",
			input:   "origin\thttps://github.com/owner/repo.git (push)\n",
			wantURL: "",
		},
		{
			name:    "malformed line with single field",
			input:   "origin\n",
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickFetchRemoteURL(tt.input)
			if got != tt.wantURL {
				t.Errorf("pickFetchRemoteURL() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

// TestHasForgePrefix_Local verifies that "local-" is recognized as a forge prefix.
func TestHasForgePrefix_Local(t *testing.T) {
	if !hasForgePrefix("local-validate") {
		t.Error("hasForgePrefix(\"local-validate\") = false, want true")
	}
	if hasForgePrefix("localhost-something") {
		t.Error("hasForgePrefix(\"localhost-something\") = true, want false (no exact 'local-' prefix)")
	}
}

package forge

import "testing"

func TestResolveToken_GitHub_GH_TOKEN(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-123")
	t.Setenv("GITHUB_TOKEN", "github-token-456")

	token := ResolveToken(ForgeGitHub)
	if token != "gh-token-123" {
		t.Errorf("ResolveToken(GitHub) = %q, want %q (GH_TOKEN takes priority)", token, "gh-token-123")
	}
}

func TestResolveToken_GitHub_GITHUB_TOKEN(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "github-token-456")

	token := ResolveToken(ForgeGitHub)
	if token != "github-token-456" {
		t.Errorf("ResolveToken(GitHub) = %q, want %q", token, "github-token-456")
	}
}

func TestResolveToken_GitLab_GITLAB_TOKEN(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "gl-token-123")
	t.Setenv("GL_TOKEN", "gl-alt-456")

	token := ResolveToken(ForgeGitLab)
	if token != "gl-token-123" {
		t.Errorf("ResolveToken(GitLab) = %q, want %q (GITLAB_TOKEN takes priority)", token, "gl-token-123")
	}
}

func TestResolveToken_GitLab_GL_TOKEN(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GL_TOKEN", "gl-alt-456")

	token := ResolveToken(ForgeGitLab)
	if token != "gl-alt-456" {
		t.Errorf("ResolveToken(GitLab) = %q, want %q", token, "gl-alt-456")
	}
}

func TestResolveToken_Bitbucket_BITBUCKET_TOKEN(t *testing.T) {
	t.Setenv("BITBUCKET_TOKEN", "bb-token-123")
	t.Setenv("BB_TOKEN", "bb-alt-456")

	token := ResolveToken(ForgeBitbucket)
	if token != "bb-token-123" {
		t.Errorf("ResolveToken(Bitbucket) = %q, want %q", token, "bb-token-123")
	}
}

func TestResolveToken_Bitbucket_BB_TOKEN(t *testing.T) {
	t.Setenv("BITBUCKET_TOKEN", "")
	t.Setenv("BB_TOKEN", "bb-alt-456")

	token := ResolveToken(ForgeBitbucket)
	if token != "bb-alt-456" {
		t.Errorf("ResolveToken(Bitbucket) = %q, want %q", token, "bb-alt-456")
	}
}

func TestResolveToken_Gitea(t *testing.T) {
	t.Setenv("GITEA_TOKEN", "gitea-token-123")

	token := ResolveToken(ForgeGitea)
	if token != "gitea-token-123" {
		t.Errorf("ResolveToken(Gitea) = %q, want %q", token, "gitea-token-123")
	}
}

func TestResolveToken_Unknown(t *testing.T) {
	token := ResolveToken(ForgeUnknown)
	if token != "" {
		t.Errorf("ResolveToken(Unknown) = %q, want empty", token)
	}
}

func TestNewClient_GitHubWithToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")

	client, err := NewClient(ForgeInfo{Type: ForgeGitHub})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client for GitHub with token")
	}
	ghClient, ok := client.(*GitHubClient)
	if !ok {
		t.Fatalf("expected *GitHubClient, got %T", client)
	}
	if ghClient.ForgeType() != ForgeGitHub {
		t.Errorf("ForgeType() = %v, want %v", ghClient.ForgeType(), ForgeGitHub)
	}
}

func TestNewClient_GitLabWithToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "test-token")

	client, err := NewClient(ForgeInfo{Type: ForgeGitLab})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client != nil {
		t.Errorf("expected nil client for unsupported forge GitLab, got %T", client)
	}
}

func TestNewClient_NoToken(t *testing.T) {
	// Ensure no tokens are set
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	client, err := NewClient(ForgeInfo{Type: ForgeGitHub})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// May still get a token from `gh auth token` if gh is installed,
	// so we just verify the function returns without error.
	_ = client
}

func TestNewClient_UnknownForge(t *testing.T) {
	client, err := NewClient(ForgeInfo{Type: ForgeUnknown})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client != nil {
		t.Errorf("expected nil client for unknown forge, got %v", client)
	}
}

package scope

import (
	"fmt"
	"testing"

	"github.com/recinq/wave/internal/forge"
)

func TestNewIntrospector(t *testing.T) {
	tests := []struct {
		name      string
		forgeType forge.ForgeType
		wantNil   bool
	}{
		{"github", forge.ForgeGitHub, false},
		{"gitlab", forge.ForgeGitLab, false},
		{"gitea", forge.ForgeGitea, false},
		{"bitbucket", forge.ForgeBitbucket, true},
		{"unknown", forge.ForgeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewIntrospector(tt.forgeType)
			if (got == nil) != tt.wantNil {
				t.Errorf("NewIntrospector(%s) nil = %v, want nil = %v", tt.forgeType, got == nil, tt.wantNil)
			}
		})
	}
}

func TestGitHubIntrospector_ClassicPAT(t *testing.T) {
	g := &GitHubIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			// Simulate gh api user --include output with OAuth scopes header
			return []byte("HTTP/2.0 200 OK\nX-OAuth-Scopes: repo, read:packages, write:org\n\n{\"login\":\"test\"}"), nil
		},
	}

	t.Setenv("GH_TOKEN", "test-token")
	info, err := g.Introspect("GH_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Error != nil {
		t.Fatalf("unexpected info error: %v", info.Error)
	}
	if info.TokenType != "classic" {
		t.Errorf("token type = %q, want classic", info.TokenType)
	}
	wantScopes := []string{"repo", "read:packages", "write:org"}
	if len(info.Scopes) != len(wantScopes) {
		t.Fatalf("scopes = %v, want %v", info.Scopes, wantScopes)
	}
	for i, s := range info.Scopes {
		if s != wantScopes[i] {
			t.Errorf("scope[%d] = %q, want %q", i, s, wantScopes[i])
		}
	}
}

func TestGitHubIntrospector_FineGrainedPAT(t *testing.T) {
	g := &GitHubIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			// Fine-grained PATs don't return X-OAuth-Scopes header
			return []byte("HTTP/2.0 200 OK\n\n{\"login\":\"test\"}"), nil
		},
	}

	t.Setenv("GH_TOKEN", "test-token")
	info, err := g.Introspect("GH_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenType != "fine-grained" {
		t.Errorf("token type = %q, want fine-grained", info.TokenType)
	}
	if info.Error == nil {
		t.Error("expected error for fine-grained PAT introspection")
	}
}

func TestGitHubIntrospector_CommandFailure(t *testing.T) {
	g := &GitHubIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("command not found: gh")
		},
	}

	t.Setenv("GH_TOKEN", "test-token")
	info, err := g.Introspect("GH_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Error == nil {
		t.Error("expected info.Error for command failure")
	}
}

func TestGitHubIntrospector_EnvVarNotSet(t *testing.T) {
	g := &GitHubIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			t.Fatal("should not run command when env var is not set")
			return nil, nil
		},
	}

	info, err := g.Introspect("NONEXISTENT_TOKEN_VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Error == nil {
		t.Error("expected info.Error for missing env var")
	}
}

func TestGitLabIntrospector(t *testing.T) {
	g := &GitLabIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			return []byte(`{"scopes":["api","read_repository"]}`), nil
		},
	}

	t.Setenv("GITLAB_TOKEN", "test-token")
	info, err := g.Introspect("GITLAB_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Error != nil {
		t.Fatalf("unexpected info error: %v", info.Error)
	}
	if info.TokenType != "project" {
		t.Errorf("token type = %q, want project", info.TokenType)
	}
	if len(info.Scopes) != 2 || info.Scopes[0] != "api" || info.Scopes[1] != "read_repository" {
		t.Errorf("scopes = %v, want [api read_repository]", info.Scopes)
	}
}

func TestGitLabIntrospector_CommandFailure(t *testing.T) {
	g := &GitLabIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("glab not found")
		},
	}

	t.Setenv("GITLAB_TOKEN", "test-token")
	info, err := g.Introspect("GITLAB_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Error == nil {
		t.Error("expected info.Error for command failure")
	}
}

func TestGiteaIntrospector(t *testing.T) {
	g := &GiteaIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			return []byte("Logged in as: testuser"), nil
		},
	}

	t.Setenv("GITEA_TOKEN", "test-token")
	info, err := g.Introspect("GITEA_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Gitea doesn't expose scopes via CLI — should have a warning error
	if info.Error == nil {
		t.Error("expected info.Error for Gitea scope limitation")
	}
}

func TestGiteaIntrospector_CommandFailure(t *testing.T) {
	g := &GiteaIntrospector{
		runCmd: func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("tea not found")
		},
	}

	t.Setenv("GITEA_TOKEN", "test-token")
	info, err := g.Introspect("GITEA_TOKEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Error == nil {
		t.Error("expected info.Error for command failure")
	}
}

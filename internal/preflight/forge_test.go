package preflight

import (
	"fmt"
	"testing"
)

func TestDetectForge(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantType  string
		wantCLI   string
	}{
		// GitHub
		{name: "github https", remoteURL: "https://github.com/org/repo.git", wantType: "github", wantCLI: "gh"},
		{name: "github ssh", remoteURL: "git@github.com:org/repo.git", wantType: "github", wantCLI: "gh"},
		{name: "github ssh no .git", remoteURL: "git@github.com:org/repo", wantType: "github", wantCLI: "gh"},
		{name: "github https no .git", remoteURL: "https://github.com/org/repo", wantType: "github", wantCLI: "gh"},

		// GitLab
		{name: "gitlab https", remoteURL: "https://gitlab.com/org/repo.git", wantType: "gitlab", wantCLI: "glab"},
		{name: "gitlab ssh", remoteURL: "git@gitlab.com:org/repo.git", wantType: "gitlab", wantCLI: "glab"},
		{name: "gitlab self-hosted", remoteURL: "https://gitlab.example.com/org/repo.git", wantType: "gitlab", wantCLI: "glab"},

		// Gitea / Forgejo
		{name: "gitea https", remoteURL: "https://gitea.example.com/org/repo.git", wantType: "gitea", wantCLI: "tea"},
		{name: "forgejo https", remoteURL: "https://forgejo.example.com/org/repo.git", wantType: "gitea", wantCLI: "tea"},
		{name: "codeberg https", remoteURL: "https://codeberg.org/org/repo.git", wantType: "gitea", wantCLI: "tea"},
		{name: "gitea ssh", remoteURL: "git@gitea.example.com:org/repo.git", wantType: "gitea", wantCLI: "tea"},

		// Bitbucket
		{name: "bitbucket https", remoteURL: "https://bitbucket.org/org/repo.git", wantType: "bitbucket", wantCLI: "bb"},
		{name: "bitbucket ssh", remoteURL: "git@bitbucket.org:org/repo.git", wantType: "bitbucket", wantCLI: "bb"},

		// Unknown / edge cases
		{name: "unknown host", remoteURL: "https://example.com/org/repo.git", wantType: "unknown", wantCLI: ""},
		{name: "empty url", remoteURL: "", wantType: "unknown", wantCLI: ""},
		{name: "ssh scheme", remoteURL: "ssh://git@github.com/org/repo.git", wantType: "github", wantCLI: "gh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectForge(tt.remoteURL)
			if got.Type != tt.wantType {
				t.Errorf("DetectForge(%q).Type = %q, want %q", tt.remoteURL, got.Type, tt.wantType)
			}
			if got.CLI != tt.wantCLI {
				t.Errorf("DetectForge(%q).CLI = %q, want %q", tt.remoteURL, got.CLI, tt.wantCLI)
			}
		})
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantHost  string
	}{
		{name: "https", remoteURL: "https://github.com/org/repo.git", wantHost: "github.com"},
		{name: "ssh colon", remoteURL: "git@github.com:org/repo.git", wantHost: "github.com"},
		{name: "ssh scheme", remoteURL: "ssh://git@github.com/org/repo.git", wantHost: "github.com"},
		{name: "empty", remoteURL: "", wantHost: ""},
		{name: "just host", remoteURL: "https://github.com", wantHost: "github.com"},
		{name: "with port", remoteURL: "ssh://git@gitlab.example.com:2222/org/repo.git", wantHost: "gitlab.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHost(tt.remoteURL)
			if got != tt.wantHost {
				t.Errorf("extractHost(%q) = %q, want %q", tt.remoteURL, got, tt.wantHost)
			}
		})
	}
}

func TestCheckForgeCLI_EmptyRemote(t *testing.T) {
	c := NewChecker(nil)
	results, err := c.CheckForgeCLI("")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Error("expected OK for empty remote")
	}
	if results[0].Kind != "forge" {
		t.Errorf("expected kind 'forge', got %q", results[0].Kind)
	}
}

func TestCheckForgeCLI_UnknownForge(t *testing.T) {
	c := NewChecker(nil)
	results, err := c.CheckForgeCLI("https://example.com/org/repo.git")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Error("expected OK for unknown forge (skip)")
	}
}

func TestCheckForgeCLI_Found(t *testing.T) {
	c := NewChecker(nil)
	c.lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	results, err := c.CheckForgeCLI("https://github.com/org/repo.git")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Error("expected OK for found CLI")
	}
	if results[0].Name != "gh" {
		t.Errorf("expected name 'gh', got %q", results[0].Name)
	}
}

func TestCheckForgeCLI_NotFound(t *testing.T) {
	c := NewChecker(nil)
	c.lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("not found: %s", file)
	}

	results, err := c.CheckForgeCLI("https://github.com/org/repo.git")
	if err == nil {
		t.Fatal("expected error for missing CLI")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].OK {
		t.Error("expected not OK for missing CLI")
	}
	if results[0].Remediation == "" {
		t.Error("expected non-empty remediation")
	}

	var toolErr *ToolError
	if !containsToolError(err, &toolErr) {
		t.Fatalf("expected ToolError, got %T", err)
	}
}

// containsToolError is a helper to check if err is a ToolError.
func containsToolError(err error, target **ToolError) bool {
	te, ok := err.(*ToolError)
	if ok {
		*target = te
		return true
	}
	return false
}

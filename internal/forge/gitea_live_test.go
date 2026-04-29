//go:build live

// Live smoke tests for the Gitea adapter. Excluded from default builds via
// the `live` build tag so CI never hits real network. Configurable via env
// so the same test runs against any Gitea/Forgejo instance.
//
// Usage (from repo root):
//
//	set -a && . .env && set +a   # loads token_full
//	GITEA_HOST=git.librete.ch GITEA_OWNER=<owner> GITEA_REPO=<repo> \
//	  GITEA_TOKEN="$token_full" \
//	  go test -tags live -v -run Live ./internal/forge/...
//
// File is deletable — kept around for ad-hoc validation when changing
// the Gitea adapter wire format.
package forge

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLive_GiteaSmoke(t *testing.T) {
	host := envOr("GITEA_HOST", "")
	owner := envOr("GITEA_OWNER", "")
	repo := envOr("GITEA_REPO", "")
	token := envOr("GITEA_TOKEN", os.Getenv("CODEBERG_TOKEN"))
	if host == "" || owner == "" || repo == "" || token == "" {
		t.Skip("set GITEA_HOST + GITEA_OWNER + GITEA_REPO + GITEA_TOKEN (or CODEBERG_TOKEN)")
	}

	ft := ForgeGitea
	if host == "codeberg.org" {
		ft = ForgeCodeberg
	}
	c, err := NewGiteaClient(GiteaConfig{Host: host, Token: token, ForgeType: ft})
	if err != nil {
		t.Fatalf("NewGiteaClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issues, err := c.ListIssues(ctx, owner, repo, ListIssuesOptions{State: "all", PerPage: 5})
	if err != nil {
		t.Fatalf("ListIssues %s/%s: %v", owner, repo, err)
	}
	t.Logf("ListIssues(%s/%s) → %d items from %s", owner, repo, len(issues), host)
	for _, i := range issues {
		t.Logf("  #%d %q (%s) by %s — labels=%d isPR=%v", i.Number, i.Title, i.State, i.Author, len(i.Labels), i.IsPR)
	}

	prs, err := c.ListPullRequests(ctx, owner, repo, ListPullRequestsOptions{State: "all", PerPage: 5})
	if err != nil {
		t.Fatalf("ListPullRequests %s/%s: %v", owner, repo, err)
	}
	t.Logf("ListPullRequests(%s/%s) → %d items", owner, repo, len(prs))
	for _, p := range prs {
		t.Logf("  #%d %q [%s] head=%s base=%s merged=%v", p.Number, p.Title, p.State, p.HeadBranch, p.BaseBranch, p.Merged)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

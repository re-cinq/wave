package forge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGiteaClient_Validation(t *testing.T) {
	_, err := NewGiteaClient(GiteaConfig{Host: "", Token: "x"})
	assert.Error(t, err, "empty Host must error")
	_, err = NewGiteaClient(GiteaConfig{Host: "h", Token: ""})
	assert.Error(t, err, "empty Token must error")
}

func TestGiteaClient_GetIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/issues/42", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "token secret", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"number": 42,
			"title": "test",
			"body": "hello",
			"state": "open",
			"user": {"login": "alice"},
			"labels": [{"name": "bug", "color": "ff0000"}],
			"assignees": [{"login": "bob"}],
			"comments": 3,
			"html_url": "https://example.com/owner/repo/issues/42",
			"pull_request": null
		}`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	got, err := c.GetIssue(context.Background(), "owner", "repo", 42)
	require.NoError(t, err)
	assert.Equal(t, 42, got.Number)
	assert.Equal(t, "alice", got.Author)
	require.Len(t, got.Labels, 1)
	assert.Equal(t, "bug", got.Labels[0].Name)
	assert.Equal(t, []string{"bob"}, got.Assignees)
	assert.False(t, got.IsPR)
}

func TestGiteaClient_GetIssue_FlagsPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/issues/7", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"number":7,"title":"pr","state":"open","user":{"login":"a"},"pull_request":{}}`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	got, err := c.GetIssue(context.Background(), "owner", "repo", 7)
	require.NoError(t, err)
	assert.True(t, got.IsPR, "issue with non-nil pull_request must flag as PR")
}

func TestGiteaClient_ListIssues_QueryString(t *testing.T) {
	var capturedQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	_, err := c.ListIssues(context.Background(), "owner", "repo", ListIssuesOptions{
		State: "open", Labels: []string{"bug", "p0"}, Sort: "updated", PerPage: 5, Page: 2,
	})
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "state=open")
	assert.Contains(t, capturedQuery, "labels=bug%2Cp0")
	assert.Contains(t, capturedQuery, "sort=updated")
	assert.Contains(t, capturedQuery, "limit=5")
	assert.Contains(t, capturedQuery, "page=2")
	assert.Contains(t, capturedQuery, "type=issues")
}

func TestGiteaClient_GetPullRequest_MergedStateOverride(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/pulls/3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"number": 3,
			"title": "feat",
			"state": "closed",
			"user": {"login": "a"},
			"merged": true,
			"head": {"ref": "feat/x", "sha": "abc"},
			"base": {"ref": "main", "sha": "def"}
		}`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	got, err := c.GetPullRequest(context.Background(), "owner", "repo", 3)
	require.NoError(t, err)
	assert.Equal(t, "merged", got.State, "merged=true must override state=closed")
	assert.Equal(t, "feat/x", got.HeadBranch)
}

func TestGiteaClient_GetCommitChecks_StatusMapping(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/statuses/abc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"context":"build","status":"success","target_url":"u1"},
			{"context":"test","status":"failure","target_url":"u2"},
			{"context":"lint","status":"pending","target_url":"u3"}
		]`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	got, err := c.GetCommitChecks(context.Background(), "owner", "repo", "abc")
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "success", got[0].Conclusion)
	assert.Equal(t, "failure", got[1].Conclusion)
	assert.Equal(t, "", got[2].Conclusion, "pending → empty conclusion")
}

func TestGiteaClient_CreatePullRequestReview_BodyShape(t *testing.T) {
	var receivedPayload map[string]string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	require.NoError(t, c.CreatePullRequestReview(context.Background(), "owner", "repo", 1, "APPROVE", "lgtm"))
	assert.Equal(t, "APPROVED", receivedPayload["event"], "APPROVE must map to APPROVED for Gitea")
	assert.Equal(t, "lgtm", receivedPayload["body"])
}

func TestGiteaClient_NonOKReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/repos/owner/repo/issues/9", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := giteaClientForTest(t, srv, "secret")
	_, err := c.GetIssue(context.Background(), "owner", "repo", 9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClassifyHost_LibreteShortcut(t *testing.T) {
	assert.Equal(t, ForgeGitea, classifyHost("git.librete.ch"))
}

// giteaClientForTest builds a GiteaClient pointed at the supplied
// httptest server. It strips the scheme, replaces baseURL with the
// server's URL+/api/v1, and reuses the server's TLS config.
func giteaClientForTest(t *testing.T, srv *httptest.Server, token string) *GiteaClient {
	t.Helper()
	host := strings.TrimPrefix(srv.URL, "https://")
	c, err := NewGiteaClient(GiteaConfig{
		Host:      host,
		Token:     token,
		ForgeType: ForgeGitea,
	})
	require.NoError(t, err)
	return c
}

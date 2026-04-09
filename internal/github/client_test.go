package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config ClientConfig
		want   func(*Client) bool
	}{
		{
			name:   "default config",
			config: ClientConfig{},
			want: func(c *Client) bool {
				return c.baseURL == DefaultAPIURL &&
					c.userAgent == DefaultUserAgent &&
					c.maxRetries == DefaultMaxRetries
			},
		},
		{
			name: "custom config",
			config: ClientConfig{
				Token:      "test-token",
				BaseURL:    "https://custom.api",
				UserAgent:  "custom-agent",
				MaxRetries: 5,
			},
			want: func(c *Client) bool {
				return c.token == "test-token" &&
					c.baseURL == "https://custom.api" &&
					c.userAgent == "custom-agent" &&
					c.maxRetries == 5
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			assert.NotNil(t, client)
			assert.True(t, tt.want(client))
		})
	}
}

func TestClient_GetIssue(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues/123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		issue := Issue{
			ID:     1,
			Number: 123,
			Title:  "Test Issue",
			Body:   "Test body",
			State:  "open",
		}
		_ = json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
	})

	issue, err := client.GetIssue(context.Background(), "owner", "repo", 123)
	require.NoError(t, err)
	assert.Equal(t, 123, issue.Number)
	assert.Equal(t, "Test Issue", issue.Title)
	assert.Equal(t, "open", issue.State)
}

func TestClient_ListIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues", r.URL.Path)
		assert.Equal(t, "open", r.URL.Query().Get("state"))

		issues := []*Issue{
			{Number: 1, Title: "Issue 1", State: "open"},
			{Number: 2, Title: "Issue 2", State: "open"},
		}
		_ = json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})

	issues, err := client.ListIssues(context.Background(), "owner", "repo", ListIssuesOptions{
		State: "open",
	})
	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, 1, issues[0].Number)
	assert.Equal(t, 2, issues[1].Number)
}

func TestClient_UpdateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/issues/123", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var update IssueUpdate
		_ = json.NewDecoder(r.Body).Decode(&update)
		assert.NotNil(t, update.Title)
		assert.Equal(t, "Updated Title", *update.Title)

		issue := Issue{
			Number: 123,
			Title:  *update.Title,
			State:  "open",
		}
		_ = json.NewEncoder(w).Encode(issue)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})

	newTitle := "Updated Title"
	issue, err := client.UpdateIssue(context.Background(), "owner", "repo", 123, IssueUpdate{
		Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", issue.Title)
}

func TestClient_CreatePullRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req CreatePullRequestRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "Test PR", req.Title)
		assert.Equal(t, "feature", req.Head)
		assert.Equal(t, "main", req.Base)

		pr := PullRequest{
			Number: 456,
			Title:  req.Title,
			State:  "open",
			Head:   &GitRef{Ref: req.Head},
			Base:   &GitRef{Ref: req.Base},
		}
		_ = json.NewEncoder(w).Encode(pr)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})

	pr, err := client.CreatePullRequest(context.Background(), "owner", "repo", CreatePullRequestRequest{
		Title: "Test PR",
		Head:  "feature",
		Base:  "main",
	})
	require.NoError(t, err)
	assert.Equal(t, 456, pr.Number)
	assert.Equal(t, "Test PR", pr.Title)
}

func TestClient_RateLimitHandling(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call returns rate limit error
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "9999999999") // Far future
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "API rate limit exceeded",
			})
			return
		}
		// Second call succeeds
		_ = json.NewEncoder(w).Encode(Issue{Number: 1})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:    server.URL,
		MaxRetries: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetIssue(ctx, "owner", "repo", 1)
	assert.Error(t, err)          // Should timeout waiting for rate limit reset
	assert.Equal(t, 1, callCount) // Only one call made before timeout
}

func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:       "404 not found",
			statusCode: 404,
			responseBody: map[string]string{
				"message": "Not Found",
			},
			expectError:   true,
			errorContains: "Not Found",
		},
		{
			name:       "validation error",
			statusCode: 422,
			responseBody: map[string]interface{}{
				"message": "Validation Failed",
				"errors": []map[string]string{
					{"field": "title", "code": "missing"},
				},
			},
			expectError:   true,
			errorContains: "Validation Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{BaseURL: server.URL})

			_, err := client.GetIssue(context.Background(), "owner", "repo", 1)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Authentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", auth)
		_ = json.NewEncoder(w).Encode(Issue{Number: 1})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	_, err := client.GetIssue(context.Background(), "owner", "repo", 1)
	assert.NoError(t, err)
}

func TestClient_GetCommitCheckRuns(t *testing.T) {
	t.Run("success with check runs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/commits/abc123/check-runs", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			assert.Equal(t, "GET", r.Method)

			resp := CheckRunsResponse{
				TotalCount: 2,
				CheckRuns: []*CheckRun{
					{
						ID:         1,
						Name:       "CI / build",
						Status:     "completed",
						Conclusion: "success",
						HTMLURL:    "https://github.com/owner/repo/runs/1",
					},
					{
						ID:         2,
						Name:       "CI / lint",
						Status:     "in_progress",
						Conclusion: "",
						HTMLURL:    "https://github.com/owner/repo/runs/2",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(ClientConfig{BaseURL: server.URL})

		result, err := client.GetCommitCheckRuns(context.Background(), "owner", "repo", "abc123")
		require.NoError(t, err)
		assert.Equal(t, 2, result.TotalCount)
		require.Len(t, result.CheckRuns, 2)
		assert.Equal(t, "CI / build", result.CheckRuns[0].Name)
		assert.Equal(t, "completed", result.CheckRuns[0].Status)
		assert.Equal(t, "success", result.CheckRuns[0].Conclusion)
		assert.Equal(t, "https://github.com/owner/repo/runs/1", result.CheckRuns[0].HTMLURL)
		assert.Equal(t, "CI / lint", result.CheckRuns[1].Name)
		assert.Equal(t, "in_progress", result.CheckRuns[1].Status)
		assert.Empty(t, result.CheckRuns[1].Conclusion)
	})

	t.Run("empty check runs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := CheckRunsResponse{
				TotalCount: 0,
				CheckRuns:  []*CheckRun{},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(ClientConfig{BaseURL: server.URL})

		result, err := client.GetCommitCheckRuns(context.Background(), "owner", "repo", "def456")
		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalCount)
		assert.Empty(t, result.CheckRuns)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Internal Server Error",
			})
		}))
		defer server.Close()

		client := NewClient(ClientConfig{BaseURL: server.URL})

		_, err := client.GetCommitCheckRuns(context.Background(), "owner", "repo", "bad")
		assert.Error(t, err)
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer server.Close()

		client := NewClient(ClientConfig{BaseURL: server.URL})

		_, err := client.GetCommitCheckRuns(context.Background(), "owner", "repo", "abc123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode check runs")
	})
}

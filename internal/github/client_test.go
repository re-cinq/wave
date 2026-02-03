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
		json.NewEncoder(w).Encode(issue)
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
		json.NewEncoder(w).Encode(issues)
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
		json.NewDecoder(r.Body).Decode(&update)
		assert.NotNil(t, update.Title)
		assert.Equal(t, "Updated Title", *update.Title)

		issue := Issue{
			Number: 123,
			Title:  *update.Title,
			State:  "open",
		}
		json.NewEncoder(w).Encode(issue)
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
		json.NewDecoder(r.Body).Decode(&req)
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
		json.NewEncoder(w).Encode(pr)
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
			json.NewEncoder(w).Encode(map[string]string{
				"message": "API rate limit exceeded",
			})
			return
		}
		// Second call succeeds
		json.NewEncoder(w).Encode(Issue{Number: 1})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:    server.URL,
		MaxRetries: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.GetIssue(ctx, "owner", "repo", 1)
	assert.Error(t, err) // Should timeout waiting for rate limit reset
	assert.Equal(t, 1, callCount) // Only one call made before timeout
}

func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		expectError    bool
		errorContains  string
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
				json.NewEncoder(w).Encode(tt.responseBody)
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
		json.NewEncoder(w).Encode(Issue{Number: 1})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	_, err := client.GetIssue(context.Background(), "owner", "repo", 1)
	assert.NoError(t, err)
}

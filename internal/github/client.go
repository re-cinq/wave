package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAPIURL     = "https://api.github.com"
	DefaultUserAgent  = "Wave-GitHub-Integration/1.0"
	DefaultMaxRetries = 3
)

// Client is a production-ready GitHub API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	userAgent  string
	maxRetries int
	rateLimiter *RateLimiter
}

// ClientConfig holds configuration for the GitHub client
type ClientConfig struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	MaxRetries int
}

// NewClient creates a new GitHub API client
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = DefaultAPIURL
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if config.UserAgent == "" {
		config.UserAgent = DefaultUserAgent
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}

	return &Client{
		baseURL:     config.BaseURL,
		httpClient:  config.HTTPClient,
		token:       config.Token,
		userAgent:   config.UserAgent,
		maxRetries:  config.MaxRetries,
		rateLimiter: NewRateLimiter(),
	}
}

// doRequest performs an HTTP request with retry logic and rate limiting
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Build URL
	fullURL := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Wait for rate limit
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", c.userAgent)
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		// Update rate limiter from response headers
		c.rateLimiter.Update(resp.Header)

		// Handle rate limiting
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == 429 {
			resetTime := c.rateLimiter.ResetTime()
			if !resetTime.IsZero() && time.Until(resetTime) > 0 {
				lastErr = &RateLimitError{
					ResetTime: resetTime,
					Message:   "GitHub API rate limit exceeded",
				}
				resp.Body.Close()
				continue
			}
		}

		// Check for errors
		if resp.StatusCode >= 400 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var apiErr APIError
			if err := json.Unmarshal(bodyBytes, &apiErr); err == nil {
				apiErr.StatusCode = resp.StatusCode
				return nil, &apiErr
			}

			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(bodyBytes),
			}
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
	}
	return nil, fmt.Errorf("request failed after %d attempts", c.maxRetries)
}

// GetIssue retrieves a single issue
func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode issue: %w", err)
	}

	return &issue, nil
}

// ListIssues retrieves issues for a repository
func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]*Issue, error) {
	params := url.Values{}
	if opts.State != "" {
		params.Set("state", opts.State)
	}
	if len(opts.Labels) > 0 {
		params.Set("labels", strings.Join(opts.Labels, ","))
	}
	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}
	if opts.Direction != "" {
		params.Set("direction", opts.Direction)
	}
	if opts.Since != nil {
		params.Set("since", opts.Since.Format(time.RFC3339))
	}
	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	}

	path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var issues []*Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("failed to decode issues: %w", err)
	}

	return issues, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(ctx context.Context, owner, repo string, number int, update IssueUpdate) (*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)

	resp, err := c.doRequest(ctx, http.MethodPatch, path, update)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode updated issue: %w", err)
	}

	return &issue, nil
}

// CreateIssueComment creates a new comment on an issue
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, number int, body string) (*IssueComment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)

	payload := map[string]string{"body": body}

	resp, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var comment IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode comment: %w", err)
	}

	return &comment, nil
}

// GetPullRequest retrieves a pull request
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode pull request: %w", err)
	}

	return &pr, nil
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, pr CreatePullRequestRequest) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)

	resp, err := c.doRequest(ctx, http.MethodPost, path, pr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pullRequest PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pullRequest); err != nil {
		return nil, fmt.Errorf("failed to decode pull request: %w", err)
	}

	return &pullRequest, nil
}

// GetRepository retrieves repository information
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var repository Repository
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf("failed to decode repository: %w", err)
	}

	return &repository, nil
}

// CreateBranch creates a new branch from a reference
func (c *Client) CreateBranch(ctx context.Context, owner, repo, branchName, fromRef string) (*Reference, error) {
	// First get the SHA of the reference we're branching from
	getRefPath := fmt.Sprintf("/repos/%s/%s/git/ref/heads/%s", owner, repo, fromRef)
	resp, err := c.doRequest(ctx, http.MethodGet, getRefPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get source reference: %w", err)
	}

	var sourceRef Reference
	if err := json.NewDecoder(resp.Body).Decode(&sourceRef); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to decode source reference: %w", err)
	}
	resp.Body.Close()

	// Create the new branch
	createRefPath := fmt.Sprintf("/repos/%s/%s/git/refs", owner, repo)
	payload := map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": sourceRef.Object.SHA,
	}

	resp, err = c.doRequest(ctx, http.MethodPost, createRefPath, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}
	defer resp.Body.Close()

	var newRef Reference
	if err := json.NewDecoder(resp.Body).Decode(&newRef); err != nil {
		return nil, fmt.Errorf("failed to decode new reference: %w", err)
	}

	return &newRef, nil
}

// GetRateLimit retrieves the current rate limit status
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimitStatus, error) {
	path := "/rate_limit"

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rateLimit struct {
		Resources struct {
			Core RateLimitStatus `json:"core"`
		} `json:"resources"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rateLimit); err != nil {
		return nil, fmt.Errorf("failed to decode rate limit: %w", err)
	}

	return &rateLimit.Resources.Core, nil
}

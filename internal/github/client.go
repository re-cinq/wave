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

	"github.com/recinq/wave/internal/httpx"
	"github.com/recinq/wave/internal/timeouts"
)

const (
	DefaultAPIURL     = "https://api.github.com"
	DefaultUserAgent  = "Wave-GitHub-Integration/1.0"
	DefaultMaxRetries = 3
)

// Client is a production-ready GitHub API client
type Client struct {
	baseURL     string
	httpClient  *httpx.Client
	token       string
	userAgent   string
	maxRetries  int
	rateLimiter *RateLimiter
}

// ClientConfig holds configuration for the GitHub client
type ClientConfig struct {
	Token      string
	BaseURL    string
	HTTPClient *httpx.Client
	UserAgent  string
	MaxRetries int
}

// NewClient creates a new GitHub API client. Transport-level requests go
// through internal/httpx for unified timeout/audit policy. Retry and rate
// limiting are handled at this layer (see doRequest), so the embedded
// httpx.Client runs in single-shot mode (MaxRetries: 0).
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = DefaultAPIURL
	}
	if config.HTTPClient == nil {
		config.HTTPClient = httpx.New(httpx.Config{
			Timeout:    timeouts.ForgeAPI,
			MaxRetries: 0,
		})
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

		resp, err := c.httpClient.Do(ctx, req)
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

// ListIssueComments retrieves comments on an issue or pull request.
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, number int, perPage int) ([]*IssueComment, error) {
	params := url.Values{}
	if perPage > 0 {
		params.Set("per_page", strconv.Itoa(perPage))
	}
	params.Set("sort", "created")
	params.Set("direction", "desc")

	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var comments []*IssueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %w", err)
	}

	return comments, nil
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

// ListPullRequests retrieves pull requests for a repository
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestsOptions) ([]*PullRequest, error) {
	params := url.Values{}
	if opts.State != "" {
		params.Set("state", opts.State)
	}
	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}
	if opts.Direction != "" {
		params.Set("direction", opts.Direction)
	}
	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var prs []*PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, fmt.Errorf("failed to decode pull requests: %w", err)
	}

	return prs, nil
}

// ListPullRequestCommits retrieves the commits on a pull request.
// Note: returns at most 100 commits (single page, no pagination).
// PRs with more than 100 commits will have their commit list silently truncated.
func (c *Client) ListPullRequestCommits(ctx context.Context, owner, repo string, number int) ([]*PullRequestCommit, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/commits?per_page=100", owner, repo, number)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var commits []*PullRequestCommit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, fmt.Errorf("failed to decode pull request commits: %w", err)
	}

	return commits, nil
}

// GetCommitCheckRuns retrieves check runs for a specific commit reference.
func (c *Client) GetCommitCheckRuns(ctx context.Context, owner, repo, ref string) (*CheckRunsResponse, error) {
	path := fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?per_page=100", owner, repo, ref)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CheckRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode check runs: %w", err)
	}

	return &result, nil
}

// CreatePullRequestReview submits a review on a pull request.
// event must be one of "APPROVE", "REQUEST_CHANGES", or "COMMENT".
func (c *Client) CreatePullRequestReview(ctx context.Context, owner, repo string, number int, event, body string) error {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)

	payload := map[string]string{
		"event": event,
		"body":  body,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

package forge

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/recinq/wave/internal/httpx"
)

// GiteaClient implements forge.Client against the Gitea v1 HTTP API.
// Works with vanilla Gitea, Forgejo (which serves the same /api/v1
// surface), and Codeberg (a hosted Forgejo). Authentication is via the
// `Authorization: token <TOKEN>` header.
//
// Self-hosted Gitea instances frequently use private CAs; the default
// transport tolerates self-signed TLS to mirror the existing classifyHost
// probe behaviour.
type GiteaClient struct {
	httpClient *httpx.Client
	baseURL    string // e.g. "https://git.librete.ch/api/v1"
	token      string
	forgeType  ForgeType
}

// GiteaConfig configures NewGiteaClient. Host is the bare hostname
// ("git.librete.ch"); the constructor builds the API URL. Scheme defaults
// to https when empty.
type GiteaConfig struct {
	Host      string
	Scheme    string // "https" (default) or "http"
	Token     string
	ForgeType ForgeType // ForgeGitea, ForgeForgejo, or ForgeCodeberg
}

// NewGiteaClient builds a Gitea HTTP client. Returns an error if Host is
// empty or Token is empty (every Gitea API call requires auth).
func NewGiteaClient(cfg GiteaConfig) (*GiteaClient, error) {
	if cfg.Host == "" {
		return nil, errors.New("forge: NewGiteaClient: Host is required")
	}
	if cfg.Token == "" {
		return nil, errors.New("forge: NewGiteaClient: Token is required")
	}
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}
	ft := cfg.ForgeType
	if ft == "" {
		ft = ForgeGitea
	}
	transport := &http.Transport{
		// Self-hosted instances often ship with private CAs. Mirror the
		// detect.go probe behaviour rather than failing on self-signed.
		//nolint:gosec // self-hosted Gitea/Forgejo commonly use private CAs
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &GiteaClient{
		httpClient: httpx.New(httpx.Config{
			Timeout:    30 * time.Second,
			MaxRetries: 2,
			Transport:  transport,
		}),
		baseURL:   fmt.Sprintf("%s://%s/api/v1", scheme, cfg.Host),
		token:     cfg.Token,
		forgeType: ft,
	}, nil
}

func (g *GiteaClient) ForgeType() ForgeType { return g.forgeType }

// do executes an authenticated request and decodes the JSON body into out
// when out is non-nil. Returns an error for any non-2xx response.
func (g *GiteaClient) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("gitea: build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := g.httpClient.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("gitea: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gitea: %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- Gitea API DTOs ----------------------------------------------------

type giteaUser struct {
	Login string `json:"login"`
}

type giteaLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type giteaIssue struct {
	Number    int          `json:"number"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	State     string       `json:"state"`
	User      giteaUser    `json:"user"`
	Labels    []giteaLabel `json:"labels"`
	Assignees []giteaUser  `json:"assignees"`
	Comments  int          `json:"comments"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	ClosedAt  *time.Time   `json:"closed_at"`
	HTMLURL   string       `json:"html_url"`
	// Gitea exposes pull_request only for issues that are PRs.
	PullRequest *struct{} `json:"pull_request,omitempty"`
}

type giteaBranch struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type giteaPR struct {
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	Body         string       `json:"body"`
	State        string       `json:"state"`
	User         giteaUser    `json:"user"`
	Labels       []giteaLabel `json:"labels"`
	Draft        bool         `json:"draft"`
	Merged       bool         `json:"merged"`
	Head         giteaBranch  `json:"head"`
	Base         giteaBranch  `json:"base"`
	Additions    int          `json:"additions"`
	Deletions    int          `json:"deletions"`
	ChangedFiles int          `json:"changed_files"`
	Commits      int          `json:"commits"`
	Comments     int          `json:"comments"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	ClosedAt     *time.Time   `json:"closed_at"`
	MergedAt     *time.Time   `json:"merged_at"`
	HTMLURL      string       `json:"html_url"`
}

type giteaCommitAuthor struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
}

type giteaCommitInner struct {
	Message string            `json:"message"`
	Author  giteaCommitAuthor `json:"author"`
}

type giteaCommit struct {
	SHA     string           `json:"sha"`
	Commit  giteaCommitInner `json:"commit"`
	HTMLURL string           `json:"html_url"`
}

type giteaCheckRun struct {
	Name       string `json:"context"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	URL        string `json:"target_url"`
}

type giteaComment struct {
	User      giteaUser `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	HTMLURL   string    `json:"html_url"`
}

// --- Conversions to forge-neutral types --------------------------------

func convertGiteaIssue(gi *giteaIssue) *Issue {
	labels := make([]Label, 0, len(gi.Labels))
	for _, l := range gi.Labels {
		labels = append(labels, Label(l))
	}
	assignees := make([]string, 0, len(gi.Assignees))
	for _, a := range gi.Assignees {
		assignees = append(assignees, a.Login)
	}
	return &Issue{
		Number:    gi.Number,
		Title:     gi.Title,
		Body:      gi.Body,
		State:     gi.State,
		Author:    gi.User.Login,
		Labels:    labels,
		Assignees: assignees,
		Comments:  gi.Comments,
		CreatedAt: gi.CreatedAt,
		UpdatedAt: gi.UpdatedAt,
		ClosedAt:  gi.ClosedAt,
		HTMLURL:   gi.HTMLURL,
		IsPR:      gi.PullRequest != nil,
	}
}

func convertGiteaPR(p *giteaPR) *PullRequest {
	labels := make([]Label, 0, len(p.Labels))
	for _, l := range p.Labels {
		labels = append(labels, Label(l))
	}
	state := p.State
	if p.Merged {
		state = "merged"
	}
	return &PullRequest{
		Number:       p.Number,
		Title:        p.Title,
		Body:         p.Body,
		State:        state,
		Author:       p.User.Login,
		Labels:       labels,
		Draft:        p.Draft,
		Merged:       p.Merged,
		HeadBranch:   p.Head.Ref,
		HeadSHA:      p.Head.SHA,
		BaseBranch:   p.Base.Ref,
		Additions:    p.Additions,
		Deletions:    p.Deletions,
		ChangedFiles: p.ChangedFiles,
		Commits:      p.Commits,
		Comments:     p.Comments,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		ClosedAt:     p.ClosedAt,
		MergedAt:     p.MergedAt,
		HTMLURL:      p.HTMLURL,
	}
}

// --- Client interface methods ------------------------------------------

func (g *GiteaClient) GetIssue(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	var gi giteaIssue
	if err := g.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number), nil, &gi); err != nil {
		return nil, err
	}
	return convertGiteaIssue(&gi), nil
}

func (g *GiteaClient) ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]*Issue, error) {
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if len(opts.Labels) > 0 {
		q.Set("labels", strings.Join(opts.Labels, ","))
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.PerPage > 0 {
		q.Set("limit", strconv.Itoa(opts.PerPage))
	}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	// Gitea returns issues + PRs from /issues; type=issues filters out PRs.
	q.Set("type", "issues")

	var raw []giteaIssue
	path := fmt.Sprintf("/repos/%s/%s/issues?%s", owner, repo, q.Encode())
	if err := g.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]*Issue, 0, len(raw))
	for i := range raw {
		out = append(out, convertGiteaIssue(&raw[i]))
	}
	return out, nil
}

func (g *GiteaClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	var p giteaPR
	if err := g.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number), nil, &p); err != nil {
		return nil, err
	}
	return convertGiteaPR(&p), nil
}

func (g *GiteaClient) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestsOptions) ([]*PullRequest, error) {
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.PerPage > 0 {
		q.Set("limit", strconv.Itoa(opts.PerPage))
	}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	var raw []giteaPR
	path := fmt.Sprintf("/repos/%s/%s/pulls?%s", owner, repo, q.Encode())
	if err := g.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]*PullRequest, 0, len(raw))
	for i := range raw {
		out = append(out, convertGiteaPR(&raw[i]))
	}
	return out, nil
}

func (g *GiteaClient) ListPullRequestCommits(ctx context.Context, owner, repo string, number int) ([]*Commit, error) {
	var raw []giteaCommit
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/commits", owner, repo, number)
	if err := g.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]*Commit, 0, len(raw))
	for _, c := range raw {
		out = append(out, &Commit{
			SHA:     c.SHA,
			Message: c.Commit.Message,
			Author:  c.Commit.Author.Name,
			Date:    c.Commit.Author.Date,
			HTMLURL: c.HTMLURL,
		})
	}
	return out, nil
}

func (g *GiteaClient) GetCommitChecks(ctx context.Context, owner, repo, ref string) ([]*CheckRun, error) {
	// Gitea's commit-status API returns the per-context build state. There
	// is no "queued"/"in_progress" stage in the same shape as GitHub
	// check-runs, so map state → conclusion + status="completed".
	var raw []giteaCheckRun
	path := fmt.Sprintf("/repos/%s/%s/statuses/%s", owner, repo, ref)
	if err := g.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]*CheckRun, 0, len(raw))
	for _, c := range raw {
		out = append(out, &CheckRun{
			Name:       c.Name,
			Status:     "completed",
			Conclusion: mapGiteaStatusToConclusion(c.Status),
			HTMLURL:    c.URL,
		})
	}
	return out, nil
}

// mapGiteaStatusToConclusion translates Gitea's commit-status states
// (success | failure | pending | error | warning) into the forge-neutral
// conclusion vocabulary (success | failure | neutral | …).
func mapGiteaStatusToConclusion(s string) string {
	switch strings.ToLower(s) {
	case "success":
		return "success"
	case "failure", "error":
		return "failure"
	case "pending":
		return ""
	case "warning":
		return "neutral"
	default:
		return s
	}
}

func (g *GiteaClient) ListIssueComments(ctx context.Context, owner, repo string, number int, limit int) ([]*Comment, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var raw []giteaComment
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments?%s", owner, repo, number, q.Encode())
	if err := g.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]*Comment, 0, len(raw))
	for _, c := range raw {
		out = append(out, &Comment{
			Author:    c.User.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
			HTMLURL:   c.HTMLURL,
		})
	}
	return out, nil
}

func (g *GiteaClient) CreatePullRequestReview(ctx context.Context, owner, repo string, number int, event, body string) error {
	// Gitea uses {"event": "APPROVED"} not "APPROVE"; map the canonical
	// values the Client interface documents into Gitea's vocabulary.
	giteaEvent := mapReviewEvent(event)
	payload := struct {
		Event string `json:"event"`
		Body  string `json:"body"`
	}{Event: giteaEvent, Body: body}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	return g.do(ctx, http.MethodPost, path, strings.NewReader(string(b)), nil)
}

func mapReviewEvent(canonical string) string {
	switch strings.ToUpper(canonical) {
	case "APPROVE", "APPROVED":
		return "APPROVED"
	case "REQUEST_CHANGES":
		return "REQUEST_CHANGES"
	case "COMMENT", "COMMENTED":
		return "COMMENT"
	default:
		return canonical
	}
}

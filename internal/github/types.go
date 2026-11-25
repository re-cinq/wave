package github

import (
	"fmt"
	"time"
)

// Issue represents a GitHub issue
type Issue struct {
	ID            int64      `json:"id"`
	Number        int        `json:"number"`
	State         string     `json:"state"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	User          *User      `json:"user"`
	Labels        []*Label   `json:"labels"`
	Assignees     []*User    `json:"assignees"`
	Milestone     *Milestone `json:"milestone,omitempty"`
	Comments      int        `json:"comments"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	HTMLURL       string     `json:"html_url"`
	RepositoryURL string     `json:"repository_url"`
	PullRequest   *struct {
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	} `json:"pull_request,omitempty"`
}

// IsPullRequest returns true if the issue is actually a pull request
func (i *Issue) IsPullRequest() bool {
	return i.PullRequest != nil
}

// User represents a GitHub user
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Type      string `json:"type"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// Label represents a GitHub label
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// Milestone represents a GitHub milestone
type Milestone struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DueOn       *time.Time `json:"due_on,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// IssueComment represents a comment on an issue or pull request
type IssueComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      *User     `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID           int64      `json:"id"`
	Number       int        `json:"number"`
	State        string     `json:"state"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	User         *User      `json:"user"`
	Labels       []*Label   `json:"labels"`
	Assignees    []*User    `json:"assignees"`
	Milestone    *Milestone `json:"milestone,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	Head         *GitRef    `json:"head"`
	Base         *GitRef    `json:"base"`
	HTMLURL      string     `json:"html_url"`
	Draft        bool       `json:"draft"`
	Merged       bool       `json:"merged"`
	Mergeable    *bool      `json:"mergeable,omitempty"`
	Comments     int        `json:"comments"`
	Commits      int        `json:"commits"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
}

// GitRef represents a git reference (branch/tag)
type GitRef struct {
	Label string      `json:"label"`
	Ref   string      `json:"ref"`
	SHA   string      `json:"sha"`
	User  *User       `json:"user,omitempty"`
	Repo  *Repository `json:"repo,omitempty"`
}

// Repository represents a GitHub repository
type Repository struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	Private         bool      `json:"private"`
	Owner           *User     `json:"owner"`
	HTMLURL         string    `json:"html_url"`
	DefaultBranch   string    `json:"default_branch"`
	Language        string    `json:"language"`
	StargazersCount int       `json:"stargazers_count"`
	ForksCount      int       `json:"forks_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PushedAt        time.Time `json:"pushed_at"`
}

// ListIssuesOptions specifies options for listing issues
type ListIssuesOptions struct {
	State     string     // open, closed, all
	Labels    []string   // Label names to filter by
	Sort      string     // created, updated, comments
	Direction string     // asc, desc
	Since     *time.Time // Only issues updated after this time
	PerPage   int        // Results per page (max 100)
	Page      int        // Page number
}

// ListPullRequestsOptions specifies options for listing pull requests
type ListPullRequestsOptions struct {
	State     string // open, closed, all
	Sort      string // created, updated, popularity, long-running
	Direction string // asc, desc
	PerPage   int    // Results per page (max 100)
	Page      int    // Page number
}

// PullRequestCommit represents a commit on a pull request.
type PullRequestCommit struct {
	SHA     string                  `json:"sha"`
	Commit  PullRequestCommitDetail `json:"commit"`
	Author  *User                   `json:"author"`
	HTMLURL string                  `json:"html_url"`
}

// PullRequestCommitDetail holds the nested commit object returned by the GitHub API.
type PullRequestCommitDetail struct {
	Message string                  `json:"message"`
	Author  PullRequestCommitAuthor `json:"author"`
}

// PullRequestCommitAuthor holds the author info inside a commit object.
type PullRequestCommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// APIError represents a GitHub API error response
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Errors     []struct {
		Resource string `json:"resource"`
		Field    string `json:"field"`
		Code     string `json:"code"`
	} `json:"errors,omitempty"`
	DocumentationURL string `json:"documentation_url,omitempty"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("GitHub API error (status %d): %s - %+v", e.StatusCode, e.Message, e.Errors)
	}
	return fmt.Sprintf("GitHub API error (status %d): %s", e.StatusCode, e.Message)
}

// CheckRun represents a GitHub Check Run.
type CheckRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`     // "queued", "in_progress", "completed"
	Conclusion string `json:"conclusion"` // "success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"
	HTMLURL    string `json:"html_url"`
}

// CheckRunsResponse is the response from the GitHub check-runs API.
type CheckRunsResponse struct {
	TotalCount int         `json:"total_count"`
	CheckRuns  []*CheckRun `json:"check_runs"`
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	ResetTime time.Time
	Message   string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s (resets at %s)", e.Message, e.ResetTime.Format(time.RFC3339))
}

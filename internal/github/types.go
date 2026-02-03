package github

import (
	"fmt"
	"time"
)

// Issue represents a GitHub issue
type Issue struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	State       string     `json:"state"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	User        *User      `json:"user"`
	Labels      []*Label   `json:"labels"`
	Assignees   []*User    `json:"assignees"`
	Milestone   *Milestone `json:"milestone,omitempty"`
	Comments    int        `json:"comments"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	HTMLURL     string     `json:"html_url"`
	RepositoryURL string   `json:"repository_url"`
	PullRequest *struct {
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
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	State       string     `json:"state"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	User        *User      `json:"user"`
	Labels      []*Label   `json:"labels"`
	Assignees   []*User    `json:"assignees"`
	Milestone   *Milestone `json:"milestone,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	MergedAt    *time.Time `json:"merged_at,omitempty"`
	Head        *GitRef    `json:"head"`
	Base        *GitRef    `json:"base"`
	HTMLURL     string     `json:"html_url"`
	Merged      bool       `json:"merged"`
	Mergeable   *bool      `json:"mergeable,omitempty"`
	Comments    int        `json:"comments"`
	Commits     int        `json:"commits"`
	Additions   int        `json:"additions"`
	Deletions   int        `json:"deletions"`
	ChangedFiles int       `json:"changed_files"`
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

// Reference represents a git reference
type Reference struct {
	Ref    string     `json:"ref"`
	NodeID string     `json:"node_id"`
	URL    string     `json:"url"`
	Object RefObject  `json:"object"`
}

// RefObject represents the object a reference points to
type RefObject struct {
	Type string `json:"type"`
	SHA  string `json:"sha"`
	URL  string `json:"url"`
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

// IssueUpdate represents fields that can be updated on an issue
type IssueUpdate struct {
	Title     *string   `json:"title,omitempty"`
	Body      *string   `json:"body,omitempty"`
	State     *string   `json:"state,omitempty"`
	Labels    *[]string `json:"labels,omitempty"`
	Assignees *[]string `json:"assignees,omitempty"`
	Milestone *int      `json:"milestone,omitempty"`
}

// CreatePullRequestRequest represents a request to create a pull request
type CreatePullRequestRequest struct {
	Title               string  `json:"title"`
	Body                string  `json:"body,omitempty"`
	Head                string  `json:"head"` // branch name
	Base                string  `json:"base"` // target branch
	MaintainerCanModify *bool   `json:"maintainer_can_modify,omitempty"`
	Draft               *bool   `json:"draft,omitempty"`
}

// RateLimitStatus represents GitHub API rate limit information
type RateLimitStatus struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     int64     `json:"reset"` // Unix timestamp
	Used      int       `json:"used"`
}

// ResetTime returns the rate limit reset time
func (r *RateLimitStatus) ResetTime() time.Time {
	return time.Unix(r.Reset, 0)
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

// RateLimitError represents a rate limit error
type RateLimitError struct {
	ResetTime time.Time
	Message   string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s (resets at %s)", e.Message, e.ResetTime.Format(time.RFC3339))
}

// IssueFilter provides a fluent interface for filtering issues
type IssueFilter struct {
	MinTitleLength int
	MaxTitleLength int
	MinBodyLength  int
	MaxBodyLength  int
	RequireBody    bool
	States         []string
	Labels         []string
}

// Matches returns true if the issue matches the filter criteria
func (f *IssueFilter) Matches(issue *Issue) bool {
	// Skip pull requests if this is an issue filter
	if issue.IsPullRequest() {
		return false
	}

	// Check title length
	titleLen := len(issue.Title)
	if f.MinTitleLength > 0 && titleLen < f.MinTitleLength {
		return false
	}
	if f.MaxTitleLength > 0 && titleLen > f.MaxTitleLength {
		return false
	}

	// Check body length
	bodyLen := len(issue.Body)
	if f.RequireBody && bodyLen == 0 {
		return false
	}
	if f.MinBodyLength > 0 && bodyLen < f.MinBodyLength {
		return false
	}
	if f.MaxBodyLength > 0 && bodyLen > f.MaxBodyLength {
		return false
	}

	// Check state
	if len(f.States) > 0 {
		stateMatch := false
		for _, state := range f.States {
			if issue.State == state {
				stateMatch = true
				break
			}
		}
		if !stateMatch {
			return false
		}
	}

	// Check labels
	if len(f.Labels) > 0 {
		issueLabels := make(map[string]bool)
		for _, label := range issue.Labels {
			issueLabels[label.Name] = true
		}
		for _, requiredLabel := range f.Labels {
			if !issueLabels[requiredLabel] {
				return false
			}
		}
	}

	return true
}

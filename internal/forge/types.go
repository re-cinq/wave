package forge

import "time"

// Label is a forge-neutral representation of a label with its display color.
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"` // hex without #, e.g. "e4e669"
}

// Issue is a forge-neutral representation of an issue/work-item.
type Issue struct {
	Number    int
	Title     string
	Body      string
	State     string // "open", "closed"
	Author    string
	Labels    []Label
	Assignees []string
	Comments  int
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
	HTMLURL   string
	IsPR      bool
}

// PullRequest is a forge-neutral representation of a PR/MR.
type PullRequest struct {
	Number       int
	Title        string
	Body         string
	State        string // "open", "closed", "merged"
	Author       string
	Labels       []Label
	Draft        bool
	Merged       bool
	HeadBranch   string
	HeadSHA      string
	BaseBranch   string
	Additions    int
	Deletions    int
	ChangedFiles int
	Commits      int
	Comments     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
	MergedAt     *time.Time
	HTMLURL      string
}

// ListIssuesOptions configures issue listing.
type ListIssuesOptions struct {
	State   string // "open", "closed", "all"
	Labels  []string
	Sort    string // "created", "updated"
	PerPage int
	Page    int
}

// ListPullRequestsOptions configures PR listing.
type ListPullRequestsOptions struct {
	State   string // "open", "closed", "all"
	Sort    string
	PerPage int
	Page    int
}

// Comment is a forge-neutral representation of a comment on an issue or PR.
type Comment struct {
	Author    string
	Body      string
	CreatedAt time.Time
	HTMLURL   string
}

// Commit is a forge-neutral representation of a commit on a pull request.
type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
	HTMLURL string
}

// CheckRun represents a CI/CD check result for a commit.
type CheckRun struct {
	Name       string // check name (e.g. "CI / build")
	Status     string // "queued", "in_progress", "completed"
	Conclusion string // "success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required" (empty if not completed)
	HTMLURL    string // link to the check details
}

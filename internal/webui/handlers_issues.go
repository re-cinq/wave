package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/timeouts"
)

// handleIssuesPage handles GET /issues - serves the HTML issues page.
func (s *Server) handleIssuesPage(w http.ResponseWriter, r *http.Request) {
	stateFilter := validateStateFilter(r.URL.Query().Get("state"))
	page := parsePageNumber(r)
	issueData := s.getIssueListData(stateFilter, page)

	data := struct {
		ActivePage string
		IssueListResponse
	}{
		ActivePage:        "issues",
		IssueListResponse: issueData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/issues.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIIssues handles GET /api/issues - returns issue list as JSON.
func (s *Server) handleAPIIssues(w http.ResponseWriter, r *http.Request) {
	stateFilter := validateStateFilter(r.URL.Query().Get("state"))
	page := parsePageNumber(r)
	data := s.getIssueListData(stateFilter, page)
	writeJSON(w, http.StatusOK, data)
}

// handleAPIStartFromIssue handles POST /api/issues/start - launches a pipeline from an issue.
func (s *Server) handleAPIStartFromIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueURL     string `json:"issue_url"`
		PipelineName string `json:"pipeline_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IssueURL == "" || req.PipelineName == "" {
		writeJSONError(w, http.StatusBadRequest, "issue_url and pipeline_name are required")
		return
	}

	// Delegate to the existing pipeline start logic
	pl, err := loadPipelineYAML(req.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "pipeline not found: "+req.PipelineName)
		return
	}

	runID, err := s.rwStore.CreateRun(req.PipelineName, req.IssueURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	s.launchPipelineExecution(runID, req.PipelineName, req.IssueURL, pl, RunOptions{})

	writeJSON(w, http.StatusCreated, StartPipelineResponse{
		RunID:        runID,
		PipelineName: req.PipelineName,
		Status:       "running",
		StartedAt:    time.Now().UTC(),
	})
}

// handleIssueDetailPage handles GET /issues/{number} - serves issue detail with related runs.
func (s *Server) handleIssueDetailPage(w http.ResponseWriter, r *http.Request) {
	numberStr := r.PathValue("number")
	number := parsePageNumber2(numberStr)
	if number <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}

	if s.forgeClient == nil || s.repoSlug == "" {
		http.Error(w, "Forge integration not configured", http.StatusServiceUnavailable)
		return
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPI)
	defer cancel()

	issue, err := s.forgeClient.GetIssue(ctx, owner, repo, number)
	if err != nil {
		http.Error(w, "issue not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Build issue URL pattern to find related runs
	issueURL := issue.HTMLURL
	// Also try shortened pattern
	patterns := []string{issueURL}
	if issueURL != "" {
		// Also match just the issue number path segment
		patterns = append(patterns, "/issues/"+numberStr)
	}

	// Find runs whose input contains this issue URL
	allRuns, err := s.store.ListRuns(state.ListRunsOptions{Limit: 500})
	if err != nil {
		allRuns = nil
	}
	var relatedRuns []RunSummary
	for _, run := range allRuns {
		if run.Input == "" {
			continue
		}
		for _, pat := range patterns {
			if strings.Contains(run.Input, pat) {
				relatedRuns = append(relatedRuns, runToSummary(run))
				break
			}
		}
	}

	// Fetch last 10 comments
	var comments []CommentSummary
	forgeComments, err := s.forgeClient.ListIssueComments(ctx, owner, repo, number, 10)
	if err == nil {
		for _, c := range forgeComments {
			comments = append(comments, CommentSummary{
				Author:    c.Author,
				Body:      c.Body,
				CreatedAt: c.CreatedAt.Format("2006-01-02 15:04"),
				TimeISO:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
				HTMLURL:   c.HTMLURL,
			})
		}
	}

	data := struct {
		ActivePage string
		Issue      IssueDetail
		Runs       []RunSummary
		Comments   []CommentSummary
	}{
		ActivePage: "issues",
		Issue: IssueDetail{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     issue.State,
			Body:      issue.Body,
			Author:    issue.Author,
			Labels:    issue.Labels,
			Assignees: issue.Assignees,
			Comments:  issue.Comments,
			CreatedAt: issue.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt: issue.UpdatedAt.Format("2006-01-02 15:04"),
			URL:       issue.HTMLURL,
		},
		Runs:     relatedRuns,
		Comments: comments,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/issue_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func parsePageNumber2(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

const issuesPerPage = 50

func (s *Server) getIssueListData(stateFilter string, page int) IssueListResponse {
	if s.forgeClient == nil || s.repoSlug == "" {
		return IssueListResponse{
			Issues:      []IssueSummary{},
			FilterState: stateFilter,
			Page:        page,
			Message:     "Forge integration not configured. Set a forge token (GH_TOKEN, GITLAB_TOKEN, etc.) to enable.",
		}
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	if owner == "" {
		return IssueListResponse{
			Issues:      []IssueSummary{},
			FilterState: stateFilter,
			Page:        page,
			Message:     "Could not determine repository from git remote.",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPIList)
	defer cancel()

	issues, err := s.forgeClient.ListIssues(ctx, owner, repo, forge.ListIssuesOptions{
		State:   stateFilter,
		PerPage: issuesPerPage + 1, // fetch one extra to detect HasMore
		Page:    page,
	})
	if err != nil {
		return IssueListResponse{
			Issues:      []IssueSummary{},
			FilterState: stateFilter,
			Page:        page,
			Message:     "Failed to fetch issues: " + err.Error(),
		}
	}

	hasMore := len(issues) > issuesPerPage
	if hasMore {
		issues = issues[:issuesPerPage]
	}

	var summaries []IssueSummary
	for _, issue := range issues {
		if issue.IsPR {
			continue
		}
		summaries = append(summaries, IssueSummary{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     issue.State,
			Author:    issue.Author,
			Labels:    issue.Labels,
			Comments:  issue.Comments,
			CreatedAt: issue.CreatedAt.Format("2006-01-02"),
			URL:       issue.HTMLURL,
		})
	}

	if summaries == nil {
		summaries = []IssueSummary{}
	}

	return IssueListResponse{
		Issues:      summaries,
		RepoSlug:    s.repoSlug,
		FilterState: stateFilter,
		Page:        page,
		HasMore:     hasMore,
	}
}

// splitRepoSlug splits "owner/repo" into owner and repo parts.
func splitRepoSlug(slug string) (string, string) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

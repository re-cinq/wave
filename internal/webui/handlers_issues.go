package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/timeouts"
)

// handleIssuesPage handles GET /issues - serves the HTML issues page.
func (s *Server) handleIssuesPage(w http.ResponseWriter, r *http.Request) {
	stateFilter := r.URL.Query().Get("state")
	if stateFilter == "" {
		stateFilter = "open"
	}
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
	if err := s.assets.templates["templates/issues.html"].Execute(w, data); err != nil {
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
	var req StartIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IssueURL == "" || req.PipelineName == "" {
		writeJSONError(w, http.StatusBadRequest, "issue_url and pipeline_name are required")
		return
	}

	// Validate mutual exclusions
	if req.Continuous && req.FromStep != "" {
		writeJSONError(w, http.StatusBadRequest, "--continuous and --from-step are mutually exclusive")
		return
	}
	if req.OnFailure != "" && req.OnFailure != "halt" && req.OnFailure != "skip" {
		writeJSONError(w, http.StatusBadRequest, "on_failure must be 'halt' or 'skip'")
		return
	}

	if _, err := loadPipelineYAML(req.PipelineName); err != nil {
		writeJSONError(w, http.StatusBadRequest, "pipeline not found: "+req.PipelineName)
		return
	}

	runID, err := s.runtime.rwStore.CreateRun(req.PipelineName, req.IssueURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create run: "+err.Error())
		return
	}

	opts := RunOptions{
		Model:         req.Model,
		Adapter:       req.Adapter,
		DryRun:        req.DryRun,
		FromStep:      req.FromStep,
		Force:         req.Force,
		Detach:        req.Detach,
		Timeout:       req.Timeout,
		Steps:         req.Steps,
		Exclude:       req.Exclude,
		OnFailure:     req.OnFailure,
		Continuous:    req.Continuous,
		Source:        req.Source,
		MaxIterations: req.MaxIterations,
		Delay:         req.Delay,
	}

	if req.FromStep != "" {
		s.launchPipelineExecution(runID, req.PipelineName, req.IssueURL, opts, req.FromStep)
	} else {
		s.launchPipelineExecution(runID, req.PipelineName, req.IssueURL, opts)
	}

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

	if s.runtime.forgeClient == nil || s.runtime.repoSlug == "" {
		http.Error(w, "Forge integration not configured", http.StatusServiceUnavailable)
		return
	}

	owner, repo := splitRepoSlug(s.runtime.repoSlug)
	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPI)
	defer cancel()

	issue, err := s.runtime.forgeClient.GetIssue(ctx, owner, repo, number)
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
	allRuns, err := s.runtime.store.ListRuns(state.ListRunsOptions{Limit: 500})
	if err != nil {
		allRuns = nil
	}
	var relatedRuns []RunSummary
	for _, run := range allRuns {
		if run.Input == "" {
			continue
		}
		matched := false
		for _, pat := range patterns {
			if strings.Contains(run.Input, pat) {
				matched = true
				break
			}
		}
		// Also match short-form "owner/repo <number>" input
		if !matched {
			if n := extractIssueNumber(run.Input); n == number {
				matched = true
			}
		}
		if matched {
			relatedRuns = append(relatedRuns, runToSummary(run))
		}
	}

	// Fetch last 10 comments
	var comments []CommentSummary
	forgeComments, err := s.runtime.forgeClient.ListIssueComments(ctx, owner, repo, number, 10)
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

	// Compute aggregate Wave stats
	runCount := len(relatedRuns)
	totalTokens := 0
	lastStatus := ""
	for _, r := range relatedRuns {
		totalTokens += r.TotalTokens
		if lastStatus == "" {
			lastStatus = r.Status
		}
	}

	data := struct {
		ActivePage  string
		Issue       IssueDetail
		Runs        []RunSummary
		Comments    []CommentSummary
		RunCount    int
		TotalTokens int
		LastStatus  string
	}{
		ActivePage: "issues",
		Issue: IssueDetail{
			Number:    issue.Number,
			Title:     issue.Title,
			State:     issue.State,
			Body:      issue.Body,
			Author:    issue.Author,
			Labels:    forgeLabelsToBadges(issue.Labels),
			Assignees: issue.Assignees,
			Comments:  issue.Comments,
			CreatedAt: issue.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt: issue.UpdatedAt.Format("2006-01-02 15:04"),
			URL:       issue.HTMLURL,
		},
		Runs:        relatedRuns,
		Comments:    comments,
		RunCount:    runCount,
		TotalTokens: totalTokens,
		LastStatus:  lastStatus,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.assets.templates["templates/issue_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
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

const issuesPerPage = 10

func (s *Server) getIssueListData(stateFilter string, page int) IssueListResponse {
	if s.runtime.forgeClient == nil || s.runtime.repoSlug == "" {
		return IssueListResponse{
			Issues:      []IssueSummary{},
			FilterState: stateFilter,
			Page:        page,
			Message:     "Forge integration not configured. Set a forge token (GH_TOKEN, GITLAB_TOKEN, etc.) to enable.",
		}
	}

	owner, repo := splitRepoSlug(s.runtime.repoSlug)
	if owner == "" {
		return IssueListResponse{
			Issues:      []IssueSummary{},
			FilterState: stateFilter,
			Page:        page,
			Message:     "Could not determine repository from git remote.",
		}
	}

	// Don't cache "running" state — stale data is misleading
	cacheKey := fmt.Sprintf("issues:list:%s:%d", stateFilter, page)
	useCache := stateFilter != "running"
	if useCache {
		if cached, ok := s.assets.cache.Get(cacheKey); ok {
			return cached.(IssueListResponse)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPIList)
	defer cancel()

	issues, err := s.runtime.forgeClient.ListIssues(ctx, owner, repo, forge.ListIssuesOptions{
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
			Labels:    forgeLabelsToBadges(issue.Labels),
			Comments:  issue.Comments,
			CreatedAt: issue.CreatedAt.Format("2006-01-02"),
			URL:       issue.HTMLURL,
		})
	}

	if summaries == nil {
		summaries = []IssueSummary{}
	}

	// Enrich with Wave run stats
	if s.runtime.store != nil {
		allRuns, err := s.runtime.store.ListRuns(state.ListRunsOptions{Limit: 10000})
		if err == nil {
			enrichSummariesWithRuns(summaries, allRuns, "issue")
		}
	}

	// Count open/closed from current page
	var openCount, closedCount int
	for _, s := range summaries {
		if s.State == "open" {
			openCount++
		} else {
			closedCount++
		}
	}

	result := IssueListResponse{
		Issues:      summaries,
		RepoSlug:    s.runtime.repoSlug,
		FilterState: stateFilter,
		Page:        page,
		HasMore:     hasMore,
		TotalOpen:   openCount,
		TotalClosed: closedCount,
	}

	if useCache {
		s.assets.cache.Set(cacheKey, result)
	}

	return result
}

// splitRepoSlug splits "owner/repo" into owner and repo parts.
func splitRepoSlug(slug string) (string, string) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// forgeLabelsToBadges converts forge.Label slice to LabelBadge slice for the web UI.
func forgeLabelsToBadges(labels []forge.Label) []LabelBadge {
	if len(labels) == 0 {
		return nil
	}
	result := make([]LabelBadge, len(labels))
	for i, l := range labels {
		result[i] = LabelBadge{Name: l.Name, Color: l.Color}
	}
	return result
}

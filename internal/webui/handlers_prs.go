package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/timeouts"
)

// handlePRsPage handles GET /prs - serves the HTML pull requests page.
func (s *Server) handlePRsPage(w http.ResponseWriter, r *http.Request) {
	stateFilter := r.URL.Query().Get("state")
	if stateFilter == "" {
		stateFilter = "open"
	}
	page := parsePageNumber(r)
	prData := s.getPRListData(stateFilter, page)

	data := struct {
		ActivePage string
		PRListResponse
	}{
		ActivePage:     "prs",
		PRListResponse: prData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/prs.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		log.Printf("[webui] template error rendering prs page: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// handleAPIPRs handles GET /api/prs - returns PR list as JSON.
func (s *Server) handleAPIPRs(w http.ResponseWriter, r *http.Request) {
	stateFilter := validateStateFilter(r.URL.Query().Get("state"))
	page := parsePageNumber(r)
	data := s.getPRListData(stateFilter, page)
	writeJSON(w, http.StatusOK, data)
}

// handlePRDetailPage handles GET /prs/{number} - serves PR detail with related runs.
func (s *Server) handlePRDetailPage(w http.ResponseWriter, r *http.Request) {
	numberStr := r.PathValue("number")
	number := parsePageNumber2(numberStr)
	if number <= 0 {
		http.Error(w, "invalid PR number", http.StatusBadRequest)
		return
	}

	if s.forgeClient == nil || s.repoSlug == "" {
		http.Error(w, "Forge integration not configured", http.StatusServiceUnavailable)
		return
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPI)
	defer cancel()

	pr, err := s.forgeClient.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		log.Printf("[webui] failed to fetch PR #%d: %v", number, err)
		http.Error(w, "PR not found", http.StatusNotFound)
		return
	}

	// Find related runs — match by PR URL or head branch name
	prURL := pr.HTMLURL

	allRuns, err := s.store.ListRuns(state.ListRunsOptions{Limit: 500})
	if err != nil {
		allRuns = nil
	}
	var relatedRuns []RunSummary
	for _, run := range allRuns {
		matched := false
		if run.Input != "" && prURL != "" && strings.Contains(run.Input, prURL) {
			matched = true
		}
		if !matched && run.Input != "" && strings.Contains(run.Input, fmt.Sprintf("/pull/%d", number)) {
			matched = true
		}
		if !matched && pr.HeadBranch != "" && run.BranchName == pr.HeadBranch {
			matched = true
		}
		// Match short-form input "owner/repo <number>" (e.g. ops-pr-review gets PR number as input)
		if !matched && run.Input != "" {
			if n := extractIssueNumber(run.Input); n == number {
				matched = true
			}
		}
		if matched {
			relatedRuns = append(relatedRuns, runToSummary(run))
		}
	}

	// Fetch status checks for the PR head commit
	var checks []PRCheck
	if pr.HeadSHA != "" {
		forgeChecks, err := s.forgeClient.GetCommitChecks(ctx, owner, repo, pr.HeadSHA)
		if err != nil {
			log.Printf("[webui] failed to fetch checks for PR #%d (SHA %s): %v", number, pr.HeadSHA, err)
		} else {
			for _, c := range forgeChecks {
				checks = append(checks, PRCheck{
					Name:       c.Name,
					Status:     c.Status,
					Conclusion: c.Conclusion,
					URL:        c.HTMLURL,
				})
			}
		}
	}

	// Fetch commits for the PR
	var commits []CommitSummary
	forgeCommits, err := s.forgeClient.ListPullRequestCommits(ctx, owner, repo, number)
	if err != nil {
		log.Printf("[webui] failed to fetch commits for PR #%d: %v", number, err)
	} else {
		for _, fc := range forgeCommits {
			msg := fc.Message
			if idx := strings.Index(msg, "\n"); idx >= 0 {
				msg = msg[:idx]
			}
			shortSHA := fc.SHA
			if len(shortSHA) > 7 {
				shortSHA = shortSHA[:7]
			}
			commits = append(commits, CommitSummary{
				SHA:      fc.SHA,
				ShortSHA: shortSHA,
				Message:  msg,
				Author:   fc.Author,
				Date:     fc.Date.Format("2006-01-02 15:04"),
				TimeISO:  fc.Date.Format("2006-01-02T15:04:05Z"),
				HTMLURL:  fc.HTMLURL,
			})
		}
	}

	// Fetch last 10 comments
	var comments []CommentSummary
	forgeComments, err := s.forgeClient.ListIssueComments(ctx, owner, repo, number, 10)
	if err != nil {
		log.Printf("[webui] failed to fetch comments for PR #%d: %v", number, err)
	} else {
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
		ActivePage    string
		PR            PRDetail
		Runs          []RunSummary
		Comments      []CommentSummary
		CommitDetails []CommitSummary
		RunCount      int
		TotalTokens   int
		LastStatus    string
	}{
		ActivePage: "prs",
		PR: PRDetail{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Body:         pr.Body,
			Author:       pr.Author,
			Labels:       forgeLabelsToBadges(pr.Labels),
			Draft:        pr.Draft,
			Merged:       pr.Merged,
			HeadBranch:   pr.HeadBranch,
			BaseBranch:   pr.BaseBranch,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
			ChangedFiles: pr.ChangedFiles,
			Commits:      pr.Commits,
			Comments:     pr.Comments,
			CreatedAt:    pr.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt:    pr.UpdatedAt.Format("2006-01-02 15:04"),
			URL:          pr.HTMLURL,
			Checks:       checks,
		},
		Runs:          relatedRuns,
		Comments:      comments,
		CommitDetails: commits,
		RunCount:      runCount,
		TotalTokens:   totalTokens,
		LastStatus:    lastStatus,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pr_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		log.Printf("[webui] template error rendering PR detail page: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

const prsPerPage = 50

// enrichPRStats concurrently fetches individual PR details to populate
// Additions/Deletions/ChangedFiles and CI check status, which the list endpoint omits.
// The returned map is keyed by PR number and contains the aggregate check status.
func enrichPRStats(ctx context.Context, client forge.Client, owner, repo string, prs []*forge.PullRequest) map[int]string {
	const workers = 5
	ch := make(chan int, len(prs))
	for i := range prs {
		ch <- i
	}
	close(ch)

	checkStatuses := make(map[int]string, len(prs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range min(workers, len(prs)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range ch {
				detail, err := client.GetPullRequest(ctx, owner, repo, prs[idx].Number)
				if err != nil {
					log.Printf("[webui] failed to enrich PR #%d: %v", prs[idx].Number, err)
					continue
				}
				prs[idx].Additions = detail.Additions
				prs[idx].Deletions = detail.Deletions
				prs[idx].ChangedFiles = detail.ChangedFiles

				// Fetch CI check status for the HEAD commit
				if detail.HeadSHA != "" {
					checks, err := client.GetCommitChecks(ctx, owner, repo, detail.HeadSHA)
					if err != nil {
						log.Printf("[webui] failed to fetch checks for PR #%d: %v", prs[idx].Number, err)
					} else {
						status := aggregateCheckStatus(checks)
						mu.Lock()
						checkStatuses[prs[idx].Number] = status
						mu.Unlock()
					}
				}
			}
		}()
	}
	wg.Wait()
	return checkStatuses
}

// aggregateCheckStatus derives a single status from a list of check runs.
// Returns "success" if all completed successfully, "failure" if any failed,
// "pending" if any are still in progress/queued, or "" if no checks exist.
func aggregateCheckStatus(checks []*forge.CheckRun) string {
	if len(checks) == 0 {
		return ""
	}
	hasPending := false
	for _, c := range checks {
		if c.Status != "completed" {
			hasPending = true
			continue
		}
		switch c.Conclusion {
		case "failure", "timed_out", "action_required":
			return "failure"
		case "cancelled":
			return "failure"
		}
	}
	if hasPending {
		return "pending"
	}
	return "success"
}

func (s *Server) getPRListData(stateFilter string, page int) PRListResponse {
	if s.forgeClient == nil || s.repoSlug == "" {
		return PRListResponse{
			PullRequests: []PRSummary{},
			FilterState:  stateFilter,
			Page:         page,
			Message:      "Forge integration not configured. Set a forge token (GH_TOKEN, GITLAB_TOKEN, etc.) to enable.",
		}
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	if owner == "" {
		return PRListResponse{
			PullRequests: []PRSummary{},
			FilterState:  stateFilter,
			Page:         page,
			Message:      "Could not determine repository from git remote.",
		}
	}

	// Don't cache "running" state — stale data is misleading
	cacheKey := fmt.Sprintf("prs:list:%s:%d", stateFilter, page)
	useCache := stateFilter != "running"
	if useCache {
		if cached, ok := s.cache.Get(cacheKey); ok {
			return cached.(PRListResponse)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPI)
	defer cancel()

	prs, err := s.forgeClient.ListPullRequests(ctx, owner, repo, forge.ListPullRequestsOptions{
		State:   stateFilter,
		PerPage: prsPerPage + 1, // fetch one extra to detect HasMore
		Page:    page,
	})
	if err != nil {
		log.Printf("[webui] failed to fetch pull requests: %v", err)
		return PRListResponse{
			PullRequests: []PRSummary{},
			FilterState:  stateFilter,
			Page:         page,
			Message:      "Failed to fetch pull requests. Check server logs for details.",
		}
	}

	hasMore := len(prs) > prsPerPage
	if hasMore {
		prs = prs[:prsPerPage]
	}

	checkStatuses := enrichPRStats(ctx, s.forgeClient, owner, repo, prs)

	var summaries []PRSummary
	for _, pr := range prs {
		summaries = append(summaries, PRSummary{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Author:       pr.Author,
			Labels:       forgeLabelsToBadges(pr.Labels),
			Draft:        pr.Draft,
			Merged:       pr.Merged,
			HeadBranch:   pr.HeadBranch,
			BaseBranch:   pr.BaseBranch,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
			ChangedFiles: pr.ChangedFiles,
			CreatedAt:    pr.CreatedAt.Format("2006-01-02"),
			URL:          pr.HTMLURL,
			CheckStatus:  checkStatuses[pr.Number],
		})
	}

	if summaries == nil {
		summaries = []PRSummary{}
	}

	// Enrich with Wave run stats
	if s.store != nil {
		allRuns, err := s.store.ListRuns(state.ListRunsOptions{Limit: 10000})
		if err == nil {
			enrichPRSummariesWithRuns(summaries, allRuns, s.store)
		}
	}

	var openCount, closedCount int
	for _, s := range summaries {
		if s.State == "open" || s.Draft {
			openCount++
		} else {
			closedCount++
		}
	}

	result := PRListResponse{
		PullRequests: summaries,
		RepoSlug:     s.repoSlug,
		FilterState:  stateFilter,
		Page:         page,
		HasMore:      hasMore,
		TotalOpen:    openCount,
		TotalClosed:  closedCount,
	}

	if useCache {
		s.cache.Set(cacheKey, result)
	}

	return result
}

// PRReviewRequest is the JSON body for POST /api/prs/{number}/review.
type PRReviewRequest struct {
	Event string `json:"event"` // "APPROVE", "REQUEST_CHANGES", or "COMMENT"
	Body  string `json:"body"`
}

// validReviewEvents is the set of accepted review event types.
var validReviewEvents = map[string]bool{
	"APPROVE":         true,
	"REQUEST_CHANGES": true,
	"COMMENT":         true,
}

// handlePRReview handles POST /api/prs/{number}/review — submits a PR review.
func (s *Server) handlePRReview(w http.ResponseWriter, r *http.Request) {
	// CSRF protection: require a custom header that triggers CORS preflight
	// for cross-origin requests, preventing drive-by review submissions.
	if r.Header.Get("X-Wave-Request") != "1" {
		writeJSONError(w, http.StatusForbidden, "missing required X-Wave-Request header")
		return
	}

	if s.forgeClient == nil || s.repoSlug == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "forge integration not configured")
		return
	}

	numberStr := r.PathValue("number")
	number := parsePageNumber2(numberStr)
	if number <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid PR number")
		return
	}

	var req PRReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !validReviewEvents[req.Event] {
		writeJSONError(w, http.StatusBadRequest, "event must be APPROVE, REQUEST_CHANGES, or COMMENT")
		return
	}

	if req.Event == "REQUEST_CHANGES" && strings.TrimSpace(req.Body) == "" {
		writeJSONError(w, http.StatusBadRequest, "body is required when requesting changes")
		return
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPI)
	defer cancel()

	if err := s.forgeClient.CreatePullRequestReview(ctx, owner, repo, number, req.Event, req.Body); err != nil {
		log.Printf("[webui] failed to submit review for PR #%d: %v", number, err)
		writeJSONError(w, http.StatusBadGateway, "failed to submit review: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "event": req.Event})
}

// handleAPIStartFromPR handles POST /api/prs/start - launches a pipeline from a PR.
func (s *Server) handleAPIStartFromPR(w http.ResponseWriter, r *http.Request) {
	var req StartPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PRURL == "" || req.PipelineName == "" {
		writeJSONError(w, http.StatusBadRequest, "pr_url and pipeline_name are required")
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

	pl, err := loadPipelineYAML(req.PipelineName)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "pipeline not found: "+req.PipelineName)
		return
	}

	runID, err := s.rwStore.CreateRun(req.PipelineName, req.PRURL)
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
		s.launchPipelineExecution(runID, req.PipelineName, req.PRURL, pl, opts, req.FromStep)
	} else {
		s.launchPipelineExecution(runID, req.PipelineName, req.PRURL, pl, opts)
	}

	writeJSON(w, http.StatusCreated, StartPipelineResponse{
		RunID:        runID,
		PipelineName: req.PipelineName,
		Status:       "running",
		StartedAt:    time.Now().UTC(),
	})
}

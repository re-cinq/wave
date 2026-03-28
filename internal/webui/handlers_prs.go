package webui

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/state"
	"github.com/recinq/wave/internal/timeouts"
)

// handlePRsPage handles GET /prs - serves the HTML pull requests page.
func (s *Server) handlePRsPage(w http.ResponseWriter, r *http.Request) {
	stateFilter := validateStateFilter(r.URL.Query().Get("state"))
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
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "PR not found: "+err.Error(), http.StatusNotFound)
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
		if matched {
			relatedRuns = append(relatedRuns, runToSummary(run))
		}
	}

	// Fetch status checks for the PR head commit
	var checks []PRCheck
	if pr.HeadSHA != "" {
		forgeChecks, err := s.forgeClient.GetCommitChecks(ctx, owner, repo, pr.HeadSHA)
		if err == nil {
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
		PR         PRDetail
		Runs       []RunSummary
		Comments   []CommentSummary
	}{
		ActivePage: "prs",
		PR: PRDetail{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Body:         pr.Body,
			Author:       pr.Author,
			Labels:       pr.Labels,
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
		Runs:     relatedRuns,
		Comments: comments,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pr_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

const prsPerPage = 50

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

	ctx, cancel := context.WithTimeout(context.Background(), timeouts.ForgeAPI)
	defer cancel()

	prs, err := s.forgeClient.ListPullRequests(ctx, owner, repo, forge.ListPullRequestsOptions{
		State:   stateFilter,
		PerPage: prsPerPage + 1, // fetch one extra to detect HasMore
		Page:    page,
	})
	if err != nil {
		return PRListResponse{
			PullRequests: []PRSummary{},
			FilterState:  stateFilter,
			Page:         page,
			Message:      "Failed to fetch pull requests: " + err.Error(),
		}
	}

	hasMore := len(prs) > prsPerPage
	if hasMore {
		prs = prs[:prsPerPage]
	}

	var summaries []PRSummary
	for _, pr := range prs {
		summaries = append(summaries, PRSummary{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Author:       pr.Author,
			Labels:       pr.Labels,
			Draft:        pr.Draft,
			Merged:       pr.Merged,
			HeadBranch:   pr.HeadBranch,
			BaseBranch:   pr.BaseBranch,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
			ChangedFiles: pr.ChangedFiles,
			CreatedAt:    pr.CreatedAt.Format("2006-01-02"),
			URL:          pr.HTMLURL,
		})
	}

	if summaries == nil {
		summaries = []PRSummary{}
	}

	return PRListResponse{
		PullRequests: summaries,
		RepoSlug:     s.repoSlug,
		FilterState:  stateFilter,
		Page:         page,
		HasMore:      hasMore,
	}
}

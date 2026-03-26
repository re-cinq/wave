package webui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"github.com/recinq/wave/internal/github"
	"github.com/recinq/wave/internal/timeouts"
	"github.com/recinq/wave/internal/state"
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

	if s.githubClient == nil || s.repoSlug == "" {
		http.Error(w, "GitHub integration not configured", http.StatusServiceUnavailable)
		return
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	ctx, cancel := context.WithTimeout(context.Background(), timeouts.GithubAPI)
	defer cancel()

	pr, err := s.githubClient.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		http.Error(w, "PR not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Find related runs — match by PR URL or head branch name
	prURL := pr.HTMLURL
	headBranch := ""
	if pr.Head != nil {
		headBranch = pr.Head.Ref
	}
	baseBranch := ""
	if pr.Base != nil {
		baseBranch = pr.Base.Ref
	}

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
		if !matched && headBranch != "" && run.BranchName == headBranch {
			matched = true
		}
		if matched {
			relatedRuns = append(relatedRuns, runToSummary(run))
		}
	}

	author := ""
	if pr.User != nil {
		author = pr.User.Login
	}
	var labels []string
	for _, l := range pr.Labels {
		labels = append(labels, l.Name)
	}

	data := struct {
		ActivePage string
		PR         PRDetail
		Runs       []RunSummary
	}{
		ActivePage: "prs",
		PR: PRDetail{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Body:         pr.Body,
			Author:       author,
			Labels:       labels,
			Draft:        pr.Draft,
			Merged:       pr.Merged,
			HeadBranch:   headBranch,
			BaseBranch:   baseBranch,
			Additions:    pr.Additions,
			Deletions:    pr.Deletions,
			ChangedFiles: pr.ChangedFiles,
			Commits:      pr.Commits,
			Comments:     pr.Comments,
			CreatedAt:    pr.CreatedAt.Format("2006-01-02 15:04"),
			UpdatedAt:    pr.UpdatedAt.Format("2006-01-02 15:04"),
			URL:          pr.HTMLURL,
		},
		Runs: relatedRuns,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/pr_detail.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

const prsPerPage = 50

func (s *Server) getPRListData(stateFilter string, page int) PRListResponse {
	if s.githubClient == nil || s.repoSlug == "" {
		return PRListResponse{
			PullRequests: []PRSummary{},
			FilterState:  stateFilter,
			Page:         page,
			Message:      "GitHub integration not configured. Set GH_TOKEN or GITHUB_TOKEN to enable.",
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

	ctx, cancel := context.WithTimeout(context.Background(), timeouts.GithubAPI)
	defer cancel()

	prs, err := s.githubClient.ListPullRequests(ctx, owner, repo, github.ListPullRequestsOptions{
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
		author := ""
		if pr.User != nil {
			author = pr.User.Login
		}
		headBranch := ""
		if pr.Head != nil {
			headBranch = pr.Head.Ref
		}
		baseBranch := ""
		if pr.Base != nil {
			baseBranch = pr.Base.Ref
		}
		summaries = append(summaries, PRSummary{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Author:       author,
			Draft:        pr.Draft,
			Merged:       pr.Merged,
			HeadBranch:   headBranch,
			BaseBranch:   baseBranch,
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

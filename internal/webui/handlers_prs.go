package webui

import (
	"context"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/github"
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

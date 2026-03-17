package webui

import (
	"context"
	"net/http"
	"time"

	"github.com/recinq/wave/internal/github"
)

// handlePRsPage handles GET /prs - serves the HTML pull requests page.
func (s *Server) handlePRsPage(w http.ResponseWriter, r *http.Request) {
	prData := s.getPRListData()
	data := struct {
		PRListResponse
		ActivePage string
	}{
		PRListResponse: prData,
		ActivePage:     "prs",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["templates/prs.html"].ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIPRs handles GET /api/prs - returns PR list as JSON.
func (s *Server) handleAPIPRs(w http.ResponseWriter, r *http.Request) {
	data := s.getPRListData()
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) getPRListData() PRListResponse {
	if s.githubClient == nil || s.repoSlug == "" {
		return PRListResponse{
			PullRequests: []PRSummary{},
			Message:      "GitHub integration not configured. Set GH_TOKEN or GITHUB_TOKEN to enable.",
		}
	}

	owner, repo := splitRepoSlug(s.repoSlug)
	if owner == "" {
		return PRListResponse{
			PullRequests: []PRSummary{},
			Message:      "Could not determine repository from git remote.",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	prs, err := s.githubClient.ListPullRequests(ctx, owner, repo, github.ListPullRequestsOptions{
		State:   "open",
		PerPage: 50,
	})
	if err != nil {
		return PRListResponse{
			PullRequests: []PRSummary{},
			Message:      "Failed to fetch pull requests: " + err.Error(),
		}
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
	}
}

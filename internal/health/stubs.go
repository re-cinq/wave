package health

import (
	"context"
	"fmt"
)

// NotImplementedError indicates that a ForgeAnalyzer method has not yet been
// implemented for a particular forge.
type NotImplementedError struct {
	Forge  string
	Method string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("TODO: %s.%s not yet implemented for %s forge", e.Forge, e.Method, e.Forge)
}

// GitLabAnalyzer is a stub ForgeAnalyzer for GitLab repositories.
type GitLabAnalyzer struct{}

func (a *GitLabAnalyzer) AnalyzeCommits(_ context.Context, _ string) (*CommitAnalysis, error) {
	return nil, &NotImplementedError{Forge: "GitLab", Method: "AnalyzeCommits"}
}

func (a *GitLabAnalyzer) AnalyzePRs(_ context.Context, _ string) (*PRSummary, error) {
	return nil, &NotImplementedError{Forge: "GitLab", Method: "AnalyzePRs"}
}

func (a *GitLabAnalyzer) AnalyzeIssues(_ context.Context, _ string) (*IssueSummary, error) {
	return nil, &NotImplementedError{Forge: "GitLab", Method: "AnalyzeIssues"}
}

func (a *GitLabAnalyzer) AnalyzeCIStatus(_ context.Context, _ string) (*CIStatus, error) {
	return nil, &NotImplementedError{Forge: "GitLab", Method: "AnalyzeCIStatus"}
}

// BitbucketAnalyzer is a stub ForgeAnalyzer for Bitbucket repositories.
type BitbucketAnalyzer struct{}

func (a *BitbucketAnalyzer) AnalyzeCommits(_ context.Context, _ string) (*CommitAnalysis, error) {
	return nil, &NotImplementedError{Forge: "Bitbucket", Method: "AnalyzeCommits"}
}

func (a *BitbucketAnalyzer) AnalyzePRs(_ context.Context, _ string) (*PRSummary, error) {
	return nil, &NotImplementedError{Forge: "Bitbucket", Method: "AnalyzePRs"}
}

func (a *BitbucketAnalyzer) AnalyzeIssues(_ context.Context, _ string) (*IssueSummary, error) {
	return nil, &NotImplementedError{Forge: "Bitbucket", Method: "AnalyzeIssues"}
}

func (a *BitbucketAnalyzer) AnalyzeCIStatus(_ context.Context, _ string) (*CIStatus, error) {
	return nil, &NotImplementedError{Forge: "Bitbucket", Method: "AnalyzeCIStatus"}
}

// GiteaAnalyzer is a stub ForgeAnalyzer for Gitea repositories.
type GiteaAnalyzer struct{}

func (a *GiteaAnalyzer) AnalyzeCommits(_ context.Context, _ string) (*CommitAnalysis, error) {
	return nil, &NotImplementedError{Forge: "Gitea", Method: "AnalyzeCommits"}
}

func (a *GiteaAnalyzer) AnalyzePRs(_ context.Context, _ string) (*PRSummary, error) {
	return nil, &NotImplementedError{Forge: "Gitea", Method: "AnalyzePRs"}
}

func (a *GiteaAnalyzer) AnalyzeIssues(_ context.Context, _ string) (*IssueSummary, error) {
	return nil, &NotImplementedError{Forge: "Gitea", Method: "AnalyzeIssues"}
}

func (a *GiteaAnalyzer) AnalyzeCIStatus(_ context.Context, _ string) (*CIStatus, error) {
	return nil, &NotImplementedError{Forge: "Gitea", Method: "AnalyzeCIStatus"}
}

package health

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Analyze is the top-level orchestrator for health analysis. It detects the
// forge type from the repository's git remotes, creates the appropriate
// ForgeAnalyzer, runs all analysis methods, and assembles a HealthReport.
func Analyze(ctx context.Context, repoPath string, opts AnalyzeOptions) (*HealthReport, error) {
	forgeType, repoID, err := DetectForge(repoPath)
	if err != nil {
		return nil, fmt.Errorf("health analyze: %w", err)
	}

	// Fill in default options for zero-valued fields.
	if opts.CommitWindowDays == 0 {
		opts.CommitWindowDays = DefaultCommitWindowDays
	}
	if opts.StalenessThresholdDays == 0 {
		opts.StalenessThresholdDays = DefaultStalenessThresholdDays
	}
	if opts.MaxItems == 0 {
		opts.MaxItems = DefaultMaxItems
	}

	analyzer, err := newAnalyzer(forgeType, repoPath, opts)
	if err != nil {
		return nil, fmt.Errorf("health analyze: %w", err)
	}

	report := &HealthReport{
		ForgeType:  forgeType,
		Repository: repoID,
		AnalyzedAt: time.Now().UTC(),
	}

	// Analyze commits: soft-fail on NotImplementedError, propagate other errors.
	commits, err := analyzer.AnalyzeCommits(ctx, repoID)
	if err != nil {
		var notImpl *NotImplementedError
		if errors.As(err, &notImpl) {
			report.Commits = &CommitAnalysis{}
		} else {
			return nil, fmt.Errorf("health analyze: %w", err)
		}
	} else {
		report.Commits = commits
	}

	// Analyze PRs: soft-fail on NotImplementedError, propagate other errors.
	prs, err := analyzer.AnalyzePRs(ctx, repoID)
	if err != nil {
		var notImpl *NotImplementedError
		if errors.As(err, &notImpl) {
			report.PullRequests = &PRSummary{}
		} else {
			return nil, fmt.Errorf("health analyze: %w", err)
		}
	} else {
		report.PullRequests = prs
	}

	// Analyze issues: soft-fail on NotImplementedError, propagate other errors.
	issues, err := analyzer.AnalyzeIssues(ctx, repoID)
	if err != nil {
		var notImpl *NotImplementedError
		if errors.As(err, &notImpl) {
			report.Issues = &IssueSummary{}
		} else {
			return nil, fmt.Errorf("health analyze: %w", err)
		}
	} else {
		report.Issues = issues
	}

	// Analyze CI status: any error (including NotImplementedError) results in nil.
	ciStatus, err := analyzer.AnalyzeCIStatus(ctx, repoID)
	if err != nil {
		report.CIStatus = nil
	} else {
		report.CIStatus = ciStatus
	}

	return report, nil
}

// newAnalyzer creates the appropriate ForgeAnalyzer based on the detected forge type.
func newAnalyzer(forgeType ForgeType, repoPath string, opts AnalyzeOptions) (ForgeAnalyzer, error) {
	switch forgeType {
	case GitHub:
		return NewGitHubAnalyzer(repoPath, opts), nil
	case GitLab:
		return &GitLabAnalyzer{}, nil
	case Bitbucket:
		return &BitbucketAnalyzer{}, nil
	case Gitea:
		return &GiteaAnalyzer{}, nil
	default:
		return nil, fmt.Errorf("unsupported forge type: %s", forgeType)
	}
}

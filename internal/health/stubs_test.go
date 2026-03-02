package health

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotImplementedError(t *testing.T) {
	err := &NotImplementedError{Forge: "TestForge", Method: "DoSomething"}
	msg := err.Error()

	assert.Contains(t, msg, "TODO", "error message should contain TODO marker")
	assert.Contains(t, msg, "TestForge", "error message should contain the forge name")
	assert.Contains(t, msg, "DoSomething", "error message should contain the method name")
	assert.Equal(t, "TODO: TestForge.DoSomething not yet implemented for TestForge forge", msg)
}

func TestStubAnalyzers(t *testing.T) {
	type analyzerCase struct {
		name      string
		analyzer  ForgeAnalyzer
		forgeName string
	}

	analyzers := []analyzerCase{
		{name: "GitLabAnalyzer", analyzer: &GitLabAnalyzer{}, forgeName: "GitLab"},
		{name: "BitbucketAnalyzer", analyzer: &BitbucketAnalyzer{}, forgeName: "Bitbucket"},
		{name: "GiteaAnalyzer", analyzer: &GiteaAnalyzer{}, forgeName: "Gitea"},
	}

	for _, ac := range analyzers {
		t.Run(ac.name, func(t *testing.T) {
			ctx := context.Background()
			repo := "owner/repo"

			t.Run("AnalyzeCommits", func(t *testing.T) {
				result, err := ac.analyzer.AnalyzeCommits(ctx, repo)
				assert.Nil(t, result, "AnalyzeCommits should return nil data")
				require.Error(t, err, "AnalyzeCommits should return an error")

				var notImpl *NotImplementedError
				require.True(t, errors.As(err, &notImpl), "error should be *NotImplementedError")
				assert.Contains(t, err.Error(), "TODO", "error message should contain TODO")
				assert.Contains(t, err.Error(), ac.forgeName, "error message should contain forge name")
			})

			t.Run("AnalyzePRs", func(t *testing.T) {
				result, err := ac.analyzer.AnalyzePRs(ctx, repo)
				assert.Nil(t, result, "AnalyzePRs should return nil data")
				require.Error(t, err, "AnalyzePRs should return an error")

				var notImpl *NotImplementedError
				require.True(t, errors.As(err, &notImpl), "error should be *NotImplementedError")
				assert.Contains(t, err.Error(), "TODO", "error message should contain TODO")
				assert.Contains(t, err.Error(), ac.forgeName, "error message should contain forge name")
			})

			t.Run("AnalyzeIssues", func(t *testing.T) {
				result, err := ac.analyzer.AnalyzeIssues(ctx, repo)
				assert.Nil(t, result, "AnalyzeIssues should return nil data")
				require.Error(t, err, "AnalyzeIssues should return an error")

				var notImpl *NotImplementedError
				require.True(t, errors.As(err, &notImpl), "error should be *NotImplementedError")
				assert.Contains(t, err.Error(), "TODO", "error message should contain TODO")
				assert.Contains(t, err.Error(), ac.forgeName, "error message should contain forge name")
			})

			t.Run("AnalyzeCIStatus", func(t *testing.T) {
				result, err := ac.analyzer.AnalyzeCIStatus(ctx, repo)
				assert.Nil(t, result, "AnalyzeCIStatus should return nil data")
				require.Error(t, err, "AnalyzeCIStatus should return an error")

				var notImpl *NotImplementedError
				require.True(t, errors.As(err, &notImpl), "error should be *NotImplementedError")
				assert.Contains(t, err.Error(), "TODO", "error message should contain TODO")
				assert.Contains(t, err.Error(), ac.forgeName, "error message should contain forge name")
			})
		})
	}
}

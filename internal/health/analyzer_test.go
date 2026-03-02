package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyzer(t *testing.T) {
	opts := AnalyzeOptions{
		CommitWindowDays:       30,
		StalenessThresholdDays: 14,
		MaxItems:               100,
	}

	tests := []struct {
		name      string
		forge     ForgeType
		wantType  interface{} // expected concrete type (nil means error expected)
		wantError bool
	}{
		{
			name:     "GitHub returns GitHubAnalyzer",
			forge:    GitHub,
			wantType: &GitHubAnalyzer{},
		},
		{
			name:     "GitLab returns GitLabAnalyzer",
			forge:    GitLab,
			wantType: &GitLabAnalyzer{},
		},
		{
			name:     "Bitbucket returns BitbucketAnalyzer",
			forge:    Bitbucket,
			wantType: &BitbucketAnalyzer{},
		},
		{
			name:     "Gitea returns GiteaAnalyzer",
			forge:    Gitea,
			wantType: &GiteaAnalyzer{},
		},
		{
			name:      "Unknown returns error",
			forge:     Unknown,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := newAnalyzer(tt.forge, "/tmp/fake-repo", opts)
			if tt.wantError {
				require.Error(t, err, "newAnalyzer should return an error for forge %s", tt.forge)
				assert.Nil(t, analyzer, "analyzer should be nil on error")
				return
			}

			require.NoError(t, err, "newAnalyzer should not return an error for forge %s", tt.forge)
			require.NotNil(t, analyzer, "analyzer should not be nil")

			// Verify the concrete type matches.
			switch tt.forge {
			case GitHub:
				assert.IsType(t, &GitHubAnalyzer{}, analyzer, "expected *GitHubAnalyzer")
			case GitLab:
				assert.IsType(t, &GitLabAnalyzer{}, analyzer, "expected *GitLabAnalyzer")
			case Bitbucket:
				assert.IsType(t, &BitbucketAnalyzer{}, analyzer, "expected *BitbucketAnalyzer")
			case Gitea:
				assert.IsType(t, &GiteaAnalyzer{}, analyzer, "expected *GiteaAnalyzer")
			}
		})
	}
}

func TestAnalyzeNoGitRepo(t *testing.T) {
	// A temp directory with no git initialization should cause DetectForge
	// to fail, which Analyze should propagate as an error.
	tmpDir := t.TempDir()

	ctx := context.Background()
	report, err := Analyze(ctx, tmpDir, AnalyzeOptions{})

	require.Error(t, err, "Analyze should return an error for a directory without a git repo")
	assert.Nil(t, report, "report should be nil when Analyze fails")
	assert.Contains(t, err.Error(), "health analyze", "error should be wrapped with health analyze prefix")
}

func TestAnalyzeUnknownForge(t *testing.T) {
	// Directly test that newAnalyzer with Unknown forge returns an error.
	opts := AnalyzeOptions{
		CommitWindowDays:       30,
		StalenessThresholdDays: 14,
		MaxItems:               100,
	}

	analyzer, err := newAnalyzer(Unknown, "/tmp/nonexistent", opts)
	require.Error(t, err, "newAnalyzer should error on Unknown forge type")
	assert.Nil(t, analyzer, "analyzer should be nil for Unknown forge type")
	assert.Contains(t, err.Error(), "unsupported forge type", "error should mention unsupported forge type")
}

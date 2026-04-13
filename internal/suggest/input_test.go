package suggest

import (
	"testing"
)

func TestClassifyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  InputType
	}{
		// GitHub issue URLs
		{
			name:  "github issue URL",
			input: "https://github.com/owner/repo/issues/123",
			want:  InputTypeIssueURL,
		},
		{
			name:  "github issue URL with trailing slash",
			input: "https://github.com/owner/repo/issues/123/",
			want:  InputTypeIssueURL,
		},
		{
			name:  "github issue URL http",
			input: "http://github.com/owner/repo/issues/456",
			want:  InputTypeIssueURL,
		},
		{
			name:  "github issue URL with fragment",
			input: "https://github.com/owner/repo/issues/789#issuecomment-1234",
			want:  InputTypeIssueURL,
		},

		// GitLab issue URLs
		{
			name:  "gitlab issue URL",
			input: "https://gitlab.com/group/project/issues/42",
			want:  InputTypeIssueURL,
		},
		{
			name:  "gitlab issue URL nested group",
			input: "https://gitlab.com/group/subgroup/project/issues/42",
			want:  InputTypeIssueURL,
		},

		// Gitea / Codeberg issue URLs
		{
			name:  "codeberg issue URL",
			input: "https://codeberg.org/owner/repo/issues/10",
			want:  InputTypeIssueURL,
		},
		{
			name:  "gitea.com issue URL",
			input: "https://gitea.com/owner/repo/issues/5",
			want:  InputTypeIssueURL,
		},

		// Self-hosted forge issue URLs
		{
			name:  "self-hosted issue URL",
			input: "https://git.example.com/owner/repo/issues/99",
			want:  InputTypeIssueURL,
		},

		// GitHub PR URLs
		{
			name:  "github PR URL",
			input: "https://github.com/owner/repo/pull/456",
			want:  InputTypePRURL,
		},
		{
			name:  "github PR URL with fragment",
			input: "https://github.com/owner/repo/pull/456#discussion_r123",
			want:  InputTypePRURL,
		},
		{
			name:  "github pulls URL",
			input: "https://github.com/owner/repo/pulls/789",
			want:  InputTypePRURL,
		},

		// GitLab merge request URLs
		{
			name:  "gitlab MR URL",
			input: "https://gitlab.com/group/project/merge_requests/42",
			want:  InputTypePRURL,
		},
		{
			name:  "gitlab MR URL with dash prefix",
			input: "https://gitlab.com/group/project/-/merge_requests/42",
			want:  InputTypePRURL,
		},

		// Self-hosted PR URLs
		{
			name:  "self-hosted PR URL",
			input: "https://git.mycompany.io/team/service/pull/33",
			want:  InputTypePRURL,
		},
		{
			name:  "self-hosted MR URL",
			input: "https://gitlab.internal.com/org/app/merge_requests/7",
			want:  InputTypePRURL,
		},

		// Repo ref patterns
		{
			name:  "repo ref with hash",
			input: "owner/repo #123",
			want:  InputTypeRepoRef,
		},
		{
			name:  "repo ref without hash",
			input: "owner/repo 123",
			want:  InputTypeRepoRef,
		},
		{
			name:  "repo ref with dots and dashes",
			input: "my-org/my.repo #456",
			want:  InputTypeRepoRef,
		},

		// Free text
		{
			name:  "simple description",
			input: "add user authentication",
			want:  InputTypeFreeText,
		},
		{
			name:  "multi-word description",
			input: "Fix the login bug that causes 500 errors on POST /api/auth",
			want:  InputTypeFreeText,
		},
		{
			name:  "empty input",
			input: "",
			want:  InputTypeFreeText,
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  InputTypeFreeText,
		},
		{
			name:  "command-like input",
			input: "go test ./...",
			want:  InputTypeFreeText,
		},

		// Edge cases
		{
			name:  "URL without issue path",
			input: "https://github.com/owner/repo",
			want:  InputTypeFreeText,
		},
		{
			name:  "URL with issues in query",
			input: "https://example.com/?q=issues/123",
			want:  InputTypeFreeText,
		},
		{
			name:  "partial repo ref missing number",
			input: "owner/repo",
			want:  InputTypeFreeText,
		},
		{
			name:  "PR URL takes precedence over issue URL when both patterns present",
			input: "https://github.com/owner/repo/pull/123",
			want:  InputTypePRURL,
		},
		{
			name:  "bitbucket issue URL",
			input: "https://bitbucket.org/team/repo/issues/88",
			want:  InputTypeIssueURL,
		},
		{
			name:  "gitea.io issue URL",
			input: "https://gitea.io/owner/repo/issues/3",
			want:  InputTypeIssueURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyInput(tt.input)
			if got != tt.want {
				t.Errorf("ClassifyInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuggestPipelineForInput(t *testing.T) {
	tests := []struct {
		name      string
		inputType InputType
		wantFirst string
		wantLen   int
	}{
		{
			name:      "issue URL suggests impl-issue first",
			inputType: InputTypeIssueURL,
			wantFirst: "impl-issue",
			wantLen:   3,
		},
		{
			name:      "PR URL suggests ops-pr-review",
			inputType: InputTypePRURL,
			wantFirst: "ops-pr-review",
			wantLen:   1,
		},
		{
			name:      "repo ref suggests impl-issue",
			inputType: InputTypeRepoRef,
			wantFirst: "impl-issue",
			wantLen:   1,
		},
		{
			name:      "free text suggests impl-feature first",
			inputType: InputTypeFreeText,
			wantFirst: "impl-feature",
			wantLen:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestPipelineForInput(tt.inputType)
			if len(got) != tt.wantLen {
				t.Errorf("SuggestPipelineForInput(%q) returned %d items, want %d", tt.inputType, len(got), tt.wantLen)
			}
			if len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("SuggestPipelineForInput(%q)[0] = %q, want %q", tt.inputType, got[0], tt.wantFirst)
			}
		})
	}
}

func TestCheckInputPipelineMismatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pipeline string
		wantNil  bool
	}{
		{
			name:     "issue URL with impl-issue is fine",
			input:    "https://github.com/owner/repo/issues/1",
			pipeline: "impl-issue",
			wantNil:  true,
		},
		{
			name:     "issue URL with plan-research is fine",
			input:    "https://github.com/owner/repo/issues/1",
			pipeline: "plan-research",
			wantNil:  true,
		},
		{
			name:     "issue URL with ops-pr-review returns nil (issue URLs never warn)",
			input:    "https://github.com/owner/repo/issues/1",
			pipeline: "ops-pr-review",
			wantNil:  true,
		},
		{
			name:     "PR URL with ops-pr-review is fine",
			input:    "https://github.com/owner/repo/pull/42",
			pipeline: "ops-pr-review",
			wantNil:  true,
		},
		{
			name:     "PR URL with impl-issue is mismatch",
			input:    "https://github.com/owner/repo/pull/42",
			pipeline: "impl-issue",
			wantNil:  false,
		},
		{
			name:     "free text with impl-feature is fine",
			input:    "add auth support",
			pipeline: "impl-feature",
			wantNil:  true,
		},
		{
			name:     "free text with impl-issue returns nil (free text never warns)",
			input:    "add auth support",
			pipeline: "impl-issue",
			wantNil:  true,
		},
		{
			name:     "empty input returns nil (no warning)",
			input:    "",
			pipeline: "anything",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckInputPipelineMismatch(tt.input, tt.pipeline)
			if tt.wantNil && got != nil {
				t.Errorf("CheckInputPipelineMismatch(%q, %q) = %+v, want nil", tt.input, tt.pipeline, got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("CheckInputPipelineMismatch(%q, %q) = nil, want mismatch", tt.input, tt.pipeline)
			}
		})
	}
}

func TestCheckInputPipelineMismatch_ReasonContainsSuggestions(t *testing.T) {
	mismatch := CheckInputPipelineMismatch("https://github.com/owner/repo/pull/42", "impl-issue")
	if mismatch == nil {
		t.Fatal("expected mismatch")
	}
	if mismatch.SuggestedReason == "" {
		t.Error("expected non-empty suggested reason")
	}
	if mismatch.InputType != InputTypePRURL {
		t.Errorf("expected input type %q, got %q", InputTypePRURL, mismatch.InputType)
	}
}

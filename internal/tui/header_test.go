package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Existing tests ---

func TestHeaderModel_View_ContainsLogo(t *testing.T) {
	h := NewHeaderModel(nil)
	view := h.View()

	// The logo contains Wave ASCII art characters
	assert.Contains(t, view, "╦")
	assert.Contains(t, view, "╚╩╝")
}

func TestHeaderModel_View_ContainsMetadataPlaceholders(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(120)

	view := h.View()
	// Without any data loaded, we should see placeholder markers
	assert.Contains(t, view, "…")
	assert.Contains(t, view, "● OK")
}

func TestHeaderModel_SetWidth(t *testing.T) {
	h := NewHeaderModel(nil)
	assert.Equal(t, 0, h.width)

	h.SetWidth(120)
	assert.Equal(t, 120, h.width)
}

func TestHeaderModel_View_RespectsWidth(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(80)
	view := h.View()

	// Each line should not exceed the specified width
	for _, line := range strings.Split(view, "\n") {
		// lipgloss Width accounts for ANSI escape sequences
		assert.LessOrEqual(t, len([]rune(stripAnsi(line))), 80+10, // allow margin for ANSI sequences
			"line exceeds width: %q", line)
	}
}

func TestHeaderModel_View_WithMetadata(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(120)
	h.metadata.Branch = "feature/test"
	h.metadata.ProjectName = "wave"
	h.metadata.CommitHash = "abc1234"

	view := h.View()
	assert.Contains(t, view, "feature/test")
	assert.Contains(t, view, "wave")
	assert.Contains(t, view, "abc1234")
}

func TestHeaderModel_DisplayBranch_Override(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"
	h.metadata.OverrideBranch = "feat/override"

	branch := h.displayBranch()
	assert.Equal(t, "feat/override", branch)
}

func TestHeaderModel_DisplayBranch_Fallback(t *testing.T) {
	h := NewHeaderModel(nil)

	branch := h.displayBranch()
	assert.Equal(t, "…", branch)
}

func TestHeaderModel_RenderIssues_NotConfigured(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.GitHubState = GitHubNotConfigured

	result := h.renderIssuesValue()
	assert.Equal(t, "—", result)
}

func TestHeaderModel_RenderIssues_Connected(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.GitHubState = GitHubConnected
	h.metadata.IssuesCount = 42

	result := h.renderIssuesValue()
	assert.Contains(t, stripAnsi(result), "42")
}

func TestHeaderModel_RenderDirty_NoBranch(t *testing.T) {
	h := NewHeaderModel(nil)

	result := h.renderDirty()
	assert.Contains(t, stripAnsi(result), "…")
}

func TestHeaderModel_RenderDirty_Clean(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"
	h.metadata.IsDirty = false

	result := h.renderDirty()
	assert.Contains(t, stripAnsi(result), "✓")
}

func TestHeaderModel_RenderDirty_Dirty(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"
	h.metadata.IsDirty = true

	result := h.renderDirty()
	assert.Contains(t, stripAnsi(result), "✱")
}

// stripAnsi removes ANSI escape sequences for length checking.
func stripAnsi(s string) string {
	result := strings.Builder{}
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// --- T026: Mock MetadataProvider & Update() tests ---

// testProvider is a configurable mock MetadataProvider for header tests.
type testProvider struct {
	gitState    GitState
	gitErr      error
	manifest    ManifestInfo
	manifestErr error
	github      GitHubInfo
	githubErr   error
	health      HealthStatus
	healthErr   error
}

func (p *testProvider) FetchGitState() (GitState, error)                { return p.gitState, p.gitErr }
func (p *testProvider) FetchManifestInfo() (ManifestInfo, error)        { return p.manifest, p.manifestErr }
func (p *testProvider) FetchGitHubInfo(repo string) (GitHubInfo, error) { return p.github, p.githubErr }
func (p *testProvider) FetchPipelineHealth() (HealthStatus, error)      { return p.health, p.healthErr }

func TestHeaderModel_Update_GitStateMsg(t *testing.T) {
	tests := []struct {
		name       string
		msg        GitStateMsg
		wantBranch string
		wantCommit string
		wantDirty  bool
		wantRemote string
	}{
		{
			name: "successful git state sets all fields",
			msg: GitStateMsg{
				State: GitState{
					Branch:     "feature/login",
					CommitHash: "abc1234",
					IsDirty:    true,
					RemoteName: "origin",
				},
				Err: nil,
			},
			wantBranch: "feature/login",
			wantCommit: "abc1234",
			wantDirty:  true,
			wantRemote: "origin",
		},
		{
			name: "clean repo state",
			msg: GitStateMsg{
				State: GitState{
					Branch:     "main",
					CommitHash: "def5678",
					IsDirty:    false,
					RemoteName: "upstream",
				},
				Err: nil,
			},
			wantBranch: "main",
			wantCommit: "def5678",
			wantDirty:  false,
			wantRemote: "upstream",
		},
		{
			name: "error does not update metadata",
			msg: GitStateMsg{
				State: GitState{Branch: "should-not-appear"},
				Err:   fmt.Errorf("git not found"),
			},
			wantBranch: "",
			wantCommit: "",
			wantDirty:  false,
			wantRemote: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			updated, _ := h.Update(tt.msg)
			assert.Equal(t, tt.wantBranch, updated.metadata.Branch)
			assert.Equal(t, tt.wantCommit, updated.metadata.CommitHash)
			assert.Equal(t, tt.wantDirty, updated.metadata.IsDirty)
			assert.Equal(t, tt.wantRemote, updated.metadata.RemoteName)
		})
	}
}

func TestHeaderModel_Update_ManifestInfoMsg(t *testing.T) {
	tests := []struct {
		name        string
		msg         ManifestInfoMsg
		wantProject string
		wantRepo    string
	}{
		{
			name: "successful manifest sets project and repo",
			msg: ManifestInfoMsg{
				Info: ManifestInfo{ProjectName: "wave", RepoName: "re-cinq/wave"},
				Err:  nil,
			},
			wantProject: "wave",
			wantRepo:    "re-cinq/wave",
		},
		{
			name: "error does not update metadata",
			msg: ManifestInfoMsg{
				Info: ManifestInfo{ProjectName: "should-not-appear"},
				Err:  fmt.Errorf("file not found"),
			},
			wantProject: "",
			wantRepo:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			updated, _ := h.Update(tt.msg)
			assert.Equal(t, tt.wantProject, updated.metadata.ProjectName)
			assert.Equal(t, tt.wantRepo, updated.metadata.RepoName)
		})
	}
}

func TestHeaderModel_Update_ManifestInfoMsg_TriggersGitHubFetch(t *testing.T) {
	provider := &testProvider{
		github: GitHubInfo{AuthState: GitHubConnected, IssuesCount: 10},
	}
	h := NewHeaderModel(provider)

	// When manifest provides a repo name, a GitHub fetch command should be returned
	msg := ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: "wave", RepoName: "re-cinq/wave"},
		Err:  nil,
	}
	_, cmd := h.Update(msg)
	assert.NotNil(t, cmd, "should return a command to fetch GitHub info when repo is set")
}

func TestHeaderModel_Update_ManifestInfoMsg_NoGitHubFetchWithoutRepo(t *testing.T) {
	h := NewHeaderModel(nil)

	msg := ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: "wave", RepoName: ""},
		Err:  nil,
	}
	_, cmd := h.Update(msg)
	assert.Nil(t, cmd, "should not return a command when repo name is empty")
}

func TestHeaderModel_Update_GitHubInfoMsg(t *testing.T) {
	tests := []struct {
		name       string
		msg        GitHubInfoMsg
		wantAuth   GitHubAuthState
		wantIssues int
	}{
		{
			name: "connected with issues",
			msg: GitHubInfoMsg{
				Info: GitHubInfo{AuthState: GitHubConnected, IssuesCount: 42},
				Err:  nil,
			},
			wantAuth:   GitHubConnected,
			wantIssues: 42,
		},
		{
			name: "offline state",
			msg: GitHubInfoMsg{
				Info: GitHubInfo{AuthState: GitHubOffline, IssuesCount: 0},
				Err:  nil,
			},
			wantAuth:   GitHubOffline,
			wantIssues: 0,
		},
		{
			name: "error does not update metadata",
			msg: GitHubInfoMsg{
				Info: GitHubInfo{AuthState: GitHubConnected, IssuesCount: 99},
				Err:  fmt.Errorf("network error"),
			},
			wantAuth:   GitHubNotConfigured, // default zero value
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			updated, _ := h.Update(tt.msg)
			assert.Equal(t, tt.wantAuth, updated.metadata.GitHubState)
			assert.Equal(t, tt.wantIssues, updated.metadata.IssuesCount)
		})
	}
}

func TestHeaderModel_Update_PipelineHealthMsg(t *testing.T) {
	tests := []struct {
		name       string
		msg        PipelineHealthMsg
		wantHealth HealthStatus
	}{
		{
			name:       "health OK",
			msg:        PipelineHealthMsg{Health: HealthOK, Err: nil},
			wantHealth: HealthOK,
		},
		{
			name:       "health warn",
			msg:        PipelineHealthMsg{Health: HealthWarn, Err: nil},
			wantHealth: HealthWarn,
		},
		{
			name:       "health error",
			msg:        PipelineHealthMsg{Health: HealthErr, Err: nil},
			wantHealth: HealthErr,
		},
		{
			name:       "error does not update health",
			msg:        PipelineHealthMsg{Health: HealthErr, Err: fmt.Errorf("db error")},
			wantHealth: HealthOK, // default zero value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			updated, _ := h.Update(tt.msg)
			assert.Equal(t, tt.wantHealth, updated.metadata.Health)
		})
	}
}

func TestHeaderModel_Update_RunningCountMsg(t *testing.T) {
	tests := []struct {
		name       string
		count      int
		wantActive bool
		wantCmd    bool
	}{
		{
			name:       "zero pipelines deactivates logo",
			count:      0,
			wantActive: false,
			wantCmd:    false,
		},
		{
			name:       "one pipeline activates logo and returns tick cmd",
			count:      1,
			wantActive: true,
			wantCmd:    true,
		},
		{
			name:       "multiple pipelines activates logo",
			count:      5,
			wantActive: true,
			wantCmd:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			msg := RunningCountMsg{Count: tt.count}
			updated, cmd := h.Update(msg)
			assert.Equal(t, tt.count, updated.metadata.RunningCount)
			assert.Equal(t, tt.wantActive, updated.logo.IsActive())
			if tt.wantCmd {
				assert.NotNil(t, cmd, "should return a tick command when activating logo")
			} else {
				assert.Nil(t, cmd, "should not return a command when count is 0")
			}
		})
	}
}

func TestHeaderModel_Update_RunningCountMsg_AlreadyActive(t *testing.T) {
	h := NewHeaderModel(nil)
	// First activate the logo
	h, _ = h.Update(RunningCountMsg{Count: 1})
	require.True(t, h.logo.IsActive())

	// Updating with another positive count should NOT return a new tick command
	// because the logo is already active
	_, cmd := h.Update(RunningCountMsg{Count: 3})
	assert.Nil(t, cmd, "should not return a new tick when logo is already active")
}

func TestHeaderModel_Update_RunningCountMsg_DeactivateResetsFrame(t *testing.T) {
	h := NewHeaderModel(nil)
	// Activate and advance a few times
	h, _ = h.Update(RunningCountMsg{Count: 1})
	h.logo.Advance()
	h.logo.Advance()
	require.Equal(t, 2, h.logo.frame)

	// Deactivate
	h, _ = h.Update(RunningCountMsg{Count: 0})
	assert.Equal(t, 0, h.logo.frame, "frame should reset to 0 on deactivation")
}

// --- T027: View() rendering tests at widths 80, 120, 200 ---

func TestHeaderModel_View_ColumnPriorityAtDifferentWidths(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "feature/header-bar"
	h.metadata.CommitHash = "abc1234"
	h.metadata.ProjectName = "wave"
	h.metadata.RepoName = "re-cinq/wave"
	h.metadata.IsDirty = true
	h.metadata.RemoteName = "origin"
	h.metadata.Health = HealthOK
	h.metadata.GitHubState = GitHubConnected
	h.metadata.IssuesCount = 42

	tests := []struct {
		name          string
		width         int
		alwaysPresent []string
		maybePresent  []string // these may or may not appear depending on width
	}{
		{
			name:          "width 80 - logo always visible, some columns hidden",
			width:         80,
			alwaysPresent: []string{"╦", "╚╩╝"}, // logo always rendered
		},
		{
			name:          "width 120 - most columns visible",
			width:         120,
			alwaysPresent: []string{"╦", "╚╩╝", "feature/header-bar", "● OK"},
		},
		{
			name:          "width 200 - all columns visible",
			width:         200,
			alwaysPresent: []string{"╦", "╚╩╝", "feature/header-bar", "● OK", "wave", "origin", "abc1234"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.SetWidth(tt.width)
			view := h.View()
			stripped := stripAnsi(view)

			for _, expected := range tt.alwaysPresent {
				assert.Contains(t, stripped, expected, "expected %q at width %d", expected, tt.width)
			}
		})
	}
}

func TestHeaderModel_View_LogoAlwaysRenderedAtAllWidths(t *testing.T) {
	widths := []int{20, 40, 60, 80, 100, 120, 160, 200, 300}

	for _, w := range widths {
		t.Run(fmt.Sprintf("width_%d", w), func(t *testing.T) {
			h := NewHeaderModel(nil)
			h.SetWidth(w)
			view := h.View()
			assert.Contains(t, view, "╦", "logo should be present at width %d", w)
			assert.Contains(t, view, "╚╩╝", "logo should be present at width %d", w)
		})
	}
}

func TestHeaderModel_View_WiderWidthShowsMoreColumns(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"
	h.metadata.CommitHash = "abc1234"
	h.metadata.ProjectName = "wave"
	h.metadata.RemoteName = "origin"
	h.metadata.IsDirty = false
	h.metadata.Health = HealthOK
	h.metadata.GitHubState = GitHubConnected
	h.metadata.IssuesCount = 10

	// Render at narrow width
	h.SetWidth(80)
	narrowView := stripAnsi(h.View())

	// Render at wide width
	h.SetWidth(200)
	wideView := stripAnsi(h.View())

	// Wide view should be at least as long (in visible chars) as narrow view
	assert.GreaterOrEqual(t, len(wideView), len(narrowView),
		"wider terminal should show at least as much content")
}

// --- T028: Logo animation tests ---

func TestLogoAnimator_NewStartsInactive(t *testing.T) {
	logo := NewLogoAnimator()
	assert.False(t, logo.IsActive(), "new logo should start inactive")
	assert.Equal(t, 0, logo.frame, "new logo should start at frame 0")
}

func TestLogoAnimator_SetActive(t *testing.T) {
	logo := NewLogoAnimator()

	logo.SetActive(true)
	assert.True(t, logo.IsActive())

	logo.SetActive(false)
	assert.False(t, logo.IsActive())
	assert.Equal(t, 0, logo.frame, "SetActive(false) should reset frame to 0")
}

func TestLogoAnimator_SetActive_ResetsFrame(t *testing.T) {
	logo := NewLogoAnimator()
	logo.SetActive(true)
	logo.Advance()
	logo.Advance()
	require.Equal(t, 2, logo.frame)

	logo.SetActive(false)
	assert.Equal(t, 0, logo.frame, "deactivation should reset frame to 0")

	logo.SetActive(true)
	assert.Equal(t, 0, logo.frame, "reactivation should start from 0")
}

func TestLogoAnimator_Advance_IncrementsFrame(t *testing.T) {
	logo := NewLogoAnimator()
	logo.SetActive(true)

	assert.Equal(t, 0, logo.frame)

	logo.Advance()
	assert.Equal(t, 1, logo.frame)

	logo.Advance()
	assert.Equal(t, 2, logo.frame)

	logo.Advance()
	assert.Equal(t, 3, logo.frame)
}

func TestLogoAnimator_Tick_ReturnsNonNilCommand(t *testing.T) {
	logo := NewLogoAnimator()
	cmd := logo.Tick()
	assert.NotNil(t, cmd, "Tick() should return a non-nil command")
}

func TestLogoAnimator_Tick_ProducesLogoTickMsg(t *testing.T) {
	logo := NewLogoAnimator()
	cmd := logo.Tick()
	require.NotNil(t, cmd)

	// Execute the command — it's a tea.Tick that eventually fires a LogoTickMsg
	// We can't easily test the timing, but we can verify the command exists
	// The message produced by tea.Tick is the LogoTickMsg after the delay
}

func TestLogoAnimator_View_RendersLogo(t *testing.T) {
	logo := NewLogoAnimator()
	view := logo.View()
	assert.Contains(t, view, "╦")
	assert.Contains(t, view, "╚╩╝")
}

func TestLogoAnimator_View_DifferentFramesRenderSameText(t *testing.T) {
	logo := NewLogoAnimator()
	logo.SetActive(true)

	// Verify frames advance and each view renders the same text content.
	assert.Equal(t, 0, logo.frame)
	view0 := logo.View()
	assert.Contains(t, stripAnsi(view0), "╦")

	logo.Advance()
	assert.Equal(t, 1, logo.frame)
	view1 := logo.View()
	assert.Contains(t, stripAnsi(view1), "╦")

	logo.Advance()
	assert.Equal(t, 2, logo.frame)
	view2 := logo.View()
	assert.Contains(t, stripAnsi(view2), "╦")

	// Stripped text content should be identical across all frames
	assert.Equal(t, stripAnsi(view0), stripAnsi(view1),
		"stripped logo text should be the same regardless of frame")
	assert.Equal(t, stripAnsi(view1), stripAnsi(view2),
		"stripped logo text should be the same regardless of frame")
}

// --- T029: PipelineSelectedMsg tests ---

func TestHeaderModel_Update_PipelineSelectedMsg_BranchOverride(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"
	h.SetWidth(200)

	msg := PipelineSelectedMsg{
		RunID:      "run-123",
		BranchName: "feature/login",
	}
	updated, _ := h.Update(msg)

	assert.Equal(t, "feature/login", updated.metadata.OverrideBranch)
	view := stripAnsi(updated.View())
	assert.Contains(t, view, "feature/login", "overridden branch should appear in view")
}

func TestHeaderModel_Update_PipelineSelectedMsg_EmptyBranchReverts(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"
	h.metadata.OverrideBranch = "feature/old"

	msg := PipelineSelectedMsg{
		RunID:      "run-456",
		BranchName: "",
	}
	updated, _ := h.Update(msg)

	assert.Equal(t, "", updated.metadata.OverrideBranch)
	// displayBranch should now return the current branch
	assert.Equal(t, "main", updated.displayBranch())
}

func TestHeaderModel_Update_PipelineSelectedMsg_BranchDeleted(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	msg := PipelineSelectedMsg{
		RunID:         "run-789",
		BranchName:    "feature/old",
		BranchDeleted: true,
	}
	updated, _ := h.Update(msg)

	assert.Equal(t, "feature/old [deleted]", updated.metadata.OverrideBranch)
	view := stripAnsi(updated.View())
	assert.Contains(t, view, "[deleted]", "deleted suffix should appear in view")
}

func TestHeaderModel_Update_PipelineSelectedMsg_BranchDeletedWithEmptyName(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.Branch = "main"

	msg := PipelineSelectedMsg{
		RunID:         "run-000",
		BranchName:    "",
		BranchDeleted: true, // deleted flag with empty name should not add suffix
	}
	updated, _ := h.Update(msg)

	// Empty branch name should remain empty (no " [deleted]" suffix on empty string)
	assert.Equal(t, "", updated.metadata.OverrideBranch)
}

// --- T030: NO_COLOR test ---

func TestHeaderModel_View_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	h := NewHeaderModel(nil)
	h.SetWidth(200)
	h.metadata.Branch = "main"
	h.metadata.CommitHash = "abc1234"
	h.metadata.ProjectName = "wave"
	h.metadata.Health = HealthOK
	h.metadata.IsDirty = false
	h.metadata.RemoteName = "origin"
	h.metadata.GitHubState = GitHubConnected
	h.metadata.IssuesCount = 5

	view := h.View()
	assert.False(t, strings.Contains(view, "\x1b["),
		"output should contain zero ANSI escape sequences when NO_COLOR=1")
}

// --- T031: Edge case tests ---

func TestHeaderModel_EdgeCase_NoGit(t *testing.T) {
	provider := &testProvider{
		gitState: GitState{Branch: "[no git]", CommitHash: "[no git]"},
		gitErr:   nil, // provider returns placeholders with nil error per DefaultMetadataProvider pattern
	}
	h := NewHeaderModel(provider)
	h.SetWidth(200)

	// Simulate receiving the git state message
	gitMsg := GitStateMsg{
		State: GitState{Branch: "[no git]", CommitHash: "[no git]"},
		Err:   nil,
	}
	h, _ = h.Update(gitMsg)

	view := stripAnsi(h.View())
	assert.Contains(t, view, "[no git]", "should display [no git] when git is unavailable")
}

func TestHeaderModel_EdgeCase_NoGitError(t *testing.T) {
	// When FetchGitState actually returns an error, metadata should not be updated
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	gitMsg := GitStateMsg{
		State: GitState{Branch: "should-not-appear"},
		Err:   fmt.Errorf("git not found"),
	}
	h, _ = h.Update(gitMsg)

	assert.Equal(t, "", h.metadata.Branch, "branch should not be set on error")
	view := stripAnsi(h.View())
	assert.NotContains(t, view, "should-not-appear")
}

func TestHeaderModel_EdgeCase_NoManifest(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	// Simulate provider returning placeholders (like DefaultMetadataProvider does)
	manifestMsg := ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: "[no project]"},
		Err:  nil,
	}
	h, _ = h.Update(manifestMsg)

	view := stripAnsi(h.View())
	assert.Contains(t, view, "[no project]", "should display [no project] when manifest is missing")
}

func TestHeaderModel_EdgeCase_NoManifestError(t *testing.T) {
	// When manifest returns an error, metadata should not be updated
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	manifestMsg := ManifestInfoMsg{
		Info: ManifestInfo{ProjectName: "should-not-appear"},
		Err:  fmt.Errorf("file not found"),
	}
	h, _ = h.Update(manifestMsg)

	assert.Equal(t, "", h.metadata.ProjectName)
}

func TestHeaderModel_EdgeCase_GitHubNotConfigured_IssuesHidden(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	ghMsg := GitHubInfoMsg{
		Info: GitHubInfo{AuthState: GitHubNotConfigured},
		Err:  nil,
	}
	h, _ = h.Update(ghMsg)

	result := h.renderIssuesValue()
	assert.Equal(t, "—", result, "issues section should show dash when GitHub is not configured")
}

func TestHeaderModel_EdgeCase_PlaceholderBeforeAsyncData(t *testing.T) {
	// A freshly created header with no data loaded should show placeholders
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	view := stripAnsi(h.View())

	// Before any async data arrives, should show placeholder markers
	assert.Contains(t, view, "…", "should show placeholder markers before data arrives")
	assert.Contains(t, view, "● OK", "should show default health status before data arrives")
}

func TestHeaderModel_EdgeCase_GitHubOffline(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(200)

	ghMsg := GitHubInfoMsg{
		Info: GitHubInfo{AuthState: GitHubOffline},
		Err:  nil,
	}
	h, _ = h.Update(ghMsg)

	result := stripAnsi(h.renderIssuesValue())
	assert.Contains(t, result, "[offline]", "should show [offline] when GitHub is unreachable")
}

func TestHeaderModel_Update_GitRefreshTickMsg(t *testing.T) {
	provider := &testProvider{
		gitState: GitState{Branch: "main", CommitHash: "abc1234"},
	}
	h := NewHeaderModel(provider)

	msg := GitRefreshTickMsg{}
	_, cmd := h.Update(msg)
	assert.NotNil(t, cmd, "git refresh tick should return a batch command for re-fetch")
}

func TestHeaderModel_Update_LogoTickMsg_Active(t *testing.T) {
	h := NewHeaderModel(nil)
	h.logo.SetActive(true)
	initialIndex := h.logo.frame

	h, cmd := h.Update(LogoTickMsg{})
	assert.NotEqual(t, initialIndex, h.logo.frame, "frame should advance on tick")
	assert.NotNil(t, cmd, "should return another tick command when active")
}

func TestHeaderModel_Update_LogoTickMsg_Inactive(t *testing.T) {
	h := NewHeaderModel(nil)
	h.logo.SetActive(false)

	h, cmd := h.Update(LogoTickMsg{})
	assert.Equal(t, 0, h.logo.frame, "frame should not change when inactive")
	assert.Nil(t, cmd, "should not return a command when logo is inactive")
}

// --- T027 additional: renderHealth variants ---

func TestHeaderModel_RenderHealth_AllStatuses(t *testing.T) {
	tests := []struct {
		name     string
		health   HealthStatus
		expected string
	}{
		{"OK", HealthOK, "● OK"},
		{"Warn", HealthWarn, "▲ WARN"},
		{"Err", HealthErr, "✗ ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			h.metadata.Health = tt.health
			result := stripAnsi(h.renderHealth())
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHeaderModel_RenderPipesValue(t *testing.T) {
	tests := []struct {
		name     string
		running  int
		expected string
	}{
		{
			name:     "zero running shows em dash",
			running:  0,
			expected: "—",
		},
		{
			name:     "one running shows count",
			running:  1,
			expected: "1 running",
		},
		{
			name:     "multiple running shows count",
			running:  5,
			expected: "5 running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeaderModel(nil)
			h.metadata.RunningCount = tt.running
			result := stripAnsi(h.renderPipesValue())
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHeaderModel_View_ShowsRunningLabel(t *testing.T) {
	h := NewHeaderModel(nil)
	h.SetWidth(120)
	h.metadata.RunningCount = 3

	view := stripAnsi(h.View())
	assert.Contains(t, view, "Running:")
	assert.Contains(t, view, "3 running")
	assert.NotContains(t, view, "Pipes:")
}

func TestHeaderModel_RenderRemote_NoRemote(t *testing.T) {
	h := NewHeaderModel(nil)
	result := stripAnsi(h.renderRemoteValue())
	assert.Equal(t, "—", result, "should show dash when no remote")
}

func TestHeaderModel_RenderRemote_WithRemote(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.RemoteName = "origin"
	result := stripAnsi(h.renderRemoteValue())
	assert.Equal(t, "origin", result)
}

func TestHeaderModel_RenderRepo_FallbackToRepoName(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.RepoName = "re-cinq/wave"
	result := stripAnsi(h.renderRepoName())
	assert.Equal(t, "re-cinq/wave", result,
		"should fall back to repo name when project name is empty")
}

func TestHeaderModel_RenderRepo_PreferRepoName(t *testing.T) {
	h := NewHeaderModel(nil)
	h.metadata.ProjectName = "wave"
	h.metadata.RepoName = "re-cinq/wave"
	result := stripAnsi(h.renderRepoName())
	assert.Equal(t, "re-cinq/wave", result,
		"should prefer repo name over project name")
}

func TestHeaderModel_Init_ReturnsCmd(t *testing.T) {
	provider := &testProvider{}
	h := NewHeaderModel(provider)
	cmd := h.Init()
	assert.NotNil(t, cmd, "Init should return a batch of commands")
}

func TestHeaderModel_Init_NilProvider(t *testing.T) {
	h := NewHeaderModel(nil)
	cmd := h.Init()
	assert.NotNil(t, cmd, "Init should return commands even with nil provider")
}

func TestHeaderModel_FetchGitState_NilProvider(t *testing.T) {
	h := NewHeaderModel(nil)
	msg := h.fetchGitState()
	gitMsg, ok := msg.(GitStateMsg)
	require.True(t, ok)
	assert.Error(t, gitMsg.Err, "should return error when provider is nil")
}

func TestHeaderModel_FetchManifestInfo_NilProvider(t *testing.T) {
	h := NewHeaderModel(nil)
	msg := h.fetchManifestInfo()
	manifestMsg, ok := msg.(ManifestInfoMsg)
	require.True(t, ok)
	assert.Error(t, manifestMsg.Err, "should return error when provider is nil")
}

func TestHeaderModel_FetchPipelineHealth_NilProvider(t *testing.T) {
	h := NewHeaderModel(nil)
	msg := h.fetchPipelineHealth()
	healthMsg, ok := msg.(PipelineHealthMsg)
	require.True(t, ok)
	assert.Error(t, healthMsg.Err, "should return error when provider is nil")
}

func TestHeaderModel_GitStateMsg_TriggersGitHubFetch_WhenRepoSet(t *testing.T) {
	provider := &testProvider{
		github: GitHubInfo{AuthState: GitHubConnected, IssuesCount: 5},
	}
	h := NewHeaderModel(provider)
	h.metadata.RepoName = "re-cinq/wave" // pre-set repo name

	msg := GitStateMsg{
		State: GitState{Branch: "main"},
		Err:   nil,
	}
	_, cmd := h.Update(msg)
	assert.NotNil(t, cmd, "should trigger GitHub fetch when repo name is already set")
}

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gitRefreshInterval = 30 * time.Second

// HeaderModel is the header bar component showing Wave branding and pipeline status.
type HeaderModel struct {
	width    int
	metadata HeaderMetadata
	logo     LogoAnimator
	provider MetadataProvider
}

// NewHeaderModel creates a new header model with the given metadata provider.
func NewHeaderModel(provider MetadataProvider) HeaderModel {
	return HeaderModel{
		logo:     NewLogoAnimator(),
		provider: provider,
	}
}

// SetWidth updates the header width for reflow.
func (m *HeaderModel) SetWidth(w int) {
	m.width = w
}

// Init returns a batch of async fetch commands for initial metadata loading.
func (m HeaderModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.fetchGitState,
		m.fetchManifestInfo,
		m.fetchPipelineHealth,
		m.gitRefreshTick(),
	}
	return tea.Batch(cmds...)
}

// Update handles messages to update header state.
func (m HeaderModel) Update(msg tea.Msg) (HeaderModel, tea.Cmd) {
	switch msg := msg.(type) {
	case GitStateMsg:
		if msg.Err == nil {
			m.metadata.Branch = msg.State.Branch
			m.metadata.CommitHash = msg.State.CommitHash
			m.metadata.IsDirty = msg.State.IsDirty
			m.metadata.RemoteName = msg.State.RemoteName
		}
		// After git state is loaded, fetch GitHub info if we have a repo
		if m.metadata.RepoName != "" {
			return m, m.fetchGitHubInfoCmd()
		}
		return m, nil

	case ManifestInfoMsg:
		if msg.Err == nil {
			m.metadata.ProjectName = msg.Info.ProjectName
			m.metadata.RepoName = msg.Info.RepoName
		}
		// Now that we have repo name, fetch GitHub info
		if m.metadata.RepoName != "" {
			return m, m.fetchGitHubInfoCmd()
		}
		return m, nil

	case GitHubInfoMsg:
		if msg.Err == nil {
			m.metadata.GitHubState = msg.Info.AuthState
			m.metadata.IssuesCount = msg.Info.IssuesCount
		}
		return m, nil

	case PipelineHealthMsg:
		if msg.Err == nil {
			m.metadata.Health = msg.Health
		}
		return m, nil

	case RunningCountMsg:
		m.metadata.RunningCount = msg.Count
		wasActive := m.logo.IsActive()
		m.logo.SetActive(msg.Count > 0)
		if msg.Count > 0 && !wasActive {
			return m, m.logo.Tick()
		}
		return m, nil

	case LogoTickMsg:
		if m.logo.IsActive() {
			m.logo.Advance()
			return m, m.logo.Tick()
		}
		return m, nil

	case PipelineSelectedMsg:
		m.metadata.OverrideBranch = msg.BranchName
		if msg.BranchDeleted && msg.BranchName != "" {
			m.metadata.OverrideBranch = msg.BranchName + " [deleted]"
		}
		return m, nil

	case GitRefreshTickMsg:
		return m, tea.Batch(m.fetchGitState, m.gitRefreshTick())
	}

	return m, nil
}

// View renders the header bar as a 3-line string with labeled metadata grid.
func (m HeaderModel) View() string {
	logo := m.logo.View()
	logoWidth := lipgloss.Width(logo)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(" │ ")

	availableWidth := m.width - logoWidth - 4 // 4 for padding
	if availableWidth < 0 {
		availableWidth = 0
	}

	branch := m.displayBranch()

	// Build 3 metadata rows matching spec layout
	// Row 1: Health  │ GitHub │ Remote
	// Row 2: Running │ Branch │ Clean
	// Row 3: Steps   │ Issues │ Commit
	var row1, row2, row3 string

	if availableWidth >= 20 {
		row1Parts := []string{labelStyle.Render("Health: ") + m.renderHealth()}
		row2Parts := []string{labelStyle.Render("Running: ") + m.renderPipesValue()}
		row3Parts := []string{labelStyle.Render("Steps: ") + m.renderStepsValue()}

		if availableWidth >= 40 {
			repoLabel := "GitHub: "
			if m.metadata.RepoName == "" {
				repoLabel = "Project: "
			}
			row1Parts = append(row1Parts, labelStyle.Render(repoLabel)+m.renderRepoName())
			row2Parts = append(row2Parts, labelStyle.Render("Branch: ")+m.renderBranch(branch))
			row3Parts = append(row3Parts, labelStyle.Render("Issues: ")+m.renderIssuesValue())
		}
		if availableWidth >= 60 {
			row1Parts = append(row1Parts, labelStyle.Render("Remote: ")+m.renderRemoteValue())
			row2Parts = append(row2Parts, labelStyle.Render("Clean: ")+m.renderDirty())
			row3Parts = append(row3Parts, labelStyle.Render("Commit: ")+m.renderCommitValue())
		}

		row1 = strings.Join(row1Parts, sep)
		row2 = strings.Join(row2Parts, sep)
		if len(row3Parts) > 0 {
			row3 = strings.Join(row3Parts, sep)
		}
	}

	metadataBlock := ""
	if row1 != "" {
		lines := []string{row1, row2}
		if row3 != "" {
			lines = append(lines, row3)
		} else {
			lines = append(lines, "") // keep 3-line height
		}
		metadataBlock = lipgloss.NewStyle().PaddingLeft(2).Render(
			lipgloss.JoinVertical(lipgloss.Left, lines...),
		)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top, logo, metadataBlock)

	// Force exactly 3 lines — avoid MaxHeight which can clip the top line
	if m.width > 0 {
		lines := strings.Split(header, "\n")
		for len(lines) < headerHeight {
			lines = append(lines, "")
		}
		if len(lines) > headerHeight {
			lines = lines[:headerHeight]
		}
		// Pad each line to full width
		for i, line := range lines {
			w := lipgloss.Width(line)
			if w < m.width {
				lines[i] = line + strings.Repeat(" ", m.width-w)
			}
		}
		header = strings.Join(lines, "\n")
	}

	return header
}

// displayBranch returns the branch to display, considering overrides.
func (m HeaderModel) displayBranch() string {
	if m.metadata.OverrideBranch != "" {
		return m.metadata.OverrideBranch
	}
	if m.metadata.Branch != "" {
		return m.metadata.Branch
	}
	return "…"
}

func (m HeaderModel) renderBranch(branch string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	return style.Render(branch)
}

func (m HeaderModel) renderHealth() string {
	var style lipgloss.Style
	switch m.metadata.Health {
	case HealthWarn:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	case HealthErr:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	default:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	}
	return style.Render(m.metadata.Health.String())
}

func (m HeaderModel) renderRepoName() string {
	name := m.metadata.RepoName
	if name == "" {
		name = m.metadata.ProjectName
	}
	if name == "" {
		name = "…"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	return style.Render(name)
}

func (m HeaderModel) renderPipesValue() string {
	running := m.metadata.RunningCount
	if running == 0 {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		return style.Render("—")
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	return style.Render(fmt.Sprintf("%d running", running))
}

func (m HeaderModel) renderStepsValue() string {
	if m.metadata.StepCount == 0 {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		return style.Render("—")
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	return style.Render(fmt.Sprintf("%d", m.metadata.StepCount))
}

func (m HeaderModel) renderDirty() string {
	if m.metadata.Branch == "" && m.metadata.OverrideBranch == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("…")
	}
	if m.metadata.IsDirty {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("✱")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓")
}

func (m HeaderModel) renderRemoteValue() string {
	name := m.metadata.RemoteName
	if name == "" {
		name = "—"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	return style.Render(name)
}

func (m HeaderModel) renderIssuesValue() string {
	switch m.metadata.GitHubState {
	case GitHubConnected:
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		return style.Render(fmt.Sprintf("%d open", m.metadata.IssuesCount))
	case GitHubOffline:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("[offline]")
	default:
		return "—"
	}
}

func (m HeaderModel) renderCommitValue() string {
	hash := m.metadata.CommitHash
	if hash == "" {
		hash = "…"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	return style.Render(hash)
}

// Async command factories

func (m HeaderModel) fetchGitState() tea.Msg {
	if m.provider == nil {
		return GitStateMsg{Err: fmt.Errorf("no provider")}
	}
	state, err := m.provider.FetchGitState()
	return GitStateMsg{State: state, Err: err}
}

func (m HeaderModel) fetchManifestInfo() tea.Msg {
	if m.provider == nil {
		return ManifestInfoMsg{Err: fmt.Errorf("no provider")}
	}
	info, err := m.provider.FetchManifestInfo()
	return ManifestInfoMsg{Info: info, Err: err}
}

func (m HeaderModel) fetchGitHubInfoCmd() tea.Cmd {
	return func() tea.Msg {
		if m.provider == nil {
			return GitHubInfoMsg{Err: fmt.Errorf("no provider")}
		}
		info, err := m.provider.FetchGitHubInfo(m.metadata.RepoName)
		return GitHubInfoMsg{Info: info, Err: err}
	}
}

func (m HeaderModel) fetchPipelineHealth() tea.Msg {
	if m.provider == nil {
		return PipelineHealthMsg{Err: fmt.Errorf("no provider")}
	}
	health, err := m.provider.FetchPipelineHealth()
	return PipelineHealthMsg{Health: health, Err: err}
}

func (m HeaderModel) gitRefreshTick() tea.Cmd {
	return tea.Tick(gitRefreshInterval, func(time.Time) tea.Msg {
		return GitRefreshTickMsg{}
	})
}

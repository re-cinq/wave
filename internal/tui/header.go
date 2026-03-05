package tui

import (
	"fmt"
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

// View renders the header bar as a 3-line string.
func (m HeaderModel) View() string {
	logo := m.logo.View()
	logoWidth := lipgloss.Width(logo)

	// Build metadata columns in priority order (FR-009)
	// Priority: logo > branch > health > repo > dirty > remote > issues > commit
	type column struct {
		content  string
		minWidth int
	}

	branch := m.displayBranch()
	columns := []column{
		{content: m.renderBranch(branch), minWidth: 10},
		{content: m.renderHealth(), minWidth: 6},
		{content: m.renderRepo(), minWidth: 10},
		{content: m.renderDirty(), minWidth: 3},
		{content: m.renderRemote(), minWidth: 8},
		{content: m.renderIssues(), minWidth: 5},
		{content: m.renderCommit(), minWidth: 9},
	}

	availableWidth := m.width - logoWidth - 2 // 2 for spacing
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Add columns that fit within available width
	var visibleColumns []string
	usedWidth := 0
	for _, col := range columns {
		colWidth := lipgloss.Width(col.content)
		if colWidth == 0 {
			continue
		}
		if usedWidth+colWidth+2 <= availableWidth { // +2 for separator spacing
			visibleColumns = append(visibleColumns, col.content)
			usedWidth += colWidth + 2
		}
	}

	// Build the metadata section - stack on first line
	metadataStyle := lipgloss.NewStyle().PaddingLeft(2)
	metadataBlock := ""
	if len(visibleColumns) > 0 {
		metadataBlock = metadataStyle.Render(
			lipgloss.JoinHorizontal(lipgloss.Top, joinWithSep(visibleColumns)...),
		)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top, logo, metadataBlock)

	if m.width > 0 {
		header = lipgloss.NewStyle().Width(m.width).MaxHeight(3).Render(header)
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
	return style.Render(fmt.Sprintf(" %s", branch))
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

func (m HeaderModel) renderRepo() string {
	name := m.metadata.ProjectName
	if name == "" {
		name = m.metadata.RepoName
	}
	if name == "" {
		name = "…"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	return style.Render(name)
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

func (m HeaderModel) renderRemote() string {
	name := m.metadata.RemoteName
	if name == "" {
		name = "—"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	return style.Render(name)
}

func (m HeaderModel) renderIssues() string {
	switch m.metadata.GitHubState {
	case GitHubConnected:
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		return style.Render(fmt.Sprintf("⚑ %d", m.metadata.IssuesCount))
	case GitHubOffline:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("[offline]")
	default:
		return "" // Not configured — don't show anything
	}
}

func (m HeaderModel) renderCommit() string {
	hash := m.metadata.CommitHash
	if hash == "" {
		hash = "…"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	return style.Render(hash)
}

// joinWithSep adds separator spacing between column strings.
func joinWithSep(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items)*2-1)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(" │ ")
	for i, item := range items {
		if i > 0 {
			result = append(result, sep)
		}
		result = append(result, item)
	}
	return result
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

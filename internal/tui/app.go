package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	headerHeight    = 3
	statusBarHeight = 1
	minWidth        = 80
	minHeight       = 24
)

// AppModel is the root Bubble Tea model composing the 3-row TUI layout.
type AppModel struct {
	width        int
	height       int
	header       HeaderModel
	content      ContentModel
	statusBar    StatusBarModel
	shuttingDown bool
	ready        bool
}

// NewAppModel creates a new root app model with default child components.
func NewAppModel(metaProvider MetadataProvider, pipelineProvider PipelineDataProvider, detailProvider DetailDataProvider, deps LaunchDependencies) AppModel {
	return AppModel{
		header:    NewHeaderModel(metaProvider),
		content:   NewContentModel(pipelineProvider, detailProvider, deps),
		statusBar: NewStatusBarModel(),
	}
}

// Init implements tea.Model. Returns commands from header and content for async data loading.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.header.Init(), m.content.Init())
}

// Update implements tea.Model. Handles key events and window resize.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.shuttingDown {
				os.Exit(0)
			}
			m.shuttingDown = true
			m.content.CancelAll()
			return m, tea.Quit
		default:
			if msg.String() == "q" && !m.content.list.filtering && m.content.focus == FocusPaneLeft {
				m.content.CancelAll()
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		m.header.SetWidth(m.width)
		m.statusBar.SetWidth(m.width)

		contentHeight := m.height - headerHeight - statusBarHeight
		if contentHeight < 0 {
			contentHeight = 0
		}
		m.content.SetSize(m.width, contentHeight)
	}

	// Forward all messages to header for state updates
	var headerCmd tea.Cmd
	m.header, headerCmd = m.header.Update(msg)
	if headerCmd != nil {
		cmds = append(cmds, headerCmd)
	}

	// Forward all messages to content for list and detail updates
	var contentCmd tea.Cmd
	m.content, contentCmd = m.content.Update(msg)
	if contentCmd != nil {
		cmds = append(cmds, contentCmd)
	}

	// Forward FocusChangedMsg, FormActiveMsg, and LiveOutputActiveMsg to status bar
	switch msg.(type) {
	case FocusChangedMsg, FormActiveMsg, LiveOutputActiveMsg:
		m.statusBar, _ = m.statusBar.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model. Renders the 3-row layout.
func (m AppModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.width < minWidth || m.height < minHeight {
		return fmt.Sprintf(
			"Terminal too small. Minimum: %d×%d. Current: %d×%d",
			minWidth, minHeight, m.width, m.height,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.header.View(),
		m.content.View(),
		m.statusBar.View(),
	)
}

// RunTUI creates and runs the Bubble Tea program with alternate screen.
func RunTUI(deps LaunchDependencies) error {
	metaProvider := &DefaultMetadataProvider{}
	model := NewAppModel(metaProvider, nil, nil, deps)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if model.content.launcher != nil {
		model.content.launcher.SetProgram(p)
	}
	_, err := p.Run()
	return err
}

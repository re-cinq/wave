package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// ===========================================================================
// ViewType tests
// ===========================================================================

func TestViewType_String(t *testing.T) {
	tests := []struct {
		view     ViewType
		expected string
	}{
		{ViewPipelines, "Pipelines"},
		{ViewPersonas, "Personas"},
		{ViewContracts, "Contracts"},
		{ViewSkills, "Skills"},
		{ViewHealth, "Health"},
		{ViewIssues, "Issues"},
		{ViewPullRequests, "Pull Requests"},
		{ViewSuggest, "Suggest"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.view.String())
		})
	}
}

func TestViewType_String_Unknown(t *testing.T) {
	v := ViewType(99)
	assert.Equal(t, "Unknown", v.String())
}

// ===========================================================================
// Content model view cycling tests
// ===========================================================================

func TestContentModel_TabCyclesView(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	assert.Equal(t, ViewPipelines, c.currentView)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	c, cmd := c.Update(msg)

	assert.Equal(t, ViewPersonas, c.currentView)
	assert.NotNil(t, cmd)
}

func TestContentModel_TabCyclesThroughAllViews(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{
			PersonaProvider:  &mockPersonaDataProvider{},
			ContractProvider: &mockContractDataProvider{},
			SkillProvider:    &mockSkillDataProvider{},
			HealthProvider:   &mockHealthDataProvider{},
			IssueProvider:    &mockIssueDataProvider{},
			PRProvider:       &mockPRDataProvider{},
		},
	)
	c.SetSize(120, 40)

	views := []ViewType{ViewPersonas, ViewContracts, ViewSkills, ViewHealth, ViewIssues, ViewPullRequests, ViewSuggest, ViewPipelines}
	msg := tea.KeyMsg{Type: tea.KeyTab}

	for _, expected := range views {
		c, _ = c.Update(msg)
		assert.Equal(t, expected, c.currentView)
	}
}

func TestContentModel_TabResetsFocusToLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	// Set focus to right pane
	c.focus = FocusPaneRight

	msg := tea.KeyMsg{Type: tea.KeyTab}
	c, _ = c.Update(msg)

	assert.Equal(t, FocusPaneLeft, c.focus)
}

func TestContentModel_TabEmitsViewChangedMsg(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	c, cmd := c.Update(msg)

	assert.NotNil(t, cmd)
	result := cmd()
	if batch, ok := result.(tea.BatchMsg); ok {
		foundViewChanged := false
		for _, batchCmd := range batch {
			if batchCmd == nil {
				continue
			}
			innerMsg := batchCmd()
			if vcMsg, ok := innerMsg.(ViewChangedMsg); ok {
				foundViewChanged = true
				assert.Equal(t, ViewPersonas, vcMsg.View)
			}
		}
		assert.True(t, foundViewChanged, "should emit ViewChangedMsg")
	}
}

func TestContentModel_TabInFormMode_ForwardedToForm(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	// Set detail to configuring state
	c.detail.paneState = stateConfiguring

	// Tab should be forwarded to the detail (form) instead of cycling
	msg := tea.KeyMsg{Type: tea.KeyTab}
	c, _ = c.Update(msg)

	// Should remain on Pipelines view
	assert.Equal(t, ViewPipelines, c.currentView)
}

func TestContentModel_ViewPersistsAfterTab(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	// Tab to Personas
	msg := tea.KeyMsg{Type: tea.KeyTab}
	c, _ = c.Update(msg)
	assert.Equal(t, ViewPersonas, c.currentView)

	// View should render with persona content
	view := c.View()
	assert.NotEmpty(t, view)
}

// ===========================================================================
// Alternative view Enter/Escape focus tests
// ===========================================================================

func TestContentModel_AlternativeView_EnterFocusesRight(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	// Tab to Personas (this initializes personaList)
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, ViewPersonas, c.currentView)

	// Press Enter
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, FocusPaneRight, c.focus)
}

func TestContentModel_AlternativeView_EscapeReturnsFocusLeft(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	// Tab to Personas, then focus right
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, FocusPaneRight, c.focus)

	// Press Escape
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, FocusPaneLeft, c.focus)
}

// ===========================================================================
// StatusBar view switching tests
// ===========================================================================

func TestStatusBarModel_ViewChangedMsg_SetsView(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(ViewChangedMsg{View: ViewPersonas})
	assert.Equal(t, ViewPersonas, sb.currentView)
}

func TestStatusBarModel_ViewChangedMsg_UpdatesLabel(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(ViewChangedMsg{View: ViewPersonas})
	view := sb.View()
	assert.Contains(t, view, "Personas")
}

func TestStatusBarModel_ViewChangedMsg_HealthLabel(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(ViewChangedMsg{View: ViewHealth})
	view := sb.View()
	assert.Contains(t, view, "Health")
}

func TestStatusBarModel_TabHint_ShowsInDefaultView(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	view := sb.View()
	assert.Contains(t, view, "Tab/Shift+Tab: views")
}

func TestStatusBarModel_HealthView_ShowsRecheckHint(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(ViewChangedMsg{View: ViewHealth})
	view := sb.View()
	assert.Contains(t, view, "r: recheck")
}

// ===========================================================================
// PersonaListModel tests
// ===========================================================================

func TestPersonaListModel_InitFetchesData(t *testing.T) {
	provider := &mockPersonaDataProvider{}
	m := NewPersonaListModel(provider)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestPersonaListModel_DataLoading(t *testing.T) {
	provider := &mockPersonaDataProvider{}
	m := NewPersonaListModel(provider)
	m.SetSize(40, 20)

	msg := PersonaDataMsg{
		Personas: []PersonaInfo{
			{Name: "navigator", Description: "Navigation persona"},
			{Name: "craftsman", Description: "Implementation persona"},
		},
	}

	m, _ = m.Update(msg)
	assert.Equal(t, 2, len(m.items))
	assert.Equal(t, 2, len(m.navigable))
	assert.True(t, m.loaded)
}

func TestPersonaListModel_Navigation(t *testing.T) {
	m := NewPersonaListModel(&mockPersonaDataProvider{})
	m.SetSize(40, 20)

	m, _ = m.Update(PersonaDataMsg{
		Personas: []PersonaInfo{
			{Name: "navigator"},
			{Name: "craftsman"},
		},
	})

	assert.Equal(t, 0, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)
}

func TestPersonaListModel_FilterMode(t *testing.T) {
	m := NewPersonaListModel(&mockPersonaDataProvider{})
	m.SetSize(40, 20)

	m, _ = m.Update(PersonaDataMsg{
		Personas: []PersonaInfo{
			{Name: "navigator"},
			{Name: "craftsman"},
		},
	})

	// Activate filter
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.filtering)

	// Deactivate filter with Escape
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.filtering)
}

func TestPersonaListModel_EmptyState(t *testing.T) {
	m := NewPersonaListModel(&mockPersonaDataProvider{})
	m.SetSize(40, 20)
	m, _ = m.Update(PersonaDataMsg{})

	view := m.View()
	assert.Contains(t, view, "No personas configured")
}

func TestPersonaListModel_ViewRendersItems(t *testing.T) {
	m := NewPersonaListModel(&mockPersonaDataProvider{})
	m.SetSize(40, 20)

	m, _ = m.Update(PersonaDataMsg{
		Personas: []PersonaInfo{{Name: "navigator"}},
	})

	view := m.View()
	assert.Contains(t, view, "navigator")
}

// ===========================================================================
// PersonaDetailModel tests
// ===========================================================================

func TestPersonaDetailModel_EmptyState(t *testing.T) {
	m := NewPersonaDetailModel(&mockPersonaDataProvider{})
	m.SetSize(80, 40)

	view := m.View()
	assert.Contains(t, view, "Select a persona to view details")
}

func TestPersonaDetailModel_SetPersona(t *testing.T) {
	m := NewPersonaDetailModel(&mockPersonaDataProvider{})
	m.SetSize(80, 40)

	info := &PersonaInfo{
		Name:        "navigator",
		Description: "Read-only analysis",
		Adapter:     "claude",
		Model:       "sonnet",
	}
	m.SetPersona(info)

	view := m.View()
	assert.Contains(t, view, "navigator")
	assert.Contains(t, view, "Read-only analysis")
	assert.Contains(t, view, "claude")
}

func TestPersonaDetailModel_SetSize(t *testing.T) {
	m := NewPersonaDetailModel(&mockPersonaDataProvider{})
	m.SetSize(80, 40)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 40, m.height)
}

// ===========================================================================
// ContractListModel tests
// ===========================================================================

func TestContractListModel_DataLoading(t *testing.T) {
	m := NewContractListModel(&mockContractDataProvider{})
	m.SetSize(40, 20)

	msg := ContractDataMsg{
		Contracts: []ContractInfo{
			{Label: "spec.json", Type: "json_schema"},
			{Label: "test-contract", Type: "test_suite"},
		},
	}

	m, _ = m.Update(msg)
	assert.Equal(t, 2, len(m.items))
	assert.True(t, m.loaded)
}

func TestContractListModel_EmptyState(t *testing.T) {
	m := NewContractListModel(&mockContractDataProvider{})
	m.SetSize(40, 20)
	m, _ = m.Update(ContractDataMsg{})

	view := m.View()
	assert.Contains(t, view, "No contracts configured")
}

func TestContractListModel_ViewRendersBadge(t *testing.T) {
	m := NewContractListModel(&mockContractDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(ContractDataMsg{
		Contracts: []ContractInfo{{Label: "spec.json", Type: "json_schema"}},
	})

	view := m.View()
	assert.Contains(t, view, "spec.json")
	assert.Contains(t, view, "[json_schema]")
}

// ===========================================================================
// ContractDetailModel tests
// ===========================================================================

func TestContractDetailModel_EmptyState(t *testing.T) {
	m := NewContractDetailModel()
	m.SetSize(80, 40)

	view := m.View()
	assert.Contains(t, view, "Select a contract to view details")
}

func TestContractDetailModel_SetContract(t *testing.T) {
	m := NewContractDetailModel()
	m.SetSize(80, 40)

	info := &ContractInfo{
		Label:      "spec.json",
		Type:       "json_schema",
		SchemaPath: ".agents/contracts/spec.json",
	}
	m.SetContract(info)

	view := m.View()
	assert.Contains(t, view, "spec.json")
	assert.Contains(t, view, "json_schema")
}

// ===========================================================================
// SkillListModel tests
// ===========================================================================

func TestSkillListModel_DataLoading(t *testing.T) {
	m := NewSkillListModel(&mockSkillDataProvider{})
	m.SetSize(40, 20)

	msg := SkillDataMsg{
		Skills: []SkillInfo{
			{Name: "speckit", CommandFiles: []string{"cmd1.md", "cmd2.md"}},
		},
	}

	m, _ = m.Update(msg)
	assert.Equal(t, 1, len(m.items))
	assert.True(t, m.loaded)
}

func TestSkillListModel_EmptyState(t *testing.T) {
	m := NewSkillListModel(&mockSkillDataProvider{})
	m.SetSize(40, 20)
	m, _ = m.Update(SkillDataMsg{})

	view := m.View()
	assert.Contains(t, view, "No skills configured")
}

func TestSkillListModel_ViewRendersBadge(t *testing.T) {
	m := NewSkillListModel(&mockSkillDataProvider{})
	m.SetSize(60, 20)

	m, _ = m.Update(SkillDataMsg{
		Skills: []SkillInfo{{Name: "speckit", CommandFiles: []string{"cmd1.md", "cmd2.md"}}},
	})

	view := m.View()
	assert.Contains(t, view, "speckit")
	assert.Contains(t, view, "(2 cmds)")
}

// ===========================================================================
// SkillDetailModel tests
// ===========================================================================

func TestSkillDetailModel_EmptyState(t *testing.T) {
	m := NewSkillDetailModel()
	m.SetSize(80, 40)

	view := m.View()
	assert.Contains(t, view, "Select a skill to view details")
}

func TestSkillDetailModel_SetSkill(t *testing.T) {
	m := NewSkillDetailModel()
	m.SetSize(80, 40)

	info := &SkillInfo{
		Name:         "speckit",
		CommandsGlob: ".claude/commands/speckit.*.md",
		CommandFiles: []string{".claude/commands/speckit.specify.md"},
	}
	m.SetSkill(info)

	view := m.View()
	assert.Contains(t, view, "speckit")
	assert.Contains(t, view, "speckit.specify.md")
}

// ===========================================================================
// HealthListModel tests
// ===========================================================================

func TestHealthListModel_InitRunsAllChecks(t *testing.T) {
	provider := &mockHealthDataProvider{}
	m := NewHealthListModel(provider)

	assert.Equal(t, len(provider.CheckNames()), len(m.checks))
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestHealthListModel_CheckResultUpdatesStatus(t *testing.T) {
	provider := &mockHealthDataProvider{}
	m := NewHealthListModel(provider)
	m.SetSize(40, 20)

	// All checks start as HealthCheckChecking
	assert.Equal(t, HealthCheckChecking, m.checks[0].Status)

	// Simulate result arriving
	m, _ = m.Update(HealthCheckResultMsg{
		Name:    provider.CheckNames()[0],
		Status:  HealthCheckOK,
		Message: "OK",
	})

	assert.Equal(t, HealthCheckOK, m.checks[0].Status)
	assert.Equal(t, "OK", m.checks[0].Message)
}

func TestHealthListModel_RKeyRerunsChecks(t *testing.T) {
	provider := &mockHealthDataProvider{}
	m := NewHealthListModel(provider)
	m.SetSize(40, 20)

	// Set first check to OK
	m.checks[0].Status = HealthCheckOK

	// Press 'r'
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// All checks should be reset to Checking
	assert.Equal(t, HealthCheckChecking, m.checks[0].Status)
	assert.NotNil(t, cmd)
}

func TestHealthListModel_Navigation(t *testing.T) {
	provider := &mockHealthDataProvider{}
	m := NewHealthListModel(provider)
	m.SetSize(40, 20)

	assert.Equal(t, 0, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)
}

// ===========================================================================
// HealthDetailModel tests
// ===========================================================================

func TestHealthDetailModel_EmptyState(t *testing.T) {
	m := NewHealthDetailModel()
	m.SetSize(80, 40)

	view := m.View()
	assert.Contains(t, view, "Select a health check to view details")
}

func TestHealthDetailModel_SetCheck(t *testing.T) {
	m := NewHealthDetailModel()
	m.SetSize(80, 40)

	check := &HealthCheck{
		Name:    "Git Repository",
		Status:  HealthCheckOK,
		Message: "Valid git repository",
	}
	m.SetCheck(check)

	view := m.View()
	assert.Contains(t, view, "Git Repository")
	assert.Contains(t, view, "OK")
}

func TestHealthDetailModel_CheckResultUpdatesView(t *testing.T) {
	m := NewHealthDetailModel()
	m.SetSize(80, 40)

	check := &HealthCheck{
		Name:   "Git Repository",
		Status: HealthCheckChecking,
	}
	m.SetCheck(check)

	m, _ = m.Update(HealthCheckResultMsg{
		Name:    "Git Repository",
		Status:  HealthCheckErr,
		Message: "Not a git repository",
	})

	view := m.View()
	assert.Contains(t, view, "Failed")
}

// ===========================================================================
// Render function tests
// ===========================================================================

func TestRenderPersonaDetail_NilInput(t *testing.T) {
	assert.Equal(t, "", renderPersonaDetail(nil, nil, 80))
}

func TestRenderPersonaDetail_WithStats(t *testing.T) {
	info := &PersonaInfo{
		Name:    "navigator",
		Adapter: "claude",
	}
	stats := &PersonaStats{
		TotalRuns:      10,
		SuccessfulRuns: 8,
		AvgDurationMs:  30000,
		LastRunAt:      time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result := renderPersonaDetail(info, stats, 80)
	assert.Contains(t, result, "navigator")
	assert.Contains(t, result, "10")
	assert.Contains(t, result, "80%")
}

func TestRenderContractDetail_NilInput(t *testing.T) {
	assert.Equal(t, "", renderContractDetail(nil, 80))
}

func TestRenderContractDetail_WithUsage(t *testing.T) {
	info := &ContractInfo{
		Label: "spec.json",
		Type:  "json_schema",
		PipelineUsage: []PipelineStepRef{
			{PipelineName: "speckit-flow", StepID: "implement"},
		},
	}

	result := renderContractDetail(info, 80)
	assert.Contains(t, result, "spec.json")
	assert.Contains(t, result, "speckit-flow")
}

func TestRenderSkillDetail_NilInput(t *testing.T) {
	assert.Equal(t, "", renderSkillDetail(nil, 80))
}

func TestRenderHealthDetail_NilInput(t *testing.T) {
	assert.Equal(t, "", renderHealthDetail(nil, 80))
}

func TestRenderHealthDetail_WithDetails(t *testing.T) {
	check := &HealthCheck{
		Name:    "Git Repository",
		Status:  HealthCheckOK,
		Message: "Valid",
		Details: map[string]string{
			"Branch": "main",
			"Remote": "origin",
		},
	}

	result := renderHealthDetail(check, 80)
	assert.Contains(t, result, "Git Repository")
	assert.Contains(t, result, "Branch")
	assert.Contains(t, result, "main")
}

// ===========================================================================
// IsFiltering tests
// ===========================================================================

func TestContentModel_IsFiltering_PipelineView(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	assert.False(t, c.IsFiltering())

	c.list.filtering = true
	assert.True(t, c.IsFiltering())
}

func TestContentModel_IsFiltering_PersonaView(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	// Tab to Personas
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, ViewPersonas, c.currentView)

	// Should not be filtering
	assert.False(t, c.IsFiltering())
}

// ===========================================================================
// Content model View rendering tests
// ===========================================================================

func TestContentModel_View_PipelinesDefault(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{})
	c.SetSize(120, 40)

	view := c.View()
	assert.Contains(t, view, "Select a pipeline to view details")
}

func TestContentModel_View_PersonasPlaceholder(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{}, nil, LaunchDependencies{},
		ContentProviders{PersonaProvider: &mockPersonaDataProvider{}},
	)
	c.SetSize(120, 40)

	// Tab to Personas
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})

	view := c.View()
	// Should contain persona content or placeholder
	assert.NotEmpty(t, view)
}

// ===========================================================================
// AppModel tests for ViewChangedMsg forwarding
// ===========================================================================

func TestAppModel_Update_ForwardsViewChangedMsgToStatusBar(t *testing.T) {
	m := NewAppModel(&mockProvider{}, &mockPipelineDataProvider{}, nil, LaunchDependencies{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(AppModel)

	viewMsg := ViewChangedMsg{View: ViewPersonas}
	updated, _ = model.Update(viewMsg)
	model = updated.(AppModel)

	assert.Equal(t, ViewPersonas, model.statusBar.currentView)
}

// ===========================================================================
// Mock providers
// ===========================================================================

type mockPersonaDataProvider struct{}

func (m *mockPersonaDataProvider) FetchPersonas() ([]PersonaInfo, error) {
	return nil, nil
}

func (m *mockPersonaDataProvider) FetchPersonaStats(name string) (*PersonaStats, error) {
	return nil, nil
}

type mockContractDataProvider struct{}

func (m *mockContractDataProvider) FetchContracts() ([]ContractInfo, error) {
	return nil, nil
}

type mockSkillDataProvider struct{}

func (m *mockSkillDataProvider) FetchSkills() ([]SkillInfo, error) {
	return nil, nil
}

type mockHealthDataProvider struct{}

func (m *mockHealthDataProvider) CheckNames() []string {
	return []string{"Git", "Adapters"}
}

func (m *mockHealthDataProvider) RunCheck(name string) HealthCheckResultMsg {
	return HealthCheckResultMsg{
		Name:    name,
		Status:  HealthCheckOK,
		Message: "OK",
	}
}

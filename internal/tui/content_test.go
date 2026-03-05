package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type contentTestPipelineProvider struct{}

func (m *contentTestPipelineProvider) FetchRunningPipelines() ([]RunningPipeline, error) {
	return nil, nil
}

func (m *contentTestPipelineProvider) FetchFinishedPipelines(limit int) ([]FinishedPipeline, error) {
	return nil, nil
}

func (m *contentTestPipelineProvider) FetchAvailablePipelines() ([]PipelineInfo, error) {
	return nil, nil
}

func TestContentModel_NewContentModel(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{})
	assert.True(t, c.list.focused)
}

func TestContentModel_SetSize(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{})
	assert.Equal(t, 0, c.width)
	assert.Equal(t, 0, c.height)

	c.SetSize(120, 40)
	assert.Equal(t, 120, c.width)
	assert.Equal(t, 40, c.height)
}

func TestContentModel_SetSize_PropagatesListDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{})
	c.SetSize(120, 40)

	// Left pane: 30% of 120 = 36, clamped to [25, 50] -> 36
	assert.Equal(t, 36, c.list.width)
	assert.Equal(t, 40, c.list.height)
}

func TestContentModel_LeftPaneWidth(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected int
	}{
		{"30 percent of 120", 120, 36},
		{"minimum 25", 60, 25},  // 30% of 60 = 18 -> clamped to 25
		{"maximum 50", 200, 50}, // 30% of 200 = 60 -> clamped to 50
		{"exact 100", 100, 30},  // 30% of 100 = 30
		{"narrow 80", 80, 25},   // 30% of 80 = 24 -> clamped to 25
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewContentModel(&contentTestPipelineProvider{})
			c.SetSize(tt.width, 40)
			assert.Equal(t, tt.expected, c.list.width)
		})
	}
}

func TestContentModel_View_RightPanePlaceholder(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{})
	c.SetSize(120, 40)
	view := c.View()
	assert.Contains(t, view, "Select a pipeline to view details")
}

func TestContentModel_View_ZeroDimensions(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{})
	view := c.View()
	assert.Equal(t, "", view)
}

func TestContentModel_Init_ReturnsCommands(t *testing.T) {
	c := NewContentModel(&contentTestPipelineProvider{})
	cmd := c.Init()
	assert.NotNil(t, cmd)
}

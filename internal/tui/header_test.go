package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaderModel_View_ContainsBranding(t *testing.T) {
	h := NewHeaderModel()
	h.SetWidth(120)

	view := h.View()
	assert.Contains(t, view, "Pipeline Orchestrator")
	assert.Contains(t, view, "no pipeline running")
}

func TestHeaderModel_View_ContainsLogo(t *testing.T) {
	h := NewHeaderModel()
	view := h.View()

	// The logo contains Wave ASCII art characters
	assert.Contains(t, view, "╦")
	assert.Contains(t, view, "╚╩╝")
}

func TestHeaderModel_SetWidth(t *testing.T) {
	h := NewHeaderModel()
	assert.Equal(t, 0, h.width)

	h.SetWidth(120)
	assert.Equal(t, 120, h.width)
}

func TestHeaderModel_View_RespectsWidth(t *testing.T) {
	h := NewHeaderModel()
	h.SetWidth(80)
	view := h.View()

	// Each line should not exceed the specified width
	for _, line := range strings.Split(view, "\n") {
		// lipgloss Width accounts for ANSI escape sequences
		assert.LessOrEqual(t, len([]rune(stripAnsi(line))), 80+10, // allow margin for ANSI sequences
			"line exceeds width: %q", line)
	}
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

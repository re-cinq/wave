package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// WaveTheme returns a huh.Theme that matches Wave's TUI design language.
// Colors are drawn from the progress display palette: cyan primary, gray muted, white text.
func WaveTheme() *huh.Theme {
	t := huh.ThemeBase()

	var (
		cyan  = lipgloss.Color("6")   // Wave primary — matches logo (standard cyan)
		white = lipgloss.Color("7")   // Primary text
		muted = lipgloss.Color("244") // Secondary/description text
		red   = lipgloss.Color("1")   // Errors
	)

	// Focused field styles.
	t.Focused.Base = t.Focused.Base.BorderForeground(cyan)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(white).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(white).Bold(true).MarginBottom(1)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)

	// Select styles.
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(cyan)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(cyan)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(cyan)
	t.Focused.Option = t.Focused.Option.Foreground(white)

	// Multi-select styles — selected items use cyan to match the border bar.
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(cyan)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(cyan)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(cyan).SetString("[✓] ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("[ ] ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(white)

	// Text input styles.
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(cyan)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(cyan)
	t.Focused.TextInput.Text = t.Focused.TextInput.Text.Foreground(white)

	// Button styles.
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("0")).Background(cyan)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(lipgloss.Color("237"))

	// Blurred field styles (inherit focused, dim the border).
	// Keep text input text cyan so entered values stay visible as confirmation.
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(cyan)

	// Group styles.
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// activeSelectionStyle returns the style for selected items in the focused pane.
// White background with dark foreground for high contrast.
func activeSelectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("7")).
		Foreground(lipgloss.Color("0")).
		Bold(true)
}

// inactiveSelectionStyle returns the style for selected items in unfocused panes.
// Dimmed background matching the border separator color.
func inactiveSelectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("7"))
}

// SelectionStyle returns the active or inactive selection style based on focus state.
func SelectionStyle(focused bool) lipgloss.Style {
	if focused {
		return activeSelectionStyle()
	}
	return inactiveSelectionStyle()
}

// WaveLogo returns the styled Wave ASCII logo matching the list/run display header.
func WaveLogo() string {
	logo := "╦ ╦╔═╗╦  ╦╔═╗\n║║║╠═╣╚╗╔╝║╣\n╚╩╝╩ ╩ ╚╝ ╚═╝"
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")). // Standard cyan — matches Formatter.Primary()
		Margin(1, 0, 1, 1).              // top, right, bottom, left (1-char indent, matches TUI margin)
		Render(logo)
}

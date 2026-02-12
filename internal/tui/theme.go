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
		cyan    = lipgloss.Color("6")   // Wave primary — matches logo (standard cyan)
		white   = lipgloss.Color("7")   // Primary text
		muted   = lipgloss.Color("244") // Secondary/description text
		red     = lipgloss.Color("1")   // Errors
	)

	// Focused field styles.
	t.Focused.Base = t.Focused.Base.BorderForeground(cyan)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(cyan).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(cyan).Bold(true).MarginBottom(1)
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

	// Button styles.
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("0")).Background(cyan)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(lipgloss.Color("237"))

	// Blurred field styles (inherit focused, dim the border).
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	// Group styles.
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// WaveLogo returns the styled Wave ASCII logo matching the list/run display header.
func WaveLogo() string {
	logo := "╦ ╦╔═╗╦  ╦╔═╗\n║║║╠═╣╚╗╔╝║╣\n╚╩╝╩ ╩ ╚╝ ╚═╝"
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")). // Standard cyan — matches Formatter.Primary()
		Margin(1, 0, 1, 2).              // top, right, bottom, left (2-char indent)
		Render(logo)
}

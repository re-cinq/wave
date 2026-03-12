package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// WaveTheme returns a huh.Theme that matches Wave's TUI design language.
// Colors are drawn from the progress display palette: green primary, gray muted, white text.
func WaveTheme() *huh.Theme {
	t := huh.ThemeBase()

	var (
		green   = lipgloss.Color("2")   // Wave primary — matches logo (standard green)
		white   = lipgloss.Color("7")   // Primary text
		muted   = lipgloss.Color("244") // Secondary/description text
		red     = lipgloss.Color("1")   // Errors
	)

	// Focused field styles.
	t.Focused.Base = t.Focused.Base.BorderForeground(green)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(white).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(white).Bold(true).MarginBottom(1)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)

	// Select styles.
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(green)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(green)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(green)
	t.Focused.Option = t.Focused.Option.Foreground(white)

	// Multi-select styles — selected items use green to match the border bar.
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(green)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(green).SetString("[✓] ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("[ ] ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(white)

	// Text input styles.
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(green)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(green)
	t.Focused.TextInput.Text = t.Focused.TextInput.Text.Foreground(white)

	// Button styles.
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("0")).Background(green)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(lipgloss.Color("237"))

	// Blurred field styles (inherit focused, dim the border).
	// Keep text input text green so entered values stay visible as confirmation.
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(green)

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
		Foreground(lipgloss.Color("2")). // Standard green — matches Formatter.Primary()
		Margin(1, 0, 1, 1).              // top, right, bottom, left (1-char indent, matches TUI margin)
		Render(logo)
}

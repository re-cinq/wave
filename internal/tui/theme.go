// Package tui re-exports a small set of theme helpers from internal/uitheme
// so existing tui consumers compile unchanged after the theme extraction
// (issue #1497 residual). New code should import internal/uitheme directly.
package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/uitheme"
)

// WaveTheme returns the shared huh.Theme. Forwards to uitheme.WaveTheme.
func WaveTheme() *huh.Theme { return uitheme.WaveTheme() }

// SelectionStyle returns the focused/inactive selection style. Forwards to
// uitheme.SelectionStyle.
func SelectionStyle(focused bool) lipgloss.Style { return uitheme.SelectionStyle(focused) }

// WaveLogo returns the ASCII logo. Forwards to uitheme.WaveLogo.
func WaveLogo() string { return uitheme.WaveLogo() }

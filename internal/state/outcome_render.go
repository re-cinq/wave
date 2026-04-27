package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/pathfmt"
)

// hasNerdFont reports whether nerd-font glyph rendering is appropriate for the
// current terminal, based on common environment indicators.
func hasNerdFont() bool {
	if os.Getenv("NERD_FONT") == "1" {
		return true
	}
	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") || strings.Contains(term, "alacritty") {
		return true
	}
	if strings.Contains(os.Getenv("TERMINAL_EMULATOR"), "JetBrains") {
		return true
	}
	return false
}

// String returns a single-line, icon-prefixed rendering of the outcome suitable
// for terminal summaries.
func (r *OutcomeRecord) String() string {
	var icon string
	if hasNerdFont() {
		switch r.Type {
		case OutcomeTypeFile:
			icon = "📄"
		case OutcomeTypeURL:
			icon = "🔗"
		case OutcomeTypePR:
			icon = "🔀"
		case OutcomeTypeDeployment:
			icon = "🚀"
		case OutcomeTypeLog:
			icon = "📝"
		case OutcomeTypeContract:
			icon = "📋"
		case OutcomeTypeArtifact:
			icon = "📦"
		case OutcomeTypeBranch:
			icon = "🌿"
		case OutcomeTypeIssue:
			icon = "📌"
		default:
			icon = "📄"
		}
	} else {
		switch r.Type {
		case OutcomeTypeFile:
			icon = "•"
		case OutcomeTypeURL:
			icon = "→"
		case OutcomeTypePR:
			icon = "↗"
		case OutcomeTypeDeployment:
			icon = "↑"
		case OutcomeTypeLog:
			icon = "~"
		case OutcomeTypeContract:
			icon = "="
		case OutcomeTypeArtifact:
			icon = "+"
		case OutcomeTypeBranch:
			icon = "⎇"
		case OutcomeTypeIssue:
			icon = "!"
		default:
			icon = "•"
		}
	}

	if r.Type == OutcomeTypeFile {
		absPath, _ := filepath.Abs(r.Value)
		return fmt.Sprintf("%s %s", icon, pathfmt.FileURI(absPath))
	}
	return fmt.Sprintf("%s %s", icon, r.Value)
}

// IsTemporary returns true when the outcome represents a transient artifact
// (logs, files explicitly marked as temporary, etc.).
func (r *OutcomeRecord) IsTemporary() bool {
	return r.Type == OutcomeTypeLog ||
		strings.Contains(strings.ToLower(r.Description), "temporary") ||
		strings.Contains(strings.ToLower(r.Label), "temp")
}

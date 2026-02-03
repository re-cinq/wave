package deliverable

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeliverableType represents the kind of deliverable
type DeliverableType string

const (
	TypeFile       DeliverableType = "file"
	TypeURL        DeliverableType = "url"
	TypePR         DeliverableType = "pr"
	TypeDeployment DeliverableType = "deployment"
	TypeLog        DeliverableType = "log"
	TypeContract   DeliverableType = "contract"
	TypeArtifact   DeliverableType = "artifact"
	TypeOther      DeliverableType = "other"
)

// Deliverable represents any output or artifact from a pipeline step
type Deliverable struct {
	Type        DeliverableType `json:"type"`
	Name        string          `json:"name"`
	Path        string          `json:"path"`        // Absolute path for files, URL for links
	Description string          `json:"description"` // Human readable description
	StepID      string          `json:"step_id"`     // Which step produced this
	CreatedAt   time.Time       `json:"created_at"`
	Metadata    map[string]any  `json:"metadata,omitempty"` // Additional type-specific data
}

// hasNerdFont detects if nerd font is available by checking common environment indicators
func hasNerdFont() bool {
	// Check for common nerd font environment indicators
	if os.Getenv("NERD_FONT") == "1" {
		return true
	}

	// Check for common terminals that support nerd fonts
	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") || strings.Contains(term, "alacritty") {
		return true
	}

	// Check for common font configurations
	if strings.Contains(os.Getenv("TERMINAL_EMULATOR"), "JetBrains") {
		return true
	}

	return false
}

// String returns a formatted string representation of the deliverable
func (d *Deliverable) String() string {
	var icon string

	if hasNerdFont() {
		// Use actual nerd font icons when available
		switch d.Type {
		case TypeFile:
			icon = "📄" // nf-fa-file (using emoji fallback for reliable display)
		case TypeURL:
			icon = "🔗" // nf-fa-external_link
		case TypePR:
			icon = "🔀" // nf-dev-git_pull_request
		case TypeDeployment:
			icon = "🚀" // nf-fa-rocket
		case TypeLog:
			icon = "📝" // nf-fa-file_text
		case TypeContract:
			icon = "📋" // nf-fa-file_code
		case TypeArtifact:
			icon = "📦" // nf-fa-archive
		default:
			icon = "📄"
		}
	} else {
		// Use simple ASCII characters when nerd font not available
		switch d.Type {
		case TypeFile:
			icon = "•"
		case TypeURL:
			icon = "→"
		case TypePR:
			icon = "↗"
		case TypeDeployment:
			icon = "↑"
		case TypeLog:
			icon = "~"
		case TypeContract:
			icon = "="
		case TypeArtifact:
			icon = "+"
		default:
			icon = "•"
		}
	}

	// Simple consistent format for all types
	if d.Type == TypeFile {
		// Show absolute path for files
		absPath, _ := filepath.Abs(d.Path)
		return fmt.Sprintf("%s %s", icon, absPath)
	}

	// For non-file types, show name and path
	return fmt.Sprintf("%s %s", icon, d.Path)
}

// IsTemporary returns true if this deliverable is temporary (logs, temp files, etc.)
func (d *Deliverable) IsTemporary() bool {
	return d.Type == TypeLog ||
		strings.Contains(strings.ToLower(d.Description), "temporary") ||
		strings.Contains(strings.ToLower(d.Name), "temp")
}

// NewFileDeliverable creates a file deliverable
func NewFileDeliverable(stepID, name, path, description string) *Deliverable {
	return &Deliverable{
		Type:        TypeFile,
		Name:        name,
		Path:        path,
		Description: description,
		StepID:      stepID,
		CreatedAt:   time.Now(),
	}
}

// NewURLDeliverable creates a URL/link deliverable
func NewURLDeliverable(stepID, name, url, description string) *Deliverable {
	return &Deliverable{
		Type:        TypeURL,
		Name:        name,
		Path:        url,
		Description: description,
		StepID:      stepID,
		CreatedAt:   time.Now(),
	}
}

// NewPRDeliverable creates a pull request deliverable
func NewPRDeliverable(stepID, name, prURL, description string) *Deliverable {
	return &Deliverable{
		Type:        TypePR,
		Name:        name,
		Path:        prURL,
		Description: description,
		StepID:      stepID,
		CreatedAt:   time.Now(),
	}
}

// NewDeploymentDeliverable creates a deployment deliverable
func NewDeploymentDeliverable(stepID, name, deployURL, description string) *Deliverable {
	return &Deliverable{
		Type:        TypeDeployment,
		Name:        name,
		Path:        deployURL,
		Description: description,
		StepID:      stepID,
		CreatedAt:   time.Now(),
	}
}

// NewLogDeliverable creates a log file deliverable
func NewLogDeliverable(stepID, name, logPath, description string) *Deliverable {
	return &Deliverable{
		Type:        TypeLog,
		Name:        name,
		Path:        logPath,
		Description: description,
		StepID:      stepID,
		CreatedAt:   time.Now(),
	}
}

// NewContractDeliverable creates a contract artifact deliverable
func NewContractDeliverable(stepID, name, contractPath, description string) *Deliverable {
	return &Deliverable{
		Type:        TypeContract,
		Name:        name,
		Path:        contractPath,
		Description: description,
		StepID:      stepID,
		CreatedAt:   time.Now(),
	}
}
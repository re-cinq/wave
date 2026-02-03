package deliverable

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Tracker manages deliverables throughout pipeline execution
type Tracker struct {
	mu           sync.RWMutex
	deliverables []*Deliverable
	pipelineID   string
}

// NewTracker creates a new deliverable tracker
func NewTracker(pipelineID string) *Tracker {
	return &Tracker{
		deliverables: make([]*Deliverable, 0),
		pipelineID:   pipelineID,
	}
}

// SetPipelineID updates the pipeline ID for the tracker
func (t *Tracker) SetPipelineID(pipelineID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pipelineID = pipelineID
}

// Add records a new deliverable (avoiding duplicates)
func (t *Tracker) Add(deliverable *Deliverable) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check for duplicates by path
	for _, existing := range t.deliverables {
		if existing.Path == deliverable.Path && existing.StepID == deliverable.StepID {
			return // Skip duplicate
		}
	}

	t.deliverables = append(t.deliverables, deliverable)
}

// AddFile is a convenience method to add a file deliverable
func (t *Tracker) AddFile(stepID, name, path, description string) {
	t.Add(NewFileDeliverable(stepID, name, path, description))
}

// AddURL is a convenience method to add a URL deliverable
func (t *Tracker) AddURL(stepID, name, url, description string) {
	t.Add(NewURLDeliverable(stepID, name, url, description))
}

// AddPR is a convenience method to add a PR deliverable
func (t *Tracker) AddPR(stepID, name, prURL, description string) {
	t.Add(NewPRDeliverable(stepID, name, prURL, description))
}

// AddDeployment is a convenience method to add a deployment deliverable
func (t *Tracker) AddDeployment(stepID, name, deployURL, description string) {
	t.Add(NewDeploymentDeliverable(stepID, name, deployURL, description))
}

// AddLog is a convenience method to add a log deliverable
func (t *Tracker) AddLog(stepID, name, logPath, description string) {
	t.Add(NewLogDeliverable(stepID, name, logPath, description))
}

// AddContract is a convenience method to add a contract deliverable
func (t *Tracker) AddContract(stepID, name, contractPath, description string) {
	t.Add(NewContractDeliverable(stepID, name, contractPath, description))
}

// GetAll returns all deliverables, sorted by creation time
func (t *Tracker) GetAll() []*Deliverable {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*Deliverable, len(t.deliverables))
	copy(result, t.deliverables)

	// Sort by creation time to maintain consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result
}

// GetByStep returns deliverables for a specific step, sorted by creation time
func (t *Tracker) GetByStep(stepID string) []*Deliverable {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*Deliverable
	for _, d := range t.deliverables {
		if d.StepID == stepID {
			result = append(result, d)
		}
	}

	// Sort by creation time to maintain consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result
}

// GetByType returns deliverables of a specific type
func (t *Tracker) GetByType(deliverableType DeliverableType) []*Deliverable {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*Deliverable
	for _, d := range t.deliverables {
		if d.Type == deliverableType {
			result = append(result, d)
		}
	}
	return result
}

// Count returns the total number of deliverables
func (t *Tracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.deliverables)
}

// FormatSummary returns a formatted summary of all deliverables
func (t *Tracker) FormatSummary() string {
	deliverables := t.GetAll()
	if len(deliverables) == 0 {
		return ""
	}

	// Sort by creation time
	sort.Slice(deliverables, func(i, j int) bool {
		return deliverables[i].CreatedAt.Before(deliverables[j].CreatedAt)
	})

	var lines []string

	// Use appropriate prefix based on nerd font availability
	prefix := "Deliverables"
	if hasNerdFont() {
		prefix = "📦 Deliverables"
	}

	lines = append(lines, fmt.Sprintf("%s (%d):", prefix, len(deliverables)))

	// Clean list format with consistent spacing
	for _, deliverable := range deliverables {
		lines = append(lines, fmt.Sprintf("   %s", deliverable.String()))
	}

	return strings.Join(lines, "\n")
}

// FormatByStep returns deliverables grouped by step in tree format
func (t *Tracker) FormatByStep() map[string][]string {
	deliverables := t.GetAll()
	result := make(map[string][]string)

	for _, d := range deliverables {
		if _, exists := result[d.StepID]; !exists {
			result[d.StepID] = []string{}
		}
		result[d.StepID] = append(result[d.StepID], d.String())
	}

	return result
}

// GetLatestForStep returns the most recent deliverable for a step
func (t *Tracker) GetLatestForStep(stepID string) *Deliverable {
	stepDeliverables := t.GetByStep(stepID)
	if len(stepDeliverables) == 0 {
		return nil
	}

	// Return most recent
	latest := stepDeliverables[0]
	for _, d := range stepDeliverables {
		if d.CreatedAt.After(latest.CreatedAt) {
			latest = d
		}
	}
	return latest
}

// AddWorkspaceFiles automatically adds common workspace files as deliverables
func (t *Tracker) AddWorkspaceFiles(stepID, workspacePath string) {
	// Track files we've already added to prevent overlapping patterns
	addedFiles := make(map[string]bool)

	// Common files to check for (ordered by specificity - most specific first)
	commonFiles := []struct {
		pattern     string
		name        string
		description string
	}{
		{"output.*", "Output File", "Generated output file"},
		{"result.*", "Result File", "Processing result"},
		{"*.log", "Execution Log", "Step execution log"},
		{"*.json", "Output Data", "JSON output file"},
		{"*.yaml", "Configuration", "YAML configuration file"},
		{"*.yml", "Configuration", "YAML configuration file"},
		{"*.md", "Documentation", "Markdown documentation"},
		{"*.txt", "Text Output", "Text output file"},
	}

	for _, file := range commonFiles {
		pattern := filepath.Join(workspacePath, file.pattern)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			// Get absolute path
			absPath, err := filepath.Abs(match)
			if err != nil {
				absPath = match
			}

			// Skip if we've already added this file
			if addedFiles[absPath] {
				continue
			}

			t.AddFile(stepID, file.name, absPath, file.description)
			addedFiles[absPath] = true
		}
	}
}
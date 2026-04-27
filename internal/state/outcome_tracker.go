package state

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// OutcomeWriter is the narrow surface OutcomeTracker uses to persist outcomes
// alongside its in-memory cache. The full StateStore satisfies it, and tests
// can pass nil to skip persistence.
type OutcomeWriter interface {
	RecordOutcome(runID, stepID, outcomeType, label, value, description string, metadata map[string]any) error
}

// OutcomeTracker is the single in-memory + persistent tracker for pipeline
// outcomes (PR URLs, issue URLs, files, branches, deployments, etc.) produced
// during a run. Each Add call writes through to the optional Store so the
// outcome survives worktree cleanup, while the cache provides fast queries
// during execution.
type OutcomeTracker struct {
	mu              sync.RWMutex
	outcomes        []*OutcomeRecord
	pipelineID      string
	store           OutcomeWriter
	outcomeWarnings []string
}

// NewOutcomeTracker returns a tracker bound to the given pipeline run ID. If
// store is non-nil, every Add also persists the outcome via RecordOutcome.
func NewOutcomeTracker(pipelineID string, store OutcomeWriter) *OutcomeTracker {
	return &OutcomeTracker{
		outcomes:   make([]*OutcomeRecord, 0),
		pipelineID: pipelineID,
		store:      store,
	}
}

// SetPipelineID rebinds the tracker to a different run ID. Used when the
// executor materialises a run lazily after the tracker is created.
func (t *OutcomeTracker) SetPipelineID(pipelineID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pipelineID = pipelineID
}

// SetStore attaches a persistence target after construction.
func (t *OutcomeTracker) SetStore(store OutcomeWriter) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.store = store
}

// Add records a new outcome, deduplicating on (Value, StepID). On success it
// also writes through to the configured Store; persistence errors are recorded
// as outcome warnings but never block the in-memory add.
func (t *OutcomeTracker) Add(record *OutcomeRecord) {
	if record == nil {
		return
	}
	t.mu.Lock()
	for _, existing := range t.outcomes {
		if existing.Value == record.Value && existing.StepID == record.StepID {
			t.mu.Unlock()
			return
		}
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	t.outcomes = append(t.outcomes, record)
	store := t.store
	pipelineID := t.pipelineID
	t.mu.Unlock()

	if store != nil && pipelineID != "" {
		if err := store.RecordOutcome(pipelineID, record.StepID, string(record.Type), record.Label, record.Value, record.Description, record.Metadata); err != nil {
			t.AddOutcomeWarning(fmt.Sprintf("[%s] failed to persist outcome %s: %v", record.StepID, record.Label, err))
		}
	}
}

// AddFile records a generic file outcome.
func (t *OutcomeTracker) AddFile(stepID, name, path, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeFile, Label: name, Value: path, Description: description, StepID: stepID})
}

// AddURL records a generic URL outcome.
func (t *OutcomeTracker) AddURL(stepID, name, url, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeURL, Label: name, Value: url, Description: description, StepID: stepID})
}

// AddPR records a pull-request outcome.
func (t *OutcomeTracker) AddPR(stepID, name, prURL, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypePR, Label: name, Value: prURL, Description: description, StepID: stepID})
}

// AddDeployment records a deployment URL outcome.
func (t *OutcomeTracker) AddDeployment(stepID, name, deployURL, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeDeployment, Label: name, Value: deployURL, Description: description, StepID: stepID})
}

// AddLog records a log file outcome.
func (t *OutcomeTracker) AddLog(stepID, name, logPath, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeLog, Label: name, Value: logPath, Description: description, StepID: stepID})
}

// AddContract records a contract artifact outcome.
func (t *OutcomeTracker) AddContract(stepID, name, contractPath, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeContract, Label: name, Value: contractPath, Description: description, StepID: stepID})
}

// AddArtifact records a generic pipeline-produced artifact.
func (t *OutcomeTracker) AddArtifact(stepID, name, artifactPath, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeArtifact, Label: name, Value: artifactPath, Description: description, StepID: stepID})
}

// AddBranch records a feature-branch outcome with an unpushed default flag.
func (t *OutcomeTracker) AddBranch(stepID, branchName, worktreePath, description string) {
	t.Add(&OutcomeRecord{
		Type:        OutcomeTypeBranch,
		Label:       branchName,
		Value:       worktreePath,
		Description: description,
		StepID:      stepID,
		Metadata:    map[string]any{"pushed": false},
	})
}

// AddIssue records a tracked-issue outcome.
func (t *OutcomeTracker) AddIssue(stepID, name, issueURL, description string) {
	t.Add(&OutcomeRecord{Type: OutcomeTypeIssue, Label: name, Value: issueURL, Description: description, StepID: stepID})
}

// GetAll returns a copy of every recorded outcome, sorted by creation time.
func (t *OutcomeTracker) GetAll() []*OutcomeRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*OutcomeRecord, len(t.outcomes))
	copy(result, t.outcomes)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

// GetByStep returns outcomes recorded against stepID, sorted by creation time.
func (t *OutcomeTracker) GetByStep(stepID string) []*OutcomeRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var result []*OutcomeRecord
	for _, r := range t.outcomes {
		if r.StepID == stepID {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

// GetByType returns every outcome of the given type.
func (t *OutcomeTracker) GetByType(outcomeType OutcomeType) []*OutcomeRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var result []*OutcomeRecord
	for _, r := range t.outcomes {
		if r.Type == outcomeType {
			result = append(result, r)
		}
	}
	return result
}

// Count returns the total number of recorded outcomes.
func (t *OutcomeTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.outcomes)
}

// FormatSummary returns the human-readable end-of-run summary block.
func (t *OutcomeTracker) FormatSummary() string {
	outcomes := t.GetAll()
	if len(outcomes) == 0 {
		return ""
	}
	prefix := "Artifacts"
	if hasNerdFont() {
		prefix = "📦 Artifacts"
	}
	lines := make([]string, 0, len(outcomes)+1)
	lines = append(lines, fmt.Sprintf("%s (%d):", prefix, len(outcomes)))
	for _, r := range outcomes {
		lines = append(lines, fmt.Sprintf("   %s", r.String()))
	}
	return strings.Join(lines, "\n")
}

// FormatByStep returns the per-step rendering used by progress displays.
func (t *OutcomeTracker) FormatByStep() map[string][]string {
	outcomes := t.GetAll()
	result := make(map[string][]string)
	for _, r := range outcomes {
		result[r.StepID] = append(result[r.StepID], r.String())
	}
	return result
}

// GetLatestForStep returns the most recently created outcome for stepID, or nil.
func (t *OutcomeTracker) GetLatestForStep(stepID string) *OutcomeRecord {
	stepOutcomes := t.GetByStep(stepID)
	if len(stepOutcomes) == 0 {
		return nil
	}
	latest := stepOutcomes[0]
	for _, r := range stepOutcomes {
		if r.CreatedAt.After(latest.CreatedAt) {
			latest = r
		}
	}
	return latest
}

// AddWorkspaceFiles scans workspacePath for common output patterns and records
// each match as a file outcome under stepID.
func (t *OutcomeTracker) AddWorkspaceFiles(stepID, workspacePath string) {
	added := make(map[string]bool)
	patterns := []struct {
		pattern, name, description string
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
	for _, p := range patterns {
		matches, err := filepath.Glob(filepath.Join(workspacePath, p.pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			absPath, err := filepath.Abs(match)
			if err != nil {
				absPath = match
			}
			if added[absPath] {
				continue
			}
			t.AddFile(stepID, p.name, absPath, p.description)
			added[absPath] = true
		}
	}
}

// AddOutcomeWarning records a non-fatal extraction warning shown in the summary.
func (t *OutcomeTracker) AddOutcomeWarning(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.outcomeWarnings = append(t.outcomeWarnings, msg)
}

// OutcomeWarnings returns a copy of every recorded warning.
func (t *OutcomeTracker) OutcomeWarnings() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]string, len(t.outcomeWarnings))
	copy(result, t.outcomeWarnings)
	return result
}

// UpdateMetadata mutates a single metadata key on the first matching outcome.
// Thread-safe; no-op if no outcome matches the (type, label) pair.
func (t *OutcomeTracker) UpdateMetadata(outcomeType OutcomeType, label string, key string, value any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, r := range t.outcomes {
		if r.Type == outcomeType && r.Label == label {
			if r.Metadata == nil {
				r.Metadata = make(map[string]any)
			}
			r.Metadata[key] = value
			return
		}
	}
}

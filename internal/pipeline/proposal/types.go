// Package proposal implements the pipeline proposal engine that consumes
// codebase health analysis artifacts and the available pipeline catalog
// to produce structured, prioritized pipeline execution proposals.
package proposal

import (
	"encoding/json"
	"fmt"
	"time"
)

// ForgeType represents a supported source code forge platform.
type ForgeType string

const (
	ForgeGitHub  ForgeType = "github"
	ForgeGitLab  ForgeType = "gitlab"
	ForgeGitea   ForgeType = "gitea"
	ForgeBitBkt  ForgeType = "bitbucket"
	ForgeUnknown ForgeType = "unknown"
)

// ValidForgeTypes returns all recognized forge types (excluding Unknown).
func ValidForgeTypes() []ForgeType {
	return []ForgeType{ForgeGitHub, ForgeGitLab, ForgeGitea, ForgeBitBkt}
}

// HealthSignal represents a single finding from the codebase health analysis.
type HealthSignal struct {
	Category string  `json:"category"`         // e.g., "test_failures", "dead_code", "doc_issues", "security"
	Severity string  `json:"severity"`         // "critical", "high", "medium", "low", "info"
	Count    int     `json:"count"`            // Number of occurrences
	Score    float64 `json:"score"`            // Normalized severity score (0.0-1.0)
	Detail   string  `json:"detail,omitempty"` // Human-readable detail
}

// HealthArtifact represents the codebase health analysis output from #207.
// This is a provisional schema that will be adapted when #207 is finalized.
type HealthArtifact struct {
	Version   string         `json:"version"`
	Timestamp time.Time      `json:"timestamp"`
	Signals   []HealthSignal `json:"signals"`
	ForgeType ForgeType      `json:"forge_type,omitempty"` // Detected forge, if available from #206
	Summary   string         `json:"summary,omitempty"`
}

// SignalsByCategory returns all signals matching the given category.
func (h *HealthArtifact) SignalsByCategory(category string) []HealthSignal {
	var result []HealthSignal
	for _, s := range h.Signals {
		if s.Category == category {
			result = append(result, s)
		}
	}
	return result
}

// MaxSeverityScore returns the highest severity score across all signals,
// or 0.0 if there are no signals.
func (h *HealthArtifact) MaxSeverityScore() float64 {
	var max float64
	for _, s := range h.Signals {
		if s.Score > max {
			max = s.Score
		}
	}
	return max
}

// ProposalItem represents a single pipeline recommendation within a proposal.
type ProposalItem struct {
	Pipeline      string   `json:"pipeline"`             // Pipeline name (e.g., "gh-implement")
	Rationale     string   `json:"rationale"`            // Why this pipeline is relevant
	Priority      int      `json:"priority"`             // 1 = highest priority
	Score         float64  `json:"score"`                // Relevance score (0.0-1.0)
	ParallelGroup int      `json:"parallel_group"`       // Items in the same group can run concurrently
	DependsOn     []string `json:"depends_on,omitempty"` // Pipeline names this item depends on
	Category      string   `json:"category,omitempty"`   // Pipeline category from metadata
}

// Validate checks that a ProposalItem has all required fields populated.
func (p *ProposalItem) Validate() error {
	if p.Pipeline == "" {
		return fmt.Errorf("proposal item: pipeline name is required")
	}
	if p.Rationale == "" {
		return fmt.Errorf("proposal item %q: rationale is required", p.Pipeline)
	}
	if p.Priority < 1 {
		return fmt.Errorf("proposal item %q: priority must be >= 1, got %d", p.Pipeline, p.Priority)
	}
	if p.Score < 0 || p.Score > 1 {
		return fmt.Errorf("proposal item %q: score must be between 0.0 and 1.0, got %f", p.Pipeline, p.Score)
	}
	if p.ParallelGroup < 0 {
		return fmt.Errorf("proposal item %q: parallel_group must be >= 0, got %d", p.Pipeline, p.ParallelGroup)
	}
	return nil
}

// Proposal is the top-level output artifact produced by the proposal engine.
type Proposal struct {
	ForgeType     ForgeType      `json:"forge_type"`               // Detected or configured forge type
	Proposals     []ProposalItem `json:"proposals"`                // Ordered list of pipeline recommendations
	Timestamp     time.Time      `json:"timestamp"`                // When the proposal was generated
	HealthSummary string         `json:"health_summary,omitempty"` // Summary of the health analysis input
}

// Validate checks that the proposal has all required fields and that
// each proposal item is valid.
func (p *Proposal) Validate() error {
	if p.ForgeType == "" {
		return fmt.Errorf("proposal: forge_type is required")
	}
	if p.Timestamp.IsZero() {
		return fmt.Errorf("proposal: timestamp is required")
	}
	for i := range p.Proposals {
		if err := p.Proposals[i].Validate(); err != nil {
			return fmt.Errorf("proposal[%d]: %w", i, err)
		}
	}
	return nil
}

// MarshalJSON produces the JSON representation of the proposal.
func (p *Proposal) MarshalJSON() ([]byte, error) {
	type Alias Proposal
	return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(p)})
}

// ParallelGroups returns the proposal items organized by parallel group.
// Items within the same group can be executed concurrently.
func (p *Proposal) ParallelGroups() map[int][]ProposalItem {
	groups := make(map[int][]ProposalItem)
	for _, item := range p.Proposals {
		groups[item.ParallelGroup] = append(groups[item.ParallelGroup], item)
	}
	return groups
}

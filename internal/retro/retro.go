// Package retro provides types for run retrospectives — structured
// post-execution analysis of pipeline runs that captures both
// quantitative metrics and qualitative narrative assessments.
package retro

import "time"

// Smoothness constants describe the overall execution quality of a run.
const (
	SmoothnessEffortless = "effortless"
	SmoothnessSmooth     = "smooth"
	SmoothnessBumpy      = "bumpy"
	SmoothnessStruggled  = "struggled"
	SmoothnessFailed     = "failed"
)

// Friction point type constants classify what went wrong during a step.
const (
	FrictionRetry           = "retry"
	FrictionTimeout         = "timeout"
	FrictionWrongApproach   = "wrong_approach"
	FrictionToolFailure     = "tool_failure"
	FrictionAmbiguity       = "ambiguity"
	FrictionContractFailure = "contract_failure"
)

// Learning category constants classify insights gained during a run.
const (
	LearningRepo     = "repo"
	LearningCode     = "code"
	LearningWorkflow = "workflow"
	LearningTool     = "tool"
)

// Open item type constants classify follow-up work identified during a run.
const (
	OpenItemTechDebt      = "tech_debt"
	OpenItemFollowUp      = "follow_up"
	OpenItemInvestigation = "investigation"
	OpenItemTestGap       = "test_gap"
)

// Retrospective is the top-level structure for a run retrospective,
// combining quantitative execution metrics with an optional qualitative
// narrative assessment.
type Retrospective struct {
	RunID        string           `json:"run_id"`
	Pipeline     string           `json:"pipeline"`
	Timestamp    time.Time        `json:"timestamp"`
	Quantitative QuantitativeData `json:"quantitative"`
	Narrative    *NarrativeData   `json:"narrative,omitempty"`
}

// QuantitativeData holds the measurable execution metrics for a run.
type QuantitativeData struct {
	TotalDurationMs int64         `json:"total_duration_ms"`
	TotalSteps      int           `json:"total_steps"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	TotalRetries    int           `json:"total_retries"`
	Steps           []StepMetrics `json:"steps"`
}

// StepMetrics captures execution metrics for a single pipeline step.
type StepMetrics struct {
	Name       string `json:"name"`
	DurationMs int64  `json:"duration_ms"`
	Retries    int    `json:"retries"`
	Status     string `json:"status"`
	Adapter    string `json:"adapter"`
	Model      string `json:"model"`
	ExitCode   int    `json:"exit_code"`
	TokensUsed int    `json:"tokens_used"`
}

// NarrativeData holds the qualitative assessment of a run, including
// friction points encountered, lessons learned, and open items for
// follow-up.
type NarrativeData struct {
	Smoothness     string          `json:"smoothness"`
	Intent         string          `json:"intent"`
	Outcome        string          `json:"outcome"`
	FrictionPoints []FrictionPoint `json:"friction_points"`
	Learnings      []Learning      `json:"learnings"`
	OpenItems      []OpenItem      `json:"open_items"`
}

// FrictionPoint describes a specific point of friction encountered
// during pipeline execution.
type FrictionPoint struct {
	Type   string `json:"type"`
	Step   string `json:"step"`
	Detail string `json:"detail"`
}

// Learning captures an insight or lesson learned during a run.
type Learning struct {
	Category string `json:"category"`
	Detail   string `json:"detail"`
}

// OpenItem represents a follow-up item identified during a run that
// requires future attention.
type OpenItem struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

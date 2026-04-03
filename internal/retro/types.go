package retro

import "time"

// Smoothness represents the 5-point smoothness rating scale.
type Smoothness string

const (
	SmoothnessEffortless Smoothness = "effortless"
	SmoothnessSmooth     Smoothness = "smooth"
	SmoothnessBumpy      Smoothness = "bumpy"
	SmoothnessStruggled  Smoothness = "struggled"
	SmoothnessFailed     Smoothness = "failed"
)

// FrictionType categorizes friction points encountered during a run.
type FrictionType string

const (
	FrictionRetry           FrictionType = "retry"
	FrictionTimeout         FrictionType = "timeout"
	FrictionWrongApproach   FrictionType = "wrong_approach"
	FrictionToolFailure     FrictionType = "tool_failure"
	FrictionAmbiguity       FrictionType = "ambiguity"
	FrictionContractFailure FrictionType = "contract_failure"
	FrictionReviewRework    FrictionType = "review_rework"
)

// LearningCategory categorizes learnings captured during a run.
type LearningCategory string

const (
	LearningRepo     LearningCategory = "repo"
	LearningCode     LearningCategory = "code"
	LearningWorkflow LearningCategory = "workflow"
	LearningTool     LearningCategory = "tool"
)

// OpenItemType categorizes open items flagged during a run.
type OpenItemType string

const (
	OpenItemTechDebt      OpenItemType = "tech_debt"
	OpenItemFollowUp      OpenItemType = "follow_up"
	OpenItemInvestigation OpenItemType = "investigation"
	OpenItemTestGap       OpenItemType = "test_gap"
)

// Retrospective is the complete retrospective for a pipeline run.
type Retrospective struct {
	RunID        string            `json:"run_id"`
	Pipeline     string            `json:"pipeline"`
	Timestamp    time.Time         `json:"timestamp"`
	Quantitative *QuantitativeData `json:"quantitative"`
	Narrative    *Narrative        `json:"narrative,omitempty"`
}

// QuantitativeData holds the metrics collected from a pipeline run.
type QuantitativeData struct {
	TotalDurationMs int64         `json:"total_duration_ms"`
	TotalSteps      int           `json:"total_steps"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	TotalRetries    int           `json:"total_retries"`
	TotalTokens     int           `json:"total_tokens"`
	Steps           []StepMetrics `json:"steps"`
}

// StepMetrics holds per-step quantitative data.
type StepMetrics struct {
	Name         string `json:"name"`
	DurationMs   int64  `json:"duration_ms"`
	Retries      int    `json:"retries"`
	Status       string `json:"status"`
	Adapter      string `json:"adapter,omitempty"`
	Model        string `json:"model,omitempty"`
	ExitCode     int    `json:"exit_code"`
	FilesChanged int    `json:"files_changed"`
	TokensUsed   int    `json:"tokens_used"`
}

// Narrative holds the LLM-generated narrative analysis.
type Narrative struct {
	Smoothness      Smoothness      `json:"smoothness"`
	Intent          string          `json:"intent"`
	Outcome         string          `json:"outcome"`
	FrictionPoints  []FrictionPoint `json:"friction_points,omitempty"`
	Learnings       []Learning      `json:"learnings,omitempty"`
	OpenItems       []OpenItem      `json:"open_items,omitempty"`
	Recommendations []string        `json:"recommendations,omitempty"`
}

// FrictionPoint identifies a specific friction encountered during a run.
type FrictionPoint struct {
	Type   FrictionType `json:"type"`
	Step   string       `json:"step"`
	Detail string       `json:"detail"`
}

// Learning captures something learned during the run.
type Learning struct {
	Category LearningCategory `json:"category"`
	Detail   string           `json:"detail"`
}

// OpenItem flags an issue for follow-up.
type OpenItem struct {
	Type   OpenItemType `json:"type"`
	Detail string       `json:"detail"`
}

// ValidSmoothness returns true if the smoothness value is valid.
func ValidSmoothness(s Smoothness) bool {
	switch s {
	case SmoothnessEffortless, SmoothnessSmooth, SmoothnessBumpy, SmoothnessStruggled, SmoothnessFailed:
		return true
	}
	return false
}

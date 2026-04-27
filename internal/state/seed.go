package state

import (
	"fmt"
	"time"
)

// SeedRunOptions configures a deterministic run insert for fixtures.
// Empty CompletedAt leaves the column NULL.
type SeedRunOptions struct {
	RunID        string
	PipelineName string
	Status       string
	Input        string
	CurrentStep  string
	TotalTokens  int
	StartedAt    time.Time
	CompletedAt  *time.Time
	ErrorMessage string
}

// SeedRun inserts a pipeline_run row with caller-specified fields.
// Intended for test fixtures where deterministic run IDs and timestamps are
// required (production code must use CreateRun).
func SeedRun(store StateStore, opts SeedRunOptions) error {
	s, ok := store.(*stateStore)
	if !ok {
		return fmt.Errorf("SeedRun requires a *stateStore implementation")
	}

	var completedUnix any
	if opts.CompletedAt != nil {
		completedUnix = opts.CompletedAt.Unix()
	}

	var step any
	if opts.CurrentStep != "" {
		step = opts.CurrentStep
	}

	var errMsg any
	if opts.ErrorMessage != "" {
		errMsg = opts.ErrorMessage
	}

	_, err := s.db.Exec(
		`INSERT INTO pipeline_run (run_id, pipeline_name, status, input, current_step, total_tokens, started_at, completed_at, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		opts.RunID, opts.PipelineName, opts.Status, opts.Input, step, opts.TotalTokens,
		opts.StartedAt.Unix(), completedUnix, errMsg,
	)
	if err != nil {
		return fmt.Errorf("seed run: %w", err)
	}
	return nil
}

// SeedEventOptions configures a deterministic event_log insert for fixtures.
type SeedEventOptions struct {
	RunID      string
	Timestamp  time.Time
	StepID     string
	State      string
	Persona    string
	Message    string
	TokensUsed int
	DurationMs int64
	Model      string
	Adapter    string
}

// SeedEvent inserts an event_log row with caller-specified fields. Intended
// for test fixtures requiring back-dated timestamps.
func SeedEvent(store StateStore, opts SeedEventOptions) error {
	s, ok := store.(*stateStore)
	if !ok {
		return fmt.Errorf("SeedEvent requires a *stateStore implementation")
	}

	_, err := s.db.Exec(
		`INSERT INTO event_log (run_id, timestamp, step_id, state, persona, message, tokens_used, duration_ms, model, configured_model, adapter)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		opts.RunID, opts.Timestamp.Unix(), opts.StepID, opts.State, opts.Persona, opts.Message,
		opts.TokensUsed, opts.DurationMs, opts.Model, "", opts.Adapter,
	)
	if err != nil {
		return fmt.Errorf("seed event: %w", err)
	}
	return nil
}

// SeedDecisionOptions configures a deterministic decision_log insert for fixtures.
type SeedDecisionOptions struct {
	RunID     string
	StepID    string
	Timestamp time.Time
	Category  string
	Decision  string
	Rationale string
	Context   string
}

// SeedDecision inserts a decision_log row with caller-specified fields.
func SeedDecision(store StateStore, opts SeedDecisionOptions) error {
	s, ok := store.(*stateStore)
	if !ok {
		return fmt.Errorf("SeedDecision requires a *stateStore implementation")
	}

	ctxJSON := opts.Context
	if ctxJSON == "" {
		ctxJSON = "{}"
	}

	_, err := s.db.Exec(
		`INSERT INTO decision_log (run_id, step_id, timestamp, category, decision, rationale, context_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		opts.RunID, opts.StepID, opts.Timestamp.Unix(), opts.Category, opts.Decision, opts.Rationale, ctxJSON,
	)
	if err != nil {
		return fmt.Errorf("seed decision: %w", err)
	}
	return nil
}

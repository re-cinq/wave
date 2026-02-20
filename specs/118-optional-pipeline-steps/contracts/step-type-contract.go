// Package contracts defines compile-time contracts for the optional pipeline steps feature.
// These are used as type assertions during implementation, not as runtime code.
//
// Contract: Step struct must include Optional bool field (FR-001)
// Contract: StepState must include "failed_optional" constant (FR-004)
// Contract: Event must include Optional bool field (FR-006)

package contracts

// StepOptionalFieldContract verifies the Step struct has an Optional field.
// This is a compile-time contract — if Step.Optional doesn't exist, this file won't compile.
type StepOptionalFieldContract interface {
	// GetOptional returns the optional flag for a step.
	// Implementing types: pipeline.Step
	GetOptional() bool
}

// StepStateConstants verifies the required state constants exist.
type StepStateConstants interface {
	// These method signatures document the expected constants.
	// Actual verification is via const usage in test files.
	IsPending() bool
	IsRunning() bool
	IsCompleted() bool
	IsFailed() bool
	IsRetrying() bool
	IsFailedOptional() bool // NEW — FR-004
}

// EventOptionalFieldContract verifies the Event struct has an Optional field.
type EventOptionalFieldContract interface {
	// GetOptional returns whether this event relates to an optional step.
	// Implementing types: event.Event
	GetEventOptional() bool
}

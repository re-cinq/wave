// Package contracts defines the event state constants for the skill
// dependency installation feature.
//
// NOTE: This file is a design artifact, not production code.

package contracts

// --- Event State Contract ---

// StatePreflight is the event state for preflight dependency checks.
// Added to internal/event/emitter.go alongside existing state constants.
//
// Events emitted with this state:
//   - One event per tool check (checking, found/not found)
//   - One event per skill check (checking, installed, installing, install_failed, init_failed)
//
// Event fields used:
//   - State: StatePreflight ("preflight")
//   - Message: human-readable status (e.g., "skill \"speckit\" installed")
//   - PipelineID: set by executor
//   - Timestamp: set by emitter
const StatePreflight = "preflight"

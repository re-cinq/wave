package contract

import (
	"fmt"
	"strings"
)

// EventPattern defines an expected event in the step's event log.
// State must match exactly; Contains is a substring match on the event message.
type EventPattern struct {
	State    string `json:"state"`              // Required: event state to match (e.g. "step_failed")
	Contains string `json:"contains,omitempty"` // Optional: substring that must appear in the event message
}

// EventRecord is a minimal event interface so the contract package
// doesn't depend on the state package directly.
type EventRecord struct {
	State   string
	StepID  string
	Message string
}

// ValidateEventContains checks that every EventPattern in cfg.Events matches
// at least one event in the provided records for the given step.
// Returns nil if all patterns matched, or an error listing the missing ones.
func ValidateEventContains(cfg ContractConfig, stepID string, events []EventRecord) error {
	if len(cfg.Events) == 0 {
		return nil
	}

	var missing []string
	for _, pattern := range cfg.Events {
		found := false
		for _, ev := range events {
			if ev.StepID != stepID {
				continue
			}
			if ev.State != pattern.State {
				continue
			}
			if pattern.Contains != "" && !strings.Contains(ev.Message, pattern.Contains) {
				continue
			}
			found = true
			break
		}
		if !found {
			desc := pattern.State
			if pattern.Contains != "" {
				desc += " containing " + fmt.Sprintf("%q", pattern.Contains)
			}
			missing = append(missing, desc)
		}
	}

	if len(missing) > 0 {
		return &ValidationError{
			ContractType: "event_contains",
			Message:      fmt.Sprintf("missing %d expected event(s)", len(missing)),
			Details:      missing,
			Retryable:    false,
		}
	}
	return nil
}

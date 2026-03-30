package continuous

import "time"

// WorkItem is a single input to a pipeline iteration.
type WorkItem struct {
	ID    string // Unique identifier (e.g., "42" for GitHub issue #42)
	Input string // Full input string passed to pipeline execution
}

// IterationResult records the outcome of one iteration.
type IterationResult struct {
	Iteration int
	WorkItem  *WorkItem
	Duration  time.Duration
}

// FailurePolicy controls loop behavior on iteration failure.
type FailurePolicy string

const (
	FailurePolicyHalt FailurePolicy = "halt"
	FailurePolicySkip FailurePolicy = "skip"
)

// ParseFailurePolicy converts a string to a FailurePolicy, defaulting to halt.
func ParseFailurePolicy(s string) FailurePolicy {
	switch s {
	case "skip":
		return FailurePolicySkip
	default:
		return FailurePolicyHalt
	}
}

// SourceConfig holds the parsed source URI configuration.
type SourceConfig struct {
	Provider string
	Params   map[string]string
}

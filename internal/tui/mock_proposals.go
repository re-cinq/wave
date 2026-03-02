package tui

// MockProposalProvider returns sample proposals for development and manual testing.
// It provides a realistic multi-pipeline proposal set that exercises all TUI features:
// sequential dependencies, parallel groups, and various priority levels.
func MockProposalProvider() []Proposal {
	return []Proposal{
		{
			Pipeline: "lint",
			Reason:   "Code quality checks should run first",
			Priority: 1,
			Input:    "./...",
		},
		{
			Pipeline:      "unit-test",
			Reason:        "Unit tests validate core logic",
			Dependencies:  []string{"lint"},
			ParallelGroup: "testing",
			Priority:      2,
			Input:         "./...",
		},
		{
			Pipeline:      "integration-test",
			Reason:        "Integration tests validate API contracts",
			Dependencies:  []string{"lint"},
			ParallelGroup: "testing",
			Priority:      2,
			Input:         "./...",
		},
		{
			Pipeline:     "build",
			Reason:       "Build after all tests pass",
			Dependencies: []string{"unit-test", "integration-test"},
			Priority:     3,
		},
		{
			Pipeline:     "deploy-staging",
			Reason:       "Deploy to staging for review",
			Dependencies: []string{"build"},
			Priority:     4,
			Input:        "staging",
		},
	}
}

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/state"
	"github.com/spf13/cobra"
)

// PostmortemOptions holds options for the postmortem command.
type PostmortemOptions struct {
	RunID string
	JSON  bool
}

// failurePattern identifies a recognized failure pattern.
type failurePattern struct {
	name        string
	diagnosis   string
	suggestions []string
}

// postmortemReport is the structured output of a post-mortem analysis.
type postmortemReport struct {
	RunID           string   `json:"run_id"`
	Pipeline        string   `json:"pipeline"`
	FailedStep      string   `json:"failed_step"`
	Attempt         int      `json:"attempt"`
	TotalDuration   string   `json:"total_duration"`
	StepDuration    string   `json:"step_duration"`
	HasStepDuration bool     `json:"-"`
	Error           string   `json:"error"`
	Diagnosis       string   `json:"diagnosis"`
	Suggestions     []string `json:"suggestions"`
	Artifacts       []string `json:"artifacts"`
	ResumeCommand   string   `json:"resume_command"`
}

// NewPostmortemCmd creates the postmortem command.
func NewPostmortemCmd() *cobra.Command {
	var opts PostmortemOptions

	cmd := &cobra.Command{
		Use:   "postmortem <run-id>",
		Short: "Analyse a failed pipeline run and suggest recovery steps",
		Long: `Produce a structured post-mortem report for a failed pipeline run.

The report includes:
  - Which step failed and how many times it was attempted
  - Total run duration and failed-step duration
  - The error message logged at failure time
  - Diagnosis of the failure pattern (token exhaustion, contract
    validation failure, timeout, missing artifact, permission denied, etc.)
  - Concrete recovery suggestions
  - A list of artifacts produced by steps that completed successfully
  - The exact 'wave resume' command to re-start from the failed step

Use --json for machine-readable output suitable for scripting.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RunID = args[0]
			return runPostmortem(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runPostmortem(opts PostmortemOptions) error {
	dbPath := ".wave/state.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return NewCLIError(CodeStateDBError, "state database not found", "Run 'wave run' to create the state database")
	}

	store, err := state.NewStateStore(dbPath)
	if err != nil {
		return NewCLIError(CodeStateDBError, fmt.Sprintf("failed to open state database: %s", err), "Check .wave/state.db file permissions").WithCause(err)
	}
	defer store.Close()

	run, err := store.GetRun(opts.RunID)
	if err != nil {
		return NewCLIError(CodeRunNotFound, fmt.Sprintf("run not found: %s", err), "Use 'wave status --all' to list available runs").WithCause(err)
	}

	// Only operate on failed runs.
	switch run.Status {
	case "completed":
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("run %s completed successfully -- no post-mortem needed", opts.RunID), "Post-mortem analysis is only for failed or cancelled runs")
	case "running":
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("run %s is still running -- wait for it to finish before analysing", opts.RunID), "Wait for the run to complete or use 'wave cancel' first")
	case "pending":
		return NewCLIError(CodeInvalidArgs, fmt.Sprintf("run %s has not started yet", opts.RunID), "The run has not started -- check 'wave status' for details")
	}
	// "failed", "cancelled" — proceed.

	// Fetch step states to find the failed step.
	stepStates, err := store.GetStepStates(opts.RunID)
	if err != nil {
		return NewCLIError(CodeInternalError, fmt.Sprintf("failed to get step states: %s", err), "State database query failed").WithCause(err)
	}

	// Identify the failed step.
	var failedStep *state.StepStateRecord
	for i := range stepStates {
		if stepStates[i].State == state.StateFailed {
			failedStep = &stepStates[i]
			break
		}
	}

	// Fall back to the error message on the run record itself when no individual
	// step is marked failed (e.g. preflight or orchestration failures).
	failedStepID := ""
	errorMessage := run.ErrorMessage
	var stepDuration time.Duration
	retryCount := 0

	if failedStep != nil {
		failedStepID = failedStep.StepID
		if failedStep.ErrorMessage != "" {
			errorMessage = failedStep.ErrorMessage
		}
		retryCount = failedStep.RetryCount
		if failedStep.StartedAt != nil && failedStep.CompletedAt != nil {
			stepDuration = failedStep.CompletedAt.Sub(*failedStep.StartedAt)
		}
	}

	// Compute total run duration.
	var totalDuration time.Duration
	if run.CompletedAt != nil {
		totalDuration = run.CompletedAt.Sub(run.StartedAt)
	} else if run.CancelledAt != nil {
		totalDuration = run.CancelledAt.Sub(run.StartedAt)
	} else {
		totalDuration = time.Since(run.StartedAt)
	}

	// Scan event log for additional clues when the step error message is sparse.
	events, _ := store.GetEvents(opts.RunID, state.EventQueryOptions{
		StepID:     failedStepID,
		ErrorsOnly: false,
		Limit:      500,
	})
	contractErrors := extractContractErrors(events)

	// Determine failure pattern.
	pattern := classifyFailure(errorMessage, contractErrors)

	// Collect artifacts from successfully completed steps.
	artifacts := collectPostmortemArtifacts(store, opts.RunID, stepStates)

	// Build resume command.
	resumeCmd := buildPostmortemResumeCmd(opts.RunID, failedStepID)

	// Assemble report.
	report := postmortemReport{
		RunID:           opts.RunID,
		Pipeline:        run.PipelineName,
		FailedStep:      failedStepID,
		Attempt:         retryCount + 1,
		TotalDuration:   formatElapsed(totalDuration),
		StepDuration:    formatElapsed(stepDuration),
		HasStepDuration: stepDuration > 0,
		Error:           errorMessage,
		Diagnosis:       pattern.diagnosis,
		Suggestions:     pattern.suggestions,
		Artifacts:       artifacts,
		ResumeCommand:   resumeCmd,
	}

	if opts.JSON {
		return printPostmortemJSON(report)
	}
	printPostmortemText(report)
	return nil
}

// classifyFailure inspects the error message and contract errors to identify
// a known failure pattern and return human-readable suggestions.
func classifyFailure(errMsg string, contractErrors []string) failurePattern {
	lower := strings.ToLower(errMsg)

	if containsAny(lower, "context window", "token limit", "context length", "max_tokens") {
		return failurePattern{
			name:      "token_exhaustion",
			diagnosis: "Token exhaustion — the context window was exceeded",
			suggestions: []string{
				"Split the step prompt into smaller sub-tasks",
				"Enable compaction in the step's handover config",
				"Use a model with a larger context window (e.g. claude-opus-4)",
				"Reduce inject_artifacts to pass only the most relevant outputs",
			},
		}
	}

	if len(contractErrors) > 0 || containsAny(lower, "contract", "validation failed", "schema", "json schema") {
		suggestions := []string{
			"Check the contract schema in the pipeline definition",
			"Review the step's output_artifacts for correctness",
			"Resume and skip validation: wave resume " + "—-from-step <step> --force",
		}
		if len(contractErrors) > 0 {
			suggestions = append([]string{"Contract errors: " + strings.Join(contractErrors, "; ")}, suggestions...)
		}
		return failurePattern{
			name:        "contract_validation",
			diagnosis:   "Contract validation failure — step output did not satisfy the contract",
			suggestions: suggestions,
		}
	}

	if containsAny(lower, "timeout", "deadline exceeded", "context deadline", "timed out") {
		return failurePattern{
			name:      "adapter_timeout",
			diagnosis: "Adapter timeout — the step exceeded its allowed execution time",
			suggestions: []string{
				"Increase timeout_minutes in the step definition",
				"Simplify the step prompt to reduce execution time",
				"Break the step into smaller sequential steps",
			},
		}
	}

	if containsAny(lower, "artifact") && containsAny(lower, "not found", "missing", "does not exist", "no such file") {
		return failurePattern{
			name:      "missing_artifact",
			diagnosis: "Missing artifact — a required artifact from a prior step was not found",
			suggestions: []string{
				"Verify the prior step completed and produced the expected artifact",
				"Check that inject_artifacts references the correct step and artifact name",
				"Mark the artifact optional: true in the pipeline if it is not always produced",
				"Resume from the step that should have produced the artifact",
			},
		}
	}

	if containsAny(lower, "permission denied", "permission", "denied", "forbidden", "unauthorized") {
		return failurePattern{
			name:      "permission_denied",
			diagnosis: "Permission denied — the step was blocked by a security or filesystem restriction",
			suggestions: []string{
				"Review the persona's deny/allow tool rules in wave.yaml",
				"Check that the workspace directory is writable",
				"Verify the adapter settings.json does not block a required tool",
				"Inspect the audit log: ls .wave/traces/",
			},
		}
	}

	if errMsg == "" {
		return failurePattern{
			name:      "unknown",
			diagnosis: "Unknown — no error message was recorded",
			suggestions: []string{
				"Re-run with debug output: wave run <pipeline> --debug",
				"Inspect the workspace: ls .wave/workspaces/" + "<run-id>/<step>/",
				"Check the event log: wave logs <run-id>",
			},
		}
	}

	return failurePattern{
		name:      "runtime_error",
		diagnosis: "Runtime error — the step failed with an unrecognised error",
		suggestions: []string{
			"Re-run with debug output: wave run <pipeline> --debug",
			"Inspect the workspace artifacts: ls .wave/workspaces/<run-id>/<step>/",
			"Check the full event log: wave logs <run-id>",
		},
	}
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// extractContractErrors scans log records for messages that look like
// contract validation errors and returns a deduplicated slice.
func extractContractErrors(events []state.LogRecord) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range events {
		msg := e.Message
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "contract") || strings.Contains(lower, "validation") || strings.Contains(lower, "schema") {
			if !seen[msg] {
				seen[msg] = true
				out = append(out, msg)
			}
		}
	}
	return out
}

// collectPostmortemArtifacts gathers artifact names from steps that completed successfully.
func collectPostmortemArtifacts(store state.StateStore, runID string, stepStates []state.StepStateRecord) []string {
	var artifacts []string
	seen := map[string]bool{}

	for _, ss := range stepStates {
		if ss.State != state.StateCompleted {
			continue
		}
		records, err := store.GetArtifacts(runID, ss.StepID)
		if err != nil {
			continue
		}
		for _, a := range records {
			label := fmt.Sprintf("%s/%s (%s)", ss.StepID, a.Name, a.Path)
			if !seen[label] {
				seen[label] = true
				artifacts = append(artifacts, label)
			}
		}
	}
	return artifacts
}

// buildPostmortemResumeCmd constructs the wave resume command string.
func buildPostmortemResumeCmd(runID, failedStepID string) string {
	if failedStepID == "" {
		return fmt.Sprintf("wave resume %s", runID)
	}
	return fmt.Sprintf("wave resume %s --from-step %s", runID, failedStepID)
}

// printPostmortemText writes the report to stdout in human-readable form.
func printPostmortemText(r postmortemReport) {
	fmt.Printf("Post-Mortem: %s\n", r.RunID)
	fmt.Printf("Pipeline:    %s\n", r.Pipeline)

	if r.FailedStep != "" {
		fmt.Printf("Failed Step: %s (attempt %d)\n", r.FailedStep, r.Attempt)
	} else {
		fmt.Printf("Failed Step: (orchestration/preflight failure)\n")
	}

	fmt.Printf("Duration:    %s", r.TotalDuration)
	if r.HasStepDuration {
		fmt.Printf(" (failed step: %s)", r.StepDuration)
	}
	fmt.Println()

	if r.Error != "" {
		fmt.Printf("Error:       %s\n", r.Error)
	}

	fmt.Println()
	fmt.Printf("Diagnosis: %s\n", r.Diagnosis)

	if len(r.Suggestions) > 0 {
		fmt.Println()
		fmt.Println("Recovery Suggestions:")
		for _, s := range r.Suggestions {
			fmt.Printf("  - %s\n", s)
		}
	}

	if len(r.Artifacts) > 0 {
		fmt.Println()
		fmt.Println("Artifacts Preserved:")
		for _, a := range r.Artifacts {
			fmt.Printf("  - %s\n", a)
		}
	} else {
		fmt.Println()
		fmt.Println("Artifacts Preserved: none")
	}

	fmt.Println()
	fmt.Println("Resume Command:")
	fmt.Printf("  %s\n", r.ResumeCommand)
}

// printPostmortemJSON writes the report to stdout as JSON.
func printPostmortemJSON(r postmortemReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

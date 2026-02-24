package recovery

import "fmt"

// HintType identifies the category of recovery hint.
type HintType string

const (
	HintResume    HintType = "resume"
	HintForce     HintType = "force"
	HintWorkspace HintType = "workspace"
	HintDebug     HintType = "debug"
)

// ErrorClass categorizes a pipeline failure for hint selection.
type ErrorClass string

const (
	ClassContractValidation ErrorClass = "contract_validation"
	ClassSecurityViolation  ErrorClass = "security_violation"
	ClassPreflight          ErrorClass = "preflight"
	ClassRuntimeError       ErrorClass = "runtime_error"
	ClassUnknown            ErrorClass = "unknown"
)

// PreflightMetadata contains details about a preflight failure.
type PreflightMetadata struct {
	MissingSkills []string `json:"missing_skills,omitempty"`
	MissingTools  []string `json:"missing_tools,omitempty"`
}

// RecoveryHint represents a single suggested recovery action.
type RecoveryHint struct {
	Label   string   `json:"label"`
	Command string   `json:"command"`
	Type    HintType `json:"type"`
}

// RecoveryBlock holds all recovery hints for a single step failure.
type RecoveryBlock struct {
	PipelineName  string         `json:"pipeline_name"`
	StepID        string         `json:"step_id"`
	Input         string         `json:"input"`
	WorkspacePath string         `json:"workspace_path"`
	ErrorClass    ErrorClass     `json:"error_class"`
	Hints         []RecoveryHint `json:"hints"`
}

// BuildRecoveryBlock constructs a RecoveryBlock with appropriate hints based on the error class.
// workspaceRoot is the resolved workspace directory (e.g. from manifest runtime.workspace_root);
// pass "" to use the default ".wave/workspaces".
// preflightMeta is optional preflight metadata containing missing skills/tools; pass nil when not a preflight error.
func BuildRecoveryBlock(pipelineName, input, stepID, runID, workspaceRoot string, errClass ErrorClass, preflightMeta *PreflightMetadata) *RecoveryBlock {
	if workspaceRoot == "" {
		workspaceRoot = ".wave/workspaces"
	}

	// Construct workspace path, avoiding double trailing slash when stepID is empty
	workspacePath := fmt.Sprintf("%s/%s", workspaceRoot, runID)
	if stepID != "" {
		workspacePath = fmt.Sprintf("%s/%s", workspacePath, stepID)
	}
	workspacePath = workspacePath + "/"

	block := &RecoveryBlock{
		PipelineName:  pipelineName,
		StepID:        stepID,
		Input:         input,
		WorkspacePath: workspacePath,
		ErrorClass:    errClass,
	}

	// Add preflight-specific hints (skills and tools) before resume hints
	if errClass == ClassPreflight && preflightMeta != nil {
		// Generate skill install hints
		for _, skill := range preflightMeta.MissingSkills {
			block.Hints = append(block.Hints, RecoveryHint{
				Label:   "Install missing skill",
				Command: fmt.Sprintf("wave skill install %s", ShellEscape(skill)),
				Type:    HintType("preflight"),
			})
		}

		// Generate tool hints
		for _, tool := range preflightMeta.MissingTools {
			block.Hints = append(block.Hints, RecoveryHint{
				Label:   "Install missing tool",
				Command: fmt.Sprintf("%s is required but not on PATH\nInstall it using your package manager or ensure it's in PATH", tool),
				Type:    HintType("preflight"),
			})
		}
	}

	// Add resume hint (skip if stepID is unknown OR if this is a preflight error)
	if stepID != "" && errClass != ClassPreflight {
		resumeCmd := buildResumeCommand(pipelineName, input, stepID)
		block.Hints = append(block.Hints, RecoveryHint{
			Label:   "Resume from failed step",
			Command: resumeCmd,
			Type:    HintResume,
		})

		// Add force hint only for contract validation errors
		if errClass == ClassContractValidation {
			forceCmd := resumeCmd + " --force"
			block.Hints = append(block.Hints, RecoveryHint{
				Label:   "Resume and skip validation checks",
				Command: forceCmd,
				Type:    HintForce,
			})
		}
	}

	// Always add workspace hint
	block.Hints = append(block.Hints, RecoveryHint{
		Label:   "Inspect workspace artifacts",
		Command: fmt.Sprintf("ls %s", ShellEscape(block.WorkspacePath)),
		Type:    HintWorkspace,
	})

	// Add debug hint for runtime errors and unknown errors
	if stepID != "" && (errClass == ClassRuntimeError || errClass == ClassUnknown) {
		debugCmd := buildResumeCommand(pipelineName, input, stepID) + " --debug"
		block.Hints = append(block.Hints, RecoveryHint{
			Label:   "Re-run with debug output",
			Command: debugCmd,
			Type:    HintDebug,
		})
	}

	return block
}

// buildResumeCommand constructs the wave run command for resuming from a step.
// Always uses --input flag to avoid ambiguity when input starts with "-".
func buildResumeCommand(pipelineName, input, stepID string) string {
	if input == "" {
		return fmt.Sprintf("wave run %s --from-step %s", ShellEscape(pipelineName), ShellEscape(stepID))
	}
	return fmt.Sprintf("wave run %s --input %s --from-step %s", ShellEscape(pipelineName), ShellEscape(input), ShellEscape(stepID))
}

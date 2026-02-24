package recovery

import (
	"fmt"
	"path/filepath"
)

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

// RecoveryBlockOpts holds parameters for building a recovery block.
type RecoveryBlockOpts struct {
	PipelineName   string
	Input          string
	StepID         string
	RunID          string
	WorkspaceRoot  string              // defaults to ".wave/workspaces" if empty
	ErrClass       ErrorClass
	PreflightMeta  *PreflightMetadata  // nil when not a preflight error
}

// BuildRecoveryBlock constructs a RecoveryBlock with appropriate hints based on the error class.
func BuildRecoveryBlock(opts RecoveryBlockOpts) *RecoveryBlock {
	workspaceRoot := opts.WorkspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = ".wave/workspaces"
	}

	// Construct workspace path using filepath.Join (handles empty segments cleanly)
	workspacePath := filepath.Join(workspaceRoot, opts.RunID)
	if opts.StepID != "" {
		workspacePath = filepath.Join(workspacePath, opts.StepID)
	}
	workspacePath = workspacePath + "/"

	block := &RecoveryBlock{
		PipelineName:  opts.PipelineName,
		StepID:        opts.StepID,
		Input:         opts.Input,
		WorkspacePath: workspacePath,
		ErrorClass:    opts.ErrClass,
	}

	// Add preflight-specific hints (skills and tools) before resume hints
	if opts.ErrClass == ClassPreflight && opts.PreflightMeta != nil {
		// Generate skill install hints
		for _, skill := range opts.PreflightMeta.MissingSkills {
			block.Hints = append(block.Hints, RecoveryHint{
				Label:   "Install missing skill",
				Command: fmt.Sprintf("Check wave.yaml skills.%s.install for the install command", skill),
				Type:    HintType("preflight"),
			})
		}

		// Generate tool hints
		for _, tool := range opts.PreflightMeta.MissingTools {
			block.Hints = append(block.Hints, RecoveryHint{
				Label:   "Install missing tool",
				Command: fmt.Sprintf("%s is required but not on PATH — install it using your package manager", tool),
				Type:    HintType("preflight"),
			})
		}
	}

	// Add resume hint (skip if stepID is unknown OR if this is a preflight error)
	if opts.StepID != "" && opts.ErrClass != ClassPreflight {
		resumeCmd := buildResumeCommand(opts.PipelineName, opts.Input, opts.StepID)
		block.Hints = append(block.Hints, RecoveryHint{
			Label:   "Resume from failed step",
			Command: resumeCmd,
			Type:    HintResume,
		})

		// Add force hint only for contract validation errors
		if opts.ErrClass == ClassContractValidation {
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
	if opts.StepID != "" && (opts.ErrClass == ClassRuntimeError || opts.ErrClass == ClassUnknown) {
		debugCmd := buildResumeCommand(opts.PipelineName, opts.Input, opts.StepID) + " --debug"
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

package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/spf13/cobra"
)

// Output format constants — aliased from internal/config to keep a single
// source of truth shared with the webui launch path.
const (
	OutputFormatAuto  = config.OutputFormatAuto
	OutputFormatJSON  = config.OutputFormatJSON
	OutputFormatText  = config.OutputFormatText
	OutputFormatQuiet = config.OutputFormatQuiet
)

// OutputConfig is aliased from internal/config so the cmd and webui layers
// share an identical shape; cmd-only consumers continue to reference
// commands.OutputConfig without churn.
type OutputConfig = config.OutputConfig

// resolvedFlagsKey is the context key for storing ResolvedFlags.
type resolvedFlagsKey struct{}

// ResolvedFlags captures the full resolved state from PersistentPreRunE.
// Stored in cobra.Command context for downstream access.
type ResolvedFlags struct {
	Output OutputConfig
	NoTUI  bool
}

// ResolveOutputConfig reads all root flag states, detects conflicts, and returns
// the resolved output configuration. It should be called from PersistentPreRunE.
func ResolveOutputConfig(cmd *cobra.Command) (*ResolvedFlags, error) {
	root := cmd.Root()
	flags := root.PersistentFlags()

	jsonFlag := flags.Changed("json")
	quietFlag := flags.Changed("quiet")
	outputFlag := flags.Changed("output")
	noColorFlag := flags.Changed("no-color")
	verboseFlag := flags.Changed("verbose")
	debugFlag, _ := flags.GetBool("debug")
	noTUIFlag, _ := flags.GetBool("no-tui")

	outputVal, _ := flags.GetString("output")
	verbose, _ := flags.GetBool("verbose")
	noColor, _ := flags.GetBool("no-color")

	// Detect conflicts: --json + --output non-json
	if jsonFlag && outputFlag && outputVal != OutputFormatJSON {
		return nil, NewCLIError(CodeFlagConflict,
			fmt.Sprintf("conflicting flags: --json and --output %s", outputVal),
			"Use either --json or --output, not both")
	}

	// Detect conflicts: --quiet + --output non-quiet
	if quietFlag && outputFlag && outputVal != OutputFormatQuiet {
		return nil, NewCLIError(CodeFlagConflict,
			fmt.Sprintf("conflicting flags: --quiet and --output %s", outputVal),
			"Use either --quiet or --output, not both")
	}

	// --quiet + --verbose: quiet wins, warn
	if quietFlag && verboseFlag {
		fmt.Fprintln(os.Stderr, "warning: --quiet and --verbose both set; --quiet takes precedence")
		verbose = false
	}

	// Resolve format: --json > --quiet > --output > default
	format := outputVal
	if jsonFlag {
		format = OutputFormatJSON
	} else if quietFlag {
		format = OutputFormatQuiet
	}

	// Resolve NoTUI
	noTUI := noTUIFlag
	if jsonFlag || quietFlag {
		noTUI = true
	}

	return &ResolvedFlags{
		Output: OutputConfig{
			Format:  format,
			Verbose: verbose,
			NoColor: noColor || noColorFlag,
			Debug:   debugFlag,
		},
		NoTUI: noTUI,
	}, nil
}

// GetOutputConfig reads the resolved output configuration from the command context.
// It checks for ResolvedFlags set by PersistentPreRunE and returns defaults if not found.
func GetOutputConfig(cmd *cobra.Command) OutputConfig {
	if ctx := cmd.Context(); ctx != nil {
		if rf, ok := ctx.Value(resolvedFlagsKey{}).(*ResolvedFlags); ok {
			return rf.Output
		}
	}
	return OutputConfig{Format: OutputFormatAuto}
}

// ResolveFormat resolves the effective output format for a subcommand.
// If a root-level output flag (--json, --quiet, --output) was explicitly set,
// the root value takes precedence. Otherwise, the local format is preserved.
func ResolveFormat(cmd *cobra.Command, localFormat string) string {
	root := cmd.Root()
	flags := root.PersistentFlags()

	// If --json was explicitly set, override
	if flags.Changed("json") {
		return "json"
	}
	// If --quiet was explicitly set, override
	if flags.Changed("quiet") {
		return "quiet"
	}
	// If --output was explicitly set, map to subcommand format
	if flags.Changed("output") {
		outputVal, _ := flags.GetString("output")
		switch outputVal {
		case OutputFormatJSON:
			return "json"
		case OutputFormatQuiet:
			return "quiet"
		case OutputFormatText:
			return "table"
		default:
			return localFormat
		}
	}
	return localFormat
}

// StoreResolvedFlags stores the resolved flags in the command context.
func StoreResolvedFlags(cmd *cobra.Command, rf *ResolvedFlags) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, resolvedFlagsKey{}, rf))
}

// EmitterResult holds the emitter, progress display, and cleanup function
// returned by CreateEmitter.
type EmitterResult struct {
	Emitter  *event.NDJSONEmitter
	Progress event.ProgressEmitter
	Cleanup  func()
}

// CreateEmitter builds the appropriate emitter and progress display based on
// the output format and verbose flag.
//
// Modes:
//   - json:  NDJSON to stdout, no progress display
//   - text:  Plain text progress to stderr, no stdout
//   - quiet: Only final result to stderr, no stdout
//   - auto:  BubbleTea TUI if TTY, plain text if pipe
func CreateEmitter(cfg OutputConfig, pipelineID, pipelineName string, steps []pipeline.Step, m *manifest.Manifest) EmitterResult {
	switch cfg.Format {
	case OutputFormatJSON:
		return EmitterResult{
			Emitter: event.NewNDJSONEmitter(),
			Cleanup: func() {},
		}

	case OutputFormatText:
		progress := display.NewBasicProgressDisplayWithVerbose(cfg.Verbose)
		throttled := display.NewThrottledProgressEmitter(progress)
		return EmitterResult{
			Emitter:  event.NewProgressOnlyEmitter(throttled),
			Progress: throttled,
			Cleanup:  func() {},
		}

	case OutputFormatQuiet:
		progress := display.NewQuietProgressDisplay()
		throttled := display.NewThrottledProgressEmitter(progress)
		return EmitterResult{
			Emitter:  event.NewProgressOnlyEmitter(throttled),
			Progress: throttled,
			Cleanup:  func() {},
		}

	default: // "auto"
		return createAutoEmitter(cfg, pipelineID, pipelineName, steps, m)
	}
}

// resolveForgePersona replaces {{ forge.type }} (and unspaced variant) in a
// persona name with the detected forge type string.
func resolveForgePersona(persona string, info forge.ForgeInfo) string {
	if !strings.Contains(persona, "forge.") {
		return persona
	}
	r := strings.NewReplacer(
		"{{ forge.type }}", string(info.Type),
		"{{forge.type}}", string(info.Type),
		"{{ forge.cli_tool }}", info.CLITool,
		"{{forge.cli_tool}}", info.CLITool,
		"{{ forge.pr_command }}", info.PRCommand,
		"{{forge.pr_command}}", info.PRCommand,
	)
	return r.Replace(persona)
}

// createAutoEmitter selects BubbleTea TUI when connected to a TTY,
// plain text otherwise.
func createAutoEmitter(cfg OutputConfig, pipelineID, pipelineName string, steps []pipeline.Step, _ *manifest.Manifest) EmitterResult {
	termInfo := display.NewTerminalInfo()
	isTTY := termInfo.IsTTY() && termInfo.SupportsANSI()

	// Detect forge for resolving persona template variables
	forgeInfo, _ := forge.DetectFromGitRemotes()

	if isTTY {
		btpd := display.NewBubbleTeaProgressDisplay(pipelineID, pipelineName, len(steps), nil, cfg.Verbose)

		// Register steps for tracking with resolved persona names
		for _, step := range steps {
			btpd.AddStep(step.ID, step.ID, resolveForgePersona(step.Persona, forgeInfo))
		}

		throttled := display.NewThrottledProgressEmitter(btpd)
		emitter := event.NewProgressOnlyEmitter(throttled)

		var once sync.Once
		cleanup := func() {
			once.Do(func() { btpd.Finish() })
		}

		return EmitterResult{
			Emitter:  emitter,
			Progress: throttled,
			Cleanup:  cleanup,
		}
	}

	// Non-TTY: plain text to stderr
	progress := display.NewBasicProgressDisplayWithVerbose(cfg.Verbose)
	throttled := display.NewThrottledProgressEmitter(progress)
	return EmitterResult{
		Emitter:  event.NewProgressOnlyEmitter(throttled),
		Progress: throttled,
		Cleanup:  func() {},
	}
}

// ValidateOutputFormat checks that the output format is valid.
func ValidateOutputFormat(format string) error {
	switch format {
	case OutputFormatAuto, OutputFormatJSON, OutputFormatText, OutputFormatQuiet:
		return nil
	default:
		return fmt.Errorf("invalid output format %q: must be auto, json, text, or quiet", format)
	}
}

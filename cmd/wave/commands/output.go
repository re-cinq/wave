package commands

import (
	"fmt"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/spf13/cobra"
)

// Output format constants
const (
	OutputFormatAuto  = "auto"
	OutputFormatJSON  = "json"
	OutputFormatText  = "text"
	OutputFormatQuiet = "quiet"
)

// OutputConfig holds the resolved output configuration from CLI flags.
type OutputConfig struct {
	Format  string
	Verbose bool
}

// GetOutputConfig reads the -o/--output and -v/--verbose persistent flags from the command.
func GetOutputConfig(cmd *cobra.Command) OutputConfig {
	format, _ := cmd.Root().PersistentFlags().GetString("output")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	return OutputConfig{Format: format, Verbose: verbose}
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
func CreateEmitter(cfg OutputConfig, pipelineName string, steps []pipeline.Step, m *manifest.Manifest) EmitterResult {
	switch cfg.Format {
	case OutputFormatJSON:
		return EmitterResult{
			Emitter: event.NewNDJSONEmitter(),
			Cleanup: func() {},
		}

	case OutputFormatText:
		progress := display.NewBasicProgressDisplayWithVerbose(cfg.Verbose)
		return EmitterResult{
			Emitter:  event.NewProgressOnlyEmitter(progress),
			Progress: progress,
			Cleanup:  func() {},
		}

	case OutputFormatQuiet:
		progress := display.NewQuietProgressDisplay()
		return EmitterResult{
			Emitter:  event.NewProgressOnlyEmitter(progress),
			Progress: progress,
			Cleanup:  func() {},
		}

	default: // "auto"
		return createAutoEmitter(cfg, pipelineName, steps, m)
	}
}

// createAutoEmitter selects BubbleTea TUI when connected to a TTY,
// plain text otherwise.
func createAutoEmitter(cfg OutputConfig, pipelineName string, steps []pipeline.Step, m *manifest.Manifest) EmitterResult {
	termInfo := display.NewTerminalInfo()
	isTTY := termInfo.IsTTY() && termInfo.SupportsANSI()

	if isTTY {
		btpd := display.NewBubbleTeaProgressDisplay(pipelineName, pipelineName, len(steps), nil, cfg.Verbose)

		// Register steps for tracking
		for _, step := range steps {
			btpd.AddStep(step.ID, step.ID, step.Persona)
		}

		emitter := event.NewProgressOnlyEmitter(btpd)

		cleanup := func() {
			btpd.Finish()
		}

		return EmitterResult{
			Emitter:  emitter,
			Progress: btpd,
			Cleanup:  cleanup,
		}
	}

	// Non-TTY: plain text to stderr
	progress := display.NewBasicProgressDisplayWithVerbose(cfg.Verbose)
	return EmitterResult{
		Emitter:  event.NewProgressOnlyEmitter(progress),
		Progress: progress,
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
